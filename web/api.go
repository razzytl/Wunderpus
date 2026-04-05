package web

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/wunderpus/wunderpus/internal/memory"
)

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	cfg := s.manager.Config()
	availableProviders := cfg.AvailableProviders()

	// Create a safe config response
	models := make([]string, 0, len(cfg.ModelList))
	for _, m := range cfg.ModelList {
		models = append(models, m.ModelName)
	}

	skillsLoader := s.manager.GetSkillsLoader()
	var loadedSkills []string
	if skillsLoader != nil {
		for _, sk := range skillsLoader.ListSkills() {
			loadedSkills = append(loadedSkills, sk.Name)
		}
	}

	resp := map[string]interface{}{
		"default_provider":    cfg.DefaultProvider,
		"available_providers": availableProviders,
		"models":              models,
		"skills":              loadedSkills,
		"tools_enabled":       cfg.Tools.Enabled,
	}

	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	store := s.manager.Store()
	if store == nil {
		http.Error(w, `{"error":"history store not available"}`, http.StatusServiceUnavailable)
		return
	}

	// Determine if requesting a specific session or just the list
	sessionID := r.URL.Query().Get("session")

	if sessionID != "" {
		// Try to extract encryption key from headers if present
		var encryptionKey []byte
		keyHeader := r.Header.Get("X-Encryption-Key")
		if keyHeader != "" {
			encryptionKey = []byte(keyHeader)
		} else {
			// fallback to config key if memory encryption is enabled
			if s.manager.Config().Security.Encryption.Enabled {
				encryptionKey = []byte(s.manager.Config().Security.Encryption.Key)
			}
		}

		msgs, err := store.LoadSession(sessionID, encryptionKey)
		if err != nil && !strings.Contains(err.Error(), "no records") {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		// Remap to WSMessage formats or just send raw
		json.NewEncoder(w).Encode(msgs)
		return
	}

	// Otherwise, list sessions
	sessions, err := store.GetSessions()
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	if sessions == nil {
		sessions = []memory.Session{} // return empty array instead of null
	}

	json.NewEncoder(w).Encode(sessions)
}

func (s *Server) handleBranches(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	store := s.manager.Store()
	if store == nil {
		http.Error(w, `{"error":"history store not available"}`, http.StatusServiceUnavailable)
		return
	}

	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		http.Error(w, `{"error":"session parameter is required"}`, http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// List all branches for a session
		branches, err := store.GetBranches(sessionID)
		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"session_id": sessionID,
			"branches":   branches,
		})

	case http.MethodPost:
		// Create a new branch from a specific message
		var req struct {
			FromMessageID int `json:"from_message_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}
		if req.FromMessageID <= 0 {
			http.Error(w, `{"error":"from_message_id is required and must be positive"}`, http.StatusBadRequest)
			return
		}

		branchID, err := store.CreateBranch(sessionID, req.FromMessageID)
		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"session_id":      sessionID,
			"branch_id":       branchID,
			"from_message_id": req.FromMessageID,
		})

	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleBranchMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	store := s.manager.Store()
	if store == nil {
		http.Error(w, `{"error":"history store not available"}`, http.StatusServiceUnavailable)
		return
	}

	sessionID := r.URL.Query().Get("session")
	branchID := r.URL.Query().Get("branch")
	if sessionID == "" || branchID == "" {
		http.Error(w, `{"error":"session and branch parameters are required"}`, http.StatusBadRequest)
		return
	}

	var encryptionKey []byte
	keyHeader := r.Header.Get("X-Encryption-Key")
	if keyHeader != "" {
		encryptionKey = []byte(keyHeader)
	} else if s.manager.Config().Security.Encryption.Enabled {
		encryptionKey = []byte(s.manager.Config().Security.Encryption.Key)
	}

	msgs, err := store.LoadSessionBranch(sessionID, branchID, encryptionKey)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(msgs)
}

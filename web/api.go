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
	var models []string
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

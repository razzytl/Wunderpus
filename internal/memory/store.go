package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wunderpus/wunderpus/internal/provider"
	"github.com/wunderpus/wunderpus/internal/security"
	_ "modernc.org/sqlite"
)

// Store manages conversation history and user preferences using SQLite.
type Store struct {
	db *sql.DB
}

// Session represents a chat session.
type Session struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Title     string    `json:"title"`
}

// NewStore initializes a new memory store using the shared core DB connection.
// Tables are namespaced with mem_ to prevent collisions.
func NewStore(db *sql.DB) (*Store, error) {
	// Create tables if not exist (namespaced with mem_)
	schema := `
	CREATE TABLE IF NOT EXISTS mem_sessions (
		id TEXT PRIMARY KEY,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		title TEXT NOT NULL DEFAULT 'New Conversation'
	);
	
	CREATE TABLE IF NOT EXISTS mem_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		tool_call_id TEXT DEFAULT '',
		tool_calls TEXT DEFAULT '',
		timestamp TEXT NOT NULL,
		FOREIGN KEY (session_id) REFERENCES mem_sessions(id) ON DELETE CASCADE
	);
	
	CREATE TABLE IF NOT EXISTS mem_preferences (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	
	CREATE INDEX IF NOT EXISTS idx_mem_messages_session ON mem_messages(session_id);
	`

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("memory: creating tables: %w", err)
	}

	store := &Store{db: db}
	if err := store.Migrate(); err != nil {
		return nil, fmt.Errorf("memory: migration failed: %w", err)
	}

	return store, nil
}

// Backup performs an online backup of the SQLite database.
func (s *Store) Backup(destPath string) error {
	_, err := s.db.Exec("VACUUM INTO ?", destPath)
	return err
}

// EnsureSession creates a new session if it doesn't exist.
func (s *Store) EnsureSession(sessionID string) error {
	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO mem_sessions (id, created_at, updated_at)
		VALUES (?, ?, ?)`,
		sessionID, now, now)
	return err
}

// SaveMessage appends a message to the specified session.
// encryptionKey is optional, if provided content will be encrypted.
func (s *Store) SaveMessage(sessionID string, msg provider.Message, encryptionKey []byte) error {
	if err := s.EnsureSession(sessionID); err != nil {
		return err
	}

	content := msg.Content
	isEncrypted := 0
	if len(encryptionKey) > 0 {
		var err error
		content, err = security.Encrypt(msg.Content, encryptionKey)
		if err != nil {
			return fmt.Errorf("memory: encrypting message: %w", err)
		}
		isEncrypted = 1
	}

	now := time.Now().Format(time.RFC3339)
	var toolCallsStr string
	if len(msg.ToolCalls) > 0 {
		b, _ := json.Marshal(msg.ToolCalls)
		toolCallsStr = string(b)
	}

	_, err := s.db.Exec(`
		INSERT INTO mem_messages (session_id, role, content, tool_call_id, tool_calls, timestamp, encrypted)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sessionID, msg.Role, content, msg.ToolCallID, toolCallsStr, now, isEncrypted,
	)

	if err == nil {
		s.db.Exec(`UPDATE mem_sessions SET updated_at = ? WHERE id = ?`, now, sessionID)
	}
	return err
}

// GetSessions retrieves a list of all chat sessions ordered by updated_at descending.
func (s *Store) GetSessions() ([]Session, error) {
	rows, err := s.db.Query(`
		SELECT id, created_at, updated_at, title
		FROM mem_sessions 
		ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var id, createdStr, updatedStr, title string
		if err := rows.Scan(&id, &createdStr, &updatedStr, &title); err != nil {
			return nil, err
		}

		created, _ := time.Parse(time.RFC3339, createdStr)
		updated, _ := time.Parse(time.RFC3339, updatedStr)

		sessions = append(sessions, Session{
			ID:        id,
			CreatedAt: created,
			UpdatedAt: updated,
			Title:     title,
		})
	}

	return sessions, rows.Err()
}

// LoadSession retrieves all messages for a given session.
// encryptionKey is required to decrypt encrypted messages.
func (s *Store) LoadSession(sessionID string, encryptionKey []byte) ([]provider.Message, error) {
	rows, err := s.db.Query(`
		SELECT role, content, tool_call_id, tool_calls, encrypted
		FROM mem_messages 
		WHERE session_id = ? 
		ORDER BY timestamp ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []provider.Message
	for rows.Next() {
		var role, content, tcID, tcsStr sql.NullString
		var encrypted int
		if err := rows.Scan(&role, &content, &tcID, &tcsStr, &encrypted); err != nil {
			return nil, err
		}

		msgContent := content.String
		if encrypted == 1 && len(encryptionKey) > 0 {
			var err error
			msgContent, err = security.Decrypt(msgContent, encryptionKey)
			if err != nil {
				// Don't fail the whole load, just report an error placeholder or keep encrypted
				msgContent = "[DECRYPTION FAILED]"
			}
		}

		msg := provider.Message{
			Role:    role.String,
			Content: msgContent,
		}
		if tcID.Valid && tcID.String != "" {
			msg.ToolCallID = tcID.String
		}
		if tcsStr.Valid && tcsStr.String != "" {
			var tcs []provider.ToolCallInfo
			if err := json.Unmarshal([]byte(tcsStr.String), &tcs); err == nil {
				msg.ToolCalls = tcs
			}
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// GetPreference gets a user preference string.
func (s *Store) GetPreference(key string, defaultVal string) string {
	var val string
	err := s.db.QueryRow(`SELECT value FROM mem_preferences WHERE key = ?`, key).Scan(&val)
	if err != nil {
		return defaultVal
	}
	return val
}

// SetPreference saves a user preference string.
func (s *Store) SetPreference(key, value string) error {
	_, err := s.db.Exec(`
		INSERT INTO mem_preferences (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value)
	return err
}

// PruneOldSessions deletes sessions that haven't been updated for the given days.
func (s *Store) PruneOldSessions(days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days).Format(time.RFC3339)
	res, err := s.db.Exec(`DELETE FROM mem_sessions WHERE updated_at < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Close is a no-op — the shared DB connection is managed by db.Manager.
func (s *Store) Close() error {
	return nil
}

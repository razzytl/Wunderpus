package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
	"github.com/wonderpus/wonderpus/internal/provider"
)

// Store manages conversation history and user preferences using SQLite.
type Store struct {
	db *sql.DB
}

// Session represents a chat session.
type Session struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	Title     string
}

// NewStore initializes a new SQLite memory store.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("memory: opening db: %w", err)
	}

	// Create tables if not exist
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		title TEXT NOT NULL DEFAULT 'New Conversation'
	);
	
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		tool_call_id TEXT DEFAULT '',
		tool_calls TEXT DEFAULT '',
		timestamp TEXT NOT NULL,
		FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
	);
	
	CREATE TABLE IF NOT EXISTS preferences (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("memory: creating tables: %w", err)
	}

	return &Store{db: db}, nil
}

// EnsureSession creates a new session if it doesn't exist.
func (s *Store) EnsureSession(sessionID string) error {
	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO sessions (id, created_at, updated_at)
		VALUES (?, ?, ?)`,
		sessionID, now, now)
	return err
}

// SaveMessage appends a message to the specified session.
func (s *Store) SaveMessage(sessionID string, msg provider.Message) error {
	if err := s.EnsureSession(sessionID); err != nil {
		return err
	}

	now := time.Now().Format(time.RFC3339)
	var toolCallsStr string
	if len(msg.ToolCalls) > 0 {
		b, _ := json.Marshal(msg.ToolCalls)
		toolCallsStr = string(b)
	}

	_, err := s.db.Exec(`
		INSERT INTO messages (session_id, role, content, tool_call_id, tool_calls, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)`,
		sessionID, msg.Role, msg.Content, msg.ToolCallID, toolCallsStr, now,
	)

	if err == nil {
		s.db.Exec(`UPDATE sessions SET updated_at = ? WHERE id = ?`, now, sessionID)
	}
	return err
}

// LoadSession retrieves all messages for a given session.
func (s *Store) LoadSession(sessionID string) ([]provider.Message, error) {
	rows, err := s.db.Query(`
		SELECT role, content, tool_call_id, tool_calls 
		FROM messages 
		WHERE session_id = ? 
		ORDER BY timestamp ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []provider.Message
	for rows.Next() {
		var role, content, tcID, tcsStr sql.NullString
		if err := rows.Scan(&role, &content, &tcID, &tcsStr); err != nil {
			return nil, err
		}

		msg := provider.Message{
			Role:    role.String,
			Content: content.String,
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
	err := s.db.QueryRow(`SELECT value FROM preferences WHERE key = ?`, key).Scan(&val)
	if err != nil {
		return defaultVal
	}
	return val
}

// SetPreference saves a user preference string.
func (s *Store) SetPreference(key, value string) error {
	_, err := s.db.Exec(`
		INSERT INTO preferences (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value)
	return err
}

// Close closes the database connection.
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

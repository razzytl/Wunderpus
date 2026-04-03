package toolsynth

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// SynthesizedToolEntry tracks a registered synthesized tool.
type SynthesizedToolEntry struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	SourcePath  string    `json:"source_path"`
	Origin      string    `json:"origin"` // "synthesized"
	Spec        ToolSpec  `json:"spec"`
	Registered  time.Time `json:"registered"`
}

// AuditWriter writes audit entries for tool synthesis events.
type AuditWriter interface {
	WriteToolSynthesized(name, sourcePath string, spec ToolSpec, testResult ToolTestResult) error
}

// EventPublisher publishes events on the event bus.
type EventPublisher interface {
	PublishToolSynthesized(name string, spec ToolSpec)
}

// RegistryRef is a minimal interface for registering tools in the tool registry.
type RegistryRef interface {
	Register(tool interface{ Name() string }) error
}

// Registrar handles the registration of synthesized tools.
// It writes source files to the generated tools directory and tracks metadata.
type Registrar struct {
	outputDir   string
	db          *sql.DB // shared core DB connection for storing tool registry metadata
	mu          sync.RWMutex
	tools       map[string]SynthesizedToolEntry
	auditWriter AuditWriter
	eventPub    EventPublisher
	registry    RegistryRef // optional: actual tool registry for runtime registration
}

// NewRegistrar creates a registrar that manages synthesized tools.
// outputDir is where .go source files are written (internal/tool/generated/).
// db is the shared database connection for storing tool registry metadata.
func NewRegistrar(outputDir string, db *sql.DB) *Registrar {
	r := &Registrar{
		outputDir: outputDir,
		db:        db,
		tools:     make(map[string]SynthesizedToolEntry),
	}
	_ = r.loadExisting()
	return r
}

// SetAuditWriter configures the audit writer for EventToolSynthesized.
func (r *Registrar) SetAuditWriter(w AuditWriter) {
	r.auditWriter = w
}

// SetEventPublisher configures the event publisher.
func (r *Registrar) SetEventPublisher(p EventPublisher) {
	r.eventPub = p
}

// SetRegistry configures the tool registry for runtime registration.
// When set, synthesized tools are registered at runtime (requires binary rebuild for code availability).
func (r *Registrar) SetRegistry(reg RegistryRef) {
	r.registry = reg
}

// Register takes a validated tool spec, source code, and test result,
// writes the source to the generated directory, and records metadata.
func (r *Registrar) Register(spec ToolSpec, source string, testResult ToolTestResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	slog.Info("registrar: registering tool", "name", spec.Name)

	// Ensure output directory exists
	if err := os.MkdirAll(r.outputDir, 0o755); err != nil {
		return fmt.Errorf("registrar: create output dir: %w", err)
	}

	// Write source file
	fileName := fmt.Sprintf("%s.go", spec.Name)
	filePath := filepath.Join(r.outputDir, fileName)

	if err := os.WriteFile(filePath, []byte(source), 0o644); err != nil {
		return fmt.Errorf("registrar: write source: %w", err)
	}

	// Create registry entry
	entry := SynthesizedToolEntry{
		Name:        spec.Name,
		Description: spec.Description,
		SourcePath:  filePath,
		Origin:      "synthesized",
		Spec:        spec,
		Registered:  time.Now(),
	}

	r.tools[spec.Name] = entry

	// Persist metadata to SQLite
	if err := r.persistEntry(entry); err != nil {
		slog.Warn("registrar: failed to persist metadata", "error", err)
		// Don't fail registration — source file is already written
	}

	// Write audit event
	if r.auditWriter != nil {
		if err := r.auditWriter.WriteToolSynthesized(spec.Name, filePath, spec, testResult); err != nil {
			slog.Warn("registrar: audit write failed", "error", err)
		}
	}

	// Publish event
	if r.eventPub != nil {
		r.eventPub.PublishToolSynthesized(spec.Name, spec)
	}

	slog.Info("registrar: tool registered",
		"name", spec.Name,
		"path", filePath,
		"origin", "synthesized")

	return nil
}

// Get returns a registered synthesized tool entry.
func (r *Registrar) Get(name string) (SynthesizedToolEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.tools[name]
	return entry, ok
}

// List returns all registered synthesized tools.
func (r *Registrar) List() []SynthesizedToolEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entries := make([]SynthesizedToolEntry, 0, len(r.tools))
	for _, e := range r.tools {
		entries = append(entries, e)
	}
	return entries
}

// SourcePath returns the file path for a synthesized tool.
func (r *Registrar) SourcePath(name string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if entry, ok := r.tools[name]; ok {
		return entry.SourcePath
	}
	return ""
}

// IsSynthesized checks if a tool was synthesized (vs. builtin).
func (r *Registrar) IsSynthesized(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.tools[name]
	return ok
}

// Remove removes a synthesized tool's source file and metadata.
func (r *Registrar) Remove(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.tools[name]
	if !ok {
		return fmt.Errorf("registrar: tool %q not found", name)
	}

	// Remove source file
	if err := os.Remove(entry.SourcePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("registrar: remove source: %w", err)
	}

	// Remove from metadata
	delete(r.tools, name)

	// Remove from SQLite
	r.removeEntry(name)

	slog.Info("registrar: tool removed", "name", name)
	return nil
}

// loadExisting loads previously registered tools from the output directory.
func (r *Registrar) loadExisting() error {
	if r.outputDir == "" {
		return nil
	}

	entries, err := os.ReadDir(r.outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".go")
		path := filepath.Join(r.outputDir, entry.Name())

		r.tools[name] = SynthesizedToolEntry{
			Name:       name,
			SourcePath: path,
			Origin:     "synthesized",
			Registered: time.Now(),
		}
	}

	if len(r.tools) > 0 {
		slog.Info("registrar: loaded existing synthesized tools", "count", len(r.tools))
	}

	return nil
}

// persistEntry saves tool metadata to SQLite.
func (r *Registrar) persistEntry(entry SynthesizedToolEntry) error {
	if r.db == nil {
		return nil
	}

	// Create table if not exists
	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS synth_tools_registry (
			name TEXT PRIMARY KEY,
			description TEXT,
			source_path TEXT,
			origin TEXT DEFAULT 'synthesized',
			spec_json TEXT,
			registered_at TEXT
		)`)
	if err != nil {
		return err
	}

	specJSON, _ := json.Marshal(entry.Spec)

	_, err = r.db.Exec(`
		INSERT OR REPLACE INTO synth_tools_registry (name, description, source_path, origin, spec_json, registered_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		entry.Name, entry.Description, entry.SourcePath, entry.Origin, string(specJSON), entry.Registered.Format(time.RFC3339))

	return err
}

// removeEntry deletes tool metadata from SQLite.
func (r *Registrar) removeEntry(name string) {
	if r.db == nil {
		return
	}
	_, _ = r.db.Exec(`DELETE FROM synth_tools_registry WHERE name = ?`, name)
}

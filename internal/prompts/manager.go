package prompts

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// Manager loads system prompts from disk with version tracking.
type Manager struct {
	mu       sync.RWMutex
	baseDir  string
	active   string            // filename of active prompt
	versions map[string]string // version -> content cache
}

// NewManager creates a prompt manager that loads from the given directory.
func NewManager(baseDir string, activeVersion string) *Manager {
	m := &Manager{
		baseDir:  baseDir,
		active:   activeVersion,
		versions: make(map[string]string),
	}
	// Pre-load the active version
	if _, err := m.Load(activeVersion); err != nil {
		slog.Warn("prompts: failed to load active version", "version", activeVersion, "error", err)
	}
	return m
}

// Load reads a prompt version from disk (or cache).
func (m *Manager) Load(version string) (string, error) {
	m.mu.RLock()
	if content, ok := m.versions[version]; ok {
		m.mu.RUnlock()
		return content, nil
	}
	m.mu.RUnlock()

	filename := fmt.Sprintf("system_%s.md", version)
	path := filepath.Join(m.baseDir, filename)

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("prompts: reading %s: %w", filename, err)
	}

	content := string(data)
	m.mu.Lock()
	m.versions[version] = content
	m.mu.Unlock()

	slog.Info("prompts: loaded version", "version", version, "path", path)
	return content, nil
}

// GetActive returns the currently active prompt version content.
func (m *Manager) GetActive() (string, error) {
	return m.Load(m.active)
}

// SetActive changes the active prompt version.
func (m *Manager) SetActive(version string) error {
	content, err := m.Load(version)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.active = version
	m.mu.Unlock()
	_ = content // already cached by Load
	slog.Info("prompts: active version changed", "version", version)
	return nil
}

// ActiveVersion returns the current active version string.
func (m *Manager) ActiveVersion() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.active
}

// ListVersions returns all prompt versions found on disk.
func (m *Manager) ListVersions() ([]string, error) {
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return nil, fmt.Errorf("prompts: reading directory: %w", err)
	}

	var versions []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		// Match pattern: system_v1.md, system_v2.md, etc.
		name := e.Name()
		if len(name) > 11 && name[:7] == "system_" && name[len(name)-3:] == ".md" {
			versions = append(versions, name[7:len(name)-3])
		}
	}
	return versions, nil
}

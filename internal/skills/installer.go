package skills

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SkillInstaller handles installation of skills from various sources.
type SkillInstaller struct {
	workspace string
}

// NewSkillInstaller creates a new skill installer.
func NewSkillInstaller(workspace string) *SkillInstaller {
	return &SkillInstaller{
		workspace: workspace,
	}
}

// InstallSource represents where a skill can be installed from.
type InstallSource string

const (
	InstallSourceGitHub  InstallSource = "github"
	InstallSourceLocal   InstallSource = "local"
	InstallSourceClawHub InstallSource = "clawhub"
)

// InstallOptions contains options for skill installation.
type InstallOptions struct {
	Source  InstallSource
	Version string // For ClawHub - specific version to install
	Force   bool   // Overwrite existing skill
}

// InstallFromGitHub fetches a SKILL.md from a repository and installs it into the workspace.
func (si *SkillInstaller) InstallFromGitHub(ctx context.Context, repo string) error {
	skillName := filepath.Base(repo)
	skillDir := filepath.Join(si.workspace, "skills", skillName)

	if _, err := os.Stat(skillDir); err == nil {
		return fmt.Errorf("skill '%s' already exists", skillName)
	}

	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/main/SKILL.md", repo)

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch skill: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch skill: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")

	if err := os.WriteFile(skillPath, body, 0o600); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}

	return nil
}

// InstallFromLocalPath copies a skill from a local path to the workspace skills directory.
// Supports both:
// - Single SKILL.md file: copies to skills/{name}/SKILL.md
// - Skill directory: copies entire directory contents
func (si *SkillInstaller) InstallFromLocalPath(ctx context.Context, sourcePath string) error {
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to access source path: %w", err)
	}

	// Determine skill name from path
	var skillName string
	var srcDir string

	if sourceInfo.IsDir() {
		// Check if it's a skill directory (contains SKILL.md)
		skillFile := filepath.Join(sourcePath, "SKILL.md")
		if _, err := os.Stat(skillFile); err == nil {
			// It's a skill directory
			srcDir = sourcePath
			skillName = filepath.Base(sourcePath)
		} else {
			// Assume it's a skills root directory, look for subdirs with SKILL.md
			return fmt.Errorf("source directory must be a skill directory containing SKILL.md")
		}
	} else {
		// It's a file - must be SKILL.md
		if !strings.HasSuffix(strings.ToLower(sourcePath), ".md") {
			return fmt.Errorf("source file must be a .md file (SKILL.md)")
		}
		srcDir = filepath.Dir(sourcePath)
		skillName = strings.TrimSuffix(filepath.Base(sourcePath), ".md")
	}

	skillDir := filepath.Join(si.workspace, "skills", skillName)

	// Check if skill already exists
	if !os.IsNotExist(err) {
		if _, err := os.Stat(skillDir); err == nil {
			return fmt.Errorf("skill '%s' already exists (use --force to overwrite)", skillName)
		}
	}

	// Create destination directory
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	// Copy SKILL.md
	srcSkillFile := filepath.Join(srcDir, "SKILL.md")
	destSkillFile := filepath.Join(skillDir, "SKILL.md")

	skillContent, err := os.ReadFile(srcSkillFile)
	if err != nil {
		return fmt.Errorf("failed to read source SKILL.md: %w", err)
	}

	if err := os.WriteFile(destSkillFile, skillContent, 0o600); err != nil {
		return fmt.Errorf("failed to write SKILL.md: %w", err)
	}

	// Optionally copy other files in the directory (references/, etc.)
	srcEntries, err := os.ReadDir(srcDir)
	if err == nil {
		for _, entry := range srcEntries {
			if entry.Name() == "SKILL.md" {
				continue // Already copied
			}
			srcPath := filepath.Join(srcDir, entry.Name())
			destPath := filepath.Join(skillDir, entry.Name())

			if entry.IsDir() {
				// Copy directory recursively
				if err := copyDir(srcPath, destPath); err != nil {
					return fmt.Errorf("failed to copy directory %s: %w", entry.Name(), err)
				}
			} else {
				// Copy file
				content, err := os.ReadFile(srcPath)
				if err != nil {
					return fmt.Errorf("failed to read %s: %w", entry.Name(), err)
				}
				if err := os.WriteFile(destPath, content, 0o600); err != nil {
					return fmt.Errorf("failed to write %s: %w", entry.Name(), err)
				}
			}
		}
	}

	return nil
}

// copyDir recursively copies a directory.
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			content, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, content, entry.Type()); err != nil {
				return err
			}
		}
	}

	return nil
}

// InstallFromClawHub installs a skill from ClawHub registry.
func (si *SkillInstaller) InstallFromClawHub(ctx context.Context, slug string, version string, registry *ClawHubRegistry) error {
	if registry == nil {
		return fmt.Errorf("ClawHub registry not configured")
	}

	skillDir := filepath.Join(si.workspace, "skills", slug)

	// Check if skill already exists
	if _, err := os.Stat(skillDir); err == nil {
		return fmt.Errorf("skill '%s' already exists (use --force to overwrite)", slug)
	}

	result, err := registry.DownloadAndInstall(ctx, slug, version, skillDir)
	if err != nil {
		return fmt.Errorf("failed to install from ClawHub: %w", err)
	}

	// Check moderation results
	if result.IsMalwareBlocked {
		return fmt.Errorf("skill '%s' was blocked by ClawHub moderation (malware detected)", slug)
	}
	if result.IsSuspicious {
		// Still allowed but could warn user
		fmt.Printf("Warning: skill '%s' is flagged as suspicious by ClawHub\n", slug)
	}

	return nil
}

// Uninstall removes a skill from the workspace.
func (si *SkillInstaller) Uninstall(skillName string) error {
	skillDir := filepath.Join(si.workspace, "skills", skillName)

	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return fmt.Errorf("skill '%s' not found", skillName)
	}

	if err := os.RemoveAll(skillDir); err != nil {
		return fmt.Errorf("failed to remove skill: %w", err)
	}

	return nil
}

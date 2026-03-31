package skills

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSkillInstaller_InstallFromLocalPath(t *testing.T) {
	// Create temp directory for workspace
	workspace, err := os.MkdirTemp("", "wunderpus-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workspace)

	// Create source skill file
	sourceDir, err := os.MkdirTemp("", "wunderpus-source-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(sourceDir)

	// Write SKILL.md
	skillContent := `---
name: test-skill
description: A test skill
---
# Test Skill
This is a test skill content.`
	if err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create installer
	installer := NewSkillInstaller(workspace)

	// Test installation
	err = installer.InstallFromLocalPath(context.Background(), sourceDir)
	if err != nil {
		t.Fatalf("InstallFromLocalPath failed: %v", err)
	}

	// Verify installed
	installedPath := filepath.Join(workspace, "skills", filepath.Base(sourceDir), "SKILL.md")
	_, statErr := os.Stat(installedPath)
	if os.IsNotExist(statErr) {
		t.Fatalf("Skill file not installed at %s", installedPath)
	}

	// Verify content
	installedContent, err := os.ReadFile(installedPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(installedContent) != skillContent {
		t.Errorf("Skill content mismatch:\nexpected: %s\ngot: %s", skillContent, string(installedContent))
	}
}

func TestSkillInstaller_Uninstall(t *testing.T) {
	// Create temp directory for workspace
	workspace, err := os.MkdirTemp("", "wunderpus-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workspace)

	skillName := "test-skill"
	skillDir := filepath.Join(workspace, "skills", skillName)

	// Create a skill directory
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Test"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create installer and uninstall
	installer := NewSkillInstaller(workspace)
	err = installer.Uninstall(skillName)
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	// Verify removed
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Errorf("Skill directory not removed: %s", skillDir)
	}
}

func TestSkillInstaller_InstallFromLocalPath_DoubleInstall(t *testing.T) {
	// Create temp directory for workspace
	workspace, err := os.MkdirTemp("", "wunderpus-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workspace)

	// Create source skill file
	sourceDir, err := os.MkdirTemp("", "wunderpus-source-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(sourceDir)

	// Write SKILL.md
	if err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("# Test"), 0o644); err != nil {
		t.Fatal(err)
	}

	installer := NewSkillInstaller(workspace)

	// First install should succeed
	err = installer.InstallFromLocalPath(context.Background(), sourceDir)
	if err != nil {
		t.Fatalf("First install failed: %v", err)
	}

	// Second install should fail
	err = installer.InstallFromLocalPath(context.Background(), sourceDir)
	if err == nil {
		t.Error("Expected error for double install, got nil")
	}
}

func TestValidateSkillIdentifier(t *testing.T) {
	tests := []struct {
		name    string
		slug    string
		wantErr bool
	}{
		{"valid simple", "test-skill", false},
		{"valid with numbers", "skill123", false},
		{"valid complex", "my-awesome-skill-123", false},
		{"empty", "", true},
		{"invalid chars", "test_skill", true},
		{"invalid start", "-test", true},
		{"invalid end", "test-", true},
		// Long but valid (regex accepts it, validation is handled by length check)
		{"long but valid", "a-b-c-d-e-f-g-h-i-j-k-l-m-n-o-p-q-r-s-t-u-v-w-x-y-z", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSkillIdentifier(tt.slug)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSkillIdentifier() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

package skills

import (
	"testing"
)

// TestSkillInfo_Validate tests skill info validation
func TestSkillInfo_Validate(t *testing.T) {
	tests := []struct {
		name    string
		info    SkillInfo
		wantErr bool
	}{
		{
			name: "valid skill",
			info: SkillInfo{
				Name:        "test-skill",
				Description: "A test skill",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			info: SkillInfo{
				Name:        "",
				Description: "A test skill",
			},
			wantErr: true,
		},
		{
			name: "invalid name with spaces",
			info: SkillInfo{
				Name:        "test skill",
				Description: "A test skill",
			},
			wantErr: true,
		},
		{
			name: "invalid name with special chars",
			info: SkillInfo{
				Name:        "test_skill",
				Description: "A test skill",
			},
			wantErr: true,
		},
		{
			name: "empty description",
			info: SkillInfo{
				Name:        "test-skill",
				Description: "",
			},
			wantErr: true,
		},
		{
			name: "valid with version",
			info: SkillInfo{
				Name:        "test-skill",
				Description: "A test skill",
				Version:     "1.0.0",
			},
			wantErr: false,
		},
		{
			name: "name too long",
			info: SkillInfo{
				Name:        "this-is-a-very-long-name-that-exceeds-the-maximum-length-for-a-skill-name-which-is-64-characters-long",
				Description: "A test skill",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.info.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSkillMetadata tests SkillMetadata structure
func TestSkillMetadata(t *testing.T) {
	meta := SkillMetadata{
		Name:        "test-skill",
		Description: "A test skill",
		Version:     "1.0.0",
	}

	if meta.Name != "test-skill" {
		t.Errorf("expected Name 'test-skill', got %q", meta.Name)
	}
	if meta.Description != "A test skill" {
		t.Errorf("expected Description 'A test skill', got %q", meta.Description)
	}
	if meta.Version != "1.0.0" {
		t.Errorf("expected Version '1.0.0', got %q", meta.Version)
	}
}

// TestSkillInfo_Structure tests SkillInfo fields
func TestSkillInfo_Structure(t *testing.T) {
	info := SkillInfo{
		Name:        "test-skill",
		Path:        "/path/to/skill/SKILL.md",
		Source:      "builtin",
		Description: "A test skill",
		Version:     "1.0.0",
	}

	if info.Name != "test-skill" {
		t.Errorf("expected Name 'test-skill', got %q", info.Name)
	}
	if info.Path != "/path/to/skill/SKILL.md" {
		t.Errorf("expected Path '/path/to/skill/SKILL.md', got %q", info.Path)
	}
	if info.Source != "builtin" {
		t.Errorf("expected Source 'builtin', got %q", info.Source)
	}
}

// TestNewSkillsLoader tests creating a new skills loader
func TestNewSkillsLoader(t *testing.T) {
	loader := NewSkillsLoader("/workspace", "/home/user/.wunderpus/skills", "./skills")

	if loader == nil {
		t.Fatal("expected non-nil loader")
	}
	if loader.workspace != "/workspace" {
		t.Errorf("expected workspace '/workspace', got %q", loader.workspace)
	}
	if loader.globalSkills != "/home/user/.wunderpus/skills" {
		t.Errorf("expected globalSkills '/home/user/.wunderpus/skills', got %q", loader.globalSkills)
	}
	if loader.builtinSkills != "./skills" {
		t.Errorf("expected builtinSkills './skills', got %q", loader.builtinSkills)
	}
}

// TestSkillsLoader_SkillRoots tests getting skill roots
func TestSkillsLoader_SkillRoots(t *testing.T) {
	loader := NewSkillsLoader("/workspace", "/global", "/builtin")

	roots := loader.SkillRoots()
	if len(roots) != 3 {
		t.Errorf("expected 3 roots, got %d", len(roots))
	}
}

// TestSkillsLoader_SkillRoots_Empty tests skill roots with empty values
func TestSkillsLoader_SkillRoots_Empty(t *testing.T) {
	// When workspace is empty, filepath.Join("", "skills") returns "skills"
	loader := NewSkillsLoader("", "", "")

	roots := loader.SkillRoots()
	// With empty inputs, workspaceSkills becomes "skills" from filepath.Join
	// So we expect 1 root ("skills")
	if len(roots) != 1 {
		t.Errorf("expected 1 root for empty dirs, got %d", len(roots))
	}
}

// TestEscapeXML tests XML escaping
func TestEscapeXML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<test>", "&lt;test&gt;"},
		{"test&value", "test&amp;value"},
		{"plain", "plain"},
		{"a & b < c > d", "a &amp; b &lt; c &gt; d"},
	}

	for _, tt := range tests {
		result := escapeXML(tt.input)
		if result != tt.expected {
			t.Errorf("escapeXML(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// TestMaxNameLength_Constant tests max name length constant
func TestMaxNameLength_Constant(t *testing.T) {
	if MaxNameLength != 64 {
		t.Errorf("expected MaxNameLength 64, got %d", MaxNameLength)
	}
}

// TestMaxDescriptionLength_Constant tests max description length constant
func TestMaxDescriptionLength_Constant(t *testing.T) {
	if MaxDescriptionLength != 1024 {
		t.Errorf("expected MaxDescriptionLength 1024, got %d", MaxDescriptionLength)
	}
}

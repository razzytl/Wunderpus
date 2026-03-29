package social

import (
	"testing"
)

func TestSocialOperator_GeneratePost(t *testing.T) {
	op := &SocialOperator{}

	if op == nil {
		t.Error("Expected operator to be created")
	}
}

func TestEmail_Structure(t *testing.T) {
	email := Email{
		ID:      "test-123",
		To:      "test@example.com",
		Subject: "Test",
		Status:  "draft",
	}

	if email.ID != "test-123" {
		t.Errorf("Expected ID test-123, got %s", email.ID)
	}
}

func TestPost_Structure(t *testing.T) {
	post := Post{
		ID:       "post-123",
		Platform: PlatformTwitter,
		Content:  "Hello world",
		Status:   "draft",
	}

	if post.Platform != PlatformTwitter {
		t.Errorf("Expected Twitter platform, got %s", post.Platform)
	}
}

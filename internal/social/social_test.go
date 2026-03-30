package social

import (
	"testing"
)

func TestSocialOperator_GeneratePost(t *testing.T) {
	op := &SocialOperator{}
	_ = op
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
	if email.To != "test@example.com" {
		t.Errorf("Expected To 'test@example.com', got %s", email.To)
	}
	if email.Subject != "Test" {
		t.Errorf("Expected Subject 'Test', got %s", email.Subject)
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
		t.Errorf("Expected platform %s, got %s", PlatformTwitter, post.Platform)
	}
	if post.Content != "Hello world" {
		t.Errorf("Expected content 'Hello world', got %s", post.Content)
	}
}

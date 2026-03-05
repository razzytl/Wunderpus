package provider

import (
	"testing"
)

func TestOpenAIMultimodal(t *testing.T) {
	msgs := []Message{
		{
			Role: RoleUser,
			MultiContent: []ContentPart{
				{Type: "text", Text: "What is in this image?"},
				{Type: "image_url", ImageURL: &ImageURL{URL: "data:image/jpeg;base64,stuff"}},
			},
		},
	}

	openaiMsgs := toOpenAIMessages(msgs)
	if len(openaiMsgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(openaiMsgs))
	}

	content := openaiMsgs[0]["content"].([]map[string]any)
	if len(content) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(content))
	}

	if content[0]["type"] != "text" || content[0]["text"] != "What is in this image?" {
		t.Errorf("unexpected text part: %v", content[0])
	}

	img := content[1]["image_url"].(map[string]any)
	if img["url"] != "data:image/jpeg;base64,stuff" {
		t.Errorf("unexpected image url: %v", img["url"])
	}
}

func TestAnthropicMultimodal(t *testing.T) {
	msgs := []Message{
		{
			Role: RoleUser,
			MultiContent: []ContentPart{
				{Type: "text", Text: "Look at this"},
				{Type: "image_url", ImageURL: &ImageURL{URL: "data:image/png;base64,iVBOR"}},
			},
		},
	}

	anthropicMsgs := toAnthropicMessages(msgs)
	if len(anthropicMsgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(anthropicMsgs))
	}

	content := anthropicMsgs[0]["content"].([]map[string]any)
	if len(content) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(content))
	}

	if content[1]["type"] != "image" {
		t.Errorf("expected type image, got %v", content[1]["type"])
	}

	source := content[1]["source"].(map[string]any)
	if source["media_type"] != "image/png" {
		t.Errorf("expected image/png, got %v", source["media_type"])
	}
}

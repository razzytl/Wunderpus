package money

import (
	"context"
	"log/slog"
	"time"
)

// ContentPlatform represents a content publishing platform.
type ContentPlatform string

const (
	PlatformGhost      ContentPlatform = "ghost"
	PlatformSubstack   ContentPlatform = "substack"
	PlatformYouTube    ContentPlatform = "youtube"
	PlatformAmazon     ContentPlatform = "amazon_kdp"
	PlatformPromptBase ContentPlatform = "promptbase"
)

// Content represents a piece of content to be published.
type Content struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Body        string            `json:"body"`
	Type        string            `json:"type"` // "article", "ebook", "video_script", "prompt_pack"
	Topic       string            `json:"topic"`
	Platforms   []ContentPlatform `json:"platforms"`
	Status      string            `json:"status"` // "draft", "published"
	PublishedAt *time.Time        `json:"published_at"`
	Revenue     float64           `json:"revenue"`
}

// ContentEngine manages content creation and monetization.
type ContentEngine struct {
	writerLLM    LLMCaller
	browserAgent BrowserAgent
	worldModel   WorldModelQuery
}

// WorldModelQuery interface for querying the world model.
type WorldModelQuery interface {
	Ask(ctx context.Context, question string) (string, error)
	Context(ctx context.Context, task string) ([]interface{}, error)
}

// ContentConfig holds content engine configuration.
type ContentConfig struct {
	Enabled          bool
	MinWordCount     int
	PublishPlatforms []ContentPlatform
}

// NewContentEngine creates a new content engine.
func NewContentEngine(cfg ContentConfig, llm LLMCaller, browser BrowserAgent, wm WorldModelQuery) *ContentEngine {
	return &ContentEngine{
		writerLLM:    llm,
		browserAgent: browser,
		worldModel:   wm,
	}
}

// GenerateArticle generates an article on a given topic.
func (e *ContentEngine) GenerateArticle(ctx context.Context, topic string) (*Content, error) {
	// Get context from world model
	var contextInfo []string
	if e.worldModel != nil {
		if entities, err := e.worldModel.Context(ctx, topic); err == nil {
			for _, e := range entities {
				if s, ok := e.(string); ok {
					contextInfo = append(contextInfo, s)
				}
			}
		}
	}

	// Build prompt with context
	prompt := "Write a high-quality article about: " + topic
	if len(contextInfo) > 0 {
		prompt += ". Context from previous research: " + joinStrings(contextInfo)
	}
	prompt += ". Make it engaging, well-researched, and suitable for publication."

	req := LLMRequest{
		SystemPrompt: "You are a professional writer specializing in technical content.",
		UserPrompt:   prompt,
		Temperature:  0.7,
		MaxTokens:    4000,
	}

	body, err := e.writerLLM.Complete(req)
	if err != nil {
		return nil, err
	}

	content := &Content{
		ID:     generateID(),
		Title:  topic,
		Body:   body,
		Type:   "article",
		Topic:  topic,
		Status: "draft",
	}

	slog.Info("content: article generated", "topic", topic, "length", len(body))
	return content, nil
}

// Publish submits content to the specified platform.
func (e *ContentEngine) Publish(ctx context.Context, content *Content, platform ContentPlatform) error {
	switch platform {
	case PlatformGhost, PlatformSubstack:
		return e.publishBlog(ctx, content, platform)
	case PlatformYouTube:
		return e.publishYouTube(ctx, content)
	case PlatformAmazon:
		return e.publishEBook(ctx, content)
	case PlatformPromptBase:
		return e.publishPromptPack(ctx, content)
	default:
		return nil
	}
}

func (e *ContentEngine) publishBlog(ctx context.Context, content *Content, platform ContentPlatform) error {
	slog.Info("content: publishing to blog", "platform", platform, "title", content.Title)
	// In production: use browser agent to navigate to CMS and publish
	content.Status = "published"
	now := time.Now()
	content.PublishedAt = &now
	return nil
}

func (e *ContentEngine) publishYouTube(ctx context.Context, content *Content) error {
	slog.Info("content: publishing to YouTube", "title", content.Title)
	// Would generate voiceover and publish
	content.Status = "published"
	now := time.Now()
	content.PublishedAt = &now
	return nil
}

func (e *ContentEngine) publishEBook(ctx context.Context, content *Content) error {
	slog.Info("content: publishing to Amazon KDP", "title", content.Title)
	// Would convert to ePub and upload
	content.Status = "published"
	now := time.Now()
	content.PublishedAt = &now
	return nil
}

func (e *ContentEngine) publishPromptPack(ctx context.Context, content *Content) error {
	slog.Info("content: publishing prompts", "title", content.Title)
	// Would package and list on PromptBase/Gumroad
	content.Status = "published"
	now := time.Now()
	content.PublishedAt = &now
	return nil
}

func joinStrings(strs []string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += "; "
		}
		result += s
	}
	return result
}

func generateID() string {
	return time.Now().Format("20060102150405")
}

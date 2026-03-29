package social

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Platform represents a social media platform.
type Platform string

const (
	PlatformTwitter    Platform = "twitter"
	PlatformLinkedIn   Platform = "linkedin"
	PlatformReddit     Platform = "reddit"
	PlatformHackerNews Platform = "hackernews"
)

// Post represents a social media post.
type Post struct {
	ID          string     `json:"id"`
	Platform    Platform   `json:"platform"`
	Content     string     `json:"content"`
	Title       string     `json:"title"`  // For platforms that support titles
	Status      string     `json:"status"` // "draft", "scheduled", "published"
	ScheduledAt *time.Time `json:"scheduled_at"`
	PublishedAt *time.Time `json:"published_at"`
	Likes       int        `json:"likes"`
	Shares      int        `json:"shares"`
	Comments    int        `json:"comments"`
}

// SocialOperator manages social media presence.
type SocialOperator struct {
	browserAgent BrowserAgent
	llm          LLMCaller
	worldModel   WorldModelQuery
	schedule     ContentSchedule
}

// ContentSchedule manages post scheduling.
type ContentSchedule struct {
	Posts []Post
}

// SocialConfig holds configuration.
type SocialConfig struct {
	Enabled        bool
	Platforms      []Platform
	MaxPostsPerDay int
}

// NewSocialOperator creates a new social operator.
func NewSocialOperator(cfg SocialConfig, browser BrowserAgent, llm LLMCaller, wm WorldModelQuery) *SocialOperator {
	return &SocialOperator{
		browserAgent: browser,
		llm:          llm,
		worldModel:   wm,
		schedule:     ContentSchedule{},
	}
}

// GeneratePost creates a post for a given topic.
func (o *SocialOperator) GeneratePost(ctx context.Context, topic string, platform Platform) (*Post, error) {
	slog.Info("social: generating post", "topic", topic, "platform", platform)

	// Get context from world model
	var contextInfo string
	if o.worldModel != nil {
		result, err := o.worldModel.Ask(ctx, "audience data for "+topic)
		if err == nil {
			contextInfo = result
		}
	}

	// Build prompt
	prompt := fmt.Sprintf("Write an engaging social media post about: %s\n", topic)
	if contextInfo != "" {
		prompt = prompt + fmt.Sprintf("Context: %s\n", contextInfo)
	}
	prompt = prompt + fmt.Sprintf("Keep it concise, engaging, and suitable for %s", platform)

	req := LLMRequest{
		SystemPrompt: "You are a social media expert. Write engaging content.",
		UserPrompt:   prompt,
		Temperature:  0.7,
		MaxTokens:    500,
	}

	content, err := o.llm.Complete(req)
	if err != nil {
		return nil, err
	}

	post := &Post{
		ID:       generateID(),
		Platform: platform,
		Content:  content,
		Status:   "draft",
	}

	slog.Info("social: post generated", "platform", platform, "length", len(content))
	return post, nil
}

// Publish posts content to the platform using browser automation.
func (o *SocialOperator) Publish(ctx context.Context, post *Post) error {
	slog.Info("social: publishing post", "platform", post.Platform, "content", post.Content[:min(50, len(post.Content))])

	// In production: use browser agent to navigate to platform and post
	// For now, just mark as published
	post.Status = "published"
	now := time.Now()
	post.PublishedAt = &now

	return nil
}

// Schedule posts content for future publishing.
func (o *SocialOperator) Schedule(post *Post, at time.Time) {
	post.ScheduledAt = &at
	newPost := *post
	newPost.Status = "scheduled"
	o.schedule.Posts = append(o.schedule.Posts, newPost)
	slog.Info("social: post scheduled", "at", at)
}

// MonitorEngagement checks engagement on published posts.
func (o *SocialOperator) MonitorEngagement(ctx context.Context) ([]Post, error) {
	// Would scrape engagement metrics using browser agent
	var publishedPosts []Post
	for _, post := range o.schedule.Posts {
		if post.Status == "published" {
			// Would update with new engagement metrics
			publishedPosts = append(publishedPosts, post)
		}
	}
	return publishedPosts, nil
}

func generateID() string {
	return time.Now().Format("20060102150405")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

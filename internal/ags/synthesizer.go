package ags

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/wunderpus/wunderpus/internal/audit"
	"github.com/wunderpus/wunderpus/internal/events"
	"github.com/wunderpus/wunderpus/internal/provider"
)

// ProposedGoal is the JSON structure the LLM returns.
type ProposedGoal struct {
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Tier            int      `json:"tier"`
	Evidence        []string `json:"evidence"`
	ParentTier0     string   `json:"parent_tier0"`
	ExpectedValue   float64  `json:"expected_value"`
	SuccessCriteria []string `json:"success_criteria"`
}

// ProposedGoalsResponse is the full LLM response structure.
type ProposedGoalsResponse struct {
	ProposedGoals []ProposedGoal `json:"proposed_goals"`
}

// GoalSynthesizer generates new goals from episodic memory patterns,
// world model observations, and weakness reports.
type GoalSynthesizer struct {
	provider    provider.Provider
	store       *GoalStore
	scorer      *PriorityScorer
	events      *events.Bus
	maxPerCycle int
	wmContext   func(taskDesc string) string // optional: world model context provider
}

// NewGoalSynthesizer creates a new synthesizer.
func NewGoalSynthesizer(p provider.Provider, store *GoalStore, scorer *PriorityScorer, bus *events.Bus) *GoalSynthesizer {
	return &GoalSynthesizer{
		provider:    p,
		store:       store,
		scorer:      scorer,
		events:      bus,
		maxPerCycle: 5,
	}
}

// SetWorldModelContext sets a function that provides world model context
// for goal synthesis. Called before LLM goal generation.
func (s *GoalSynthesizer) SetWorldModelContext(fn func(taskDesc string) string) {
	s.wmContext = fn
}

// MemoryEntry represents an episodic memory record for pattern detection.
type MemoryEntry struct {
	EventType string
	Content   string
	Timestamp time.Time
	Duration  time.Duration
	Success   bool
}

// WeaknessSnapshot is a lightweight snapshot for the synthesizer.
type WeaknessSnapshot struct {
	FunctionName string
	Score        float64
}

// Synthesize generates new goals from memory patterns and weakness reports.
func (s *GoalSynthesizer) Synthesize(ctx context.Context, memories []MemoryEntry, weaknesses []WeaknessSnapshot) ([]Goal, error) {
	// 1. Analyze memory patterns
	findings := s.analyzeMemories(memories)

	// 2. Check weakness reports
	for _, w := range weaknesses {
		if w.Score > 0.7 {
			findings = append(findings, fmt.Sprintf(
				"High weakness score (%.2f) for function %s — RSI improvement candidate",
				w.Score, w.FunctionName))
		}
	}

	if len(findings) == 0 {
		return nil, nil
	}

	// 3. LLM synthesis
	systemPrompt := `You are a goal synthesis engine for the Wunderpus AI agent.
Given the following findings from memory analysis, propose new goals as JSON.

CONSTRAINTS:
- Output ONLY valid JSON, no explanation
- Maximum 5 goals per cycle
- Each goal must have: title, description, tier (1-3), evidence (array of strings), parent_tier0 (which Tier 0 goal this serves), expected_value (0.0-1.0), success_criteria (array of strings)
- parent_tier0 must be one of: "Be maximally useful to operators", "Improve own capabilities", "Maintain operational continuity", "Expand knowledge and world-model"

OUTPUT FORMAT:
{
  "proposed_goals": [
    {
      "title": "...",
      "description": "...",
      "tier": 2,
      "evidence": ["..."],
      "parent_tier0": "...",
      "expected_value": 0.75,
      "success_criteria": ["..."]
    }
  ]
}`

	userPrompt := fmt.Sprintf("Findings:\n%s", formatFindings(findings))

	if s.provider == nil {
		return nil, fmt.Errorf("ags synthesizer: no LLM provider configured")
	}

	resp, err := s.provider.Complete(ctx, &provider.CompletionRequest{
		Messages: []provider.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("ags synthesizer: LLM call failed: %w", err)
	}

	// 4. Parse response
	var parsed ProposedGoalsResponse
	if err := json.Unmarshal([]byte(resp.Content), &parsed); err != nil {
		// Attempt to extract JSON if it was wrapped in markdown
		cleaned := strings.TrimSpace(resp.Content)
		if strings.HasPrefix(cleaned, "```json") {
			cleaned = strings.TrimPrefix(cleaned, "```json")
			cleaned = strings.TrimSuffix(cleaned, "```")
		}
		if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
			return nil, fmt.Errorf("ags synthesizer: failed to parse LLM response: %w", err)
		}
	}

	// 5. Validate proposals
	valid := make([]Goal, 0, len(parsed.ProposedGoals))
	for _, pg := range parsed.ProposedGoals {
		if err := s.validateProposal(pg); err != nil {
			slog.Warn("ags synthesizer: invalid proposal rejected", "title", pg.Title, "error", err)
			continue
		}

		goal := NewGoal(
			pg.Title, pg.Description, pg.Tier,
			pg.ParentTier0, pg.Evidence, pg.SuccessCriteria,
			pg.ExpectedValue,
		)
		goal.Priority = s.scorer.Score(goal)
		valid = append(valid, goal)
	}

	// 6. Deduplicate against existing goals
	deduped := s.deduplicate(valid)

	// 7. Limit to maxPerCycle
	if len(deduped) > s.maxPerCycle {
		deduped = deduped[:s.maxPerCycle]
	}

	// 8. Save and return
	for _, g := range deduped {
		if err := s.store.Save(g); err != nil {
			slog.Warn("ags synthesizer: failed to save goal", "id", g.ID, "error", err)
		}
		slog.Info("ags synthesizer: new goal created", "title", g.Title, "tier", g.Tier, "priority", g.Priority)

		if s.events != nil {
			s.events.Publish(events.Event{
				Type:   audit.EventGoalCreated,
				Source: "ags.synthesizer",
				Payload: map[string]interface{}{
					"goal_id":   g.ID,
					"title":     g.Title,
					"tier":      g.Tier,
					"priority":  g.Priority,
					"parent_id": g.ParentID,
				},
			})
		}
	}

	return deduped, nil
}

func (s *GoalSynthesizer) analyzeMemories(memories []MemoryEntry) []string {
	var findings []string

	// Count error types
	errorCounts := make(map[string]int)
	for _, m := range memories {
		if !m.Success {
			errorCounts[m.EventType]++
		}
	}
	for evtType, count := range errorCounts {
		if count >= 3 {
			findings = append(findings, fmt.Sprintf(
				"Repeated error pattern: '%s' occurred %d times — capability gap",
				evtType, count))
		}
	}

	// Check for slow tasks
	var totalDuration time.Duration
	slowCount := 0
	for _, m := range memories {
		totalDuration += m.Duration
	}
	if len(memories) > 0 {
		avgDuration := totalDuration / time.Duration(len(memories))
		for _, m := range memories {
			if avgDuration > 0 && m.Duration > avgDuration*10 {
				slowCount++
			}
		}
	}
	if slowCount > 0 {
		findings = append(findings, fmt.Sprintf(
			"%d tasks took >10x average duration — efficiency improvement needed", slowCount))
	}

	return findings
}

func (s *GoalSynthesizer) validateProposal(pg ProposedGoal) error {
	if pg.Title == "" {
		return fmt.Errorf("title is required")
	}
	if pg.Tier < 1 || pg.Tier > 3 {
		return fmt.Errorf("tier must be 1-3, got %d", pg.Tier)
	}
	if pg.ExpectedValue < 0 || pg.ExpectedValue > 1 {
		return fmt.Errorf("expected_value must be 0.0-1.0, got %.2f", pg.ExpectedValue)
	}
	if pg.ParentTier0 == "" {
		return fmt.Errorf("parent_tier0 is required")
	}
	if len(pg.SuccessCriteria) == 0 {
		return fmt.Errorf("at least one success criterion is required")
	}

	// Validate parent_tier0 is a known Tier 0 goal
	validParent := false
	for _, t0 := range AllTier0Goals {
		if pg.ParentTier0 == t0.Title {
			validParent = true
			break
		}
	}
	if !validParent {
		return fmt.Errorf("unknown parent_tier0: %s", pg.ParentTier0)
	}

	return nil
}

// deduplicate removes goals that are too similar to existing active/pending goals.
// Uses word-overlap similarity (Jaccard) with threshold 0.85 as specified in the plan.
func (s *GoalSynthesizer) deduplicate(proposals []Goal) []Goal {
	existing, _ := s.store.GetByStatus(GoalStatusPending)
	active, _ := s.store.GetByStatus(GoalStatusActive)
	existing = append(existing, active...)

	var result []Goal
	for _, p := range proposals {
		isDuplicate := false
		pWords := tokenize(p.Title)
		for _, e := range existing {
			// Fast path: exact title match
			if p.Title == e.Title {
				isDuplicate = true
				break
			}
			// Similarity path: word-overlap Jaccard similarity
			eWords := tokenize(e.Title)
			if jaccardSimilarity(pWords, eWords) > 0.85 {
				isDuplicate = true
				break
			}
		}
		if !isDuplicate {
			result = append(result, p)
		}
	}
	return result
}

// tokenize normalizes a title into a set of lowercase words.
func tokenize(s string) map[string]struct{} {
	words := make(map[string]struct{})
	for _, w := range strings.Fields(strings.ToLower(s)) {
		w = strings.Trim(w, ".,;:!?\"'()[]{}")
		if w != "" {
			words[w] = struct{}{}
		}
	}
	return words
}

// jaccardSimilarity computes the Jaccard similarity between two word sets.
func jaccardSimilarity(a, b map[string]struct{}) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0
	}
	intersection := 0
	for word := range a {
		if _, ok := b[word]; ok {
			intersection++
		}
	}
	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func formatFindings(findings []string) string {
	s := ""
	for i, f := range findings {
		s += fmt.Sprintf("%d. %s\n", i+1, f)
	}
	return s
}

// StartScheduler runs Synthesize() on a background goroutine.
func (s *GoalSynthesizer) StartScheduler(
	ctx context.Context,
	memoriesFn func() []MemoryEntry,
	weaknessesFn func() []WeaknessSnapshot,
) func() {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(60 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				memories := memoriesFn()
				weaknesses := weaknessesFn()
				goals, err := s.Synthesize(ctx, memories, weaknesses)
				if err != nil {
					slog.Warn("ags synthesizer: scheduled synthesis failed", "error", err)
				} else if len(goals) > 0 {
					slog.Info("ags synthesizer: scheduled synthesis produced goals", "count", len(goals))
				}
			case <-stop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	return func() { close(stop) }
}

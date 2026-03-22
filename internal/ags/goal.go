package ags

import (
	"time"

	"github.com/google/uuid"
)

// GoalStatus is a typed string representing the lifecycle state of a goal.
type GoalStatus string

const (
	GoalStatusPending   GoalStatus = "pending"
	GoalStatusActive    GoalStatus = "active"
	GoalStatusCompleted GoalStatus = "completed"
	GoalStatusAbandoned GoalStatus = "abandoned"
)

// Tier 0 goals are hardcoded, immutable, and not stored in the database.
// They serve as the root of the goal hierarchy — every generated goal
// must trace back to one of these.
var (
	GoalBeUseful            = Tier0Goal{Title: "Be maximally useful to operators"}
	GoalImproveCapabilities = Tier0Goal{Title: "Improve own capabilities"}
	GoalMaintainContinuity  = Tier0Goal{Title: "Maintain operational continuity"}
	GoalExpandKnowledge     = Tier0Goal{Title: "Expand knowledge and world-model"}

	AllTier0Goals = []Tier0Goal{
		GoalBeUseful,
		GoalImproveCapabilities,
		GoalMaintainContinuity,
		GoalExpandKnowledge,
	}
)

// Tier0Goal is an immutable root goal.
type Tier0Goal struct {
	Title string
}

// Goal represents a single objective in the agent's goal hierarchy.
type Goal struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	Tier            int        `json:"tier"`     // 1-3 (Tier 0 is hardcoded, not stored)
	Priority        float64    `json:"priority"` // 0.0-1.0, recomputed each cycle
	Status          GoalStatus `json:"status"`
	ParentID        string     `json:"parent_id"` // parent Tier 0 goal title or parent goal ID
	ChildIDs        []string   `json:"child_ids"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Evidence        []string   `json:"evidence"` // why this goal was created
	SuccessCriteria []string   `json:"success_criteria"`
	ExpectedValue   float64    `json:"expected_value"` // 0.0-1.0, estimated value if completed
	AttemptCount    int        `json:"attempt_count"`
	LastAttempt     *time.Time `json:"last_attempt"`
	CompletedAt     *time.Time `json:"completed_at"`
	ActualValue     *float64   `json:"actual_value"` // filled in after completion
}

// NewGoal creates a new goal with defaults.
func NewGoal(title, description string, tier int, parentID string, evidence []string, criteria []string, expectedValue float64) Goal {
	now := time.Now().UTC()
	return Goal{
		ID:              uuid.New().String(),
		Title:           title,
		Description:     description,
		Tier:            tier,
		Priority:        0.5,
		Status:          GoalStatusPending,
		ParentID:        parentID,
		CreatedAt:       now,
		UpdatedAt:       now,
		Evidence:        evidence,
		SuccessCriteria: criteria,
		ExpectedValue:   expectedValue,
	}
}

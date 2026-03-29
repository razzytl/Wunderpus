package a2a

import "time"

// AgentCard describes an agent's capabilities for discovery.
// Implements Google's Agent2Agent (A2A) protocol.
type AgentCard struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
	Endpoint     string   `json:"endpoint"`
	Version      string   `json:"version"`
}

// Task represents a unit of work to be assigned to an agent.
type Task struct {
	ID                   string                 `json:"id"`
	Description          string                 `json:"description"`
	RequiredCapabilities []string               `json:"required_capabilities"`
	Input                map[string]interface{} `json:"input"`
	ExpectedOutput       string                 `json:"expected_output,omitempty"`
	Priority             int                    `json:"priority"` // 1=low, 5=high
	CreatedAt            time.Time              `json:"created_at"`
}

// TaskResult is the outcome of executing a task.
type TaskResult struct {
	TaskID   string        `json:"task_id"`
	Status   TaskStatus    `json:"status"`
	Output   string        `json:"output"`
	AgentID  string        `json:"agent_id"`
	Duration time.Duration `json:"duration"`
	Cost     float64       `json:"cost"`
	Error    string        `json:"error,omitempty"`
}

// TaskStatus represents the lifecycle state of a task.
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskAssigned  TaskStatus = "assigned"
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
	TaskTimedOut  TaskStatus = "timed_out"
)

// Bid is a response from an agent indicating it can handle a task.
type Bid struct {
	AgentID     string        `json:"agent_id"`
	AgentCard   AgentCard     `json:"agent_card"`
	TaskID      string        `json:"task_id"`
	Score       float64       `json:"score"` // capability match score 0-1
	EstCost     float64       `json:"estimated_cost"`
	EstDuration time.Duration `json:"estimated_duration"`
}

// A2AMessage is a message between agents in the A2A protocol.
type A2AMessage struct {
	Type      string      `json:"type"` // task_assign, task_bid, task_result, discover, advertise
	Payload   interface{} `json:"payload"`
	SenderID  string      `json:"sender_id"`
	Timestamp time.Time   `json:"timestamp"`
}

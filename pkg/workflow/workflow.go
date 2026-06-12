package workflow

import "time"

type StepType string

const (
	StepCapability StepType = "capability"
	StepDelay      StepType = "delay"
	StepCondition  StepType = "condition"
)

type Workflow struct {
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Steps       []Step    `json:"steps"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Step struct {
	Name       string         `json:"name"`
	Type       StepType       `json:"type"`
	Capability string         `json:"capability,omitempty"`
	Command    map[string]any `json:"command,omitempty"`
	Delay      string         `json:"delay,omitempty"`
	Condition  *Condition     `json:"condition,omitempty"`
	Loop       *Loop          `json:"loop,omitempty"`
}

type Condition struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value any    `json:"value"`
}

type Loop struct {
	Count int    `json:"count,omitempty"`
	Until string `json:"until,omitempty"`
}

type ExecutionState string

const (
	ExecutionStateRunning   ExecutionState = "running"
	ExecutionStateCompleted ExecutionState = "completed"
	ExecutionStateFailed    ExecutionState = "failed"
	ExecutionStateCancelled ExecutionState = "cancelled"
)

type Execution struct {
	ID        string          `json:"id"`
	Workflow  string          `json:"workflow"`
	State     ExecutionState  `json:"state"`
	StartedAt time.Time       `json:"started_at"`
	EndedAt   time.Time       `json:"ended_at,omitempty"`
	Output    map[string]any  `json:"output,omitempty"`
	Error     string          `json:"error,omitempty"`
	Steps     []StepExecution `json:"steps,omitempty"`
}

type StepExecution struct {
	Name      string         `json:"name"`
	State     ExecutionState `json:"state"`
	StartedAt time.Time      `json:"started_at"`
	EndedAt   time.Time      `json:"ended_at,omitempty"`
	Output    map[string]any `json:"output,omitempty"`
	Error     string         `json:"error,omitempty"`
}

func (w Workflow) Normalize() Workflow {
	now := time.Now().UTC()
	if w.CreatedAt.IsZero() {
		w.CreatedAt = now
	}
	w.UpdatedAt = now
	if w.Steps == nil {
		w.Steps = []Step{}
	}
	return w
}

package session

import (
	"fmt"
	"sync"
	"time"
)

// WorkflowEngine محرك سير العمل - يدير الـ 16 مرحلة
type WorkflowEngine struct {
	SessionID    string          `json:"session_id"`
	Phases       []WorkflowPhase `json:"phases"`
	CurrentPhase int             `json:"current_phase"`
	Progress     float64         `json:"progress"` // 0-100
	State        string          `json:"state"`    // idle, running, paused, completed
	StartedAt    time.Time       `json:"started_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	mu           sync.RWMutex
}

// WorkflowPhase مرحلة في سير العمل
type WorkflowPhase struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // pending, active, completed, failed
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Tasks       []Task    `json:"tasks"`
	Progress    float64   `json:"progress"` // 0-100
}

// Task مهمة في المرحلة
type Task struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Status      string        `json:"status"` // pending, assigned, in_progress, completed, failed
	AssignedTo  string        `json:"assigned_to"` // Agent DID
	Priority    int           `json:"priority"` // 1-10
	StartedAt   time.Time     `json:"started_at"`
	CompletedAt time.Time     `json:"completed_at"`
	Progress    float64       `json:"progress"` // 0-100
	Result      string        `json:"result,omitempty"`
	DependsOn   []string      `json:"depends_on"` // Task IDs
}

// NewWorkflowEngine ينشئ محرك وورك فلو جديد
func NewWorkflowEngine(sessionID string) *WorkflowEngine {
	return &WorkflowEngine{
		SessionID: sessionID,
		Phases:    make([]WorkflowPhase, 0),
		State:     "idle",
	}
}

// InitializePhases يهيئ المراحل
func (we *WorkflowEngine) InitializePhases(phases []WorkflowPhase) {
	we.mu.Lock()
	defer we.mu.Unlock()

	we.Phases = phases
	we.CurrentPhase = 0
	we.State = "initialized"
	we.StartedAt = time.Now()
	we.UpdatedAt = time.Now()
}

// StartPhase يبدأ مرحلة
func (we *WorkflowEngine) StartPhase(phaseIndex int) error {
	we.mu.Lock()
	defer we.mu.Unlock()

	if phaseIndex >= len(we.Phases) {
		return fmt.Errorf("phase index out of range")
	}

	we.Phases[phaseIndex].Status = "active"
	we.Phases[phaseIndex].StartedAt = time.Now()
	we.CurrentPhase = phaseIndex
	we.State = "running"
	we.UpdatedAt = time.Now()

	return nil
}

// CompletePhase يكمل مرحلة
func (we *WorkflowEngine) CompletePhase(phaseIndex int) error {
	we.mu.Lock()
	defer we.mu.Unlock()

	if phaseIndex >= len(we.Phases) {
		return fmt.Errorf("phase index out of range")
	}

	we.Phases[phaseIndex].Status = "completed"
	we.Phases[phaseIndex].CompletedAt = time.Now()
	we.Phases[phaseIndex].Progress = 100

	we.UpdatedAt = time.Now()
	we.calculateProgress()

	return nil
}

// AddTask يضيف مهمة لمرحلة
func (we *WorkflowEngine) AddTask(phaseIndex int, task Task) error {
	we.mu.Lock()
	defer we.mu.Unlock()

	if phaseIndex >= len(we.Phases) {
		return fmt.Errorf("phase index out of range")
	}

	task.ID = fmt.Sprintf("task_%d_%d", phaseIndex, len(we.Phases[phaseIndex].Tasks)+1)
	task.Status = "pending"
	task.StartedAt = time.Now()

	we.Phases[phaseIndex].Tasks = append(we.Phases[phaseIndex].Tasks, task)
	we.UpdatedAt = time.Now()

	return nil
}

// UpdateTaskStatus يحدث حالة مهمة
func (we *WorkflowEngine) UpdateTaskStatus(phaseIndex, taskIndex int, status string, progress float64) error {
	we.mu.Lock()
	defer we.mu.Unlock()

	if phaseIndex >= len(we.Phases) {
		return fmt.Errorf("phase index out of range")
	}

	if taskIndex >= len(we.Phases[phaseIndex].Tasks) {
		return fmt.Errorf("task index out of range")
	}

	we.Phases[phaseIndex].Tasks[taskIndex].Status = status
	we.Phases[phaseIndex].Tasks[taskIndex].Progress = progress

	if status == "completed" {
		we.Phases[phaseIndex].Tasks[taskIndex].CompletedAt = time.Now()
	}

	we.UpdatedAt = time.Now()
	we.calculateProgress()

	return nil
}

// calculateProgress يحسب التقدم العام
func (we *WorkflowEngine) calculateProgress() {
	totalTasks := 0
	completedTasks := 0

	for _, phase := range we.Phases {
		for _, task := range phase.Tasks {
			totalTasks++
			if task.Status == "completed" {
				completedTasks++
			}
		}
	}

	if totalTasks > 0 {
		we.Progress = float64(completedTasks) / float64(totalTasks) * 100
	}
}

// GetProgress يعيد التقدم العام
func (we *WorkflowEngine) GetProgress() float64 {
	we.mu.RLock()
	defer we.mu.RUnlock()
	return we.Progress
}

// GetCurrentPhase يعيد المرحلة الحالية
func (we *WorkflowEngine) GetCurrentPhase() *WorkflowPhase {
	we.mu.RLock()
	defer we.mu.RUnlock()

	if we.CurrentPhase >= len(we.Phases) {
		return nil
	}

	return &we.Phases[we.CurrentPhase]
}

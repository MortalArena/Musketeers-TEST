package domain

import "time"

// Task Domain Model - الكيان الأساسي للمهمة
type Task struct {
	ID          string
	Title       string
	Description string
	Status      TaskStatus
	Priority    TaskPriority
	SessionID   string
	AssignedTo  string // Agent ID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
}

// TaskStatus Value Object
type TaskStatus string

const (
	TaskStatusCreated    TaskStatus = "created"
	TaskStatusAssigned   TaskStatus = "assigned"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

// TaskPriority Value Object
type TaskPriority string

const (
	TaskPriorityLow    TaskPriority = "low"
	TaskPriorityMedium TaskPriority = "medium"
	TaskPriorityHigh   TaskPriority = "high"
	TaskPriorityUrgent TaskPriority = "urgent"
)

// IsValid يتحقق من صحة Task
func (t *Task) IsValid() bool {
	if t.ID == "" {
		return false
	}
	if t.Title == "" {
		return false
	}
	if t.Status == "" {
		return false
	}
	return true
}

// IsCompleted يتحقق مما إذا كانت المهمة مكتملة
func (t *Task) IsCompleted() bool {
	return t.Status == TaskStatusCompleted
}

// IsInProgress يتحقق مما إذا كانت المهمة قيد التنفيذ
func (t *Task) IsInProgress() bool {
	return t.Status == TaskStatusInProgress
}

// CanAssign يتحقق مما إذا كان يمكن تعيين المهمة
func (t *Task) CanAssign() bool {
	return t.Status == TaskStatusCreated
}

// Assign يعين المهمة لوكيل
func (t *Task) Assign(agentID string) {
	t.AssignedTo = agentID
	t.Status = TaskStatusAssigned
	t.UpdatedAt = time.Now()
}

// Complete يكمل المهمة
func (t *Task) Complete() {
	t.Status = TaskStatusCompleted
	now := time.Now()
	t.CompletedAt = &now
	t.UpdatedAt = now
}

// Fail يفشل المهمة
func (t *Task) Fail() {
	t.Status = TaskStatusFailed
	t.UpdatedAt = time.Now()
}

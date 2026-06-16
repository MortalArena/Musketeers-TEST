package tasks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// TaskPriority أولوية المهمة
type TaskPriority string

const (
	PriorityCritical TaskPriority = "critical"
	PriorityHigh     TaskPriority = "high"
	PriorityMedium   TaskPriority = "medium"
	PriorityLow      TaskPriority = "low"
)

// TaskStatus حالة المهمة
type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusInProgress TaskStatus = "in_progress"
	StatusCompleted TaskStatus = "completed"
	StatusFailed    TaskStatus = "failed"
	StatusBlocked   TaskStatus = "blocked"
)

// SubTask مهمة فرعية
type SubTask struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Priority    TaskPriority  `json:"priority"`
	Status      TaskStatus     `json:"status"`
	ParentID    string         `json:"parent_id"`
	Dependencies []string      `json:"dependencies"`
	AssignedTo  string         `json:"assigned_to"`
	EstimatedTime time.Duration `json:"estimated_time"`
	ActualTime   time.Duration `json:"actual_time"`
	CreatedAt   time.Time      `json:"created_at"`
	StartedAt   *time.Time     `json:"started_at,omitempty"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	Result      interface{}    `json:"result,omitempty"`
	Error       string         `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// TaskDecomposer مفكك المهام
type TaskDecomposer struct {
	tasks      map[string]*SubTask
	logger     *zap.Logger
	mu         sync.RWMutex
	sessionID  string
	agentID    string
}

// NewTaskDecomposer ينشئ مفكك مهام جديد
func NewTaskDecomposer(sessionID, agentID string, logger *zap.Logger) *TaskDecomposer {
	return &TaskDecomposer{
		tasks:     make(map[string]*SubTask),
		logger:    logger,
		sessionID: sessionID,
		agentID:   agentID,
	}
}

// DecomposeTask يفكك المهمة إلى مهام فرعية
func (td *TaskDecomposer) DecomposeTask(ctx context.Context, task string, complexity string) ([]*SubTask, error) {
	td.mu.Lock()
	defer td.mu.Unlock()

	// [WHY] تفكيك المهمة الكبيرة إلى مهام فرعية قابلة للتنفيذ
	// [HOW] يحلل المهمة ويقسمها إلى خطوات منطقية
	// [SAFETY] يضمن أن كل مهمة فرعية لها هدف واضح ومحدد

	var subTasks []*SubTask

	switch complexity {
	case "high":
		// تفكيك مهمة معقدة
		subTasks = []*SubTask{
			{
				ID:          fmt.Sprintf("subtask_%d", time.Now().UnixNano()),
				Title:       "تحليل المتطلبات",
				Description: "تحليل متطلبات المهمة وفهمها بالكامل",
				Priority:    PriorityCritical,
				Status:      StatusPending,
				Dependencies: []string{},
				EstimatedTime: 2 * time.Minute,
				CreatedAt:   time.Now(),
				Metadata:    map[string]interface{}{"phase": "analysis"},
			},
			{
				ID:          fmt.Sprintf("subtask_%d", time.Now().UnixNano()+1),
				Title:       "تخطيط التنفيذ",
				Description: "تخطيط خطوات التنفيذ المطلوبة",
				Priority:    PriorityHigh,
				Status:      StatusPending,
				Dependencies: []string{},
				EstimatedTime: 3 * time.Minute,
				CreatedAt:   time.Now(),
				Metadata:    map[string]interface{}{"phase": "planning"},
			},
			{
				ID:          fmt.Sprintf("subtask_%d", time.Now().UnixNano()+2),
				Title:       "تنفيذ الخطوات",
				Description: "تنفيذ الخطوات المخطط لها",
				Priority:    PriorityHigh,
				Status:      StatusPending,
				Dependencies: []string{},
				EstimatedTime: 5 * time.Minute,
				CreatedAt:   time.Now(),
				Metadata:    map[string]interface{}{"phase": "execution"},
			},
			{
				ID:          fmt.Sprintf("subtask_%d", time.Now().UnixNano()+3),
				Title:       "التحقق من النتائج",
				Description: "التحقق من صحة النتائج المكتملة",
				Priority:    PriorityMedium,
				Status:      StatusPending,
				Dependencies: []string{},
				EstimatedTime: 2 * time.Minute,
				CreatedAt:   time.Now(),
				Metadata:    map[string]interface{}{"phase": "verification"},
			},
		}
	case "medium":
		// تفكيك مهمة متوسطة
		subTasks = []*SubTask{
			{
				ID:          fmt.Sprintf("subtask_%d", time.Now().UnixNano()),
				Title:       "تحليل المهمة",
				Description: "تحليل المهمة وفهم المتطلبات",
				Priority:    PriorityHigh,
				Status:      StatusPending,
				Dependencies: []string{},
				EstimatedTime: 1 * time.Minute,
				CreatedAt:   time.Now(),
				Metadata:    map[string]interface{}{"phase": "analysis"},
			},
			{
				ID:          fmt.Sprintf("subtask_%d", time.Now().UnixNano()+1),
				Title:       "تنفيذ المهمة",
				Description: "تنفيذ المهمة المطلوبة",
				Priority:    PriorityHigh,
				Status:      StatusPending,
				Dependencies: []string{},
				EstimatedTime: 3 * time.Minute,
				CreatedAt:   time.Now(),
				Metadata:    map[string]interface{}{"phase": "execution"},
			},
		}
	default:
		// مهمة بسيطة
		subTasks = []*SubTask{
			{
				ID:          fmt.Sprintf("subtask_%d", time.Now().UnixNano()),
				Title:       "تنفيذ المهمة",
				Description: task,
				Priority:    PriorityMedium,
				Status:      StatusPending,
				Dependencies: []string{},
				EstimatedTime: 2 * time.Minute,
				CreatedAt:   time.Now(),
				Metadata:    map[string]interface{}{"phase": "execution"},
			},
		}
	}

	// تخزين المهام الفرعية
	for _, subTask := range subTasks {
		td.tasks[subTask.ID] = subTask
	}

	td.logger.Info("تم تفكيك المهمة",
		zap.String("session_id", td.sessionID),
		zap.String("agent_id", td.agentID),
		zap.Int("subtasks_count", len(subTasks)),
		zap.String("complexity", complexity),
	)

	return subTasks, nil
}

// AddSubTask يضيف مهمة فرعية
func (td *TaskDecomposer) AddSubTask(ctx context.Context, subTask *SubTask) error {
	td.mu.Lock()
	defer td.mu.Unlock()

	td.tasks[subTask.ID] = subTask

	td.logger.Info("تم إضافة مهمة فرعية",
		zap.String("session_id", td.sessionID),
		zap.String("agent_id", td.agentID),
		zap.String("subtask_id", subTask.ID),
		zap.String("title", subTask.Title),
	)

	return nil
}

// GetSubTask يرجع مهمة فرعية
func (td *TaskDecomposer) GetSubTask(ctx context.Context, subTaskID string) (*SubTask, error) {
	td.mu.RLock()
	defer td.mu.RUnlock()

	subTask, ok := td.tasks[subTaskID]
	if !ok {
		return nil, fmt.Errorf("مهمة فرعية غير موجودة: %s", subTaskID)
	}

	return subTask, nil
}

// GetAllSubTasks يرجع جميع المهام الفرعية
func (td *TaskDecomposer) GetAllSubTasks(ctx context.Context) ([]*SubTask, error) {
	td.mu.RLock()
	defer td.mu.RUnlock()

	subTasks := make([]*SubTask, 0, len(td.tasks))
	for _, subTask := range td.tasks {
		subTasks = append(subTasks, subTask)
	}

	return subTasks, nil
}

// GetSubTasksByStatus يرجع المهام الفرعية حسب الحالة
func (td *TaskDecomposer) GetSubTasksByStatus(ctx context.Context, status TaskStatus) ([]*SubTask, error) {
	td.mu.RLock()
	defer td.mu.RUnlock()

	var result []*SubTask
	for _, subTask := range td.tasks {
		if subTask.Status == status {
			result = append(result, subTask)
		}
	}

	return result, nil
}

// UpdateSubTaskStatus يحدث حالة مهمة فرعية
func (td *TaskDecomposer) UpdateSubTaskStatus(ctx context.Context, subTaskID string, status TaskStatus) error {
	td.mu.Lock()
	defer td.mu.Unlock()

	subTask, ok := td.tasks[subTaskID]
	if !ok {
		return fmt.Errorf("مهمة فرعية غير موجودة: %s", subTaskID)
	}

	oldStatus := subTask.Status
	subTask.Status = status

	now := time.Now()
	if status == StatusInProgress && subTask.StartedAt == nil {
		subTask.StartedAt = &now
	}
	if status == StatusCompleted && subTask.CompletedAt == nil {
		subTask.CompletedAt = &now
	}

	td.logger.Info("تم تحديث حالة المهمة الفرعية",
		zap.String("session_id", td.sessionID),
		zap.String("agent_id", td.agentID),
		zap.String("subtask_id", subTaskID),
		zap.String("old_status", string(oldStatus)),
		zap.String("new_status", string(status)),
	)

	return nil
}

// AssignSubTask يخصص مهمة فرعية لوكيل
func (td *TaskDecomposer) AssignSubTask(ctx context.Context, subTaskID, agentID string) error {
	td.mu.Lock()
	defer td.mu.Unlock()

	subTask, ok := td.tasks[subTaskID]
	if !ok {
		return fmt.Errorf("مهمة فرعية غير موجودة: %s", subTaskID)
	}

	subTask.AssignedTo = agentID

	td.logger.Info("تم تخصيص المهمة الفرعية",
		zap.String("session_id", td.sessionID),
		zap.String("subtask_id", subTaskID),
		zap.String("assigned_to", agentID),
	)

	return nil
}

// SetSubTaskResult يضبط نتيجة مهمة فرعية
func (td *TaskDecomposer) SetSubTaskResult(ctx context.Context, subTaskID string, result interface{}, err error) error {
	td.mu.Lock()
	defer td.mu.Unlock()

	subTask, ok := td.tasks[subTaskID]
	if !ok {
		return fmt.Errorf("مهمة فرعية غير موجودة: %s", subTaskID)
	}

	subTask.Result = result
	if err != nil {
		subTask.Error = err.Error()
		subTask.Status = StatusFailed
	} else {
		subTask.Status = StatusCompleted
		now := time.Now()
		subTask.CompletedAt = &now
	}

	td.logger.Info("تم ضبط نتيجة المهمة الفرعية",
		zap.String("session_id", td.sessionID),
		zap.String("subtask_id", subTaskID),
		zap.Bool("success", err == nil),
	)

	return nil
}

// GetPendingSubTasks يرجع المهام الفرعية المعلقة
func (td *TaskDecomposer) GetPendingSubTasks(ctx context.Context) ([]*SubTask, error) {
	return td.GetSubTasksByStatus(ctx, StatusPending)
}

// GetInProgressSubTasks يرجع المهام الفرعية قيد التنفيذ
func (td *TaskDecomposer) GetInProgressSubTasks(ctx context.Context) ([]*SubTask, error) {
	return td.GetSubTasksByStatus(ctx, StatusInProgress)
}

// GetCompletedSubTasks يرجع المهام الفرعية المكتملة
func (td *TaskDecomposer) GetCompletedSubTasks(ctx context.Context) ([]*SubTask, error) {
	return td.GetSubTasksByStatus(ctx, StatusCompleted)
}

// GetFailedSubTasks يرجع المهام الفرعية الفاشلة
func (td *TaskDecomposer) GetFailedSubTasks(ctx context.Context) ([]*SubTask, error) {
	return td.GetSubTasksByStatus(ctx, StatusFailed)
}

// GetProgress يحسب تقدم المهام
func (td *TaskDecomposer) GetProgress(ctx context.Context) (map[string]interface{}, error) {
	td.mu.RLock()
	defer td.mu.RUnlock()

	total := len(td.tasks)
	completed := 0
	failed := 0
	inProgress := 0
	pending := 0

	for _, subTask := range td.tasks {
		switch subTask.Status {
		case StatusCompleted:
			completed++
		case StatusFailed:
			failed++
		case StatusInProgress:
			inProgress++
		case StatusPending:
			pending++
		}
	}

	progress := map[string]interface{}{
		"total":       total,
		"completed":   completed,
		"failed":      failed,
		"in_progress": inProgress,
		"pending":     pending,
		"percentage":  0.0,
	}

	if total > 0 {
		progress["percentage"] = float64(completed) / float64(total) * 100
	}

	return progress, nil
}

// CheckDependencies يتحقق من تلبية الاعتمادات
func (td *TaskDecomposer) CheckDependencies(ctx context.Context, subTaskID string) (bool, error) {
	td.mu.RLock()
	defer td.mu.RUnlock()

	subTask, ok := td.tasks[subTaskID]
	if !ok {
		return false, fmt.Errorf("مهمة فرعية غير موجودة: %s", subTaskID)
	}

	for _, depID := range subTask.Dependencies {
		depTask, ok := td.tasks[depID]
		if !ok {
			return false, fmt.Errorf("اعتماد غير موجود: %s", depID)
		}
		if depTask.Status != StatusCompleted {
			return false, nil
		}
	}

	return true, nil
}

// GetNextSubTask يرجع المهمة الفرعية التالية القابلة للتنفيذ
func (td *TaskDecomposer) GetNextSubTask(ctx context.Context) (*SubTask, error) {
	td.mu.RLock()
	defer td.mu.RUnlock()

	for _, subTask := range td.tasks {
		if subTask.Status == StatusPending {
			// التحقق من الاعتمادات
			dependenciesMet := true
			for _, depID := range subTask.Dependencies {
				depTask, ok := td.tasks[depID]
				if !ok || depTask.Status != StatusCompleted {
					dependenciesMet = false
					break
				}
			}

			if dependenciesMet {
				return subTask, nil
			}
		}
	}

	return nil, fmt.Errorf("لا توجد مهام فرعية قابلة للتنفيذ")
}

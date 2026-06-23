package session

import (
	"container/heap"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TaskPriority أولوية المهمة
type TaskPriority int

const (
	PriorityLow    TaskPriority = 1
	PriorityMedium TaskPriority = 2
	PriorityHigh   TaskPriority = 3
	PriorityUrgent TaskPriority = 4
)

// [SAFETY] حدود الموارد لمنع استهلاك غير محدود
const (
	// [SAFETY] الحد الأقصى لعدد المهام في قائمة الانتظار
	MaxPendingTasks = 1000
	// [SAFETY] الحد الأقصى لعدد المهام الجارية
	MaxRunningTasks = 50
	// [SAFETY] الحد الأقصى لمدة المهمة
	MaxTaskTimeout = 24 * time.Hour
	// [SAFETY] الحد الأقصى لعنوان المهمة
	MaxTaskTitleLength = 200
	// [SAFETY] الحد الأقصى لوصف المهمة
	MaxTaskDescriptionLength = 2000
	// [SAFETY] الحد الأقصى لعدد الوكلاء
	MaxAgents = 100
)

// TaskStatus حالة المهمة
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusAssigned  TaskStatus = "assigned"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// ManagedTask مهمة مُدارة في النظام
type ManagedTask struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Priority    TaskPriority           `json:"priority"`
	Status      TaskStatus             `json:"status"`
	AgentID     string                 `json:"agent_id,omitempty"`
	Inputs      map[string]interface{} `json:"inputs"`
	Outputs     map[string]interface{} `json:"outputs"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Timeout     time.Duration          `json:"timeout"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// TaskHeap كومة الأولويات للمهام
type TaskHeap []*ManagedTask

func (h TaskHeap) Len() int           { return len(h) }
func (h TaskHeap) Less(i, j int) bool { return h[i].Priority > h[j].Priority }
func (h TaskHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *TaskHeap) Push(x interface{}) {
	*h = append(*h, x.(*ManagedTask))
}

func (h *TaskHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// AgentState حالة الوكيل
type AgentState struct {
	AgentID      string                  `json:"agent_id"`
	CurrentTask  string                  `json:"current_task,omitempty"`
	Status       string                  `json:"status"` // idle, busy, offline
	Load         int                     `json:"load"`   // 0-100
	LastSeen     time.Time               `json:"last_seen"`
	SuccessRate  float64                 `json:"success_rate"`
	TotalTasks   int                     `json:"total_tasks"`
	FailedTasks  int                     `json:"failed_tasks"`
	Capabilities []agent.AgentCapability `json:"capabilities"`
}

// TaskManager مدير المهام - يدير قوائم المهام وتوزيعها على الوكلاء
type TaskManager struct {
	sessionID string
	logger    *zap.Logger
	eventBus  *eventbus.EventBus

	// قوائم المهام
	pendingQueue   *TaskHeap
	runningTasks   map[string]*ManagedTask
	completedTasks map[string]*ManagedTask
	failedTasks    map[string]*ManagedTask

	// حالة الوكلاء
	agentStates map[string]*AgentState

	mu sync.RWMutex
}

// NewTaskManager ينشئ مدير مهام جديد
func NewTaskManager(sessionID string) *TaskManager {
	h := &TaskHeap{}
	heap.Init(h)

	return &TaskManager{
		sessionID:      sessionID,
		logger:         zap.NewNop(), // سيتم استبداله بـ logger حقيقي
		eventBus:       nil,          // سيتم تعيينه لاحقاً
		pendingQueue:   h,
		runningTasks:   make(map[string]*ManagedTask),
		completedTasks: make(map[string]*ManagedTask),
		failedTasks:    make(map[string]*ManagedTask),
		agentStates:    make(map[string]*AgentState),
	}
}

// SetLogger يضبط logger
func (tm *TaskManager) SetLogger(logger *zap.Logger) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.logger = logger
}

// SetEventBus يضبط event bus
func (tm *TaskManager) SetEventBus(eb *eventbus.EventBus) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.eventBus = eb
}

// CreateTask ينشئ مهمة جديدة
func (tm *TaskManager) CreateTask(ctx context.Context, title, description string, priority TaskPriority, inputs map[string]interface{}, timeout time.Duration) (*ManagedTask, error) {
	// [SAFETY] التحقق من صحة المدخلات
	if title == "" {
		return nil, fmt.Errorf("task title cannot be empty")
	}
	if len(title) > MaxTaskTitleLength {
		return nil, fmt.Errorf("task title too long (max %d characters)", MaxTaskTitleLength)
	}
	if len(description) > MaxTaskDescriptionLength {
		return nil, fmt.Errorf("task description too long (max %d characters)", MaxTaskDescriptionLength)
	}
	if priority < PriorityLow || priority > PriorityUrgent {
		return nil, fmt.Errorf("invalid priority (must be between %d and %d)", PriorityLow, PriorityUrgent)
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("timeout must be positive")
	}
	if timeout > MaxTaskTimeout {
		return nil, fmt.Errorf("timeout too long (max %v)", MaxTaskTimeout)
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// [SAFETY] التحقق من الحد الأقصى لقائمة الانتظار
	if tm.pendingQueue.Len() >= MaxPendingTasks {
		return nil, fmt.Errorf("maximum pending tasks limit reached (%d)", MaxPendingTasks)
	}

	task := &ManagedTask{
		ID:          fmt.Sprintf("task_%s", uuid.New().String()),
		Title:       title,
		Description: description,
		Priority:    priority,
		Status:      TaskStatusPending,
		Inputs:      inputs,
		Outputs:     make(map[string]interface{}),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Timeout:     timeout,
		Metadata:    make(map[string]interface{}),
	}

	heap.Push(tm.pendingQueue, task)

	tm.logger.Info("Task created",
		zap.String("task_id", task.ID),
		zap.String("title", task.Title),
		zap.Int("priority", int(priority)),
	)

	if tm.eventBus != nil {
		tm.eventBus.Publish(eventbus.Event{
			Type:      "task.created",
			Payload:   task,
			Source:    "task_manager",
			SessionID: tm.sessionID,
		})
	}

	return task, nil
}

// AssignTask يعين مهمة لوكيل
func (tm *TaskManager) AssignTask(ctx context.Context, taskID, agentID string) error {
	// [SAFETY] التحقق من صحة المدخلات
	if taskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}
	if agentID == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// [SAFETY] التحقق من الحد الأقصى للمهام الجارية
	if len(tm.runningTasks) >= MaxRunningTasks {
		return fmt.Errorf("maximum running tasks limit reached (%d)", MaxRunningTasks)
	}

	// البحث عن المهمة في قائمة الانتظار
	var task *ManagedTask
	for _, t := range *tm.pendingQueue {
		if t.ID == taskID {
			task = t
			break
		}
	}

	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// تحديث حالة المهمة
	task.Status = TaskStatusAssigned
	task.AgentID = agentID
	task.UpdatedAt = time.Now()

	// إزالة من قائمة الانتظار وإضافتها للقائمة الجارية
	tm.runningTasks[taskID] = task

	tm.logger.Info("Task assigned",
		zap.String("task_id", task.ID),
		zap.String("agent_id", agentID),
	)

	if tm.eventBus != nil {
		tm.eventBus.Publish(eventbus.Event{
			Type: "task.assigned",
			Payload: map[string]string{
				"task_id":  taskID,
				"agent_id": agentID,
			},
			Source:    "task_manager",
			SessionID: tm.sessionID,
		})
	}

	return nil
}

// StartTask يبدأ تنفيذ مهمة
func (tm *TaskManager) StartTask(ctx context.Context, taskID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.runningTasks[taskID]
	if !exists {
		return fmt.Errorf("task not found in running tasks: %s", taskID)
	}

	task.Status = TaskStatusRunning
	now := time.Now()
	task.StartedAt = &now
	task.UpdatedAt = now

	tm.logger.Info("Task started",
		zap.String("task_id", task.ID),
		zap.String("agent_id", task.AgentID),
	)

	if tm.eventBus != nil {
		tm.eventBus.Publish(eventbus.Event{
			Type:      "task.started",
			Payload:   task,
			Source:    "task_manager",
			SessionID: tm.sessionID,
		})
	}

	return nil
}

// CompleteTask يكمل مهمة
func (tm *TaskManager) CompleteTask(ctx context.Context, taskID string, outputs map[string]interface{}) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.runningTasks[taskID]
	if !exists {
		return fmt.Errorf("task not found in running tasks: %s", taskID)
	}

	task.Status = TaskStatusCompleted
	task.Outputs = outputs
	now := time.Now()
	task.CompletedAt = &now
	task.UpdatedAt = now

	// نقل إلى المهام المكتملة
	tm.completedTasks[taskID] = task
	delete(tm.runningTasks, taskID)

	// تحديث حالة الوكيل
	if agentState, exists := tm.agentStates[task.AgentID]; exists {
		agentState.TotalTasks++
		agentState.CurrentTask = ""
		agentState.Load = max(0, agentState.Load-10)
		agentState.LastSeen = now
	}

	tm.logger.Info("Task completed",
		zap.String("task_id", task.ID),
		zap.String("agent_id", task.AgentID),
		zap.Duration("duration", now.Sub(*task.StartedAt)),
	)

	if tm.eventBus != nil {
		tm.eventBus.Publish(eventbus.Event{
			Type:      "task.completed",
			Payload:   task,
			Source:    "task_manager",
			SessionID: tm.sessionID,
		})
	}

	return nil
}

// FailTask يفشل مهمة
func (tm *TaskManager) FailTask(ctx context.Context, taskID, errorMsg string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.runningTasks[taskID]
	if !exists {
		return fmt.Errorf("task not found in running tasks: %s", taskID)
	}

	task.Status = TaskStatusFailed
	task.Metadata["error"] = errorMsg
	task.UpdatedAt = time.Now()

	// نقل إلى المهام الفاشلة
	tm.failedTasks[taskID] = task
	delete(tm.runningTasks, taskID)

	// تحديث حالة الوكيل
	if agentState, exists := tm.agentStates[task.AgentID]; exists {
		agentState.TotalTasks++
		agentState.FailedTasks++
		agentState.CurrentTask = ""
		agentState.Load = max(0, agentState.Load-10)
		agentState.LastSeen = time.Now()
		agentState.SuccessRate = float64(agentState.TotalTasks-agentState.FailedTasks) / float64(agentState.TotalTasks)
	}

	tm.logger.Error("Task failed",
		zap.String("task_id", task.ID),
		zap.String("agent_id", task.AgentID),
		zap.String("error", errorMsg),
	)

	if tm.eventBus != nil {
		tm.eventBus.Publish(eventbus.Event{
			Type: "task.failed",
			Payload: map[string]string{
				"task_id": taskID,
				"error":   errorMsg,
			},
			Source:    "task_manager",
			SessionID: tm.sessionID,
		})
	}

	return nil
}

// CancelTask يلغي مهمة
func (tm *TaskManager) CancelTask(ctx context.Context, taskID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// البحث في قائمة الانتظار
	var found bool
	for i, t := range *tm.pendingQueue {
		if t.ID == taskID {
			t.Status = TaskStatusCancelled
			t.UpdatedAt = time.Now()
			heap.Remove(tm.pendingQueue, i)
			found = true
			break
		}
	}

	// البحث في المهام الجارية
	if !found {
		if task, exists := tm.runningTasks[taskID]; exists {
			task.Status = TaskStatusCancelled
			task.UpdatedAt = time.Now()
			delete(tm.runningTasks, taskID)
			found = true

			// تحديث حالة الوكيل
			if agentState, exists := tm.agentStates[task.AgentID]; exists {
				agentState.CurrentTask = ""
				agentState.Load = max(0, agentState.Load-10)
			}
		}
	}

	if !found {
		return fmt.Errorf("task not found: %s", taskID)
	}

	tm.logger.Info("Task cancelled",
		zap.String("task_id", taskID),
	)

	if tm.eventBus != nil {
		tm.eventBus.Publish(eventbus.Event{
			Type:      "task.cancelled",
			Payload:   taskID,
			Source:    "task_manager",
			SessionID: tm.sessionID,
		})
	}

	return nil
}

// GetTask يحصل على مهمة
func (tm *TaskManager) GetTask(taskID string) (*ManagedTask, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// البحث في جميع القوائم
	for _, t := range *tm.pendingQueue {
		if t.ID == taskID {
			return t, nil
		}
	}

	if task, exists := tm.runningTasks[taskID]; exists {
		return task, nil
	}

	if task, exists := tm.completedTasks[taskID]; exists {
		return task, nil
	}

	if task, exists := tm.failedTasks[taskID]; exists {
		return task, nil
	}

	return nil, fmt.Errorf("task not found: %s", taskID)
}

// GetNextTask يحصل على المهمة التالية من قائمة الانتظار
func (tm *TaskManager) GetNextTask() *ManagedTask {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.pendingQueue.Len() == 0 {
		return nil
	}

	return (*tm.pendingQueue)[0]
}

// RegisterAgent يسجل وكيل
func (tm *TaskManager) RegisterAgent(agentID string, capabilities []agent.AgentCapability) error {
	// [SAFETY] التحقق من صحة المدخلات
	if agentID == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// [SAFETY] التحقق من الحد الأقصى للوكلاء
	if len(tm.agentStates) >= MaxAgents {
		return fmt.Errorf("maximum agents limit reached (%d)", MaxAgents)
	}

	tm.agentStates[agentID] = &AgentState{
		AgentID:      agentID,
		Status:       "idle",
		Load:         0,
		LastSeen:     time.Now(),
		SuccessRate:  1.0,
		TotalTasks:   0,
		FailedTasks:  0,
		Capabilities: capabilities,
	}

	tm.logger.Info("Agent registered",
		zap.String("agent_id", agentID),
		zap.Int("capabilities", len(capabilities)),
	)

	if tm.eventBus != nil {
		tm.eventBus.Publish(eventbus.Event{
			Type: "agent.registered",
			Payload: map[string]interface{}{
				"agent_id":     agentID,
				"capabilities": capabilities,
			},
			Source:    "task_manager",
			SessionID: tm.sessionID,
		})
	}

	return nil
}

// UnregisterAgent يلغى تسجيل وكيل
func (tm *TaskManager) UnregisterAgent(agentID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	delete(tm.agentStates, agentID)

	tm.logger.Info("Agent unregistered",
		zap.String("agent_id", agentID),
	)

	if tm.eventBus != nil {
		tm.eventBus.Publish(eventbus.Event{
			Type:      "agent.unregistered",
			Payload:   agentID,
			Source:    "task_manager",
			SessionID: tm.sessionID,
		})
	}
}

// GetAgentState يحصل على حالة وكيل
func (tm *TaskManager) GetAgentState(agentID string) (*AgentState, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	state, exists := tm.agentStates[agentID]
	if !exists {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	// إنشاء نسخة لتجنب التعديل الخارجي
	stateCopy := *state
	return &stateCopy, nil
}

// UpdateAgentLoad يحدث حمل الوكيل
func (tm *TaskManager) UpdateAgentLoad(agentID string, load int) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if state, exists := tm.agentStates[agentID]; exists {
		state.Load = load
		state.LastSeen = time.Now()

		if load > 80 {
			state.Status = "busy"
		} else {
			state.Status = "idle"
		}
	}
}

// GetStats يحصل على إحصائيات
func (tm *TaskManager) GetStats() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return map[string]interface{}{
		"pending_count":   tm.pendingQueue.Len(),
		"running_count":   len(tm.runningTasks),
		"completed_count": len(tm.completedTasks),
		"failed_count":    len(tm.failedTasks),
		"agent_count":     len(tm.agentStates),
	}
}

// Save يحفظ حالة TaskManager
func (tm *TaskManager) Save() ([]byte, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	data := struct {
		PendingTasks   []*ManagedTask          `json:"pending_tasks"`
		RunningTasks   map[string]*ManagedTask `json:"running_tasks"`
		CompletedTasks map[string]*ManagedTask `json:"completed_tasks"`
		FailedTasks    map[string]*ManagedTask `json:"failed_tasks"`
		AgentStates    map[string]*AgentState  `json:"agent_states"`
	}{
		PendingTasks:   make([]*ManagedTask, tm.pendingQueue.Len()),
		RunningTasks:   tm.runningTasks,
		CompletedTasks: tm.completedTasks,
		FailedTasks:    tm.failedTasks,
		AgentStates:    tm.agentStates,
	}

	copy(data.PendingTasks, *tm.pendingQueue)

	return json.Marshal(data)
}

// Load يحمل حالة TaskManager
func (tm *TaskManager) Load(data []byte) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	var loaded struct {
		PendingTasks   []*ManagedTask          `json:"pending_tasks"`
		RunningTasks   map[string]*ManagedTask `json:"running_tasks"`
		CompletedTasks map[string]*ManagedTask `json:"completed_tasks"`
		FailedTasks    map[string]*ManagedTask `json:"failed_tasks"`
		AgentStates    map[string]*AgentState  `json:"agent_states"`
	}

	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}

	// إعادة بناء قائمة الانتظار
	tm.pendingQueue = &TaskHeap{}
	for _, t := range loaded.PendingTasks {
		heap.Push(tm.pendingQueue, t)
	}

	tm.runningTasks = loaded.RunningTasks
	tm.completedTasks = loaded.CompletedTasks
	tm.failedTasks = loaded.FailedTasks
	tm.agentStates = loaded.AgentStates

	return nil
}

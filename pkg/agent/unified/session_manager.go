package unified

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SessionManager مدير الجلسة المتطور
type SessionManager struct {
	sessionID string
	logger    *zap.Logger
	mu        sync.RWMutex

	// معلومات الجلسة
	clientPrompt        string
	sessionStartTime    time.Time
	sessionStatus       SessionStatus
	activeAgents        []string
	sessionManagerAgent string // وكيل مدير الجلسة

	// إدارة المهام
	taskDistributionStrategy TaskDistributionStrategy
	activeTasks              map[string]*SessionTask
	taskHistory              []*SessionTask

	// مزامنة لحظية
	memorySync *RealTimeMemorySync
	skillSync  *RealTimeSkillSync
	eventBus   *SessionEventBus

	// مجدول المهام
	taskScheduler *TaskScheduler

	// منفذ المهام (بدلاً من unifiedAgent لتجنب الدورة المرجعية)
	agentExecutor AgentExecutor
}

// SessionStatus حالة الجلسة
type SessionStatus string

const (
	SessionStatusInitializing SessionStatus = "initializing"
	SessionStatusActive       SessionStatus = "active"
	SessionStatusPaused       SessionStatus = "paused"
	SessionStatusCompleted    SessionStatus = "completed"
	SessionStatusFailed       SessionStatus = "failed"
)

// SessionTask مهمة في الجلسة
type SessionTask struct {
	ID           string
	Description  string
	AssignedTo   string
	Status       TaskStatus
	Strategy     TaskDistributionStrategy
	Dependencies []string
	CreatedAt    time.Time
	StartedAt    *time.Time
	CompletedAt  *time.Time
	Result       interface{}
	Error        error
}

// TaskStatus حالة المهمة
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

// TaskDistributionStrategy استراتيجية توزيع المهام
type TaskDistributionStrategy string

const (
	StrategyConcurrent TaskDistributionStrategy = "concurrent" // تزامن
	StrategySequential TaskDistributionStrategy = "sequential" // دور
	StrategyMixed      TaskDistributionStrategy = "mixed"      // مختلط
)

// TaskComplexity تعقيد المهمة
type TaskComplexity string

const (
	ComplexityLow      TaskComplexity = "low"
	ComplexityMedium   TaskComplexity = "medium"
	ComplexityHigh     TaskComplexity = "high"
	ComplexityCritical TaskComplexity = "critical"
)

// TaskEvaluation تقييم المهمة
type TaskEvaluation struct {
	SessionID           string
	Prompt              string
	Complexity          TaskComplexity
	EstimatedTime       time.Duration
	RequiredAgents      []string
	RecommendedStrategy TaskDistributionStrategy
}

// NewSessionManager ينشئ مدير جلسة جديد
func NewSessionManager(sessionID string, logger *zap.Logger) *SessionManager {
	return &SessionManager{
		sessionID:                sessionID,
		logger:                   logger,
		sessionStatus:            SessionStatusInitializing,
		taskDistributionStrategy: StrategyMixed,
		activeTasks:              make(map[string]*SessionTask),
		taskHistory:              []*SessionTask{},
		memorySync:               NewRealTimeMemorySync(sessionID, logger),
		skillSync:                NewRealTimeSkillSync(sessionID, logger),
		eventBus:                 NewSessionEventBus(sessionID, logger),
		taskScheduler:            NewTaskScheduler(sessionID, logger),
	}
}

// Initialize يهيئ مدير الجلسة
func (sm *SessionManager) Initialize(ctx context.Context, agentExecutor AgentExecutor) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.agentExecutor = agentExecutor
	sm.sessionStartTime = time.Now()
	sm.sessionStatus = SessionStatusActive

	// بدء ناقل الأحداث
	sm.eventBus.Start(ctx)

	// بدء مزامنة الذاكرة
	sm.memorySync.StartSync(ctx)

	// بدء مزامنة المهارات
	sm.skillSync.StartSync(ctx)

	// بدء مجدول المهام
	sm.taskScheduler.Start(ctx)

	sm.logger.Info("تم تهيئة مدير الجلسة",
		zap.String("session_id", sm.sessionID),
		zap.Time("start_time", sm.sessionStartTime))

	return nil
}

// ReceivePrompt يستقبل البرومبت من العميل
func (sm *SessionManager) ReceivePrompt(ctx context.Context, prompt string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.clientPrompt = prompt
	sm.logger.Info("تم استقبال البرومبت من العميل",
		zap.String("session_id", sm.sessionID),
		zap.String("prompt", prompt))

	return nil
}

// EvaluateTask يقيم المهمة
func (sm *SessionManager) EvaluateTask(ctx context.Context) (*TaskEvaluation, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	evaluation := &TaskEvaluation{
		SessionID:           sm.sessionID,
		Prompt:              sm.clientPrompt,
		Complexity:          sm.evaluateComplexity(),
		EstimatedTime:       sm.estimateTime(),
		RequiredAgents:      sm.determineRequiredAgents(),
		RecommendedStrategy: sm.recommendStrategy(),
	}

	sm.logger.Info("تم تقييم المهمة",
		zap.String("session_id", sm.sessionID),
		zap.String("complexity", string(evaluation.Complexity)),
		zap.Int("required_agents", len(evaluation.RequiredAgents)),
		zap.String("strategy", string(evaluation.RecommendedStrategy)))

	return evaluation, nil
}

// DecomposeTask يفكك المهمة إلى مهام فرعية
func (sm *SessionManager) DecomposeTask(ctx context.Context, evaluation *TaskEvaluation) ([]*SessionTask, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	tasks := sm.createSubtasks(evaluation)

	sm.logger.Info("تم تفكيك المهمة",
		zap.String("session_id", sm.sessionID),
		zap.Int("total_tasks", len(tasks)))

	return tasks, nil
}

// createSubtasks ينشئ مهام فرعية
func (sm *SessionManager) createSubtasks(evaluation *TaskEvaluation) []*SessionTask {
	tasks := []*SessionTask{}

	switch evaluation.Complexity {
	case ComplexityLow:
		tasks = sm.createSimpleTasks()
	case ComplexityMedium:
		tasks = sm.createMediumTasks()
	case ComplexityHigh:
		tasks = sm.createComplexTasks()
	case ComplexityCritical:
		tasks = sm.createCriticalTasks()
	}

	return tasks
}

// createSimpleTasks ينشئ مهام بسيطة
func (sm *SessionManager) createSimpleTasks() []*SessionTask {
	return []*SessionTask{
		{
			ID:          generateID(),
			Description: "تنفيذ المهمة البسيطة",
			Status:      TaskStatusPending,
			Strategy:    StrategySequential,
			CreatedAt:   time.Now(),
		},
	}
}

// createMediumTasks ينشئ مهام متوسطة
func (sm *SessionManager) createMediumTasks() []*SessionTask {
	return []*SessionTask{
		{
			ID:          generateID(),
			Description: "تحليل المتطلبات",
			Status:      TaskStatusPending,
			Strategy:    StrategySequential,
			CreatedAt:   time.Now(),
		},
		{
			ID:           generateID(),
			Description:  "تنفيذ الحل",
			Status:       TaskStatusPending,
			Strategy:     StrategySequential,
			Dependencies: []string{},
			CreatedAt:    time.Now(),
		},
	}
}

// createComplexTasks ينشئ مهام معقدة
func (sm *SessionManager) createComplexTasks() []*SessionTask {
	return []*SessionTask{
		{
			ID:          generateID(),
			Description: "تحليل المتطلبات",
			Status:      TaskStatusPending,
			Strategy:    StrategySequential,
			CreatedAt:   time.Now(),
		},
		{
			ID:           generateID(),
			Description:  "تصميم الحل",
			Status:       TaskStatusPending,
			Strategy:     StrategySequential,
			Dependencies: []string{},
			CreatedAt:    time.Now(),
		},
		{
			ID:           generateID(),
			Description:  "تنفيذ الواجهة",
			Status:       TaskStatusPending,
			Strategy:     StrategyConcurrent,
			Dependencies: []string{},
			CreatedAt:    time.Now(),
		},
		{
			ID:           generateID(),
			Description:  "تنفيذ المنطق",
			Status:       TaskStatusPending,
			Strategy:     StrategyConcurrent,
			Dependencies: []string{},
			CreatedAt:    time.Now(),
		},
		{
			ID:           generateID(),
			Description:  "الاختبار والتكامل",
			Status:       TaskStatusPending,
			Strategy:     StrategySequential,
			Dependencies: []string{},
			CreatedAt:    time.Now(),
		},
	}
}

// createCriticalTasks ينشئ مهام حرجة
func (sm *SessionManager) createCriticalTasks() []*SessionTask {
	return []*SessionTask{
		{
			ID:          generateID(),
			Description: "تحليل شامل للمتطلبات",
			Status:      TaskStatusPending,
			Strategy:    StrategySequential,
			CreatedAt:   time.Now(),
		},
		{
			ID:           generateID(),
			Description:  "تصميم معماري",
			Status:       TaskStatusPending,
			Strategy:     StrategySequential,
			Dependencies: []string{},
			CreatedAt:    time.Now(),
		},
		{
			ID:           generateID(),
			Description:  "تنفيذ الواجهات",
			Status:       TaskStatusPending,
			Strategy:     StrategyConcurrent,
			Dependencies: []string{},
			CreatedAt:    time.Now(),
		},
		{
			ID:           generateID(),
			Description:  "تنفيذ الخدمات",
			Status:       TaskStatusPending,
			Strategy:     StrategyConcurrent,
			Dependencies: []string{},
			CreatedAt:    time.Now(),
		},
		{
			ID:           generateID(),
			Description:  "تنفيذ قاعدة البيانات",
			Status:       TaskStatusPending,
			Strategy:     StrategyConcurrent,
			Dependencies: []string{},
			CreatedAt:    time.Now(),
		},
		{
			ID:           generateID(),
			Description:  "الاختبار الشامل",
			Status:       TaskStatusPending,
			Strategy:     StrategySequential,
			Dependencies: []string{},
			CreatedAt:    time.Now(),
		},
		{
			ID:           generateID(),
			Description:  "النشر والتكامل",
			Status:       TaskStatusPending,
			Strategy:     StrategySequential,
			Dependencies: []string{},
			CreatedAt:    time.Now(),
		},
	}
}

// DistributeTasks يوزع المهام على الوكلاء
func (sm *SessionManager) DistributeTasks(ctx context.Context, tasks []*SessionTask) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, task := range tasks {
		agent := sm.selectAgentForTask(task)
		task.AssignedTo = agent
		sm.activeTasks[task.ID] = task

		sm.logger.Info("تم توزيع المهمة",
			zap.String("session_id", sm.sessionID),
			zap.String("task_id", task.ID),
			zap.String("assigned_to", agent),
			zap.String("strategy", string(task.Strategy)))
	}

	return nil
}

// selectAgentForTask يختار وكيل للمهمة
func (sm *SessionManager) selectAgentForTask(task *SessionTask) string {
	if len(sm.activeAgents) == 0 {
		return sm.sessionManagerAgent
	}
	return sm.activeAgents[0]
}

// ExecuteTasks ينفذ المهام
func (sm *SessionManager) ExecuteTasks(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, task := range sm.activeTasks {
		switch task.Strategy {
		case StrategyConcurrent:
			go sm.executeTaskConcurrently(ctx, task)
		case StrategySequential:
			if err := sm.executeTaskSequentially(ctx, task); err != nil {
				return err
			}
		case StrategyMixed:
			go sm.executeTaskConcurrently(ctx, task)
		}
	}

	return nil
}

// executeTaskConcurrently ينفذ مهمة بالتزامن
func (sm *SessionManager) executeTaskConcurrently(ctx context.Context, task *SessionTask) {
	task.Status = TaskStatusRunning
	now := time.Now()
	task.StartedAt = &now

	sm.eventBus.BroadcastToAll(ctx, sm.sessionManagerAgent, TaskStarted, map[string]interface{}{
		"task_id":     task.ID,
		"description": task.Description,
		"assigned_to": task.AssignedTo,
	})

	result, err := sm.agentExecutor.ExecuteTask(ctx, task.Description)
	if err != nil {
		task.Status = TaskStatusFailed
		task.Error = err

		sm.eventBus.BroadcastToAll(ctx, sm.sessionManagerAgent, TaskFailed, map[string]interface{}{
			"task_id":     task.ID,
			"error":       err.Error(),
			"assigned_to": task.AssignedTo,
		})

		return
	}

	task.Status = TaskStatusCompleted
	task.Result = result
	completedAt := time.Now()
	task.CompletedAt = &completedAt

	sm.eventBus.BroadcastToAll(ctx, sm.sessionManagerAgent, TaskCompleted, map[string]interface{}{
		"task_id":     task.ID,
		"description": task.Description,
		"assigned_to": task.AssignedTo,
		"result":      result,
	})
}

// executeTaskSequentially ينفذ مهمة بالدور
func (sm *SessionManager) executeTaskSequentially(ctx context.Context, task *SessionTask) error {
	task.Status = TaskStatusRunning
	now := time.Now()
	task.StartedAt = &now

	result, err := sm.agentExecutor.ExecuteTask(ctx, task.Description)
	if err != nil {
		task.Status = TaskStatusFailed
		task.Error = err
		return err
	}

	task.Status = TaskStatusCompleted
	task.Result = result
	completedAt := time.Now()
	task.CompletedAt = &completedAt

	return nil
}

// MonitorTasks يراقب المهام بشكل لحظي
func (sm *SessionManager) MonitorTasks(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, task := range sm.activeTasks {
		sm.monitorTask(ctx, task)
	}

	return nil
}

// monitorTask يراقب مهمة واحدة
func (sm *SessionManager) monitorTask(ctx context.Context, task *SessionTask) {
	if task.Status == TaskStatusRunning {
		sm.memorySync.SyncMemory(ctx, task.AssignedTo)
		sm.skillSync.SyncSkills(ctx, task.AssignedTo)
	}
}

// SyncMemory يزامن الذاكرة بشكل لحظي
func (sm *SessionManager) SyncMemory(ctx context.Context, agentID string) error {
	return sm.memorySync.SyncMemory(ctx, agentID)
}

// SyncSkills يزامن المهارات بشكل لحظي
func (sm *SessionManager) SyncSkills(ctx context.Context, agentID string) error {
	return sm.skillSync.SyncSkills(ctx, agentID)
}

// GetSessionSummary يحصل على ملخص الجلسة
func (sm *SessionManager) GetSessionSummary(ctx context.Context) (*SessionSummary, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	summary := &SessionSummary{
		SessionID:           sm.sessionID,
		ClientPrompt:        sm.clientPrompt,
		SessionStartTime:    sm.sessionStartTime,
		SessionStatus:       sm.sessionStatus,
		ActiveAgents:        sm.activeAgents,
		SessionManagerAgent: sm.sessionManagerAgent,
		ActiveTasks:         len(sm.activeTasks),
		TaskHistory:         len(sm.taskHistory),
		MemorySyncStatus:    sm.memorySync.GetStatus(),
		SkillSyncStatus:     sm.skillSync.GetStatus(),
	}

	return summary, nil
}

// SessionSummary ملخص الجلسة
type SessionSummary struct {
	SessionID           string
	ClientPrompt        string
	SessionStartTime    time.Time
	SessionStatus       SessionStatus
	ActiveAgents        []string
	SessionManagerAgent string
	ActiveTasks         int
	TaskHistory         int
	MemorySyncStatus    map[string]interface{}
	SkillSyncStatus     map[string]interface{}
}

// evaluateComplexity يقيم تعقيد المهمة
func (sm *SessionManager) evaluateComplexity() TaskComplexity {
	promptLength := len(sm.clientPrompt)

	if promptLength < 100 {
		return ComplexityLow
	} else if promptLength < 500 {
		return ComplexityMedium
	} else if promptLength < 1000 {
		return ComplexityHigh
	} else {
		return ComplexityCritical
	}
}

// estimateTime يقدر الوقت المطلوب
func (sm *SessionManager) estimateTime() time.Duration {
	complexity := sm.evaluateComplexity()

	switch complexity {
	case ComplexityLow:
		return 5 * time.Minute
	case ComplexityMedium:
		return 30 * time.Minute
	case ComplexityHigh:
		return 2 * time.Hour
	case ComplexityCritical:
		return 8 * time.Hour
	default:
		return 1 * time.Hour
	}
}

// determineRequiredAgents يحدد الوكلاء المطلوبين
func (sm *SessionManager) determineRequiredAgents() []string {
	complexity := sm.evaluateComplexity()

	switch complexity {
	case ComplexityLow:
		return []string{"coder"}
	case ComplexityMedium:
		return []string{"coder", "reviewer"}
	case ComplexityHigh:
		return []string{"coder", "reviewer", "architect"}
	case ComplexityCritical:
		return []string{"coder", "reviewer", "architect", "tester"}
	default:
		return []string{"coder"}
	}
}

// recommendStrategy يوصي بالاستراتيجية
func (sm *SessionManager) recommendStrategy() TaskDistributionStrategy {
	complexity := sm.evaluateComplexity()

	switch complexity {
	case ComplexityLow:
		return StrategySequential
	case ComplexityMedium:
		return StrategySequential
	case ComplexityHigh:
		return StrategyMixed
	case ComplexityCritical:
		return StrategyMixed
	default:
		return StrategySequential
	}
}

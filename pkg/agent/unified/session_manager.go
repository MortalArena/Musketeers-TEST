package unified

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent/thinking"
	"go.uber.org/zap"
)

// SessionMode وضع الجلسة — تحكم أوتوماتيكي أو يدوي
type SessionMode string

const (
	SessionModeAuto   SessionMode = "auto"   // Session Manager Agent يقرر كل شيء
	SessionModeManual SessionMode = "manual" // البشر يحددون أدوار الوكلاء مسبقاً
)

// SessionManager مدير الجلسة المتطور
type SessionManager struct {
	sessionID string
	logger    *zap.Logger
	mu        sync.RWMutex

	// AgentPool — يدير جميع وكلاء الجلسة (كل وكيل حقيقي بمكوناته الكاملة)
	agentPool *AgentPool

	// معلومات الجلسة
	clientPrompt        string
	sessionStartTime    time.Time
	sessionStatus       SessionStatus
	sessionManagerAgent string // وكيل مدير الجلسة
	sessionMode         SessionMode
	manualAssignments   map[string]string // agentID -> role (للوضع اليدوي فقط)
	agentIndex          int               // عداد round-robin لتوزيع المهام

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

	// المكونات تحت سيطرة SessionManager فقط
	orchestratorEngine interface{} // محرك التنسيق الرئيسي (interface{} لتجنب import cycle)
	contextReranker    interface{} // محرك البحث السياقي
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

// NewSessionManager ينشئ مدير جلسة جديد (الوضع الافتراضي: auto)
// [FIX] لا ينشئ EventBus داخلياً — يستقبله من الخارج (UnifiedAgent) لضمان وجود EventBus واحد فقط
func NewSessionManager(sessionID string, logger *zap.Logger) *SessionManager {
	return &SessionManager{
		sessionID:                sessionID,
		logger:                   logger,
		sessionStatus:            SessionStatusInitializing,
		sessionMode:              SessionModeAuto,
		taskDistributionStrategy: StrategyMixed,
		activeTasks:              make(map[string]*SessionTask),
		taskHistory:              []*SessionTask{},
		memorySync:               NewRealTimeMemorySync(sessionID, logger),
		skillSync:                NewRealTimeSkillSync(sessionID, logger),
		eventBus:                 nil,
		taskScheduler:            NewTaskScheduler(sessionID, logger),
	}
}

// SetAgentPool يضبط AgentPool الخاص بالجلسة ويربطه بـ EventBus
func (sm *SessionManager) SetAgentPool(pool *AgentPool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.agentPool = pool
	if sm.eventBus != nil {
		pool.SetEventBus(sm.eventBus)
	}
}

// SetEventBus يضبط EventBus للجلسة (يُستخدم عندما يكون هناك EventBus موحد من UnifiedAgent)
// [WHY] يجب استدعاؤها بعد NewSessionManager وقبل أي استخدام لـ EventBus
func (sm *SessionManager) SetEventBus(eventBus *SessionEventBus) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.eventBus = eventBus
}

// SetOrchestratorEngine يضبط OrchestratorEngine تحت سيطرة SessionManager
// [WHY] SessionManager هو الوحيد الذي يتحكم في OrchestratorEngine لمنع الفوضى
func (sm *SessionManager) SetOrchestratorEngine(engine interface{}) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.orchestratorEngine = engine
	sm.logger.Info("تم ضبط OrchestratorEngine تحت سيطرة SessionManager",
		zap.String("session_id", sm.sessionID))
}

// SetContextReranker يضبط ContextReranker تحت سيطرة SessionManager
// [WHY] SessionManager هو الوحيد الذي يتحكم في ContextReranker لمنع الفوضى
func (sm *SessionManager) SetContextReranker(reranker interface{}) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.contextReranker = reranker
	sm.logger.Info("تم ضبط ContextReranker تحت سيطرة SessionManager",
		zap.String("session_id", sm.sessionID))
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

// SetMode يضبط وضع الجلسة (auto/manual)
func (sm *SessionManager) SetMode(mode SessionMode) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessionMode = mode
	sm.logger.Info("تم ضبط وضع الجلسة",
		zap.String("session_id", sm.sessionID),
		zap.String("mode", string(mode)))
}

// SetManualAssignments يضبط التوزيع اليدوي للوكلاء على الأدوار (للوضع manual فقط)
func (sm *SessionManager) SetManualAssignments(assignments map[string]string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.manualAssignments = make(map[string]string)
	for k, v := range assignments {
		sm.manualAssignments[k] = v
	}
	sm.logger.Info("تم ضبط التوزيع اليدوي للوكلاء",
		zap.String("session_id", sm.sessionID),
		zap.Int("assignments", len(assignments)))
}

// GetMode يرجع وضع الجلسة الحالي
func (sm *SessionManager) GetMode() SessionMode {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.sessionMode
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

// selectAgentForTask يختار وكيل للمهمة حسب وضع الجلسة
// Auto: round-robin عبر الوكلاء النشطين من AgentPool
// Manual: يبحث في manualAssignments أولاً، ثم round-robin
func (sm *SessionManager) selectAgentForTask(task *SessionTask) string {
	// الحصول على الوكلاء النشطين من AgentPool
	activeAgents := sm.getActiveAgentIDs()
	if len(activeAgents) == 0 {
		return sm.sessionManagerAgent
	}

	// Manual: ابحث عن وكيل مخصص لهذه المهمة
	if sm.sessionMode == SessionModeManual && len(sm.manualAssignments) > 0 {
		for agentID, role := range sm.manualAssignments {
			if contains(task.Description, role) || contains(role, task.Description) {
				for _, active := range activeAgents {
					if active == agentID {
						return agentID
					}
				}
			}
		}
	}

	// Auto أو fallback: round-robin عبر الوكلاء النشطين
	sm.agentIndex = (sm.agentIndex + 1) % len(activeAgents)
	return activeAgents[sm.agentIndex]
}

// getActiveAgentIDs يعيد قائمة IDs الوكلاء النشطين من AgentPool
// [FIXED] تستخدم GetActiveAgents() بدلاً من GetAllAgents()
// لمنع تعيين مهام لوكلاء مركونين (parked) ليس لديهم ThinkingEngine
func (sm *SessionManager) getActiveAgentIDs() []string {
	if sm.agentPool != nil {
		return sm.agentPool.GetActiveAgents()
	}
	return nil
}

// contains helper بسيط للبحث عن نص داخل نص آخر
func contains(str, substr string) bool {
	return strings.Contains(str, substr)
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

// executeTaskConcurrently ينفذ مهمة بالتزامن — مع توجيه لـ ThinkingEngine الوكيل المحدد
func (sm *SessionManager) executeTaskConcurrently(ctx context.Context, task *SessionTask) {
	sm.mu.Lock()
	task.Status = TaskStatusRunning
	now := time.Now()
	task.StartedAt = &now
	sm.mu.Unlock()

	sm.eventBus.BroadcastToAll(ctx, sm.sessionManagerAgent, TaskStarted, map[string]interface{}{
		"task_id":     task.ID,
		"description": task.Description,
		"assigned_to": task.AssignedTo,
	})

	// [FIX] توجيه المهمة لـ ThinkingEngine الوكيل المحدد (وليس main agent فقط)
	result, err := sm.routeTaskToAgent(ctx, task)
	sm.mu.Lock()
	if err != nil {
		task.Status = TaskStatusFailed
		task.Error = err
		sm.mu.Unlock()

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
	sm.mu.Unlock()

	sm.eventBus.BroadcastToAll(ctx, sm.sessionManagerAgent, TaskCompleted, map[string]interface{}{
		"task_id":     task.ID,
		"description": task.Description,
		"assigned_to": task.AssignedTo,
		"result":      result,
	})
}

// executeTaskSequentially ينفذ مهمة بالدور — مع توجيه لـ ThinkingEngine الوكيل المحدد
func (sm *SessionManager) executeTaskSequentially(ctx context.Context, task *SessionTask) error {
	task.Status = TaskStatusRunning
	now := time.Now()
	task.StartedAt = &now

	result, err := sm.routeTaskToAgent(ctx, task)
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

// routeTaskToAgent يوجه المهمة لـ ThinkingEngine الوكيل المحدد
// [FIX] المهمة تذهب للوكيل المعين في task.AssignedTo بدلاً من main agent دائماً
func (sm *SessionManager) routeTaskToAgent(ctx context.Context, task *SessionTask) (interface{}, error) {
	if task.AssignedTo != "" && task.AssignedTo != sm.sessionManagerAgent && sm.agentPool != nil {
		agentTE, err := sm.agentPool.GetOrCreateThinkingEngine(task.AssignedTo)
		if err == nil && agentTE != nil {
			sm.logger.Info("توجيه المهمة لـ ThinkingEngine الوكيل المحدد",
				zap.String("task_id", task.ID),
				zap.String("assigned_to", task.AssignedTo),
			)
			result, err := agentTE.AnalyzeTask(ctx, task.Description)
			if err != nil {
				return nil, fmt.Errorf("فشل تنفيذ المهمة عبر الوكيل %s: %w", task.AssignedTo, err)
			}
			// تمرير النتيجة عبر ExecuteWithWorkflow للتنفيذ الكامل
			workflowResult, err := agentTE.ExecuteWithWorkflow(ctx, task.Description)
			if err != nil {
				return nil, fmt.Errorf("فشل سير العمل للوكيل %s: %w", task.AssignedTo, err)
			}
			return map[string]interface{}{
				"analysis": result,
				"workflow": workflowResult,
				"agent_id": task.AssignedTo,
			}, nil
		}
		sm.logger.Warn("فشل الحصول على ThinkingEngine للوكيل، استخدام main agent كاحتياط",
			zap.String("assigned_to", task.AssignedTo),
			zap.Error(err),
		)
	}

	// احتياط: main agent
	return sm.agentExecutor.ExecuteTask(ctx, task.Description)
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
		ActiveAgents:        sm.getActiveAgentIDs(),
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
// لم يعد يستخدم أدواراً وهمية — Session Manager Agent يقرر الاحتياجات الفعلية
// في Auto Mode: Session Manager Agent يحلل المهمة ويحدد الوكلاء المطلوبين
// في Manual Mode: البشر يحددون التوزيع يدوياً
func (sm *SessionManager) determineRequiredAgents() []string {
	if sm.sessionMode == SessionModeManual && len(sm.manualAssignments) > 0 {
		agents := make([]string, 0, len(sm.manualAssignments))
		for agentID := range sm.manualAssignments {
			agents = append(agents, agentID)
		}
		return agents
	}

	// Auto Mode: لا نقيد بعدد محدد من الوكلاء
	// Session Manager Agent سيقرر الاحتياجات وقت التنفيذ
	return []string{}
}

// QueryProjectContext يجيب على أسئلة حول قاعدة الشيفرة بالكامل
// يستخدم ContextReranker للبحث في جميع ملفات المشروع
func (sm *SessionManager) QueryProjectContext(ctx context.Context, query string) (*thinking.CodeContextResult, error) {
	// الحصول على ThinkingEngine الخاص بمدير الجلسة
	agentTE, err := sm.agentPool.GetOrCreateThinkingEngine(sm.sessionManagerAgent)
	if err != nil {
		return nil, fmt.Errorf("فشل الوصول لمحرك التفكير: %w", err)
	}

	// تهيئة ContextReranker إذا لم يكن موجوداً
	if agentTE.GetContextReranker() == nil {
		// تحديد مسار المشروع (نبحث عن go.mod)
		projectRoot := sm.detectProjectRoot()
		agentTE.InitContextReranker(projectRoot)
	}

	// تنفيذ البحث السياقي
	chunks, err := agentTE.SearchContext(ctx, query, 10)
	if err != nil {
		return nil, fmt.Errorf("فشل البحث في السياق: %w", err)
	}

	// تحويل RerankedChunks إلى CodeChunks للتوافق
	codeChunks := make([]*thinking.CodeChunk, len(chunks))
	for i, c := range chunks {
		codeChunks[i] = c.CodeChunk
	}

	// بناء النتيجة
	var summaryBuilder strings.Builder
	summaryBuilder.WriteString(fmt.Sprintf("نتائج البحث عن \"%s\":\n", query))

	if len(codeChunks) == 0 {
		summaryBuilder.WriteString("لم أجد نتائج متعلقة في قاعدة الشيفرة.")
	} else {
		fileSet := make(map[string][]string)
		for _, c := range codeChunks {
			name := c.Name
			if name == "" {
				name = fmt.Sprintf("line %d", c.StartLine)
			}
			fileSet[c.FilePath] = append(fileSet[c.FilePath], name)
		}
		for path, names := range fileSet {
			summaryBuilder.WriteString(fmt.Sprintf("  • %s\n", path))
			for _, n := range names {
				summaryBuilder.WriteString(fmt.Sprintf("      - %s\n", n))
			}
		}
	}

	sm.logger.Info("استعلام سياق المشروع",
		zap.String("query", query),
		zap.Int("results", len(codeChunks)),
	)

	return &thinking.CodeContextResult{
		Query:      query,
		Chunks:     codeChunks,
		Summary:    summaryBuilder.String(),
		TotalFound: len(codeChunks),
	}, nil
}

// detectProjectRoot يكتشف مسار المشروع بالبحث عن go.mod
func (sm *SessionManager) detectProjectRoot() string {
	// محاولة استخدام المسار المحفوظ أو البحث عن go.mod
	searchPaths := []string{
		".",
		"..",
		"../..",
	}
	for _, p := range searchPaths {
		if _, err := os.Stat(filepath.Join(p, "go.mod")); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	// Fallback: مسار التنفيذ الحالي
	wd, _ := os.Getwd()
	return wd
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

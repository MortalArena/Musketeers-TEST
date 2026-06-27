package advanced

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SessionMode وضع الجلسة — تحكم أوتوماتيكي أو يدوي
type SessionMode string

const (
	SessionModeAuto   SessionMode = "auto"   // Session Manager Agent يقرر كل شيء
	SessionModeManual SessionMode = "manual" // البشر يحددون أدوار الوكلاء مسبقاً
)

// AdvancedSessionManager مدير الجلسة المتطور
type AdvancedSessionManager struct {
	sessionID string
	logger    *zap.Logger
	mu        sync.RWMutex

	// معلومات الجلسة
	clientPrompt        string
	sessionStartTime    time.Time
	sessionStatus       SessionStatus
	activeAgents        []string
	sessionManagerAgent string
	sessionMode         SessionMode
	manualAssignments   map[string]string // agentID -> role (للوضع اليدوي فقط)
	agentIndex          int               // عداد round-robin لتوزيع المهام

	// إدارة المهام
	taskDistributionStrategy TaskDistributionStrategy
	activeTasks              map[string]*SessionTask
	taskHistory              []*SessionTask

	// مجدول المهام
	taskScheduler *TaskScheduler
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
	StrategyConcurrent TaskDistributionStrategy = "concurrent"
	StrategySequential TaskDistributionStrategy = "sequential"
	StrategyMixed      TaskDistributionStrategy = "mixed"
)

// TaskComplexity تعقيد المهمة
type TaskComplexity string

const (
	ComplexityLow      TaskComplexity = "low"
	ComplexityMedium   TaskComplexity = "medium"
	ComplexityHigh     TaskComplexity = "high"
	ComplexityCritical TaskComplexity = "critical"
)

// TaskScheduler مجدول المهام
type TaskScheduler struct {
	sessionID string
	logger    *zap.Logger
}

// NewAdvancedSessionManager ينشئ مدير جلسة متطور جديد (الوضع الافتراضي: auto)
func NewAdvancedSessionManager(sessionID string, logger *zap.Logger) *AdvancedSessionManager {
	return &AdvancedSessionManager{
		sessionID:                sessionID,
		logger:                   logger,
		sessionStatus:            SessionStatusInitializing,
		sessionMode:              SessionModeAuto,
		taskDistributionStrategy: StrategyMixed,
		activeTasks:              make(map[string]*SessionTask),
		taskHistory:              []*SessionTask{},
		taskScheduler:            NewTaskScheduler(sessionID, logger),
	}
}

// NewTaskScheduler ينشئ مجدول مهام جديد
func NewTaskScheduler(sessionID string, logger *zap.Logger) *TaskScheduler {
	return &TaskScheduler{
		sessionID: sessionID,
		logger:    logger,
	}
}

// Initialize يهيئ مدير الجلسة المتطور
func (asm *AdvancedSessionManager) Initialize(ctx context.Context) error {
	asm.mu.Lock()
	defer asm.mu.Unlock()

	asm.sessionStartTime = time.Now()
	asm.sessionStatus = SessionStatusActive

	asm.logger.Info("تم تهيئة مدير الجلسة المتطور",
		zap.String("session_id", asm.sessionID),
		zap.Time("start_time", asm.sessionStartTime))

	return nil
}

// SetMode يضبط وضع الجلسة (auto/manual)
func (asm *AdvancedSessionManager) SetMode(mode SessionMode) {
	asm.mu.Lock()
	defer asm.mu.Unlock()
	asm.sessionMode = mode
	asm.logger.Info("تم ضبط وضع الجلسة",
		zap.String("session_id", asm.sessionID),
		zap.String("mode", string(mode)))
}

// SetManualAssignments يضبط التوزيع اليدوي للوكلاء على الأدوار (للوضع manual فقط)
func (asm *AdvancedSessionManager) SetManualAssignments(assignments map[string]string) {
	asm.mu.Lock()
	defer asm.mu.Unlock()
	asm.manualAssignments = make(map[string]string)
	for k, v := range assignments {
		asm.manualAssignments[k] = v
	}
	asm.logger.Info("تم ضبط التوزيع اليدوي للوكلاء",
		zap.String("session_id", asm.sessionID),
		zap.Int("assignments", len(assignments)))
}

// GetMode يرجع وضع الجلسة الحالي
func (asm *AdvancedSessionManager) GetMode() SessionMode {
	asm.mu.RLock()
	defer asm.mu.RUnlock()
	return asm.sessionMode
}

// ReceivePrompt يستقبل البرومبت من العميل
func (asm *AdvancedSessionManager) ReceivePrompt(ctx context.Context, prompt string) error {
	asm.mu.Lock()
	defer asm.mu.Unlock()

	asm.clientPrompt = prompt
	asm.logger.Info("تم استقبال البرومبت من العميل",
		zap.String("session_id", asm.sessionID),
		zap.String("prompt", prompt))

	return nil
}

// EvaluateTask يقيم المهمة
func (asm *AdvancedSessionManager) EvaluateTask(ctx context.Context) (*TaskEvaluation, error) {
	asm.mu.Lock()
	defer asm.mu.Unlock()

	evaluation := &TaskEvaluation{
		SessionID:           asm.sessionID,
		Prompt:              asm.clientPrompt,
		Complexity:          asm.evaluateComplexity(),
		EstimatedTime:       asm.estimateTime(),
		RequiredAgents:      asm.determineRequiredAgents(),
		RecommendedStrategy: asm.recommendStrategy(),
	}

	asm.logger.Info("تم تقييم المهمة",
		zap.String("session_id", asm.sessionID),
		zap.String("complexity", string(evaluation.Complexity)),
		zap.Int("required_agents", len(evaluation.RequiredAgents)),
		zap.String("strategy", string(evaluation.RecommendedStrategy)))

	return evaluation, nil
}

// TaskEvaluation تقييم المهمة
type TaskEvaluation struct {
	SessionID           string
	Prompt              string
	Complexity          TaskComplexity
	EstimatedTime       time.Duration
	RequiredAgents      []string
	RecommendedStrategy TaskDistributionStrategy
}

// evaluateComplexity يقيم تعقيد المهمة
func (asm *AdvancedSessionManager) evaluateComplexity() TaskComplexity {
	promptLength := len(asm.clientPrompt)

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
func (asm *AdvancedSessionManager) estimateTime() time.Duration {
	complexity := asm.evaluateComplexity()

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
// لم يعد يستخدم أدواراً وهمية — Session Manager Agent أو البشر يقررون
func (asm *AdvancedSessionManager) determineRequiredAgents() []string {
	if asm.sessionMode == SessionModeManual && len(asm.manualAssignments) > 0 {
		agents := make([]string, 0, len(asm.manualAssignments))
		for agentID := range asm.manualAssignments {
			agents = append(agents, agentID)
		}
		return agents
	}

	// Auto Mode: لا نقيد بعدد محدد من الوكلاء
	return []string{}
}

// recommendStrategy يوصي بالاستراتيجية
func (asm *AdvancedSessionManager) recommendStrategy() TaskDistributionStrategy {
	complexity := asm.evaluateComplexity()

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

// GetSessionSummary يحصل على ملخص الجلسة
func (asm *AdvancedSessionManager) GetSessionSummary(ctx context.Context) (*SessionSummary, error) {
	asm.mu.RLock()
	defer asm.mu.RUnlock()

	summary := &SessionSummary{
		SessionID:           asm.sessionID,
		ClientPrompt:        asm.clientPrompt,
		SessionStartTime:    asm.sessionStartTime,
		SessionStatus:       asm.sessionStatus,
		ActiveAgents:        asm.activeAgents,
		SessionManagerAgent: asm.sessionManagerAgent,
		ActiveTasks:         len(asm.activeTasks),
		TaskHistory:         len(asm.taskHistory),
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
}

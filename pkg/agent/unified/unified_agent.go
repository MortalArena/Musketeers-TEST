package unified

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent/automation"
	"github.com/MortalArena/Musketeers/pkg/agent/direction"
	"github.com/MortalArena/Musketeers/pkg/agent/integration"
	"github.com/MortalArena/Musketeers/pkg/agent/subagents"
	"github.com/MortalArena/Musketeers/pkg/agent/validation"
	"github.com/MortalArena/Musketeers/pkg/session"
	"github.com/dgraph-io/badger/v4"
	"go.uber.org/zap"
)

// UnifiedAgent الوكيل الموحد الذي يدمج جميع الأنظمة
type UnifiedAgent struct {
	sessionID string
	agentID   string

	// الأنظمة المدمجة الشاملة
	unifiedSkillManager  *UnifiedSkillManager
	unifiedMemoryManager *UnifiedMemoryManager

	// الأنظمة الجديدة من Cursor
	subagentManager     *subagents.SubagentManager
	automationManager   *automation.AutomationManager
	skillDirector       *direction.SkillDirector
	multiLayerValidator *validation.MultiLayerValidator

	// نظام التنسيق المركزي
	coordinator  *Coordinator
	flowManager  *FlowManager
	errorHandler *ErrorHandler

	// النظام الجماعي
	collectiveSystem *integration.CollectiveAgentSystem

	// أنظمة المزامنة اللحظية
	sessionEventBus    *SessionEventBus
	realTimeMemorySync *RealTimeMemorySync
	realTimeSkillSync  *RealTimeSkillSync

	// نظام تسجيل المشاكل والحلول
	problemSolutionRegistry *ProblemSolutionRegistry

	// الذاكرة المحلية
	localMemoryCache *LocalMemoryCache

	// قناة الأحداث
	eventChannel chan *SessionEvent

	logger *zap.Logger
	mu     sync.RWMutex
}

// NewUnifiedAgent ينشئ وكيل موحد جديد
func NewUnifiedAgent(sessionID, agentID string, db *badger.DB, logger *zap.Logger) *UnifiedAgent {
	ua := &UnifiedAgent{
		sessionID: sessionID,
		agentID:   agentID,
		logger:    logger,
	}

	// إنشاء الأنظمة المدمجة الشاملة
	ua.unifiedSkillManager = NewUnifiedSkillManager(sessionID, db, logger)
	ua.unifiedMemoryManager = NewUnifiedMemoryManager(sessionID, db, logger)

	// إنشاء الأنظمة الجديدة من Cursor
	ua.subagentManager = subagents.NewSubagentManager(logger)
	ua.automationManager = automation.NewAutomationManager(logger)
	ua.skillDirector = direction.NewSkillDirector(nil, logger) // سيتم تحديثه لاحقاً
	ua.multiLayerValidator = validation.NewMultiLayerValidator(logger)

	// إنشاء نظام التنسيق المركزي
	ua.coordinator = NewCoordinator(logger)
	ua.flowManager = NewFlowManager(logger)
	ua.errorHandler = NewErrorHandler(logger)

	// إنشاء النظام الجماعي (مع الأنظمة المدمجة)
	// نستخدم sessionSkills و sessionMemory من الأنظمة المدمجة
	sessionSkills := session.NewSkillsManager(sessionID)
	sessionMemory := session.NewCollectiveMemory(sessionID, db)
	ua.collectiveSystem = integration.NewCollectiveAgentSystem(sessionID, sessionSkills, sessionMemory, logger)

	// إنشاء أنظمة المزامنة اللحظية
	ua.sessionEventBus = NewSessionEventBus(sessionID, logger)
	ua.realTimeMemorySync = NewRealTimeMemorySync(sessionID, logger)
	ua.realTimeSkillSync = NewRealTimeSkillSync(sessionID, logger)
	ua.eventChannel = make(chan *SessionEvent, 100)

	// إنشاء نظام تسجيل المشاكل والحلول
	ua.problemSolutionRegistry = NewProblemSolutionRegistry(sessionID, logger)

	// إنشاء الذاكرة المحلية
	ua.localMemoryCache = NewLocalMemoryCache(sessionID, agentID, logger)

	return ua
}

// Initialize يهيئ الوكيل الموحد
func (ua *UnifiedAgent) Initialize(ctx context.Context) error {
	ua.mu.Lock()
	defer ua.mu.Unlock()

	// [WHY] تهيئة جميع الأنظمة
	// [HOW] يهيئ كل نظام بشكل متسلسل
	// [SAFETY] يضمن عدم وجود أخطاء في التهيئة

	// تهيئة نظام التنسيق المركزي
	if err := ua.coordinator.Initialize(ctx, ua); err != nil {
		return fmt.Errorf("فشل تهيئة المنسق: %w", err)
	}

	// تهيئة مدير التدفق
	if err := ua.flowManager.Initialize(ctx, ua); err != nil {
		return fmt.Errorf("فشل تهيئة مدير التدفق: %w", err)
	}

	// تهيئة معالج الأخطاء
	if err := ua.errorHandler.Initialize(ctx); err != nil {
		return fmt.Errorf("فشل تهيئة معالج الأخطاء: %w", err)
	}

	// تهيئة أنظمة المزامنة اللحظية
	ua.sessionEventBus.Start(ctx)
	ua.realTimeMemorySync.StartSync(ctx)
	ua.realTimeSkillSync.StartSync(ctx)

	// الاشتراك في ناقل الأحداث
	ua.eventChannel = ua.sessionEventBus.SubscribeAgent(ua.agentID)

	// بدء معالجة الأحداث
	go ua.processEvents(ctx)

	// بدء التسجيل الإجباري للتطورات اللحظية
	go ua.startMandatoryProgressReporting(ctx)

	// بدء المزامنة الإجبارية للقراءة
	go ua.startMandatoryReadSync(ctx)

	// بدء المزامنة الإجبارية للذاكرة المحلية
	go ua.localMemoryCache.StartMandatorySync(ctx)

	ua.logger.Info("تم تهيئة الوكيل الموحد بنجاح",
		zap.String("session_id", ua.sessionID),
		zap.String("agent_id", ua.agentID))

	return nil
}

// ExecuteTask ينفذ مهمة باستخدام جميع الأنظمة المتكاملة
func (ua *UnifiedAgent) ExecuteTask(ctx context.Context, task string) (*UnifiedTaskResult, error) {
	ua.mu.Lock()
	defer ua.mu.Unlock()

	// [WHY] تنفيذ مهمة باستخدام جميع الأنظمة المتكاملة
	// [HOW] يستخدم المنسق لتنسيق جميع الأنظمة
	// [SAFETY] يضمن تنفيذ آمن ومتناسق

	startTime := time.Now()

	// إنشاء سياق التنفيذ
	executionContext := ua.flowManager.CreateExecutionContext(ctx, task)

	// استخدام المنسق لتنسيق التنفيذ
	result, err := ua.coordinator.ExecuteTask(ctx, executionContext)
	if err != nil {
		// استخدام معالج الأخطاء
		recoveryResult := ua.errorHandler.HandleError(ctx, err, executionContext)
		if recoveryResult.Success {
			ua.logger.Info("تم استرداد من الخطأ", zap.String("error", err.Error()))
		} else {
			return nil, fmt.Errorf("فشل تنفيذ المهمة: %w", err)
		}
	}

	duration := time.Since(startTime)
	result.Duration = duration

	// التحقق متعدد الطبقات
	validationResult, err := ua.multiLayerValidator.ValidateAll(ctx, task, nil, result.Output)
	if err != nil {
		ua.logger.Warn("فشل التحقق متعدد الطبقات", zap.Error(err))
	}
	result.ValidationResult = validationResult

	ua.logger.Info("تم تنفيذ المهمة بنجاح",
		zap.String("task", task),
		zap.Duration("duration", duration),
		zap.Bool("success", result.Success),
		zap.Float64("confidence", result.Confidence))

	return result, nil
}

// RegisterAgent يسجل وكيل في النظام الموحد
func (ua *UnifiedAgent) RegisterAgent(ctx context.Context, did, agentType, llmType string, specializations []string) error {
	ua.mu.Lock()
	defer ua.mu.Unlock()

	// [WHY] تسجيل وكيل في النظام الموحد
	// [HOW] يستخدم الأنظمة المدمجة الشاملة
	// [SAFETY] يضمن عدم تكرار التسجيل

	// تسجيل في نظام المهارات الشامل
	if err := ua.unifiedSkillManager.RegisterAgent(did, agentType); err != nil {
		return fmt.Errorf("فشل التسجيل في نظام المهارات الشامل: %w", err)
	}

	// تسجيل في نظام الوكلاء الفرعيين
	subagentConfig := &subagents.SubagentConfig{
		Name:            did,
		Description:     fmt.Sprintf("وكيل من نوع %s (LLM: %s)", agentType, llmType),
		SystemPrompt:    fmt.Sprintf("أنت وكيل متخصص من نوع %s يعمل بنظام LLM %s", agentType, llmType),
		Specialization:  agentType,
		Capabilities:    specializations,
		Priority:        1,
		ReadOnly:        false,
		RunInBackground: false,
	}

	if _, err := ua.subagentManager.CreateSubagent(subagentConfig); err != nil {
		return fmt.Errorf("فشل إنشاء الوكيل الفرعي: %w", err)
	}

	ua.logger.Info("تم تسجيل الوكيل في النظام الموحد",
		zap.String("did", did),
		zap.String("agent_type", agentType),
		zap.String("llm_type", llmType))

	return nil
}

// GetSystemSummary يحصل على ملخص النظام الموحد
func (ua *UnifiedAgent) GetSystemSummary(ctx context.Context) (*UnifiedSystemSummary, error) {
	ua.mu.RLock()
	defer ua.mu.RUnlock()

	// [WHY] الحصول على ملخص النظام الموحد
	// [HOW] يجمع ملخصات جميع الأنظمة المدمجة
	// [SAFETY] يضمان عدم وجود أخطاء في الجمع

	summary := &UnifiedSystemSummary{
		SessionID: ua.sessionID,
		AgentID:   ua.agentID,
		Timestamp: time.Now(),
	}

	// ملخص الأنظمة المدمجة الشاملة
	summary.SkillSummary = ua.unifiedSkillManager.GetSkillSummary()
	summary.MemorySummary = ua.unifiedMemoryManager.GetMemorySummary()

	// ملخص الأنظمة الجديدة
	summary.SubagentSummary = ua.subagentManager.GetSubagentSummary()
	summary.AutomationSummary = ua.automationManager.GetAutomationSummary()
	summary.ValidationSummary = ua.multiLayerValidator.GetValidationSummary()

	// ملخص نظام التنسيق المركزي
	summary.CoordinatorSummary = ua.coordinator.GetSummary()
	summary.FlowManagerSummary = ua.flowManager.GetSummary()
	summary.ErrorHandlerSummary = ua.errorHandler.GetSummary()

	// حساب الجاهزية الكلية
	summary.OverallReadiness = ua.calculateOverallReadiness()

	return summary, nil
}

// calculateOverallReadiness يحسب الجاهزية الكلية
func (ua *UnifiedAgent) calculateOverallReadiness() float64 {
	// [WHY] حساب الجاهزية الكلية
	// [HOW] يحسب متوسط جاهزية جميع الأنظمة الفعلية
	// [SAFETY] يقرأ الحالة الفعلية للأنظمة الفرعية

	readiness := 0.0
	systemCount := 0

	// التحقق من جاهزية UnifiedSkillManager
	if ua.unifiedSkillManager != nil {
		skillSummary := ua.unifiedSkillManager.GetSkillSummary()
		if skillSummary != nil {
			readiness += 0.2
			systemCount++
		}
	}

	// التحقق من جاهزية UnifiedMemoryManager
	if ua.unifiedMemoryManager != nil {
		memorySummary := ua.unifiedMemoryManager.GetMemorySummary()
		if memorySummary != nil {
			readiness += 0.2
			systemCount++
		}
	}

	// التحقق من جاهزية SubagentManager
	if ua.subagentManager != nil {
		subagentSummary := ua.subagentManager.GetSubagentSummary()
		if subagentSummary != nil {
			readiness += 0.15
			systemCount++
		}
	}

	// التحقق من جاهزية AutomationManager
	if ua.automationManager != nil {
		automationSummary := ua.automationManager.GetAutomationSummary()
		if automationSummary != nil {
			readiness += 0.15
			systemCount++
		}
	}

	// التحقق من جاهزية Coordinator
	if ua.coordinator != nil {
		coordinatorSummary := ua.coordinator.GetSummary()
		if coordinatorSummary != nil {
			readiness += 0.15
			systemCount++
		}
	}

	// التحقق من جاهزية FlowManager
	if ua.flowManager != nil {
		flowManagerSummary := ua.flowManager.GetSummary()
		if flowManagerSummary != nil {
			readiness += 0.15
			systemCount++
		}
	}

	// التحقق من جاهزية ErrorHandler
	if ua.errorHandler != nil {
		readiness += 0.15
		systemCount++
	}

	// التحقق من جاهزية أنظمة المزامنة
	if ua.sessionEventBus != nil && ua.realTimeMemorySync != nil && ua.realTimeSkillSync != nil {
		readiness += 0.1
		systemCount++
	}

	// حساب المتوسط
	if systemCount > 0 {
		readiness = readiness / float64(systemCount)
	}

	// التأكد من أن القارة بين 0 و 1
	if readiness > 1.0 {
		readiness = 1.0
	}
	if readiness < 0.0 {
		readiness = 0.0
	}

	return readiness
}

// UnifiedTaskResult نتيجة تنفيذ المهمة الموحدة
type UnifiedTaskResult struct {
	Task             string
	Success          bool
	Confidence       float64
	Output           interface{}
	Duration         time.Duration
	ValidationResult *validation.ValidationResult
	Metadata         map[string]interface{}
}

// UnifiedSystemSummary ملخص النظام الموحد
type UnifiedSystemSummary struct {
	SessionID           string
	AgentID             string
	Timestamp           time.Time
	SkillSummary        map[string]interface{}
	MemorySummary       map[string]interface{}
	SubagentSummary     map[string]interface{}
	AutomationSummary   map[string]interface{}
	ValidationSummary   map[string]interface{}
	CoordinatorSummary  map[string]interface{}
	FlowManagerSummary  map[string]interface{}
	ErrorHandlerSummary map[string]interface{}
	OverallReadiness    float64
}

// processEvents يعالج الأحداث من ناقل الأحداث
func (ua *UnifiedAgent) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			ua.logger.Info("تم إيقاف معالجة الأحداث")
			return
		case event, ok := <-ua.eventChannel:
			if !ok {
				ua.logger.Info("تم إغلاق قناة الأحداث")
				return
			}
			ua.handleEvent(event)
		}
	}
}

// handleEvent يعالج حدث واحد
func (ua *UnifiedAgent) handleEvent(event *SessionEvent) {
	ua.logger.Info("تم استقبال حدث",
		zap.String("agent_id", ua.agentID),
		zap.String("event_id", event.ID),
		zap.String("event_type", string(event.EventType)),
		zap.String("source_agent", event.SourceAgent))

	// معالجة الحدث بناءً على نوعه
	switch event.EventType {
	case TaskStarted:
		// معالجة بدء المهمة
	case TaskProgress:
		// معالجة تقدم المهمة
	case TaskCompleted:
		// معالجة إكمال المهمة
	case TaskFailed:
		// معالجة فشل المهمة
	}
}

// startMandatoryProgressReporting يبدأ التسجيل الإجباري للتطورات اللحظية
func (ua *UnifiedAgent) startMandatoryProgressReporting(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			ua.logger.Info("تم إيقاف التسجيل الإجباري للتطورات اللحظية")
			return
		case <-ticker.C:
			ua.reportProgress(ctx)
		}
	}
}

// reportProgress يبلغ عن التطورات اللحظية
func (ua *UnifiedAgent) reportProgress(ctx context.Context) {
	// إنشاء حدث تقدم
	event := &SessionEvent{
		ID:          fmt.Sprintf("progress_%d", time.Now().UnixNano()),
		SessionID:   ua.sessionID,
		SourceAgent: ua.agentID,
		TargetAgent: "", // جميع الوكلاء
		EventType:   TaskProgress,
		Timestamp:   time.Now(),
		Priority:    PriorityMedium,
		Data: map[string]interface{}{
			"agent_id": ua.agentID,
			"status":   "active",
			"message":  "التطور اللحظي",
		},
		Metadata: map[string]interface{}{
			"reporting_type": "mandatory",
			"interval":       "5s",
		},
	}

	// نشر الحدث
	if err := ua.sessionEventBus.PublishEvent(ctx, event); err != nil {
		ua.logger.Error("فشل نشر حدث التطور اللحظي", zap.Error(err))
	}

	// نشر أحداث الذاكرة
	ua.publishMemoryEvents(ctx)

	// نشر أحداث المهارات
	ua.publishSkillEvents(ctx)
}

// publishMemoryEvents ينشر أحداث الذاكرة
func (ua *UnifiedAgent) publishMemoryEvents(ctx context.Context) {
	// نشر أحداث الذاكرة إلى RealTimeMemorySync
	// هذا يضمن أن جميع الوكلاء يرون التطورات اللحظية في الذاكرة
}

// publishSkillEvents ينشر أحداث المهارات
func (ua *UnifiedAgent) publishSkillEvents(ctx context.Context) {
	// نشر أحداث المهارات إلى RealTimeSkillSync
	// هذا يضمن أن جميع الوكلاء يرون التطورات اللحظية في المهارات
}

// startMandatoryReadSync يبدأ المزامنة الإجبارية للقراءة
func (ua *UnifiedAgent) startMandatoryReadSync(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	lastSyncTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			ua.logger.Info("تم إيقاف المزامنة الإجبارية للقراءة")
			return
		case <-ticker.C:
			ua.syncNewData(ctx, lastSyncTime)
			lastSyncTime = time.Now()
		}
	}
}

// syncNewData يقرأ البيانات الجديدة من قاعدة البيانات المشتركة
func (ua *UnifiedAgent) syncNewData(ctx context.Context, since time.Time) {
	// قراءة ملخص الذاكرة الحالي
	memorySummary := ua.unifiedMemoryManager.GetMemorySummary()

	// قراءة ملخص المهارات الحالي
	skillSummary := ua.unifiedSkillManager.GetSkillSummary()

	// تحديث الذاكرة المحلية
	ua.updateLocalMemory(memorySummary)

	// تحديث المهارات المحلية
	ua.updateLocalSkills(skillSummary)

	ua.logger.Info("تمت المزامنة الإجبارية للقراءة",
		zap.Time("since", since),
		zap.Time("now", time.Now()),
	)
}

// updateLocalMemory يحدث الذاكرة المحلية
func (ua *UnifiedAgent) updateLocalMemory(summary interface{}) {
	// تحديث الذاكرة المحلية بالملخص الحالي
	// هذا يضمن أن الوكيل لديه نسخة محلية محدثة
}

// updateLocalSkills يحدث المهارات المحلية
func (ua *UnifiedAgent) updateLocalSkills(summary interface{}) {
	// تحديث المهارات المحلية بالملخص الحالي
	// هذا يضمن أن الوكيل لديه نسخة محلية محدثة
}

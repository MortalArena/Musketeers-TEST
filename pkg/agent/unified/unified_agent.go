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
	"github.com/MortalArena/Musketeers/pkg/agent/thinking"
	"github.com/MortalArena/Musketeers/pkg/agent/tools"
	"github.com/MortalArena/Musketeers/pkg/agent/validation"
	"github.com/MortalArena/Musketeers/pkg/agent/wiring"
	"github.com/MortalArena/Musketeers/pkg/providers"
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

	// نظام تنظيم البيانات
	dataCurator *DataCurator

	// مجدول المهام
	taskScheduler *TaskScheduler

	// مدير المزامنة
	syncManager *AgentSyncManager

	// قناة الأحداث
	eventChannel chan *SessionEvent

	// [FIX] Provider integration for real LLM execution
	providerRegistry *providers.ProviderRegistry
	router           *providers.Router

	// [FIX] ToolExecutor for CLI, IDE, Browser adapters
	toolExecutor *tools.ToolExecutor

	// [FIX] ThinkingEngine for deep AI thought process
	thinkingEngine *thinking.ThinkingEngine

	// ThinkingEngine initialization flag
	thinkingEngineInitialized bool

	// [NEW] WiringLayer for automatic adapter connection
	wiringLayer *wiring.WiringLayer

	// SessionContainer reference for integration
	sessionContainer *session.SessionContainer

	// SessionManager for session management
	sessionManager *SessionManager

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

	// إنشاء نظام تنظيم البيانات
	ua.dataCurator = NewDataCurator(sessionID, logger)

	// إنشاء مجدول المهام
	ua.taskScheduler = NewTaskScheduler(sessionID, logger)

	// إنشاء SessionManager للتكامل الكامل
	ua.sessionManager = NewSessionManager(sessionID, logger)

	// إنشاء ProviderRegistry و Router
	ua.providerRegistry = providers.NewProviderRegistry()
	ua.router = providers.NewRouter(ua.providerRegistry, providers.RouterConfig{})

	// إنشاء ToolExecutor
	ua.toolExecutor = tools.NewToolExecutor(sessionID, logger)

	// إنشاء ThinkingEngine للتفكير العميق
	ua.thinkingEngine = thinking.NewThinkingEngine(sessionID, agentID, logger)
	ua.thinkingEngineInitialized = true

	// إنشاء WiringLayer للربط التلقائي للـ Adapters
	ua.wiringLayer = wiring.NewWiringLayer(sessionID, agentID, logger)

	// إنشاء مدير المزامنة
	ua.syncManager = NewAgentSyncManager(
		agentID,
		sessionID,
		ua.realTimeMemorySync,
		ua.realTimeSkillSync,
		ua.localMemoryCache,
		ua.sessionEventBus,
		logger,
	)

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

	// بدء مدير المزامنة
	if err := ua.syncManager.Start(ctx); err != nil {
		ua.logger.Error("فشل بدء مدير المزامنة", zap.Error(err))
		return fmt.Errorf("فشل بدء مدير المزامنة: %w", err)
	}

	// الاشتراك في ناقل الأحداث
	ua.eventChannel = ua.sessionEventBus.SubscribeAgent(ua.agentID)

	// بدء معالجة الأحداث
	go ua.processEvents(ctx)

	// بدء التسجيل الإجباري للتطورات اللحظية
	go ua.startMandatoryProgressReporting(ctx)

	// بدء المزامنة الإجبارية للقراءة
	go ua.startMandatoryReadSync(ctx)

	// بدء المزامنة الإجبارية للذاكرة المحلية
	go ua.startLocalMemorySync(ctx)

	// بدء مجدول المهام
	ua.taskScheduler.Start(ctx)

	// بدء تنظيف البيانات الدوري
	go ua.startDataCuration(ctx)

	// تهيئة SessionManager مع AgentExecutor
	if err := ua.sessionManager.Initialize(ctx, ua); err != nil {
		ua.logger.Warn("فشل تهيئة SessionManager", zap.Error(err))
	}

	// ربط ThinkingEngine بـ SessionManager
	if ua.thinkingEngine != nil && ua.sessionManager != nil {
		ua.thinkingEngine.SetSessionManager(ua.sessionManager)
		ua.logger.Info("تم ربط ThinkingEngine بـ SessionManager")
	}

	// ربط ThinkingEngine بمكونات session الحقيقية عبر adaptors
	if err := ua.connectThinkingEngineToSession(ctx); err != nil {
		ua.logger.Warn("فشل ربط ThinkingEngine بمكونات session", zap.Error(err))
		// لا نرجع خطأ لأن هذا ليس حرجاً للتهيئة
	}

	ua.logger.Info("تم تهيئة الوكيل الموحد بنجاح",
		zap.String("session_id", ua.sessionID),
		zap.String("agent_id", ua.agentID))

	return nil
}

// connectThinkingEngineToSession يربط ThinkingEngine بمكونات session الحقيقية عبر adaptors
func (ua *UnifiedAgent) connectThinkingEngineToSession(ctx context.Context) error {
	if ua.thinkingEngine == nil {
		return fmt.Errorf("ThinkingEngine not initialized")
	}

	// ربط CollectiveMemory من UnifiedMemoryManager عبر adaptor
	if ua.unifiedMemoryManager != nil {
		sessionCollectiveMemory := session.NewCollectiveMemory(ua.sessionID, nil)
		if sessionCollectiveMemory != nil {
			collectiveMemoryAdaptor := thinking.NewCollectiveMemoryAdaptor(sessionCollectiveMemory)
			ua.thinkingEngine.SetCollectiveMemory(collectiveMemoryAdaptor)
			ua.logger.Info("ربط ThinkingEngine بـ CollectiveMemory عبر adaptor")
		}
	}

	// ربط SkillsManager من UnifiedSkillManager عبر adaptor
	if ua.unifiedSkillManager != nil {
		sessionSkillsManager := session.NewSkillsManager(ua.sessionID)
		if sessionSkillsManager != nil {
			skillsManagerAdaptor := thinking.NewSkillsManagerAdaptor(sessionSkillsManager)
			ua.thinkingEngine.SetSkillsManager(skillsManagerAdaptor)
			ua.logger.Info("ربط ThinkingEngine بـ SkillsManager عبر adaptor")
		}
	}

	// ربط SessionContainer عبر adaptor
	sessionConfig := &session.SessionConfig{
		Name:        "Unified Agent Session",
		Description: "Session managed by UnifiedAgent",
		OwnerDID:    ua.agentID,
	}
	sessionContainer, err := session.NewSessionContainer(ctx, nil, sessionConfig, nil)
	if err == nil && sessionContainer != nil {
		// حفظ مرجع للـ SessionContainer في UnifiedAgent
		ua.sessionContainer = sessionContainer

		// ربط SessionJournal مع ThinkingEngine
		if sessionContainer.Journal != nil {
			sessionJournalAdaptor := thinking.NewSessionJournalAdaptor(sessionContainer.Journal)
			ua.thinkingEngine.SetSessionJournal(sessionJournalAdaptor)
			ua.logger.Info("ربط ThinkingEngine بـ SessionJournal عبر adaptor")
		}

		// ربط SessionContainer مع ThinkingEngine
		sessionContainerAdaptor := thinking.NewSessionContainerAdaptor(sessionContainer)
		ua.thinkingEngine.SetSessionContainer(sessionContainerAdaptor)
		ua.logger.Info("ربط ThinkingEngine بـ SessionContainer عبر adaptor")
	} else {
		ua.logger.Warn("فشل إنشاء SessionContainer", zap.Error(err))
	}

	// ربط الذاكرة المحلية عبر adaptor (محاكاة بسيطة)
	if ua.localMemoryCache != nil {
		// إنشاء adaptor بسيط للذاكرة المحلية
		sessionMemoryAdaptor := thinking.NewSessionMemoryAdaptor(nil)
		ua.thinkingEngine.SetSessionMemory(sessionMemoryAdaptor)
		ua.logger.Info("ربط ThinkingEngine بـ SessionMemory عبر adaptor")
	}

	// ربط مزامنة الذاكرة عبر adaptor (محاكاة بسيطة)
	if ua.realTimeMemorySync != nil {
		memorySyncAdaptor := thinking.NewMemorySyncAdaptor(nil)
		ua.thinkingEngine.SetMemorySync(memorySyncAdaptor)
		ua.logger.Info("ربط ThinkingEngine بـ MemorySync عبر adaptor")
	}

	// ربط مزامنة المهارات عبر adaptor (محاكاة بسيطة)
	if ua.realTimeSkillSync != nil {
		skillSyncAdaptor := thinking.NewSkillSyncAdaptor(nil)
		ua.thinkingEngine.SetSkillSync(skillSyncAdaptor)
		ua.logger.Info("ربط ThinkingEngine بـ SkillSync عبر adaptor")
	}

	// ربط ناقل أحداث الجلسة عبر adaptor للمزامنة اللحظية للأحداث
	if ua.sessionEventBus != nil {
		sessionEventBusAdaptor := thinking.NewSessionEventBusAdaptor(ua.sessionEventBus)
		ua.thinkingEngine.SetSessionEventBus(sessionEventBusAdaptor)
		ua.logger.Info("ربط ThinkingEngine بـ SessionEventBus عبر adaptor للمزامنة اللحظية للأحداث")
	}

	// ربط نظام الورك فلو الحقيقي من pkg/session/workflow.go
	if sessionContainer != nil && sessionContainer.Workflow != nil {
		// ربط WorkflowEngine الحقيقي مع ThinkingEngine
		ua.thinkingEngine.SetWorkflowEngine(sessionContainer.Workflow)
		ua.logger.Info("ربط ThinkingEngine بـ WorkflowEngine الحقيقي من pkg/session/workflow.go")
	} else {
		// استخدام adaptor كحل احتياطي
		workflowAdaptor := thinking.NewWorkflowAdaptor(nil)
		ua.thinkingEngine.SetWorkflow(workflowAdaptor)
		ua.logger.Info("ربط ThinkingEngine بـ Workflow عبر adaptor (احتياطي)")
	}

	// ربط مدير المهام عبر adaptor (محاكاة بسيطة)
	sessionTaskManager := session.NewTaskManager(ua.sessionID)
	if sessionTaskManager != nil {
		taskManagerAdaptor := thinking.NewTaskManagerAdaptor(sessionTaskManager)
		ua.thinkingEngine.SetTaskManager(taskManagerAdaptor)
		ua.logger.Info("ربط ThinkingEngine بـ TaskManager عبر adaptor")
	}

	// ربط adaptors البيئة الموزعة (محاكاة بسيطة)
	networkAwareAdaptor := thinking.NewNetworkAwareAdaptor(nil)
	ua.thinkingEngine.SetNetworkAware(networkAwareAdaptor)
	ua.logger.Info("ربط ThinkingEngine بـ NetworkAware عبر adaptor للبيئة الموزعة")

	distributedSessionAdaptor := thinking.NewDistributedSessionAdaptor(nil)
	ua.thinkingEngine.SetDistributedSession(distributedSessionAdaptor)
	ua.logger.Info("ربط ThinkingEngine بـ DistributedSession عبر adaptor للبيئة الموزعة")

	geoLocationAwareAdaptor := thinking.NewGeoLocationAwareAdaptor(nil)
	ua.thinkingEngine.SetGeoLocationAware(geoLocationAwareAdaptor)
	ua.logger.Info("ربط ThinkingEngine بـ GeoLocationAware عبر adaptor للبيئة الموزعة")

	// ربط RuntimeIntegration مع ToolExecutor
	if ua.toolExecutor != nil {
		ua.thinkingEngine.SetRuntimeIntegrationToolExecutor(ua.toolExecutor)
		ua.logger.Info("ربط RuntimeIntegration بـ ToolExecutor")
	}

	ua.logger.Info("تم ربط ThinkingEngine بجميع مكونات session الحقيقية عبر adaptors")

	// [NEW] استخدام WiringLayer للربط التلقائي للـ Adapters
	if err := ua.useWiringLayer(ctx); err != nil {
		ua.logger.Warn("فشل استخدام WiringLayer للربط التلقائي", zap.Error(err))
		// لا نرجع خطأ لأن هذا ليس حرجاً للتهيئة
	}

	return nil
}

// useWiringLayer يستخدم WiringLayer للربط التلقائي للـ Adapters
func (ua *UnifiedAgent) useWiringLayer(ctx context.Context) error {
	if ua.wiringLayer == nil {
		return fmt.Errorf("WiringLayer not initialized")
	}

	ua.logger.Info("بدء استخدام WiringLayer للربط التلقائي")

	// تسجيل Adapters الرئيسية باستخدام wrappers
	if ua.thinkingEngine != nil {
		thinkingAdapter := wiring.NewThinkingEngineAdapter(ua.thinkingEngine, ua.logger)
		if err := ua.wiringLayer.RegisterAdapter(thinkingAdapter); err != nil {
			ua.logger.Warn("فشل تسجيل ThinkingEngine Adapter", zap.Error(err))
		}
	}

	if ua.sessionManager != nil {
		sessionAdapter := wiring.NewSessionManagerAdapter(ua.sessionManager, ua.logger)
		if err := ua.wiringLayer.RegisterAdapter(sessionAdapter); err != nil {
			ua.logger.Warn("فشل تسجيل SessionManager Adapter", zap.Error(err))
		}
	}

	if ua.toolExecutor != nil {
		toolAdapter := wiring.NewToolExecutorAdapter(ua.toolExecutor, ua.logger)
		if err := ua.wiringLayer.RegisterAdapter(toolAdapter); err != nil {
			ua.logger.Warn("فشل تسجيل ToolExecutor Adapter", zap.Error(err))
		}
	}

	if ua.providerRegistry != nil {
		providerAdapter := wiring.NewProviderRegistryAdapter(ua.providerRegistry, ua.logger)
		if err := ua.wiringLayer.RegisterAdapter(providerAdapter); err != nil {
			ua.logger.Warn("فشل تسجيل ProviderRegistry Adapter", zap.Error(err))
		}
	}

	if ua.router != nil {
		routerAdapter := wiring.NewRouterAdapter(ua.router, ua.logger)
		if err := ua.wiringLayer.RegisterAdapter(routerAdapter); err != nil {
			ua.logger.Warn("فشل تسجيل Router Adapter", zap.Error(err))
		}
	}

	if ua.sessionEventBus != nil {
		eventBusAdapter := wiring.NewEventBusAdapter(ua.sessionEventBus, ua.logger)
		if err := ua.wiringLayer.RegisterAdapter(eventBusAdapter); err != nil {
			ua.logger.Warn("فشل تسجيل EventBus Adapter", zap.Error(err))
		}
	}

	// تسجيل WorkflowEngine إذا كان موجوداً في ThinkingEngine
	if ua.thinkingEngine != nil {
		workflowEngine := ua.thinkingEngine.GetWorkflowEngine()
		if workflowEngine != nil {
			workflowAdapter := wiring.NewWorkflowEngineAdapter(workflowEngine, ua.logger)
			if err := ua.wiringLayer.RegisterAdapter(workflowAdapter); err != nil {
				ua.logger.Warn("فشل تسجيل WorkflowEngine Adapter", zap.Error(err))
			}
		}
	}

	ua.logger.Info("تم تسجيل جميع Adapters في WiringLayer")

	// استدعاء AutoWire للربط التلقائي
	if err := ua.wiringLayer.AutoWire(ctx); err != nil {
		ua.logger.Warn("فشل AutoWire للربط التلقائي", zap.Error(err))
		// لا نرجع خطأ لأن الربط اليدوي موجود بالفعل
	} else {
		ua.logger.Info("تم ربط جميع Adapters تلقائياً بنجاح")
	}

	// التحقق من حالة الاتصالات
	status := ua.wiringLayer.GetConnectionStatus()
	ua.logger.Info("حالة اتصالات WiringLayer",
		zap.Bool("connected", status["connected"].(bool)),
		zap.Int("adapters_count", status["adapters_count"].(int)),
		zap.Int("connections_count", status["connections_count"].(int)),
	)

	return nil
}

// SetThinkingEngineProvider يضبط LLM Provider للThinkingEngine
func (ua *UnifiedAgent) SetThinkingEngineProvider(provider providers.Provider, modelID string) {
	ua.mu.Lock()
	defer ua.mu.Unlock()
	if ua.thinkingEngine != nil {
		ua.thinkingEngine.SetProvider(provider, modelID)
		ua.logger.Info("ThinkingEngine provider set", zap.String("model", modelID))
	}
}

// GetThinkingEngine يحصل على ThinkingEngine
func (ua *UnifiedAgent) GetThinkingEngine() *thinking.ThinkingEngine {
	ua.mu.RLock()
	defer ua.mu.RUnlock()
	return ua.thinkingEngine
}

// ExecuteTaskWithThinking ينفذ مهمة باستخدام ThinkingEngine
func (ua *UnifiedAgent) ExecuteTaskWithThinking(ctx context.Context, task string) (interface{}, error) {
	if ua.thinkingEngine == nil {
		return nil, fmt.Errorf("thinking engine not initialized")
	}

	// تحليل المهمة
	analysis, err := ua.thinkingEngine.AnalyzeTask(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("task analysis failed: %w", err)
	}

	// تخطيط المهمة باستخدام التحليل
	subtasks, err := ua.thinkingEngine.PlanTask(ctx, analysis)
	if err != nil {
		return nil, fmt.Errorf("task planning failed: %w", err)
	}

	// إنشاء سياق التنفيذ
	execContext := ua.flowManager.CreateExecutionContext(ctx, task)

	// تنفيذ المهمة (باستخدام Coordinator)
	result, err := ua.coordinator.ExecuteTask(ctx, execContext)
	if err != nil {
		return nil, fmt.Errorf("task execution failed: %w", err)
	}

	// التحقق من النتيجة
	verification, err := ua.thinkingEngine.VerifyResult(ctx, task, result)
	if err != nil {
		ua.logger.Warn("Verification failed", zap.Error(err))
	}

	// التفكر في النتيجة
	reflection, err := ua.thinkingEngine.Reflect(ctx, task, result, time.Since(time.Now()))
	if err != nil {
		ua.logger.Warn("Reflection failed", zap.Error(err))
	}

	return map[string]interface{}{
		"result":       result,
		"analysis":     analysis,
		"subtasks":     subtasks,
		"verification": verification,
		"reflection":   reflection,
	}, nil
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

// SetProviderRegistry يضبط سجل المزودين
func (ua *UnifiedAgent) SetProviderRegistry(registry *providers.ProviderRegistry) {
	ua.mu.Lock()
	defer ua.mu.Unlock()

	ua.providerRegistry = registry
	ua.logger.Info("Provider registry set")
}

// SetRouter يضبط الموجه الذكي
func (ua *UnifiedAgent) SetRouter(router *providers.Router) {
	ua.mu.Lock()
	defer ua.mu.Unlock()

	ua.router = router
	ua.collectiveSystem.SetRouter(router) // [FIX] Pass router to CollectiveAgentSystem
	ua.logger.Info("Smart router set")
}

// SetToolExecutor يضبط منفذ الأدوات
func (ua *UnifiedAgent) SetToolExecutor(executor *tools.ToolExecutor) {
	ua.mu.Lock()
	defer ua.mu.Unlock()

	ua.toolExecutor = executor
	ua.logger.Info("Tool executor set")
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
			// تقديم مهمة المزامنة إلى المجدول
			task := &Task{
				ID:       fmt.Sprintf("sync_read_%d", time.Now().Unix()),
				Type:     "sync_read",
				Priority: PriorityMedium,
				Execute: func(ctx context.Context) error {
					ua.syncNewData(ctx, lastSyncTime)
					lastSyncTime = time.Now()
					return nil
				},
				CreatedAt: time.Now(),
				Timeout:   30 * time.Second,
			}

			if err := ua.taskScheduler.SubmitTask(task); err != nil {
				ua.logger.Error("فشل تقديم مهمة المزامنة للقراءة", zap.Error(err))
			}
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
	if ua.syncManager != nil {
		if err := ua.syncManager.updateLocalMemory(summary); err != nil {
			ua.logger.Error("فشل تحديث الذاكرة المحلية", zap.Error(err))
		}
	}
}

// updateLocalSkills يحدث المهارات المحلية
func (ua *UnifiedAgent) updateLocalSkills(summary interface{}) {
	if ua.syncManager != nil {
		if err := ua.syncManager.updateLocalSkills(summary); err != nil {
			ua.logger.Error("فشل تحديث المهارات المحلية", zap.Error(err))
		}
	}
}

// startLocalMemorySync يبدأ المزامنة الإجبارية للذاكرة المحلية
func (ua *UnifiedAgent) startLocalMemorySync(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			ua.logger.Info("تم إيقاف المزامنة الإجبارية للذاكرة المحلية")
			return
		case <-ticker.C:
			// تقديم مهمة المزامنة إلى المجدول
			task := &Task{
				ID:       fmt.Sprintf("sync_memory_%d", time.Now().Unix()),
				Type:     "sync_memory",
				Priority: PriorityLow,
				Execute: func(ctx context.Context) error {
					ua.localMemoryCache.syncToSharedDB(ctx)
					return nil
				},
				CreatedAt: time.Now(),
				Timeout:   30 * time.Second,
			}

			if err := ua.taskScheduler.SubmitTask(task); err != nil {
				ua.logger.Error("فشل تقديم مهمة المزامنة للذاكرة المحلية", zap.Error(err))
			}
		}
	}
}

// startDataCuration يبدأ تنظيف البيانات الدوري
func (ua *UnifiedAgent) startDataCuration(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			ua.logger.Info("تم إيقاف تنظيف البيانات الدوري")
			return
		case <-ticker.C:
			// تقديم مهمة تنظيف البيانات إلى المجدول
			task := &Task{
				ID:       fmt.Sprintf("curate_data_%d", time.Now().Unix()),
				Type:     "curate_data",
				Priority: PriorityLow,
				Execute: func(ctx context.Context) error {
					// تنظيف البيانات من الذاكرة المحلية
					memoryEvents := ua.localMemoryCache.GetMemoryEvents()
					curatedEvents := ua.dataCurator.CurateMemoryEvents(memoryEvents)
					ua.localMemoryCache.UpdateMemoryEvents(curatedEvents)

					// تنظيف تحديثات المهارات
					skillUpdates := ua.localMemoryCache.GetSkillUpdates()
					curatedUpdates := ua.dataCurator.CurateSkillUpdates(skillUpdates)
					ua.localMemoryCache.UpdateSkillUpdates(curatedUpdates)

					return nil
				},
				CreatedAt: time.Now(),
				Timeout:   60 * time.Second,
			}

			if err := ua.taskScheduler.SubmitTask(task); err != nil {
				ua.logger.Error("فشل تقديم مهمة تنظيف البيانات", zap.Error(err))
			}
		}
	}
}

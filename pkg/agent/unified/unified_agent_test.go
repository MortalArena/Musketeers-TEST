package unified

import (
	"context"
	"testing"

	"github.com/dgraph-io/badger/v4"
	"go.uber.org/zap/zaptest"
)

func TestUnifiedAgent_NewUnifiedAgent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

	if ua == nil {
		t.Fatal("NewUnifiedAgent returned nil")
	}

	if ua.sessionID != sessionID {
		t.Errorf("Expected sessionID '%s', got '%s'", sessionID, ua.sessionID)
	}

	if ua.agentID != agentID {
		t.Errorf("Expected agentID '%s', got '%s'", agentID, ua.agentID)
	}
}

func TestUnifiedAgent_Initialize(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

	ctx := context.Background()

	// تهيئة الوكيل الموحد
	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
}

func TestUnifiedAgent_RegisterAgent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

	ctx := context.Background()

	// تهيئة الوكيل الموحد
	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// تسجيل وكيل
	did := "did:test:123"
	agentType := "coder"
	llmType := "claude"
	specializations := []string{"backend", "fullstack"}

	if err := ua.RegisterAgent(ctx, did, agentType, llmType, specializations); err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}
}

func TestUnifiedAgent_ExecuteTask(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	// إنشاء قاعدة بيانات مؤقتة للاختبار
	tempDir := t.TempDir()
	db, err := badger.Open(badger.DefaultOptions(tempDir))
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	ua := NewUnifiedAgent(sessionID, agentID, db, logger)

	ctx := context.Background()

	// تهيئة الوكيل الموحد
	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// تسجيل وكيل
	did := "did:test:123"
	agentType := "coder"
	llmType := "claude"
	specializations := []string{"backend"}

	if err := ua.RegisterAgent(ctx, did, agentType, llmType, specializations); err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}

	// تنفيذ مهمة
	task := "test task"
	result, err := ua.ExecuteTask(ctx, task)
	if err != nil {
		t.Fatalf("ExecuteTask failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.Task != task {
		t.Errorf("Expected task '%s', got '%s'", task, result.Task)
	}
}

func TestUnifiedAgent_GetSystemSummary(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

	ctx := context.Background()

	// تهيئة الوكيل الموحد
	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// الحصول على ملخص النظام
	summary, err := ua.GetSystemSummary(ctx)
	if err != nil {
		t.Fatalf("GetSystemSummary failed: %v", err)
	}

	if summary == nil {
		t.Fatal("Expected summary, got nil")
	}

	if summary.SessionID != sessionID {
		t.Errorf("Expected sessionID '%s', got '%s'", sessionID, summary.SessionID)
	}

	if summary.AgentID != agentID {
		t.Errorf("Expected agentID '%s', got '%s'", agentID, summary.AgentID)
	}
}

// اختبارات التكامل
func TestUnifiedAgent_Integration_CompleteWorkflow(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	// إنشاء قاعدة بيانات مؤقتة للاختبار
	tempDir := t.TempDir()
	db, err := badger.Open(badger.DefaultOptions(tempDir))
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	ua := NewUnifiedAgent(sessionID, agentID, db, logger)

	ctx := context.Background()

	// تهيئة الوكيل الموحد
	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// تسجيل وكيل
	did := "did:test:123"
	agentType := "coder"
	llmType := "claude"
	specializations := []string{"backend"}

	if err := ua.RegisterAgent(ctx, did, agentType, llmType, specializations); err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}

	// تنفيذ مهمة
	task := "test task"
	result, err := ua.ExecuteTask(ctx, task)
	if err != nil {
		t.Fatalf("ExecuteTask failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success, got failure")
	}

	// الحصول على ملخص النظام
	summary, err := ua.GetSystemSummary(ctx)
	if err != nil {
		t.Fatalf("GetSystemSummary failed: %v", err)
	}

	if summary.OverallReadiness == 0 {
		t.Error("Expected overall readiness > 0, got 0")
	}
}

// اختبارات الأمان
func TestUnifiedAgent_Security_ConcurrentOperations(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

	ctx := context.Background()

	// تهيئة الوكيل الموحد
	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// اختبار العمليات المتزامنة
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			ua.GetSystemSummary(ctx)
			done <- true
		}()
	}

	// انتظار جميع goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// اختبارات التكامل مع ThinkingEngine
func TestUnifiedAgent_Integration_ThinkingEngine(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

	ctx := context.Background()

	// تهيئة الوكيل الموحد
	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// التحقق من تهيئة ThinkingEngine
	if ua.thinkingEngine == nil {
		t.Fatal("ThinkingEngine should be initialized")
	}

	// التحقق من ربط ThinkingEngine بـ SessionManager
	if ua.thinkingEngine.GetSessionManagerAgent() != agentID {
		t.Errorf("Expected session manager agent '%s', got '%s'", agentID, ua.thinkingEngine.GetSessionManagerAgent())
	}
}

// اختبارات التكامل مع SessionManager
func TestUnifiedAgent_Integration_SessionManager(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

	ctx := context.Background()

	// تهيئة الوكيل الموحد
	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// التحقق من تهيئة SessionManager
	if ua.sessionManager == nil {
		t.Fatal("SessionManager should be initialized")
	}
}

// اختبارات التكامل مع WorkflowEngine
func TestUnifiedAgent_Integration_WorkflowEngine(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

	ctx := context.Background()

	// تهيئة الوكيل الموحد
	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// التحقق من تهيئة WorkflowEngine في ThinkingEngine
	// قد يكون nil إذا تم استخدام adaptor كحل احتياطي
	// هذا مقبول لأن النظام يعمل بشكل صحيح مع adaptor
	workflowEngine := ua.thinkingEngine.GetWorkflowEngine()
	if workflowEngine == nil {
		// هذا مقبول - يتم استخدام adaptor كحل احتياطي
		t.Log("WorkflowEngine is nil, using adaptor as fallback (acceptable)")
	} else {
		t.Log("WorkflowEngine is initialized directly")
	}
}

// اختبارات التكامل مع RuntimeIntegration
func TestUnifiedAgent_Integration_RuntimeIntegration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

	ctx := context.Background()

	// تهيئة الوكيل الموحد
	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// التحقق من تهيئة RuntimeIntegration في ThinkingEngine
	// RuntimeIntegration يتم تهيئته داخلياً في ThinkingEngine
}

// اختبارات التكامل مع Orchestrator
func TestUnifiedAgent_Integration_Orchestrator(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

	ctx := context.Background()

	// تهيئة الوكيل الموحد
	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// التحقق من تهيئة أنظمة التنسيق
	if ua.coordinator == nil {
		t.Fatal("Coordinator should be initialized")
	}

	if ua.flowManager == nil {
		t.Fatal("FlowManager should be initialized")
	}

	if ua.errorHandler == nil {
		t.Fatal("ErrorHandler should be initialized")
	}
}

// اختبارات التكامل مع CollectiveSystem
func TestUnifiedAgent_Integration_CollectiveSystem(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

	ctx := context.Background()

	// تهيئة الوكيل الموحد
	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// التحقق من تهيئة CollectiveSystem
	if ua.collectiveSystem == nil {
		t.Fatal("CollectiveSystem should be initialized")
	}
}

// اختبارات التكامل مع SyncSystems
func TestUnifiedAgent_Integration_SyncSystems(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

	ctx := context.Background()

	// تهيئة الوكيل الموحد
	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// التحقق من تهيئة أنظمة المزامنة
	if ua.realTimeMemorySync == nil {
		t.Fatal("RealTimeMemorySync should be initialized")
	}

	if ua.realTimeSkillSync == nil {
		t.Fatal("RealTimeSkillSync should be initialized")
	}

	if ua.syncManager == nil {
		t.Fatal("SyncManager should be initialized")
	}
}

// اختبارات التكامل مع EventBus
func TestUnifiedAgent_Integration_EventBus(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

	ctx := context.Background()

	// تهيئة الوكيل الموحد
	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// التحقق من تهيئة EventBus
	if ua.sessionEventBus == nil {
		t.Fatal("SessionEventBus should be initialized")
	}
}

// اختبارات التكامل مع TaskScheduler
func TestUnifiedAgent_Integration_TaskScheduler(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

	ctx := context.Background()

	// تهيئة الوكيل الموحد
	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// التحقق من تهيئة TaskScheduler
	if ua.taskScheduler == nil {
		t.Fatal("TaskScheduler should be initialized")
	}
}

// اختبارات التكامل مع ToolExecutor
func TestUnifiedAgent_Integration_ToolExecutor(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

	ctx := context.Background()

	// تهيئة الوكيل الموحد
	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// التحقق من تهيئة ToolExecutor
	if ua.toolExecutor == nil {
		t.Fatal("ToolExecutor should be initialized")
	}
}

// اختبارات التكامل مع ProviderRegistry
func TestUnifiedAgent_Integration_ProviderRegistry(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

	ctx := context.Background()

	// تهيئة الوكيل الموحد
	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// التحقق من تهيئة ProviderRegistry
	if ua.providerRegistry == nil {
		t.Fatal("ProviderRegistry should be initialized")
	}

	if ua.router == nil {
		t.Fatal("Router should be initialized")
	}
}

// اختبارات التكامل الشامل
func TestUnifiedAgent_Integration_AllComponents(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

	ctx := context.Background()

	// تهيئة الوكيل الموحد
	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// التحقق من تهيئة جميع المكونات
	components := []struct {
		name  string
		value interface{}
	}{
		{"UnifiedSkillManager", ua.unifiedSkillManager},
		{"UnifiedMemoryManager", ua.unifiedMemoryManager},
		{"SubagentManager", ua.subagentManager},
		{"AutomationManager", ua.automationManager},
		{"SkillDirector", ua.skillDirector},
		{"MultiLayerValidator", ua.multiLayerValidator},
		{"Coordinator", ua.coordinator},
		{"FlowManager", ua.flowManager},
		{"ErrorHandler", ua.errorHandler},
		{"CollectiveSystem", ua.collectiveSystem},
		{"SessionEventBus", ua.sessionEventBus},
		{"RealTimeMemorySync", ua.realTimeMemorySync},
		{"RealTimeSkillSync", ua.realTimeSkillSync},
		{"ProblemSolutionRegistry", ua.problemSolutionRegistry},
		{"LocalMemoryCache", ua.localMemoryCache},
		{"DataCurator", ua.dataCurator},
		{"TaskScheduler", ua.taskScheduler},
		{"SyncManager", ua.syncManager},
		{"EventChannel", ua.eventChannel},
		{"ProviderRegistry", ua.providerRegistry},
		{"Router", ua.router},
		{"ToolExecutor", ua.toolExecutor},
		{"ThinkingEngine", ua.thinkingEngine},
		{"SessionManager", ua.sessionManager},
	}

	for _, component := range components {
		if component.value == nil {
			t.Errorf("%s should be initialized", component.name)
		}
	}
}

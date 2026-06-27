package unified

import (
	"context"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
	"go.uber.org/zap"
)

func TestSessionManager_NewSessionManager(t *testing.T) {
	logger := zap.NewNop()
	sessionID := "test_session"

	sm := NewSessionManager(sessionID, logger)
	if sm == nil {
		t.Fatal("فشل إنشاء SessionManager")
	}

	if sm.sessionID != sessionID {
		t.Errorf("sessionID غير متطابق: got %s, want %s", sm.sessionID, sessionID)
	}

	if sm.sessionStatus != SessionStatusInitializing {
		t.Errorf("sessionStatus غير متطابق: got %s, want %s", sm.sessionStatus, SessionStatusInitializing)
	}
}

func TestSessionManager_Initialize(t *testing.T) {
	logger := zap.NewNop()
	sessionID := "test_session"

	// إنشاء قاعدة بيانات مؤقتة
	opts := badger.DefaultOptions("")
	opts.InMemory = true
	db, err := badger.Open(opts)
	if err != nil {
		t.Fatalf("فشل فتح قاعدة البيانات: %v", err)
	}
	defer db.Close()

	sm := NewSessionManager(sessionID, logger)
	ua := NewUnifiedAgent(sessionID, "test_agent", db, logger)
	sm.SetEventBus(ua.sessionEventBus) // [FIX] EventBus مطلوب قبل Initialize

	ctx := context.Background()
	if err := sm.Initialize(ctx, ua); err != nil {
		t.Fatalf("فشل تهيئة SessionManager: %v", err)
	}

	if sm.sessionStatus != SessionStatusActive {
		t.Errorf("sessionStatus غير متطابق: got %s, want %s", sm.sessionStatus, SessionStatusActive)
	}

	if sm.agentExecutor != ua {
		t.Error("agentExecutor غير متطابق")
	}
}

func TestSessionManager_ReceivePrompt(t *testing.T) {
	logger := zap.NewNop()
	sessionID := "test_session"

	sm := NewSessionManager(sessionID, logger)

	ctx := context.Background()
	prompt := "اختبار البرومبت"
	if err := sm.ReceivePrompt(ctx, prompt); err != nil {
		t.Fatalf("فشل استقبال البرومبت: %v", err)
	}

	if sm.clientPrompt != prompt {
		t.Errorf("clientPrompt غير متطابق: got %s, want %s", sm.clientPrompt, prompt)
	}
}

func TestSessionManager_EvaluateTask(t *testing.T) {
	logger := zap.NewNop()
	sessionID := "test_session"

	sm := NewSessionManager(sessionID, logger)

	ctx := context.Background()
	sm.ReceivePrompt(ctx, "برومبت بسيط")

	evaluation, err := sm.EvaluateTask(ctx)
	if err != nil {
		t.Fatalf("فشل تقييم المهمة: %v", err)
	}

	if evaluation.SessionID != sessionID {
		t.Errorf("SessionID غير متطابق: got %s, want %s", evaluation.SessionID, sessionID)
	}

	if evaluation.Complexity == "" {
		t.Error("Complexity فارغ")
	}

	if evaluation.RecommendedStrategy == "" {
		t.Error("RecommendedStrategy فارغ")
	}
}

func TestSessionManager_DecomposeTask(t *testing.T) {
	logger := zap.NewNop()
	sessionID := "test_session"

	sm := NewSessionManager(sessionID, logger)

	ctx := context.Background()
	sm.ReceivePrompt(ctx, "برومبت بسيط")

	evaluation, _ := sm.EvaluateTask(ctx)

	tasks, err := sm.DecomposeTask(ctx, evaluation)
	if err != nil {
		t.Fatalf("فشل تفكيك المهمة: %v", err)
	}

	if len(tasks) == 0 {
		t.Error("لم يتم إنشاء أي مهام")
	}
}

func TestSessionManager_DistributeTasks(t *testing.T) {
	logger := zap.NewNop()
	sessionID := "test_session"

	sm := NewSessionManager(sessionID, logger)

	ctx := context.Background()
	sm.ReceivePrompt(ctx, "برومبت بسيط")

	evaluation, _ := sm.EvaluateTask(ctx)
	tasks, _ := sm.DecomposeTask(ctx, evaluation)

	if err := sm.DistributeTasks(ctx, tasks); err != nil {
		t.Fatalf("فشل توزيع المهام: %v", err)
	}

	if len(sm.activeTasks) != len(tasks) {
		t.Errorf("activeTasks غير متطابق: got %d, want %d", len(sm.activeTasks), len(tasks))
	}
}

func TestSessionManager_GetSessionSummary(t *testing.T) {
	logger := zap.NewNop()
	sessionID := "test_session"

	sm := NewSessionManager(sessionID, logger)

	ctx := context.Background()
	sm.ReceivePrompt(ctx, "برومبت بسيط")

	summary, err := sm.GetSessionSummary(ctx)
	if err != nil {
		t.Fatalf("فشل الحصول على ملخص الجلسة: %v", err)
	}

	if summary.SessionID != sessionID {
		t.Errorf("SessionID غير متطابق: got %s, want %s", summary.SessionID, sessionID)
	}

	if summary.ClientPrompt != "برومبت بسيط" {
		t.Errorf("ClientPrompt غير متطابق: got %s, want %s", summary.ClientPrompt, "برومبت بسيط")
	}
}

func TestSessionManager_evaluateComplexity(t *testing.T) {
	logger := zap.NewNop()
	sessionID := "test_session"

	sm := NewSessionManager(sessionID, logger)

	// اختبار تعقيد منخفض
	sm.clientPrompt = "قصير"
	if complexity := sm.evaluateComplexity(); complexity != ComplexityLow {
		t.Errorf("تعقيد منخفض غير متطابق: got %s, want %s", complexity, ComplexityLow)
	}

	// اختبار تعقيد متوسط
	sm.clientPrompt = string(make([]byte, 300))
	if complexity := sm.evaluateComplexity(); complexity != ComplexityMedium {
		t.Errorf("تعقيد متوسط غير متطابق: got %s, want %s", complexity, ComplexityMedium)
	}

	// اختبار تعقيد عالي
	sm.clientPrompt = string(make([]byte, 600))
	if complexity := sm.evaluateComplexity(); complexity != ComplexityHigh {
		t.Errorf("تعقيد عالي غير متطابق: got %s, want %s", complexity, ComplexityHigh)
	}

	// اختبار تعقيد حرج
	sm.clientPrompt = string(make([]byte, 1200))
	if complexity := sm.evaluateComplexity(); complexity != ComplexityCritical {
		t.Errorf("تعقيد حرج غير متطابق: got %s, want %s", complexity, ComplexityCritical)
	}
}

func TestSessionManager_recommendStrategy(t *testing.T) {
	logger := zap.NewNop()
	sessionID := "test_session"

	sm := NewSessionManager(sessionID, logger)

	// اختبار استراتيجية منخفضة
	sm.clientPrompt = "قصير"
	if strategy := sm.recommendStrategy(); strategy != StrategySequential {
		t.Errorf("استراتيجية منخفضة غير متطابقة: got %s, want %s", strategy, StrategySequential)
	}

	// اختبار استراتيجية متوسطة
	sm.clientPrompt = string(make([]byte, 300))
	if strategy := sm.recommendStrategy(); strategy != StrategySequential {
		t.Errorf("استراتيجية متوسطة غير متطابقة: got %s, want %s", strategy, StrategySequential)
	}

	// اختبار استراتيجية عالية
	sm.clientPrompt = string(make([]byte, 600))
	if strategy := sm.recommendStrategy(); strategy != StrategyMixed {
		t.Errorf("استراتيجية عالية غير متطابقة: got %s, want %s", strategy, StrategyMixed)
	}

	// اختبار استراتيجية حرجة
	sm.clientPrompt = string(make([]byte, 1200))
	if strategy := sm.recommendStrategy(); strategy != StrategyMixed {
		t.Errorf("استراتيجية حرجة غير متطابقة: got %s, want %s", strategy, StrategyMixed)
	}
}

func TestSessionManager_estimateTime(t *testing.T) {
	logger := zap.NewNop()
	sessionID := "test_session"

	sm := NewSessionManager(sessionID, logger)

	// اختبار تقدير الوقت منخفض
	sm.clientPrompt = "قصير"
	if duration := sm.estimateTime(); duration != 5*time.Minute {
		t.Errorf("تقدير الوقت منخفض غير متطابق: got %v, want %v", duration, 5*time.Minute)
	}

	// اختبار تقدير الوقت متوسط
	sm.clientPrompt = string(make([]byte, 300))
	if duration := sm.estimateTime(); duration != 30*time.Minute {
		t.Errorf("تقدير الوقت متوسط غير متطابق: got %v, want %v", duration, 30*time.Minute)
	}

	// اختبار تقدير الوقت عالي
	sm.clientPrompt = string(make([]byte, 600))
	if duration := sm.estimateTime(); duration != 2*time.Hour {
		t.Errorf("تقدير الوقت عالي غير متطابق: got %v, want %v", duration, 2*time.Hour)
	}

	// اختبار تقدير الوقت حرج
	sm.clientPrompt = string(make([]byte, 1200))
	if duration := sm.estimateTime(); duration != 8*time.Hour {
		t.Errorf("تقدير الوقت حرج غير متطابق: got %v, want %v", duration, 8*time.Hour)
	}
}

func TestSessionManager_determineRequiredAgents(t *testing.T) {
	logger := zap.NewNop()
	sessionID := "test_session"

	sm := NewSessionManager(sessionID, logger)

	// Auto mode: لا ترجع أدواراً وهمية — Session Manager Agent يقرر
	sm.clientPrompt = "قصير"
	agents := sm.determineRequiredAgents()
	if len(agents) != 0 {
		t.Errorf("Auto mode: توقعت قائمة فارغة لكن وجدت: %v", agents)
	}

	// Auto mode مع برومبت طويل
	sm.clientPrompt = string(make([]byte, 1200))
	agents = sm.determineRequiredAgents()
	if len(agents) != 0 {
		t.Errorf("Auto mode مع برومبت طويل: توقعت قائمة فارغة لكن وجدت: %v", agents)
	}

	// Manual mode: ترجع الوكلاء المعينين يدوياً
	sm.SetMode(SessionModeManual)
	sm.SetManualAssignments(map[string]string{
		"agent-1": "backend",
		"agent-2": "frontend",
	})
	agents = sm.determineRequiredAgents()
	if len(agents) != 2 {
		t.Errorf("Manual mode: توقعت وكيلين لكن وجدت %d: %v", len(agents), agents)
	}
}

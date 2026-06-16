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

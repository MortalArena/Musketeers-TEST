package integration

import (
	"context"
	"testing"

	"github.com/MortalArena/Musketeers/pkg/session"
	"github.com/dgraph-io/badger/v4"
	"go.uber.org/zap/zaptest"
)

// TestCollectiveAgentSystem_NewCollectiveAgentSystem اختبار إنشاء النظام الجماعي
func TestCollectiveAgentSystem_NewCollectiveAgentSystem(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	sessionSkills := session.NewSkillsManager(sessionID)
	sessionMemory := session.NewCollectiveMemory(sessionID, nil)

	cas := NewCollectiveAgentSystem(sessionID, sessionSkills, sessionMemory, logger)

	if cas == nil {
		t.Fatal("NewCollectiveAgentSystem returned nil")
	}

	if cas.sessionID != sessionID {
		t.Errorf("Expected sessionID '%s', got '%s'", sessionID, cas.sessionID)
	}
}

// TestCollectiveAgentSystem_RegisterAgent اختبار تسجيل وكيل
func TestCollectiveAgentSystem_RegisterAgent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	sessionSkills := session.NewSkillsManager(sessionID)
	sessionMemory := session.NewCollectiveMemory(sessionID, nil)

	cas := NewCollectiveAgentSystem(sessionID, sessionSkills, sessionMemory, logger)

	ctx := context.Background()
	did := "did:test:123"
	agentType := "coder"
	llmType := "claude"
	specializations := []string{"backend"}

	if err := cas.RegisterAgent(ctx, did, agentType, llmType, specializations); err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}
}

// TestCollectiveAgentSystem_ExecuteTask اختبار تنفيذ مهمة
func TestCollectiveAgentSystem_ExecuteTask(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	sessionSkills := session.NewSkillsManager(sessionID)

	// إنشاء قاعدة بيانات مؤقتة للاختبار
	tempDir := t.TempDir()
	db, err := badger.Open(badger.DefaultOptions(tempDir))
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	sessionMemory := session.NewCollectiveMemory(sessionID, db)

	cas := NewCollectiveAgentSystem(sessionID, sessionSkills, sessionMemory, logger)

	ctx := context.Background()
	did := "did:test:123"
	agentType := "coder"
	llmType := "claude"
	specializations := []string{"backend"}

	if err := cas.RegisterAgent(ctx, did, agentType, llmType, specializations); err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}

	task := "test task"
	result, err := cas.ExecuteTask(ctx, task, did)
	if err != nil {
		t.Fatalf("ExecuteTask failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}
}

// TestCollectiveAgentSystem_GetSystemSummary اختبار الحصول على ملخص النظام
func TestCollectiveAgentSystem_GetSystemSummary(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	sessionSkills := session.NewSkillsManager(sessionID)
	sessionMemory := session.NewCollectiveMemory(sessionID, nil)

	cas := NewCollectiveAgentSystem(sessionID, sessionSkills, sessionMemory, logger)

	ctx := context.Background()
	summary, err := cas.GetSystemSummary(ctx)
	if err != nil {
		t.Fatalf("GetSystemSummary failed: %v", err)
	}

	if summary == nil {
		t.Fatal("Expected summary, got nil")
	}
}

// اختبارات الأمان
func TestCollectiveAgentSystem_Security_ConcurrentOperations(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	sessionSkills := session.NewSkillsManager(sessionID)
	sessionMemory := session.NewCollectiveMemory(sessionID, nil)

	cas := NewCollectiveAgentSystem(sessionID, sessionSkills, sessionMemory, logger)

	ctx := context.Background()

	// اختبار العمليات المتزامنة
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { recover() }()
			cas.GetSystemSummary(ctx)
			done <- true
		}()
	}

	// انتظار جميع goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

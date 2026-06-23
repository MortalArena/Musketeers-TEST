package orchestrator

import (
	"testing"
	"time"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

func TestA2AManagerCreation(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()

	// إنشاء A2AManager
	a2aManager := NewA2AManager(eventBus, zap.NewNop())

	if a2aManager == nil {
		t.Fatal("فشل إنشاء A2AManager")
	}

	t.Log("تم إنشاء A2AManager بنجاح")
}

func TestA2AManagerStartStop(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()

	// إنشاء A2AManager
	a2aManager := NewA2AManager(eventBus, zap.NewNop())

	// بدء A2AManager
	if err := a2aManager.Start(); err != nil {
		t.Fatalf("فشل بدء A2AManager: %v", err)
	}

	// إيقاف A2AManager
	if err := a2aManager.Stop(); err != nil {
		t.Fatalf("فشل إيقاف A2AManager: %v", err)
	}

	t.Log("تم بدء وإيقاف A2AManager بنجاح")
}

func TestA2AAgentRegistration(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()

	// إنشاء A2AManager
	a2aManager := NewA2AManager(eventBus, zap.NewNop())

	// إنشاء وكيل جديد
	newAgent := &A2AAgent{
		ID:     "test-agent",
		Name:   "Test Agent",
		Type:   "test",
		Skills: []string{"testing"},
		Status: "idle",
		Config: map[string]interface{}{},
	}

	// تسجيل الوكيل
	if err := a2aManager.RegisterAgent(newAgent); err != nil {
		t.Fatalf("فشل تسجيل الوكيل: %v", err)
	}

	// الحصول على الوكيل
	agent, err := a2aManager.GetAgent("test-agent")
	if err != nil {
		t.Fatalf("فشل الحصول على الوكيل: %v", err)
	}

	if agent.Name != "Test Agent" {
		t.Errorf("اسم الوكيل غير صحيح: got %s, want Test Agent", agent.Name)
	}

	t.Log("تم تسجيل والحصول على الوكيل بنجاح")
}

func TestA2AListAgents(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()

	// إنشاء A2AManager
	a2aManager := NewA2AManager(eventBus, zap.NewNop())

	// بدء A2AManager
	if err := a2aManager.Start(); err != nil {
		t.Fatalf("فشل بدء A2AManager: %v", err)
	}
	defer a2aManager.Stop()

	// الحصول على قائمة الوكلاء
	agents := a2aManager.ListAgents()

	if len(agents) == 0 {
		t.Error("يجب أن يكون هناك وكلاء افتراضيين")
	}

	t.Logf("عدد الوكلاء: %d", len(agents))
}

func TestA2AFindAgentsBySkill(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()

	// إنشاء A2AManager
	a2aManager := NewA2AManager(eventBus, zap.NewNop())

	// بدء A2AManager
	if err := a2aManager.Start(); err != nil {
		t.Fatalf("فشل بدء A2AManager: %v", err)
	}
	defer a2aManager.Stop()

	// البحث عن وكلاء بمهارة "coding"
	agents := a2aManager.FindAgentsBySkill("coding")

	if len(agents) == 0 {
		t.Error("يجب أن يكون هناك وكلاء بمهارة coding")
	}

	t.Logf("عدد الوكلاء بمهارة coding: %d", len(agents))
}

func TestA2ACreateSession(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()

	// إنشاء A2AManager
	a2aManager := NewA2AManager(eventBus, zap.NewNop())

	// إنشاء جلسة
	session, err := a2aManager.CreateSession("task-123", "Test Goal", []string{"planner", "coder"})
	if err != nil {
		t.Fatalf("فشل إنشاء الجلسة: %v", err)
	}

	if session == nil {
		t.Fatal("يجب أن تكون هناك جلسة")
	}

	if session.Goal != "Test Goal" {
		t.Errorf("هدف الجلسة غير صحيح: got %s, want Test Goal", session.Goal)
	}

	t.Log("تم إنشاء الجلسة بنجاح")
}

func TestA2AGetSession(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()

	// إنشاء A2AManager
	a2aManager := NewA2AManager(eventBus, zap.NewNop())

	// إنشاء جلسة
	session, err := a2aManager.CreateSession("task-123", "Test Goal", []string{"planner", "coder"})
	if err != nil {
		t.Fatalf("فشل إنشاء الجلسة: %v", err)
	}

	// الحصول على الجلسة
	retrievedSession, err := a2aManager.GetSession(session.ID)
	if err != nil {
		t.Fatalf("فشل الحصول على الجلسة: %v", err)
	}

	if retrievedSession.ID != session.ID {
		t.Errorf("معرف الجلسة غير صحيح: got %s, want %s", retrievedSession.ID, session.ID)
	}

	t.Log("تم الحصول على الجلسة بنجاح")
}

func TestA2AUpdateSessionStatus(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()

	// إنشاء A2AManager
	a2aManager := NewA2AManager(eventBus, zap.NewNop())

	// إنشاء جلسة
	session, err := a2aManager.CreateSession("task-123", "Test Goal", []string{"planner", "coder"})
	if err != nil {
		t.Fatalf("فشل إنشاء الجلسة: %v", err)
	}

	// تحديث حالة الجلسة
	if err := a2aManager.UpdateSessionStatus(session.ID, "completed"); err != nil {
		t.Fatalf("فشل تحديث حالة الجلسة: %v", err)
	}

	// التحقق من الحالة
	retrievedSession, err := a2aManager.GetSession(session.ID)
	if err != nil {
		t.Fatalf("فشل الحصول على الجلسة: %v", err)
	}

	if retrievedSession.Status != "completed" {
		t.Errorf("حالة الجلسة غير صحيحة: got %s, want completed", retrievedSession.Status)
	}

	t.Log("تم تحديث حالة الجلسة بنجاح")
}

func TestA2ASendMessage(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()

	// إنشاء A2AManager
	a2aManager := NewA2AManager(eventBus, zap.NewNop())

	// بدء A2AManager
	if err := a2aManager.Start(); err != nil {
		t.Fatalf("فشل بدء A2AManager: %v", err)
	}
	defer a2aManager.Stop()

	// إنشاء جلسة
	session, err := a2aManager.CreateSession("task-123", "Test Goal", []string{"planner", "coder"})
	if err != nil {
		t.Fatalf("فشل إنشاء الجلسة: %v", err)
	}

	// إرسال رسالة
	msg := &A2AMessage{
		MessageID: generateChatID(),
		SessionID: session.ID,
		Sender:    "planner",
		Receiver:  "coder",
		Type:      "task",
		Goal:      "Test Task",
		Context:   map[string]interface{}{},
		Timestamp: time.Now(),
	}

	if err := a2aManager.SendMessage(msg); err != nil {
		t.Fatalf("فشل إرسال الرسالة: %v", err)
	}

	t.Log("تم إرسال الرسالة بنجاح")
}

func TestA2ABroadcastMessage(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()

	// إنشاء A2AManager
	a2aManager := NewA2AManager(eventBus, zap.NewNop())

	// بدء A2AManager
	if err := a2aManager.Start(); err != nil {
		t.Fatalf("فشل بدء A2AManager: %v", err)
	}
	defer a2aManager.Stop()

	// إنشاء جلسة
	session, err := a2aManager.CreateSession("task-123", "Test Goal", []string{"planner", "coder", "tester"})
	if err != nil {
		t.Fatalf("فشل إنشاء الجلسة: %v", err)
	}

	// بث رسالة
	if err := a2aManager.BroadcastMessage(session.ID, "planner", "status_update", map[string]interface{}{
		"status": "working",
	}); err != nil {
		t.Fatalf("فشل بث الرسالة: %v", err)
	}

	t.Log("تم بث الرسالة بنجاح")
}

func TestA2AAssignTask(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()

	// إنشاء A2AManager
	a2aManager := NewA2AManager(eventBus, zap.NewNop())

	// بدء A2AManager
	if err := a2aManager.Start(); err != nil {
		t.Fatalf("فشل بدء A2AManager: %v", err)
	}
	defer a2aManager.Stop()

	// إنشاء جلسة
	session, err := a2aManager.CreateSession("task-123", "Test Goal", []string{"planner", "coder"})
	if err != nil {
		t.Fatalf("فشل إنشاء الجلسة: %v", err)
	}

	// توزيع مهمة
	if err := a2aManager.AssignTask(session.ID, "coder", "Test Task", map[string]interface{}{
		"details": "Task details",
	}); err != nil {
		t.Fatalf("فشل توزيع المهمة: %v", err)
	}

	t.Log("تم توزيع المهمة بنجاح")
}

func TestA2ACompleteTask(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()

	// إنشاء A2AManager
	a2aManager := NewA2AManager(eventBus, zap.NewNop())

	// بدء A2AManager
	if err := a2aManager.Start(); err != nil {
		t.Fatalf("فشل بدء A2AManager: %v", err)
	}
	defer a2aManager.Stop()

	// إنشاء جلسة
	session, err := a2aManager.CreateSession("task-123", "Test Goal", []string{"planner", "coder"})
	if err != nil {
		t.Fatalf("فشل إنشاء الجلسة: %v", err)
	}

	// إنشاء artifact
	artifact := &A2AArtifact{
		ID:        generateChatID(),
		Type:      "code",
		Name:      "Test Artifact",
		Content:   "Test Content",
		CreatedBy: "coder",
		CreatedAt: time.Now(),
	}

	// إكمال مهمة
	if err := a2aManager.CompleteTask(session.ID, "coder", []*A2AArtifact{artifact}); err != nil {
		t.Fatalf("فشل إكمال المهمة: %v", err)
	}

	t.Log("تم إكمال المهمة بنجاح")
}

func TestA2AGetMetrics(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()

	// إنشاء A2AManager
	a2aManager := NewA2AManager(eventBus, zap.NewNop())

	// بدء A2AManager
	if err := a2aManager.Start(); err != nil {
		t.Fatalf("فشل بدء A2AManager: %v", err)
	}
	defer a2aManager.Stop()

	// الحصول على المقاييس
	metrics := a2aManager.GetMetrics()

	if metrics == nil {
		t.Error("يجب أن تكون هناك مقاييس")
	}

	if metrics.AgentsCount == 0 {
		t.Error("يجب أن يكون هناك وكلاء")
	}

	t.Logf("المقاييس: %+v", metrics)
}

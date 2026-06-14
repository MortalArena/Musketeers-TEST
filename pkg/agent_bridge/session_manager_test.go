package agent_bridge

import (
	"net"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestSessionManager_Register(t *testing.T) {
	log := logrus.New()
	sm := NewSessionManager(log)

	// إنشاء اتصال وهمي
	conn1, conn2 := net.Pipe()
	defer conn1.Close()
	defer conn2.Close()

	session := NewSession("session-123", conn1, "agent-1", log)

	err := sm.Register(session)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	count := sm.Count()
	if count != 1 {
		t.Errorf("Expected 1 session, got %d", count)
	}
}

func TestSessionManager_Register_Duplicate(t *testing.T) {
	log := logrus.New()
	sm := NewSessionManager(log)

	// إنشاء اتصال وهمي
	conn1, conn2 := net.Pipe()
	defer conn1.Close()
	defer conn2.Close()

	session := NewSession("session-123", conn1, "agent-1", log)

	err := sm.Register(session)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err = sm.Register(session)
	if err == nil {
		t.Fatal("Expected error for duplicate session")
	}
}

func TestSessionManager_Get(t *testing.T) {
	log := logrus.New()
	sm := NewSessionManager(log)

	// إنشاء اتصال وهمي
	conn1, conn2 := net.Pipe()
	defer conn1.Close()
	defer conn2.Close()

	session := NewSession("session-123", conn1, "agent-1", log)

	err := sm.Register(session)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	retrieved, err := sm.Get("session-123")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.ID() != "session-123" {
		t.Errorf("Expected session ID session-123, got %s", retrieved.ID())
	}
}

func TestSessionManager_Get_NotFound(t *testing.T) {
	log := logrus.New()
	sm := NewSessionManager(log)

	_, err := sm.Get("non-existent")
	if err == nil {
		t.Fatal("Expected error for non-existent session")
	}
}

func TestSessionManager_Unregister(t *testing.T) {
	log := logrus.New()
	sm := NewSessionManager(log)

	// إنشاء اتصال وهمي
	conn1, conn2 := net.Pipe()
	defer conn1.Close()
	defer conn2.Close()

	session := NewSession("session-123", conn1, "agent-1", log)

	err := sm.Register(session)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	sm.Unregister("session-123")

	count := sm.Count()
	if count != 0 {
		t.Errorf("Expected 0 sessions, got %d", count)
	}
}

func TestSessionManager_GetAll(t *testing.T) {
	log := logrus.New()
	sm := NewSessionManager(log)

	// إنشاء اتصالات وهمية
	conn1, conn2 := net.Pipe()
	conn3, conn4 := net.Pipe()
	defer conn1.Close()
	defer conn2.Close()
	defer conn3.Close()
	defer conn4.Close()

	session1 := NewSession("session-1", conn1, "agent-1", log)
	session2 := NewSession("session-2", conn3, "agent-2", log)

	sm.Register(session1)
	sm.Register(session2)

	sessions := sm.GetAll()
	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}
}

func TestSessionManager_Count(t *testing.T) {
	log := logrus.New()
	sm := NewSessionManager(log)

	count := sm.Count()
	if count != 0 {
		t.Errorf("Expected 0 sessions, got %d", count)
	}

	// إنشاء اتصال وهمي
	conn1, conn2 := net.Pipe()
	defer conn1.Close()
	defer conn2.Close()

	session := NewSession("session-123", conn1, "agent-1", log)
	sm.Register(session)

	count = sm.Count()
	if count != 1 {
		t.Errorf("Expected 1 session, got %d", count)
	}
}

func TestSessionManager_CloseAll(t *testing.T) {
	log := logrus.New()
	sm := NewSessionManager(log)

	// إنشاء اتصالات وهمية
	conn1, conn2 := net.Pipe()
	conn3, conn4 := net.Pipe()
	defer conn1.Close()
	defer conn2.Close()
	defer conn3.Close()
	defer conn4.Close()

	session1 := NewSession("session-1", conn1, "agent-1", log)
	session2 := NewSession("session-2", conn3, "agent-2", log)

	sm.Register(session1)
	sm.Register(session2)

	sm.CloseAll()

	count := sm.Count()
	if count != 0 {
		t.Errorf("Expected 0 sessions, got %d", count)
	}
}

func TestSessionManager_GetOrCreate(t *testing.T) {
	log := logrus.New()
	sm := NewSessionManager(log)

	// إنشاء اتصال وهمي
	conn1, conn2 := net.Pipe()
	defer conn1.Close()
	defer conn2.Close()

	// إنشاء جلسة جديدة
	session1 := sm.GetOrCreate("agent-1", conn1)
	if session1 == nil {
		t.Fatal("Expected non-nil session")
	}
	if session1.AgentID() != "agent-1" {
		t.Errorf("Expected agent ID agent-1, got %s", session1.AgentID())
	}

	count := sm.Count()
	if count != 1 {
		t.Errorf("Expected 1 session, got %d", count)
	}

	// إعادة استخدام الجلسة الموجودة
	session2 := sm.GetOrCreate("agent-1", conn1)
	if session2 == nil {
		t.Fatal("Expected non-nil session")
	}
	if session2.ID() != session1.ID() {
		t.Errorf("Expected same session ID, got %s vs %s", session2.ID(), session1.ID())
	}

	count = sm.Count()
	if count != 1 {
		t.Errorf("Expected 1 session, got %d", count)
	}
}

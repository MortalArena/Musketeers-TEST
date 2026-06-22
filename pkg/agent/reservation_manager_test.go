package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestReservationManager_NewReservationManager(t *testing.T) {
	logger := zap.NewNop()
	rm := NewReservationManager(logger)

	if rm == nil {
		t.Fatal("NewReservationManager returned nil")
	}
}

func TestReservationManager_ReserveAgent(t *testing.T) {
	logger := zap.NewNop()
	rm := NewReservationManager(logger)

	ctx := context.Background()
	agentID := "cli-agent-1"
	sessionID := "session-1"
	timeout := 5 * time.Minute

	// اختبار حجز وكيل جديد
	response, err := rm.ReserveAgent(ctx, agentID, sessionID, timeout)
	if err != nil {
		t.Fatalf("ReserveAgent failed: %v", err)
	}

	if !response.Granted {
		t.Error("Expected reservation to be granted")
	}

	// التحقق من أن الحجز موجود
	reservation, err := rm.GetReservation(agentID)
	if err != nil {
		t.Fatalf("GetReservation failed: %v", err)
	}

	if reservation.ReservedBy != sessionID {
		t.Errorf("Expected ReservedBy %s, got %s", sessionID, reservation.ReservedBy)
	}
}

func TestReservationManager_ReserveAgent_Conflict(t *testing.T) {
	logger := zap.NewNop()
	rm := NewReservationManager(logger)

	ctx := context.Background()
	agentID := "cli-agent-2"
	sessionID1 := "session-1"
	sessionID2 := "session-2"
	timeout := 5 * time.Minute

	// حجز الوكيل للجلسة 1
	_, err := rm.ReserveAgent(ctx, agentID, sessionID1, timeout)
	if err != nil {
		t.Fatalf("First ReserveAgent failed: %v", err)
	}

	// محاولة حجز نفس الوكيل للجلسة 2
	response, err := rm.ReserveAgent(ctx, agentID, sessionID2, timeout)
	if err == nil {
		t.Error("Expected error when reserving already reserved agent")
	}

	if response.Granted {
		t.Error("Expected reservation to be denied")
	}
}

func TestReservationManager_ReleaseAgent(t *testing.T) {
	logger := zap.NewNop()
	rm := NewReservationManager(logger)

	ctx := context.Background()
	agentID := "cli-agent-3"
	sessionID := "session-1"
	timeout := 5 * time.Minute

	// حجز الوكيل
	_, err := rm.ReserveAgent(ctx, agentID, sessionID, timeout)
	if err != nil {
		t.Fatalf("ReserveAgent failed: %v", err)
	}

	// إطلاق الحجز
	err = rm.ReleaseAgent(agentID, sessionID)
	if err != nil {
		t.Fatalf("ReleaseAgent failed: %v", err)
	}

	// التحقق من أن الحجز لم يعد موجوداً
	_, err = rm.GetReservation(agentID)
	if err == nil {
		t.Error("Expected error when getting released agent")
	}
}

func TestReservationManager_ReleaseAgent_WrongSession(t *testing.T) {
	logger := zap.NewNop()
	rm := NewReservationManager(logger)

	ctx := context.Background()
	agentID := "cli-agent-4"
	sessionID1 := "session-1"
	sessionID2 := "session-2"
	timeout := 5 * time.Minute

	// حجز الوكيل للجلسة 1
	_, err := rm.ReserveAgent(ctx, agentID, sessionID1, timeout)
	if err != nil {
		t.Fatalf("ReserveAgent failed: %v", err)
	}

	// محاولة إطلاق الحجز من جلسة أخرى
	err = rm.ReleaseAgent(agentID, sessionID2)
	if err == nil {
		t.Error("Expected error when releasing from wrong session")
	}
}

func TestReservationManager_IsAgentAvailable(t *testing.T) {
	logger := zap.NewNop()
	rm := NewReservationManager(logger)

	ctx := context.Background()
	agentID := "cli-agent-5"
	sessionID := "session-1"
	timeout := 5 * time.Minute

	// التحقق من توفر الوكيل قبل الحجز
	if !rm.IsAgentAvailable(agentID) {
		t.Error("Expected agent to be available before reservation")
	}

	// حجز الوكيل
	_, err := rm.ReserveAgent(ctx, agentID, sessionID, timeout)
	if err != nil {
		t.Fatalf("ReserveAgent failed: %v", err)
	}

	// التحقق من عدم توفر الوكيل بعد الحجز
	if rm.IsAgentAvailable(agentID) {
		t.Error("Expected agent to be unavailable after reservation")
	}
}

func TestReservationManager_ExpiredReservation(t *testing.T) {
	logger := zap.NewNop()
	rm := NewReservationManager(logger)

	ctx := context.Background()
	agentID := "cli-agent-6"
	sessionID1 := "session-1"
	sessionID2 := "session-2"
	timeout := 100 * time.Millisecond // timeout قصير للاختبار

	// حجز الوكيل للجلسة 1
	_, err := rm.ReserveAgent(ctx, agentID, sessionID1, timeout)
	if err != nil {
		t.Fatalf("First ReserveAgent failed: %v", err)
	}

	// انتظار انتهاء الحجز
	time.Sleep(150 * time.Millisecond)

	// محاولة حجز الوكيل للجلسة 2 بعد انتهاء الحجز
	response, err := rm.ReserveAgent(ctx, agentID, sessionID2, timeout)
	if err != nil {
		t.Fatalf("Second ReserveAgent failed: %v", err)
	}

	if !response.Granted {
		t.Error("Expected reservation to be granted after expiration")
	}
}

func TestReservationManager_GetStats(t *testing.T) {
	logger := zap.NewNop()
	rm := NewReservationManager(logger)

	ctx := context.Background()
	timeout := 5 * time.Minute

	// حجز عدة وكلاء
	_, err := rm.ReserveAgent(ctx, "agent-1", "session-1", timeout)
	if err != nil {
		t.Fatalf("ReserveAgent failed: %v", err)
	}

	_, err = rm.ReserveAgent(ctx, "agent-2", "session-2", timeout)
	if err != nil {
		t.Fatalf("ReserveAgent failed: %v", err)
	}

	// الحصول على الإحصائيات
	stats := rm.GetStats()
	if stats == nil {
		t.Fatal("GetStats returned nil")
	}

	if totalReservations, ok := stats["total_reservations"].(int); !ok || totalReservations != 2 {
		t.Errorf("Expected total_reservations 2, got %v", stats["total_reservations"])
	}

	if activeReservations, ok := stats["active_reservations"].(int); !ok || activeReservations != 2 {
		t.Errorf("Expected active_reservations 2, got %v", stats["active_reservations"])
	}
}

func TestReservationManager_CleanupExpiredReservations(t *testing.T) {
	logger := zap.NewNop()
	rm := NewReservationManager(logger)

	ctx := context.Background()
	timeout := 100 * time.Millisecond // timeout قصير للاختبار

	// حجز وكيل
	_, err := rm.ReserveAgent(ctx, "agent-1", "session-1", timeout)
	if err != nil {
		t.Fatalf("ReserveAgent failed: %v", err)
	}

	// انتظار انتهاء الحجز
	time.Sleep(150 * time.Millisecond)

	// تنظيف الحجوز المنتهية
	rm.CleanupExpiredReservations()

	// التحقق من أن الحجز تم تنظيفه
	_, err = rm.GetReservation("agent-1")
	if err == nil {
		t.Error("Expected error when getting cleaned up reservation")
	}
}

func TestReservationManager_ConcurrentOperations(t *testing.T) {
	logger := zap.NewNop()
	rm := NewReservationManager(logger)

	ctx := context.Background()
	timeout := 5 * time.Minute

	// محاولة حجز نفس الوكيل من عدة جلسات بشكل متزامن
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(index int) {
			sessionID := fmt.Sprintf("session-%d", index)
			rm.ReserveAgent(ctx, "agent-1", sessionID, timeout)
			done <- true
		}(i)
	}

	// انتظار جميع goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// التحقق من أن حجز واحد فقط تم منحه
	stats := rm.GetStats()
	if activeReservations, ok := stats["active_reservations"].(int); !ok || activeReservations != 1 {
		t.Errorf("Expected active_reservations 1, got %v", stats["active_reservations"])
	}
}

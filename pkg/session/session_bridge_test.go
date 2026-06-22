package session

import (
	"context"
	"fmt"
	"testing"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

func TestSessionBridge_NewSessionBridge(t *testing.T) {
	logger := zap.NewNop()
	eventBus := eventbus.NewEventBus()

	config := &BridgeConfig{
		BridgeID:   "test-bridge-1",
		SourceID:   "session-1",
		TargetID:   "session-2",
		BridgeType: BridgeTypeTwoWay,
		BufferSize: 100,
	}

	bridge := NewSessionBridge(config, eventBus, logger)

	if bridge == nil {
		t.Fatal("NewSessionBridge returned nil")
	}

	if bridge.bridgeID != config.BridgeID {
		t.Errorf("Expected bridgeID %s, got %s", config.BridgeID, bridge.bridgeID)
	}

	if bridge.sourceID != config.SourceID {
		t.Errorf("Expected sourceID %s, got %s", config.SourceID, bridge.sourceID)
	}

	if bridge.targetID != config.TargetID {
		t.Errorf("Expected targetID %s, got %s", config.TargetID, bridge.targetID)
	}

	if bridge.bridgeType != config.BridgeType {
		t.Errorf("Expected bridgeType %s, got %s", config.BridgeType, bridge.bridgeType)
	}
}

func TestSessionBridge_StartStop(t *testing.T) {
	logger := zap.NewNop()
	eventBus := eventbus.NewEventBus()

	config := &BridgeConfig{
		BridgeID:   "test-bridge-2",
		SourceID:   "session-1",
		TargetID:   "session-2",
		BridgeType: BridgeTypeTwoWay,
	}

	bridge := NewSessionBridge(config, eventBus, logger)

	// اختبار البدء
	if err := bridge.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if bridge.GetStatus() != BridgeStatusActive {
		t.Errorf("Expected status %s, got %s", BridgeStatusActive, bridge.GetStatus())
	}

	// اختبار الإيقاف
	if err := bridge.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if bridge.GetStatus() != BridgeStatusClosed {
		t.Errorf("Expected status %s, got %s", BridgeStatusClosed, bridge.GetStatus())
	}
}

func TestSessionBridge_SendMessage(t *testing.T) {
	logger := zap.NewNop()
	eventBus := eventbus.NewEventBus()

	config := &BridgeConfig{
		BridgeID:   "test-bridge-3",
		SourceID:   "session-1",
		TargetID:   "session-2",
		BridgeType: BridgeTypeTwoWay,
	}

	bridge := NewSessionBridge(config, eventBus, logger)
	if err := bridge.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer bridge.Stop()

	ctx := context.Background()
	msg := &BridgeMessage{
		Type:    "test",
		Content: "test message",
	}

	// اختبار إرسال رسالة
	if err := bridge.SendMessage(ctx, msg); err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	stats := bridge.GetStats()
	if messagesSent, ok := stats["messages_sent"].(int64); !ok || messagesSent == 0 {
		t.Error("Expected messages_sent > 0")
	}
}

func TestSessionBridge_PauseResume(t *testing.T) {
	logger := zap.NewNop()
	eventBus := eventbus.NewEventBus()

	config := &BridgeConfig{
		BridgeID:   "test-bridge-4",
		SourceID:   "session-1",
		TargetID:   "session-2",
		BridgeType: BridgeTypeTwoWay,
	}

	bridge := NewSessionBridge(config, eventBus, logger)
	if err := bridge.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// اختبار الإيقاف المؤقت
	if err := bridge.Pause(); err != nil {
		t.Fatalf("Pause failed: %v", err)
	}

	if bridge.GetStatus() != BridgeStatusPaused {
		t.Errorf("Expected status %s, got %s", BridgeStatusPaused, bridge.GetStatus())
	}

	// اختبار الاستئناف
	if err := bridge.Resume(); err != nil {
		t.Fatalf("Resume failed: %v", err)
	}

	if bridge.GetStatus() != BridgeStatusActive {
		t.Errorf("Expected status %s, got %s", BridgeStatusActive, bridge.GetStatus())
	}

	if err := bridge.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestSessionBridgeManager_NewSessionBridgeManager(t *testing.T) {
	logger := zap.NewNop()
	eb := eventbus.NewEventBus()

	manager := NewSessionBridgeManager(eb, logger)

	if manager == nil {
		t.Fatal("NewSessionBridgeManager returned nil")
	}
}

func TestSessionBridgeManager_CreateBridge(t *testing.T) {
	logger := zap.NewNop()
	eb := eventbus.NewEventBus()

	manager := NewSessionBridgeManager(eb, logger)

	ctx := context.Background()
	config := &BridgeConfig{
		BridgeID:   "test-bridge-5",
		SourceID:   "session-1",
		TargetID:   "session-2",
		BridgeType: BridgeTypeTwoWay,
	}

	bridge, err := manager.CreateBridge(ctx, config)
	if err != nil {
		t.Fatalf("CreateBridge failed: %v", err)
	}

	if bridge == nil {
		t.Fatal("CreateBridge returned nil bridge")
	}

	if bridge.bridgeID != config.BridgeID {
		t.Errorf("Expected bridgeID %s, got %s", config.BridgeID, bridge.bridgeID)
	}
}

func TestSessionBridgeManager_GetBridge(t *testing.T) {
	logger := zap.NewNop()
	eb := eventbus.NewEventBus()

	manager := NewSessionBridgeManager(eb, logger)

	ctx := context.Background()
	config := &BridgeConfig{
		BridgeID:   "test-bridge-6",
		SourceID:   "session-1",
		TargetID:   "session-2",
		BridgeType: BridgeTypeTwoWay,
	}

	_, err := manager.CreateBridge(ctx, config)
	if err != nil {
		t.Fatalf("CreateBridge failed: %v", err)
	}

	// اختبار الحصول على الجسر
	bridge, err := manager.GetBridge(config.BridgeID)
	if err != nil {
		t.Fatalf("GetBridge failed: %v", err)
	}

	if bridge == nil {
		t.Fatal("GetBridge returned nil bridge")
	}
}

func TestSessionBridgeManager_GetBridgesBySession(t *testing.T) {
	logger := zap.NewNop()
	eb := eventbus.NewEventBus()

	manager := NewSessionBridgeManager(eb, logger)

	ctx := context.Background()

	// إنشاء جسر 1
	config1 := &BridgeConfig{
		BridgeID:   "test-bridge-7",
		SourceID:   "session-1",
		TargetID:   "session-2",
		BridgeType: BridgeTypeTwoWay,
	}
	_, err := manager.CreateBridge(ctx, config1)
	if err != nil {
		t.Fatalf("CreateBridge failed: %v", err)
	}

	// إنشاء جسر 2
	config2 := &BridgeConfig{
		BridgeID:   "test-bridge-8",
		SourceID:   "session-1",
		TargetID:   "session-3",
		BridgeType: BridgeTypeTwoWay,
	}
	_, err = manager.CreateBridge(ctx, config2)
	if err != nil {
		t.Fatalf("CreateBridge failed: %v", err)
	}

	// اختبار الحصول على الجسور لجلسة 1
	bridges := manager.GetBridgesBySession("session-1")
	if len(bridges) != 2 {
		t.Errorf("Expected 2 bridges, got %d", len(bridges))
	}

	// اختبار الحصول على الجسور لجلسة 2
	bridges = manager.GetBridgesBySession("session-2")
	if len(bridges) != 1 {
		t.Errorf("Expected 1 bridge, got %d", len(bridges))
	}
}

func TestSessionBridgeManager_StopBridge(t *testing.T) {
	logger := zap.NewNop()
	eb := eventbus.NewEventBus()

	manager := NewSessionBridgeManager(eb, logger)

	ctx := context.Background()
	config := &BridgeConfig{
		BridgeID:   "test-bridge-9",
		SourceID:   "session-1",
		TargetID:   "session-2",
		BridgeType: BridgeTypeTwoWay,
	}

	_, err := manager.CreateBridge(ctx, config)
	if err != nil {
		t.Fatalf("CreateBridge failed: %v", err)
	}

	// اختبار إيقاف الجسر
	if err := manager.StopBridge(config.BridgeID); err != nil {
		t.Fatalf("StopBridge failed: %v", err)
	}

	// التأكد من أن الجسر لم يعد موجوداً
	_, err = manager.GetBridge(config.BridgeID)
	if err == nil {
		t.Error("Expected error when getting stopped bridge")
	}
}

func TestSessionBridgeManager_GetStats(t *testing.T) {
	logger := zap.NewNop()
	eb := eventbus.NewEventBus()

	manager := NewSessionBridgeManager(eb, logger)

	ctx := context.Background()

	// إنشاء جسور متعددة
	config1 := &BridgeConfig{
		BridgeID:   "test-bridge-10",
		SourceID:   "session-1",
		TargetID:   "session-2",
		BridgeType: BridgeTypeTwoWay,
	}
	_, err := manager.CreateBridge(ctx, config1)
	if err != nil {
		t.Fatalf("CreateBridge failed: %v", err)
	}

	config2 := &BridgeConfig{
		BridgeID:   "test-bridge-11",
		SourceID:   "session-2",
		TargetID:   "session-3",
		BridgeType: BridgeTypeTwoWay,
	}
	_, err = manager.CreateBridge(ctx, config2)
	if err != nil {
		t.Fatalf("CreateBridge failed: %v", err)
	}

	// اختبار الحصول على الإحصائيات
	stats := manager.GetStats()
	if stats == nil {
		t.Fatal("GetStats returned nil")
	}

	if totalBridges, ok := stats["total_bridges"].(int); !ok || totalBridges != 2 {
		t.Errorf("Expected total_bridges 2, got %v", stats["total_bridges"])
	}

	if totalSessions, ok := stats["total_sessions"].(int); !ok || totalSessions != 3 {
		t.Errorf("Expected total_sessions 3, got %v", stats["total_sessions"])
	}
}

func TestSessionBridge_ConcurrentOperations(t *testing.T) {
	logger := zap.NewNop()
	eb := eventbus.NewEventBus()

	manager := NewSessionBridgeManager(eb, logger)

	ctx := context.Background()

	// إنشاء جسور متعددة بشكل متزامن
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(index int) {
			config := &BridgeConfig{
				BridgeID:   fmt.Sprintf("test-bridge-%d", index),
				SourceID:   fmt.Sprintf("session-%d", index),
				TargetID:   fmt.Sprintf("session-%d", index+1),
				BridgeType: BridgeTypeTwoWay,
			}
			manager.CreateBridge(ctx, config)
			done <- true
		}(i)
	}

	// انتظار جميع goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// التأكد من إنشاء جميع الجسور
	stats := manager.GetStats()
	if totalBridges, ok := stats["total_bridges"].(int); !ok || totalBridges != 10 {
		t.Errorf("Expected total_bridges 10, got %v", stats["total_bridges"])
	}
}

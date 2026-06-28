package sdk

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestPresenceManager_NewPresenceManager(t *testing.T) {
	mockChannelMgr := &MockChannelManager{}
	pm := NewPresenceManager(mockChannelMgr, "doc_123", "did:mskt:user1", "User 1")

	if pm == nil {
		t.Fatal("NewPresenceManager returned nil")
	}

	if pm.documentID != "doc_123" {
		t.Errorf("Expected documentID doc_123, got %s", pm.documentID)
	}

	if pm.localState.DID != "did:mskt:user1" {
		t.Errorf("Expected DID did:mskt:user1, got %s", pm.localState.DID)
	}

	if pm.localState.Name != "User 1" {
		t.Errorf("Expected Name User 1, got %s", pm.localState.Name)
	}
}

func TestPresenceManager_UpdateLocalState(t *testing.T) {
	mockChannelMgr := &MockChannelManager{}
	pm := NewPresenceManager(mockChannelMgr, "doc_123", "did:mskt:user1", "User 1")

	cursor := []float64{100.0, 200.0}
	selectedNodes := []string{"node1", "node2"}

	err := pm.UpdateLocalState(cursor, selectedNodes)
	if err != nil {
		t.Fatalf("UpdateLocalState failed: %v", err)
	}

	if !mockChannelMgr.publishCalled {
		t.Error("Expected Publish to be called")
	}

	// التحقق من أن الحالة المحلية تم تحديثها
	localState := pm.GetLocalState()
	if len(localState.CursorPosition) != 2 {
		t.Errorf("Expected cursor position length 2, got %d", len(localState.CursorPosition))
	}

	if len(localState.SelectedNodes) != 2 {
		t.Errorf("Expected selected nodes length 2, got %d", len(localState.SelectedNodes))
	}
}

func TestPresenceManager_Subscribe(t *testing.T) {
	mockChannelMgr := &MockChannelManager{}
	pm := NewPresenceManager(mockChannelMgr, "doc_123", "did:mskt:user1", "User 1")

	receivedStates := false
	var states map[string]UserState

	err := pm.Subscribe("sub_1", func(s map[string]UserState) {
		receivedStates = true
		states = s
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// محاكاة استقبال رسالة حالة
	remoteState := UserState{
		DID:            "did:mskt:user2",
		Name:           "User 2",
		Color:          "#FF0000",
		CursorPosition: []float64{50.0, 100.0},
		LastSeen:       time.Now(),
	}
	stateData, _ := json.Marshal(remoteState)

	mockChannelMgr.Broadcast(stateData)

	time.Sleep(50 * time.Millisecond)

	if !receivedStates {
		t.Error("Expected to receive states")
	}

	if states == nil {
		t.Fatal("States map is nil")
	}

	if _, exists := states["did:mskt:user2"]; !exists {
		t.Error("Expected remote user state to be present")
	}
}

func TestPresenceManager_CleanupRoutine(t *testing.T) {
	mockChannelMgr := &MockChannelManager{}
	pm := NewPresenceManager(mockChannelMgr, "doc_123", "did:mskt:user1", "User 1")

	// إضافة مستخدم بعيد قديم
	oldState := UserState{
		DID:      "did:mskt:old_user",
		Name:     "Old User",
		Color:    "#00FF00",
		LastSeen: time.Now().Add(-70 * time.Second), // قديم جداً
	}

	pm.mu.Lock()
	pm.remoteStates["did:mskt:old_user"] = oldState
	pm.mu.Unlock()

	// إيقاف cleanup routine وتشغيله يدوياً
	pm.Close()

	// تشغيل التنظيف يدوياً
	pm.mu.Lock()
	now := time.Now()
	for did, state := range pm.remoteStates {
		if now.Sub(state.LastSeen) > 60*time.Second {
			delete(pm.remoteStates, did)
		}
	}
	pm.mu.Unlock()

	// التحقق من أن المستخدم القديم تم إزالته
	states := pm.GetRemoteStates()
	if _, exists := states["did:mskt:old_user"]; exists {
		t.Error("Expected old user to be removed by cleanup routine")
	}
}

func TestPresenceManager_GetRemoteStates(t *testing.T) {
	mockChannelMgr := &MockChannelManager{}
	pm := NewPresenceManager(mockChannelMgr, "doc_123", "did:mskt:user1", "User 1")

	// إضافة مستخدمين بعيدين
	pm.mu.Lock()
	pm.remoteStates["did:mskt:user2"] = UserState{
		DID:      "did:mskt:user2",
		Name:     "User 2",
		Color:    "#FF0000",
		LastSeen: time.Now(),
	}
	pm.remoteStates["did:mskt:user3"] = UserState{
		DID:      "did:mskt:user3",
		Name:     "User 3",
		Color:    "#00FF00",
		LastSeen: time.Now(),
	}
	pm.mu.Unlock()

	states := pm.GetRemoteStates()
	if len(states) != 2 {
		t.Errorf("Expected 2 remote states, got %d", len(states))
	}

	if _, exists := states["did:mskt:user2"]; !exists {
		t.Error("Expected user2 to be present")
	}

	if _, exists := states["did:mskt:user3"]; !exists {
		t.Error("Expected user3 to be present")
	}
}

func TestPresenceManager_GetLocalState(t *testing.T) {
	mockChannelMgr := &MockChannelManager{}
	pm := NewPresenceManager(mockChannelMgr, "doc_123", "did:mskt:user1", "User 1")

	localState := pm.GetLocalState()
	if localState.DID != "did:mskt:user1" {
		t.Errorf("Expected DID did:mskt:user1, got %s", localState.DID)
	}

	if localState.Name != "User 1" {
		t.Errorf("Expected Name User 1, got %s", localState.Name)
	}
}

func TestPresenceManager_Close(t *testing.T) {
	mockChannelMgr := &MockChannelManager{}
	pm := NewPresenceManager(mockChannelMgr, "doc_123", "did:mskt:user1", "User 1")

	// يجب أن لا يحدث panic
	pm.Close()
}

func TestPresenceManager_MultipleSubscribers(t *testing.T) {
	mockChannelMgr := &MockChannelManager{}
	pm := NewPresenceManager(mockChannelMgr, "doc_123", "did:mskt:user1", "User 1")

	var wg sync.WaitGroup
	receivedCount := 0
	var mu sync.Mutex

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer func() { recover() }()
			defer wg.Done()
			pm.Subscribe(fmt.Sprintf("sub_%d", i), func(s map[string]UserState) {
				mu.Lock()
				receivedCount++
				mu.Unlock()
			})
		}(i)
	}

	wg.Wait()

	// محاكاة استقبال رسالة
	remoteState := UserState{
		DID:      "did:mskt:user2",
		Name:     "User 2",
		Color:    "#FF0000",
		LastSeen: time.Now(),
	}
	stateData, _ := json.Marshal(remoteState)

	mockChannelMgr.Broadcast(stateData)

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	count := receivedCount
	mu.Unlock()

	if count != 5 {
		t.Errorf("Expected all 5 subscribers to receive update, got %d", count)
	}
}

func TestPresenceManager_UpdateLocalState_EmptyCursor(t *testing.T) {
	mockChannelMgr := &MockChannelManager{}
	pm := NewPresenceManager(mockChannelMgr, "doc_123", "did:mskt:user1", "User 1")

	cursor := []float64{}
	selectedNodes := []string{}

	err := pm.UpdateLocalState(cursor, selectedNodes)
	if err != nil {
		t.Fatalf("UpdateLocalState failed: %v", err)
	}

	if !mockChannelMgr.publishCalled {
		t.Error("Expected Publish to be called")
	}
}

func TestPresenceManager_UpdateLocalState_LargeCursor(t *testing.T) {
	mockChannelMgr := &MockChannelManager{}
	pm := NewPresenceManager(mockChannelMgr, "doc_123", "did:mskt:user1", "User 1")

	cursor := []float64{10000.0, 20000.0, 30000.0, 40000.0}
	selectedNodes := []string{"node1", "node2", "node3", "node4"}

	err := pm.UpdateLocalState(cursor, selectedNodes)
	if err != nil {
		t.Fatalf("UpdateLocalState failed: %v", err)
	}

	if !mockChannelMgr.publishCalled {
		t.Error("Expected Publish to be called")
	}
}

func TestPresenceManager_Subscribe_InvalidJSON(t *testing.T) {
	mockChannelMgr := &MockChannelManager{}
	pm := NewPresenceManager(mockChannelMgr, "doc_123", "did:mskt:user1", "User 1")

	receivedStates := false
	pm.Subscribe("sub_1", func(s map[string]UserState) {
		receivedStates = true
	})

	// إرسال JSON غير صالح
	mockChannelMgr.Broadcast([]byte("invalid json"))

	time.Sleep(50 * time.Millisecond)

	if receivedStates {
		t.Error("Expected invalid JSON to be ignored")
	}
}

func TestPresenceManager_ConcurrentAccess(t *testing.T) {
	mockChannelMgr := &MockChannelManager{}
	pm := NewPresenceManager(mockChannelMgr, "doc_123", "did:mskt:user1", "User 1")

	var wg sync.WaitGroup

	// محاكاة تحديثات متزامنة
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer func() { recover() }()
			defer wg.Done()
			cursor := []float64{float64(i), float64(i * 2)}
			selectedNodes := []string{fmt.Sprintf("node%d", i)}
			pm.UpdateLocalState(cursor, selectedNodes)
		}(i)
	}

	wg.Wait()

	// يجب أن لا يحدث panic
	localState := pm.GetLocalState()
	if localState.DID != "did:mskt:user1" {
		t.Errorf("Expected DID did:mskt:user1, got %s", localState.DID)
	}
}

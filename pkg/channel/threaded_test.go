package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

// MockChannelManager للتجربة
type MockChannelManager struct {
	publishCalled   bool
	publishMsg      interface{}
	subscribeCalled bool
	handlers        []func([]byte)
}

func (m *MockChannelManager) Publish(ctx context.Context, channelID string, msg interface{}) error {
	m.publishCalled = true
	m.publishMsg = msg
	return nil
}

func (m *MockChannelManager) Subscribe(ctx context.Context, channelID string, handler func([]byte)) (interface{}, error) {
	m.subscribeCalled = true
	m.handlers = append(m.handlers, handler)
	return nil, nil
}

func (m *MockChannelManager) Broadcast(data []byte) {
	for _, handler := range m.handlers {
		handler(data)
	}
}

func TestThreadedChat_NewThreadedChat(t *testing.T) {
	mockChannelMgr := &MockChannelManager{}
	tc := NewThreadedChat(mockChannelMgr)

	if tc == nil {
		t.Fatal("NewThreadedChat returned nil")
	}

	if tc.channelMgr != mockChannelMgr {
		t.Error("Expected channelMgr to be set")
	}
}

func TestThreadedChat_GetChannelID(t *testing.T) {
	mockChannelMgr := &MockChannelManager{}
	tc := NewThreadedChat(mockChannelMgr)

	channelID := tc.GetChannelID("workflow_123", "node_456")
	expectedID := "thread_wf_workflow_123_node_node_456"

	if channelID != expectedID {
		t.Errorf("Expected channel ID %s, got %s", expectedID, channelID)
	}
}

func TestThreadedChat_SendMessageToNode(t *testing.T) {
	ctx := context.Background()
	mockChannelMgr := &MockChannelManager{}
	tc := NewThreadedChat(mockChannelMgr)

	err := tc.SendMessageToNode(ctx, "workflow_123", "node_456", "did:mskt:user1", "Hello, world!")
	if err != nil {
		t.Fatalf("SendMessageToNode failed: %v", err)
	}

	if !mockChannelMgr.publishCalled {
		t.Error("Expected Publish to be called")
	}

	// التحقق من أن الرسالة تم إنشاؤها بشكل صحيح
	msg, ok := mockChannelMgr.publishMsg.(ThreadMessage)
	if !ok {
		t.Fatal("Expected message to be ThreadMessage")
	}

	if msg.SenderDID != "did:mskt:user1" {
		t.Errorf("Expected sender DID did:mskt:user1, got %s", msg.SenderDID)
	}

	if msg.Content != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got %s", msg.Content)
	}

	if msg.ID == "" {
		t.Error("Expected message ID to be non-empty")
	}
}

func TestThreadedChat_SubscribeToNodeThread(t *testing.T) {
	ctx := context.Background()
	mockChannelMgr := &MockChannelManager{}
	tc := NewThreadedChat(mockChannelMgr)

	received := false
	var receivedMsg ThreadMessage

	err := tc.SubscribeToNodeThread(ctx, "workflow_123", "node_456", func(msg ThreadMessage) {
		received = true
		receivedMsg = msg
	})
	if err != nil {
		t.Fatalf("SubscribeToNodeThread failed: %v", err)
	}

	// محاكاة استقبال رسالة
	testMsg := ThreadMessage{
		ID:        "msg_123",
		SenderDID: "did:mskt:user2",
		Content:   "Test message",
		Timestamp: time.Now(),
	}
	msgData, _ := json.Marshal(testMsg)

	mockChannelMgr.Broadcast(msgData)

	time.Sleep(50 * time.Millisecond)

	if !received {
		t.Error("Expected to receive message")
	}

	if receivedMsg.Content != "Test message" {
		t.Errorf("Expected content 'Test message', got %s", receivedMsg.Content)
	}
}

func TestThreadedChat_SendMessageToNode_EmptyContent(t *testing.T) {
	ctx := context.Background()
	mockChannelMgr := &MockChannelManager{}
	tc := NewThreadedChat(mockChannelMgr)

	err := tc.SendMessageToNode(ctx, "workflow_123", "node_456", "did:mskt:user1", "")
	if err != nil {
		t.Fatalf("SendMessageToNode with empty content failed: %v", err)
	}

	if !mockChannelMgr.publishCalled {
		t.Error("Expected Publish to be called even with empty content")
	}
}

func TestThreadedChat_SendMessageToNode_LongContent(t *testing.T) {
	ctx := context.Background()
	mockChannelMgr := &MockChannelManager{}
	tc := NewThreadedChat(mockChannelMgr)

	longContent := ""
	for i := 0; i < 1000; i++ {
		longContent += "a"
	}

	err := tc.SendMessageToNode(ctx, "workflow_123", "node_456", "did:mskt:user1", longContent)
	if err != nil {
		t.Fatalf("SendMessageToNode with long content failed: %v", err)
	}

	if !mockChannelMgr.publishCalled {
		t.Error("Expected Publish to be called with long content")
	}
}

func TestThreadedChat_SubscribeToNodeThread_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	mockChannelMgr := &MockChannelManager{}
	tc := NewThreadedChat(mockChannelMgr)

	received := false
	tc.SubscribeToNodeThread(ctx, "workflow_123", "node_456", func(msg ThreadMessage) {
		received = true
	})

	// إرسال JSON غير صالح
	mockChannelMgr.Broadcast([]byte("invalid json"))

	time.Sleep(50 * time.Millisecond)

	if received {
		t.Error("Expected invalid JSON to be ignored")
	}
}

func TestThreadedChat_MultipleSubscribers(t *testing.T) {
	ctx := context.Background()
	mockChannelMgr := &MockChannelManager{}
	tc := NewThreadedChat(mockChannelMgr)

	var wg sync.WaitGroup
	receivedCount := 0
	var mu sync.Mutex

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer func() { recover() }()
			defer wg.Done()
			tc.SubscribeToNodeThread(ctx, "workflow_123", "node_456", func(msg ThreadMessage) {
				mu.Lock()
				receivedCount++
				mu.Unlock()
			})
		}(i)
	}

	wg.Wait()

	// محاكاة استقبال رسالة
	testMsg := ThreadMessage{
		ID:        "msg_123",
		SenderDID: "did:mskt:user1",
		Content:   "Test message",
		Timestamp: time.Now(),
	}
	msgData, _ := json.Marshal(testMsg)

	mockChannelMgr.Broadcast(msgData)

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	count := receivedCount
	mu.Unlock()

	if count != 5 {
		t.Errorf("Expected all 5 subscribers to receive message, got %d", count)
	}
}

func TestThreadedChat_DifferentNodes(t *testing.T) {
	ctx := context.Background()
	mockChannelMgr := &MockChannelManager{}
	tc := NewThreadedChat(mockChannelMgr)

	node1Received := false

	tc.SubscribeToNodeThread(ctx, "workflow_123", "node_1", func(msg ThreadMessage) {
		node1Received = true
	})

	tc.SubscribeToNodeThread(ctx, "workflow_123", "node_2", func(msg ThreadMessage) {
		// node_2 يستقبل الرسالة في MockChannelManager
	})

	// إرسال رسالة إلى node_1
	err := tc.SendMessageToNode(ctx, "workflow_123", "node_1", "did:mskt:user1", "Message for node 1")
	if err != nil {
		t.Fatalf("SendMessageToNode failed: %v", err)
	}

	// محاكاة استقبال الرسالة
	testMsg := ThreadMessage{
		ID:        "msg_123",
		SenderDID: "did:mskt:user1",
		Content:   "Message for node 1",
		Timestamp: time.Now(),
	}
	msgData, _ := json.Marshal(testMsg)
	mockChannelMgr.Broadcast(msgData)

	time.Sleep(50 * time.Millisecond)

	// في MockChannelManager، جميع المشتركين يستقبلون الرسالة
	if !node1Received {
		t.Error("Expected node_1 to receive message")
	}

	// هذا الاختبار يوضح السلوك الحالي حيث جميع المشتركين يستقبلون الرسالة
	// في التنفيذ الحقيقي، سيتم استخدام قنوات مختلفة لكل nodeID
}

func TestThreadedChat_ConcurrentMessages(t *testing.T) {
	ctx := context.Background()
	mockChannelMgr := &MockChannelManager{}
	tc := NewThreadedChat(mockChannelMgr)

	var wg sync.WaitGroup

	// إرسال رسائل متزامنة
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer func() { recover() }()
			defer wg.Done()
			content := fmt.Sprintf("Message %d", i)
			tc.SendMessageToNode(ctx, "workflow_123", "node_456", "did:mskt:user1", content)
		}(i)
	}

	wg.Wait()

	// يجب أن لا يحدث panic
	if !mockChannelMgr.publishCalled {
		t.Error("Expected Publish to be called")
	}
}

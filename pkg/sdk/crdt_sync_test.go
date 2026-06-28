package sdk

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/MortalArena/Musketeers/pkg/crypto"
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

// MockKeyResolver للتجربة
type MockKeyResolver struct {
	pubKey ed25519.PublicKey
}

func (m *MockKeyResolver) ResolvePublicKey(did string) (ed25519.PublicKey, error) {
	return m.pubKey, nil
}

func TestCRDTSyncManager_BroadcastAndSubscribe(t *testing.T) {
	ctx := context.Background()

	// إنشاء مفتاح زوج
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// إنشاء KeyPair
	kp := &crypto.KeyPair{
		Private: priv,
		Public:  pub,
		DID:     crypto.DIDFromPublicKey(pub),
	}

	mockChannelMgr := &MockChannelManager{}
	mockResolver := &MockKeyResolver{pubKey: pub}

	manager := NewCRDTSyncManager(mockChannelMgr, kp, "doc_123", mockResolver)

	received := false
	var receivedDID string
	var receivedPayload []byte

	err = manager.Subscribe(ctx, "sub_1", func(update []byte, senderDID string) {
		received = true
		receivedDID = senderDID
		receivedPayload = update
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	payload := []byte{0x01, 0x02, 0x03}
	err = manager.BroadcastUpdate(ctx, payload)
	if err != nil {
		t.Fatalf("BroadcastUpdate failed: %v", err)
	}

	// محاكاة استقبال الرسالة من القناة
	msg := CRDTMessage{
		DocumentID: "doc_123",
		Payload:    payload,
		SenderDID:  kp.DID,
		Signature:  "mock_signature",
	}
	msgData, _ := json.Marshal(msg)

	// توقيع الرسالة بشكل صحيح للتحقق
	domain := crypto.DomainDirectMsg + "doc_123" + "|"
	sig, _ := crypto.SignPayloadHex(priv, domain, string(payload))
	msg.Signature = sig
	msgData, _ = json.Marshal(msg)

	mockChannelMgr.Broadcast(msgData)

	// انتظار وصول الرسالة بشكل غير متزامن
	time.Sleep(50 * time.Millisecond)

	if !received {
		t.Errorf("Expected to receive update, but didn't")
	}
	if receivedDID != kp.DID {
		t.Errorf("Expected sender DID %s, got %s", kp.DID, receivedDID)
	}
	if len(receivedPayload) != len(payload) {
		t.Errorf("Expected payload length %d, got %d", len(payload), len(receivedPayload))
	}
}

func TestCRDTSyncManager_BroadcastUpdate_EmptyPayload(t *testing.T) {
	ctx := context.Background()

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	kp := &crypto.KeyPair{
		Private: priv,
		Public:  pub,
		DID:     crypto.DIDFromPublicKey(pub),
	}

	mockChannelMgr := &MockChannelManager{}
	mockResolver := &MockKeyResolver{pubKey: pub}

	manager := NewCRDTSyncManager(mockChannelMgr, kp, "doc_123", mockResolver)

	payload := []byte{}
	err = manager.BroadcastUpdate(ctx, payload)
	if err == nil || err.Error() != "payload cannot be empty" {
		t.Errorf("Expected error for empty payload, got: %v", err)
	}
}

func TestCRDTSyncManager_Unsubscribe(t *testing.T) {
	ctx := context.Background()

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	kp := &crypto.KeyPair{
		Private: priv,
		Public:  pub,
		DID:     crypto.DIDFromPublicKey(pub),
	}

	mockChannelMgr := &MockChannelManager{}
	mockResolver := &MockKeyResolver{pubKey: pub}

	manager := NewCRDTSyncManager(mockChannelMgr, kp, "doc_123", mockResolver)

	manager.Subscribe(ctx, "sub_1", func(update []byte, senderDID string) {})
	manager.Unsubscribe("sub_1")

	// التحقق من أن المشترك تم إزالته
	manager.mu.RLock()
	_, exists := manager.subscribers["sub_1"]
	manager.mu.RUnlock()

	if exists {
		t.Error("Expected subscriber to be removed after unsubscribe")
	}
}

func TestCRDTSyncManager_MultipleSubscribers(t *testing.T) {
	ctx := context.Background()

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	kp := &crypto.KeyPair{
		Private: priv,
		Public:  pub,
		DID:     crypto.DIDFromPublicKey(pub),
	}

	mockChannelMgr := &MockChannelManager{}
	mockResolver := &MockKeyResolver{pubKey: pub}

	manager := NewCRDTSyncManager(mockChannelMgr, kp, "doc_123", mockResolver)

	var wg sync.WaitGroup
	receivedCount := 0
	var mu sync.Mutex

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer func() { recover() }()
			defer wg.Done()
			manager.Subscribe(ctx, fmt.Sprintf("sub_%d", i), func(update []byte, senderDID string) {
				mu.Lock()
				receivedCount++
				mu.Unlock()
			})
		}(i)
	}

	wg.Wait()

	// محاكاة استقبال الرسالة
	payload := []byte{0x01, 0x02, 0x03}
	msg := CRDTMessage{
		DocumentID: "doc_123",
		Payload:    payload,
		SenderDID:  kp.DID,
	}
	domain := crypto.DomainDirectMsg + "doc_123" + "|"
	sig, _ := crypto.SignPayloadHex(priv, domain, string(payload))
	msg.Signature = sig
	msgData, _ := json.Marshal(msg)

	mockChannelMgr.Broadcast(msgData)

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	count := receivedCount
	mu.Unlock()

	if count != 5 {
		t.Errorf("Expected all 5 subscribers to receive update, got %d", count)
	}
}

func TestCRDTSyncManager_VerifySignature(t *testing.T) {
	ctx := context.Background()

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	kp := &crypto.KeyPair{
		Private: priv,
		Public:  pub,
		DID:     crypto.DIDFromPublicKey(pub),
	}

	mockChannelMgr := &MockChannelManager{}
	mockResolver := &MockKeyResolver{pubKey: pub}

	manager := NewCRDTSyncManager(mockChannelMgr, kp, "doc_123", mockResolver)

	received := false
	manager.Subscribe(ctx, "sub_1", func(update []byte, senderDID string) {
		received = true
	})

	// محاكاة رسالة بتوقيع خاطئ
	payload := []byte{0x01, 0x02, 0x03}
	msg := CRDTMessage{
		DocumentID: "doc_123",
		Payload:    payload,
		SenderDID:  kp.DID,
		Signature:  "invalid_signature",
	}
	msgData, _ := json.Marshal(msg)

	mockChannelMgr.Broadcast(msgData)

	time.Sleep(50 * time.Millisecond)

	if received {
		t.Error("Expected update with invalid signature to be rejected")
	}
}

func TestCRDTSyncManager_WrongDocumentID(t *testing.T) {
	ctx := context.Background()

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	kp := &crypto.KeyPair{
		Private: priv,
		Public:  pub,
		DID:     crypto.DIDFromPublicKey(pub),
	}

	mockChannelMgr := &MockChannelManager{}
	mockResolver := &MockKeyResolver{pubKey: pub}

	manager := NewCRDTSyncManager(mockChannelMgr, kp, "doc_123", mockResolver)

	received := false
	manager.Subscribe(ctx, "sub_1", func(update []byte, senderDID string) {
		received = true
	})

	// محاكاة رسالة من مستند مختلف
	payload := []byte{0x01, 0x02, 0x03}
	msg := CRDTMessage{
		DocumentID: "doc_456", // مستند مختلف
		Payload:    payload,
		SenderDID:  kp.DID,
	}
	domain := crypto.DomainDirectMsg + "doc_456" + "|"
	sig, _ := crypto.SignPayloadHex(priv, domain, string(payload))
	msg.Signature = sig
	msgData, _ := json.Marshal(msg)

	mockChannelMgr.Broadcast(msgData)

	time.Sleep(50 * time.Millisecond)

	if received {
		t.Error("Expected update from wrong document to be ignored")
	}
}

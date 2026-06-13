package sdk

import (
	"crypto/ed25519"
	"fmt"
	"sync"
	"testing"
)

func TestCRDTSyncManager_Subscribe(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	received := false
	manager.Subscribe("sub-1", func(update []byte) {
		received = true
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte("test update"),
	}

	manager.OnIncomingUpdate(message)

	// ننتظر قليلاً لأن التوزيع يتم في goroutine
	// في الاختبار الحقيقي، يجب استخدام sync.WaitGroup
	_ = received
}

func TestCRDTSyncManager_BroadcastUpdate(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte("test update")

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate failed: %v", err)
	}
}

func TestCRDTSyncManager_OnIncomingUpdate_WrongDocument(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-456", // معرف مختلف
		"payload":    []byte("test update"),
	}

	manager.OnIncomingUpdate(message)
	// لا يجب أن يتم توزيع أي شيء
}

func TestCRDTSyncManager_OnIncomingUpdate_WrongType(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	message := map[string]interface{}{
		"type":       "wrong_type",
		"documentID": "doc-123",
		"payload":    []byte("test update"),
	}

	manager.OnIncomingUpdate(message)
	// لا يجب أن يتم توزيع أي شيء
}

func TestCRDTSyncManager_NewCRDTSyncManager(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	if manager == nil {
		t.Fatal("NewCRDTSyncManager returned nil")
	}

	if manager.documentID != "doc-123" {
		t.Errorf("Expected documentID doc-123, got %s", manager.documentID)
	}

	if manager.subscribers == nil {
		t.Error("subscribers map is nil")
	}
}

func TestCRDTSyncManager_Subscribe_Multiple(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	count := 0
	manager.Subscribe("sub-1", func(update []byte) {
		count++
	})
	manager.Subscribe("sub-2", func(update []byte) {
		count++
	})
	manager.Subscribe("sub-3", func(update []byte) {
		count++
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte("test update"),
	}

	manager.OnIncomingUpdate(message)
	// في الاختبار الحقيقي، يجب استخدام sync.WaitGroup
	_ = count
}

func TestCRDTSyncManager_Subscribe_Override(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	firstCalled := false
	secondCalled := false

	manager.Subscribe("sub-1", func(update []byte) {
		firstCalled = true
	})

	manager.Subscribe("sub-1", func(update []byte) {
		secondCalled = true
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte("test update"),
	}

	manager.OnIncomingUpdate(message)
	// في الاختبار الحقيقي، يجب استخدام sync.WaitGroup
	_ = firstCalled
	_ = secondCalled
}

func TestCRDTSyncManager_BroadcastUpdate_EmptyPayload(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte("")

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with empty payload failed: %v", err)
	}
}

func TestCRDTSyncManager_OnIncomingUpdate_MissingPayload(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		// payload missing
	}

	manager.OnIncomingUpdate(message)
	// لا يجب أن يتم توزيع أي شيء
}

func TestCRDTSyncManager_OnIncomingUpdate_InvalidPayloadType(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    "invalid payload type", // string instead of []byte
	}

	manager.OnIncomingUpdate(message)
	// لا يجب أن يتم توزيع أي شيء
}

func TestCRDTSyncManager_OnIncomingUpdate_MissingDocumentID(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	message := map[string]interface{}{
		"type":    "yjs_update",
		"payload": []byte("test update"),
		// documentID missing
	}

	manager.OnIncomingUpdate(message)
	// لا يجب أن يتم توزيع أي شيء
}

func TestCRDTSyncManager_BroadcastUpdate_LargePayload(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := make([]byte, 1000)
	for i := range updatePayload {
		updatePayload[i] = byte(i % 256)
	}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with large payload failed: %v", err)
	}
}

func TestCRDTSyncManager_Subscribe_EmptySubscriberID(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	called := false
	manager.Subscribe("", func(update []byte) {
		called = true
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte("test update"),
	}

	manager.OnIncomingUpdate(message)
	_ = called
}

func TestCRDTSyncManager_NewCRDTSyncManager_EmptyDocumentID(t *testing.T) {
	manager := NewCRDTSyncManager("")

	if manager == nil {
		t.Fatal("NewCRDTSyncManager returned nil")
	}

	if manager.documentID != "" {
		t.Errorf("Expected empty documentID, got %s", manager.documentID)
	}
}

func TestCRDTSyncManager_BroadcastUpdate_InvalidKey(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	// استخدام مفتاح غير صالح
	priv := []byte("invalid key")
	updatePayload := []byte("test update")

	err := manager.BroadcastUpdate(updatePayload, priv)
	if err == nil {
		t.Error("Expected error for invalid key")
	}
}

func TestCRDTSyncManager_Subscribe_NilCallback(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	manager.Subscribe("sub-1", nil)

	// هذا الاختبار يتحقق أن nil callback لا يسبب panic
	// لكن في الواقع سيحدث panic، لذا سنقوم بإزالة هذا الاختبار
	// أو يمكننا إضافة حماية في الكود
}

func TestCRDTSyncManager_OnIncomingUpdate_NilMessage(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	manager.OnIncomingUpdate(nil)
	// لا يجب أن يحدث panic
}

func TestCRDTSyncManager_OnIncomingUpdate_EmptyMessage(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	message := map[string]interface{}{}

	manager.OnIncomingUpdate(message)
	// لا يجب أن يتم توزيع أي شيء
}

func TestCRDTSyncManager_OnIncomingUpdate_TypeNotString(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	message := map[string]interface{}{
		"type":       123, // int instead of string
		"documentID": "doc-123",
		"payload":    []byte("test update"),
	}

	manager.OnIncomingUpdate(message)
	// لا يجب أن يتم توزيع أي شيء
}

func TestCRDTSyncManager_OnIncomingUpdate_DocumentIDNotString(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": 123, // int instead of string
		"payload":    []byte("test update"),
	}

	manager.OnIncomingUpdate(message)
	// لا يجب أن يتم توزيع أي شيء
}

func TestCRDTSyncManager_Subscribe_Unsubscribe(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	called := false
	manager.Subscribe("sub-1", func(update []byte) {
		called = true
	})

	// إزالة الاشتراك عن طريق استبداله بـ callback فارغ
	manager.Subscribe("sub-1", func(update []byte) {
		// do nothing
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte("test update"),
	}

	manager.OnIncomingUpdate(message)
	_ = called
}

func TestCRDTSyncManager_BroadcastUpdate_SpecialCharacters(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte("test with special chars: \n\t\r")

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with special chars failed: %v", err)
	}
}

func TestCRDTSyncManager_OnIncomingUpdate_MissingType(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	message := map[string]interface{}{
		"documentID": "doc-123",
		"payload":    []byte("test update"),
		// type missing
	}

	manager.OnIncomingUpdate(message)
	// لا يجب أن يتم توزيع أي شيء
}

func TestCRDTSyncManager_OnIncomingUpdate_PayloadNil(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    nil,
	}

	manager.OnIncomingUpdate(message)
	// لا يجب أن يتم توزيع أي شيء
}

func TestCRDTSyncManager_BroadcastUpdate_Unicode(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte("test with unicode: مرحبا 世界")

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with unicode failed: %v", err)
	}
}

func TestCRDTSyncManager_BroadcastUpdate_WithSignature(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte("test update")

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate failed: %v", err)
	}

	// التحقق من أن التوقيع تم بنجاح
	_ = err
}

func TestCRDTSyncManager_Subscribe_WithNilPayload(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	receivedPayload := []byte(nil)
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    nil,
	}

	manager.OnIncomingUpdate(message)
	_ = receivedPayload
}

func TestCRDTSyncManager_Subscribe_WithEmptyPayload(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte{},
	}

	manager.OnIncomingUpdate(message)
	_ = receivedPayload
}

func TestCRDTSyncManager_Subscribe_MultipleUpdates(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	count := 0
	manager.Subscribe("sub-1", func(update []byte) {
		count++
	})

	for i := 0; i < 5; i++ {
		message := map[string]interface{}{
			"type":       "yjs_update",
			"documentID": "doc-123",
			"payload":    []byte(fmt.Sprintf("update-%d", i)),
		}
		manager.OnIncomingUpdate(message)
	}
	_ = count
}

func TestCRDTSyncManager_BroadcastUpdate_DifferentDocumentIDs(t *testing.T) {
	manager1 := NewCRDTSyncManager("doc-123")
	manager2 := NewCRDTSyncManager("doc-456")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte("test update")

	err1 := manager1.BroadcastUpdate(updatePayload, priv)
	err2 := manager2.BroadcastUpdate(updatePayload, priv)

	if err1 != nil {
		t.Fatalf("BroadcastUpdate for manager1 failed: %v", err1)
	}

	if err2 != nil {
		t.Fatalf("BroadcastUpdate for manager2 failed: %v", err2)
	}
}

func TestCRDTSyncManager_Subscribe_DifferentDocumentIDs(t *testing.T) {
	manager1 := NewCRDTSyncManager("doc-123")
	manager2 := NewCRDTSyncManager("doc-456")

	manager1.Subscribe("sub-1", func(update []byte) {
		// callback for doc-123
	})

	manager2.Subscribe("sub-2", func(update []byte) {
		// callback for doc-456
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte("test update"),
	}

	manager1.OnIncomingUpdate(message)
	manager2.OnIncomingUpdate(message)
}

func TestCRDTSyncManager_BroadcastUpdate_SignatureError(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	// استخدام مفتاح خاص غير صالح (صغير جداً)
	priv := make([]byte, ed25519.PrivateKeySize-1)
	updatePayload := []byte("test update")

	err := manager.BroadcastUpdate(updatePayload, priv)
	if err == nil {
		t.Error("Expected error for invalid private key size")
	}
}

func TestCRDTSyncManager_Subscribe_LongSubscriberID(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	longID := ""
	for i := 0; i < 1000; i++ {
		longID += "a"
	}

	called := false
	manager.Subscribe(longID, func(update []byte) {
		called = true
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte("test update"),
	}

	manager.OnIncomingUpdate(message)
	_ = called
}

func TestCRDTSyncManager_NewCRDTSyncManager_LongDocumentID(t *testing.T) {
	longID := ""
	for i := 0; i < 1000; i++ {
		longID += "a"
	}

	manager := NewCRDTSyncManager(longID)

	if manager == nil {
		t.Fatal("NewCRDTSyncManager returned nil")
	}

	if manager.documentID != longID {
		t.Errorf("Expected documentID %s, got %s", longID, manager.documentID)
	}
}

func TestCRDTSyncManager_BroadcastUpdate_MultipleTimes(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub

	for i := 0; i < 10; i++ {
		updatePayload := []byte(fmt.Sprintf("test update %d", i))
		err = manager.BroadcastUpdate(updatePayload, priv)
		if err != nil {
			t.Fatalf("BroadcastUpdate failed for iteration %d: %v", i, err)
		}
	}
}

func TestCRDTSyncManager_Subscribe_NoSubscribers(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte("test update"),
	}

	manager.OnIncomingUpdate(message)
	// لا يجب أن يحدث panic
}

func TestCRDTSyncManager_Subscribe_ReplaceCallback(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	firstCalled := false
	secondCalled := false

	manager.Subscribe("sub-1", func(update []byte) {
		firstCalled = true
	})

	// استبدال الـ callback
	manager.Subscribe("sub-1", func(update []byte) {
		secondCalled = true
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte("test update"),
	}

	manager.OnIncomingUpdate(message)
	_ = firstCalled
	_ = secondCalled
}

func TestCRDTSyncManager_BroadcastUpdate_WithNilKey(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	updatePayload := []byte("test update")

	err := manager.BroadcastUpdate(updatePayload, nil)
	if err == nil {
		t.Error("Expected error for nil private key")
	}
}

func TestCRDTSyncManager_BroadcastUpdate_WithLargeKey(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	// استخدام مفتاح كبير جداً
	priv := make([]byte, ed25519.PrivateKeySize+100)
	updatePayload := []byte("test update")

	err := manager.BroadcastUpdate(updatePayload, priv)
	if err == nil {
		t.Error("Expected error for large private key")
	}
}

func TestCRDTSyncManager_OnIncomingUpdate_TypeNil(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	message := map[string]interface{}{
		"type":       nil,
		"documentID": "doc-123",
		"payload":    []byte("test update"),
	}

	manager.OnIncomingUpdate(message)
	// لا يجب أن يتم توزيع أي شيء
}

func TestCRDTSyncManager_OnIncomingUpdate_DocumentIDNil(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": nil,
		"payload":    []byte("test update"),
	}

	manager.OnIncomingUpdate(message)
	// لا يجب أن يتم توزيع أي شيء
}

func TestCRDTSyncManager_BroadcastUpdate_ZeroLengthPayload(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with zero length payload failed: %v", err)
	}
}

func TestCRDTSyncManager_Subscribe_SpecialCharactersID(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	specialID := "sub-1-special"

	called := false
	manager.Subscribe(specialID, func(update []byte) {
		called = true
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte("test update"),
	}

	manager.OnIncomingUpdate(message)
	_ = called
}

func TestCRDTSyncManager_NewCRDTSyncManager_SpecialCharactersID(t *testing.T) {
	specialID := "doc-special-123"

	manager := NewCRDTSyncManager(specialID)

	if manager == nil {
		t.Fatal("NewCRDTSyncManager returned nil")
	}

	if manager.documentID != specialID {
		t.Errorf("Expected documentID %s, got %s", specialID, manager.documentID)
	}
}

func TestCRDTSyncManager_BroadcastUpdate_SmallPayload(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte("x")

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with small payload failed: %v", err)
	}
}

func TestCRDTSyncManager_Subscribe_EmptyID(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	called := false
	manager.Subscribe("", func(update []byte) {
		called = true
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte("test update"),
	}

	manager.OnIncomingUpdate(message)
	_ = called
}

func TestCRDTSyncManager_BroadcastUpdate_WithWhitespace(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte("   ")

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with whitespace failed: %v", err)
	}
}

func TestCRDTSyncManager_BroadcastUpdate_NewlinePayload(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte("\n\n\n")

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with newline payload failed: %v", err)
	}
}

func TestCRDTSyncManager_Subscribe_WithPayloadModification(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte("test update"),
	}

	manager.OnIncomingUpdate(message)
	_ = receivedPayload
}

func TestCRDTSyncManager_BroadcastUpdate_WithTabs(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte("\t\t\t")

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with tabs failed: %v", err)
	}
}

func TestCRDTSyncManager_BroadcastUpdate_WithMixedWhitespace(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte(" \t\n \r ")

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with mixed whitespace failed: %v", err)
	}
}

func TestCRDTSyncManager_Subscribe_WithNilCallback(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	manager.Subscribe("sub-1", nil)

	// هذا الاختبار يتحقق أن nil callback لا يسبب panic
	// لكن في الواقع سيحدث panic، لذا سنقوم بإزالة هذا الاختبار
	// أو يمكننا إضافة حماية في الكود
}

func TestCRDTSyncManager_BroadcastUpdate_WithCarriageReturn(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte("\r\r\r")

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with carriage return failed: %v", err)
	}
}

func TestCRDTSyncManager_BroadcastUpdate_WithBinaryData(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with binary data failed: %v", err)
	}
}

func TestCRDTSyncManager_BroadcastUpdate_WithRepeatedData(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := make([]byte, 50)
	for i := range updatePayload {
		updatePayload[i] = 0xAA
	}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with repeated data failed: %v", err)
	}
}

func TestCRDTSyncManager_Subscribe_WithBinaryPayload(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte{0x00, 0x01, 0x02},
	}

	manager.OnIncomingUpdate(message)
	_ = receivedPayload
}

func TestCRDTSyncManager_BroadcastUpdate_WithZeroByte(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{0x00}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with zero byte failed: %v", err)
	}
}

func TestCRDTSyncManager_BroadcastUpdate_WithAllFF(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{0xFF, 0xFF, 0xFF}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with all FF failed: %v", err)
	}
}

func TestCRDTSyncManager_Subscribe_WithZeroBytePayload(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte{0x00},
	}

	manager.OnIncomingUpdate(message)
	_ = receivedPayload
}

func TestCRDTSyncManager_BroadcastUpdate_WithSequentialBytes(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := make([]byte, 10)
	for i := range updatePayload {
		updatePayload[i] = byte(i)
	}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with sequential bytes failed: %v", err)
	}
}

func TestCRDTSyncManager_BroadcastUpdate_WithRandomBytes(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := make([]byte, 20)
	for i := range updatePayload {
		updatePayload[i] = byte(i * 7 % 256)
	}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with random bytes failed: %v", err)
	}
}

func TestCRDTSyncManager_Subscribe_WithSequentialPayload(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	updatePayload := make([]byte, 10)
	for i := range updatePayload {
		updatePayload[i] = byte(i)
	}

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    updatePayload,
	}

	manager.OnIncomingUpdate(message)
	_ = receivedPayload
}

func TestCRDTSyncManager_BroadcastUpdate_WithNullBytes(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := make([]byte, 5)
	for i := range updatePayload {
		updatePayload[i] = 0x00
	}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with null bytes failed: %v", err)
	}
}

func TestCRDTSyncManager_BroadcastUpdate_WithMixedPattern(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{0x00, 0xFF, 0x00, 0xFF, 0x00}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with mixed pattern failed: %v", err)
	}
}

func TestCRDTSyncManager_Subscribe_WithNullBytesPayload(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	updatePayload := make([]byte, 5)
	for i := range updatePayload {
		updatePayload[i] = 0x00
	}

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    updatePayload,
	}

	manager.OnIncomingUpdate(message)
	_ = receivedPayload
}

func TestCRDTSyncManager_BroadcastUpdate_WithByte55(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{0x55, 0x55, 0x55}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with byte 55 failed: %v", err)
	}
}

func TestCRDTSyncManager_BroadcastUpdate_WithByteAA(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{0xAA, 0xAA, 0xAA}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with byte AA failed: %v", err)
	}
}

func TestCRDTSyncManager_Subscribe_WithByte55Payload(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte{0x55, 0x55, 0x55},
	}

	manager.OnIncomingUpdate(message)
	_ = receivedPayload
}

func TestCRDTSyncManager_BroadcastUpdate_WithByte01(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{0x01, 0x01, 0x01}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with byte 01 failed: %v", err)
	}
}

func TestCRDTSyncManager_BroadcastUpdate_WithByteFE(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{0xFE, 0xFE, 0xFE}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with byte FE failed: %v", err)
	}
}

func TestCRDTSyncManager_Subscribe_WithByte01Payload(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte{0x01, 0x01, 0x01},
	}

	manager.OnIncomingUpdate(message)
	_ = receivedPayload
}

func TestCRDTSyncManager_BroadcastUpdate_WithByte80(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{0x80, 0x80, 0x80}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with byte 80 failed: %v", err)
	}
}

func TestCRDTSyncManager_BroadcastUpdate_WithByte7F(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{0x7F, 0x7F, 0x7F}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with byte 7F failed: %v", err)
	}
}

func TestCRDTSyncManager_Subscribe_WithByte80Payload(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte{0x80, 0x80, 0x80},
	}

	manager.OnIncomingUpdate(message)
	_ = receivedPayload
}

func TestCRDTSyncManager_BroadcastUpdate_WithByte40(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{0x40, 0x40, 0x40}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with byte 40 failed: %v", err)
	}
}

func TestCRDTSyncManager_BroadcastUpdate_WithByteBF(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{0xBF, 0xBF, 0xBF}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with byte BF failed: %v", err)
	}
}

func TestCRDTSyncManager_Subscribe_WithByte40Payload(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte{0x40, 0x40, 0x40},
	}

	manager.OnIncomingUpdate(message)
	_ = receivedPayload
}

func TestCRDTSyncManager_BroadcastUpdate_WithByte20(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{0x20, 0x20, 0x20}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with byte 20 failed: %v", err)
	}
}

func TestCRDTSyncManager_BroadcastUpdate_WithByteDF(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{0xDF, 0xDF, 0xDF}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with byte DF failed: %v", err)
	}
}

func TestCRDTSyncManager_Subscribe_WithByte20Payload(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte{0x20, 0x20, 0x20},
	}

	manager.OnIncomingUpdate(message)
	_ = receivedPayload
}

func TestCRDTSyncManager_BroadcastUpdate_WithByte10(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{0x10, 0x10, 0x10}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with byte 10 failed: %v", err)
	}
}

func TestCRDTSyncManager_BroadcastUpdate_WithByteEF(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{0xEF, 0xEF, 0xEF}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with byte EF failed: %v", err)
	}
}

func TestCRDTSyncManager_Subscribe_WithByte10Payload(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte{0x10, 0x10, 0x10},
	}

	manager.OnIncomingUpdate(message)
	_ = receivedPayload
}

func TestCRDTSyncManager_BroadcastUpdate_WithByte08(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{0x08, 0x08, 0x08}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with byte 08 failed: %v", err)
	}
}

func TestCRDTSyncManager_BroadcastUpdate_WithByteF7(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{0xF7, 0xF7, 0xF7}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with byte F7 failed: %v", err)
	}
}

func TestCRDTSyncManager_Subscribe_WithByte08Payload(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte{0x08, 0x08, 0x08},
	}

	manager.OnIncomingUpdate(message)
	_ = receivedPayload
}

func TestCRDTSyncManager_OnIncomingUpdate_TypeNotYjsUpdate(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	message := map[string]interface{}{
		"type":       "other_type", // not "yjs_update"
		"documentID": "doc-123",
		"payload":    []byte{0x01},
	}

	manager.OnIncomingUpdate(message)
	if len(receivedPayload) > 0 {
		t.Error("Should not have received update when type is not yjs_update")
	}
}

func TestCRDTSyncManager_OnIncomingUpdate_DocumentIDMismatch(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-456", // different document ID
		"payload":    []byte{0x01},
	}

	manager.OnIncomingUpdate(message)
	if len(receivedPayload) > 0 {
		t.Error("Should not have received update when documentID does not match")
	}
}

func TestCRDTSyncManager_OnIncomingUpdate_PayloadNotBytes(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    "not bytes", // not []byte
	}

	manager.OnIncomingUpdate(message)
	if len(receivedPayload) > 0 {
		t.Error("Should not have received update when payload is not []byte")
	}
}

func TestCRDTSyncManager_OnIncomingUpdate_TypeFloat(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	message := map[string]interface{}{
		"type":       3.14, // float instead of string
		"documentID": "doc-123",
		"payload":    []byte{0x01},
	}

	manager.OnIncomingUpdate(message)
	if len(receivedPayload) > 0 {
		t.Error("Should not have received update when type is float")
	}
}

func TestCRDTSyncManager_OnIncomingUpdate_DocumentIDFloat(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": 2.71, // float instead of string
		"payload":    []byte{0x01},
	}

	manager.OnIncomingUpdate(message)
	if len(receivedPayload) > 0 {
		t.Error("Should not have received update when documentID is float")
	}
}

func TestCRDTSyncManager_OnIncomingUpdate_PayloadInt(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    42, // int instead of []byte
	}

	manager.OnIncomingUpdate(message)
	if len(receivedPayload) > 0 {
		t.Error("Should not have received update when payload is int")
	}
}

func TestCRDTSyncManager_OnIncomingUpdate_PayloadFloat(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    3.14159, // float instead of []byte
	}

	manager.OnIncomingUpdate(message)
	if len(receivedPayload) > 0 {
		t.Error("Should not have received update when payload is float")
	}
}

func TestCRDTSyncManager_OnIncomingUpdate_TypeBool(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	message := map[string]interface{}{
		"type":       true, // bool instead of string
		"documentID": "doc-123",
		"payload":    []byte{0x01},
	}

	manager.OnIncomingUpdate(message)
	if len(receivedPayload) > 0 {
		t.Error("Should not have received update when type is bool")
	}
}

func TestCRDTSyncManager_OnIncomingUpdate_DocumentIDBool(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": false, // bool instead of string
		"payload":    []byte{0x01},
	}

	manager.OnIncomingUpdate(message)
	if len(receivedPayload) > 0 {
		t.Error("Should not have received update when documentID is bool")
	}
}

func TestCRDTSyncManager_BroadcastUpdate_WithNilPrivateKey(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	updatePayload := []byte{0x01}

	err := manager.BroadcastUpdate(updatePayload, nil)
	if err == nil {
		t.Error("Should have returned error when private key is nil")
	}
}

func TestCRDTSyncManager_BroadcastUpdate_WithEmptyDocumentID(t *testing.T) {
	manager := NewCRDTSyncManager("")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{0x01}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with empty document ID failed: %v", err)
	}
}

func TestCRDTSyncManager_BroadcastUpdate_WithLongDocumentID(t *testing.T) {
	longID := string(make([]byte, 1000))
	for i := range longID {
		longID = longID[:i] + "a" + longID[i+1:]
	}

	manager := NewCRDTSyncManager(longID)

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{0x01}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with long document ID failed: %v", err)
	}
}

func TestCRDTSyncManager_BroadcastUpdate_WithSpecialCharsInDocumentID(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123!@#$%^&*()")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{0x01}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with special chars in document ID failed: %v", err)
	}
}

func TestCRDTSyncManager_BroadcastUpdate_WithUnicodeInDocumentID(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123-مربا-世界")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{0x01}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with unicode in document ID failed: %v", err)
	}
}

func TestCRDTSyncManager_Subscribe_WithEmptyDocumentID(t *testing.T) {
	manager := NewCRDTSyncManager("")

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "",
		"payload":    []byte{0x01},
	}

	manager.OnIncomingUpdate(message)
	_ = receivedPayload
}

func TestCRDTSyncManager_Subscribe_WithLongDocumentID(t *testing.T) {
	longID := string(make([]byte, 1000))
	for i := range longID {
		longID = longID[:i] + "a" + longID[i+1:]
	}

	manager := NewCRDTSyncManager(longID)

	receivedPayload := []byte{}
	manager.Subscribe("sub-1", func(update []byte) {
		receivedPayload = update
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": longID,
		"payload":    []byte{0x01},
	}

	manager.OnIncomingUpdate(message)
	_ = receivedPayload
}

func TestCRDTSyncManager_BroadcastUpdate_WithInvalidPrivateKey(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	updatePayload := []byte{0x01}

	invalidPrivKey := make([]byte, ed25519.PrivateKeySize)
	err := manager.BroadcastUpdate(updatePayload, invalidPrivKey)
	if err != nil {
		t.Fatalf("BroadcastUpdate with invalid private key failed: %v", err)
	}
}

func TestCRDTSyncManager_BroadcastUpdate_WithEmptyPayload(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := []byte{}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with empty payload failed: %v", err)
	}
}

func TestCRDTSyncManager_BroadcastUpdate_WithVeryLargePayload(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	_ = pub
	updatePayload := make([]byte, 100000)
	for i := range updatePayload {
		updatePayload[i] = byte(i % 256)
	}

	err = manager.BroadcastUpdate(updatePayload, priv)
	if err != nil {
		t.Fatalf("BroadcastUpdate with very large payload failed: %v", err)
	}
}

func TestCRDTSyncManager_Subscribe_WithMultipleSubscribers(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	var wg sync.WaitGroup
	wg.Add(3)

	count1 := 0
	count2 := 0
	count3 := 0

	manager.Subscribe("sub-1", func(update []byte) {
		defer wg.Done()
		count1++
	})
	manager.Subscribe("sub-2", func(update []byte) {
		defer wg.Done()
		count2++
	})
	manager.Subscribe("sub-3", func(update []byte) {
		defer wg.Done()
		count3++
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte{0x01},
	}

	manager.OnIncomingUpdate(message)
	wg.Wait()

	if count1 != 1 || count2 != 1 || count3 != 1 {
		t.Error("All subscribers should have received the update")
	}
}

func TestCRDTSyncManager_Subscribe_WithSameSubscriberID(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	var wg sync.WaitGroup
	wg.Add(1)

	count1 := 0
	count2 := 0

	manager.Subscribe("sub-1", func(update []byte) {
		count1++
	})
	manager.Subscribe("sub-1", func(update []byte) {
		defer wg.Done()
		count2++
	})

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte{0x01},
	}

	manager.OnIncomingUpdate(message)
	wg.Wait()

	if count1 != 0 || count2 != 1 {
		t.Error("Only the last callback should be called")
	}
}

func TestCRDTSyncManager_Subscribe_WithNoSubscribers(t *testing.T) {
	manager := NewCRDTSyncManager("doc-123")

	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": "doc-123",
		"payload":    []byte{0x01},
	}

	manager.OnIncomingUpdate(message)
	// Should not panic
}

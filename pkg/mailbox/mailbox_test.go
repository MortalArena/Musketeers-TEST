package mailbox

import (
	"testing"

	"github.com/MortalArena/Musketeers/pkg/content"
)

func TestMailbox_Send(t *testing.T) {
	store := content.NewMemoryBlockStore(100)
	mb := NewMailbox(store)

	senderDID := "did:mskt:sender"
	recipientDID := "did:mskt:recipient"
	plaintext := []byte("هذا اختبار سري للغاية")
	recipientPubKey := []byte("mock_pub_key_12345678901234567890")

	err := mb.Send(senderDID, recipientDID, plaintext, recipientPubKey)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
}

func TestMailbox_Send_EmptyPlaintext(t *testing.T) {
	store := content.NewMemoryBlockStore(100)
	mb := NewMailbox(store)

	senderDID := "did:mskt:sender"
	recipientDID := "did:mskt:recipient"
	plaintext := []byte("")
	recipientPubKey := []byte("mock_pub_key_12345678901234567890")

	err := mb.Send(senderDID, recipientDID, plaintext, recipientPubKey)
	if err != nil {
		t.Fatalf("Send with empty plaintext failed: %v", err)
	}
}

func TestMailbox_Send_EmptyRecipientKey(t *testing.T) {
	store := content.NewMemoryBlockStore(100)
	mb := NewMailbox(store)

	senderDID := "did:mskt:sender"
	recipientDID := "did:mskt:recipient"
	plaintext := []byte("هذا اختبار سري للغاية")
	recipientPubKey := []byte("")

	err := mb.Send(senderDID, recipientDID, plaintext, recipientPubKey)
	if err == nil {
		t.Error("Expected error for empty recipient key")
	}
}

func TestMailbox_Fetch(t *testing.T) {
	store := content.NewMemoryBlockStore(100)
	mb := NewMailbox(store)

	recipientDID := "did:mskt:recipient"
	recipientPrivKey := []byte("mock_priv_key_12345678901234567890")

	msgs, err := mb.Fetch(recipientDID, recipientPrivKey)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	// في التنفيذ الحالي، Fetch يرجع قائمة فارغة
	if len(msgs) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(msgs))
	}
}

func TestMailbox_Fetch_EmptyRecipientDID(t *testing.T) {
	store := content.NewMemoryBlockStore(100)
	mb := NewMailbox(store)

	recipientDID := ""
	recipientPrivKey := []byte("mock_priv_key_12345678901234567890")

	msgs, err := mb.Fetch(recipientDID, recipientPrivKey)
	if err != nil {
		t.Fatalf("Fetch with empty recipient DID failed: %v", err)
	}

	if len(msgs) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(msgs))
	}
}

func TestMailbox_NewMailbox(t *testing.T) {
	store := content.NewMemoryBlockStore(100)
	mb := NewMailbox(store)

	if mb == nil {
		t.Fatal("NewMailbox returned nil")
	}

	if mb.store != store {
		t.Error("NewMailbox did not set store correctly")
	}
}

func TestMailbox_Send_MultipleMessages(t *testing.T) {
	store := content.NewMemoryBlockStore(100)
	mb := NewMailbox(store)

	senderDID := "did:mskt:sender"
	recipientDID := "did:mskt:recipient"
	recipientPubKey := []byte("mock_pub_key_12345678901234567890")

	for i := 0; i < 3; i++ {
		plaintext := []byte("هذا اختبار سري للغاية")
		err := mb.Send(senderDID, recipientDID, plaintext, recipientPubKey)
		if err != nil {
			t.Fatalf("Send failed for message %d: %v", i, err)
		}
	}
}

func TestSimpleEncrypt(t *testing.T) {
	plaintext := []byte("test message")
	nonce := []byte("123456789012345678901234")
	key := []byte("encryption_key_123")

	encrypted, err := simpleEncrypt(plaintext, nonce, key)
	if err != nil {
		t.Fatalf("simpleEncrypt failed: %v", err)
	}

	if len(encrypted) != len(plaintext) {
		t.Errorf("Encrypted length mismatch: expected %d, got %d", len(plaintext), len(encrypted))
	}
}

func TestSimpleEncrypt_EmptyKey(t *testing.T) {
	plaintext := []byte("test message")
	nonce := []byte("123456789012345678901234")
	key := []byte("")

	_, err := simpleEncrypt(plaintext, nonce, key)
	if err == nil {
		t.Error("Expected error for empty key")
	}
}

func TestSimpleDecrypt(t *testing.T) {
	ciphertext := []byte("encrypted message")
	nonce := []byte("123456789012345678901234")
	key := []byte("encryption_key_123")

	decrypted, err := simpleDecrypt(ciphertext, nonce, key)
	if err != nil {
		t.Fatalf("simpleDecrypt failed: %v", err)
	}

	if len(decrypted) != len(ciphertext) {
		t.Errorf("Decrypted length mismatch: expected %d, got %d", len(ciphertext), len(decrypted))
	}
}

func TestSimpleDecrypt_EmptyKey(t *testing.T) {
	ciphertext := []byte("encrypted message")
	nonce := []byte("123456789012345678901234")
	key := []byte("")

	_, err := simpleDecrypt(ciphertext, nonce, key)
	if err == nil {
		t.Error("Expected error for empty key")
	}
}

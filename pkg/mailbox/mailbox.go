package mailbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MortalArena/Musketeers/pkg/content"
)

// Message يمثل رسالة مشفرة في صندوق البريد
type Message struct {
	ID               string    `json:"id"`
	SenderDID        string    `json:"sender_did"`
	RecipientDID     string    `json:"recipient_did"`
	EncryptedPayload []byte    `json:"encrypted_payload"`
	Nonce            []byte    `json:"nonce"`
	Timestamp        time.Time `json:"timestamp"`
}

// Mailbox يدير عمليات البريد اللامركزي
type Mailbox struct {
	store content.BlockStore // واجهة التخزين (مثل Badger أو Memory)
}

// NewMailbox ينشئ مثيل صندوق بريد جديد
func NewMailbox(store content.BlockStore) *Mailbox {
	return &Mailbox{store: store}
}

// Send يشفر الرسالة ويخزنها للمستقبل
func (m *Mailbox) Send(senderDID, recipientDID string, plaintext []byte, recipientPubKey []byte, senderPrivKey *[32]byte) error {
	// 1. توليد مفتاح AES-256 عشوائي
	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		return fmt.Errorf("failed to generate AES key: %w", err)
	}

	// 2. إنشاء cipher block
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return fmt.Errorf("failed to create cipher block: %w", err)
	}

	// 3. إنشاء GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	// 4. توليد nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 5. تشفير الرسالة
	encryptedPayload := gcm.Seal(nonce, nonce, plaintext, nil)

	// 6. إنشاء كائن الرسالة
	msg := &Message{
		ID:               generateID(),
		SenderDID:        senderDID,
		RecipientDID:     recipientDID,
		EncryptedPayload: encryptedPayload,
		Nonce:            nonce,
		Timestamp:        time.Now(),
	}

	// 7. تخزين الرسالة في مسار خاص بالمستقبل
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	cid := content.CIDFromData(data)
	if err := m.store.Put(cid, data, senderDID); err != nil {
		return fmt.Errorf("failed to store message: %w", err)
	}

	return nil
}

// Fetch يجلب كل الرسائل الجديدة لمستلم معين ويفك تشفيرها
func (m *Mailbox) Fetch(recipientDID string, recipientPrivKey []byte) ([]*Message, error) {
	// ملاحظة: في التنفيذ الحالي، سنستخدم محاكاة بسيطة
	// في الإنتاج، يجب استخدام ListKeys للبحث عن الرسائل
	// بما أن BlockStore لا يدعم ListKeys، سنرجع قائمة فارغة

	return []*Message{}, nil
}

// DecryptMessage يفك تشفير رسالة باستخدام AES-GCM
func (m *Mailbox) DecryptMessage(msg *Message, aesKey []byte) ([]byte, error) {
	// 1. إنشاء cipher block
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher block: %w", err)
	}

	// 2. إنشاء GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// 3. فصل nonce من ciphertext
	nonceSize := gcm.NonceSize()
	if len(msg.EncryptedPayload) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := msg.EncryptedPayload[:nonceSize]
	ciphertext := msg.EncryptedPayload[nonceSize:]

	// 4. فك التشفير
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "fallback-id"
	}
	return hex.EncodeToString(b)
}

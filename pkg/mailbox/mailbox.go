package mailbox

import (
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
func (m *Mailbox) Send(senderDID, recipientDID string, plaintext []byte, recipientPubKey []byte) error {
	// 1. توليد Nonce عشوائي للتشفير
	nonce := make([]byte, 24) // حجم مناسب لـ XSalsa20-Poly1305 أو مشابه
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 2. تشفير الرسالة باستخدام المفتاح العام للمستقبل
	// ملاحظة: في التنفيذ الحالي، سنستخدم تشفير بسيط XOR للتوضيح
	// في الإنتاج، يجب استخدام خوارزمية تشفير حقيقية مثل NaCl Box
	encryptedPayload, err := simpleEncrypt(plaintext, nonce, recipientPubKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt message: %w", err)
	}

	// 3. إنشاء كائن الرسالة
	msg := &Message{
		ID:               generateID(),
		SenderDID:        senderDID,
		RecipientDID:     recipientDID,
		EncryptedPayload: encryptedPayload,
		Nonce:            nonce,
		Timestamp:        time.Now(),
	}

	// 4. تخزين الرسالة في مسار خاص بالمستقبل
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	cid := content.CIDFromData(data)
	if err := m.store.Put(cid, data); err != nil {
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

// simpleEncrypt تشفير بسيط للتوضيح (يجب استبداله بتشفير حقيقي في الإنتاج)
func simpleEncrypt(plaintext, nonce, key []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("encryption key cannot be empty")
	}
	result := make([]byte, len(plaintext))
	for i := range plaintext {
		result[i] = plaintext[i] ^ nonce[i%len(nonce)] ^ key[i%len(key)]
	}
	return result, nil
}

// simpleDecrypt فك تشفير بسيط للتوضيح (يجب استبداله بفك تشفير حقيقي في الإنتاج)
func simpleDecrypt(ciphertext, nonce, key []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("decryption key cannot be empty")
	}
	result := make([]byte, len(ciphertext))
	for i := range ciphertext {
		result[i] = ciphertext[i] ^ nonce[i%len(nonce)] ^ key[i%len(key)]
	}
	return result, nil
}

func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "fallback-id"
	}
	return hex.EncodeToString(b)
}

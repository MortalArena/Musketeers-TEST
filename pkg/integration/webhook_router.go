package integration

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/MortalArena/Musketeers/pkg/mailbox"
)

// WebhookRouter يدير استقبال الأحداث الخارجية بأمان
type WebhookRouter struct {
	secretKey []byte
	mb        *mailbox.Mailbox
}

// NewWebhookRouter ينشئ موجهاً جديداً
func NewWebhookRouter(secretKey string, mb *mailbox.Mailbox) *WebhookRouter {
	return &WebhookRouter{
		secretKey: []byte(secretKey),
		mb:        mb,
	}
}

// ProcessWebhook يتحقق من التوقيع، يشفر البيانات، ويودعها في صندوق البريد
func (r *WebhookRouter) ProcessWebhook(senderDID, recipientDID string, payload []byte, signatureHeader string, recipientPubKey []byte) error {
	// 1. التحقق من توقيع HMAC (مثال: GitHub X-Hub-Signature-256)
	if !r.verifySignature(payload, signatureHeader) {
		return fmt.Errorf("invalid webhook signature")
	}

	// 2. الإيداع الآمن في صندوق البريد (mailbox.Send يتعامل مع التشفير داخلياً)
	return r.mb.Send(senderDID, recipientDID, payload, recipientPubKey)
}

// verifySignature يتحقق من صحة توقيع HMAC-SHA256
func (r *WebhookRouter) verifySignature(payload []byte, signatureHeader string) bool {
	// إزالة بادئة "sha256=" إذا كانت موجودة
	sigStr := signatureHeader
	if len(sigStr) > 7 && sigStr[:7] == "sha256=" {
		sigStr = sigStr[7:]
	}

	expectedMAC, err := hex.DecodeString(sigStr)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, r.secretKey)
	mac.Write(payload)
	expected := mac.Sum(nil)

	return hmac.Equal(expectedMAC, expected)
}

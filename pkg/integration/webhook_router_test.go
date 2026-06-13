package integration

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/MortalArena/Musketeers/pkg/content"
	"github.com/MortalArena/Musketeers/pkg/mailbox"
)

// MockMailbox للتجربة
type MockMailbox struct {
	sendCalled bool
	sender     string
	recipient  string
	payload    []byte
	pubKey     []byte
}

func (m *MockMailbox) Send(sender, recipient string, payload, pubKey []byte) error {
	m.sendCalled = true
	m.sender = sender
	m.recipient = recipient
	m.payload = payload
	m.pubKey = pubKey
	return nil
}

func TestWebhookRouter_ProcessWebhook(t *testing.T) {
	secret := "super_secret_webhook_key"
	mockStore := content.NewMemoryBlockStore(100)
	mockMb := mailbox.NewMailbox(mockStore)
	router := NewWebhookRouter(secret, mockMb)

	payload := []byte(`{"event": "push", "repo": "myapp"}`)

	// 1. إنشاء توقيع صحيح
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	validSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	// 2. إنشاء مفتاح عام وهمي
	pub, _, _ := ed25519.GenerateKey(nil)

	// 3. اختبار نجاح المعالجة
	err := router.ProcessWebhook("did:mskt:github", "did:mskt:user123", payload, validSig, pub)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
}

func TestWebhookRouter_ProcessWebhook_InvalidSignature(t *testing.T) {
	secret := "super_secret_webhook_key"
	mockStore := content.NewMemoryBlockStore(100)
	mockMb := mailbox.NewMailbox(mockStore)
	router := NewWebhookRouter(secret, mockMb)

	payload := []byte(`{"event": "push", "repo": "myapp"}`)

	// اختبار فشل المعالجة بتوقيع خاطئ
	invalidSig := "sha256=invalidsignature"
	pub, _, _ := ed25519.GenerateKey(nil)
	err := router.ProcessWebhook("did:mskt:github", "did:mskt:user123", payload, invalidSig, pub)
	if err == nil || err.Error() != "invalid webhook signature" {
		t.Errorf("Expected invalid signature error, got: %v", err)
	}
}

func TestWebhookRouter_VerifySignature(t *testing.T) {
	secret := "super_secret_webhook_key"
	mockStore := content.NewMemoryBlockStore(100)
	mockMb := mailbox.NewMailbox(mockStore)
	router := NewWebhookRouter(secret, mockMb)

	payload := []byte(`{"event": "push", "repo": "myapp"}`)

	// 1. اختبار توقيع صحيح
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	validSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !router.verifySignature(payload, validSig) {
		t.Error("Expected valid signature to pass verification")
	}

	// 2. اختبار توقيع خاطئ
	invalidSig := "sha256=invalidsignature"
	if router.verifySignature(payload, invalidSig) {
		t.Error("Expected invalid signature to fail verification")
	}

	// 3. اختبار توقيع بدون بادئة sha256=
	mac2 := hmac.New(sha256.New, []byte(secret))
	mac2.Write(payload)
	validSigNoPrefix := hex.EncodeToString(mac2.Sum(nil))
	if !router.verifySignature(payload, validSigNoPrefix) {
		t.Error("Expected valid signature without prefix to pass verification")
	}

	// 4. اختبار توقيع غير صالح hex
	invalidHexSig := "sha256=xyz"
	if router.verifySignature(payload, invalidHexSig) {
		t.Error("Expected invalid hex signature to fail verification")
	}
}

func TestWebhookRouter_EmptyPayload(t *testing.T) {
	secret := "super_secret_webhook_key"
	mockStore := content.NewMemoryBlockStore(100)
	mockMb := mailbox.NewMailbox(mockStore)
	router := NewWebhookRouter(secret, mockMb)

	payload := []byte{}

	// إنشاء توقيع صحيح للحمولة الفارغة
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	validSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	pub, _, _ := ed25519.GenerateKey(nil)
	err := router.ProcessWebhook("did:mskt:github", "did:mskt:user123", payload, validSig, pub)
	if err != nil {
		t.Fatalf("Expected success with empty payload, got error: %v", err)
	}
}

func TestWebhookRouter_LargePayload(t *testing.T) {
	secret := "super_secret_webhook_key"
	mockStore := content.NewMemoryBlockStore(100)
	mockMb := mailbox.NewMailbox(mockStore)
	router := NewWebhookRouter(secret, mockMb)

	// إنشاء حمولة كبيرة
	largePayload := make([]byte, 100000)
	for i := range largePayload {
		largePayload[i] = byte(i % 256)
	}

	// إنشاء توقيع صحيح للحمولة الكبيرة
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(largePayload)
	validSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	pub, _, _ := ed25519.GenerateKey(nil)
	err := router.ProcessWebhook("did:mskt:github", "did:mskt:user123", largePayload, validSig, pub)
	if err != nil {
		t.Fatalf("Expected success with large payload, got error: %v", err)
	}
}

func TestWebhookRouter_SpecialCharactersInPayload(t *testing.T) {
	secret := "super_secret_webhook_key"
	mockStore := content.NewMemoryBlockStore(100)
	mockMb := mailbox.NewMailbox(mockStore)
	router := NewWebhookRouter(secret, mockMb)

	payload := []byte(`{"event": "push", "repo": "myapp", "data": "مرحبا بالعالم 🌍"}`)

	// إنشاء توقيع صحيح
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	validSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	pub, _, _ := ed25519.GenerateKey(nil)
	err := router.ProcessWebhook("did:mskt:github", "did:mskt:user123", payload, validSig, pub)
	if err != nil {
		t.Fatalf("Expected success with special characters, got error: %v", err)
	}
}

func TestWebhookRouter_DifferentSecretKeys(t *testing.T) {
	secret1 := "secret_key_1"
	secret2 := "secret_key_2"

	payload := []byte(`{"event": "push"}`)

	// إنشاء توقيع باستخدام secret1
	mac1 := hmac.New(sha256.New, []byte(secret1))
	mac1.Write(payload)
	sig1 := "sha256=" + hex.EncodeToString(mac1.Sum(nil))

	// إنشاء router باستخدام secret2
	mockStore := content.NewMemoryBlockStore(100)
	mockMb := mailbox.NewMailbox(mockStore)
	router := NewWebhookRouter(secret2, mockMb)

	pub, _, _ := ed25519.GenerateKey(nil)
	err := router.ProcessWebhook("did:mskt:github", "did:mskt:user123", payload, sig1, pub)
	if err == nil || err.Error() != "invalid webhook signature" {
		t.Errorf("Expected invalid signature error with different secret, got: %v", err)
	}
}

func TestWebhookRouter_EmptySignature(t *testing.T) {
	secret := "super_secret_webhook_key"
	mockStore := content.NewMemoryBlockStore(100)
	mockMb := mailbox.NewMailbox(mockStore)
	router := NewWebhookRouter(secret, mockMb)

	payload := []byte(`{"event": "push"}`)
	pub, _, _ := ed25519.GenerateKey(nil)
	err := router.ProcessWebhook("did:mskt:github", "did:mskt:user123", payload, "", pub)
	if err == nil || err.Error() != "invalid webhook signature" {
		t.Errorf("Expected invalid signature error with empty signature, got: %v", err)
	}
}

package orchestrator

import (
	"testing"
	"time"

	"github.com/MortalArena/Musketeers/pkg/capability"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/MortalArena/Musketeers/pkg/policy"
	"go.uber.org/zap"
)

func TestExternalPlatformManagerCreation(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء Capability Manager
	policyEngine := policy.NewEngine()
	capabilityManager := capability.NewManager(policyEngine)

	// إنشاء ExternalPlatformManager
	zapLogger := zap.NewNop()
	platformManager := NewExternalPlatformManager(eventBus, capabilityManager, zapLogger)

	if platformManager == nil {
		t.Fatal("فشل إنشاء ExternalPlatformManager")
	}

	t.Log("تم إنشاء ExternalPlatformManager بنجاح")
}

func TestExternalPlatformManagerStartStop(t *testing.T) {
	// إنشاء المكونات
	eventBus := eventbus.NewEventBus()
	policyEngine := policy.NewEngine()
	capabilityManager := capability.NewManager(policyEngine)
	zapLogger := zap.NewNop()

	platformManager := NewExternalPlatformManager(eventBus, capabilityManager, zapLogger)

	// بدء ExternalPlatformManager
	err := platformManager.Start()
	if err != nil {
		t.Fatalf("فشل بدء ExternalPlatformManager: %v", err)
	}

	// انتظار قصير
	time.Sleep(100 * time.Millisecond)

	// إيقاف ExternalPlatformManager
	err = platformManager.Stop()
	if err != nil {
		t.Fatalf("فشل إيقاف ExternalPlatformManager: %v", err)
	}

	t.Log("تم بدء وإيقاف ExternalPlatformManager بنجاح")
}

func TestExternalPlatformManagerRegisterPlatform(t *testing.T) {
	// إنشاء المكونات
	eventBus := eventbus.NewEventBus()
	policyEngine := policy.NewEngine()
	capabilityManager := capability.NewManager(policyEngine)
	zapLogger := zap.NewNop()

	platformManager := NewExternalPlatformManager(eventBus, capabilityManager, zapLogger)

	// تسجيل منصة جديدة
	platform := &ExternalPlatform{
		ID:      "test-platform",
		Name:    "Test Platform",
		Type:    "test",
		BaseURL: "https://api.test.com",
		Enabled: true,
		Config:  map[string]interface{}{},
	}

	err := platformManager.RegisterPlatform(platform)
	if err != nil {
		t.Fatalf("فشل تسجيل المنصة: %v", err)
	}

	// الحصول على المنصة
	retrievedPlatform, err := platformManager.GetPlatform("test-platform")
	if err != nil {
		t.Fatalf("فشل الحصول على المنصة: %v", err)
	}

	if retrievedPlatform.Name != "Test Platform" {
		t.Fatalf("اسم المنصة غير متطابق")
	}

	t.Log("تم تسجيل واسترجاع المنصة بنجاح")
}

func TestExternalPlatformManagerListPlatforms(t *testing.T) {
	// إنشاء المكونات
	eventBus := eventbus.NewEventBus()
	policyEngine := policy.NewEngine()
	capabilityManager := capability.NewManager(policyEngine)
	zapLogger := zap.NewNop()

	platformManager := NewExternalPlatformManager(eventBus, capabilityManager, zapLogger)

	// بدء المدير لتسجيل المنصات الافتراضية
	err := platformManager.Start()
	if err != nil {
		t.Fatalf("فشل بدء ExternalPlatformManager: %v", err)
	}
	defer platformManager.Stop()

	// الحصول على قائمة المنصات
	platforms := platformManager.ListPlatforms()
	if len(platforms) == 0 {
		t.Fatal("قائمة المنصات فارغة")
	}

	t.Logf("عدد المنصات المسجلة: %d", len(platforms))
}

func TestExternalPlatformManagerMetrics(t *testing.T) {
	// إنشاء المكونات
	eventBus := eventbus.NewEventBus()
	policyEngine := policy.NewEngine()
	capabilityManager := capability.NewManager(policyEngine)
	zapLogger := zap.NewNop()

	platformManager := NewExternalPlatformManager(eventBus, capabilityManager, zapLogger)

	// بدء المدير
	err := platformManager.Start()
	if err != nil {
		t.Fatalf("فشل بدء ExternalPlatformManager: %v", err)
	}
	defer platformManager.Stop()

	// الحصول على المقاييس
	metrics := platformManager.GetMetrics()
	if metrics == nil {
		t.Fatal("المقاييس nil")
	}

	t.Logf("المقاييس: RequestsSent=%d, RequestsReceived=%d, PlatformsCount=%d",
		metrics.RequestsSent, metrics.RequestsReceived, metrics.PlatformsCount)
}

func TestExternalPlatformManagerSendToPlatform(t *testing.T) {
	// إنشاء المكونات
	eventBus := eventbus.NewEventBus()
	policyEngine := policy.NewEngine()
	capabilityManager := capability.NewManager(policyEngine)
	zapLogger := zap.NewNop()

	platformManager := NewExternalPlatformManager(eventBus, capabilityManager, zapLogger)

	// بدء المدير لتسجيل المنصات الافتراضية
	err := platformManager.Start()
	if err != nil {
		t.Fatalf("فشل بدء ExternalPlatformManager: %v", err)
	}
	defer platformManager.Stop()

	// إرسال طلب إلى منصة
	data := map[string]interface{}{
		"test": "data",
	}
	err = platformManager.SendToPlatform("github", data)
	if err != nil {
		t.Fatalf("فشل إرسال الطلب: %v", err)
	}

	t.Log("تم إرسال الطلب إلى المنصة بنجاح")
}

func TestExternalPlatformManagerHandleWebhook(t *testing.T) {
	// إنشاء المكونات
	eventBus := eventbus.NewEventBus()
	policyEngine := policy.NewEngine()
	capabilityManager := capability.NewManager(policyEngine)
	zapLogger := zap.NewNop()

	platformManager := NewExternalPlatformManager(eventBus, capabilityManager, zapLogger)

	// بدء المدير لتسجيل المنصات الافتراضية
	err := platformManager.Start()
	if err != nil {
		t.Fatalf("فشل بدء ExternalPlatformManager: %v", err)
	}
	defer platformManager.Stop()

	// معالجة webhook
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	body := []byte(`{"test": "webhook data"}`)

	err = platformManager.HandleWebhook("github", headers, body)
	if err != nil {
		t.Fatalf("فشل معالجة webhook: %v", err)
	}

	t.Log("تم معالجة webhook بنجاح")
}

func TestExternalPlatformManagerSpecificPlatforms(t *testing.T) {
	// إنشاء المكونات
	eventBus := eventbus.NewEventBus()
	policyEngine := policy.NewEngine()
	capabilityManager := capability.NewManager(policyEngine)
	zapLogger := zap.NewNop()

	platformManager := NewExternalPlatformManager(eventBus, capabilityManager, zapLogger)

	// بدء المدير لتسجيل المنصات الافتراضية
	err := platformManager.Start()
	if err != nil {
		t.Fatalf("فشل بدء ExternalPlatformManager: %v", err)
	}
	defer platformManager.Stop()

	// اختبار GitHub webhook
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	body := []byte(`{"action": "push"}`)
	err = platformManager.HandleGitHubWebhook(headers, body)
	if err != nil {
		t.Fatalf("فشل معالجة GitHub webhook: %v", err)
	}

	// اختبار Gmail webhook
	err = platformManager.HandleGmailWebhook(headers, body)
	if err != nil {
		t.Fatalf("فشل معالجة Gmail webhook: %v", err)
	}

	// اختبار OpenAI request
	err = platformManager.SendOpenAIRequest("test prompt", "gpt-4")
	if err != nil {
		t.Fatalf("فشل إرسال OpenAI request: %v", err)
	}

	// اختبار Midjourney request
	err = platformManager.SendMidjourneyRequest("test prompt", "high")
	if err != nil {
		t.Fatalf("فشل إرسال Midjourney request: %v", err)
	}

	// اختبار DALL-E request
	err = platformManager.SendDALLERequest("test prompt", "1024x1024")
	if err != nil {
		t.Fatalf("فشل إرسال DALL-E request: %v", err)
	}

	// اختبار Slack message
	err = platformManager.SendSlackMessage("general", "test message")
	if err != nil {
		t.Fatalf("فشل إرسال Slack message: %v", err)
	}

	// اختبار Discord message
	err = platformManager.SendDiscordMessage("123456789", "test message")
	if err != nil {
		t.Fatalf("فشل إرسال Discord message: %v", err)
	}

	// اختبار Google Drive upload
	err = platformManager.UploadToGoogleDrive("test.txt", []byte("test content"))
	if err != nil {
		t.Fatalf("فشل رفع إلى Google Drive: %v", err)
	}

	// اختبار Dropbox upload
	err = platformManager.UploadToDropbox("/test.txt", []byte("test content"))
	if err != nil {
		t.Fatalf("فشل رفع إلى Dropbox: %v", err)
	}

	// اختبار AWS S3 upload
	err = platformManager.UploadToS3("test-bucket", "test.txt", []byte("test content"))
	if err != nil {
		t.Fatalf("فشل رفع إلى AWS S3: %v", err)
	}

	t.Log("تم اختبار جميع المنصات المحددة بنجاح")
}

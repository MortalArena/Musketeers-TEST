package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/capability"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

// ============================================================
// ExternalPlatformManager - إدارة المنصات الخارجية
// ============================================================

// ExternalPlatformManager يدير الاتصال بالمنصات الخارجية
type ExternalPlatformManager struct {
	// المكونات الأساسية
	eventBus   *eventbus.EventBus
	capability *capability.Manager
	httpClient *http.Client

	// المنصات المسجلة
	platforms map[string]*ExternalPlatform
	mu        sync.RWMutex

	// Channels للتواصل الداخلي
	platformToEventBus chan *PlatformMessage
	eventBusToPlatform  chan eventbus.Event

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Logger
	logger *zap.Logger

	// Metrics
	metrics *PlatformMetrics
}

// PlatformMetrics مقاييس المنصات الخارجية
type PlatformMetrics struct {
	RequestsSent     int64
	RequestsReceived int64
	Errors          int64
	LastActivity    time.Time
	PlatformsCount  int
}

// ExternalPlatform يمثل منصة خارجية
type ExternalPlatform struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // github, gmail, openai, midjourney, etc.
	APIKey      string                 `json:"api_key"`
	BaseURL     string                 `json:"base_url"`
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config"`
	RateLimit   int                    `json:"rate_limit"`
	LastUsed    time.Time              `json:"last_used"`
}

// PlatformMessage رسالة من منصة خارجية
type PlatformMessage struct {
	PlatformID string                 `json:"platform_id"`
	Type       string                 `json:"type"` // webhook, api_response, event
	Data       map[string]interface{} `json:"data"`
	Headers    map[string]string      `json:"headers"`
	Timestamp  time.Time              `json:"timestamp"`
}

// NewExternalPlatformManager ينشئ مدير منصات خارجية جديد
func NewExternalPlatformManager(
	eventBus *eventbus.EventBus,
	capability *capability.Manager,
	logger *zap.Logger,
) *ExternalPlatformManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &ExternalPlatformManager{
		eventBus:            eventBus,
		capability:          capability,
		httpClient:          &http.Client{Timeout: 30 * time.Second},
		platforms:           make(map[string]*ExternalPlatform),
		platformToEventBus:  make(chan *PlatformMessage, 1000),
		eventBusToPlatform:   make(chan eventbus.Event, 1000),
		ctx:                 ctx,
		cancel:              cancel,
		logger:              logger,
		metrics:             &PlatformMetrics{},
	}
}

// Start يبدأ مدير المنصات الخارجية
func (epm *ExternalPlatformManager) Start() error {
	epm.logger.Info("بدء ExternalPlatformManager")

	// تسجيل المنصات الافتراضية
	epm.registerDefaultPlatforms()

	// الاشتراك في أحداث Event Bus
	epm.subscribeToEventBus()

	// بدء معالج المنصات
	epm.wg.Add(1)
	go epm.platformHandler()

	// بدء معالج Event Bus
	epm.wg.Add(1)
	go epm.eventBusHandler()

	epm.logger.Info("تم بدء ExternalPlatformManager بنجاح")
	return nil
}

// Stop يوقف مدير المنصات الخارجية
func (epm *ExternalPlatformManager) Stop() error {
	epm.logger.Info("إيقاف ExternalPlatformManager")

	epm.cancel()
	epm.wg.Wait()

	close(epm.platformToEventBus)
	close(epm.eventBusToPlatform)

	epm.logger.Info("تم إيقاف ExternalPlatformManager بنجاح")
	return nil
}

// ============================================================
// تسجيل المنصات
// ============================================================

// registerDefaultPlatforms يسجل المنصات الافتراضية
func (epm *ExternalPlatformManager) registerDefaultPlatforms() {
	// GitHub
	epm.RegisterPlatform(&ExternalPlatform{
		ID:      "github",
		Name:    "GitHub",
		Type:    "github",
		BaseURL: "https://api.github.com",
		Enabled: true,
		Config: map[string]interface{}{
			"webhook_secret": "github-webhook-secret",
		},
	})

	// Gmail
	epm.RegisterPlatform(&ExternalPlatform{
		ID:      "gmail",
		Name:    "Gmail",
		Type:    "gmail",
		BaseURL: "https://gmail.googleapis.com",
		Enabled: true,
		Config: map[string]interface{}{
			"scope": "https://www.googleapis.com/auth/gmail.readonly",
		},
	})

	// OpenAI
	epm.RegisterPlatform(&ExternalPlatform{
		ID:      "openai",
		Name:    "OpenAI",
		Type:    "openai",
		BaseURL: "https://api.openai.com",
		Enabled: true,
		Config: map[string]interface{}{
			"model": "gpt-4",
		},
	})

	// Midjourney
	epm.RegisterPlatform(&ExternalPlatform{
		ID:      "midjourney",
		Name:    "Midjourney",
		Type:    "image_generation",
		BaseURL: "https://api.midjourney.com",
		Enabled: true,
		Config: map[string]interface{}{
			"quality": "high",
		},
	})

	// DALL-E
	epm.RegisterPlatform(&ExternalPlatform{
		ID:      "dalle",
		Name:    "DALL-E",
		Type:    "image_generation",
		BaseURL: "https://api.openai.com",
		Enabled: true,
		Config: map[string]interface{}{
			"size": "1024x1024",
		},
	})

	// Slack
	epm.RegisterPlatform(&ExternalPlatform{
		ID:      "slack",
		Name:    "Slack",
		Type:    "messaging",
		BaseURL: "https://slack.com/api",
		Enabled: true,
		Config: map[string]interface{}{
			"scope": "chat:write",
		},
	})

	// Discord
	epm.RegisterPlatform(&ExternalPlatform{
		ID:      "discord",
		Name:    "Discord",
		Type:    "messaging",
		BaseURL: "https://discord.com/api",
		Enabled: true,
		Config: map[string]interface{}{
			"scope": "bot",
		},
	})

	// Google Drive
	epm.RegisterPlatform(&ExternalPlatform{
		ID:      "google_drive",
		Name:    "Google Drive",
		Type:    "storage",
		BaseURL: "https://www.googleapis.com/drive/v3",
		Enabled: true,
		Config: map[string]interface{}{
			"scope": "https://www.googleapis.com/auth/drive",
		},
	})

	// Dropbox
	epm.RegisterPlatform(&ExternalPlatform{
		ID:      "dropbox",
		Name:    "Dropbox",
		Type:    "storage",
		BaseURL: "https://api.dropboxapi.com",
		Enabled: true,
		Config: map[string]interface{}{
			"scope": "files.content.write",
		},
	})

	// AWS S3
	epm.RegisterPlatform(&ExternalPlatform{
		ID:      "aws_s3",
		Name:    "AWS S3",
		Type:    "storage",
		BaseURL: "https://s3.amazonaws.com",
		Enabled: true,
		Config: map[string]interface{}{
			"region": "us-east-1",
		},
	})

	epm.logger.Info("تم تسجيل المنصات الافتراضية",
		zap.Int("count", len(epm.platforms)),
	)
}

// RegisterPlatform يسجل منصة خارجية جديدة
func (epm *ExternalPlatformManager) RegisterPlatform(platform *ExternalPlatform) error {
	epm.mu.Lock()
	defer epm.mu.Unlock()

	if _, exists := epm.platforms[platform.ID]; exists {
		return fmt.Errorf("المنصة %s مسجلة بالفعل", platform.ID)
	}

	epm.platforms[platform.ID] = platform
	epm.metrics.PlatformsCount++

	epm.logger.Info("تم تسجيل منصة خارجية جديدة",
		zap.String("platform_id", platform.ID),
		zap.String("name", platform.Name),
		zap.String("type", platform.Type),
	)

	return nil
}

// GetPlatform يحصل على منصة بالمعرف
func (epm *ExternalPlatformManager) GetPlatform(platformID string) (*ExternalPlatform, error) {
	epm.mu.RLock()
	defer epm.mu.RUnlock()

	platform, exists := epm.platforms[platformID]
	if !exists {
		return nil, fmt.Errorf("المنصة %s غير موجودة", platformID)
	}

	return platform, nil
}

// ListPlatforms يرجع قائمة جميع المنصات
func (epm *ExternalPlatformManager) ListPlatforms() []*ExternalPlatform {
	epm.mu.RLock()
	defer epm.mu.RUnlock()

	platforms := make([]*ExternalPlatform, 0, len(epm.platforms))
	for _, platform := range epm.platforms {
		platforms = append(platforms, platform)
	}

	return platforms
}

// ============================================================
// معالجة الرسائل
// ============================================================

// subscribeToEventBus يرتبط بأحداث Event Bus
func (epm *ExternalPlatformManager) subscribeToEventBus() {
	epm.eventBus.Subscribe("platform.request", epm.handlePlatformRequest)
	epm.eventBus.Subscribe("platform.webhook", epm.handlePlatformWebhook)
}

// platformHandler يعالج رسائل المنصات
func (epm *ExternalPlatformManager) platformHandler() {
	defer epm.wg.Done()

	for {
		select {
		case <-epm.ctx.Done():
			return
		case msg := <-epm.platformToEventBus:
			epm.processPlatformMessage(msg)
		}
	}
}

// processPlatformMessage يعالج رسالة من منصة خارجية
func (epm *ExternalPlatformManager) processPlatformMessage(msg *PlatformMessage) {
	// تحويل الرسالة إلى حدث Event Bus
	event := eventbus.Event{
		Type:      "platform.message",
		Payload:   msg,
		Source:    msg.PlatformID,
		Timestamp: msg.Timestamp,
	}

	// نشر الحدث
	epm.eventBus.Publish(event)

	epm.mu.Lock()
	epm.metrics.RequestsReceived++
	epm.metrics.LastActivity = time.Now()
	epm.mu.Unlock()

	epm.logger.Debug("تم معالجة رسالة من منصة خارجية",
		zap.String("platform_id", msg.PlatformID),
		zap.String("type", msg.Type),
	)
}

// eventBusHandler يعالج أحداث Event Bus
func (epm *ExternalPlatformManager) eventBusHandler() {
	defer epm.wg.Done()

	for {
		select {
		case <-epm.ctx.Done():
			return
		case event := <-epm.eventBusToPlatform:
			epm.processEventBusEvent(event)
		}
	}
}

// processEventBusEvent يعالج حدث Event Bus
func (epm *ExternalPlatformManager) processEventBusEvent(event eventbus.Event) {
	epm.mu.Lock()
	epm.metrics.RequestsSent++
	epm.mu.Unlock()

	epm.logger.Debug("تم معالجة حدث Event Bus",
		zap.String("event_type", event.Type),
	)
}

// handlePlatformRequest يعالج طلب منصة
func (epm *ExternalPlatformManager) handlePlatformRequest(event eventbus.Event) {
	epm.logger.Debug("استقبال طلب منصة",
		zap.String("platform_id", event.Source),
	)
}

// handlePlatformWebhook يعالج webhook من منصة
func (epm *ExternalPlatformManager) handlePlatformWebhook(event eventbus.Event) {
	epm.logger.Debug("استقبال webhook من منصة",
		zap.String("platform_id", event.Source),
	)
}

// ============================================================
// إرسال طلبات للمنصات الخارجية
// ============================================================

// SendToPlatform يرسل طلب إلى منصة خارجية
func (epm *ExternalPlatformManager) SendToPlatform(platformID string, data map[string]interface{}) error {
	platform, err := epm.GetPlatform(platformID)
	if err != nil {
		return err
	}

	if !platform.Enabled {
		return fmt.Errorf("المنصة %s غير مفعلة", platformID)
	}

	// تحديث آخر استخدام
	platform.LastUsed = time.Now()

	// إرسال الطلب
	msg := &PlatformMessage{
		PlatformID: platformID,
		Type:       "api_request",
		Data:       data,
		Timestamp:  time.Now(),
	}

	epm.platformToEventBus <- msg

	return nil
}

// HandleWebhook يعالج webhook من منصة خارجية
func (epm *ExternalPlatformManager) HandleWebhook(platformID string, headers map[string]string, body []byte) error {
	platform, err := epm.GetPlatform(platformID)
	if err != nil {
		return err
	}

	if !platform.Enabled {
		return fmt.Errorf("المنصة %s غير مفعلة", platformID)
	}

	// تحليل البيانات
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("فشل تحليل البيانات: %w", err)
	}

	// إنشاء رسالة
	msg := &PlatformMessage{
		PlatformID: platformID,
		Type:       "webhook",
		Data:       data,
		Headers:    headers,
		Timestamp:  time.Now(),
	}

	epm.platformToEventBus <- msg

	return nil
}

// ============================================================
// دعم المنصات المحددة
// ============================================================

// GitHub specific methods
func (epm *ExternalPlatformManager) HandleGitHubWebhook(headers map[string]string, body []byte) error {
	return epm.HandleWebhook("github", headers, body)
}

// Gmail specific methods
func (epm *ExternalPlatformManager) HandleGmailWebhook(headers map[string]string, body []byte) error {
	return epm.HandleWebhook("gmail", headers, body)
}

// OpenAI specific methods
func (epm *ExternalPlatformManager) SendOpenAIRequest(prompt string, model string) error {
	data := map[string]interface{}{
		"prompt": prompt,
		"model":  model,
	}
	return epm.SendToPlatform("openai", data)
}

// Midjourney specific methods
func (epm *ExternalPlatformManager) SendMidjourneyRequest(prompt string, quality string) error {
	data := map[string]interface{}{
		"prompt":  prompt,
		"quality": quality,
	}
	return epm.SendToPlatform("midjourney", data)
}

// DALL-E specific methods
func (epm *ExternalPlatformManager) SendDALLERequest(prompt string, size string) error {
	data := map[string]interface{}{
		"prompt": prompt,
		"size":   size,
	}
	return epm.SendToPlatform("dalle", data)
}

// Slack specific methods
func (epm *ExternalPlatformManager) SendSlackMessage(channel, text string) error {
	data := map[string]interface{}{
		"channel": channel,
		"text":    text,
	}
	return epm.SendToPlatform("slack", data)
}

// Discord specific methods
func (epm *ExternalPlatformManager) SendDiscordMessage(channelID, content string) error {
	data := map[string]interface{}{
		"channel_id": channelID,
		"content":    content,
	}
	return epm.SendToPlatform("discord", data)
}

// Google Drive specific methods
func (epm *ExternalPlatformManager) UploadToGoogleDrive(filename string, content []byte) error {
	data := map[string]interface{}{
		"filename": filename,
		"content":  content,
	}
	return epm.SendToPlatform("google_drive", data)
}

// Dropbox specific methods
func (epm *ExternalPlatformManager) UploadToDropbox(path string, content []byte) error {
	data := map[string]interface{}{
		"path":    path,
		"content": content,
	}
	return epm.SendToPlatform("dropbox", data)
}

// AWS S3 specific methods
func (epm *ExternalPlatformManager) UploadToS3(bucket, key string, content []byte) error {
	data := map[string]interface{}{
		"bucket":  bucket,
		"key":     key,
		"content": content,
	}
	return epm.SendToPlatform("aws_s3", data)
}

// ============================================================
// HTTP Client Helper
// ============================================================

// makeHTTPRequest يرسل طلب HTTP
func (epm *ExternalPlatformManager) makeHTTPRequest(method, url string, headers map[string]string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return epm.httpClient.Do(req)
}

// ============================================================
// المقاييس
// ============================================================

// GetMetrics يحصل على المقاييس
func (epm *ExternalPlatformManager) GetMetrics() *PlatformMetrics {
	epm.mu.RLock()
	defer epm.mu.RUnlock()

	return &PlatformMetrics{
		RequestsSent:     epm.metrics.RequestsSent,
		RequestsReceived: epm.metrics.RequestsReceived,
		Errors:           epm.metrics.Errors,
		LastActivity:     epm.metrics.LastActivity,
		PlatformsCount:    epm.metrics.PlatformsCount,
	}
}

package core

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

// NotificationManager مدير الإشعارات
type NotificationManager struct {
	channels    map[string]*NotificationChannel
	templates   map[string]*NotificationTemplate
	logger      *zap.Logger
	mu          sync.RWMutex
	sender      NotificationSender
	eventBus    EventBus
}

// NotificationSender واجهة مرسل الإشعارات
type NotificationSender interface {
	SendEmail(to, subject, body string) error
	SendSMS(to, message string) error
	SendPush(to, title, body string) error
	SendWebhook(url string, data interface{}) error
}

// EventBusNotificationSender مرسل إشعارات حقيقي ينشر الأحداث عبر EventBus
type EventBusNotificationSender struct {
	eb *eventbus.EventBus
}

// NewEventBusNotificationSender ينشئ مرسل إشعارات جديد
func NewEventBusNotificationSender(eb *eventbus.EventBus) *EventBusNotificationSender {
	return &EventBusNotificationSender{eb: eb}
}

func (s *EventBusNotificationSender) SendEmail(to, subject, body string) error {
	s.eb.Publish(eventbus.Event{
		Type: "notification.email",
		Payload: map[string]interface{}{
			"to":      to,
			"subject": subject,
			"body":    body,
		},
	})
	return nil
}

func (s *EventBusNotificationSender) SendSMS(to, message string) error {
	s.eb.Publish(eventbus.Event{
		Type: "notification.sms",
		Payload: map[string]interface{}{
			"to":      to,
			"message": message,
		},
	})
	return nil
}

func (s *EventBusNotificationSender) SendPush(to, title, body string) error {
	s.eb.Publish(eventbus.Event{
		Type: "notification.push",
		Payload: map[string]interface{}{
			"to":    to,
			"title": title,
			"body":  body,
		},
	})
	return nil
}

func (s *EventBusNotificationSender) SendWebhook(url string, data interface{}) error {
	s.eb.Publish(eventbus.Event{
		Type: "notification.webhook",
		Payload: map[string]interface{}{
			"url":  url,
			"data": data,
		},
	})
	return nil
}

// EventBus واجهة ناقل الأحداث
type EventBus interface {
	Publish(event string, data interface{}) error
	Subscribe(event string, handler func(data interface{})) error
}

// Notification إشعار
type Notification struct {
	ID          string                 `json:"id"`
	Type        NotificationType       `json:"type"`
	Priority    NotificationPriority   `json:"priority"`
	Title       string                 `json:"title"`
	Body        string                 `json:"body"`
	Recipient   string                 `json:"recipient"`
	Channel     string                 `json:"channel"`
	Status      NotificationStatus     `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	SentAt      time.Time              `json:"sent_at,omitempty"`
	ReadAt      time.Time              `json:"read_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NotificationType نوع الإشعار
type NotificationType string

const (
	NotificationTypeInfo    NotificationType = "info"
	NotificationTypeWarning NotificationType = "warning"
	NotificationTypeError   NotificationType = "error"
	NotificationTypeSuccess NotificationType = "success"
)

// NotificationPriority أولوية الإشعار
type NotificationPriority string

const (
	NotificationPriorityLow      NotificationPriority = "low"
	NotificationPriorityMedium   NotificationPriority = "medium"
	NotificationPriorityHigh     NotificationPriority = "high"
	NotificationPriorityCritical NotificationPriority = "critical"
)

// NotificationStatus حالة الإشعار
type NotificationStatus string

const (
	NotificationStatusPending  NotificationStatus = "pending"
	NotificationStatusSent     NotificationStatus = "sent"
	NotificationStatusDelivered NotificationStatus = "delivered"
	NotificationStatusFailed   NotificationStatus = "failed"
	NotificationStatusRead     NotificationStatus = "read"
)

// NotificationChannel قناة الإشعارات
type NotificationChannel struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        ChannelType            `json:"type"`
	Config      map[string]interface{} `json:"config"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ChannelType نوع القناة
type ChannelType string

const (
	ChannelTypeEmail   ChannelType = "email"
	ChannelTypeSMS     ChannelType = "sms"
	ChannelTypePush    ChannelType = "push"
	ChannelTypeWebhook ChannelType = "webhook"
)

// NotificationTemplate قالب الإشعار
type NotificationTemplate struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        NotificationType       `json:"type"`
	Subject     string                 `json:"subject"`
	Body        string                 `json:"body"`
	Variables   []string               `json:"variables"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// NewNotificationManager ينشئ مدير إشعارات جديد
func NewNotificationManager(logger *zap.Logger, sender NotificationSender, eventBus EventBus) *NotificationManager {
	return &NotificationManager{
		channels:  make(map[string]*NotificationChannel),
		templates: make(map[string]*NotificationTemplate),
		logger:    logger,
		sender:    sender,
		eventBus:  eventBus,
	}
}

// RegisterChannel يسجل قناة إشعارات جديدة
func (nm *NotificationManager) RegisterChannel(channel *NotificationChannel) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if _, exists := nm.channels[channel.ID]; exists {
		return fmt.Errorf("channel already registered: %s", channel.ID)
	}

	channel.CreatedAt = time.Now()
	channel.UpdatedAt = time.Now()
	nm.channels[channel.ID] = channel

	nm.logger.Info("تم تسجيل قناة إشعارات جديدة",
		zap.String("channel_id", channel.ID),
		zap.String("channel_name", channel.Name),
		zap.String("channel_type", string(channel.Type)))

	return nil
}

// UnregisterChannel يلغي تسجيل قناة إشعارات
func (nm *NotificationManager) UnregisterChannel(channelID string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if _, exists := nm.channels[channelID]; !exists {
		return fmt.Errorf("channel not found: %s", channelID)
	}

	delete(nm.channels, channelID)

	nm.logger.Info("تم إلغاء تسجيل قناة الإشعارات",
		zap.String("channel_id", channelID))

	return nil
}

// RegisterTemplate يسجل قالب إشعارات جديد
func (nm *NotificationManager) RegisterTemplate(template *NotificationTemplate) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if _, exists := nm.templates[template.ID]; exists {
		return fmt.Errorf("template already registered: %s", template.ID)
	}

	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()
	nm.templates[template.ID] = template

	nm.logger.Info("تم تسجيل قالب إشعارات جديد",
		zap.String("template_id", template.ID),
		zap.String("template_name", template.Name))

	return nil
}

// SendNotification يرسل إشعار
func (nm *NotificationManager) SendNotification(notification *Notification) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	notification.ID = fmt.Sprintf("notif_%d", time.Now().UnixNano())
	notification.Status = NotificationStatusPending
	notification.CreatedAt = time.Now()

	// الحصول على القناة
	channel, exists := nm.channels[notification.Channel]
	if !exists {
		return fmt.Errorf("channel not found: %s", notification.Channel)
	}

	if !channel.Enabled {
		return fmt.Errorf("channel is disabled: %s", notification.Channel)
	}

	// إرسال الإشعار حسب نوع القناة
	var err error
	switch channel.Type {
	case ChannelTypeEmail:
		err = nm.sender.SendEmail(notification.Recipient, notification.Title, notification.Body)
	case ChannelTypeSMS:
		err = nm.sender.SendSMS(notification.Recipient, notification.Body)
	case ChannelTypePush:
		err = nm.sender.SendPush(notification.Recipient, notification.Title, notification.Body)
	case ChannelTypeWebhook:
		err = nm.sender.SendWebhook(channel.Config["url"].(string), notification)
	default:
		return fmt.Errorf("unsupported channel type: %s", channel.Type)
	}

	if err != nil {
		notification.Status = NotificationStatusFailed
		nm.logger.Error("فشل إرسال الإشعار",
			zap.String("notification_id", notification.ID),
			zap.String("channel", notification.Channel),
			zap.Error(err))
		return err
	}

	notification.Status = NotificationStatusSent
	notification.SentAt = time.Now()

	nm.logger.Info("تم إرسال الإشعار بنجاح",
		zap.String("notification_id", notification.ID),
		zap.String("channel", notification.Channel),
		zap.String("recipient", notification.Recipient))

	// نشر حدث الإشعار
	if nm.eventBus != nil {
		nm.eventBus.Publish("notification.sent", map[string]interface{}{
			"notification_id": notification.ID,
			"channel":        notification.Channel,
			"recipient":      notification.Recipient,
		})
	}

	return nil
}

// SendNotificationFromTemplate يرسل إشعار من قالب
func (nm *NotificationManager) SendNotificationFromTemplate(templateID string, variables map[string]interface{}, recipient, channel string) error {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	template, exists := nm.templates[templateID]
	if !exists {
		return fmt.Errorf("template not found: %s", templateID)
	}

	// استبدال المتغيرات في القالب
	subject := template.Subject
	body := template.Body

	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		subject = fmt.Sprintf("%s", replacePlaceholder(subject, placeholder, fmt.Sprintf("%v", value)))
		body = fmt.Sprintf("%s", replacePlaceholder(body, placeholder, fmt.Sprintf("%v", value)))
	}

	notification := &Notification{
		Type:      template.Type,
		Title:     subject,
		Body:      body,
		Recipient: recipient,
		Channel:   channel,
		Metadata:  variables,
	}

	nm.mu.RUnlock()
	err := nm.SendNotification(notification)
	nm.mu.RLock()

	return err
}

// GetChannel يحصل على قناة
func (nm *NotificationManager) GetChannel(channelID string) (*NotificationChannel, error) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	channel, exists := nm.channels[channelID]
	if !exists {
		return nil, fmt.Errorf("channel not found: %s", channelID)
	}

	return channel, nil
}

// GetTemplate يحصل على قالب
func (nm *NotificationManager) GetTemplate(templateID string) (*NotificationTemplate, error) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	template, exists := nm.templates[templateID]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateID)
	}

	return template, nil
}

// GetAllChannels يحصل على جميع القنوات
func (nm *NotificationManager) GetAllChannels() []*NotificationChannel {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	channels := make([]*NotificationChannel, 0, len(nm.channels))
	for _, channel := range nm.channels {
		channels = append(channels, channel)
	}

	return channels
}

// GetAllTemplates يحصل على جميع القوالب
func (nm *NotificationManager) GetAllTemplates() []*NotificationTemplate {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	templates := make([]*NotificationTemplate, 0, len(nm.templates))
	for _, template := range nm.templates {
		templates = append(templates, template)
	}

	return templates
}

// GetSummary يحصل على ملخص الإشعارات
func (nm *NotificationManager) GetSummary() map[string]interface{} {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	totalChannels := len(nm.channels)
	totalTemplates := len(nm.templates)
	enabledChannels := 0

	for _, channel := range nm.channels {
		if channel.Enabled {
			enabledChannels++
		}
	}

	return map[string]interface{}{
		"total_channels":   totalChannels,
		"enabled_channels": enabledChannels,
		"total_templates":  totalTemplates,
	}
}

// replacePlaceholder يستبدل placeholder في النص
func replacePlaceholder(text, placeholder, value string) string {
	return strings.ReplaceAll(text, placeholder, value)
}

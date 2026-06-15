package orchestrator

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/channel"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/MortalArena/Musketeers/pkg/session"
	"go.uber.org/zap"
)

// ============================================================
// ChatConnector - ربط الشات والقنوات بالنظام
// ============================================================

// ChatConnector يربط الشات والقنوات مع Event Bus والوكلاء
type ChatConnector struct {
	// المكونات الأساسية
	eventBus      *eventbus.EventBus
	agentRegistry *agent.AgentRegistry
	sessionMgr    *session.SessionContainer

	// القنوات
	privateChannels map[string]*channel.ChannelConfig // channelID -> config
	publicChannels  map[string]bool                   // channelID -> exists
	sessionChannels map[string]string                 // sessionID -> channelID

	// المفاتيح
	privateKey ed25519.PrivateKey

	// Channels للتواصل الداخلي
	chatToEventBus chan *ChatMessage
	eventBusToChat chan eventbus.Event

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Logger
	logger *zap.Logger

	// Metrics
	metrics *ChatMetrics
	mu      sync.RWMutex
}

// ChatMetrics مقاييس الشات
type ChatMetrics struct {
	MessagesSent     int64
	MessagesReceived int64
	PrivateChannels  int
	PublicChannels   int
	SessionChannels  int
	Errors           int64
	LastActivity     time.Time
}

// ChatMessage رسالة شات
type ChatMessage struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // private, public, session
	ChannelID   string                 `json:"channel_id"`
	SenderDID   string                 `json:"sender_did"`
	Content     string                 `json:"content"`
	Prompt      string                 `json:"prompt,omitempty"`
	TargetAgent string                 `json:"target_agent,omitempty"` // للقنوات الخاصة
	SessionID   string                 `json:"session_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// NewChatConnector ينشئ ChatConnector جديد
func NewChatConnector(
	eventBus *eventbus.EventBus,
	agentRegistry *agent.AgentRegistry,
	sessionMgr *session.SessionContainer,
	privateKey ed25519.PrivateKey,
	logger *zap.Logger,
) *ChatConnector {
	ctx, cancel := context.WithCancel(context.Background())

	return &ChatConnector{
		eventBus:        eventBus,
		agentRegistry:   agentRegistry,
		sessionMgr:      sessionMgr,
		privateChannels: make(map[string]*channel.ChannelConfig),
		publicChannels:  make(map[string]bool),
		sessionChannels: make(map[string]string),
		privateKey:      privateKey,
		chatToEventBus:  make(chan *ChatMessage, 1000),
		eventBusToChat:  make(chan eventbus.Event, 1000),
		ctx:             ctx,
		cancel:          cancel,
		logger:          logger,
		metrics:         &ChatMetrics{},
	}
}

// Start يبدأ ChatConnector
func (cc *ChatConnector) Start() error {
	cc.logger.Info("بدء ChatConnector")

	// الاشتراك في أحداث Event Bus
	cc.subscribeToEventBus()

	// بدء معالج الشات
	cc.wg.Add(1)
	go cc.chatHandler()

	// بدء معالج Event Bus
	cc.wg.Add(1)
	go cc.eventBusHandler()

	cc.logger.Info("تم بدء ChatConnector بنجاح")
	return nil
}

// Stop يوقف ChatConnector
func (cc *ChatConnector) Stop() error {
	cc.logger.Info("إيقاف ChatConnector")

	cc.cancel()
	cc.wg.Wait()

	close(cc.chatToEventBus)
	close(cc.eventBusToChat)

	cc.logger.Info("تم إيقاف ChatConnector بنجاح")
	return nil
}

// ============================================================
// القنوات الخاصة (Private Channels)
// ============================================================

// CreatePrivateChannel ينشئ قناة خاصة بين العميل والوكيل
func (cc *ChatConnector) CreatePrivateChannel(clientDID, agentID string) (string, error) {
	channelID := fmt.Sprintf("private_%s_%s", clientDID, agentID)

	members := []string{clientDID, agentID}
	admins := []string{clientDID}

	cfg, _, err := channel.NewPrivateChannel(channelID, clientDID, cc.privateKey, members, admins)
	if err != nil {
		return "", fmt.Errorf("فشل إنشاء قناة خاصة: %w", err)
	}

	cc.mu.Lock()
	cc.privateChannels[channelID] = cfg
	cc.metrics.PrivateChannels++
	cc.mu.Unlock()

	cc.logger.Info("تم إنشاء قناة خاصة",
		zap.String("channel_id", channelID),
		zap.String("client", clientDID),
		zap.String("agent", agentID),
	)

	return channelID, nil
}

// SendToPrivateChannel يرسل رسالة إلى قناة خاصة
func (cc *ChatConnector) SendToPrivateChannel(channelID, senderDID, content, prompt string) error {
	cc.mu.RLock()
	cfg, exists := cc.privateChannels[channelID]
	cc.mu.RUnlock()

	if !exists {
		return fmt.Errorf("القناة %s غير موجودة", channelID)
	}

	// التحقق من أن المرسل عضو في القناة
	isMember := false
	for _, member := range cfg.Members {
		if member == senderDID {
			isMember = true
			break
		}
	}

	if !isMember {
		return fmt.Errorf("المرسل ليس عضواً في القناة")
	}

	msg := &ChatMessage{
		ID:        generateChatID(),
		Type:      "private",
		ChannelID: channelID,
		SenderDID: senderDID,
		Content:   content,
		Prompt:    prompt,
		Timestamp: time.Now(),
	}

	cc.chatToEventBus <- msg

	cc.updateMetrics()
	return nil
}

// ============================================================
// القنوات العامة (Public Channels)
// ============================================================

// CreatePublicChannel ينشئ قناة عامة
func (cc *ChatConnector) CreatePublicChannel(channelID string) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if _, exists := cc.publicChannels[channelID]; exists {
		return fmt.Errorf("القناة %s موجودة بالفعل", channelID)
	}

	cc.publicChannels[channelID] = true
	cc.metrics.PublicChannels++

	cc.logger.Info("تم إنشاء قناة عامة",
		zap.String("channel_id", channelID),
	)

	return nil
}

// SendToPublicChannel يرسل رسالة إلى قناة عامة
func (cc *ChatConnector) SendToPublicChannel(channelID, senderDID, content string) error {
	cc.mu.RLock()
	_, exists := cc.publicChannels[channelID]
	cc.mu.RUnlock()

	if !exists {
		return fmt.Errorf("القناة %s غير موجودة", channelID)
	}

	msg := &ChatMessage{
		ID:        generateChatID(),
		Type:      "public",
		ChannelID: channelID,
		SenderDID: senderDID,
		Content:   content,
		Timestamp: time.Now(),
	}

	cc.chatToEventBus <- msg

	cc.updateMetrics()
	return nil
}

// ============================================================
// قنوات الجلسة (Session Channels)
// ============================================================

// CreateSessionChannel ينشئ قناة لجلسة
func (cc *ChatConnector) CreateSessionChannel(sessionID string) (string, error) {
	channelID := fmt.Sprintf("session_%s", sessionID)

	cc.mu.Lock()
	cc.sessionChannels[sessionID] = channelID
	cc.metrics.SessionChannels++
	cc.mu.Unlock()

	cc.logger.Info("تم إنشاء قناة جلسة",
		zap.String("session_id", sessionID),
		zap.String("channel_id", channelID),
	)

	return channelID, nil
}

// SendToSessionChannel يرسل رسالة إلى قناة جلسة
func (cc *ChatConnector) SendToSessionChannel(sessionID, senderDID, content, prompt string) error {
	cc.mu.RLock()
	channelID, exists := cc.sessionChannels[sessionID]
	cc.mu.RUnlock()

	if !exists {
		return fmt.Errorf("قناة الجلسة %s غير موجودة", sessionID)
	}

	msg := &ChatMessage{
		ID:        generateChatID(),
		Type:      "session",
		ChannelID: channelID,
		SenderDID: senderDID,
		Content:   content,
		Prompt:    prompt,
		SessionID: sessionID,
		Timestamp: time.Now(),
	}

	cc.chatToEventBus <- msg

	cc.updateMetrics()
	return nil
}

// ============================================================
// معالجات الرسائل
// ============================================================

// subscribeToEventBus يرتبط بأحداث Event Bus
func (cc *ChatConnector) subscribeToEventBus() {
	cc.eventBus.Subscribe("agent.response", cc.handleAgentResponse)
	cc.eventBus.Subscribe("agent.message", cc.handleAgentMessage)
}

// chatHandler يعالج رسائل الشات
func (cc *ChatConnector) chatHandler() {
	defer cc.wg.Done()

	for {
		select {
		case <-cc.ctx.Done():
			return
		case msg := <-cc.chatToEventBus:
			cc.processChatMessage(msg)
		}
	}
}

// processChatMessage يعالج رسالة شات
func (cc *ChatConnector) processChatMessage(msg *ChatMessage) {
	// تحويل الرسالة إلى حدث Event Bus
	event := eventbus.Event{
		Type:      "chat.message",
		Payload:   msg,
		Source:    msg.SenderDID,
		SessionID: msg.SessionID,
	}

	// نشر الحدث
	cc.eventBus.Publish(event)

	cc.mu.Lock()
	cc.metrics.MessagesSent++
	cc.mu.Unlock()

	cc.logger.Debug("تم معالجة رسالة شات",
		zap.String("type", msg.Type),
		zap.String("channel_id", msg.ChannelID),
		zap.String("sender", msg.SenderDID),
	)
}

// eventBusHandler يعالج أحداث Event Bus
func (cc *ChatConnector) eventBusHandler() {
	defer cc.wg.Done()

	for {
		select {
		case <-cc.ctx.Done():
			return
		case event := <-cc.eventBusToChat:
			cc.processEventBusEvent(event)
		}
	}
}

// processEventBusEvent يعالج حدث Event Bus
func (cc *ChatConnector) processEventBusEvent(event eventbus.Event) {
	cc.mu.Lock()
	cc.metrics.MessagesReceived++
	cc.mu.Unlock()

	cc.logger.Debug("تم معالجة حدث Event Bus",
		zap.String("event_type", event.Type),
	)
}

// handleAgentResponse يعالج رد من وكيل
func (cc *ChatConnector) handleAgentResponse(event eventbus.Event) {
	cc.logger.Debug("استقبال رد من وكيل",
		zap.String("agent_id", event.Source),
	)
}

// handleAgentMessage يعالج رسالة من وكيل
func (cc *ChatConnector) handleAgentMessage(event eventbus.Event) {
	cc.logger.Debug("استقبال رسالة من وكيل",
		zap.String("agent_id", event.Source),
	)
}

// ============================================================
// المقاييس
// ============================================================

func (cc *ChatConnector) updateMetrics() {
	cc.mu.Lock()
	cc.metrics.LastActivity = time.Now()
	cc.mu.Unlock()
}

// GetMetrics يحصل على المقاييس
func (cc *ChatConnector) GetMetrics() *ChatMetrics {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	return &ChatMetrics{
		MessagesSent:     cc.metrics.MessagesSent,
		MessagesReceived: cc.metrics.MessagesReceived,
		PrivateChannels:  cc.metrics.PrivateChannels,
		PublicChannels:   cc.metrics.PublicChannels,
		SessionChannels:  cc.metrics.SessionChannels,
		Errors:           cc.metrics.Errors,
		LastActivity:     cc.metrics.LastActivity,
	}
}

// ============================================================
// دوال مساعدة
// ============================================================

func generateChatID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

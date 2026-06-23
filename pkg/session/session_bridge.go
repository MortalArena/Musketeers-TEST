package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

// SessionBridge جسر لربط جلسات منفصلة معاً
// [WHY] يسمح للعميل البشري بربط مشاريع منفصلة لتنفيذ مشروع ضخم
// [HOW] ينقل الأحداث والبيانات بين الجلسات عبر قنوات آمنة
// [SAFETY] يضمن عدم التضارب وعزل البيانات
type SessionBridge struct {
	bridgeID   string
	sourceID   string
	targetID   string
	bridgeType BridgeType
	status     BridgeStatus
	eventBus   *eventbus.EventBus
	logger     *zap.Logger
	mu         sync.RWMutex

	// قنوات الاتصال
	sourceToTarget chan *BridgeMessage
	targetToSource chan *BridgeMessage

	// إحصائيات
	messagesSent     int64
	messagesReceived int64
	bytesTransferred int64
	lastActivity     time.Time

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// BridgeType نوع الجسر
type BridgeType string

const (
	BridgeTypeOneWay BridgeType = "one_way" // اتجاه واحد
	BridgeTypeTwoWay BridgeType = "two_way" // اتجاهين
	BridgeTypeMulti  BridgeType = "multi"   // متعدد الاتجاهات
	BridgeTypeSync   BridgeType = "sync"    // مزامنة كاملة
)

// BridgeStatus حالة الجسر
type BridgeStatus string

const (
	BridgeStatusIdle   BridgeStatus = "idle"
	BridgeStatusActive BridgeStatus = "active"
	BridgeStatusPaused BridgeStatus = "paused"
	BridgeStatusError  BridgeStatus = "error"
	BridgeStatusClosed BridgeStatus = "closed"
)

// BridgeMessage رسالة جسر
type BridgeMessage struct {
	ID        string                 `json:"id"`
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Type      string                 `json:"type"` // event, data, command, response
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
}

// BridgeConfig إعدادات الجسر
type BridgeConfig struct {
	BridgeID   string
	SourceID   string
	TargetID   string
	BridgeType BridgeType
	BufferSize int // حجم الـ buffer للقنوات
}

// [SAFETY] حدود الموارد لمنع استهلاك غير محدود
const (
	// [SAFETY] الحد الأقصى لحجم الـ buffer
	MaxBridgeBufferSize = 10000
	// [SAFETY] الحد الأقصى لعدد الرسائل
	MaxBridgeMessages = 10000
	// [SAFETY] الحد الأقصى لحجم الرسالة (10MB)
	MaxBridgeMessageSize = 10 * 1024 * 1024
	// [SAFETY] الحد الأقصى لعدد الجسور
	MaxBridges = 100
)

// NewSessionBridge ينشئ جسر جلسة جديد
func NewSessionBridge(config *BridgeConfig, eventBus *eventbus.EventBus, logger *zap.Logger) *SessionBridge {
	// [SAFETY] التحقق من صحة الإعدادات
	if config == nil {
		config = &BridgeConfig{}
	}
	if config.BridgeID == "" {
		config.BridgeID = fmt.Sprintf("bridge_%d", time.Now().UnixNano())
	}
	if config.SourceID == "" {
		config.SourceID = "unknown"
	}
	if config.TargetID == "" {
		config.TargetID = "unknown"
	}

	ctx, cancel := context.WithCancel(context.Background())

	bufferSize := config.BufferSize
	if bufferSize == 0 {
		bufferSize = 1000 // حجم افتراضي
	}

	// [SAFETY] التحقق من الحد الأقصى لحجم الـ buffer
	if bufferSize > MaxBridgeBufferSize {
		bufferSize = MaxBridgeBufferSize
	}

	return &SessionBridge{
		bridgeID:       config.BridgeID,
		sourceID:       config.SourceID,
		targetID:       config.TargetID,
		bridgeType:     config.BridgeType,
		status:         BridgeStatusIdle,
		eventBus:       eventBus,
		logger:         logger,
		sourceToTarget: make(chan *BridgeMessage, bufferSize),
		targetToSource: make(chan *BridgeMessage, bufferSize),
		lastActivity:   time.Now(),
		ctx:            ctx,
		cancel:         cancel,
	}
}

// Start يبدأ الجسر
func (sb *SessionBridge) Start() error {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	sb.status = BridgeStatusActive
	sb.logger.Info("بدء جسر الجلسة",
		zap.String("bridge_id", sb.bridgeID),
		zap.String("source", sb.sourceID),
		zap.String("target", sb.targetID),
		zap.String("type", string(sb.bridgeType)),
	)

	// بدء معالج الرسائل من المصدر إلى الهدف
	sb.wg.Add(1)
	go sb.processSourceToTarget()

	// بدء معالج الرسائل من الهدف إلى المصدر (إذا كان جسر اتجاهين)
	if sb.bridgeType == BridgeTypeTwoWay || sb.bridgeType == BridgeTypeMulti {
		sb.wg.Add(1)
		go sb.processTargetToSource()
	}

	return nil
}

// Stop يوقف الجسر
func (sb *SessionBridge) Stop() error {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	sb.status = BridgeStatusClosed
	sb.cancel()
	sb.wg.Wait()

	close(sb.sourceToTarget)
	close(sb.targetToSource)

	sb.logger.Info("إيقاف جسر الجلسة",
		zap.String("bridge_id", sb.bridgeID),
		zap.Int64("messages_sent", sb.messagesSent),
		zap.Int64("messages_received", sb.messagesReceived),
	)

	return nil
}

// SendMessage يرسل رسالة عبر الجسر
func (sb *SessionBridge) SendMessage(ctx context.Context, msg *BridgeMessage) error {
	// [SAFETY] التحقق من صحة المدخلات
	if msg == nil {
		return fmt.Errorf("message cannot be nil")
	}
	if msg.From == "" {
		return fmt.Errorf("message from cannot be empty")
	}
	if msg.To == "" {
		return fmt.Errorf("message to cannot be empty")
	}
	if msg.Type == "" {
		return fmt.Errorf("message type cannot be empty")
	}

	// [SAFETY] التحقق من الحد الأقصى لحجم الرسالة
	if len(msg.Content) > MaxBridgeMessageSize {
		return fmt.Errorf("message size too large (max %d bytes)", MaxBridgeMessageSize)
	}

	sb.mu.RLock()
	defer sb.mu.RUnlock()

	if sb.status != BridgeStatusActive {
		return fmt.Errorf("جسر غير نشط: %s", sb.status)
	}

	// [SAFETY] التحقق من الحد الأقصى لعدد الرسائل
	if sb.messagesSent >= MaxBridgeMessages {
		return fmt.Errorf("maximum messages limit reached (%d)", MaxBridgeMessages)
	}

	msg.ID = fmt.Sprintf("msg_%d", time.Now().UnixNano())
	msg.Timestamp = time.Now()

	// إرسال الرسالة
	select {
	case sb.sourceToTarget <- msg:
		sb.messagesSent++
		sb.bytesTransferred += int64(len(msg.Content))
		sb.lastActivity = time.Now()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout sending message")
	}
}

// processSourceToTarget يعالج الرسائل من المصدر إلى الهدف
func (sb *SessionBridge) processSourceToTarget() {
	defer sb.wg.Done()

	for {
		select {
		case msg, ok := <-sb.sourceToTarget:
			if !ok {
				return
			}

			// نشر الحدث على EventBus
			sb.publishBridgeEvent(msg)

			sb.messagesReceived++
			sb.lastActivity = time.Now()

		case <-sb.ctx.Done():
			return
		}
	}
}

// processTargetToSource يعالج الرسائل من الهدف إلى المصدر
func (sb *SessionBridge) processTargetToSource() {
	defer sb.wg.Done()

	for {
		select {
		case msg, ok := <-sb.targetToSource:
			if !ok {
				return
			}

			// نشر الحدث على EventBus
			sb.publishBridgeEvent(msg)

			sb.messagesReceived++
			sb.lastActivity = time.Now()

		case <-sb.ctx.Done():
			return
		}
	}
}

// publishBridgeEvent ينشر حدث الجسر على EventBus
func (sb *SessionBridge) publishBridgeEvent(msg *BridgeMessage) {
	event := eventbus.Event{
		Type:      "bridge.message",
		Source:    sb.bridgeID,
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"bridge_id": sb.bridgeID,
			"message":   msg,
		},
	}

	sb.eventBus.Publish(event)
}

// GetStatus يحصل على حالة الجسر
func (sb *SessionBridge) GetStatus() BridgeStatus {
	sb.mu.RLock()
	defer sb.mu.RUnlock()
	return sb.status
}

// GetStats يحصل على إحصائيات الجسر
func (sb *SessionBridge) GetStats() map[string]interface{} {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	return map[string]interface{}{
		"bridge_id":         sb.bridgeID,
		"source_id":         sb.sourceID,
		"target_id":         sb.targetID,
		"bridge_type":       sb.bridgeType,
		"status":            sb.status,
		"messages_sent":     sb.messagesSent,
		"messages_received": sb.messagesReceived,
		"bytes_transferred": sb.bytesTransferred,
		"last_activity":     sb.lastActivity,
	}
}

// Pause يوقف الجسر مؤقتاً
func (sb *SessionBridge) Pause() error {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	sb.status = BridgeStatusPaused
	sb.logger.Info("إيقاف جسر الجلسة مؤقتاً", zap.String("bridge_id", sb.bridgeID))
	return nil
}

// Resume يستأنف الجسر
func (sb *SessionBridge) Resume() error {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	sb.status = BridgeStatusActive
	sb.logger.Info("استئناف جسر الجلسة", zap.String("bridge_id", sb.bridgeID))
	return nil
}

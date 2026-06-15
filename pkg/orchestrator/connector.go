package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/agent_bridge"
	"github.com/MortalArena/Musketeers/pkg/agent_bridge/protocol"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

// ============================================================
// Connector - الواجهة البسيطة لربط Bridge و Event Bus و Adapters
// ============================================================

// Connector يربط Bridge و Event Bus و Adapters معاً
type Connector struct {
	// المكونات الأساسية
	eventBus      *eventbus.EventBus
	bridge        *agent_bridge.MultiplexedBridge
	agentRegistry *agent.AgentRegistry

	// Adapters - المحولات بين الأنظمة المختلفة
	adapters map[string]Adapter

	// Channels - القنوات للتواصل الداخلي
	bridgeToEventBus chan *protocol.Message
	eventBusToBridge chan eventbus.Event

	// Lifecycle - دورة الحياة
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Logger
	logger *zap.Logger

	// Metrics - المقاييس
	metrics *ConnectorMetrics
	mu      sync.RWMutex
}

// ConnectorMetrics مقاييس Connector
type ConnectorMetrics struct {
	MessagesReceived int64
	MessagesSent     int64
	EventsPublished  int64
	EventsHandled    int64
	Errors           int64
	LastActivity     time.Time
}

// NewConnector ينشئ Connector جديد
func NewConnector(
	eventBus *eventbus.EventBus,
	bridge *agent_bridge.MultiplexedBridge,
	agentRegistry *agent.AgentRegistry,
	logger *zap.Logger,
) *Connector {
	ctx, cancel := context.WithCancel(context.Background())

	return &Connector{
		eventBus:         eventBus,
		bridge:           bridge,
		agentRegistry:    agentRegistry,
		adapters:         make(map[string]Adapter),
		bridgeToEventBus: make(chan *protocol.Message, 1000),
		eventBusToBridge: make(chan eventbus.Event, 1000),
		ctx:              ctx,
		cancel:           cancel,
		logger:           logger,
		metrics:          &ConnectorMetrics{},
	}
}

// Start يبدأ Connector
func (c *Connector) Start() error {
	c.logger.Info("بدء Connector")

	// تسجيل Adapters الافتراضية
	c.registerDefaultAdapters()

	// الاشتراك في أحداث Event Bus
	c.subscribeToEventBus()

	// بدء معالج Bridge
	c.wg.Add(1)
	go c.bridgeHandler()

	// بدء معالج Event Bus
	c.wg.Add(1)
	go c.eventBusHandler()

	// بدء معالج الرسائل من Bridge
	c.wg.Add(1)
	go c.bridgeMessageProcessor()

	c.logger.Info("تم بدء Connector بنجاح")
	return nil
}

// Stop يوقف Connector
func (c *Connector) Stop() error {
	c.logger.Info("إيقاف Connector")

	c.cancel()
	c.wg.Wait()

	close(c.bridgeToEventBus)
	close(c.eventBusToBridge)

	c.logger.Info("تم إيقاف Connector بنجاح")
	return nil
}

// ============================================================
// Adapters - تسجيل واستخدام المحولات
// ============================================================

// registerDefaultAdapters يسجل المحولات الافتراضية
func (c *Connector) registerDefaultAdapters() {
	// Bridge Adapter - يحول رسائل Bridge إلى أحداث Event Bus
	c.adapters["bridge_to_event"] = &BridgeToEventAdapter{
		connector: c,
	}

	// Event Adapter - يحول أحداث Event Bus إلى رسائل Bridge
	c.adapters["event_to_bridge"] = &EventToBridgeAdapter{
		connector: c,
	}

	// Agent Adapter - يحول رسائل الوكلاء
	c.adapters["agent"] = &AgentAdapter{
		registry: c.agentRegistry,
	}

	c.logger.Info("تم تسجيل المحولات الافتراضية")
}

// RegisterAdapter يسجل محول جديد
func (c *Connector) RegisterAdapter(name string, adapter Adapter) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.adapters[name]; exists {
		return fmt.Errorf("المحول %s مسجل بالفعل", name)
	}

	c.adapters[name] = adapter
	c.logger.Info("تم تسجيل محول جديد", zap.String("name", name))
	return nil
}

// GetAdapter يحصل على محول بالاسم
func (c *Connector) GetAdapter(name string) (Adapter, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	adapter, exists := c.adapters[name]
	if !exists {
		return nil, fmt.Errorf("المحول %s غير موجود", name)
	}

	return adapter, nil
}

// ============================================================
// Bridge Handler - معالج Bridge
// ============================================================

// subscribeToEventBus يرتبط بأحداث Event Bus
func (c *Connector) subscribeToEventBus() {
	// الاستماع لكل الأحداث
	c.eventBus.Subscribe("*", func(event eventbus.Event) {
		c.eventBusToBridge <- event
	})

	// أحداث محددة مهمة
	c.eventBus.Subscribe("agent.message", c.handleAgentMessage)
	c.eventBus.Subscribe("agent.response", c.handleAgentResponse)
	c.eventBus.Subscribe("task.created", c.handleTaskCreated)
	c.eventBus.Subscribe("task.completed", c.handleTaskCompleted)
}

// bridgeHandler يعالج الرسائل من Bridge
func (c *Connector) bridgeHandler() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			// قراءة من كل مسار في Bridge
			c.processLane(agent_bridge.LaneEmergency)
			c.processLane(agent_bridge.LaneChat)
			c.processLane(agent_bridge.LaneWorkflow)
			c.processLane(agent_bridge.LaneFileUpload)
			c.processLane(agent_bridge.LaneFileDownload)

			time.Sleep(10 * time.Millisecond)
		}
	}
}

// processLane يعالج مسار معين
func (c *Connector) processLane(laneType agent_bridge.LaneType) {
	// قراءة رسالة من المسار
	msg, err := c.bridge.Receive(laneType)
	if err != nil {
		return
	}
	if msg == nil {
		return
	}

	// تحديث المقاييس
	c.mu.Lock()
	c.metrics.MessagesReceived++
	c.metrics.LastActivity = time.Now()
	c.mu.Unlock()

	// إرسال للمعالج
	c.bridgeToEventBus <- msg
}

// bridgeMessageProcessor يعالج الرسائل من Bridge
func (c *Connector) bridgeMessageProcessor() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		case msg := <-c.bridgeToEventBus:
			c.processBridgeMessage(msg)
		}
	}
}

// processBridgeMessage يعالج رسالة Bridge
func (c *Connector) processBridgeMessage(msg *protocol.Message) {
	// استخدام Bridge Adapter لتحويل الرسالة إلى حدث
	adapter, err := c.GetAdapter("bridge_to_event")
	if err != nil {
		c.logger.Error("فشل الحصول على المحول", zap.Error(err))
		c.recordError()
		return
	}

	// تحويل الرسالة
	event, err := adapter.Convert(msg)
	if err != nil {
		c.logger.Error("فشل تحويل الرسالة", zap.Error(err))
		c.recordError()
		return
	}

	// نشر الحدث
	c.eventBus.Publish(event.(eventbus.Event))

	// تحديث المقاييس
	c.mu.Lock()
	c.metrics.EventsPublished++
	c.mu.Unlock()

	c.logger.Debug("تم تحويل رسالة Bridge إلى حدث",
		zap.String("message_type", string(msg.Type)),
		zap.String("event_type", event.(eventbus.Event).Type),
	)
}

// ============================================================
// Event Bus Handler - معالج Event Bus
// ============================================================

// eventBusHandler يعالج الأحداث من Event Bus
func (c *Connector) eventBusHandler() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		case event := <-c.eventBusToBridge:
			c.processEventBusEvent(event)
		}
	}
}

// processEventBusEvent يعالج حدث Event Bus
func (c *Connector) processEventBusEvent(event eventbus.Event) {
	// استخدام Event Adapter لتحويل الحدث إلى رسالة Bridge
	adapter, err := c.GetAdapter("event_to_bridge")
	if err != nil {
		c.logger.Error("فشل الحصول على المحول", zap.Error(err))
		c.recordError()
		return
	}

	// تحويل الحدث
	msg, err := adapter.Convert(event)
	if err != nil {
		c.logger.Error("فشل تحويل الحدث", zap.Error(err))
		c.recordError()
		return
	}

	// إرسال الرسالة عبر Bridge
	laneType := c.determineLane(event)
	err = c.bridge.Send(laneType, msg.(*protocol.Message))
	if err != nil {
		c.logger.Error("فشل إرسال الرسالة عبر Bridge", zap.Error(err))
		c.recordError()
		return
	}

	// تحديث المقاييس
	c.mu.Lock()
	c.metrics.MessagesSent++
	c.mu.Unlock()

	c.logger.Debug("تم تحويل حدث Event Bus إلى رسالة",
		zap.String("event_type", event.Type),
		zap.String("lane", laneType.String()),
	)
}

// determineLane يحدد المسار المناسب للحدث
func (c *Connector) determineLane(event eventbus.Event) agent_bridge.LaneType {
	switch event.Type {
	case "emergency", "error", "critical":
		return agent_bridge.LaneEmergency
	case "chat", "message":
		return agent_bridge.LaneChat
	case "workflow", "task":
		return agent_bridge.LaneWorkflow
	case "file.upload":
		return agent_bridge.LaneFileUpload
	case "file.download":
		return agent_bridge.LaneFileDownload
	default:
		return agent_bridge.LaneChat
	}
}

// ============================================================
// Event Handlers - معالجات الأحداث المحددة
// ============================================================

func (c *Connector) handleAgentMessage(event eventbus.Event) {
	c.logger.Debug("استقبال رسالة من وكيل",
		zap.String("agent_id", event.Source),
	)
}

func (c *Connector) handleAgentResponse(event eventbus.Event) {
	c.logger.Debug("استقبال رد من وكيل",
		zap.String("agent_id", event.Source),
	)
}

func (c *Connector) handleTaskCreated(event eventbus.Event) {
	c.logger.Debug("تم إنشاء مهمة جديدة",
		zap.String("task_id", fmt.Sprintf("%v", event.Payload)),
	)
}

func (c *Connector) handleTaskCompleted(event eventbus.Event) {
	c.logger.Debug("تم إكمال مهمة",
		zap.String("task_id", fmt.Sprintf("%v", event.Payload)),
	)
}

// ============================================================
// Metrics - المقاييس
// ============================================================

func (c *Connector) recordError() {
	c.mu.Lock()
	c.metrics.Errors++
	c.mu.Unlock()
}

// GetMetrics يحصل على المقاييس
func (c *Connector) GetMetrics() *ConnectorMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return &ConnectorMetrics{
		MessagesReceived: c.metrics.MessagesReceived,
		MessagesSent:     c.metrics.MessagesSent,
		EventsPublished:  c.metrics.EventsPublished,
		EventsHandled:    c.metrics.EventsHandled,
		Errors:           c.metrics.Errors,
		LastActivity:     c.metrics.LastActivity,
	}
}

// ============================================================
// Adapter Interface - واجهة المحول
// ============================================================

// Adapter واجهة المحول
type Adapter interface {
	// Convert يحول البيانات من نوع إلى آخر
	Convert(data interface{}) (interface{}, error)

	// Name يرجع اسم المحول
	Name() string
}

// ============================================================
// BridgeToEventAdapter - محول من Bridge إلى Event Bus
// ============================================================

// BridgeToEventAdapter يحول رسائل Bridge إلى أحداث Event Bus
type BridgeToEventAdapter struct {
	connector *Connector
}

func (a *BridgeToEventAdapter) Name() string {
	return "bridge_to_event"
}

func (a *BridgeToEventAdapter) Convert(data interface{}) (interface{}, error) {
	msg, ok := data.(*protocol.Message)
	if !ok {
		return nil, fmt.Errorf("البيانات ليست رسالة Bridge")
	}

	// تحويل رسالة Bridge إلى حدث Event Bus
	event := eventbus.Event{
		Type:    a.determineEventType(msg),
		Payload: msg.Data,
	}

	return event, nil
}

func (a *BridgeToEventAdapter) determineEventType(msg *protocol.Message) string {
	switch msg.Type {
	case protocol.MessageTypeTaskRequest:
		return "task.request"
	case protocol.MessageTypeTaskResponse:
		return "task.response"
	case protocol.MessageTypeHeartbeat:
		return "heartbeat"
	case protocol.MessageTypeError:
		return "error"
	default:
		return "message"
	}
}

// ============================================================
// EventToBridgeAdapter - محول من Event Bus إلى Bridge
// ============================================================

// EventToBridgeAdapter يحول أحداث Event Bus إلى رسائل Bridge
type EventToBridgeAdapter struct {
	connector *Connector
}

func (a *EventToBridgeAdapter) Name() string {
	return "event_to_bridge"
}

func (a *EventToBridgeAdapter) Convert(data interface{}) (interface{}, error) {
	event, ok := data.(eventbus.Event)
	if !ok {
		return nil, fmt.Errorf("البيانات ليست حدث Event Bus")
	}

	// تحويل حدث Event Bus إلى رسالة Bridge
	msg := &protocol.Message{
		Type: a.determineMessageType(event),
		Data: a.eventToBytes(event),
	}

	return msg, nil
}

func (a *EventToBridgeAdapter) determineMessageType(event eventbus.Event) protocol.MessageType {
	switch event.Type {
	case "task.request":
		return protocol.MessageTypeTaskRequest
	case "task.response":
		return protocol.MessageTypeTaskResponse
	case "heartbeat":
		return protocol.MessageTypeHeartbeat
	case "error":
		return protocol.MessageTypeError
	default:
		return protocol.MessageTypeTaskRequest
	}
}

func (a *EventToBridgeAdapter) eventToBytes(event eventbus.Event) []byte {
	// تحويل الحدث إلى bytes
	// في التنفيذ الحقيقي، يجب استخدام JSON أو أي تنسيق آخر
	data := map[string]interface{}{
		"type":       event.Type,
		"payload":    event.Payload,
		"source":     event.Source,
		"session_id": event.SessionID,
		"timestamp":  event.Timestamp,
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return []byte("{}")
	}
	return bytes
}

// ============================================================
// AgentAdapter - محول الوكلاء
// ============================================================

// AgentAdapter يحول رسائل الوكلاء
type AgentAdapter struct {
	registry *agent.AgentRegistry
}

func (a *AgentAdapter) Name() string {
	return "agent"
}

func (a *AgentAdapter) Convert(data interface{}) (interface{}, error) {
	// تحويل البيانات بناءً على النوع
	switch v := data.(type) {
	case *agent.AgentTask:
		// تحويل مهمة وكيل إلى حدث
		return eventbus.Event{
			Type:    "agent.task",
			Payload: v,
		}, nil
	case *agent.TaskExecutionResult:
		// تحويل نتيجة مهمة إلى حدث
		return eventbus.Event{
			Type:    "agent.result",
			Payload: v,
		}, nil
	default:
		return nil, fmt.Errorf("نوع البيانات غير مدعوم: %T", v)
	}
}

// ============================================================
// Utility Functions - دوال مساعدة
// ============================================================

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

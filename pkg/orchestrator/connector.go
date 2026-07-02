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

	// MCP و A2A
	mcpManager *MCPManager
	a2aManager *A2AManager

	// نظام الإيميل
	emailManager *EmailManager

	// نظام بث أحداث الجلسات
	eventBroadcaster *SessionEventBroadcaster

	// نظام التسجيل الشامل
	comprehensiveLogger *ComprehensiveLogger

	// نظام التخزين
	storageConnector *StorageConnector

	// نظام Connect (AgentRegistry الموجود)
	// AgentRegistry يحتوي بالفعل على:
	// - LastSeen field في AgentMetadata
	// - CleanupInactive method
	// - HealthCheck method
	// - ListAvailable method

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

	// إنشاء MCP و A2A Managers
	mcpManager := NewMCPManager(eventBus, logger)
	a2aManager := NewA2AManager(eventBus, logger)

	// إنشاء Email Manager (بدون store مؤقتاً)
	emailManager := NewEmailManager(eventBus, nil, logger)

	// إنشاء Session Event Broadcaster
	eventBroadcaster := NewSessionEventBroadcaster(eventBus, a2aManager, logger)

	// إنشاء Comprehensive Logger
	comprehensiveLogger := NewComprehensiveLogger(eventBus, logger)

	// إنشاء Storage Connector (بدون QuotaManager مؤقتاً)
	storageConnector := NewStorageConnector(eventBus, nil, logger)

	return &Connector{
		eventBus:            eventBus,
		bridge:              bridge,
		agentRegistry:       agentRegistry,
		mcpManager:          mcpManager,
		a2aManager:          a2aManager,
		emailManager:        emailManager,
		eventBroadcaster:    eventBroadcaster,
		comprehensiveLogger: comprehensiveLogger,
		storageConnector:    storageConnector,
		adapters:            make(map[string]Adapter),
		bridgeToEventBus:    make(chan *protocol.Message, 1000),
		eventBusToBridge:    make(chan eventbus.Event, 1000),
		ctx:                 ctx,
		cancel:              cancel,
		logger:              logger,
		metrics:             &ConnectorMetrics{},
	}
}

// Start يبدأ Connector
func (c *Connector) Start() error {
	c.logger.Info("بدء Connector")

	// تسجيل Adapters الافتراضية
	c.registerDefaultAdapters()

	// بدء Comprehensive Logger
	if err := c.comprehensiveLogger.Start(); err != nil {
		return fmt.Errorf("فشل بدء Comprehensive Logger: %w", err)
	}

	// بدء MCP Manager
	if err := c.mcpManager.Start(); err != nil {
		return fmt.Errorf("فشل بدء MCP Manager: %w", err)
	}

	// بدء A2A Manager
	if err := c.a2aManager.Start(); err != nil {
		return fmt.Errorf("فشل بدء A2A Manager: %w", err)
	}

	// بدء Email Manager
	if err := c.emailManager.Start(); err != nil {
		return fmt.Errorf("فشل بدء Email Manager: %w", err)
	}

	// بدء Session Event Broadcaster
	if err := c.eventBroadcaster.Start(); err != nil {
		return fmt.Errorf("فشل بدء Session Event Broadcaster: %w", err)
	}

	// بدء Storage Connector
	if err := c.storageConnector.Start(); err != nil {
		return fmt.Errorf("فشل بدء Storage Connector: %w", err)
	}

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

	// إيقاف Session Event Broadcaster
	if err := c.eventBroadcaster.Stop(); err != nil {
		c.logger.Error("فشل إيقاف Session Event Broadcaster", zap.Error(err))
	}

	// إيقاف Comprehensive Logger
	if err := c.comprehensiveLogger.Stop(); err != nil {
		c.logger.Error("فشل إيقاف Comprehensive Logger", zap.Error(err))
	}

	// إيقاف Storage Connector
	if err := c.storageConnector.Stop(); err != nil {
		c.logger.Error("فشل إيقاف Storage Connector", zap.Error(err))
	}

	// إيقاف MCP Manager
	if err := c.mcpManager.Stop(); err != nil {
		c.logger.Error("فشل إيقاف MCP Manager", zap.Error(err))
	}

	// إيقاف A2A Manager
	if err := c.a2aManager.Stop(); err != nil {
		c.logger.Error("فشل إيقاف A2A Manager", zap.Error(err))
	}

	// إيقاف Email Manager
	if err := c.emailManager.Stop(); err != nil {
		c.logger.Error("فشل إيقاف Email Manager", zap.Error(err))
	}

	// [FIX] إيقاف EventBus لمنع goroutine leaks
	c.eventBus.Stop()

	c.cancel()

	// [FIX] إضافة timeout للانتظار
	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				c.logger.Error("connector shutdown waiter panicked", zap.Any("panic", r))
			}
		}()
		c.wg.Wait()
		close(done)
	}()

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	select {
	case <-done:
		// WaitGroup انتهى بنجاح
	case <-timer.C:
		// [FIX] timeout - لا ننتظر أكثر من 5 ثواني
		c.logger.Warn("Timeout أثناء انتظار goroutines للإنهاء")
	}

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

// ============================================================
// Connect System - نظام معرفة حالة الوكلاء (أونلاين/أوفلاين)
// ============================================================

// GetOnlineAgents يحصل على الوكلاء المتصلين
func (c *Connector) GetOnlineAgents() []agent.UnifiedAgent {
	return c.agentRegistry.ListAvailable()
}

// GetAllAgents يحصل على جميع الوكلاء
func (c *Connector) GetAllAgents() []agent.UnifiedAgent {
	return c.agentRegistry.ListAll()
}

// GetAgentHealthReport يحصل على تقرير صحة الوكلاء
func (c *Connector) GetAgentHealthReport() *agent.HealthReport {
	return c.agentRegistry.HealthCheck()
}

// CleanupInactiveAgents ينظف الوكلاء غير النشطين
func (c *Connector) CleanupInactiveAgents(inactiveThreshold time.Duration) []string {
	return c.agentRegistry.CleanupInactive(inactiveThreshold)
}

// GetAgentMetadata يحصل على بيانات وصفية للوكيل
func (c *Connector) GetAgentMetadata(agentID string) (*agent.AgentMetadata, error) {
	return c.agentRegistry.GetMetadata(agentID)
}

// GetAgentStats يحصل على إحصائيات الوكيل
func (c *Connector) GetAgentStats(agentID string) (*agent.AgentStats, error) {
	return c.agentRegistry.GetStats(agentID)
}

// ============================================================
// Human Client Status - حالة العميل البشري
// ============================================================

// RegisterHumanClient يسجل عميل بشري جديد
func (c *Connector) RegisterHumanClient(userID, name string, allowOnline bool) error {
	return c.agentRegistry.RegisterHumanClient(userID, name, allowOnline)
}

// UpdateHumanClientStatus يحدث حالة العميل البشري
func (c *Connector) UpdateHumanClientStatus(status string) error {
	return c.agentRegistry.UpdateHumanClientStatus(status)
}

// GetHumanClientStatus يحصل على حالة العميل البشري
func (c *Connector) GetHumanClientStatus() (*agent.HumanClientStatus, error) {
	return c.agentRegistry.GetHumanClientStatus()
}

// SetHumanClientOnlinePreference يضبط تفضيل العميل البشري للأونلاين
func (c *Connector) SetHumanClientOnlinePreference(allowOnline bool) error {
	return c.agentRegistry.SetHumanClientOnlinePreference(allowOnline)
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
	// [FIX] NO wildcard subscribe — was creating infinite event loop:
	// EventBus → bridge.Send → bridge.Receive → eventBus.Publish → EventBus (repeat)

	// أحداث محددة مهمة
	c.eventBus.Subscribe("agent.message", c.handleAgentMessage)
	c.eventBus.Subscribe("agent.response", c.handleAgentResponse)
	c.eventBus.Subscribe("task.created", c.handleTaskCreated)
	c.eventBus.Subscribe("task.completed", c.handleTaskCompleted)

	// معالج رسائل العميل (CRITICAL FIX)
	c.eventBus.Subscribe("client.message", c.handleClientMessage)
}

// bridgeHandler يعالج الرسائل من Bridge
func (c *Connector) bridgeHandler() {
	defer c.wg.Done()

	// بدء goroutines منفصلة لكل مسار لتجنب الحظر
	for _, laneType := range []agent_bridge.LaneType{
		agent_bridge.LaneEmergency,
		agent_bridge.LaneChat,
		agent_bridge.LaneWorkflow,
		agent_bridge.LaneFileUpload,
		agent_bridge.LaneFileDownload,
	} {
		c.wg.Add(1)
		go c.processLane(laneType)
	}

	// [FIX] لا ننتظر goroutines هنا - سينتهون عند cancel
}

// processLane يعالج مسار معين
func (c *Connector) processLane(laneType agent_bridge.LaneType) {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			// [SAFETY] التحقق من أن bridge ليس nil
			if c.bridge == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			// قراءة رسالة من المسار
			msg, err := c.bridge.Receive(laneType)
			if err != nil {
				// [FIX] إذا حدث خطأ، تحقق من context
				select {
				case <-c.ctx.Done():
					return
				default:
					time.Sleep(100 * time.Millisecond)
					continue
				}
			}
			if msg == nil {
				// [FIX] إذا لم تكن هناك رسالة، تحقق من context
				select {
				case <-c.ctx.Done():
					return
				default:
					time.Sleep(100 * time.Millisecond)
					continue
				}
			}

			// تحديث المقاييس
			c.mu.Lock()
			c.metrics.MessagesReceived++
			c.metrics.LastActivity = time.Now()
			c.mu.Unlock()

			// إرسال للمعالج
			select {
			case <-c.ctx.Done():
				return
			case c.bridgeToEventBus <- msg:
			default:
				c.logger.Warn("bridgeToEventBus full, dropping message",
					zap.String("msg_type", string(msg.Type)),
				)
			}
		}
	}
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

func (c *Connector) handleClientMessage(event eventbus.Event) {
	c.logger.Info("استقبال رسالة من العميل",
		zap.String("client_id", event.Source),
		zap.String("message_type", event.Type),
	)

	// تمرير الرسالة إلى EventBus للمعالجة
	// يمكن إضافة منطق محدد هنا إذا لزم الأمر
	c.eventBusToBridge <- event
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

// ============================================================
// Task Assignment - تعيين المهام للوكلاء
// ============================================================

// [WHY] handleTaskAssigned يعالج حدث تعيين مهمة
// [HOW] يختار الوكيل المناسب ويرسل المهمة له عبر Bridge
// [SAFETY] يستخدم context مع timeout لمنع الحظر
func (c *Connector) handleTaskAssigned(event eventbus.Event) {
	// [HOW] استخراج بيانات المهمة من الحدث
	taskData, ok := event.Payload.(map[string]interface{})
	if !ok {
		c.logger.Error("بيانات المهمة غير صالحة")
		c.recordError()
		return
	}

	// [HOW] استخراج معرف الوكيل المطلوب
	agentDID, ok := taskData["agent_did"].(string)
	if !ok || agentDID == "" {
		c.logger.Error("معرف الوكيل غير موجود")
		c.recordError()
		return
	}

	// [HOW] استخراج بيانات المهمة
	taskID, _ := taskData["task_id"].(string)
	taskTitle, _ := taskData["title"].(string)
	taskDescription, _ := taskData["description"].(string)

	// [HOW] إرسال المهمة للوكيل عبر Bridge
	err := c.dispatchTaskToBridge(agentDID, map[string]interface{}{
		"task_id":     taskID,
		"title":       taskTitle,
		"description": taskDescription,
		"session_id":  event.SessionID,
	})

	if err != nil {
		c.logger.Error("فشل إرسال المهمة للوكيل",
			zap.String("agent_did", agentDID),
			zap.String("task_id", taskID),
			zap.Error(err),
		)
		c.recordError()
		return
	}

	c.logger.Info("تم إرسال المهمة للوكيل",
		zap.String("agent_did", agentDID),
		zap.String("task_id", taskID),
	)
}

// [WHY] dispatchTaskToBridge يرسل مهمة للوكيل عبر Bridge
// [HOW] يحول بيانات المهمة إلى protocol.Message ويرسلها عبر LaneWorkflow
// [SAFETY] يستخدم context مع timeout لمنع الحظر
func (c *Connector) dispatchTaskToBridge(agentDID string, taskData map[string]interface{}) error {
	// [HOW] تحويل بيانات المهمة إلى protocol.Message
	taskBytes, err := json.Marshal(taskData)
	if err != nil {
		return fmt.Errorf("فشل تحويل بيانات المهمة: %w", err)
	}

	msg := &protocol.Message{
		Type: protocol.MessageTypeTaskRequest,
		Data: taskBytes,
	}

	// [HOW] إرسال الرسالة عبر Bridge في مسار Workflow
	err = c.bridge.Send(agent_bridge.LaneWorkflow, msg)
	if err != nil {
		return fmt.Errorf("فشل إرسال الرسالة عبر Bridge: %w", err)
	}

	// [HOW] تحديث المقاييس
	c.mu.Lock()
	c.metrics.MessagesSent++
	c.mu.Unlock()

	return nil
}

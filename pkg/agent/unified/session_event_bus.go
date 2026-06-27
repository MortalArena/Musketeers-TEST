package unified

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// [SAFETY] حدود الموارد لمنع تسرب الذاكرة
const (
	// [SAFETY] الحد الأقصى لعدد الأحداث في التاريخ
	MaxEventHistory = 10000
	// [SAFETY] حجم قناة الأحداث الرئيسية — كبير بما يكفي لـ 70 وكيل
	EventQueueBufferSize = 10000
	// [SAFETY] حجم قناة كل مشترك
	SubscriberChannelBufferSize = 500
	// [SAFETY] الحد الأقصى لعدد المشتركين (وكلاء + مدير جلسة)
	MaxSubscribers = 100
	// [SAFETY] إجمالي سعة جميع قنوات المشتركين — يمنع استنزاف الذاكرة
	MaxTotalSubscriberBuffer = MaxSubscribers * SubscriberChannelBufferSize
)

// SessionEventBus ناقل أحداث الجلسة لمزامنة لحظية
type SessionEventBus struct {
	sessionID string
	logger    *zap.Logger
	mu        sync.RWMutex

	// قنوات الأحداث
	eventQueue       chan *SessionEvent
	agentSubscribers map[string]chan *SessionEvent
	sessionManager   chan *SessionEvent

	// حالة الأحداث
	eventHistory  []*SessionEvent
	active        atomic.Bool
	started       atomic.Bool // [SAFETY] يمنع Start() المتعدد
	stopOnce      sync.Once
	stopCh        chan struct{} // [SAFETY] قناة إيقاف منفصلة عن context
	lastEventTime time.Time
	totalEvents   int

	// [SAFETY] عداد الغوروتينات — يمنع التسرب
	processWG sync.WaitGroup
}

// SessionEvent حدث في الجلسة
type SessionEvent struct {
	ID          string
	SessionID   string
	SourceAgent string
	TargetAgent string // فارغ يعني جميع الوكلاء
	EventType   SessionEventType
	Timestamp   time.Time
	Priority    EventPriority
	Data        interface{}
	Metadata    map[string]interface{}
}

// SessionEventType نوع حدث الجلسة
type SessionEventType string

const (
	// أحداث المهام
	TaskStarted   SessionEventType = "task_started"
	TaskProgress  SessionEventType = "task_progress"
	TaskCompleted SessionEventType = "task_completed"
	TaskFailed    SessionEventType = "task_failed"
	TaskAssigned  SessionEventType = "task_assigned"

	// أحداث الذاكرة
	MemoryUpdated  SessionEventType = "memory_updated"
	MemoryAccessed SessionEventType = "memory_accessed"
	MemoryCreated  SessionEventType = "memory_created"

	// أحداث المهارات
	SkillLearned  SessionEventType = "skill_learned"
	SkillImproved SessionEventType = "skill_improved"
	SkillUsed     SessionEventType = "skill_used"

	// أحداث التواصل
	AgentMessage       SessionEventType = "agent_message"
	AgentStatus        SessionEventType = "agent_status"
	SessionStatusEvent SessionEventType = "session_status"

	// أحداث النظام
	SystemAlert SessionEventType = "system_alert"
	SystemError SessionEventType = "system_error"
)

// EventPriority أولوية الحدث
type EventPriority string

const (
	PriorityLow      EventPriority = "low"
	PriorityMedium   EventPriority = "medium"
	PriorityHigh     EventPriority = "high"
	PriorityCritical EventPriority = "critical"
)

// NewSessionEventBus ينشئ ناقل أحداث جلسة جديد
func NewSessionEventBus(sessionID string, logger *zap.Logger) *SessionEventBus {
	seb := &SessionEventBus{
		sessionID:        sessionID,
		logger:           logger,
		eventQueue:       make(chan *SessionEvent, EventQueueBufferSize),
		agentSubscribers: make(map[string]chan *SessionEvent),
		sessionManager:   make(chan *SessionEvent, EventQueueBufferSize),
		eventHistory:     make([]*SessionEvent, 0, MaxEventHistory),
		stopCh:           make(chan struct{}),
		lastEventTime:    time.Now(),
	}
	seb.active.Store(true)
	return seb
}

// Start يبدأ ناقل الأحداث — آمن ضد الاستدعاء المتعدد
func (seb *SessionEventBus) Start(ctx context.Context) {
	if !seb.started.CompareAndSwap(false, true) {
		seb.logger.Warn("SessionEventBus: Start() استدعاء متكرر — تم التجاهل",
			zap.String("session_id", seb.sessionID))
		return
	}

	seb.active.Store(true)
	seb.processWG.Add(1)
	go seb.processEvents(ctx)
}

// Stop يوقف ناقل الأحداث — آمن ضد الاستدعاء المتعدد
func (seb *SessionEventBus) Stop() {
	seb.stopOnce.Do(func() {
		seb.active.Store(false)

		// [SAFETY] إشارة إيقاف للـ processEvents
		close(seb.stopCh)

		// [SAFETY] القفل لضمان عدم وجود كتابات نشطة أثناء الإغلاق
		seb.mu.Lock()
		close(seb.eventQueue)
		close(seb.sessionManager)

		// [SAFETY] إغلاق جميع قنوات الوكلاء بأمان
		for id, ch := range seb.agentSubscribers {
			close(ch)
			delete(seb.agentSubscribers, id)
		}
		seb.mu.Unlock()

		// [SAFETY] انتظار إنهاء الـ goroutine
		seb.processWG.Wait()

		seb.logger.Debug("SessionEventBus: تم إيقاف ناقل الأحداث بنجاح",
			zap.String("session_id", seb.sessionID))
	})
}

// processEvents يعالج الأحداث — يراقب context و stopCh معاً
func (seb *SessionEventBus) processEvents(ctx context.Context) {
	defer seb.processWG.Done()

	for {
		select {
		case <-ctx.Done():
			seb.logger.Info("تم إيقاف معالجة أحداث الجلسة (context)",
				zap.String("session_id", seb.sessionID))
			return
		case <-seb.stopCh:
			seb.logger.Info("تم إيقاف معالجة أحداث الجلسة (stop signal)",
				zap.String("session_id", seb.sessionID))
			return
		case event, ok := <-seb.eventQueue:
			if !ok {
				seb.logger.Info("تم إغلاق قناة أحداث الجلسة",
					zap.String("session_id", seb.sessionID))
				return
			}
			seb.distributeEvent(event)
		}
	}
}

// distributeEvent يوزع الحدث على المشتركين
func (seb *SessionEventBus) distributeEvent(event *SessionEvent) {
	seb.mu.Lock()
	defer seb.mu.Unlock()

	// [SAFETY] قائمة المتسربين — ستنظف بعد التوزيع
	var staleAgents []string

	// [SAFETY] إضافة إلى التاريخ مع حد أقصى — استخدام ring buffer (ثابت الحجم)
	if len(seb.eventHistory) >= MaxEventHistory {
		// [FIX] إزاحة النصف الأقدم ومسح المراجع لتجنب تسرب الذاكرة
		trimIdx := MaxEventHistory / 2
		// مسح المراجع القديمة لـ GC
		for i := 0; i < trimIdx; i++ {
			seb.eventHistory[i] = nil
		}
		seb.eventHistory = append(seb.eventHistory[trimIdx:], event)
	} else {
		seb.eventHistory = append(seb.eventHistory, event)
	}
	seb.lastEventTime = event.Timestamp
	seb.totalEvents++

	// إرسال لمدير الجلسة دائماً — مع حماية الإرسال لقناة مغلقة
	seb.sendNonBlocking(seb.sessionManager, event, "مدير الجلسة")

	// إرسال للوكيل المستهدف أو جميع الوكلاء
	if event.TargetAgent == "" {
		// إرسال لجميع الوكلاء — مع اكتشاف القنوات المغلقة
		for agentID, ch := range seb.agentSubscribers {
			if !seb.sendNonBlocking(ch, event, agentID) {
				staleAgents = append(staleAgents, agentID)
			}
		}
	} else {
		// إرسال للوكيل المستهدف
		if ch, exists := seb.agentSubscribers[event.TargetAgent]; exists {
			if !seb.sendNonBlocking(ch, event, event.TargetAgent) {
				staleAgents = append(staleAgents, event.TargetAgent)
			}
		}
	}

	// [SAFETY] تنظيف القنوات المغلقة (المشتركين الذين تم فصلهم)
	for _, agentID := range staleAgents {
		delete(seb.agentSubscribers, agentID)
	}

	seb.logger.Debug("تم توزيع الحدث",
		zap.String("session_id", seb.sessionID),
		zap.String("event_id", event.ID),
		zap.String("event_type", string(event.EventType)),
		zap.String("source_agent", event.SourceAgent),
		zap.String("target_agent", event.TargetAgent))
}

// sendNonBlocking يرسل حدث بشكل غير blocking مع حماية من القنوات المغلقة
// [SAFETY] يعيد false إذا كانت القناة مغلقة
func (seb *SessionEventBus) sendNonBlocking(ch chan *SessionEvent, event *SessionEvent, label string) (ok bool) {
	// [SAFETY] recover من panic الإرسال لقناة مغلقة
	defer func() {
		if r := recover(); r != nil {
			seb.logger.Warn("إرسال لقناة مغلقة — سيتم تنظيفها",
				zap.String("target", label))
			ok = false
		}
	}()

	select {
	case ch <- event:
		return true
	default:
		seb.logger.Warn("قناة ممتلئة", zap.String("target", label))
		return true
	}
}

// PublishEvent ينشر حدث
// [SAFETY] القفل المحتفظ به عبر التحقق والإرسال يمنع TOCTOU مع Stop()
func (seb *SessionEventBus) PublishEvent(ctx context.Context, event *SessionEvent) error {
	seb.mu.RLock()
	if !seb.active.Load() {
		seb.mu.RUnlock()
		return nil
	}

	// إرسال الحدث — القفل لا يزال محتفظًا به لمنع Stop() من إغلاق القناة
	select {
	case seb.eventQueue <- event:
		seb.mu.RUnlock()
		seb.logger.Info("تم نشر الحدث",
			zap.String("session_id", seb.sessionID),
			zap.String("event_id", event.ID),
			zap.String("event_type", string(event.EventType)),
			zap.String("source_agent", event.SourceAgent))
		return nil
	default:
		seb.mu.RUnlock()
		seb.logger.Warn("قناة الأحداث ممتلئة",
			zap.String("session_id", seb.sessionID),
			zap.String("event_id", event.ID))
		return nil
	}
}

// SubscribeAgent يربط وكيل بناقل الأحداث
func (seb *SessionEventBus) SubscribeAgent(agentID string) chan *SessionEvent {
	seb.mu.Lock()
	defer seb.mu.Unlock()

	// [SAFETY] فرض الحد الأقصى للمشتركين
	if len(seb.agentSubscribers) >= MaxSubscribers {
		seb.logger.Warn("تجاوز الحد الأقصى للمشتركين — رفض الاشتراك",
			zap.String("agent_id", agentID),
			zap.Int("max", MaxSubscribers))
		return nil
	}

	// [SAFETY] استبدال القناة القديمة إن وجدت (منع تسرب القناة السابقة)
	if oldCh, exists := seb.agentSubscribers[agentID]; exists {
		close(oldCh)
	}

	ch := make(chan *SessionEvent, SubscriberChannelBufferSize)
	seb.agentSubscribers[agentID] = ch

	seb.logger.Info("تم اشتراك الوكيل في ناقل الأحداث",
		zap.String("session_id", seb.sessionID),
		zap.String("agent_id", agentID),
		zap.Int("total_subscribers", len(seb.agentSubscribers)))

	return ch
}

// UnsubscribeAgent يفصل وكيل من ناقل الأحداث
// [SAFETY] آمن ضد الاستدعاء المزدوج — no-op بعد الفصل الأول
// [FIX] إغلاق القناة فقط إذا كانت مفتوحة — يمنع double-close panic
func (seb *SessionEventBus) UnsubscribeAgent(agentID string) {
	seb.mu.Lock()
	if ch, exists := seb.agentSubscribers[agentID]; exists {
		// [SAFETY] الإزالة من الخريطة أولاً — يمنع أي إرسال جديد للقناة
		delete(seb.agentSubscribers, agentID)
		// [SAFETY] إغلاق القناة بعد الإزالة — المتلقون سيحصلون على zero value
		close(ch)
		seb.mu.Unlock()
		seb.logger.Info("تم فصل الوكيل من ناقل الأحداث",
			zap.String("session_id", seb.sessionID),
			zap.String("agent_id", agentID))
	} else {
		seb.mu.Unlock()
	}
}

// GetSessionManagerChannel يحصل على قناة مدير الجلسة
func (seb *SessionEventBus) GetSessionManagerChannel() chan *SessionEvent {
	return seb.sessionManager
}

// GetAgentChannel يحصل على قناة وكيل — يحذر من أن القناة قد تُغلق
// [SAFETY] المتلقي يجب أن يستخدم recover() أو select مع default للقراءة من القناة
func (seb *SessionEventBus) GetAgentChannel(agentID string) (chan *SessionEvent, bool) {
	seb.mu.RLock()
	defer seb.mu.RUnlock()

	ch, exists := seb.agentSubscribers[agentID]
	return ch, exists
}

// GetEventHistory يحصل على تاريخ الأحداث
func (seb *SessionEventBus) GetEventHistory(limit int) []*SessionEvent {
	seb.mu.RLock()
	defer seb.mu.RUnlock()

	if limit <= 0 || limit > len(seb.eventHistory) {
		limit = len(seb.eventHistory)
	}

	start := len(seb.eventHistory) - limit
	if start < 0 {
		start = 0
	}

	return seb.eventHistory[start:]
}

// GetRecentEventsForAgent يحصل على الأحداث الأخيرة لوكيل معين
func (seb *SessionEventBus) GetRecentEventsForAgent(agentID string, limit int) []*SessionEvent {
	seb.mu.RLock()
	defer seb.mu.RUnlock()

	var events []*SessionEvent
	for i := len(seb.eventHistory) - 1; i >= 0 && len(events) < limit; i-- {
		event := seb.eventHistory[i]
		// أحداث مرتبطة بالوكيل (مصدر أو مستهدف)
		if event.SourceAgent == agentID || event.TargetAgent == agentID || event.TargetAgent == "" {
			events = append(events, event)
		}
	}

	return events
}

// GetStatus يحصل على حالة ناقل الأحداث
func (seb *SessionEventBus) GetStatus() map[string]interface{} {
	seb.mu.RLock()
	defer seb.mu.RUnlock()

	return map[string]interface{}{
		"active":         seb.active.Load(),
		"last_event":     seb.lastEventTime,
		"total_events":   seb.totalEvents,
		"subscribers":    len(seb.agentSubscribers),
		"pending_events": len(seb.eventQueue),
		"history_size":   len(seb.eventHistory),
	}
}

// BroadcastToAll يرسل حدث لجميع الوكلاء
func (seb *SessionEventBus) BroadcastToAll(ctx context.Context, sourceAgent string, eventType SessionEventType, data interface{}) error {
	event := &SessionEvent{
		ID:          generateID(),
		SessionID:   seb.sessionID,
		SourceAgent: sourceAgent,
		TargetAgent: "", // فارغ يعني جميع الوكلاء
		EventType:   eventType,
		Timestamp:   time.Now(),
		Priority:    PriorityMedium,
		Data:        data,
		Metadata:    make(map[string]interface{}),
	}

	return seb.PublishEvent(ctx, event)
}

// SendToAgent يرسل حدث لوكيل محدد
func (seb *SessionEventBus) SendToAgent(ctx context.Context, sourceAgent, targetAgent string, eventType SessionEventType, data interface{}) error {
	event := &SessionEvent{
		ID:          generateID(),
		SessionID:   seb.sessionID,
		SourceAgent: sourceAgent,
		TargetAgent: targetAgent,
		EventType:   eventType,
		Timestamp:   time.Now(),
		Priority:    PriorityHigh,
		Data:        data,
		Metadata:    make(map[string]interface{}),
	}

	return seb.PublishEvent(ctx, event)
}

// SendToSessionManager يرسل حدث لمدير الجلسة
func (seb *SessionEventBus) SendToSessionManager(ctx context.Context, sourceAgent string, eventType SessionEventType, data interface{}) error {
	event := &SessionEvent{
		ID:          generateID(),
		SessionID:   seb.sessionID,
		SourceAgent: sourceAgent,
		TargetAgent: "session_manager",
		EventType:   eventType,
		Timestamp:   time.Now(),
		Priority:    PriorityHigh,
		Data:        data,
		Metadata:    make(map[string]interface{}),
	}

	return seb.PublishEvent(ctx, event)
}

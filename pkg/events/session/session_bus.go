package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
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
	active        bool
	lastEventTime time.Time
	totalEvents   int
}

// SessionEvent حدث في الجلسة
type SessionEvent struct {
	ID          string
	SessionID   string
	SourceAgent string
	TargetAgent string
	EventType   SessionEventType
	Timestamp   time.Time
	Priority    EventPriority
	Data        interface{}
	Metadata    map[string]interface{}
}

// SessionEventType نوع حدث الجلسة
type SessionEventType string

const (
	TaskStarted   SessionEventType = "task_started"
	TaskProgress  SessionEventType = "task_progress"
	TaskCompleted SessionEventType = "task_completed"
	TaskFailed    SessionEventType = "task_failed"
	MemoryUpdated SessionEventType = "memory_updated"
	SkillLearned  SessionEventType = "skill_learned"
	AgentMessage  SessionEventType = "agent_message"
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
	return &SessionEventBus{
		sessionID:        sessionID,
		logger:           logger,
		eventQueue:       make(chan *SessionEvent, 1000),
		agentSubscribers: make(map[string]chan *SessionEvent),
		sessionManager:   make(chan *SessionEvent, 1000),
		eventHistory:     []*SessionEvent{},
		active:           true,
		lastEventTime:    time.Now(),
		totalEvents:      0,
	}
}

// Start يبدأ ناقل الأحداث
func (seb *SessionEventBus) Start(ctx context.Context) {
	seb.mu.Lock()
	seb.active = true
	seb.mu.Unlock()

	go seb.processEvents(ctx)
}

// Stop يوقف ناقل الأحداث
func (seb *SessionEventBus) Stop() {
	seb.mu.Lock()
	defer seb.mu.Unlock()

	seb.active = false
	close(seb.eventQueue)
	close(seb.sessionManager)

	for _, ch := range seb.agentSubscribers {
		close(ch)
	}
}

// processEvents يعالج الأحداث
func (seb *SessionEventBus) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			seb.logger.Info("تم إيقاف معالجة أحداث الجلسة")
			return
		case event, ok := <-seb.eventQueue:
			if !ok {
				seb.logger.Info("تم إغلاق قناة أحداث الجلسة")
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

	seb.eventHistory = append(seb.eventHistory, event)
	seb.lastEventTime = event.Timestamp
	seb.totalEvents++

	select {
	case seb.sessionManager <- event:
	default:
		seb.logger.Warn("قناة مدير الجلسة ممتلئة")
	}

	if event.TargetAgent == "" {
		for agentID, ch := range seb.agentSubscribers {
			select {
			case ch <- event:
			default:
				seb.logger.Warn("قناة الوكيل ممتلئة", zap.String("agent_id", agentID))
			}
		}
	} else {
		if ch, exists := seb.agentSubscribers[event.TargetAgent]; exists {
			select {
			case ch <- event:
			default:
				seb.logger.Warn("قناة الوكيل المستهدف ممتلئة", zap.String("agent_id", event.TargetAgent))
			}
		}
	}
}

// PublishEvent ينشر حدث
func (seb *SessionEventBus) PublishEvent(ctx context.Context, event *SessionEvent) error {
	seb.mu.RLock()
	defer seb.mu.RUnlock()

	if !seb.active {
		return nil
	}

	select {
	case seb.eventQueue <- event:
		return nil
	default:
		seb.logger.Warn("قناة الأحداث ممتلئة")
		return nil
	}
}

// SubscribeAgent يربط وكيل بناقل الأحداث
func (seb *SessionEventBus) SubscribeAgent(agentID string) chan *SessionEvent {
	seb.mu.Lock()
	defer seb.mu.Unlock()

	ch := make(chan *SessionEvent, 100)
	seb.agentSubscribers[agentID] = ch

	return ch
}

// GetStatus يحصل على حالة ناقل الأحداث
func (seb *SessionEventBus) GetStatus() map[string]interface{} {
	seb.mu.RLock()
	defer seb.mu.RUnlock()

	return map[string]interface{}{
		"active":         seb.active,
		"last_event":     seb.lastEventTime,
		"total_events":   seb.totalEvents,
		"subscribers":    len(seb.agentSubscribers),
		"pending_events": len(seb.eventQueue),
		"history_size":   len(seb.eventHistory),
	}
}

// generateID يولد معرف فريد
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

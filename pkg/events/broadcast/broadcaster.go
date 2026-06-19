package broadcast

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// EventBroadcaster نظام بث الأحداث
type EventBroadcaster struct {
	sessionID string
	logger    *zap.Logger
	mu        sync.RWMutex

	// قنوات البث
	broadcastChannels map[string]chan *SessionEvent
	sessionManager   chan *SessionEvent

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics
	metrics *BroadcasterMetrics
}

// BroadcasterMetrics مقاييس البث
type BroadcasterMetrics struct {
	EventsBroadcasted int64
	AgentsNotified    int64
	SessionsActive    int
	Errors            int64
	LastActivity      time.Time
}

// SessionEvent حدث جلسة
type SessionEvent struct {
	ID          string                 `json:"id"`
	SessionID   string                 `json:"session_id"`
	Type        string                 `json:"type"`
	AgentID     string                 `json:"agent_id"`
	Description string                 `json:"description"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
	Priority    string                 `json:"priority"`
}

// NewEventBroadcaster ينشئ نظام بث أحداث جديد
func NewEventBroadcaster(sessionID string, logger *zap.Logger) *EventBroadcaster {
	ctx, cancel := context.WithCancel(context.Background())

	return &EventBroadcaster{
		sessionID:        sessionID,
		logger:           logger,
		broadcastChannels: make(map[string]chan *SessionEvent),
		sessionManager:   make(chan *SessionEvent, 1000),
		ctx:              ctx,
		cancel:           cancel,
		metrics:          &BroadcasterMetrics{},
	}
}

// Start يبدأ نظام البث
func (eb *EventBroadcaster) Start() error {
	eb.logger.Info("بدء نظام بث الأحداث")

	eb.wg.Add(1)
	go eb.broadcastHandler()

	eb.logger.Info("تم بدء نظام بث الأحداث بنجاح")
	return nil
}

// Stop يوقف نظام البث
func (eb *EventBroadcaster) Stop() error {
	eb.logger.Info("إيقاف نظام بث الأحداث")

	eb.cancel()
	eb.wg.Wait()

	eb.mu.Lock()
	for _, ch := range eb.broadcastChannels {
		close(ch)
	}
	eb.broadcastChannels = make(map[string]chan *SessionEvent)
	eb.mu.Unlock()

	eb.logger.Info("تم إيقاف نظام بث الأحداث بنجاح")
	return nil
}

// BroadcastEvent يبث حدث لجميع الوكلاء في جلسة
func (eb *EventBroadcaster) BroadcastEvent(event *SessionEvent) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	// بث الحدث لجميع المشتركين
	for agentID, ch := range eb.broadcastChannels {
		select {
		case ch <- event:
			eb.metrics.AgentsNotified++
		default:
			eb.logger.Warn("قناة الوكيل ممتلئة", zap.String("agent_id", agentID))
			eb.metrics.Errors++
		}
	}

	eb.metrics.EventsBroadcasted++
	eb.metrics.LastActivity = time.Now()

	return nil
}

// BroadcastTaskAssigned يبث حدث توزيع مهمة
func (eb *EventBroadcaster) BroadcastTaskAssigned(sessionID, agentID, task string, context map[string]interface{}) error {
	event := &SessionEvent{
		ID:          generateID(),
		SessionID:   sessionID,
		Type:        "task_assigned",
		AgentID:     agentID,
		Description: fmt.Sprintf("تم توزيع مهمة '%s' على %s", task, agentID),
		Data: map[string]interface{}{
			"task":    task,
			"context": context,
		},
		Timestamp: time.Now(),
		Priority:  "normal",
	}

	return eb.BroadcastEvent(event)
}

// BroadcastTaskCompleted يبث حدث إكمال مهمة
func (eb *EventBroadcaster) BroadcastTaskCompleted(sessionID, agentID string) error {
	event := &SessionEvent{
		ID:          generateID(),
		SessionID:   sessionID,
		Type:        "task_completed",
		AgentID:     agentID,
		Description: fmt.Sprintf("أكمل %s مهمته", agentID),
		Data:        map[string]interface{}{},
		Timestamp:   time.Now(),
		Priority:    "normal",
	}

	return eb.BroadcastEvent(event)
}

// broadcastHandler يعالج البث
func (eb *EventBroadcaster) broadcastHandler() {
	defer eb.wg.Done()

	for {
		select {
		case <-eb.ctx.Done():
			return
		}
	}
}

// GetMetrics يحصل على المقاييس
func (eb *EventBroadcaster) GetMetrics() *BroadcasterMetrics {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	return &BroadcasterMetrics{
		EventsBroadcasted: eb.metrics.EventsBroadcasted,
		AgentsNotified:    eb.metrics.AgentsNotified,
		SessionsActive:    eb.metrics.SessionsActive,
		Errors:            eb.metrics.Errors,
		LastActivity:      eb.metrics.LastActivity,
	}
}

// generateID يولد معرف فريد
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

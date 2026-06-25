package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

// ============================================================
// SessionEventBroadcaster - نظام بث أحداث الجلسات
// ============================================================

// SessionEventBroadcaster يبث أحداث الجلسات لجميع الوكلاء لمنع "العمى"
type SessionEventBroadcaster struct {
	// المكونات الأساسية
	eventBus *eventbus.EventBus
	a2aMgr   *A2AManager

	// قنوات البث
	broadcastChannels map[string]chan *SessionEvent
	mu               sync.RWMutex

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Logger
	logger *zap.Logger

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
	Type        string                 `json:"type"` // task_assigned, task_completed, artifact_shared, status_update, error, progress
	AgentID     string                 `json:"agent_id"`
	Description string                 `json:"description"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
	Priority    string                 `json:"priority"` // low, normal, high, urgent
}

// NewSessionEventBroadcaster ينشئ SessionEventBroadcaster جديد
func NewSessionEventBroadcaster(eventBus *eventbus.EventBus, a2aMgr *A2AManager, logger *zap.Logger) *SessionEventBroadcaster {
	ctx, cancel := context.WithCancel(context.Background())

	return &SessionEventBroadcaster{
		eventBus:          eventBus,
		a2aMgr:            a2aMgr,
		broadcastChannels: make(map[string]chan *SessionEvent),
		ctx:               ctx,
		cancel:            cancel,
		logger:            logger,
		metrics:           &BroadcasterMetrics{},
	}
}

// Start يبدأ SessionEventBroadcaster
func (seb *SessionEventBroadcaster) Start() error {
	seb.logger.Info("بدء SessionEventBroadcaster")

	// الاشتراك في أحداث Event Bus
	seb.subscribeToEventBus()

	// بدء معالج البث
	seb.wg.Add(1)
	go seb.broadcastHandler()

	seb.logger.Info("تم بدء SessionEventBroadcaster بنجاح")
	return nil
}

// Stop يوقف SessionEventBroadcaster
func (seb *SessionEventBroadcaster) Stop() error {
	seb.logger.Info("إيقاف SessionEventBroadcaster")

	seb.cancel()
	seb.wg.Wait()

	// إغلاق جميع قنوات البث
	seb.mu.Lock()
	for _, ch := range seb.broadcastChannels {
		close(ch)
	}
	seb.broadcastChannels = make(map[string]chan *SessionEvent)
	seb.mu.Unlock()

	seb.logger.Info("تم إيقاف SessionEventBroadcaster بنجاح")
	return nil
}

// ============================================================
// بث الأحداث
// ============================================================

// BroadcastEvent يبث حدث لجميع الوكلاء في جلسة
func (seb *SessionEventBroadcaster) BroadcastEvent(event *SessionEvent) error {
	session, err := seb.a2aMgr.GetSession(event.SessionID)
	if err != nil {
		return fmt.Errorf("الجلسة %s غير موجودة", event.SessionID)
	}

	// إضافة الحدث إلى سجل الجلسة
	sessionEvent := &A2AEvent{
		ID:          generateChatID(),
		Type:        event.Type,
		AgentID:     event.AgentID,
		Description: event.Description,
		Data:        event.Data,
		Timestamp:   event.Timestamp,
	}
	session.Events = append(session.Events, sessionEvent)
	session.UpdatedAt = time.Now()

	// بث الحدث لجميع المشاركين في الجلسة
	for _, participantID := range session.Participants {
		// إرسال الحدث عبر A2A
		msg := &A2AMessage{
			MessageID: generateChatID(),
			SessionID: event.SessionID,
			Sender:    "system",
			Receiver:  participantID,
			Type:      "session_event",
			Context: map[string]interface{}{
				"event_type":    event.Type,
				"event_id":      event.ID,
				"description":   event.Description,
				"data":          event.Data,
				"priority":      event.Priority,
			},
			Timestamp: time.Now(),
		}

		if err := seb.a2aMgr.SendMessage(msg); err != nil {
			seb.logger.Error("فشل إرسال حدث جلسة",
				zap.String("agent_id", participantID),
				zap.String("session_id", event.SessionID),
				zap.Error(err),
			)
			seb.metrics.Errors++
		} else {
			seb.metrics.AgentsNotified++
		}
	}

	seb.metrics.EventsBroadcasted++
	seb.metrics.LastActivity = time.Now()

	seb.logger.Debug("تم بث حدث جلسة",
		zap.String("session_id", event.SessionID),
		zap.String("event_type", event.Type),
		zap.Int("participants_count", len(session.Participants)),
	)

	return nil
}

// ============================================================
// أنواع الأحداث المحددة
// ============================================================

// BroadcastTaskAssigned يبث حدث توزيع مهمة
func (seb *SessionEventBroadcaster) BroadcastTaskAssigned(sessionID, agentID, task string, context map[string]interface{}) error {
	event := &SessionEvent{
		ID:          generateChatID(),
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

	return seb.BroadcastEvent(event)
}

// BroadcastTaskCompleted يبث حدث إكمال مهمة
func (seb *SessionEventBroadcaster) BroadcastTaskCompleted(sessionID, agentID string, artifacts []*A2AArtifact) error {
	event := &SessionEvent{
		ID:          generateChatID(),
		SessionID:   sessionID,
		Type:        "task_completed",
		AgentID:     agentID,
		Description: fmt.Sprintf("أكمل %s مهمته", agentID),
		Data: map[string]interface{}{
			"artifacts_count": len(artifacts),
			"artifacts":      artifacts,
		},
		Timestamp: time.Now(),
		Priority:  "normal",
	}

	return seb.BroadcastEvent(event)
}

// BroadcastArtifactShared يبث حدث مشاركة artifact
func (seb *SessionEventBroadcaster) BroadcastArtifactShared(sessionID, agentID string, artifact *A2AArtifact) error {
	event := &SessionEvent{
		ID:          generateChatID(),
		SessionID:   sessionID,
		Type:        "artifact_shared",
		AgentID:     agentID,
		Description: fmt.Sprintf("شارك %s artifact: %s", agentID, artifact.Name),
		Data: map[string]interface{}{
			"artifact": artifact,
		},
		Timestamp: time.Now(),
		Priority:  "normal",
	}

	return seb.BroadcastEvent(event)
}

// BroadcastProgressUpdate يبث تحديث تقدم
func (seb *SessionEventBroadcaster) BroadcastProgressUpdate(sessionID, agentID string, progress int, message string) error {
	event := &SessionEvent{
		ID:          generateChatID(),
		SessionID:   sessionID,
		Type:        "progress_update",
		AgentID:     agentID,
		Description: fmt.Sprintf("%s: %d%% - %s", agentID, progress, message),
		Data: map[string]interface{}{
			"progress": progress,
			"message":  message,
		},
		Timestamp: time.Now(),
		Priority:  "normal",
	}

	return seb.BroadcastEvent(event)
}

// BroadcastError يبث حدث خطأ
func (seb *SessionEventBroadcaster) BroadcastError(sessionID, agentID string, errorMsg string, context map[string]interface{}) error {
	event := &SessionEvent{
		ID:          generateChatID(),
		SessionID:   sessionID,
		Type:        "error",
		AgentID:     agentID,
		Description: fmt.Sprintf("خطأ من %s: %s", agentID, errorMsg),
		Data: map[string]interface{}{
			"error":   errorMsg,
			"context": context,
		},
		Timestamp: time.Now(),
		Priority:  "urgent",
	}

	return seb.BroadcastEvent(event)
}

// BroadcastStatusUpdate يبث تحديث حالة
func (seb *SessionEventBroadcaster) BroadcastStatusUpdate(sessionID, agentID, status string) error {
	event := &SessionEvent{
		ID:          generateChatID(),
		SessionID:   sessionID,
		Type:        "status_update",
		AgentID:     agentID,
		Description: fmt.Sprintf("تحديث حالة %s: %s", agentID, status),
		Data: map[string]interface{}{
			"status": status,
		},
		Timestamp: time.Now(),
		Priority:  "normal",
	}

	return seb.BroadcastEvent(event)
}

// ============================================================
// معالجة الرسائل
// ============================================================

// subscribeToEventBus يرتبط بأحداث Event Bus
func (seb *SessionEventBroadcaster) subscribeToEventBus() {
	seb.eventBus.Subscribe("session.event", seb.handleSessionEvent)
	seb.eventBus.Subscribe("agent.status", seb.handleAgentStatus)
	seb.eventBus.Subscribe("task.assigned", seb.handleTaskAssigned)
	seb.eventBus.Subscribe("task.completed", seb.handleTaskCompleted)
}

// broadcastHandler يعالج البث عبر قنوات A2A
func (seb *SessionEventBroadcaster) broadcastHandler() {
	defer seb.wg.Done()

	for {
		select {
		case <-seb.ctx.Done():
			return
		default:
			seb.mu.RLock()
			channels := make([]chan *SessionEvent, 0, len(seb.broadcastChannels))
			for _, ch := range seb.broadcastChannels {
				channels = append(channels, ch)
			}
			seb.mu.RUnlock()

		for _, ch := range channels {
			select {
			case event := <-ch:
				if seb.a2aMgr != nil && event != nil {
					msg := &A2AMessage{
						MessageID: generateChatID(),
						SessionID: event.SessionID,
						Type:      "session.event",
						Context:   event.Data,
					}
					seb.a2aMgr.SendMessage(msg)
				}
			default:
			}
			}
		}
	}
}

// handleSessionEvent يعالج حدث جلسة
func (seb *SessionEventBroadcaster) handleSessionEvent(event eventbus.Event) {
	seb.logger.Debug("استقبال حدث جلسة",
		zap.String("session_id", event.SessionID),
	)
}

// handleAgentStatus يعالج حالة وكيل
func (seb *SessionEventBroadcaster) handleAgentStatus(event eventbus.Event) {
	seb.logger.Debug("استقبال حالة وكيل",
		zap.String("agent_id", event.Source),
	)
}

// handleTaskAssigned يعالج توزيع مهمة
func (seb *SessionEventBroadcaster) handleTaskAssigned(event eventbus.Event) {
	seb.logger.Debug("استقبال توزيع مهمة")
}

// handleTaskCompleted يعالج إكمال مهمة
func (seb *SessionEventBroadcaster) handleTaskCompleted(event eventbus.Event) {
	seb.logger.Debug("استقبال إكمال مهمة")
}

// ============================================================
// المقاييس
// ============================================================

// GetMetrics يحصل على المقاييس
func (seb *SessionEventBroadcaster) GetMetrics() *BroadcasterMetrics {
	seb.mu.RLock()
	defer seb.mu.RUnlock()

	return &BroadcasterMetrics{
		EventsBroadcasted: seb.metrics.EventsBroadcasted,
		AgentsNotified:    seb.metrics.AgentsNotified,
		SessionsActive:    seb.metrics.SessionsActive,
		Errors:            seb.metrics.Errors,
		LastActivity:      seb.metrics.LastActivity,
	}
}

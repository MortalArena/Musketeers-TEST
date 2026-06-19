package sync

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// RealTimeMemorySync مزامنة الذاكرة بشكل لحظي
type RealTimeMemorySync struct {
	sessionID string
	logger    *zap.Logger
	mu        sync.RWMutex

	// قناة المزامنة
	memoryEvents chan *RealTimeMemoryEvent
	agentStates  map[string]*AgentMemoryState

	// حالة المزامنة
	syncActive bool
	lastSync   time.Time
}

// RealTimeMemoryEvent حدث ذاكرة لحظي
type RealTimeMemoryEvent struct {
	ID         string
	SessionID  string
	AgentID    string
	EventType  MemoryEventType
	Timestamp  time.Time
	MemoryType string
	Content    interface{}
	Metadata   map[string]interface{}
}

// MemoryEventType نوع حدث الذاكرة
type MemoryEventType string

const (
	MemoryEventCreated  MemoryEventType = "created"
	MemoryEventUpdated  MemoryEventType = "updated"
	MemoryEventDeleted  MemoryEventType = "deleted"
	MemoryEventAccessed MemoryEventType = "accessed"
)

// AgentMemoryState حالة ذاكرة الوكيل
type AgentMemoryState struct {
	AgentID        string
	LastSync       time.Time
	MemoryCount    int
	ActiveMemories map[string]interface{}
}

// NewRealTimeMemorySync ينشئ نظام مزامنة ذاكرة جديد
func NewRealTimeMemorySync(sessionID string, logger *zap.Logger) *RealTimeMemorySync {
	return &RealTimeMemorySync{
		sessionID:    sessionID,
		logger:       logger,
		memoryEvents: make(chan *RealTimeMemoryEvent, 100),
		agentStates:  make(map[string]*AgentMemoryState),
		syncActive:   true,
		lastSync:     time.Now(),
	}
}

// StartSync يبدأ مزامنة الذاكرة
func (rtms *RealTimeMemorySync) StartSync(ctx context.Context) {
	rtms.mu.Lock()
	rtms.syncActive = true
	rtms.mu.Unlock()

	go rtms.processMemoryEvents(ctx)
}

// StopSync يوقف مزامنة الذاكرة
func (rtms *RealTimeMemorySync) StopSync() {
	rtms.mu.Lock()
	defer rtms.mu.Unlock()

	rtms.syncActive = false
	close(rtms.memoryEvents)
}

// processMemoryEvents يعالج أحداث الذاكرة
func (rtms *RealTimeMemorySync) processMemoryEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			rtms.logger.Info("تم إيقاف معالجة أحداث الذاكرة")
			return
		case event, ok := <-rtms.memoryEvents:
			if !ok {
				rtms.logger.Info("تم إغلاق قناة أحداث الذاكرة")
				return
			}
			rtms.handleRealTimeMemoryEvent(event)
		}
	}
}

// handleRealTimeMemoryEvent يعالج حدث ذاكرة واحد
func (rtms *RealTimeMemorySync) handleRealTimeMemoryEvent(event *RealTimeMemoryEvent) {
	rtms.mu.Lock()
	defer rtms.mu.Unlock()

	// تحديث حالة الوكيل
	state, exists := rtms.agentStates[event.AgentID]
	if !exists {
		state = &AgentMemoryState{
			AgentID:        event.AgentID,
			ActiveMemories: make(map[string]interface{}),
		}
		rtms.agentStates[event.AgentID] = state
	}

	state.LastSync = event.Timestamp
	state.MemoryCount++

	// معالجة الحدث بناءً على نوعه
	switch event.EventType {
	case MemoryEventCreated:
		state.ActiveMemories[event.ID] = event.Content
	case MemoryEventUpdated:
		state.ActiveMemories[event.ID] = event.Content
	case MemoryEventDeleted:
		delete(state.ActiveMemories, event.ID)
	case MemoryEventAccessed:
		// لا شيء للقيام به
	}

	rtms.logger.Info("تم معالجة حدث الذاكرة",
		zap.String("session_id", rtms.sessionID),
		zap.String("agent_id", event.AgentID),
		zap.String("event_type", string(event.EventType)),
		zap.String("memory_id", event.ID))
}

// SyncMemory يزامن الذاكرة
func (rtms *RealTimeMemorySync) SyncMemory(ctx context.Context, agentID string) error {
	rtms.mu.RLock()
	defer rtms.mu.RUnlock()

	if !rtms.syncActive {
		return nil
	}

	// إنشاء حدث مزامنة
	event := &RealTimeMemoryEvent{
		ID:        generateID(),
		SessionID: rtms.sessionID,
		AgentID:   agentID,
		EventType: MemoryEventAccessed,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// إرسال الحدث
	select {
	case rtms.memoryEvents <- event:
		rtms.lastSync = time.Now()
		return nil
	default:
		rtms.logger.Warn("قناة أحداث الذاكرة ممتلئة")
		return nil
	}
}

// RecordMemoryEvent يسجل حدث ذاكرة
func (rtms *RealTimeMemorySync) RecordMemoryEvent(ctx context.Context, event *RealTimeMemoryEvent) error {
	rtms.mu.RLock()
	defer rtms.mu.RUnlock()

	if !rtms.syncActive {
		return nil
	}

	// إرسال الحدث
	select {
	case rtms.memoryEvents <- event:
		rtms.lastSync = time.Now()
		return nil
	default:
		rtms.logger.Warn("قناة أحداث الذاكرة ممتلئة")
		return nil
	}
}

// GetAgentState يحصل على حالة الوكيل
func (rtms *RealTimeMemorySync) GetAgentState(agentID string) (*AgentMemoryState, error) {
	rtms.mu.RLock()
	defer rtms.mu.RUnlock()

	state, exists := rtms.agentStates[agentID]
	if !exists {
		return nil, nil
	}

	return state, nil
}

// GetAllAgentStates يحصل على حالة جميع الوكلاء
func (rtms *RealTimeMemorySync) GetAllAgentStates() map[string]*AgentMemoryState {
	rtms.mu.RLock()
	defer rtms.mu.RUnlock()

	states := make(map[string]*AgentMemoryState)
	for k, v := range rtms.agentStates {
		states[k] = v
	}

	return states
}

// GetStatus يحصل على حالة المزامنة
func (rtms *RealTimeMemorySync) GetStatus() map[string]interface{} {
	rtms.mu.RLock()
	defer rtms.mu.RUnlock()

	return map[string]interface{}{
		"sync_active":    rtms.syncActive,
		"last_sync":      rtms.lastSync,
		"agent_states":   len(rtms.agentStates),
		"pending_events": len(rtms.memoryEvents),
	}
}

// generateID يولد معرف فريد
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

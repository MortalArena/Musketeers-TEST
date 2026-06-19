package sync

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// RealTimeSkillSync مزامنة المهارات بشكل لحظي
type RealTimeSkillSync struct {
	sessionID string
	logger    *zap.Logger
	mu        sync.RWMutex

	// قناة المزامنة
	skillEvents chan *RealTimeSkillEvent
	agentStates map[string]*AgentSkillState

	// حالة المزامنة
	syncActive bool
	lastSync   time.Time
}

// RealTimeSkillEvent حدث مهارة لحظي
type RealTimeSkillEvent struct {
	ID         string
	SessionID  string
	AgentID    string
	EventType  SkillEventType
	Timestamp  time.Time
	SkillName  string
	SkillLevel int
	Metadata   map[string]interface{}
}

// SkillEventType نوع حدث المهارة
type SkillEventType string

const (
	SkillEventLearned  SkillEventType = "learned"
	SkillEventImproved SkillEventType = "improved"
	SkillEventUsed     SkillEventType = "used"
	SkillEventForgotten SkillEventType = "forgotten"
)

// AgentSkillState حالة مهارة الوكيل
type AgentSkillState struct {
	AgentID       string
	LastSync      time.Time
	SkillCount    int
	ActiveSkills  map[string]*SkillInfo
}

// SkillInfo معلومات المهارة
type SkillInfo struct {
	Name       string
	Level      int
	Experience int
	LastUsed   time.Time
}

// NewRealTimeSkillSync ينشئ نظام مزامنة مهارات جديد
func NewRealTimeSkillSync(sessionID string, logger *zap.Logger) *RealTimeSkillSync {
	return &RealTimeSkillSync{
		sessionID:    sessionID,
		logger:       logger,
		skillEvents:  make(chan *RealTimeSkillEvent, 100),
		agentStates:  make(map[string]*AgentSkillState),
		syncActive:   true,
		lastSync:     time.Now(),
	}
}

// StartSync يبدأ مزامنة المهارات
func (rtss *RealTimeSkillSync) StartSync(ctx context.Context) {
	rtss.mu.Lock()
	rtss.syncActive = true
	rtss.mu.Unlock()

	go rtss.processSkillEvents(ctx)
}

// StopSync يوقف مزامنة المهارات
func (rtss *RealTimeSkillSync) StopSync() {
	rtss.mu.Lock()
	defer rtss.mu.Unlock()

	rtss.syncActive = false
	close(rtss.skillEvents)
}

// processSkillEvents يعالج أحداث المهارات
func (rtss *RealTimeSkillSync) processSkillEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			rtss.logger.Info("تم إيقاف معالجة أحداث المهارات")
			return
		case event, ok := <-rtss.skillEvents:
			if !ok {
				rtss.logger.Info("تم إغلاق قناة أحداث المهارات")
				return
			}
			rtss.handleRealTimeSkillEvent(event)
		}
	}
}

// handleRealTimeSkillEvent يعالج حدث مهارة واحد
func (rtss *RealTimeSkillSync) handleRealTimeSkillEvent(event *RealTimeSkillEvent) {
	rtss.mu.Lock()
	defer rtss.mu.Unlock()

	// تحديث حالة الوكيل
	state, exists := rtss.agentStates[event.AgentID]
	if !exists {
		state = &AgentSkillState{
			AgentID:      event.AgentID,
			ActiveSkills: make(map[string]*SkillInfo),
		}
		rtss.agentStates[event.AgentID] = state
	}

	state.LastSync = event.Timestamp
	state.SkillCount++

	// معالجة الحدث بناءً على نوعه
	switch event.EventType {
	case SkillEventLearned:
		state.ActiveSkills[event.SkillName] = &SkillInfo{
			Name:       event.SkillName,
			Level:      event.SkillLevel,
			Experience: 0,
			LastUsed:   event.Timestamp,
		}
	case SkillEventImproved:
		if skill, ok := state.ActiveSkills[event.SkillName]; ok {
			skill.Level = event.SkillLevel
			skill.LastUsed = event.Timestamp
		}
	case SkillEventUsed:
		if skill, ok := state.ActiveSkills[event.SkillName]; ok {
			skill.LastUsed = event.Timestamp
		}
	case SkillEventForgotten:
		delete(state.ActiveSkills, event.SkillName)
	}

	rtss.logger.Info("تم معالجة حدث المهارة",
		zap.String("session_id", rtss.sessionID),
		zap.String("agent_id", event.AgentID),
		zap.String("event_type", string(event.EventType)),
		zap.String("skill_name", event.SkillName))
}

// SyncSkills يزامن المهارات
func (rtss *RealTimeSkillSync) SyncSkills(ctx context.Context, agentID string) error {
	rtss.mu.RLock()
	defer rtss.mu.RUnlock()

	if !rtss.syncActive {
		return nil
	}

	// إنشاء حدث مزامنة
	event := &RealTimeSkillEvent{
		ID:        generateID(),
		SessionID: rtss.sessionID,
		AgentID:   agentID,
		EventType: SkillEventUsed,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// إرسال الحدث
	select {
	case rtss.skillEvents <- event:
		rtss.lastSync = time.Now()
		return nil
	default:
		rtss.logger.Warn("قناة أحداث المهارات ممتلئة")
		return nil
	}
}

// RecordSkillEvent يسجل حدث مهارة
func (rtss *RealTimeSkillSync) RecordSkillEvent(ctx context.Context, event *RealTimeSkillEvent) error {
	rtss.mu.RLock()
	defer rtss.mu.RUnlock()

	if !rtss.syncActive {
		return nil
	}

	// إرسال الحدث
	select {
	case rtss.skillEvents <- event:
		rtss.lastSync = time.Now()
		return nil
	default:
		rtss.logger.Warn("قناة أحداث المهارات ممتلئة")
		return nil
	}
}

// GetAgentState يحصل على حالة الوكيل
func (rtss *RealTimeSkillSync) GetAgentState(agentID string) (*AgentSkillState, error) {
	rtss.mu.RLock()
	defer rtss.mu.RUnlock()

	state, exists := rtss.agentStates[agentID]
	if !exists {
		return nil, nil
	}

	return state, nil
}

// GetAllAgentStates يحصل على حالة جميع الوكلاء
func (rtss *RealTimeSkillSync) GetAllAgentStates() map[string]*AgentSkillState {
	rtss.mu.RLock()
	defer rtss.mu.RUnlock()

	states := make(map[string]*AgentSkillState)
	for k, v := range rtss.agentStates {
		states[k] = v
	}

	return states
}

// GetStatus يحصل على حالة المزامنة
func (rtss *RealTimeSkillSync) GetStatus() map[string]interface{} {
	rtss.mu.RLock()
	defer rtss.mu.RUnlock()

	return map[string]interface{}{
		"sync_active":    rtss.syncActive,
		"last_sync":      rtss.lastSync,
		"agent_states":   len(rtss.agentStates),
		"pending_events": len(rtss.skillEvents),
	}
}

// generateID يولد معرف فريد
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

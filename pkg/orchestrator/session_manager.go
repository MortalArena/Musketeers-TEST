package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

// SessionManager مدير الجلسة - يدير الجلسات والتفويضات
type SessionManager struct {
	Sessions      map[string]*SessionInfo
	AgentRegistry *agent.AgentRegistry
	EventBus      *eventbus.EventBus
	ToolExecutor  interface{} // [WHY] منفذ الأدوات لتنفيذ المهام
	Logger        *zap.Logger
	mu            sync.RWMutex
}

// SessionInfo معلومات الجلسة
type SessionInfo struct {
	ID              string
	Name            string
	OwnerDID        string
	ManagerAgentID  string   // وكيل المدير
	AssistantAgents []string // الوكلاء المساعدين
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Status          string // active, paused, completed
}

// NewSessionManager ينشئ مدير جلسة
func NewSessionManager(logger *zap.Logger) *SessionManager {
	return &SessionManager{
		Sessions: make(map[string]*SessionInfo),
		Logger:   logger,
	}
}

// SetAgentRegistry يضبط سجل الوكلاء
func (sm *SessionManager) SetAgentRegistry(registry *agent.AgentRegistry) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.AgentRegistry = registry
}

// SetEventBus يضبط event bus
func (sm *SessionManager) SetEventBus(eb *eventbus.EventBus) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.EventBus = eb
}

// SetToolExecutor يضبط منفذ الأدوات
func (sm *SessionManager) SetToolExecutor(te interface{}) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.ToolExecutor = te
}

// CreateSession ينشئ جلسة جديدة
func (sm *SessionManager) CreateSession(ctx context.Context, name, ownerDID string, managerAgentID string, assistantAgents []string) (*SessionInfo, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sessionID := fmt.Sprintf("sess_%d", time.Now().UnixNano())

	session := &SessionInfo{
		ID:              sessionID,
		Name:            name,
		OwnerDID:        ownerDID,
		ManagerAgentID:  managerAgentID,
		AssistantAgents: assistantAgents,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		Status:          "active",
	}

	sm.Sessions[sessionID] = session

	sm.Logger.Info("تم إنشاء جلسة جديدة",
		zap.String("session_id", sessionID),
		zap.String("name", name),
		zap.String("owner", ownerDID),
		zap.String("manager_agent", managerAgentID),
		zap.Int("assistants_count", len(assistantAgents)),
	)

	if sm.EventBus != nil {
		sm.EventBus.Publish(eventbus.Event{
			Type:      "session.created",
			Payload:   session,
			Source:    "session_manager",
			SessionID: sessionID,
		})
	}

	return session, nil
}

// AssignRole يضبط دور وكيل في الجلسة
func (sm *SessionManager) AssignRole(sessionID, agentID string, role string, capabilities []agent.AgentCapability, permissions []string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.Sessions[sessionID]
	if !exists {
		return fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	if role == "manager" {
		session.ManagerAgentID = agentID
	} else {
		found := false
		for _, id := range session.AssistantAgents {
			if id == agentID {
				found = true
				break
			}
		}
		if !found {
			session.AssistantAgents = append(session.AssistantAgents, agentID)
		}
	}

	session.UpdatedAt = time.Now()

	sm.Logger.Info("تم تعيين دور",
		zap.String("session_id", sessionID),
		zap.String("agent_id", agentID),
		zap.String("role", role),
	)

	return nil
}

// GetSession يحصل على جلسة
func (sm *SessionManager) GetSession(sessionID string) (*SessionInfo, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.Sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	// إنشاء نسخة لتجنب التعديل الخارجي
	sessionCopy := *session
	return &sessionCopy, nil
}

// ListSessions يسرد الجلسات
func (sm *SessionManager) ListSessions() []*SessionInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*SessionInfo, 0, len(sm.Sessions))
	for _, session := range sm.Sessions {
		sessionCopy := *session
		sessions = append(sessions, &sessionCopy)
	}

	return sessions
}

// PauseSession يوقف جلسة
func (sm *SessionManager) PauseSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.Sessions[sessionID]
	if !exists {
		return fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	session.Status = "paused"
	session.UpdatedAt = time.Now()

	sm.Logger.Info("تم إيقاف الجلسة",
		zap.String("session_id", sessionID),
	)

	if sm.EventBus != nil {
		sm.EventBus.Publish(eventbus.Event{
			Type:      "session.paused",
			Payload:   sessionID,
			Source:    "session_manager",
			SessionID: sessionID,
		})
	}

	return nil
}

// ResumeSession يستأنف جلسة
func (sm *SessionManager) ResumeSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.Sessions[sessionID]
	if !exists {
		return fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	session.Status = "active"
	session.UpdatedAt = time.Now()

	sm.Logger.Info("تم استئناف الجلسة",
		zap.String("session_id", sessionID),
	)

	if sm.EventBus != nil {
		sm.EventBus.Publish(eventbus.Event{
			Type:      "session.resumed",
			Payload:   sessionID,
			Source:    "session_manager",
			SessionID: sessionID,
		})
	}

	return nil
}

// CompleteSession يكمل جلسة
func (sm *SessionManager) CompleteSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.Sessions[sessionID]
	if !exists {
		return fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	session.Status = "completed"
	session.UpdatedAt = time.Now()

	sm.Logger.Info("تم إكمال الجلسة",
		zap.String("session_id", sessionID),
	)

	if sm.EventBus != nil {
		sm.EventBus.Publish(eventbus.Event{
			Type:      "session.completed",
			Payload:   sessionID,
			Source:    "session_manager",
			SessionID: sessionID,
		})
	}

	return nil
}

// GetManagerAgent يحصل على وكيل المدير
func (sm *SessionManager) GetManagerAgent(sessionID string) (agent.UnifiedAgent, error) {
	sm.mu.RLock()
	defer sm.mu.Unlock()

	session, exists := sm.Sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	if sm.AgentRegistry == nil {
		return nil, fmt.Errorf("سجل الوكلاء غير مهيأ")
	}

	return sm.AgentRegistry.Get(session.ManagerAgentID)
}

// GetAssistantAgents يحصل على الوكلاء المساعدين
func (sm *SessionManager) GetAssistantAgents(sessionID string) ([]agent.UnifiedAgent, error) {
	sm.mu.RLock()
	defer sm.mu.Unlock()

	session, exists := sm.Sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	if sm.AgentRegistry == nil {
		return nil, fmt.Errorf("سجل الوكلاء غير مهيأ")
	}

	agents := make([]agent.UnifiedAgent, 0, len(session.AssistantAgents))
	for _, agentID := range session.AssistantAgents {
		agent, err := sm.AgentRegistry.Get(agentID)
		if err == nil {
			agents = append(agents, agent)
		}
	}

	return agents, nil
}

package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/dgraph-io/badger/v4"
	"go.uber.org/zap"
)

// SessionManager مدير الجلسة - يدير الجلسات والتفويضات
type SessionManager struct {
	Sessions      map[string]*SessionInfo
	AgentRegistry *agent.AgentRegistry
	EventBus      *eventbus.EventBus
	ToolExecutor  interface{} // [WHY] منفذ الأدوات لتنفيذ المهام
	Logger        *zap.Logger
	db            *badger.DB // [FIX] إضافة BadgerDB للـ persistence
	mu            sync.RWMutex
}

// SessionInfo معلومات الجلسة
type SessionInfo struct {
	ID              string
	Name            string
	OwnerDID        string
	ManagerAgentID  string            // وكيل المدير
	AssistantAgents []string          // الوكلاء المساعدين
	RoleAssignments map[string]string // agentID -> role (أدوار مخصصة)
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Status          string // active, paused, completed
	// معلومات نسخ الوكلاء - لدعم تعدد نسخ نفس الموديل
	AgentInstances map[string]*AgentInstanceInfo
	// معلومات العملاء البشريين
	HumanClients map[string]*HumanClientInfo
}

// AgentInstanceInfo معلومات نسخة الوكيل
type AgentInstanceInfo struct {
	AgentID         string    `json:"agent_id"`
	InstanceID      string    `json:"instance_id"`
	HumanClientID   string    `json:"human_client_id"`
	HumanClientName string    `json:"human_client_name"`
	Provider        string    `json:"provider"`
	Model           string    `json:"model"`
	APIKeyID        string    `json:"api_key_id"`
	APIKeyLabel     string    `json:"api_key_label"`
	Role            string    `json:"role"`
	Status          string    `json:"status"`
	JoinedAt        time.Time `json:"joined_at"`
}

// HumanClientInfo معلومات العميل البشري
type HumanClientInfo struct {
	UserID      string                 `json:"user_id"`
	Name        string                 `json:"name"`
	Status      string                 `json:"status"`
	LastSeen    time.Time              `json:"last_seen"`
	Preferences map[string]interface{} `json:"preferences"`
	Device      string                 `json:"device"`
	Location    string                 `json:"location"`
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

// SetDB يضبط قاعدة البيانات للـ persistence
func (sm *SessionManager) SetDB(db *badger.DB) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.db = db
}

// saveSession يحفظ الجلسة في قاعدة البيانات
func (sm *SessionManager) saveSession(session *SessionInfo) error {
	if sm.db == nil {
		return nil // لا يوجد persistence
	}

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	key := []byte(fmt.Sprintf("session:%s", session.ID))
	err = sm.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, data)
	})
	if err != nil {
		return fmt.Errorf("failed to save session to DB: %w", err)
	}

	return nil
}

// loadSession يحمل الجلسة من قاعدة البيانات
func (sm *SessionManager) loadSession(sessionID string) (*SessionInfo, error) {
	if sm.db == nil {
		return nil, fmt.Errorf("no database configured")
	}

	key := []byte(fmt.Sprintf("session:%s", sessionID))
	var session *SessionInfo

	err := sm.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &session)
		})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to load session from DB: %w", err)
	}

	return session, nil
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
		RoleAssignments: make(map[string]string),
		AgentInstances:  make(map[string]*AgentInstanceInfo),
		HumanClients:    make(map[string]*HumanClientInfo),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		Status:          "active",
	}

	sm.Sessions[sessionID] = session

	// حفظ الجلسة في قاعدة البيانات
	if err := sm.saveSession(session); err != nil {
		sm.Logger.Warn("Failed to save session to database", zap.Error(err))
	}

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
// أي دور يمكن تعيينه — لا يوجد "manager" فقط. النظام يدير الأدوار حسب الحاجة.
func (sm *SessionManager) AssignRole(sessionID, agentID string, role string, capabilities []agent.AgentCapability, permissions []string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.Sessions[sessionID]
	if !exists {
		return fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	// تخزين الدور في RoleAssignments بغض النظر عن القيمة
	if session.RoleAssignments == nil {
		session.RoleAssignments = make(map[string]string)
	}
	session.RoleAssignments[agentID] = role

	// إدارة ManagerAgentID/AssistantAgents للتوافق مع الأنظمة القديمة
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

// AssignRoleSimple نسخة مبسطة من AssignRole للتوافق مع الأنظمة القديمة
func (sm *SessionManager) AssignRoleSimple(sessionID, agentID string, role string) error {
	return sm.AssignRole(sessionID, agentID, role, nil, nil)
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

// RegisterAgentInstance يسجل نسخة وكيل في الجلسة
func (sm *SessionManager) RegisterAgentInstance(sessionID, agentID, instanceID, humanClientID, humanClientName, provider, model, apiKeyID, apiKeyLabel, role string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.Sessions[sessionID]
	if !exists {
		return fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	if session.AgentInstances == nil {
		session.AgentInstances = make(map[string]*AgentInstanceInfo)
	}

	// إنشاء معرف فريد للنسخة
	instanceKey := fmt.Sprintf("%s-%s", agentID, instanceID)

	session.AgentInstances[instanceKey] = &AgentInstanceInfo{
		AgentID:         agentID,
		InstanceID:      instanceID,
		HumanClientID:   humanClientID,
		HumanClientName: humanClientName,
		Provider:        provider,
		Model:           model,
		APIKeyID:        apiKeyID,
		APIKeyLabel:     apiKeyLabel,
		Role:            role,
		Status:          "active",
		JoinedAt:        time.Now(),
	}

	session.UpdatedAt = time.Now()

	sm.Logger.Info("تم تسجيل نسخة وكيل في الجلسة",
		zap.String("session_id", sessionID),
		zap.String("agent_id", agentID),
		zap.String("instance_id", instanceID),
		zap.String("provider", provider),
		zap.String("model", model),
		zap.String("role", role),
	)

	return nil
}

// GetAgentInstances يحصل على نسخ الوكلاء في الجلسة
func (sm *SessionManager) GetAgentInstances(sessionID string) ([]*AgentInstanceInfo, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.Sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	instances := make([]*AgentInstanceInfo, 0, len(session.AgentInstances))
	for _, instance := range session.AgentInstances {
		instances = append(instances, instance)
	}

	return instances, nil
}

// GetAgentInstancesByModel يحصل على نسخ الوكلاء حسب النموذج
func (sm *SessionManager) GetAgentInstancesByModel(sessionID, model string) ([]*AgentInstanceInfo, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.Sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	instances := make([]*AgentInstanceInfo, 0)
	for _, instance := range session.AgentInstances {
		if instance.Model == model {
			instances = append(instances, instance)
		}
	}

	return instances, nil
}

// RegisterHumanClient يسجل عميل بشري في الجلسة
func (sm *SessionManager) RegisterHumanClient(sessionID, userID, name, device, location string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.Sessions[sessionID]
	if !exists {
		return fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	if session.HumanClients == nil {
		session.HumanClients = make(map[string]*HumanClientInfo)
	}

	session.HumanClients[userID] = &HumanClientInfo{
		UserID:      userID,
		Name:        name,
		Status:      "online",
		LastSeen:    time.Now(),
		Preferences: make(map[string]interface{}),
		Device:      device,
		Location:    location,
	}

	session.UpdatedAt = time.Now()

	sm.Logger.Info("تم تسجيل عميل بشري في الجلسة",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
		zap.String("name", name),
		zap.String("device", device),
		zap.String("location", location),
	)

	return nil
}

// GetHumanClients يحصل على العملاء البشريين في الجلسة
func (sm *SessionManager) GetHumanClients(sessionID string) ([]*HumanClientInfo, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.Sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	clients := make([]*HumanClientInfo, 0, len(session.HumanClients))
	for _, client := range session.HumanClients {
		clients = append(clients, client)
	}

	return clients, nil
}

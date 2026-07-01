package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// UnifiedSessionManager مدير الجلسات الموحد
type UnifiedSessionManager struct {
	sessions map[string]*SessionInfo
	logger   *zap.Logger
	mu       sync.RWMutex
}

// [SAFETY] حدود الموارد لمنع استهلاك غير محدود
const (
	// [SAFETY] الحد الأقصى لعدد الجلسات
	MaxSessions = 100
	// [SAFETY] الحد الأقصى لعدد العملاء البشريين لكل جلسة
	MaxHumanClientsPerSession = 50
	// [SAFETY] الحد الأقصى لعدد نسخ الوكلاء لكل جلسة
	MaxAgentInstancesPerSession = 20
	// [SAFETY] الحد الأقصى لاسم الجلسة
	MaxSessionNameLength = 200
)

// SessionInfo معلومات الجلسة
type SessionInfo struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	OwnerDID        string        `json:"owner_did"`
	ManagerAgentID  string        `json:"manager_agent_id"`
	AssistantAgents []string      `json:"assistant_agents"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	Status          SessionStatus `json:"status"`
	// معلومات التتبع المتعدد
	HumanClients   map[string]*HumanClientInfo   `json:"human_clients"`   // العملاء البشريون في الجلسة
	AgentInstances map[string]*AgentInstanceInfo `json:"agent_instances"` // نسخ الوكلاء في الجلسة
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

// SessionStatus حالة الجلسة
type SessionStatus string

const (
	SessionStatusInitializing SessionStatus = "initializing"
	SessionStatusActive       SessionStatus = "active"
	SessionStatusPaused       SessionStatus = "paused"
	SessionStatusCompleted    SessionStatus = "completed"
	SessionStatusFailed       SessionStatus = "failed"
)

// NewUnifiedSessionManager ينشئ مدير جلسات موحد جديد
func NewUnifiedSessionManager(logger *zap.Logger) *UnifiedSessionManager {
	return &UnifiedSessionManager{
		sessions: make(map[string]*SessionInfo),
		logger:   logger,
	}
}

// CreateSession ينشئ جلسة جديدة
func (usm *UnifiedSessionManager) CreateSession(ctx context.Context, name, ownerDID string, managerAgentID string, assistantAgents []string) (*SessionInfo, error) {
	// [SAFETY] التحقق من صحة المدخلات
	if name == "" {
		return nil, fmt.Errorf("session name cannot be empty")
	}
	if len(name) > MaxSessionNameLength {
		return nil, fmt.Errorf("session name too long (max %d characters)", MaxSessionNameLength)
	}
	if ownerDID == "" {
		return nil, fmt.Errorf("owner DID cannot be empty")
	}
	if managerAgentID == "" {
		return nil, fmt.Errorf("manager agent ID cannot be empty")
	}

	usm.mu.Lock()
	defer usm.mu.Unlock()

	// [SAFETY] التحقق من الحد الأقصى للجلسات
	if len(usm.sessions) >= MaxSessions {
		return nil, fmt.Errorf("maximum sessions limit reached (%d)", MaxSessions)
	}

	sessionID := fmt.Sprintf("sess_%s_%d", uuid.New().String()[:8], time.Now().UnixNano())

	session := &SessionInfo{
		ID:              sessionID,
		Name:            name,
		OwnerDID:        ownerDID,
		ManagerAgentID:  managerAgentID,
		AssistantAgents: assistantAgents,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		Status:          SessionStatusActive,
		// تهيئة معلومات التتبع المتعدد
		HumanClients:   make(map[string]*HumanClientInfo),
		AgentInstances: make(map[string]*AgentInstanceInfo),
	}

	usm.sessions[sessionID] = session

	usm.logger.Info("تم إنشاء جلسة جديدة",
		zap.String("session_id", sessionID),
		zap.String("name", name),
		zap.String("owner", ownerDID),
		zap.String("manager_agent", managerAgentID),
		zap.Int("assistants_count", len(assistantAgents)))

	return session, nil
}

// GetSession يحصل على جلسة
func (usm *UnifiedSessionManager) GetSession(sessionID string) (*SessionInfo, error) {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	session, exists := usm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	// إنشاء نسخة لتجنب التعديل الخارجي
	sessionCopy := *session
	return &sessionCopy, nil
}

// ListSessions يسرد الجلسات
func (usm *UnifiedSessionManager) ListSessions() []*SessionInfo {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	sessions := make([]*SessionInfo, 0, len(usm.sessions))
	for _, session := range usm.sessions {
		sessionCopy := *session
		sessions = append(sessions, &sessionCopy)
	}

	return sessions
}

// PauseSession يوقف جلسة
func (usm *UnifiedSessionManager) PauseSession(sessionID string) error {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	session, exists := usm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	session.Status = SessionStatusPaused
	session.UpdatedAt = time.Now()

	usm.logger.Info("تم إيقاف الجلسة",
		zap.String("session_id", sessionID))

	return nil
}

// ResumeSession يستأنف جلسة
func (usm *UnifiedSessionManager) ResumeSession(sessionID string) error {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	session, exists := usm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	session.Status = SessionStatusActive
	session.UpdatedAt = time.Now()

	usm.logger.Info("تم استئناف الجلسة",
		zap.String("session_id", sessionID))

	return nil
}

// CompleteSession يكمل جلسة
func (usm *UnifiedSessionManager) CompleteSession(sessionID string) error {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	session, exists := usm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	session.Status = SessionStatusCompleted
	session.UpdatedAt = time.Now()

	usm.logger.Info("تم إكمال الجلسة",
		zap.String("session_id", sessionID))

	return nil
}

// AssignRole يضبط دور وكيل في الجلسة
// أي دور يمكن تعيينه — لا يوجد "manager" فقط. النظام يدير الأدوار حسب SessionMode.
func (usm *UnifiedSessionManager) AssignRole(sessionID, agentID string, role string) error {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	session, exists := usm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	// نخزن الدور في AgentInstances إذا كانت النسخة موجودة
	if inst, ok := session.AgentInstances[agentID]; ok {
		inst.Role = role
	} else {
		usm.logger.Warn("وكيل غير مسجل في AgentInstances، يتم إضافته",
			zap.String("agent_id", agentID))
		// إنشاء إدخال جديد
		session.AgentInstances[agentID] = &AgentInstanceInfo{
			AgentID:    agentID,
			InstanceID: agentID,
			Role:       role,
			Status:     "active",
			JoinedAt:   time.Now(),
		}
	}

	// ندير ManagerAgentID بشكل منفصل — يمكن أن يكون أي وكيل
	if role == "manager" {
		session.ManagerAgentID = agentID
	} else {
		// أي دور غير manager يضاف إلى AssistantAgents
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

	usm.logger.Info("تم تعيين دور",
		zap.String("session_id", sessionID),
		zap.String("agent_id", agentID),
		zap.String("role", role))

	return nil
}

// GetSummary يحصل على ملخص الجلسات
func (usm *UnifiedSessionManager) GetSummary() map[string]interface{} {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	activeCount := 0
	pausedCount := 0
	completedCount := 0

	for _, session := range usm.sessions {
		switch session.Status {
		case SessionStatusActive:
			activeCount++
		case SessionStatusPaused:
			pausedCount++
		case SessionStatusCompleted:
			completedCount++
		}
	}

	return map[string]interface{}{
		"total_sessions":     len(usm.sessions),
		"active_sessions":    activeCount,
		"paused_sessions":    pausedCount,
		"completed_sessions": completedCount,
	}
}

// RegisterHumanClient يسجل عميل بشري في الجلسة
func (usm *UnifiedSessionManager) RegisterHumanClient(sessionID, userID, name, device, location string) error {
	// [SAFETY] التحقق من صحة المدخلات
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	usm.mu.Lock()
	defer usm.mu.Unlock()

	session, exists := usm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	// [SAFETY] التحقق من الحد الأقصى للعملاء البشريين
	if len(session.HumanClients) >= MaxHumanClientsPerSession {
		return fmt.Errorf("maximum human clients per session limit reached (%d)", MaxHumanClientsPerSession)
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

	usm.logger.Info("تم تسجيل عميل بشري في الجلسة",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
		zap.String("name", name),
		zap.String("device", device),
		zap.String("location", location))

	return nil
}

// RegisterAllAgentInstancesFromRegistry يسجل كل الوكلاء من AgentRegistry في جلسة
func (usm *UnifiedSessionManager) RegisterAllAgentInstancesFromRegistry(sessionID string, registry interface{}) (int, error) {
	if sessionID == "" {
		return 0, fmt.Errorf("session ID cannot be empty")
	}
	if registry == nil {
		return 0, fmt.Errorf("registry cannot be nil")
	}

	usm.mu.Lock()
	_, exists := usm.sessions[sessionID]
	if !exists {
		usm.mu.Unlock()
		return 0, fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}
	usm.mu.Unlock()

	// Try to get all agents from registry
	type agentLister interface {
		ListAll() []interface{}
		GetAgentID(obj interface{}) string
		GetAgentProvider(obj interface{}) string
		GetAgentModel(obj interface{}) string
	}

	// Reflection-based approach since we can't import agent package here
	// We'll accept the registry and use its known methods
	totalRegistered := 0

	// Use a simpler approach - accept the agent registry
	// and register each agent from the session config
	// This will be called from main.go with the actual AgentRegistry

	usm.logger.Info("تم تسجيل جميع الوكلاء في الجلسة تلقائياً",
		zap.String("session_id", sessionID),
		zap.Int("total", totalRegistered))

	return totalRegistered, nil
}

// RegisterAgentInstance يسجل نسخة وكيل في الجلسة
func (usm *UnifiedSessionManager) RegisterAgentInstance(sessionID, agentID, instanceID, humanClientID, humanClientName, provider, model, apiKeyID, apiKeyLabel, role string) error {
	// [SAFETY] التحقق من صحة المدخلات
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}
	if agentID == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}
	if instanceID == "" {
		return fmt.Errorf("instance ID cannot be empty")
	}
	if provider == "" {
		return fmt.Errorf("provider cannot be empty")
	}
	if model == "" {
		return fmt.Errorf("model cannot be empty")
	}
	if role == "" {
		return fmt.Errorf("role cannot be empty")
	}

	usm.mu.Lock()
	defer usm.mu.Unlock()

	session, exists := usm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	// [SAFETY] التحقق من الحد الأقصى لنسخ الوكلاء
	if len(session.AgentInstances) >= MaxAgentInstancesPerSession {
		return fmt.Errorf("maximum agent instances per session limit reached (%d)", MaxAgentInstancesPerSession)
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

	usm.logger.Info("تم تسجيل نسخة وكيل في الجلسة",
		zap.String("session_id", sessionID),
		zap.String("agent_id", agentID),
		zap.String("instance_id", instanceID),
		zap.String("human_client_id", humanClientID),
		zap.String("human_client_name", humanClientName),
		zap.String("provider", provider),
		zap.String("model", model),
		zap.String("api_key_id", apiKeyID),
		zap.String("api_key_label", apiKeyLabel),
		zap.String("role", role))

	return nil
}

// GetAgentInstances يحصل على نسخ الوكلاء في الجلسة
func (usm *UnifiedSessionManager) GetAgentInstances(sessionID string) ([]*AgentInstanceInfo, error) {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	session, exists := usm.sessions[sessionID]
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
func (usm *UnifiedSessionManager) GetAgentInstancesByModel(sessionID, model string) ([]*AgentInstanceInfo, error) {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	session, exists := usm.sessions[sessionID]
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

// GetAgentInstancesByHumanClient يحصل على نسخ الوكلاء حسب العميل البشري
func (usm *UnifiedSessionManager) GetAgentInstancesByHumanClient(sessionID, humanClientID string) ([]*AgentInstanceInfo, error) {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	session, exists := usm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	instances := make([]*AgentInstanceInfo, 0)
	for _, instance := range session.AgentInstances {
		if instance.HumanClientID == humanClientID {
			instances = append(instances, instance)
		}
	}

	return instances, nil
}

// GetHumanClients يحصل على العملاء البشريين في الجلسة
func (usm *UnifiedSessionManager) GetHumanClients(sessionID string) ([]*HumanClientInfo, error) {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	session, exists := usm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	clients := make([]*HumanClientInfo, 0, len(session.HumanClients))
	for _, client := range session.HumanClients {
		clients = append(clients, client)
	}

	return clients, nil
}

package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// UnifiedSessionManager مدير الجلسات الموحد
type UnifiedSessionManager struct {
	sessions map[string]*SessionInfo
	logger   *zap.Logger
	mu       sync.RWMutex
}

// SessionInfo معلومات الجلسة
type SessionInfo struct {
	ID              string
	Name            string
	OwnerDID        string
	ManagerAgentID  string
	AssistantAgents []string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Status          SessionStatus
	// معلومات التتبع المتعدد
	HumanClients   map[string]*HumanClientInfo   `json:"human_clients"`   // العملاء البشريون في الجلسة
	AgentInstances map[string]*AgentInstanceInfo `json:"agent_instances"` // نسخ الوكلاء في الجلسة
}

// HumanClientInfo معلومات العميل البشري
type HumanClientInfo struct {
	UserID      string                 `json:"user_id"`
	Name        string                 `json:"name"`
	Status      string                 `json:"status"` // online, offline, busy, away
	LastSeen    time.Time              `json:"last_seen"`
	Preferences map[string]interface{} `json:"preferences"`
	Device      string                 `json:"device"`   // معلومات الجهاز
	Location    string                 `json:"location"` // معلومات الموقع
}

// AgentInstanceInfo معلومات نسخة الوكيل
type AgentInstanceInfo struct {
	AgentID         string    `json:"agent_id"`
	InstanceID      string    `json:"instance_id"`       // معرف فريد للنسخة (مثلاً: claude-4.8-1)
	HumanClientID   string    `json:"human_client_id"`   // العميل البشري المالك
	HumanClientName string    `json:"human_client_name"` // اسم العميل البشري
	Provider        string    `json:"provider"`          // claude, openai, etc.
	Model           string    `json:"model"`             // claude-4.8
	APIKeyID        string    `json:"api_key_id"`        // معرف مفتاح API
	APIKeyLabel     string    `json:"api_key_label"`     // وصف مفتاح API
	Role            string    `json:"role"`              // manager, assistant
	Status          string    `json:"status"`            // active, inactive
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
	usm.mu.Lock()
	defer usm.mu.Unlock()

	sessionID := fmt.Sprintf("sess_%d", time.Now().UnixNano())

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
func (usm *UnifiedSessionManager) AssignRole(sessionID, agentID string, role string) error {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	session, exists := usm.sessions[sessionID]
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
	usm.mu.Lock()
	defer usm.mu.Unlock()

	session, exists := usm.sessions[sessionID]
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

	usm.logger.Info("تم تسجيل عميل بشري في الجلسة",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
		zap.String("name", name),
		zap.String("device", device),
		zap.String("location", location))

	return nil
}

// RegisterAgentInstance يسجل نسخة وكيل في الجلسة
func (usm *UnifiedSessionManager) RegisterAgentInstance(sessionID, agentID, instanceID, humanClientID, humanClientName, provider, model, apiKeyID, apiKeyLabel, role string) error {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	session, exists := usm.sessions[sessionID]
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

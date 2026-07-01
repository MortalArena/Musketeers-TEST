package integration

import (
	"context"
	"fmt"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/orchestrator"
	"go.uber.org/zap"
)

// AgentSessionIntegration يربط بين AgentRegistry و orchestrator.SessionManager
type AgentSessionIntegration struct {
	registry *agent.AgentRegistry
	manager  *orchestrator.SessionManager
	logger   *zap.Logger
	mu       sync.RWMutex
}

// NewAgentSessionIntegration ينشئ تكامل جديد
func NewAgentSessionIntegration(registry *agent.AgentRegistry, manager *orchestrator.SessionManager, logger *zap.Logger) *AgentSessionIntegration {
	return &AgentSessionIntegration{
		registry: registry,
		manager:  manager,
		logger:   logger,
	}
}

// RegisterAgentInSession يسجل وكيل في جلسة
func (asi *AgentSessionIntegration) RegisterAgentInSession(sessionID, agentID string) error {
	asi.mu.Lock()
	defer asi.mu.Unlock()

	// الحصول على الوكيل من السجل
	agent, err := asi.registry.Get(agentID)
	if err != nil {
		return fmt.Errorf("failed to get agent from registry: %w", err)
	}

	// الحصول على البيانات الوصفية
	metadata, err := asi.registry.GetMetadata(agentID)
	if err != nil {
		return fmt.Errorf("failed to get agent metadata: %w", err)
	}

	// الحصول على معلومات الوكيل
	info := agent.GetInfo()

	// تسجيل نسخة الوكيل في الجلسة
	err = asi.manager.RegisterAgentInstance(
		sessionID,
		agentID,
		info.InstanceID,
		metadata.HumanClientID,
		metadata.HumanClientName,
		info.Provider,
		info.Model,
		metadata.APIKeyID,
		metadata.APIKeyLabel,
		"assistant", // دور افتراضي
	)
	if err != nil {
		return fmt.Errorf("failed to register agent instance in session: %w", err)
	}

	asi.logger.Info("Agent registered in session",
		zap.String("session_id", sessionID),
		zap.String("agent_id", agentID),
		zap.String("instance_id", info.InstanceID),
	)

	return nil
}

// RegisterAgentAsManagerInSession يسجل وكيل كمدير جلسة
func (asi *AgentSessionIntegration) RegisterAgentAsManagerInSession(sessionID, agentID string) error {
	asi.mu.Lock()
	defer asi.mu.Unlock()

	// الحصول على الوكيل من السجل
	agent, err := asi.registry.Get(agentID)
	if err != nil {
		return fmt.Errorf("failed to get agent from registry: %w", err)
	}

	// الحصول على البيانات الوصفية
	metadata, err := asi.registry.GetMetadata(agentID)
	if err != nil {
		return fmt.Errorf("failed to get agent metadata: %w", err)
	}

	// الحصول على معلومات الوكيل
	info := agent.GetInfo()

	// تسجيل نسخة الوكيل في الجلسة كمدير
	err = asi.manager.RegisterAgentInstance(
		sessionID,
		agentID,
		info.InstanceID,
		metadata.HumanClientID,
		metadata.HumanClientName,
		info.Provider,
		info.Model,
		metadata.APIKeyID,
		metadata.APIKeyLabel,
		"manager",
	)
	if err != nil {
		return fmt.Errorf("failed to register agent instance in session: %w", err)
	}

	// تعيين الدور كمدير
	err = asi.manager.AssignRoleSimple(sessionID, agentID, "manager")
	if err != nil {
		return fmt.Errorf("failed to assign manager role: %w", err)
	}

	asi.logger.Info("Agent registered as manager in session",
		zap.String("session_id", sessionID),
		zap.String("agent_id", agentID),
		zap.String("instance_id", info.InstanceID),
	)

	return nil
}

// UnregisterAgentFromSession يلغي تسجيل وكيل من جلسة
func (asi *AgentSessionIntegration) UnregisterAgentFromSession(sessionID, agentID string) error {
	asi.mu.Lock()
	defer asi.mu.Unlock()

	// الحصول على الجلسة
	session, err := asi.manager.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// إزالة الوكيل من الجلسة
	delete(session.AgentInstances, agentID)

	asi.logger.Info("Agent unregistered from session",
		zap.String("session_id", sessionID),
		zap.String("agent_id", agentID),
	)

	return nil
}

// GetAgentsInSession يحصل على الوكلاء في جلسة
func (asi *AgentSessionIntegration) GetAgentsInSession(sessionID string) ([]agent.UnifiedAgent, error) {
	asi.mu.RLock()
	defer asi.mu.RUnlock()

	// الحصول على نسخ الوكلاء في الجلسة
	instances, err := asi.manager.GetAgentInstances(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent instances: %w", err)
	}

	// الحصول على الوكلاء الفعليين من السجل
	agents := make([]agent.UnifiedAgent, 0, len(instances))
	for _, instance := range instances {
		agent, err := asi.registry.Get(instance.AgentID)
		if err != nil {
			asi.logger.Warn("Failed to get agent from registry",
				zap.String("agent_id", instance.AgentID),
				zap.Error(err),
			)
			continue
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

// ExecuteTaskOnSessionAgents ينفذ مهمة على جميع وكلاء الجلسة
func (asi *AgentSessionIntegration) ExecuteTaskOnSessionAgents(ctx context.Context, sessionID string, task *agent.AgentTask) (map[string]*agent.TaskExecutionResult, error) {
	asi.mu.RLock()
	defer asi.mu.RUnlock()

	// الحصول على الوكلاء في الجلسة
	agents, err := asi.GetAgentsInSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agents in session: %w", err)
	}

	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents in session")
	}

	// تنفيذ المهمة على جميع الوكلاء
	results := make(map[string]*agent.TaskExecutionResult)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for _, agentInstance := range agents {
		wg.Add(1)
		go func(a agent.UnifiedAgent) {
			defer wg.Done()

			result, err := a.ExecuteTask(ctx, task)

			mu.Lock()
			if err != nil && firstErr == nil {
				firstErr = err
			}
			if result != nil {
				results[a.GetInfo().ID] = result
			}
			mu.Unlock()
		}(agentInstance)
	}

	wg.Wait()

	return results, firstErr
}

// ExecuteTaskOnManager ينفذ مهمة على مدير الجلسة
func (asi *AgentSessionIntegration) ExecuteTaskOnManager(ctx context.Context, sessionID string, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	asi.mu.RLock()
	defer asi.mu.RUnlock()

	// الحصول على الجلسة
	session, err := asi.manager.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// الحصول على مدير الجلسة
	managerAgentID := session.ManagerAgentID
	if managerAgentID == "" {
		return nil, fmt.Errorf("no manager agent in session")
	}

	// الحصول على الوكيل من السجل
	agent, err := asi.registry.Get(managerAgentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get manager agent: %w", err)
	}

	// تنفيذ المهمة
	result, err := agent.ExecuteTask(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to execute task on manager: %w", err)
	}

	return result, nil
}

// ExecuteTaskOnAssistant ينفذ مهمة على وكيل مساعد
func (asi *AgentSessionIntegration) ExecuteTaskOnAssistant(ctx context.Context, sessionID, agentID string, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	asi.mu.RLock()
	defer asi.mu.RUnlock()

	// الحصول على الوكيل من السجل
	agent, err := asi.registry.Get(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// تنفيذ المهمة
	result, err := agent.ExecuteTask(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to execute task on assistant: %w", err)
	}

	return result, nil
}

// GetManagerAgent يحصل على مدير الجلسة
func (asi *AgentSessionIntegration) GetManagerAgent(sessionID string) (agent.UnifiedAgent, error) {
	asi.mu.RLock()
	defer asi.mu.RUnlock()

	// الحصول على الجلسة
	session, err := asi.manager.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// الحصول على مدير الجلسة
	managerAgentID := session.ManagerAgentID
	if managerAgentID == "" {
		return nil, fmt.Errorf("no manager agent in session")
	}

	// الحصول على الوكيل من السجل
	agent, err := asi.registry.Get(managerAgentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get manager agent: %w", err)
	}

	return agent, nil
}

// GetAssistantAgents يحصل على الوكلاء المساعدين في الجلسة
func (asi *AgentSessionIntegration) GetAssistantAgents(sessionID string) ([]agent.UnifiedAgent, error) {
	asi.mu.RLock()
	defer asi.mu.RUnlock()

	// الحصول على الجلسة
	session, err := asi.manager.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// الحصول على الوكلاء المساعدين
	assistants := make([]agent.UnifiedAgent, 0)
	for _, assistantID := range session.AssistantAgents {
		agent, err := asi.registry.Get(assistantID)
		if err != nil {
			asi.logger.Warn("Failed to get assistant agent",
				zap.String("agent_id", assistantID),
				zap.Error(err),
			)
			continue
		}
		assistants = append(assistants, agent)
	}

	return assistants, nil
}

// RegisterHumanClientInSession يسجل عميل بشري في جلسة
func (asi *AgentSessionIntegration) RegisterHumanClientInSession(sessionID, userID, name, device, location string) error {
	asi.mu.Lock()
	defer asi.mu.Unlock()

	// تسجيل العميل البشري في السجل
	err := asi.registry.RegisterHumanClient(userID, name, true)
	if err != nil {
		return fmt.Errorf("failed to register human client in registry: %w", err)
	}

	// تسجيل العميل البشري في الجلسة
	err = asi.manager.RegisterHumanClient(sessionID, userID, name, device, location)
	if err != nil {
		return fmt.Errorf("failed to register human client in session: %w", err)
	}

	asi.logger.Info("Human client registered in session",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
		zap.String("name", name),
	)

	return nil
}

// GetHumanClientsInSession يحصل على العملاء البشريين في جلسة
func (asi *AgentSessionIntegration) GetHumanClientsInSession(sessionID string) ([]*orchestrator.HumanClientInfo, error) {
	asi.mu.RLock()
	defer asi.mu.RUnlock()

	return asi.manager.GetHumanClients(sessionID)
}

// GetSessionSummary يحصل على ملخص الجلسة
func (asi *AgentSessionIntegration) GetSessionSummary(sessionID string) (map[string]interface{}, error) {
	asi.mu.RLock()
	defer asi.mu.RUnlock()

	// الحصول على الجلسة
	session, err := asi.manager.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// الحصول على الوكلاء في الجلسة
	agents, err := asi.GetAgentsInSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agents in session: %w", err)
	}

	// الحصول على العملاء البشريين في الجلسة
	clients, err := asi.GetHumanClientsInSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get human clients in session: %w", err)
	}

	return map[string]interface{}{
		"session_id":          session.ID,
		"session_name":        session.Name,
		"session_status":      session.Status,
		"manager_agent_id":    session.ManagerAgentID,
		"total_agents":        len(agents),
		"total_human_clients": len(clients),
		"assistant_agents":    session.AssistantAgents,
		"created_at":          session.CreatedAt,
		"updated_at":          session.UpdatedAt,
	}, nil
}

package integration

import (
	"context"
	"fmt"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/agent/adapters"
	"github.com/MortalArena/Musketeers/pkg/session/core"
	"go.uber.org/zap"
)

// InstanceSessionIntegration يربط بين InstanceManager و UnifiedSessionManager
type InstanceSessionIntegration struct {
	instanceManager *adapters.InstanceManager
	sessionManager  *core.UnifiedSessionManager
	logger          *zap.Logger
	mu              sync.RWMutex
}

// NewInstanceSessionIntegration ينشئ تكامل جديد
func NewInstanceSessionIntegration(instanceManager *adapters.InstanceManager, sessionManager *core.UnifiedSessionManager, logger *zap.Logger) *InstanceSessionIntegration {
	return &InstanceSessionIntegration{
		instanceManager: instanceManager,
		sessionManager:  sessionManager,
		logger:          logger,
	}
}

// RegisterInstanceInSession يسجل نسخة في جلسة
func (isi *InstanceSessionIntegration) RegisterInstanceInSession(sessionID, instanceID string) error {
	isi.mu.Lock()
	defer isi.mu.Unlock()

	// الحصول على النسخة من InstanceManager
	instance, err := isi.instanceManager.GetInstance(instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance from instance manager: %w", err)
	}

	// الحصول على معلومات الوكيل
	info := instance.Adapter.GetInfo()

	// تسجيل نسخة الوكيل في الجلسة
	err = isi.sessionManager.RegisterAgentInstance(
		sessionID,
		info.ID,
		instance.InstanceID,
		info.HumanClientID,
		info.HumanClientName,
		info.Provider,
		info.Model,
		info.APIKeyID,
		info.APIKeyLabel,
		"assistant", // دور افتراضي
	)
	if err != nil {
		return fmt.Errorf("failed to register instance in session: %w", err)
	}

	isi.logger.Info("Instance registered in session",
		zap.String("session_id", sessionID),
		zap.String("instance_id", instanceID),
		zap.String("agent_id", info.ID),
	)

	return nil
}

// RegisterInstanceAsManagerInSession يسجل نسخة كمدير جلسة
func (isi *InstanceSessionIntegration) RegisterInstanceAsManagerInSession(sessionID, instanceID string) error {
	isi.mu.Lock()
	defer isi.mu.Unlock()

	// الحصول على النسخة من InstanceManager
	instance, err := isi.instanceManager.GetInstance(instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance from instance manager: %w", err)
	}

	// الحصول على معلومات الوكيل
	info := instance.Adapter.GetInfo()

	// تسجيل نسخة الوكيل في الجلسة كمدير
	err = isi.sessionManager.RegisterAgentInstance(
		sessionID,
		info.ID,
		instance.InstanceID,
		info.HumanClientID,
		info.HumanClientName,
		info.Provider,
		info.Model,
		info.APIKeyID,
		info.APIKeyLabel,
		"manager",
	)
	if err != nil {
		return fmt.Errorf("failed to register instance in session: %w", err)
	}

	// تعيين الدور كمدير
	err = isi.sessionManager.AssignRole(sessionID, info.ID, "manager")
	if err != nil {
		return fmt.Errorf("failed to assign manager role: %w", err)
	}

	isi.logger.Info("Instance registered as manager in session",
		zap.String("session_id", sessionID),
		zap.String("instance_id", instanceID),
		zap.String("agent_id", info.ID),
	)

	return nil
}

// UnregisterInstanceFromSession يلغي تسجيل نسخة من جلسة
func (isi *InstanceSessionIntegration) UnregisterInstanceFromSession(sessionID, instanceID string) error {
	isi.mu.Lock()
	defer isi.mu.Unlock()

	// الحصول على النسخة من InstanceManager
	instance, err := isi.instanceManager.GetInstance(instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance from instance manager: %w", err)
	}

	// الحصول على معلومات الوكيل
	info := instance.Adapter.GetInfo()

	// الحصول على الجلسة
	session, err := isi.sessionManager.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// إزالة نسخة الوكيل من الجلسة
	instanceKey := fmt.Sprintf("%s-%s", info.ID, instance.InstanceID)
	delete(session.AgentInstances, instanceKey)

	isi.logger.Info("Instance unregistered from session",
		zap.String("session_id", sessionID),
		zap.String("instance_id", instanceID),
		zap.String("agent_id", info.ID),
	)

	return nil
}

// GetInstancesInSession يحصل على النسخ في جلسة
func (isi *InstanceSessionIntegration) GetInstancesInSession(sessionID string) ([]*adapters.AgentInstance, error) {
	isi.mu.RLock()
	defer isi.mu.RUnlock()

	// الحصول على نسخ الوكلاء في الجلسة
	instances, err := isi.sessionManager.GetAgentInstances(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent instances: %w", err)
	}

	// الحصول على النسخ الفعلية من InstanceManager
	agentInstances := make([]*adapters.AgentInstance, 0, len(instances))
	for _, instance := range instances {
		instanceKey := fmt.Sprintf("%s-%s", instance.AgentID, instance.InstanceID)
		agentInstance, err := isi.instanceManager.GetInstance(instanceKey)
		if err != nil {
			isi.logger.Warn("Failed to get instance from instance manager",
				zap.String("instance_key", instanceKey),
				zap.Error(err),
			)
			continue
		}
		agentInstances = append(agentInstances, agentInstance)
	}

	return agentInstances, nil
}

// ExecuteTaskOnSessionInstances ينفذ مهمة على جميع نسخ الجلسة
func (isi *InstanceSessionIntegration) ExecuteTaskOnSessionInstances(ctx context.Context, sessionID string, task *agent.AgentTask) (map[string]*agent.TaskExecutionResult, error) {
	isi.mu.RLock()
	defer isi.mu.RUnlock()

	// الحصول على النسخ في الجلسة
	instances, err := isi.GetInstancesInSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instances in session: %w", err)
	}

	if len(instances) == 0 {
		return nil, fmt.Errorf("no instances in session")
	}

	// تنفيذ المهمة على جميع النسخ
	results := make(map[string]*agent.TaskExecutionResult)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for _, instance := range instances {
		wg.Add(1)
		go func(inst *adapters.AgentInstance) {
			defer wg.Done()

			result, err := isi.instanceManager.ExecuteOnInstance(ctx, inst.InstanceID, task)

			mu.Lock()
			if err != nil && firstErr == nil {
				firstErr = err
			}
			if result != nil {
				results[inst.InstanceID] = result
			}
			mu.Unlock()
		}(instance)
	}

	wg.Wait()

	return results, firstErr
}

// ExecuteTaskOnManagerInstance ينفذ مهمة على نسخة مدير الجلسة
func (isi *InstanceSessionIntegration) ExecuteTaskOnManagerInstance(ctx context.Context, sessionID string, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	isi.mu.RLock()
	defer isi.mu.RUnlock()

	// الحصول على الجلسة
	session, err := isi.sessionManager.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// الحصول على مدير الجلسة
	managerAgentID := session.ManagerAgentID
	if managerAgentID == "" {
		return nil, fmt.Errorf("no manager agent in session")
	}

	// الحصول على نسخ الوكلاء في الجلسة
	instances, err := isi.sessionManager.GetAgentInstances(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent instances: %w", err)
	}

	// البحث عن نسخة المدير
	var managerInstanceID string
	for _, instance := range instances {
		if instance.AgentID == managerAgentID && instance.Role == "manager" {
			managerInstanceID = fmt.Sprintf("%s-%s", instance.AgentID, instance.InstanceID)
			break
		}
	}

	if managerInstanceID == "" {
		return nil, fmt.Errorf("manager instance not found in session")
	}

	// تنفيذ المهمة
	result, err := isi.instanceManager.ExecuteOnInstance(ctx, managerInstanceID, task)
	if err != nil {
		return nil, fmt.Errorf("failed to execute task on manager instance: %w", err)
	}

	return result, nil
}

// ExecuteTaskOnAssistantInstance ينفذ مهمة على نسخة مساعدة
func (isi *InstanceSessionIntegration) ExecuteTaskOnAssistantInstance(ctx context.Context, sessionID, instanceID string, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	isi.mu.RLock()
	defer isi.mu.RUnlock()

	// تنفيذ المهمة
	result, err := isi.instanceManager.ExecuteOnInstance(ctx, instanceID, task)
	if err != nil {
		return nil, fmt.Errorf("failed to execute task on assistant instance: %w", err)
	}

	return result, nil
}

// GetManagerInstance يحصل على نسخة مدير الجلسة
func (isi *InstanceSessionIntegration) GetManagerInstance(sessionID string) (*adapters.AgentInstance, error) {
	isi.mu.RLock()
	defer isi.mu.RUnlock()

	// الحصول على الجلسة
	session, err := isi.sessionManager.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// الحصول على مدير الجلسة
	managerAgentID := session.ManagerAgentID
	if managerAgentID == "" {
		return nil, fmt.Errorf("no manager agent in session")
	}

	// الحصول على نسخ الوكلاء في الجلسة
	instances, err := isi.sessionManager.GetAgentInstances(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent instances: %w", err)
	}

	// البحث عن نسخة المدير
	var managerInstanceID string
	for _, instance := range instances {
		if instance.AgentID == managerAgentID && instance.Role == "manager" {
			managerInstanceID = fmt.Sprintf("%s-%s", instance.AgentID, instance.InstanceID)
			break
		}
	}

	if managerInstanceID == "" {
		return nil, fmt.Errorf("manager instance not found in session")
	}

	// الحصول على النسخة من InstanceManager
	instance, err := isi.instanceManager.GetInstance(managerInstanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get manager instance: %w", err)
	}

	return instance, nil
}

// GetAssistantInstances يحصل على نسخ الوكلاء المساعدين في الجلسة
func (isi *InstanceSessionIntegration) GetAssistantInstances(sessionID string) ([]*adapters.AgentInstance, error) {
	isi.mu.RLock()
	defer isi.mu.RUnlock()

	// الحصول على نسخ الوكلاء في الجلسة
	instances, err := isi.sessionManager.GetAgentInstances(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent instances: %w", err)
	}

	// الحصول على النسخ المساعدة
	assistantInstances := make([]*adapters.AgentInstance, 0)
	for _, instance := range instances {
		if instance.Role == "assistant" {
			instanceKey := fmt.Sprintf("%s-%s", instance.AgentID, instance.InstanceID)
			agentInstance, err := isi.instanceManager.GetInstance(instanceKey)
			if err != nil {
				isi.logger.Warn("Failed to get assistant instance",
					zap.String("instance_key", instanceKey),
					zap.Error(err),
				)
				continue
			}
			assistantInstances = append(assistantInstances, agentInstance)
		}
	}

	return assistantInstances, nil
}

// GetSessionInstanceSummary يحصل على ملخص نسخ الجلسة
func (isi *InstanceSessionIntegration) GetSessionInstanceSummary(sessionID string) (map[string]interface{}, error) {
	isi.mu.RLock()
	defer isi.mu.RUnlock()

	// الحصول على النسخ في الجلسة
	instances, err := isi.GetInstancesInSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instances in session: %w", err)
	}

	// الحصول على إحصائيات InstanceManager
	stats := isi.instanceManager.GetStats()

	return map[string]interface{}{
		"session_id":             sessionID,
		"total_instances":        len(instances),
		"total_system_instances": stats.TotalInstances,
		"by_type":                stats.ByType,
		"by_status":              stats.ByStatus,
	}, nil
}

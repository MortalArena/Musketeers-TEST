package adapters

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"go.uber.org/zap"
)

// AgentInstance - نسخة واحدة من الوكيل
type AgentInstance struct {
	InstanceID    string                 // معرف فريد للنسخة
	AgentType     string                 // "cli", "ide", "desktop", "api", "local", "browser", "custom"
	AgentName     string                 // "claude-code", "cursor", etc.
	Config        interface{}            // CLIConfig, IDEConfig, etc.
	Adapter       agent.UnifiedAgent     // الـ adapter الفعلي
	Status        string                 // "running", "stopped", "error"
	StartedAt     time.Time
	LastActivity  time.Time
	Metadata      map[string]interface{} // معلومات إضافية
	mu            sync.RWMutex
}

// InstanceManager - مدير النسخ المتعددة
type InstanceManager struct {
	instances map[string]*AgentInstance // instanceID -> instance
	byType    map[string][]string       // agentType -> []instanceID
	byName    map[string][]string       // agentName -> []instanceID
	logger    *zap.Logger
	mu        sync.RWMutex
}

// NewInstanceManager - إنشاء مدير النسخ
func NewInstanceManager(logger *zap.Logger) *InstanceManager {
	return &InstanceManager{
		instances: make(map[string]*AgentInstance),
		byType:    make(map[string][]string),
		byName:    make(map[string][]string),
		logger:    logger,
	}
}

// RegisterInstance - تسجيل نسخة جديدة
func (m *InstanceManager) RegisterInstance(instance *AgentInstance) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// التحقق من عدم التكرار
	if _, exists := m.instances[instance.InstanceID]; exists {
		return fmt.Errorf("instance %s already exists", instance.InstanceID)
	}

	// حفظ النسخة
	m.instances[instance.InstanceID] = instance

	// إضافة إلى الفهارس
	m.byType[instance.AgentType] = append(m.byType[instance.AgentType], instance.InstanceID)
	m.byName[instance.AgentName] = append(m.byName[instance.AgentName], instance.InstanceID)

	m.logger.Info("registered instance",
		zap.String("instance_id", instance.InstanceID),
		zap.String("type", instance.AgentType),
		zap.String("name", instance.AgentName),
	)

	return nil
}

// UnregisterInstance - إلغاء تسجيل نسخة
func (m *InstanceManager) UnregisterInstance(instanceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance, exists := m.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance %s not found", instanceID)
	}

	// حذف من الفهارس
	m.removeFromIndex(m.byType[instance.AgentType], instanceID)
	m.removeFromIndex(m.byName[instance.AgentName], instanceID)

	// حذف النسخة
	delete(m.instances, instanceID)

	m.logger.Info("unregistered instance", zap.String("instance_id", instanceID))
	return nil
}

// GetInstance - الحصول على نسخة محددة
func (m *InstanceManager) GetInstance(instanceID string) (*AgentInstance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instance, exists := m.instances[instanceID]
	if !exists {
		return nil, fmt.Errorf("instance %s not found", instanceID)
	}

	return instance, nil
}

// GetInstancesByType - الحصول على جميع النسخ من نوع معين
func (m *InstanceManager) GetInstancesByType(agentType string) []*AgentInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instanceIDs := m.byType[agentType]
	instances := make([]*AgentInstance, 0, len(instanceIDs))

	for _, id := range instanceIDs {
		if instance, exists := m.instances[id]; exists {
			instances = append(instances, instance)
		}
	}

	return instances
}

// GetInstancesByName - الحصول على جميع النسخ من اسم معين
func (m *InstanceManager) GetInstancesByName(agentName string) []*AgentInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instanceIDs := m.byName[agentName]
	instances := make([]*AgentInstance, 0, len(instanceIDs))

	for _, id := range instanceIDs {
		if instance, exists := m.instances[id]; exists {
			instances = append(instances, instance)
		}
	}

	return instances
}

// GetAllInstances - الحصول على جميع النسخ
func (m *InstanceManager) GetAllInstances() []*AgentInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instances := make([]*AgentInstance, 0, len(m.instances))
	for _, instance := range m.instances {
		instances = append(instances, instance)
	}

	return instances
}

// ExecuteOnInstance - تنفيذ مهمة على نسخة محددة
func (m *InstanceManager) ExecuteOnInstance(ctx context.Context, instanceID string, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	instance, err := m.GetInstance(instanceID)
	if err != nil {
		return nil, err
	}

	instance.mu.Lock()
	instance.Status = "running"
	instance.LastActivity = time.Now()
	instance.mu.Unlock()

	result, err := instance.Adapter.ExecuteTask(ctx, task)

	instance.mu.Lock()
	if err != nil {
		instance.Status = "error"
	} else {
		instance.Status = "stopped"
	}
	instance.LastActivity = time.Now()
	instance.mu.Unlock()

	return result, err
}

// ExecuteOnAllByType - تنفيذ مهمة على جميع النسخ من نوع معين
func (m *InstanceManager) ExecuteOnAllByType(ctx context.Context, agentType string, task *agent.AgentTask) (map[string]*agent.TaskExecutionResult, error) {
	instances := m.GetInstancesByType(agentType)
	if len(instances) == 0 {
		return nil, fmt.Errorf("no instances found for type: %s", agentType)
	}

	results := make(map[string]*agent.TaskExecutionResult)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for _, instance := range instances {
		wg.Add(1)
		go func(inst *AgentInstance) {
			defer wg.Done()

			result, err := m.ExecuteOnInstance(ctx, inst.InstanceID, task)

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

// removeFromIndex - إزالة من الفهرس
func (m *InstanceManager) removeFromIndex(slice []string, value string) []string {
	for i, v := range slice {
		if v == value {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// Stats - إحصائيات
type InstanceStats struct {
	TotalInstances int
	ByType         map[string]int
	ByStatus       map[string]int
}

// GetStats - الحصول على الإحصائيات
func (m *InstanceManager) GetStats() *InstanceStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &InstanceStats{
		TotalInstances: len(m.instances),
		ByType:         make(map[string]int),
		ByStatus:       make(map[string]int),
	}

	for _, instance := range m.instances {
		stats.ByType[instance.AgentType]++
		stats.ByStatus[instance.Status]++
	}

	return stats
}

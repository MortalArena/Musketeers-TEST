package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"go.uber.org/zap"
)

// MultiCLIAdapter - adapter يدعم عدة CLI agents في نفس الوقت
type MultiCLIAdapter struct {
	instanceManager *InstanceManager
	logger          *zap.Logger
}

// NewMultiCLIAdapter - إنشاء adapter متعدد النسخ
func NewMultiCLIAdapter(logger *zap.Logger) *MultiCLIAdapter {
	return &MultiCLIAdapter{
		instanceManager: NewInstanceManager(logger),
		logger:          logger,
	}
}

// AddCLIInstance - إضافة نسخة CLI جديدة
func (a *MultiCLIAdapter) AddCLIInstance(instanceID, agentName string, config *CLIConfig) error {
	// إنشاء adapter للنسخة باستخدام CLIAdapter الموجود
	adapter := NewCLIAdapter(config)
	adapter.SetLogger(a.logger)

	// إنشاء instance
	instance := &AgentInstance{
		InstanceID: instanceID,
		AgentType:  "cli",
		AgentName:  agentName,
		Config:     config,
		Adapter:    adapter,
		Status:     "stopped",
		StartedAt:  time.Now(),
		Metadata:   make(map[string]interface{}),
	}

	// تسجيل النسخة
	return a.instanceManager.RegisterInstance(instance)
}

// RemoveCLIInstance - إزالة نسخة CLI
func (a *MultiCLIAdapter) RemoveCLIInstance(instanceID string) error {
	return a.instanceManager.UnregisterInstance(instanceID)
}

// ExecuteOnCLI - تنفيذ مهمة على نسخة CLI محددة
func (a *MultiCLIAdapter) ExecuteOnCLI(ctx context.Context, instanceID string, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	return a.instanceManager.ExecuteOnInstance(ctx, instanceID, task)
}

// ExecuteOnAllCLI - تنفيذ مهمة على جميع نسخ CLI
func (a *MultiCLIAdapter) ExecuteOnAllCLI(ctx context.Context, task *agent.AgentTask) (map[string]*agent.TaskExecutionResult, error) {
	return a.instanceManager.ExecuteOnAllByType(ctx, "cli", task)
}

// GetAllCLIInstances - الحصول على جميع نسخ CLI
func (a *MultiCLIAdapter) GetAllCLIInstances() []*AgentInstance {
	return a.instanceManager.GetInstancesByType("cli")
}

// ExecuteTask - تنفيذ مهمة (interface implementation)
func (a *MultiCLIAdapter) ExecuteTask(ctx context.Context, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	// تنفيذ على جميع النسخ
	results, err := a.ExecuteOnAllCLI(ctx, task)
	if err != nil {
		return nil, err
	}

	// دمج النتائج
	return a.mergeResults(results), nil
}

// mergeResults - دمج نتائج عدة نسخ
func (a *MultiCLIAdapter) mergeResults(results map[string]*agent.TaskExecutionResult) *agent.TaskExecutionResult {
	if len(results) == 0 {
		return nil
	}

	// إذا كانت هناك نسخة واحدة فقط
	if len(results) == 1 {
		for _, result := range results {
			return result
		}
	}

	// دمج عدة نتائج
	merged := &agent.TaskExecutionResult{
		Success: true,
		Metrics: map[string]interface{}{
			"total_instances": len(results),
			"results":         results,
		},
	}

	// دمج الـ outputs
	var combinedOutput string
	for instanceID, result := range results {
		combinedOutput += fmt.Sprintf("\n=== Instance: %s ===\n%s\n", instanceID, result.Output)
	}
	merged.Output = combinedOutput

	return merged
}

// GetInfo - الحصول على معلومات الـ adapter
func (a *MultiCLIAdapter) GetInfo() *agent.AgentInfo {
	return &agent.AgentInfo{
		ID:            "multi-cli-adapter",
		Name:          "Multi-CLI Adapter",
		Type:          agent.AgentTypeCLI,
		Provider:      "local",
		Model:         "multi-cli",
		Version:       "1.0.0",
		Endpoint:      "",
		AuthMethod:    "none",
		MaxTokens:     4096,
		ContextWindow: 8192,
		CreatedAt:     time.Now(),
	}
}

// GetStatus - الحصول على حالة الـ adapter
func (a *MultiCLIAdapter) GetStatus() *agent.AgentStatus {
	instances := a.GetAllCLIInstances()
	stats := a.instanceManager.GetStats()

	return &agent.AgentStatus{
		IsAvailable:  len(instances) > 0,
		CurrentTask:  "",
		Load:         0,
		LastSeen:     time.Now(),
		ResponseTime: 200 * time.Millisecond,
		SuccessRate:  1.0,
		TotalTasks:   stats.TotalInstances,
		FailedTasks:  stats.ByStatus["error"],
	}
}

// IsAvailable - الحصول على مدى توفر الـ adapter
func (a *MultiCLIAdapter) IsAvailable() bool {
	instances := a.GetAllCLIInstances()
	return len(instances) > 0
}

// Close - إغلاق الـ adapter
func (a *MultiCLIAdapter) Close() error {
	instances := a.GetAllCLIInstances()
	for _, instance := range instances {
		if adapter, ok := instance.Adapter.(interface{ Close() error }); ok {
			adapter.Close()
		}
	}
	a.logger.Info("Multi-CLI adapter closed")
	return nil
}

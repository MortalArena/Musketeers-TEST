package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"go.uber.org/zap"
)

// MultiDesktopAdapter - adapter يدعم عدة Desktop apps في نفس الوقت
type MultiDesktopAdapter struct {
	instanceManager *InstanceManager
	logger          *zap.Logger
}

// NewMultiDesktopAdapter - إنشاء adapter متعدد النسخ
func NewMultiDesktopAdapter(logger *zap.Logger) *MultiDesktopAdapter {
	return &MultiDesktopAdapter{
		instanceManager: NewInstanceManager(logger),
		logger:          logger,
	}
}

// AddDesktopInstance - إضافة نسخة Desktop جديدة
func (a *MultiDesktopAdapter) AddDesktopInstance(instanceID, appName string, config *DesktopAppConfig) error {
	// إنشاء adapter للنسخة
	adapter, err := NewDesktopAppAdapter(config, a.logger)
	if err != nil {
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	// إنشاء instance
	instance := &AgentInstance{
		InstanceID: instanceID,
		AgentType:  "desktop",
		AgentName:  appName,
		Config:     config,
		Adapter:    adapter,
		Status:     "stopped",
		StartedAt:  time.Now(),
		Metadata:   make(map[string]interface{}),
	}

	// تسجيل النسخة
	return a.instanceManager.RegisterInstance(instance)
}

// RemoveDesktopInstance - إزالة نسخة Desktop
func (a *MultiDesktopAdapter) RemoveDesktopInstance(instanceID string) error {
	return a.instanceManager.UnregisterInstance(instanceID)
}

// ExecuteOnDesktop - تنفيذ مهمة على نسخة Desktop محددة
func (a *MultiDesktopAdapter) ExecuteOnDesktop(ctx context.Context, instanceID string, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	return a.instanceManager.ExecuteOnInstance(ctx, instanceID, task)
}

// ExecuteOnAllDesktop - تنفيذ مهمة على جميع نسخ Desktop
func (a *MultiDesktopAdapter) ExecuteOnAllDesktop(ctx context.Context, task *agent.AgentTask) (map[string]*agent.TaskExecutionResult, error) {
	return a.instanceManager.ExecuteOnAllByType(ctx, "desktop", task)
}

// GetAllDesktopInstances - الحصول على جميع نسخ Desktop
func (a *MultiDesktopAdapter) GetAllDesktopInstances() []*AgentInstance {
	return a.instanceManager.GetInstancesByType("desktop")
}

// ExecuteTask - تنفيذ مهمة (interface implementation)
func (a *MultiDesktopAdapter) ExecuteTask(ctx context.Context, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	// تنفيذ على جميع النسخ
	results, err := a.ExecuteOnAllDesktop(ctx, task)
	if err != nil {
		return nil, err
	}

	// دمج النتائج
	return a.mergeResults(results), nil
}

// mergeResults - دمج نتائج عدة نسخ
func (a *MultiDesktopAdapter) mergeResults(results map[string]*agent.TaskExecutionResult) *agent.TaskExecutionResult {
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
func (a *MultiDesktopAdapter) GetInfo() *agent.AgentInfo {
	return &agent.AgentInfo{
		ID:            "multi-desktop-adapter",
		Name:          "Multi-Desktop Adapter",
		Type:          agent.AgentTypeCustom,
		Provider:      "desktop",
		Model:         "multi-desktop",
		Version:       "1.0.0",
		Endpoint:      "",
		AuthMethod:    "none",
		MaxTokens:     4096,
		ContextWindow: 8192,
		CreatedAt:     time.Now(),
	}
}

// GetStatus - الحصول على حالة الـ adapter
func (a *MultiDesktopAdapter) GetStatus() *agent.AgentStatus {
	instances := a.GetAllDesktopInstances()
	stats := a.instanceManager.GetStats()

	return &agent.AgentStatus{
		IsAvailable:  len(instances) > 0,
		CurrentTask:  "",
		Load:         0,
		LastSeen:     time.Now(),
		ResponseTime: 300 * time.Millisecond,
		SuccessRate:  1.0,
		TotalTasks:   stats.TotalInstances,
		FailedTasks:  stats.ByStatus["error"],
	}
}

// IsAvailable - الحصول على مدى توفر الـ adapter
func (a *MultiDesktopAdapter) IsAvailable() bool {
	instances := a.GetAllDesktopInstances()
	return len(instances) > 0
}

// Close - إغلاق الـ adapter
func (a *MultiDesktopAdapter) Close() error {
	instances := a.GetAllDesktopInstances()
	for _, instance := range instances {
		if adapter, ok := instance.Adapter.(interface{ Close() error }); ok {
			adapter.Close()
		}
	}
	a.logger.Info("Multi-Desktop adapter closed")
	return nil
}

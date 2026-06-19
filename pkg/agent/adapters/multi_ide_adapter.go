package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"go.uber.org/zap"
)

// MultiIDEAdapter - adapter يدعم عدة IDEs ووكلاء في نفس الوقت
type MultiIDEAdapter struct {
	instanceManager *InstanceManager
	logger          *zap.Logger
}

// NewMultiIDEAdapter - إنشاء adapter متعدد النسخ
func NewMultiIDEAdapter(logger *zap.Logger) *MultiIDEAdapter {
	return &MultiIDEAdapter{
		instanceManager: NewInstanceManager(logger),
		logger:          logger,
	}
}

// AddIDEInstance - إضافة نسخة IDE جديدة
func (a *MultiIDEAdapter) AddIDEInstance(instanceID, ideType string, config *IDEConfig) error {
	// إنشاء adapter للنسخة باستخدام IDEAdapter الموجود
	adapter := NewIDEAdapter(config)
	adapter.SetLogger(a.logger)

	// إنشاء instance
	instance := &AgentInstance{
		InstanceID: instanceID,
		AgentType:  "ide",
		AgentName:  ideType,
		Config:     config,
		Adapter:    adapter,
		Status:     "stopped",
		StartedAt:  time.Now(),
		Metadata: map[string]interface{}{
			"ide_type": ideType,
		},
	}

	// تسجيل النسخة
	return a.instanceManager.RegisterInstance(instance)
}

// AddIDEExtensionInstance - إضافة نسخة extension داخل IDE
func (a *MultiIDEAdapter) AddIDEExtensionInstance(instanceID, ideType, extensionName string, config *IDEExtensionConfig) error {
	// إنشاء adapter للextension
	adapter, err := NewIDEExtensionAdapter(config, a.logger)
	if err != nil {
		return fmt.Errorf("failed to create extension adapter: %w", err)
	}

	// إنشاء instance
	instance := &AgentInstance{
		InstanceID: instanceID,
		AgentType:  "ide-extension",
		AgentName:  fmt.Sprintf("%s/%s", ideType, extensionName),
		Config:     config,
		Adapter:    adapter,
		Status:     "stopped",
		StartedAt:  time.Now(),
		Metadata: map[string]interface{}{
			"ide_type":       ideType,
			"extension_name": extensionName,
		},
	}

	// تسجيل النسخة
	return a.instanceManager.RegisterInstance(instance)
}

// RemoveIDEInstance - إزالة نسخة IDE
func (a *MultiIDEAdapter) RemoveIDEInstance(instanceID string) error {
	return a.instanceManager.UnregisterInstance(instanceID)
}

// ExecuteOnIDE - تنفيذ مهمة على نسخة IDE محددة
func (a *MultiIDEAdapter) ExecuteOnIDE(ctx context.Context, instanceID string, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	return a.instanceManager.ExecuteOnInstance(ctx, instanceID, task)
}

// ExecuteOnAllIDEs - تنفيذ مهمة على جميع نسخ IDEs
func (a *MultiIDEAdapter) ExecuteOnAllIDEs(ctx context.Context, task *agent.AgentTask) (map[string]*agent.TaskExecutionResult, error) {
	return a.instanceManager.ExecuteOnAllByType(ctx, "ide", task)
}

// ExecuteOnAllExtensions - تنفيذ مهمة على جميع extensions
func (a *MultiIDEAdapter) ExecuteOnAllExtensions(ctx context.Context, task *agent.AgentTask) (map[string]*agent.TaskExecutionResult, error) {
	return a.instanceManager.ExecuteOnAllByType(ctx, "ide-extension", task)
}

// GetAllIDEInstances - الحصول على جميع نسخ IDEs
func (a *MultiIDEAdapter) GetAllIDEInstances() []*AgentInstance {
	return a.instanceManager.GetInstancesByType("ide")
}

// GetAllExtensionInstances - الحصول على جميع نسخ extensions
func (a *MultiIDEAdapter) GetAllExtensionInstances() []*AgentInstance {
	return a.instanceManager.GetInstancesByType("ide-extension")
}

// GetExtensionsByIDE - الحصول على جميع extensions لـ IDE معين
func (a *MultiIDEAdapter) GetExtensionsByIDE(ideType string) []*AgentInstance {
	allExtensions := a.GetAllExtensionInstances()
	var filtered []*AgentInstance

	for _, ext := range allExtensions {
		if meta, ok := ext.Metadata["ide_type"].(string); ok && meta == ideType {
			filtered = append(filtered, ext)
		}
	}

	return filtered
}

// ExecuteTask - تنفيذ مهمة (interface implementation)
func (a *MultiIDEAdapter) ExecuteTask(ctx context.Context, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	// تنفيذ على جميع IDEs وextensions
	ideResults, _ := a.ExecuteOnAllIDEs(ctx, task)
	extResults, _ := a.ExecuteOnAllExtensions(ctx, task)

	// دمج النتائج
	allResults := make(map[string]*agent.TaskExecutionResult)
	for k, v := range ideResults {
		allResults[k] = v
	}
	for k, v := range extResults {
		allResults[k] = v
	}

	return a.mergeResults(allResults), nil
}

// mergeResults - دمج نتائج عدة نسخ
func (a *MultiIDEAdapter) mergeResults(results map[string]*agent.TaskExecutionResult) *agent.TaskExecutionResult {
	if len(results) == 0 {
		return nil
	}

	if len(results) == 1 {
		for _, result := range results {
			return result
		}
	}

	merged := &agent.TaskExecutionResult{
		Success: true,
		Metrics: map[string]interface{}{
			"total_instances": len(results),
			"results":         results,
		},
	}

	var combinedOutput string
	for instanceID, result := range results {
		combinedOutput += fmt.Sprintf("\n=== Instance: %s ===\n%s\n", instanceID, result.Output)
	}
	merged.Output = combinedOutput

	return merged
}

// GetInfo - الحصول على معلومات الـ adapter
func (a *MultiIDEAdapter) GetInfo() *agent.AgentInfo {
	return &agent.AgentInfo{
		ID:            "multi-ide-adapter",
		Name:          "Multi-IDE Adapter",
		Type:          agent.AgentTypeIDE,
		Provider:      "ide",
		Model:         "multi-ide",
		Version:       "1.0.0",
		Endpoint:      "",
		AuthMethod:    "none",
		MaxTokens:     4096,
		ContextWindow: 8192,
		CreatedAt:     time.Now(),
	}
}

// GetStatus - الحصول على حالة الـ adapter
func (a *MultiIDEAdapter) GetStatus() *agent.AgentStatus {
	ideInstances := a.GetAllIDEInstances()
	extInstances := a.GetAllExtensionInstances()
	stats := a.instanceManager.GetStats()

	return &agent.AgentStatus{
		IsAvailable:  len(ideInstances) > 0 || len(extInstances) > 0,
		CurrentTask:  "",
		Load:         0,
		LastSeen:     time.Now(),
		ResponseTime: 150 * time.Millisecond,
		SuccessRate:  1.0,
		TotalTasks:   stats.TotalInstances,
		FailedTasks:  stats.ByStatus["error"],
	}
}

// IsAvailable - الحصول على مدى توفر الـ adapter
func (a *MultiIDEAdapter) IsAvailable() bool {
	ideInstances := a.GetAllIDEInstances()
	extInstances := a.GetAllExtensionInstances()
	return len(ideInstances) > 0 || len(extInstances) > 0
}

// Close - إغلاق الـ adapter
func (a *MultiIDEAdapter) Close() error {
	ideInstances := a.GetAllIDEInstances()
	extInstances := a.GetAllExtensionInstances()

	for _, instance := range ideInstances {
		if adapter, ok := instance.Adapter.(interface{ Close() error }); ok {
			adapter.Close()
		}
	}

	for _, instance := range extInstances {
		if adapter, ok := instance.Adapter.(interface{ Close() error }); ok {
			adapter.Close()
		}
	}

	a.logger.Info("Multi-IDE adapter closed")
	return nil
}

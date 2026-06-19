package integration

import (
	"context"
	"fmt"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"go.uber.org/zap"
)

// TaskRouting يدير توجيه المهام إلى الوكلاء المناسبين
type TaskRouting struct {
	registry *agent.AgentRegistry
	logger   *zap.Logger
	mu       sync.RWMutex
}

// NewTaskRouting ينشئ نظام توجيه مهام جديد
func NewTaskRouting(registry *agent.AgentRegistry, logger *zap.Logger) *TaskRouting {
	return &TaskRouting{
		registry: registry,
		logger:   logger,
	}
}

// RouteTask يوجه مهمة إلى الوكلاء المناسبين
func (tr *TaskRouting) RouteTask(ctx context.Context, task *agent.AgentTask) (map[string]*agent.TaskExecutionResult, error) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	// الحصول على جميع الوكلاء المتاحين
	agents := tr.registry.ListAvailable()
	if len(agents) == 0 {
		return nil, fmt.Errorf("no available agents")
	}

	// توجيه المهمة إلى جميع الوكلاء المتاحين
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

// RouteTaskByRole يوجه مهمة حسب الدور
func (tr *TaskRouting) RouteTaskByRole(ctx context.Context, role AgentRole, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	// الحصول على جميع الوكلاء
	agents := tr.registry.ListAll()
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents available")
	}

	// البحث عن وكيل بالدور المطلوب
	// ملاحظة: في التنفيذ الحالي، نحتاج إلى تخزين معلومات الدور في مكان ما
	// هنا سنقوم باختيار أول وكيل متاح
	for _, agentInstance := range agents {
		if agentInstance.IsAvailable() {
			result, err := agentInstance.ExecuteTask(ctx, task)
			if err != nil {
				return nil, fmt.Errorf("failed to execute task: %w", err)
			}

			tr.logger.Info("Task routed by role",
				zap.String("role", string(role)),
				zap.String("agent_id", agentInstance.GetInfo().ID),
				zap.String("task_id", task.ID),
				zap.Bool("success", result.Success),
			)

			return result, nil
		}
	}

	return nil, fmt.Errorf("no available agent for role: %s", role)
}

// RouteTaskByCapability يوجه مهمة حسب القدرات
func (tr *TaskRouting) RouteTaskByCapability(ctx context.Context, capabilities []agent.AgentCapability, task *agent.AgentTask) (map[string]*agent.TaskExecutionResult, error) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	// البحث عن وكلاء لديهم القدرات المطلوبة
	agents := tr.registry.ListByCapability(capabilities[0])
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents with required capabilities")
	}

	// توجيه المهمة إلى الوكلاء الذين لديهم القدرات المطلوبة
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

	tr.logger.Info("Task routed by capability",
		zap.Int("required_capabilities", len(capabilities)),
		zap.Int("targeted_agents", len(agents)),
		zap.Int("results", len(results)),
	)

	return results, firstErr
}

// RouteTaskToBestAgent يوجه مهمة إلى أفضل وكيل
func (tr *TaskRouting) RouteTaskToBestAgent(ctx context.Context, requiredCapabilities []agent.AgentCapability, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	// البحث عن أفضل وكيل
	agent, err := tr.registry.FindBestAgent(requiredCapabilities)
	if err != nil {
		return nil, fmt.Errorf("failed to find best agent: %w", err)
	}

	// تنفيذ المهمة على أفضل وكيل
	result, err := agent.ExecuteTask(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to execute task on best agent: %w", err)
	}

	tr.logger.Info("Task routed to best agent",
		zap.String("agent_id", agent.GetInfo().ID),
		zap.String("task_id", task.ID),
		zap.Bool("success", result.Success),
	)

	return result, nil
}

// RouteTaskByType يوجه مهمة حسب نوع الوكيل
func (tr *TaskRouting) RouteTaskByType(ctx context.Context, agentType agent.AgentType, task *agent.AgentTask) (map[string]*agent.TaskExecutionResult, error) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	// الحصول على الوكلاء حسب النوع
	agents := tr.registry.ListByType(agentType)
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents of type: %s", agentType)
	}

	// توجيه المهمة إلى الوكلاء من النوع المطلوب
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

	tr.logger.Info("Task routed by type",
		zap.String("agent_type", string(agentType)),
		zap.Int("targeted_agents", len(agents)),
		zap.Int("results", len(results)),
	)

	return results, firstErr
}

// MergeResults يدمج نتائج عدة وكلاء
func (tr *TaskRouting) MergeResults(results map[string]*agent.TaskExecutionResult) *agent.TaskExecutionResult {
	if len(results) == 0 {
		return nil
	}

	// إذا كانت هناك نتيجة واحدة فقط
	if len(results) == 1 {
		for _, result := range results {
			return result
		}
	}

	// دمج عدة نتائج
	merged := &agent.TaskExecutionResult{
		Success: true,
		Metrics: map[string]interface{}{
			"total_agents": len(results),
			"results":      results,
		},
	}

	// دمج الـ outputs
	var combinedOutput string
	var totalSuccess int
	for agentID, result := range results {
		if result.Success {
			totalSuccess++
		}
		combinedOutput += fmt.Sprintf("\n=== Agent: %s ===\n%s\n", agentID, result.Output)
	}
	merged.Output = combinedOutput

	// تحديث النجاح الإجمالي
	merged.Success = totalSuccess > 0
	merged.Metrics["total_success"] = totalSuccess

	return merged
}

// RouteTaskWithStrategy يوجه مهمة باستخدام استراتيجية معينة
func (tr *TaskRouting) RouteTaskWithStrategy(ctx context.Context, strategy string, task *agent.AgentTask) (map[string]*agent.TaskExecutionResult, error) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	switch strategy {
	case "all":
		return tr.RouteTask(ctx, task)
	case "best":
		result, err := tr.RouteTaskToBestAgent(ctx, task.Inputs["capabilities"].([]agent.AgentCapability), task)
		if err != nil {
			return nil, err
		}
		return map[string]*agent.TaskExecutionResult{"best": result}, nil
	case "capability":
		return tr.RouteTaskByCapability(ctx, task.Inputs["capabilities"].([]agent.AgentCapability), task)
	case "type":
		return tr.RouteTaskByType(ctx, task.Inputs["agent_type"].(agent.AgentType), task)
	default:
		return nil, fmt.Errorf("unknown routing strategy: %s", strategy)
	}
}

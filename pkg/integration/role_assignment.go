package integration

import (
	"context"
	"fmt"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"go.uber.org/zap"
)

// RoleAssignment يدير تعيين الأدوار الفعلي للوكلاء
type RoleAssignment struct {
	registry *agent.AgentRegistry
	logger   *zap.Logger
	mu       sync.RWMutex
}

// NewRoleAssignment ينشئ نظام تعيين أدوار جديد
func NewRoleAssignment(registry *agent.AgentRegistry, logger *zap.Logger) *RoleAssignment {
	return &RoleAssignment{
		registry: registry,
		logger:   logger,
	}
}

// AgentRole دور الوكيل
type AgentRole string

const (
	RoleManager    AgentRole = "manager"    // مدير الجلسة - يدير الجلسة ويوزع المهام
	RoleAssistant  AgentRole = "assistant"  // مساعد - ينفذ المهام
	RoleObserver   AgentRole = "observer"   // مراقب - يراقب الجلسة والوكلاء
	RoleSpecialist AgentRole = "specialist" // متخصص - متخصص في مجال معين
)

// AgentRoleInfo معلومات دور الوكيل
type AgentRoleInfo struct {
	AgentID        string                  `json:"agent_id"`
	Role           AgentRole               `json:"role"`
	Capabilities   []agent.AgentCapability `json:"capabilities"`
	Specialization string                  `json:"specialization"` // التخصص (للمتخصصين)
	AssignedAt     string                  `json:"assigned_at"`
}

// AssignRole assigns a role to an agent
func (ra *RoleAssignment) AssignRole(agentID string, role AgentRole, specialization string) error {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	// الحصول على الوكيل
	agent, err := ra.registry.Get(agentID)
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// التحقق من القدرات المطلوبة للدور
	capabilities := agent.GetCapabilities()
	if !ra.validateRoleCapabilities(role, capabilities) {
		return fmt.Errorf("agent does not have required capabilities for role: %s", role)
	}

	ra.logger.Info("Role assigned to agent",
		zap.String("agent_id", agentID),
		zap.String("role", string(role)),
		zap.String("specialization", specialization),
	)

	return nil
}

// validateRoleCapabilities يتحقق من أن الوكيل لديه القدرات المطلوبة للدور
func (ra *RoleAssignment) validateRoleCapabilities(role AgentRole, capabilities []agent.AgentCapability) bool {
	switch role {
	case RoleManager:
		// المدير يحتاج إلى قدرات إدارية وتحليلية
		required := []agent.AgentCapability{
			agent.CapabilityAnalysis,
			agent.CapabilityCodeGeneration,
		}
		return ra.hasCapabilities(capabilities, required)
	case RoleAssistant:
		// المساعد يحتاج إلى قدرات تنفيذية
		required := []agent.AgentCapability{
			agent.CapabilityCodeGeneration,
			agent.CapabilityCodeReview,
		}
		return ra.hasCapabilities(capabilities, required)
	case RoleObserver:
		// المراقب يحتاج إلى قدرات تحليلية
		required := []agent.AgentCapability{
			agent.CapabilityAnalysis,
		}
		return ra.hasCapabilities(capabilities, required)
	case RoleSpecialist:
		// المتخصص يحتاج إلى قدرات محددة حسب التخصص
		return len(capabilities) > 0
	default:
		return false
	}
}

// hasCapabilities يتحقق من أن الوكيل لديه القدرات المطلوبة
func (ra *RoleAssignment) hasCapabilities(has, required []agent.AgentCapability) bool {
	requiredMap := make(map[agent.AgentCapability]bool)
	for _, cap := range required {
		requiredMap[cap] = true
	}

	for _, cap := range has {
		delete(requiredMap, cap)
	}

	return len(requiredMap) == 0
}

// ExecuteTaskAsManager ينفذ مهمة كمدير
func (ra *RoleAssignment) ExecuteTaskAsManager(ctx context.Context, agentID string, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	// الحصول على الوكيل
	agent, err := ra.registry.Get(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get manager agent: %w", err)
	}

	// تنفيذ المهمة
	result, err := agent.ExecuteTask(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to execute task as manager: %w", err)
	}

	ra.logger.Info("Task executed as manager",
		zap.String("agent_id", agentID),
		zap.String("task_id", task.ID),
		zap.Bool("success", result.Success),
	)

	return result, nil
}

// ExecuteTaskAsAssistant ينفذ مهمة كمساعد
func (ra *RoleAssignment) ExecuteTaskAsAssistant(ctx context.Context, agentID string, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	// الحصول على الوكيل
	agent, err := ra.registry.Get(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assistant agent: %w", err)
	}

	// تنفيذ المهمة
	result, err := agent.ExecuteTask(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to execute task as assistant: %w", err)
	}

	ra.logger.Info("Task executed as assistant",
		zap.String("agent_id", agentID),
		zap.String("task_id", task.ID),
		zap.Bool("success", result.Success),
	)

	return result, nil
}

// ExecuteTaskAsObserver ينفذ مهمة كمراقب
func (ra *RoleAssignment) ExecuteTaskAsObserver(ctx context.Context, agentID string, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	// الحصول على الوكيل
	agent, err := ra.registry.Get(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get observer agent: %w", err)
	}

	// تنفيذ المهمة (المراقب يراقب ولا ينفذ)
	result, err := agent.ExecuteTask(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to execute task as observer: %w", err)
	}

	ra.logger.Info("Task executed as observer",
		zap.String("agent_id", agentID),
		zap.String("task_id", task.ID),
		zap.Bool("success", result.Success),
	)

	return result, nil
}

// ExecuteTaskAsSpecialist ينفذ مهمة كمتخصص
func (ra *RoleAssignment) ExecuteTaskAsSpecialist(ctx context.Context, agentID, specialization string, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	// الحصول على الوكيل
	agent, err := ra.registry.Get(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get specialist agent: %w", err)
	}

	// تنفيذ المهمة
	result, err := agent.ExecuteTask(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to execute task as specialist: %w", err)
	}

	ra.logger.Info("Task executed as specialist",
		zap.String("agent_id", agentID),
		zap.String("specialization", specialization),
		zap.String("task_id", task.ID),
		zap.Bool("success", result.Success),
	)

	return result, nil
}

// GetAgentsByRole يحصل على الوكلاء حسب الدور
func (ra *RoleAssignment) GetAgentsByRole(role AgentRole) ([]agent.UnifiedAgent, error) {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	// الحصول على جميع الوكلاء
	agents := ra.registry.ListAll()

	// تصفية الوكلاء حسب الدور
	// ملاحظة: في التنفيذ الحالي، نحتاج إلى تخزين معلومات الدور في مكان ما
	// هنا سنقوم بإرجاع جميع الوكلاء لأننا لا نملك نظام تخزين للأدوار حالياً
	// في التنفيذ الكامل، سنحتاج إلى إضافة حقل الدور إلى AgentMetadata

	return agents, nil
}

// GetBestAgentForRole يحصل على أفضل وكيل لدور معين
func (ra *RoleAssignment) GetBestAgentForRole(role AgentRole, requiredCapabilities []agent.AgentCapability) (agent.UnifiedAgent, error) {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	// البحث عن أفضل وكيل حسب القدرات
	agent, err := ra.registry.FindBestAgent(requiredCapabilities)
	if err != nil {
		return nil, fmt.Errorf("failed to find best agent for role: %w", err)
	}

	return agent, nil
}

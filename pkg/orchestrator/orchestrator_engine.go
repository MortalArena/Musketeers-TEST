package orchestrator

import (
	"context"
	"fmt"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/agent/unified"
	"github.com/MortalArena/Musketeers/pkg/verification"
	"go.uber.org/zap"
)

// CapabilityMatcher يطابق القدرات المطلوبة مع الوكلاء
type CapabilityMatcher struct {
	agentCapabilities map[string][]agent.AgentCapability
	capabilityAgents  map[agent.AgentCapability][]string
	mu                sync.RWMutex
}

// NewCapabilityMatcher ينشئ مطابق قدرات جديد
func NewCapabilityMatcher() *CapabilityMatcher {
	return &CapabilityMatcher{
		agentCapabilities: make(map[string][]agent.AgentCapability),
		capabilityAgents:  make(map[agent.AgentCapability][]string),
	}
}

// RegisterCapabilities يسجل قدرات وكيل
func (cm *CapabilityMatcher) RegisterCapabilities(agentID string, capabilities []agent.AgentCapability) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.agentCapabilities[agentID] = capabilities
	for _, cap := range capabilities {
		cm.capabilityAgents[cap] = append(cm.capabilityAgents[cap], agentID)
	}
}

// FindBestAgent يجد أفضل وكيل للقدرات المطلوبة
func (cm *CapabilityMatcher) FindBestAgent(requiredCapabilities []agent.AgentCapability) (string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// البحث عن وكلاء لديهم جميع القدرات المطلوبة
	candidates := make(map[string]int)
	for _, reqCap := range requiredCapabilities {
		agents, exists := cm.capabilityAgents[reqCap]
		if !exists {
			continue
		}
		for _, agentID := range agents {
			candidates[agentID]++
		}
	}

	// اختيار الوكلاء الذين لديهم جميع القدرات
	var bestAgent string
	maxMatches := 0
	for agentID, matches := range candidates {
		if matches == len(requiredCapabilities) && matches > maxMatches {
			bestAgent = agentID
			maxMatches = matches
		}
	}

	if bestAgent == "" {
		return "", fmt.Errorf("no agent found with required capabilities")
	}

	return bestAgent, nil
}

// GetAgentsByCapability يحصل على وكلاء حسب قدرة
func (cm *CapabilityMatcher) GetAgentsByCapability(capability agent.AgentCapability) []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.capabilityAgents[capability]
}

// OrchestratorEngine محرك التنسيق - ينسق جميع مكونات النظام
type OrchestratorEngine struct {
	registry          *agent.AgentRegistry
	lifecycleManager  *AgentLifecycleManager
	roleAssigner      *RoleAssigner
	verifier          *verification.MultiStageVerifier
	capabilityMatcher *CapabilityMatcher
	unifiedAgent      *unified.UnifiedAgent // مرجع للتكامل مع UnifiedAgent
	logger            *zap.Logger
	mu                sync.RWMutex
	running           bool
}

// NewOrchestratorEngine ينشئ محرك تنسيق جديد
func NewOrchestratorEngine(registry *agent.AgentRegistry) *OrchestratorEngine {
	return &OrchestratorEngine{
		registry:          registry,
		lifecycleManager:  NewAgentLifecycleManager(registry),
		roleAssigner:      NewRoleAssigner(registry),
		verifier:          verification.NewMultiStageVerifier(),
		capabilityMatcher: NewCapabilityMatcher(),
		logger:            zap.NewNop(),
		running:           false,
	}
}

// SetLogger يضبط logger
func (oe *OrchestratorEngine) SetLogger(logger *zap.Logger) {
	oe.mu.Lock()
	defer oe.mu.Unlock()
	oe.logger = logger
	oe.lifecycleManager.SetLogger(logger)
	oe.roleAssigner.SetLogger(logger)
	oe.verifier.SetLogger(logger)
}

// SetUnifiedAgent يضبط مرجع UnifiedAgent للتكامل
func (oe *OrchestratorEngine) SetUnifiedAgent(ua *unified.UnifiedAgent) {
	oe.mu.Lock()
	defer oe.mu.Unlock()
	oe.unifiedAgent = ua
	oe.logger.Info("تم ضبط UnifiedAgent في OrchestratorEngine")
}

// GetUnifiedAgent يرجع مرجع UnifiedAgent
func (oe *OrchestratorEngine) GetUnifiedAgent() *unified.UnifiedAgent {
	oe.mu.RLock()
	defer oe.mu.RUnlock()
	return oe.unifiedAgent
}

// Start يبدأ المحرك
func (oe *OrchestratorEngine) Start(ctx context.Context) error {
	oe.mu.Lock()
	defer oe.mu.Unlock()

	if oe.running {
		return fmt.Errorf("orchestrator engine is already running")
	}

	oe.running = true

	// تسجيل المدخلات الافتراضية للتحقق
	oe.verifier.RegisterVerifier(verification.NewDefaultSyntaxVerifier())
	oe.verifier.RegisterVerifier(verification.NewDefaultSemanticsVerifier())
	oe.verifier.RegisterVerifier(verification.NewDefaultSecurityVerifier())
	oe.verifier.RegisterVerifier(verification.NewDefaultPerformanceVerifier())
	oe.verifier.RegisterVerifier(verification.NewDefaultIntegrationVerifier())

	oe.logger.Info("Orchestrator engine started")

	return nil
}

// Stop يوقف المحرك
func (oe *OrchestratorEngine) Stop(ctx context.Context) error {
	oe.mu.Lock()
	defer oe.mu.Unlock()

	if !oe.running {
		return fmt.Errorf("orchestrator engine is not running")
	}

	oe.running = false

	// إيقاف جميع الوكلاء
	agents := oe.registry.ListAll()
	for _, agent := range agents {
		agentID := agent.GetInfo().ID
		if err := oe.lifecycleManager.StopAgent(ctx, agentID); err != nil {
			oe.logger.Error("Failed to stop agent",
				zap.String("agent_id", agentID),
				zap.Error(err),
			)
		}
	}

	oe.logger.Info("Orchestrator engine stopped")

	return nil
}

// IsRunning يعيد ما إذا كان المحرك يعمل
func (oe *OrchestratorEngine) IsRunning() bool {
	oe.mu.RLock()
	defer oe.mu.RUnlock()
	return oe.running
}

// ExecuteTask ينفذ مهمة باستخدام أفضل وكيل متاح
func (oe *OrchestratorEngine) ExecuteTask(ctx context.Context, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	oe.mu.RLock()
	defer oe.mu.RUnlock()

	if !oe.running {
		return nil, fmt.Errorf("orchestrator engine is not running")
	}

	// تحديد القدرات المطلوبة للمهمة
	requiredCapabilities := oe.getRequiredCapabilities(task)

	// البحث عن أفضل وكيل
	bestAgentObj, err := oe.registry.FindBestAgent(requiredCapabilities)
	if err != nil {
		return nil, fmt.Errorf("failed to find suitable agent: %w", err)
	}

	bestAgentID := bestAgentObj.GetInfo().ID

	// تنفيذ المهمة
	result, err := bestAgentObj.ExecuteTask(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to execute task: %w", err)
	}

	// تحديث إحصائيات الوكيل
	tokensUsed := 0
	if result.Metrics != nil {
		if val, ok := result.Metrics["tokens"].(int); ok {
			tokensUsed = val
		}
	}
	oe.registry.UpdateStats(bestAgentID, result.Success, tokensUsed, result.Duration)

	oe.logger.Info("Task executed",
		zap.String("task_id", task.ID),
		zap.String("agent_id", bestAgentID),
		zap.Bool("success", result.Success),
		zap.Duration("duration", result.Duration),
	)

	return result, nil
}

// ExecuteTaskWithRole ينفذ مهمة باستخدام وكيل بدور محدد
func (oe *OrchestratorEngine) ExecuteTaskWithRole(ctx context.Context, task *agent.AgentTask, role AgentRole) (*agent.TaskExecutionResult, error) {
	oe.mu.RLock()
	defer oe.mu.RUnlock()

	if !oe.running {
		return nil, fmt.Errorf("orchestrator engine is not running")
	}

	// تحديد القدرات المطلوبة للمهمة
	requiredCapabilities := oe.getRequiredCapabilities(task)

	// الحصول على أفضل وكيل للدور المحدد
	agentID, err := oe.roleAssigner.GetBestAgentForRole(role, requiredCapabilities)
	if err != nil {
		return nil, fmt.Errorf("failed to find agent for role %s: %w", role, err)
	}

	// الحصول على الوكيل
	agentObj, err := oe.registry.Get(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// تنفيذ المهمة
	result, err := agentObj.ExecuteTask(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to execute task: %w", err)
	}

	// تحديث إحصائيات الوكيل
	tokensUsed := 0
	if result.Metrics != nil {
		if val, ok := result.Metrics["tokens"].(int); ok {
			tokensUsed = val
		}
	}
	oe.registry.UpdateStats(agentID, result.Success, tokensUsed, result.Duration)

	oe.logger.Info("Task executed with role",
		zap.String("task_id", task.ID),
		zap.String("agent_id", agentID),
		zap.String("role", string(role)),
		zap.Bool("success", result.Success),
		zap.Duration("duration", result.Duration),
	)

	return result, nil
}

// VerifyResult يتحقق من نتيجة مهمة
func (oe *OrchestratorEngine) VerifyResult(ctx context.Context, taskID string, agentID string, output string) ([]*verification.VerificationResult, error) {
	oe.mu.RLock()
	defer oe.mu.RUnlock()

	if !oe.running {
		return nil, fmt.Errorf("orchestrator engine is not running")
	}

	request := &verification.VerificationRequest{
		TaskID:  taskID,
		AgentID: agentID,
		Output:  output,
		Stages:  []verification.VerificationStage{},
	}

	results, err := oe.verifier.Verify(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("verification failed: %w", err)
	}

	overallScore := oe.verifier.GetOverallScore(results)
	oe.logger.Info("Result verified",
		zap.String("task_id", taskID),
		zap.String("agent_id", agentID),
		zap.Float64("overall_score", overallScore),
	)

	return results, nil
}

// RegisterAgent يسجل وكيل جديد
func (oe *OrchestratorEngine) RegisterAgent(agentObj agent.UnifiedAgent, metadata *agent.AgentMetadata) error {
	oe.mu.Lock()
	defer oe.mu.Unlock()

	if !oe.running {
		return fmt.Errorf("orchestrator engine is not running")
	}

	// تسجيل الوكيل في السجل
	if err := oe.registry.Register(agentObj, metadata); err != nil {
		return err
	}

	// تهيئة الوكيل في مدير دورة الحياة
	oe.lifecycleManager.InitializeAgent(agentObj.GetInfo().ID)

	// اقتراح دور للوكيل
	role, err := oe.roleAssigner.SuggestRole(agentObj.GetInfo().ID)
	if err != nil {
		oe.logger.Warn("Failed to suggest role for agent",
			zap.String("agent_id", agentObj.GetInfo().ID),
			zap.Error(err),
		)
	} else {
		// تعيين الدور المقترح
		if err := oe.roleAssigner.AssignRole(agentObj.GetInfo().ID, role, 1.0); err != nil {
			oe.logger.Warn("Failed to assign suggested role",
				zap.String("agent_id", agentObj.GetInfo().ID),
				zap.String("role", string(role)),
				zap.Error(err),
			)
		}
	}

	oe.logger.Info("Agent registered in orchestrator",
		zap.String("agent_id", agentObj.GetInfo().ID),
		zap.String("suggested_role", string(role)),
	)

	return nil
}

// UnregisterAgent يلغي تسجيل وكيل
func (oe *OrchestratorEngine) UnregisterAgent(ctx context.Context, agentID string) error {
	oe.mu.Lock()
	defer oe.mu.Unlock()

	if !oe.running {
		return fmt.Errorf("orchestrator engine is not running")
	}

	// إيقاف الوكيل
	if err := oe.lifecycleManager.StopAgent(ctx, agentID); err != nil {
		oe.logger.Error("Failed to stop agent",
			zap.String("agent_id", agentID),
			zap.Error(err),
		)
	}

	// إلغاء تسجيل الوكيل من السجل
	if err := oe.registry.Unregister(agentID); err != nil {
		return err
	}

	// إزالة الوكيل من مدير دورة الحياة
	oe.lifecycleManager.RemoveAgent(agentID)

	oe.logger.Info("Agent unregistered from orchestrator",
		zap.String("agent_id", agentID),
	)

	return nil
}

// GetStats يحصل على إحصائيات المحرك
func (oe *OrchestratorEngine) GetStats() map[string]interface{} {
	oe.mu.RLock()
	defer oe.mu.RUnlock()

	lifecycleStats := oe.lifecycleManager.GetStats()
	registryStats := map[string]interface{}{
		"total_agents":     oe.registry.GetCount(),
		"available_agents": oe.registry.GetAvailableCount(),
	}

	return map[string]interface{}{
		"running":          oe.running,
		"lifecycle_stats":  lifecycleStats,
		"registry_stats":   registryStats,
		"role_assignments": oe.roleAssigner.GetRoleCount(),
	}
}

// getRequiredCapabilities يحدد القدرات المطلوبة للمهمة
func (oe *OrchestratorEngine) getRequiredCapabilities(task *agent.AgentTask) []agent.AgentCapability {
	// في التطبيق الحقيقي، سيتم تحليل المهمة لتحديد القدرات المطلوبة
	// هنا نستخدم قائمة افتراضية
	return []agent.AgentCapability{
		agent.CapabilityCodeGeneration,
		agent.CapabilityCodeReview,
	}
}

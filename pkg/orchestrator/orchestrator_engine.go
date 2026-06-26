package orchestrator

import (
	"context"
	"fmt"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/agent/unified"
	"github.com/MortalArena/Musketeers/pkg/capability"
	capgithub "github.com/MortalArena/Musketeers/pkg/capability/github"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/MortalArena/Musketeers/pkg/policy"
	"github.com/MortalArena/Musketeers/pkg/session"
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
	mcpManager        *MCPManager
	connector         *Connector
	capabilityManager *capability.Manager
	eventBus          *eventbus.EventBus
	policyEngine      *policy.Engine
	unifiedAgent      *unified.UnifiedAgent // مرجع للتكامل مع UnifiedAgent
	sessionContainer  *session.SessionContainer // [NEW] مرجع للجلسة لمزامنة قدرات الوكلاء المحققة
	logger            *zap.Logger
	mu                sync.RWMutex
	running           bool
}

// NewOrchestratorEngine ينشئ محرك تنسيق جديد
func NewOrchestratorEngine(registry *agent.AgentRegistry) *OrchestratorEngine {
	// إنشاء EventBus
	evBus := eventbus.NewEventBus()

	// إنشاء مدير القدرات مع القدرات الحقيقية
	polEng := policy.NewEngine()
	capMgr := capability.NewManager(polEng)
	githubCap := capgithub.NewGitHubCapability("")
	capMgr.Register(githubCap)

	// إنشاء MCPManager مع EventBus وربطه بـ CapabilityManager
	logger := zap.NewNop()
	mcpMgr := NewMCPManager(evBus, logger)
	mcpMgr.SetCapabilityManager(capMgr)

	return &OrchestratorEngine{
		registry:          registry,
		lifecycleManager:  NewAgentLifecycleManager(registry),
		roleAssigner:      NewRoleAssigner(registry),
		verifier:          verification.NewMultiStageVerifier(),
		capabilityMatcher: NewCapabilityMatcher(),
		mcpManager:        mcpMgr,
		capabilityManager: capMgr,
		eventBus:          evBus,
		policyEngine:      polEng,
		logger:            logger,
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

// SetSessionContainer يضبط مرجع SessionContainer لمزامنة قدرات الوكلاء
func (oe *OrchestratorEngine) SetSessionContainer(sc *session.SessionContainer) {
	oe.mu.Lock()
	defer oe.mu.Unlock()
	oe.sessionContainer = sc
	oe.logger.Info("تم ضبط SessionContainer في OrchestratorEngine")
}

// SetConnector يضبط Connector System في OrchestratorEngine
func (oe *OrchestratorEngine) SetConnector(c *Connector) {
	oe.mu.Lock()
	defer oe.mu.Unlock()
	oe.connector = c
	oe.logger.Info("تم ضبط Connector في OrchestratorEngine")
}

// SetPolicyMode يضبط وضع الـ Policy للـ Capability Manager
func (oe *OrchestratorEngine) SetPolicyMode(mode capability.PolicyMode) {
	oe.capabilityManager.SetPolicyMode(mode)
	oe.logger.Info("تم ضبط وضع الـ Policy",
		zap.Int("mode", int(mode)),
	)
}

// PolicyEngine يرجع Policy Engine الخاص بـ OrchestratorEngine لإضافة القواعد
func (oe *OrchestratorEngine) PolicyEngine() *policy.Engine {
	return oe.policyEngine
}

// GetSessionContainer يرجع SessionContainer
func (oe *OrchestratorEngine) GetSessionContainer() *session.SessionContainer {
	oe.mu.RLock()
	defer oe.mu.RUnlock()
	return oe.sessionContainer
}

// GetConnector يرجع Connector System
func (oe *OrchestratorEngine) GetConnector() *Connector {
	oe.mu.RLock()
	defer oe.mu.RUnlock()
	return oe.connector
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
	if !oe.running {
		oe.mu.RUnlock()
		return nil, fmt.Errorf("orchestrator engine is not running")
	}
	ua := oe.unifiedAgent
	sc := oe.sessionContainer
	oe.mu.RUnlock()

	if ua != nil {
		thinkingResult, err := ua.ExecuteTaskWithThinking(ctx, task.Title)
		if err != nil {
			return nil, fmt.Errorf("failed to execute task via UnifiedAgent: %w", err)
		}
		output, _ := thinkingResult.(string)
		result := &agent.TaskExecutionResult{
			Success:  true,
			Output:   output,
			Duration: 0,
		}
		// تسجيل النتيجة في الجلسة
		if sc != nil {
			sc.UpdateAgentTaskResult("unified", result.Success)
		}
		return result, nil
	}

	// Fallback: تحديد القدرات المطلوبة للمهمة والبحث عن أفضل وكيل
	requiredCapabilities := oe.getRequiredCapabilities(task)

	bestAgentObj, err := oe.registry.FindBestAgent(requiredCapabilities)
	if err != nil {
		return nil, fmt.Errorf("failed to find suitable agent: %w", err)
	}

	bestAgentID := bestAgentObj.GetInfo().ID

	result, err := bestAgentObj.ExecuteTask(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to execute task: %w", err)
	}

	tokensUsed := 0
	if result.Metrics != nil {
		if val, ok := result.Metrics["tokens"].(int); ok {
			tokensUsed = val
		}
	}
	oe.registry.UpdateStats(bestAgentID, result.Success, tokensUsed, result.Duration)

	// تسجيل النتيجة في الجلسة
	if sc != nil {
		sc.UpdateAgentTaskResult(bestAgentID, result.Success)
	}

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
	if !oe.running {
		oe.mu.RUnlock()
		return nil, fmt.Errorf("orchestrator engine is not running")
	}
	sc := oe.sessionContainer
	oe.mu.RUnlock()

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

	// تسجيل النتيجة في الجلسة
	if sc != nil {
		sc.UpdateAgentTaskResult(agentID, result.Success)
	}

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

	agentID := agentObj.GetInfo().ID

	// [1] تسجيل الوكيل في السجل
	if err := oe.registry.Register(agentObj, metadata); err != nil {
		return err
	}

	// [2] تهيئة الوكيل في مدير دورة الحياة
	oe.lifecycleManager.InitializeAgent(agentID)

	// [3] تسجيل الوكيل في الجلسة مع التحقق من القدرات
	if oe.sessionContainer != nil {
		suggestedRole, _ := oe.roleAssigner.SuggestRole(agentID)
		agentRole := string(suggestedRole)
		agentInfo, err := oe.sessionContainer.RegisterAgentFromUnified(agentObj, agentRole)
		if err != nil {
			oe.logger.Warn("Failed to register agent in session",
				zap.String("agent_id", agentID),
				zap.Error(err),
			)
		} else {
			// [4] تسجيل القدرات المحققة في CapabilityMatcher
			verifiedCaps := agentInfo.VerifiedCapabilities
			if len(verifiedCaps) == 0 {
				verifiedCaps = agentInfo.ClaimedCapabilities
			}
			oe.registerAgentCapabilities(agentID, verifiedCaps)
		}
	} else {
		// بدون جلسة — نسجل القدرات المعلنة مباشرة
		caps := agentObj.GetCapabilities()
		capStrs := make([]string, len(caps))
		for i, c := range caps {
			capStrs[i] = string(c)
		}
		oe.registerAgentCapabilities(agentID, capStrs)
	}

	// [5] اقتراح دور للوكيل
	role, err := oe.roleAssigner.SuggestRole(agentID)
	if err != nil {
		oe.logger.Warn("Failed to suggest role for agent",
			zap.String("agent_id", agentID),
			zap.Error(err),
		)
	} else {
		// تعيين الدور المقترح
		if err := oe.roleAssigner.AssignRole(agentID, role, 1.0); err != nil {
			oe.logger.Warn("Failed to assign suggested role",
				zap.String("agent_id", agentID),
				zap.String("role", string(role)),
				zap.Error(err),
			)
		}
	}

	oe.logger.Info("Agent registered in orchestrator",
		zap.String("agent_id", agentID),
		zap.String("suggested_role", string(role)),
	)

	return nil
}

// registerAgentCapabilities يسجل القدرات في CapabilityMatcher
func (oe *OrchestratorEngine) registerAgentCapabilities(agentID string, capStrs []string) {
	caps := make([]agent.AgentCapability, len(capStrs))
	for i, s := range capStrs {
		caps[i] = agent.AgentCapability(s)
	}
	oe.capabilityMatcher.RegisterCapabilities(agentID, caps)
}

// SyncAgentCapabilities يزامن القدرات المحققة من الجلسة إلى CapabilityMatcher
// [WHY] يُستدعى بعد اكتمال التحقق من قدرات وكيل
func (oe *OrchestratorEngine) SyncAgentCapabilities(agentID string) {
	oe.mu.Lock()
	defer oe.mu.Unlock()

	if oe.sessionContainer == nil {
		return
	}

	// الحصول على القدرات المحققة من الجلسة
	verifiedCaps := oe.sessionContainer.GetVerifiedCapabilities(agentID)
	if len(verifiedCaps) > 0 {
		oe.registerAgentCapabilities(agentID, verifiedCaps)
		oe.logger.Info("Synced verified capabilities from session",
			zap.String("agent_id", agentID),
			zap.Int("verified_count", len(verifiedCaps)),
		)
	}
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

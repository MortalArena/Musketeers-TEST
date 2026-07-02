package unified

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/agent/adapters"
	"github.com/MortalArena/Musketeers/pkg/agent/thinking"
	"github.com/MortalArena/Musketeers/pkg/agent/tools"
	"github.com/MortalArena/Musketeers/pkg/lifecycle"
	"github.com/MortalArena/Musketeers/pkg/providers"
	"github.com/MortalArena/Musketeers/pkg/session"
	"go.uber.org/zap"
)

// PoolAgentStatus حالة الوكيل في AgentPool
type PoolAgentStatus string

const (
	PoolAgentStatusRegistered PoolAgentStatus = "registered" // مسجل فقط، ThinkingEngine غير مهيأ
	PoolAgentStatusActive     PoolAgentStatus = "active"     // ThinkingEngine مهيأ ويعمل
	PoolAgentStatusParked     PoolAgentStatus = "parked"     // مخزن، ThinkingEngine محرر للذاكرة
	PoolAgentStatusError      PoolAgentStatus = "error"      // خطأ
)

// AgentPoolConfig إعدادات AgentPool
type AgentPoolConfig struct {
	MaxAgents          int           // الحد الأقصى لعدد الوكلاء (100 = default)
	MaxActiveAgents    int           // الحد الأقصى للوكلاء النشطين في نفس الوقت (20 = default)
	ParkAfterIdle      time.Duration // مدة الخمول قبل park (5 دقائق = default)
	ThinkingEngineInit bool          // هل نهيئ ThinkingEngine فور التسجيل (false = lazy)
}

// DefaultAgentPoolConfig الإعدادات الافتراضية
func DefaultAgentPoolConfig() AgentPoolConfig {
	return AgentPoolConfig{
		MaxAgents:          100,
		MaxActiveAgents:    20,
		ParkAfterIdle:      5 * time.Minute,
		ThinkingEngineInit: false,
	}
}

// AgentPool يدير جميع AgentInstance في الجلسة
// مسؤول عن دورة حياة الوكلاء: تسجيل، تنشيط، تعطيل، إزالة
type AgentPool struct {
	sessionID string
	config    AgentPoolConfig
	logger    *zap.Logger
	mu        sync.RWMutex

	// جميع AgentInstance — كل وكيل حقيقي بكل مكوناته
	instances map[string]*AgentInstance // agentID -> instance

	// مراجع للأنظمة المشتركة (لا يتملكها — يشاركها مع SessionManager)
	toolRegistry     *tools.ToolRegistry
	sharedExecutor   *tools.ToolExecutor       // منفذ مشترك (اختياري)
	sessionContainer *session.SessionContainer // جلسة حقيقية لربط الـ ThinkingEngines
	eventBus         *SessionEventBus          // [SAFETY] لفصل الوكيل من ناقل الأحداث عند الإزالة
	defaultProvider  providers.Provider        // [FIX] Provider الافتراضي لربط الموديلات بكل ThinkingEngine
	defaultModelID   string                    // [FIX] Model ID الافتراضي

	// Lifecycle
	lifecycle *lifecycle.LifecycleMixin
}

// AgentInstance يمثل وكيلاً حقيقياً في الجلسة
// كل instance له ThinkingEngine مستقل، ToolExecutor بصلاحياته، والـ adapter الأصلي
type AgentInstance struct {
	AgentID   string
	AgentType string
	Role      string // manager, assistant, regular
	Adapter   agent.UnifiedAgent // الـ adapter الأصلي (CLI/API/IDE/Browser)

	mu sync.RWMutex

	// ThinkingEngine — ينشأ lazily عند أول استخدام
	thinkingEngine     *thinking.ThinkingEngine
	thinkingEngineInit bool

	// ToolExecutor — له صلاحياته هو (مش صلاحيات مدير الجلسة)
	toolExecutor *tools.ToolExecutor

	// Runtime info — الربط الفعلي بين Provider و Model و ThinkingEngine
	runtimeProvider providers.Provider
	runtimeModel    string

	// حالة الوكيل
	status      PoolAgentStatus
	statusSince time.Time
	lastActive  time.Time

	// إحصائيات
	totalTasks   int
	successTasks int
	failedTasks  int

	// [SAFETY] علامة الإزالة — تمنع استخدام الوكيل بعد إزالته
	removed bool

	// [SAFETY] دالة إلغاء سياق الوكيل — تُستدعى عند الإزالة لإيقاف الغوروتينات
	cancel context.CancelFunc
}

// GetStatus يرجع حالة الوكيل
func (ai *AgentInstance) GetStatus() PoolAgentStatus {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.status
}

// NewAgentPool ينشئ AgentPool جديد
func NewAgentPool(sessionID string, config AgentPoolConfig, toolRegistry *tools.ToolRegistry, logger *zap.Logger) *AgentPool {
	if config.MaxAgents <= 0 {
		config.MaxAgents = 100
	}
	if config.MaxActiveAgents <= 0 {
		config.MaxActiveAgents = 20
	}
	if config.ParkAfterIdle <= 0 {
		config.ParkAfterIdle = 5 * time.Minute
	}

	return &AgentPool{
		sessionID:    sessionID,
		config:       config,
		logger:       logger,
		instances:    make(map[string]*AgentInstance),
		toolRegistry: toolRegistry,
		lifecycle:    lifecycle.NewLifecycleMixin(),
	}
}

// SetSharedExecutor يضبط منفذ مشترك (اختياري)
func (ap *AgentPool) SetSharedExecutor(executor *tools.ToolExecutor) {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.sharedExecutor = executor
}

// SetSessionContainer يضبط SessionContainer لربط الـ ThinkingEngines بمكونات الجلسة
func (ap *AgentPool) SetSessionContainer(sc *session.SessionContainer) {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.sessionContainer = sc
}

// SetToolRegistry يضبط ToolRegistry بعد أن يصبح SessionContainer جاهزاً
func (ap *AgentPool) SetToolRegistry(registry *tools.ToolRegistry) {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.toolRegistry = registry
}

// SetEventBus يضبط EventBus لفصل الوكلاء عند الإزالة
func (ap *AgentPool) SetEventBus(eventBus *SessionEventBus) {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.eventBus = eventBus
}

// SetDefaultProvider يضبط Provider الافتراضي لربط الموديلات بكل ThinkingEngine
func (ap *AgentPool) SetDefaultProvider(provider providers.Provider) {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.defaultProvider = provider
}

// SetDefaultModelID يضبط Model ID الافتراضي لكل ThinkingEngine
func (ap *AgentPool) SetDefaultModelID(modelID string) {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.defaultModelID = modelID
}

// SetAgentCancelFunc يخزن دالة إلغاء سياق الوكيل لإيقاف الغوروتينات عند الإزالة
func (ap *AgentPool) SetAgentCancelFunc(agentID string, cancel context.CancelFunc) {
	ap.mu.RLock()
	instance, exists := ap.instances[agentID]
	if !exists {
		ap.mu.RUnlock()
		return
	}
	// [SAFETY] نسخ المرجع للاستخدام خارج القفل
	instance.mu.Lock()
	ap.mu.RUnlock()
	instance.cancel = cancel
	instance.mu.Unlock()
}

// RegisterAgent يسجل وكيلاً جديداً في AgentPool
// ينشئ AgentInstance مع adapter معين ولكن بدون ThinkingEngine (lazy init)
func (ap *AgentPool) RegisterAgent(adapter agent.UnifiedAgent, role tools.AgentRole) (*AgentInstance, error) {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if adapter == nil {
		return nil, fmt.Errorf("adapter cannot be nil")
	}

	info := adapter.GetInfo()
	if info == nil {
		return nil, fmt.Errorf("adapter info cannot be nil")
	}

	agentID := info.ID
	if existing, exists := ap.instances[agentID]; exists {
		ap.logger.Debug("Agent already registered in pool, returning existing instance",
			zap.String("agent_id", agentID))
		return existing, nil
	}

	if len(ap.instances) >= ap.config.MaxAgents {
		return nil, fmt.Errorf("agent pool full: max %d agents", ap.config.MaxAgents)
	}

	// [FIXED] استخدام sessionID الحقيقي بدلاً من "." لربط الوكيل بالجلسة
	agentExecutor := tools.NewToolExecutorWithRegistry(
		ap.sessionID,
		ap.toolRegistry,
		role,
		ap.logger,
	)

	instance := &AgentInstance{
		AgentID:      agentID,
		AgentType:    string(info.Type),
		Role:         string(role),
		Adapter:      adapter,
		status:       PoolAgentStatusRegistered,
		statusSince:  time.Now(),
		lastActive:   time.Now(),
		toolExecutor: agentExecutor,
	}

	// إذا كان التهيئة الفورية مفعلة
	if ap.config.ThinkingEngineInit {
		if err := ap.initThinkingEngine(instance); err != nil {
			ap.logger.Warn("فشل تهيئة ThinkingEngine فوراً", zap.String("agent_id", agentID), zap.Error(err))
			instance.status = PoolAgentStatusRegistered
		}
		instance.status = PoolAgentStatusActive
	}

	ap.instances[agentID] = instance

	ap.logger.Info("تم تسجيل وكيل في AgentPool",
		zap.String("agent_id", agentID),
		zap.String("type", instance.AgentType),
		zap.Int("total_agents", len(ap.instances)))

	return instance, nil
}

// GetAgent يحصل على AgentInstance
// إذا كان الوكيل موجوداً، يعيده مباشرة
func (ap *AgentPool) GetAgent(agentID string) (*AgentInstance, error) {
	ap.mu.RLock()
	instance, exists := ap.instances[agentID]
	ap.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("agent not found in pool: %s", agentID)
	}
	return instance, nil
}

// ListAgents يرجع جميع AgentInstance في AgentPool
func (ap *AgentPool) ListAgents() []*AgentInstance {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	agents := make([]*AgentInstance, 0, len(ap.instances))
	for _, instance := range ap.instances {
		agents = append(agents, instance)
	}
	return agents
}

// HasThinkingEngine يتحقق مما إذا كان ThinkingEngine موجوداً دون إنشائه
// [SAFETY] لا يؤسس أي موارد — آمن للاستعلام من API
func (ap *AgentPool) HasThinkingEngine(agentID string) error {
	ap.mu.RLock()
	instance, exists := ap.instances[agentID]
	ap.mu.RUnlock()
	if !exists {
		return fmt.Errorf("agent not found: %s", agentID)
	}
	instance.mu.RLock()
	init := instance.thinkingEngineInit
	instance.mu.RUnlock()
	if !init {
		return fmt.Errorf("thinking engine not initialized for agent: %s", agentID)
	}
	return nil
}

// GetOrCreateThinkingEngine يحصل على ThinkingEngine للوكيل (lazy init)
// [SAFETY] القفل على ap.mu محتفظ به أثناء الحصول على instance.mu
// لمنع سباق TOCTOU مع RemoveAgent/ParkAgent
func (ap *AgentPool) GetOrCreateThinkingEngine(agentID string) (*thinking.ThinkingEngine, error) {
	// [DEADLOCK FIX] countActive يمسك RLock على كل instance.mu
	// لذا يجب استدعاؤه قبل قفل instance الحالي لتجنب self-deadlock
	activeCount := ap.countActive()
	if activeCount >= ap.config.MaxActiveAgents {
		return nil, fmt.Errorf("max active agents reached: %d", ap.config.MaxActiveAgents)
	}
	preInitCount := activeCount // للاستخدام في log بعد init (حيث instance.mu ما زال مقفولاً)

	ap.mu.Lock()
	instance, exists := ap.instances[agentID]
	if !exists {
		ap.mu.Unlock()
		return nil, fmt.Errorf("agent not found in pool: %s", agentID)
	}
	instance.mu.Lock()
	ap.mu.Unlock()
	defer instance.mu.Unlock()

	if instance.removed {
		return nil, fmt.Errorf("agent has been removed: %s", agentID)
	}

	// إذا كان مهيأً مسبقاً
	if instance.thinkingEngineInit && instance.thinkingEngine != nil {
		instance.lastActive = time.Now()
		instance.status = PoolAgentStatusActive
		instance.statusSince = time.Now()
		return instance.thinkingEngine, nil
	}

	// تهيئة ThinkingEngine — هذه هي اللحظة التي يصبح فيها الوكيل "حقيقياً"
	if err := ap.initThinkingEngine(instance); err != nil {
		return nil, fmt.Errorf("فشل تهيئة ThinkingEngine لـ %s: %w", agentID, err)
	}

	instance.status = PoolAgentStatusActive
	instance.statusSince = time.Now()
	instance.lastActive = time.Now()

	ap.logger.Info("تم تهيئة ThinkingEngine للوكيل",
		zap.String("agent_id", agentID),
		zap.Int("active_agents", preInitCount+1))

	return instance.thinkingEngine, nil
}

// isExternalAgentType يتحقق مما إذا كان الوكيل من نوع خارجي (CLI/IDE/Browser/Custom)
// هذه الوكلاء لديهم ذكاء خاص بهم ولا يحتاجون إلى Provider في ThinkingEngine
func IsExternalAgentType(agentType string) bool {
	switch agentType {
	case string(agent.AgentTypeCLI),
		string(agent.AgentTypeIDE),
		string(agent.AgentTypeBrowser),
		string(agent.AgentTypeCustom):
		return true
	}
	return false
}

// initThinkingEngine يهيئ ThinkingEngine للـ AgentInstance
// [SAFETY] يتحقق من الحد الأقصى للوكلاء النشطين قبل التهيئة
func (ap *AgentPool) initThinkingEngine(instance *AgentInstance) error {
	if instance.thinkingEngineInit {
		return nil
	}

	// [SAFETY] فرض الحد الأقصى للوكلاء النشطين — يتم التحقق منه قبل instance.mu.Lock في GetOrCreateThinkingEngine
	// لا نستدعي countActive هنا لأنه يمسك RLock على instance.mu ويسبب self-deadlock

	// إنشاء ThinkingEngine جديد — مستقل تماماً لكل وكيل
	te := thinking.NewThinkingEngine(ap.sessionID, instance.AgentID, ap.logger)

	// تعيين منفذ الأدوات الخاص بالوكيل — بصلاحياته هو
	te.SetToolExecutor(instance.toolExecutor)

	// ربط بـ WorkflowEngine الحقيقي (مشترك بين الوكلاء)
	if ap.sessionContainer != nil && ap.sessionContainer.Workflow != nil {
		te.SetWorkflowEngine(ap.sessionContainer.Workflow)
	}

	// [EXTERNAL BRIDGE] الوكلاء الخارجيون (CLI/IDE/Browser) لديهم ذكاء خاص بهم
	// نتخطى تعيين Provider لأنهم لا يحتاجون ThinkingEngine للـ LLM
	if IsExternalAgentType(instance.AgentType) {
		instance.runtimeProvider = nil
		instance.runtimeModel = "external"
		ap.logger.Info("External agent — ThinkingEngine skips provider setup (agent has its own intelligence)",
			zap.String("agent_id", instance.AgentID),
			zap.String("type", instance.AgentType))
		instance.thinkingEngine = te
		instance.thinkingEngineInit = true
		return nil
	}

	// [RUNTIME AGENT] استخدام Provider و Model الخاصين بالوكيل إن أمكن
	// لكل وكيل من نوع ProviderAdapter مزود وموديل خاص به (مثلاً agent-1-mistral له Mistral)
	// هذا هو العمود الفقري لـ Provider → Model → Runtime Agent → ThinkingEngine
	providerSet := false
	if pa, ok := instance.Adapter.(*adapters.ProviderAdapter); ok {
		agentProvider, agentModel := pa.GetProvider()
		if agentProvider != nil && agentModel != "" {
			te.SetProvider(agentProvider, agentModel)
			providerSet = true
			instance.runtimeProvider = agentProvider
			instance.runtimeModel = agentModel
			ap.logger.Info("Runtime Agent linked to its own Provider+Model",
				zap.String("agent_id", instance.AgentID),
				zap.String("provider", string(agentProvider.Type())),
				zap.String("model", agentModel))
		}
	}
	if !providerSet && ap.defaultProvider != nil && ap.defaultModelID != "" {
		te.SetProvider(ap.defaultProvider, ap.defaultModelID)
		providerSet = true
		instance.runtimeProvider = ap.defaultProvider
		instance.runtimeModel = ap.defaultModelID
		ap.logger.Info("Pool default provider set for non-provider agent's ThinkingEngine",
			zap.String("agent_id", instance.AgentID),
			zap.String("provider", string(ap.defaultProvider.Type())),
			zap.String("model", ap.defaultModelID))
	}
	if !providerSet {
		ap.logger.Warn("No provider available for ThinkingEngine — agent cannot execute LLM tasks",
			zap.String("agent_id", instance.AgentID))
	}

	// [NEW] Auto-wire ContextReranker من SessionContainer
	if ap.sessionContainer != nil {
		contextReranker := ap.sessionContainer.GetContextReranker(ap.logger)
		if contextReranker != nil {
			// تعيين ContextReranker في ThinkingEngine
			te.SetContextReranker(contextReranker)
			ap.logger.Info("تم توصيل ContextReranker تلقائياً للوكيل",
				zap.String("agent_id", instance.AgentID))
		}
	}

	instance.thinkingEngine = te
	instance.thinkingEngineInit = true

	return nil
}

// SetPrebuiltThinkingEngine يحقن ThinkingEngine موجود مسبقاً في AgentInstance
func (ap *AgentPool) SetPrebuiltThinkingEngine(agentID string, te *thinking.ThinkingEngine) error {
	ap.mu.RLock()
	instance, exists := ap.instances[agentID]
	if !exists {
		ap.mu.RUnlock()
		return fmt.Errorf("agent not found: %s", agentID)
	}
	instance.mu.Lock()
	ap.mu.RUnlock()
	defer instance.mu.Unlock()

	if instance.removed {
		return fmt.Errorf("agent has been removed: %s", agentID)
	}

	instance.thinkingEngine = te
	instance.thinkingEngineInit = true
	instance.lastActive = time.Now()
	instance.status = PoolAgentStatusActive
	instance.statusSince = time.Now()

	ap.logger.Info("تم حقن ThinkingEngine موجود مسبقاً في AgentInstance",
		zap.String("agent_id", agentID))

	return nil
}

// ConnectThinkingEngineToSession يربط ThinkingEngine لمكونات الجلسة المشتركة
func (ap *AgentPool) ConnectThinkingEngineToSession(agentID string) error {
	ap.mu.RLock()
	instance, exists := ap.instances[agentID]
	if !exists {
		ap.mu.RUnlock()
		return fmt.Errorf("agent not found: %s", agentID)
	}
	instance.mu.Lock()
	ap.mu.RUnlock()
	defer instance.mu.Unlock()

	if instance.removed || !instance.thinkingEngineInit || instance.thinkingEngine == nil {
		return nil
	}

	te := instance.thinkingEngine
	sc := ap.sessionContainer

	// ربط CollectiveMemory
	if sc != nil && sc.Memory != nil {
		collectiveAdaptor := thinking.NewCollectiveMemoryAdaptor(sc.Memory)
		te.SetCollectiveMemory(collectiveAdaptor)
	}

	// ربط SkillsManager
	if sc != nil && sc.Skills != nil {
		skillsAdaptor := thinking.NewSkillsManagerAdaptor(sc.Skills)
		te.SetSkillsManager(skillsAdaptor)
	}

	// ربط SessionJournal
	if sc != nil && sc.Journal != nil {
		journalAdaptor := thinking.NewSessionJournalAdaptor(sc.Journal)
		te.SetSessionJournal(journalAdaptor)
	}

	// ربط TaskManager
	if sc != nil && sc.Tasks != nil {
		taskAdaptor := thinking.NewTaskManagerAdaptor(sc.Tasks)
		te.SetTaskManager(taskAdaptor)
	}

	// ربط WorkflowEngine
	if sc != nil && sc.Workflow != nil {
		te.SetWorkflowEngine(sc.Workflow)
	}

	return nil
}

// ConnectAllToSession يربط جميع الوكلاء النشطين بمكونات الجلسة
func (ap *AgentPool) ConnectAllToSession() {
	ap.mu.RLock()
	ids := make([]string, 0, len(ap.instances))
	for id := range ap.instances {
		ids = append(ids, id)
	}
	ap.mu.RUnlock()

	for _, id := range ids {
		_ = ap.ConnectThinkingEngineToSession(id)
	}
}

// ParkAgent يخزن الوكيل — يحرر ThinkingEngine لتوفير الذاكرة
func (ap *AgentPool) ParkAgent(agentID string) error {
	ap.mu.Lock()
	instance, exists := ap.instances[agentID]
	if !exists {
		ap.mu.Unlock()
		return fmt.Errorf("agent not found: %s", agentID)
	}
	instance.mu.Lock()
	ap.mu.Unlock()
	defer instance.mu.Unlock()

	if instance.removed {
		return fmt.Errorf("agent has been removed: %s", agentID)
	}

	if instance.status != PoolAgentStatusActive {
		return nil
	}

	instance.thinkingEngine = nil
	instance.thinkingEngineInit = false

	instance.status = PoolAgentStatusParked
	instance.statusSince = time.Now()

	ap.logger.Debug("تم تعطيل الوكيل (parked)",
		zap.String("agent_id", agentID))

	return nil
}

// WakeAgent يستيقظ الوكيل — يعيد تهيئة ThinkingEngine
func (ap *AgentPool) WakeAgent(agentID string) (*thinking.ThinkingEngine, error) {
	return ap.GetOrCreateThinkingEngine(agentID)
}

// RemoveAgent يزيل وكيلاً بالكامل من AgentPool مع تنظيف كامل
// [FIX] عدم استدعاء دوال خارجية (cancel.Close) داخل instance.mu — يمنع deadlock
func (ap *AgentPool) RemoveAgent(agentID string) error {
	ap.mu.Lock()
	instance, exists := ap.instances[agentID]
	if !exists {
		ap.mu.Unlock()
		return fmt.Errorf("agent not found: %s", agentID)
	}

	// [SAFETY] إزالة من الخريطة أولاً — يمنع أي استخدام جديد
	delete(ap.instances, agentID)

	// [SAFETY] قفل الـ instance للتنظيف الآمن
	instance.mu.Lock()
	instance.removed = true
	// [FIX] نسخ المرجع لاستدعائه خارج القفل — يمنع deadlock
	cancelFn := instance.cancel
	adapter := instance.Adapter
	instance.mu.Unlock()
	ap.mu.Unlock()

	// 1. إلغاء سياق الوكيل — يوقف جميع الغوروتينات التابعة (خارج instance.mu)
	if cancelFn != nil {
		cancelFn()
	}

	// 2. إلغاء اشتراك الوكيل من ناقل الأحداث (خارج instance.mu)
	if ap.eventBus != nil {
		ap.eventBus.UnsubscribeAgent(agentID)
	}

	// 3. إغلاق الـ adapter (خارج instance.mu)
	if err := adapter.Close(); err != nil {
		ap.logger.Warn("فشل إغلاق adapter", zap.String("agent_id", agentID), zap.Error(err))
	}

	ap.logger.Info("تم إزالة الوكيل من AgentPool مع تنظيف كامل",
		zap.String("agent_id", agentID),
		zap.Int("remaining", len(ap.instances)))

	return nil
}

// GetRuntimeProvider يرجع Provider المرتبط فعلياً بـ ThinkingEngine الخاص بالوكيل
func (ai *AgentInstance) GetRuntimeProvider() (providers.Provider, string) {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.runtimeProvider, ai.runtimeModel
}

// GetRole يرجع دور الوكيل
func (ai *AgentInstance) GetRole() string {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.Role
}

// GetThinkingEngineInit يرجع حالة تهيئة ThinkingEngine
func (ai *AgentInstance) GetThinkingEngineInit() bool {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.thinkingEngineInit
}

// GetTaskStats يرجع إحصائيات المهام
func (ai *AgentInstance) GetTaskStats() (int, int, int) {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.totalTasks, ai.successTasks, ai.failedTasks
}

// RuntimeComponentsState يعيد حالة مكونات Runtime للوكيل
type RuntimeComponentsState struct {
	ThinkingEngine bool   `json:"thinking_engine"`
	Provider       bool   `json:"provider"`
	Workflow       bool   `json:"workflow"`
	Memory         bool   `json:"memory"`
	Skills         bool   `json:"skills"`
	Journal        bool   `json:"journal"`
	Tasks          bool   `json:"tasks"`
	ToolExecutor   bool   `json:"tool_executor"`
	ProviderName   string `json:"provider_name,omitempty"`
	ModelName      string `json:"model_name,omitempty"`
}

// GetRuntimeComponents يعيد حالة جميع مكونات Runtime للوكيل
func (ai *AgentInstance) GetRuntimeComponents(ap *AgentPool) *RuntimeComponentsState {
	ai.mu.RLock()
	defer ai.mu.RUnlock()

	state := &RuntimeComponentsState{
		ThinkingEngine: ai.thinkingEngineInit && ai.thinkingEngine != nil,
		Provider:       ai.runtimeProvider != nil,
		Workflow:       false,
		Memory:         false,
		Skills:         false,
		Journal:        false,
		Tasks:          false,
		ToolExecutor:   ai.toolExecutor != nil,
	}

	if ai.runtimeProvider != nil {
		state.ProviderName = string(ai.runtimeProvider.Type())
	}
	state.ModelName = ai.runtimeModel

	// Check session component connections via ThinkingEngine adaptors
	if ai.thinkingEngine != nil && ai.thinkingEngineInit {
		state.Workflow = ai.thinkingEngine.HasWorkflowEngine()
		state.Memory = ai.thinkingEngine.HasCollectiveMemory()
		state.Skills = ai.thinkingEngine.HasSkillsManager()
		state.Journal = ai.thinkingEngine.HasSessionJournal()
		state.Tasks = ai.thinkingEngine.HasTaskManager()
	}

	return state
}

// GetSessionID يرجع معرف الجلسة المرتبطة بـ AgentPool
func (ap *AgentPool) GetSessionID() string {
	ap.mu.RLock()
	defer ap.mu.RUnlock()
	return ap.sessionID
}

// GetSessionContainer يرجع SessionContainer المرتبط بـ AgentPool
func (ap *AgentPool) GetSessionContainer() *session.SessionContainer {
	ap.mu.RLock()
	defer ap.mu.RUnlock()
	return ap.sessionContainer
}

// GetToolExecutor يحصل على ToolExecutor للوكيل
func (ap *AgentPool) GetToolExecutor(agentID string) (*tools.ToolExecutor, error) {
	ap.mu.RLock()
	instance, exists := ap.instances[agentID]
	if !exists {
		ap.mu.RUnlock()
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}
	instance.mu.RLock()
	ap.mu.RUnlock()
	defer instance.mu.RUnlock()

	if instance.removed {
		return nil, fmt.Errorf("agent has been removed: %s", agentID)
	}

	return instance.toolExecutor, nil
}

// SetAgentRole يغير صلاحيات الوكيل
func (ap *AgentPool) SetAgentRole(agentID string, role tools.AgentRole) error {
	ap.mu.RLock()
	instance, exists := ap.instances[agentID]
	if !exists {
		ap.mu.RUnlock()
		return fmt.Errorf("agent not found: %s", agentID)
	}
	instance.mu.Lock()
	ap.mu.RUnlock()
	defer instance.mu.Unlock()

	if instance.removed {
		return fmt.Errorf("agent has been removed: %s", agentID)
	}

	instance.toolExecutor.SetAgentRole(role)
	return nil
}

// GetActiveAgents يعيد قائمة الوكلاء النشطين حالياً
func (ap *AgentPool) GetActiveAgents() []string {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	result := make([]string, 0, len(ap.instances))
	for id, instance := range ap.instances {
		instance.mu.RLock()
		if instance.status == PoolAgentStatusActive {
			result = append(result, id)
		}
		instance.mu.RUnlock()
	}
	return result
}

// GetAllAgents يعيد قائمة جميع الوكلاء المسجلين
func (ap *AgentPool) GetAllAgents() []string {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	result := make([]string, 0, len(ap.instances))
	for id := range ap.instances {
		result = append(result, id)
	}
	return result
}

// Count يعيد عدد الوكلاء
func (ap *AgentPool) Count() int {
	ap.mu.RLock()
	defer ap.mu.RUnlock()
	return len(ap.instances)
}

// countActive يحسب عدد الوكلاء النشطين — مع قفل قراءة على الخريطة
// [FIX] قراءة ap.instances تحت RLock يمنع map iteration + write race
func (ap *AgentPool) countActive() int {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	count := 0
	for _, instance := range ap.instances {
		instance.mu.RLock()
		if instance.status == PoolAgentStatusActive {
			count++
		}
		instance.mu.RUnlock()
	}
	return count
}

// AutoParkWorker يبدأ عاملًا دوريًا لتعطيل الوكلاء الخاملين
func (ap *AgentPool) AutoParkWorker(ctx context.Context) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				ap.logger.Error("AutoParkWorker panicked", zap.Any("panic", r))
			}
		}()
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ap.parkIdleAgents()
			}
		}
	}()
}

// parkIdleAgents يعطل الوكلاء الخاملين لتوفير الذاكرة
// [FIX] التحقق من وجود الوكيل بعد الحصول عليه تحت RLock لمنع race مع RemoveAgent
func (ap *AgentPool) parkIdleAgents() {
	ap.mu.RLock()
	type agentInfo struct {
		id      string
		active  bool
		idleFor time.Duration
	}
	agents := make([]agentInfo, 0, len(ap.instances))
	for id, instance := range ap.instances {
		instance.mu.RLock()
		agents = append(agents, agentInfo{
			id:      id,
			active:  instance.status == PoolAgentStatusActive,
			idleFor: time.Since(instance.lastActive),
		})
		instance.mu.RUnlock()
	}
	ap.mu.RUnlock()

	parked := 0
	for _, info := range agents {
		if info.active && info.idleFor > ap.config.ParkAfterIdle {
			if err := ap.ParkAgent(info.id); err == nil {
				parked++
			}
		}
	}

	if parked > 0 {
		ap.logger.Debug("تم تعطيل وكلاء خاملين", zap.Int("parked", parked))
	}
}

// ============================================================
// Lifecycle Methods - تطبيق Lifecycle Interface
// ============================================================

// Start يبدأ AgentPool
func (ap *AgentPool) Start(ctx context.Context) error {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	ap.lifecycle.SetStatus(lifecycle.LifecycleStatusStarting)
	ap.lifecycle.SetStatus(lifecycle.LifecycleStatusRunning)
	return nil
}

// Stop يوقف AgentPool
func (ap *AgentPool) Stop(ctx context.Context) error {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	ap.lifecycle.SetStatus(lifecycle.LifecycleStatusStopping)
	ap.lifecycle.SetStatus(lifecycle.LifecycleStatusStopped)
	return nil
}

// Close يغلق AgentPool
func (ap *AgentPool) Close() error {
	return ap.Stop(ap.lifecycle.Context())
}

// Shutdown يوقف AgentPool بشكل آمن
func (ap *AgentPool) Shutdown(ctx context.Context) error {
	return ap.Stop(ctx)
}

// Cancel يلغي العمليات الجارية
func (ap *AgentPool) Cancel() error {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	ap.lifecycle.CancelContext()
	return nil
}

// IsRunning يتحقق مما إذا كان يعمل
func (ap *AgentPool) IsRunning() bool {
	return ap.lifecycle.IsRunningMixin()
}

// Status يرجع الحالة
func (ap *AgentPool) Status() lifecycle.LifecycleStatus {
	return ap.lifecycle.GetStatus()
}

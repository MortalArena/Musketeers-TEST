package unified

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/agent/thinking"
	"github.com/MortalArena/Musketeers/pkg/agent/tools"
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
}

// AgentInstance يمثل وكيلاً حقيقياً في الجلسة
// كل instance له ThinkingEngine مستقل، ToolExecutor بصلاحياته، والـ adapter الأصلي
type AgentInstance struct {
	AgentID   string
	AgentType string
	Adapter   agent.UnifiedAgent // الـ adapter الأصلي (CLI/API/IDE/Browser)

	mu sync.RWMutex

	// ThinkingEngine — ينشأ lazily عند أول استخدام
	thinkingEngine     *thinking.ThinkingEngine
	thinkingEngineInit bool

	// ToolExecutor — له صلاحياته هو (مش صلاحيات مدير الجلسة)
	toolExecutor *tools.ToolExecutor

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
	if _, exists := ap.instances[agentID]; exists {
		return nil, fmt.Errorf("agent already registered in pool: %s", agentID)
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

// GetOrCreateThinkingEngine يحصل على ThinkingEngine للوكيل (lazy init)
// [SAFETY] القفل على ap.mu محتفظ به أثناء الحصول على instance.mu
// لمنع سباق TOCTOU مع RemoveAgent/ParkAgent
func (ap *AgentPool) GetOrCreateThinkingEngine(agentID string) (*thinking.ThinkingEngine, error) {
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
		zap.Int("active_agents", ap.countActive()))

	return instance.thinkingEngine, nil
}

// initThinkingEngine يهيئ ThinkingEngine للـ AgentInstance
// [SAFETY] يتحقق من الحد الأقصى للوكلاء النشطين قبل التهيئة
func (ap *AgentPool) initThinkingEngine(instance *AgentInstance) error {
	if instance.thinkingEngineInit {
		return nil
	}

	// [SAFETY] فرض الحد الأقصى للوكلاء النشطين
	if ap.countActive() >= ap.config.MaxActiveAgents {
		return fmt.Errorf("max active agents reached: %d", ap.config.MaxActiveAgents)
	}

	// إنشاء ThinkingEngine جديد — مستقل تماماً لكل وكيل
	te := thinking.NewThinkingEngine(ap.sessionID, instance.AgentID, ap.logger)

	// تعيين منفذ الأدوات الخاص بالوكيل — بصلاحياته هو
	te.SetToolExecutor(instance.toolExecutor)

	// ربط بـ WorkflowEngine الحقيقي (مشترك بين الوكلاء)
	if ap.sessionContainer != nil && ap.sessionContainer.Workflow != nil {
		te.SetWorkflowEngine(ap.sessionContainer.Workflow)
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

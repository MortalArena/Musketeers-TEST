package automation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AutomationManager يدير الأتمتة بناءً على نظام Cursor
type AutomationManager struct {
	automations    map[string]*Automation
	triggerManager *TriggerManager
	actionManager  *ActionManager
	mcpManager     *MCPManager
	logger         *zap.Logger
	mu             sync.RWMutex
}

// Automation يمثل أتمتة واحدة
type Automation struct {
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Triggers       []Trigger              `json:"triggers"`
	Actions        []Action               `json:"actions"`
	Prompts        []string               `json:"prompts"`
	Model          string                 `json:"model"`
	MemoryEnabled  bool                   `json:"memory_enabled"`
	AgentOptions   AgentOptions           `json:"agent_options"`
	Metadata       map[string]interface{} `json:"metadata"`
	Enabled        bool                   `json:"enabled"`
	CreatedAt      time.Time              `json:"created_at"`
	LastExecuted   time.Time              `json:"last_executed"`
	ExecutionCount int                    `json:"execution_count"`
}

// AgentOptions خيارات الوكيل
type AgentOptions struct {
	SkipInstall bool `json:"skip_install"`
}

// Trigger واجهة للتشغيل
type Trigger interface {
	Type() string
	Evaluate(ctx context.Context) bool
	GetPayload() map[string]interface{}
}

// Action واجهة للإجراء
type Action interface {
	Type() string
	Execute(ctx context.Context, payload map[string]interface{}) error
}

// TriggerManager يدير التشغيلات
type TriggerManager struct {
	triggers map[string]Trigger
	logger   *zap.Logger
	mu       sync.RWMutex
}

// ActionManager يدير الإجراءات
type ActionManager struct {
	actions map[string]Action
	logger  *zap.Logger
	mu      sync.RWMutex
}

// MCPManager يدير خوادم MCP
type MCPManager struct {
	servers map[string]*MCPServer
	logger  *zap.Logger
	mu      sync.RWMutex
}

// MCPServer يمثل خادم MCP
type MCPServer struct {
	Name         string `json:"name"`
	ServerName   string `json:"server_name"`
	ServerID     string `json:"server_id"`
	Authenticated bool `json:"authenticated"`
	Enabled      bool   `json:"enabled"`
}

// NewAutomationManager ينشئ مدير أتمتة جديد
func NewAutomationManager(logger *zap.Logger) *AutomationManager {
	return &AutomationManager{
		automations:    make(map[string]*Automation),
		triggerManager: NewTriggerManager(logger),
		actionManager:  NewActionManager(logger),
		mcpManager:     NewMCPManager(logger),
		logger:         logger,
	}
}

// NewTriggerManager ينشئ مدير تشغيلات جديد
func NewTriggerManager(logger *zap.Logger) *TriggerManager {
	return &TriggerManager{
		triggers: make(map[string]Trigger),
		logger:   logger,
	}
}

// NewActionManager ينشئ مدير إجراءات جديد
func NewActionManager(logger *zap.Logger) *ActionManager {
	return &ActionManager{
		actions: make(map[string]Action),
		logger:  logger,
	}
}

// NewMCPManager ينشئ مدير MCP جديد
func NewMCPManager(logger *zap.Logger) *MCPManager {
	return &MCPManager{
		servers: make(map[string]*MCPServer),
		logger:  logger,
	}
}

// CreateAutomation ينشئ أتمتة جديدة
func (am *AutomationManager) CreateAutomation(config *AutomationConfig) (*Automation, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	// [WHY] إنشاء أتمتة جديدة
	// [HOW] يحول التكوين إلى أتمتة
	// [SAFETY] يتحقق من صحة التكوين

	automation := &Automation{
		Name:          config.Name,
		Description:   config.Description,
		Triggers:      config.Triggers,
		Actions:       config.Actions,
		Prompts:       config.Prompts,
		Model:         config.Model,
		MemoryEnabled: config.MemoryEnabled,
		AgentOptions:  config.AgentOptions,
		Metadata:      config.Metadata,
		Enabled:       true,
		CreatedAt:     time.Now(),
	}

	// التحقق من الحقول المطلوبة
	if automation.Name == "" {
		return nil, fmt.Errorf("اسم الأتمتة مطلوب")
	}
	if len(automation.Triggers) == 0 {
		return nil, fmt.Errorf("يجب أن تحتوي الأتمتة على تشغيل واحد على الأقل")
	}
	if len(automation.Actions) == 0 {
		return nil, fmt.Errorf("يجب أن تحتوي الأتمتة على إجراء واحد على الأقل")
	}

	am.automations[automation.Name] = automation
	am.logger.Info("تم إنشاء أتمتة", 
		zap.String("name", automation.Name),
		zap.Int("triggers", len(automation.Triggers)),
		zap.Int("actions", len(automation.Actions)))

	return automation, nil
}

// GetAutomation يحصل على أتمتة بالاسم
func (am *AutomationManager) GetAutomation(name string) (*Automation, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	automation, exists := am.automations[name]
	if !exists {
		return nil, fmt.Errorf("الأتمتة غير موجودة: %s", name)
	}

	return automation, nil
}

// GetAllAutomations يحصل على جميع الأتمتات
func (am *AutomationManager) GetAllAutomations() []*Automation {
	am.mu.RLock()
	defer am.mu.RUnlock()

	automations := make([]*Automation, 0, len(am.automations))
	for _, automation := range am.automations {
		automations = append(automations, automation)
	}

	return automations
}

// ExecuteAutomation ينفذ أتمتة
func (am *AutomationManager) ExecuteAutomation(ctx context.Context, automationName string) (*AutomationResult, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	automation, exists := am.automations[automationName]
	if !exists {
		return nil, fmt.Errorf("الأتمتة غير موجودة: %s", automationName)
	}

	if !automation.Enabled {
		return nil, fmt.Errorf("الأتمتة معطلة: %s", automationName)
	}

	return am.executeAutomation(ctx, automation)
}

// executeAutomation ينفذ أتمتة داخلياً
func (am *AutomationManager) executeAutomation(ctx context.Context, automation *Automation) (*AutomationResult, error) {
	// [WHY] تنفيذ أتمتة
	// [HOW] يتحقق من التشغيلات وينفذ الإجراءات
	// [SAFETY] يتحقق من السياق والتنفيذ

	result := &AutomationResult{
		AutomationName: automation.Name,
		Triggered:       false,
		Success:         false,
		ActionsExecuted: []ActionResult{},
		Errors:          []error{},
		Metadata:        make(map[string]interface{}),
	}

	// التحقق من التشغيلات
	for _, trigger := range automation.Triggers {
		if trigger.Evaluate(ctx) {
			result.Triggered = true
			result.Metadata["trigger_type"] = trigger.Type()
			result.Metadata["trigger_payload"] = trigger.GetPayload()
			break
		}
	}

	if !result.Triggered {
		result.Metadata["reason"] = "no_trigger_fired"
		return result, nil
	}

	// تنفيذ الإجراءات
	for _, action := range automation.Actions {
		actionResult := ActionResult{
			ActionType: action.Type(),
			Success:    false,
		}

		payload := triggerPayload(automation.Triggers)
		if err := action.Execute(ctx, payload); err != nil {
			actionResult.Error = err.Error()
			result.Errors = append(result.Errors, err)
		} else {
			actionResult.Success = true
		}

		result.ActionsExecuted = append(result.ActionsExecuted, actionResult)
	}

	// تحديث إحصائيات التنفيذ
	automation.LastExecuted = time.Now()
	automation.ExecutionCount++

	result.Success = len(result.Errors) == 0
	result.Metadata["execution_count"] = automation.ExecutionCount

	am.logger.Info("تم تنفيذ أتمتة", 
		zap.String("name", automation.Name),
		zap.Bool("success", result.Success),
		zap.Int("actions_executed", len(result.ActionsExecuted)))

	return result, nil
}

// triggerPayload يستخرج حمولة التشغيل
func triggerPayload(triggers []Trigger) map[string]interface{} {
	for _, trigger := range triggers {
		if payload := trigger.GetPayload(); payload != nil {
			return payload
		}
	}
	return make(map[string]interface{})
}

// EnableAutomation يفعّل أتمتة
func (am *AutomationManager) EnableAutomation(name string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	automation, exists := am.automations[name]
	if !exists {
		return fmt.Errorf("الأتمتة غير موجودة: %s", name)
	}

	automation.Enabled = true
	am.logger.Info("تم تفعيل أتمتة", zap.String("name", name))
	return nil
}

// DisableAutomation يعطل أتمتة
func (am *AutomationManager) DisableAutomation(name string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	automation, exists := am.automations[name]
	if !exists {
		return fmt.Errorf("الأتمتة غير موجودة: %s", name)
	}

	automation.Enabled = false
	am.logger.Info("تم تعطيل أتمتة", zap.String("name", name))
	return nil
}

// DeleteAutomation يحذف أتمتة
func (am *AutomationManager) DeleteAutomation(name string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	_, exists := am.automations[name]
	if !exists {
		return fmt.Errorf("الأتمتة غير موجودة: %s", name)
	}

	delete(am.automations, name)
	am.logger.Info("تم حذف أتمتة", zap.String("name", name))
	return nil
}

// RegisterTrigger يسجل تشغيل
func (tm *TriggerManager) RegisterTrigger(id string, trigger Trigger) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.triggers[id] = trigger
	tm.logger.Info("تم تسجيل تشغيل", zap.String("id", id), zap.String("type", trigger.Type()))
	return nil
}

// RegisterAction يسجل إجراء
func (am *ActionManager) RegisterAction(id string, action Action) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.actions[id] = action
	am.logger.Info("تم تسجيل إجراء", zap.String("id", id), zap.String("type", action.Type()))
	return nil
}

// RegisterMCPServer يسجل خادم MCP
func (mm *MCPManager) RegisterMCPServer(server *MCPServer) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.servers[server.ServerID] = server
	mm.logger.Info("تم تسجيل خادم MCP", 
		zap.String("server_id", server.ServerID),
		zap.String("server_name", server.ServerName))
	return nil
}

// GetMCPServer يحصل على خادم MCP
func (mm *MCPManager) GetMCPServer(serverID string) (*MCPServer, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	server, exists := mm.servers[serverID]
	if !exists {
		return nil, fmt.Errorf("خادم MCP غير موجود: %s", serverID)
	}

	return server, nil
}

// GetAuthenticatedMCPServers يحصل على خوادم MCP المصادق عليها
func (mm *MCPManager) GetAuthenticatedMCPServers() []*MCPServer {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	servers := make([]*MCPServer, 0)
	for _, server := range mm.servers {
		if server.Authenticated && server.Enabled {
			servers = append(servers, server)
		}
	}

	return servers
}

// AutomationConfig تكوين الأتمتة
type AutomationConfig struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Triggers      []Trigger              `json:"triggers"`
	Actions       []Action               `json:"actions"`
	Prompts       []string               `json:"prompts"`
	Model         string                 `json:"model"`
	MemoryEnabled bool                   `json:"memory_enabled"`
	AgentOptions  AgentOptions           `json:"agent_options"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// AutomationResult نتيجة تنفيذ الأتمتة
type AutomationResult struct {
	AutomationName string          `json:"automation_name"`
	Triggered       bool            `json:"triggered"`
	Success         bool            `json:"success"`
	ActionsExecuted []ActionResult  `json:"actions_executed"`
	Errors          []error         `json:"errors"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// ActionResult نتيجة تنفيذ الإجراء
type ActionResult struct {
	ActionType string `json:"action_type"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

// GetAutomationSummary يحصل على ملخص الأتمتات
func (am *AutomationManager) GetAutomationSummary() map[string]interface{} {
	am.mu.RLock()
	defer am.mu.RUnlock()

	summary := map[string]interface{}{
		"total_automations":   len(am.automations),
		"enabled_automations": 0,
		"disabled_automations": 0,
		"total_executions":     0,
	}

	for _, automation := range am.automations {
		if automation.Enabled {
			summary["enabled_automations"] = summary["enabled_automations"].(int) + 1
		} else {
			summary["disabled_automations"] = summary["disabled_automations"].(int) + 1
		}
		summary["total_executions"] = summary["total_executions"].(int) + automation.ExecutionCount
	}

	return summary
}

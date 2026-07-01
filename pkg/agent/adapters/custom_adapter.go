package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
)

// CustomAdapter محول للوكلاء المخصصة
// يدعم: أي وكيل مخصص من قبل المستخدم
type CustomAdapter struct {
	info        *agent.AgentInfo
	handler     CustomHandler // دالة معالجة مخصصة
	config      map[string]interface{}
	initialized bool
}

// CustomHandler دالة معالجة مخصصة
type CustomHandler func(ctx context.Context, task *agent.AgentTask) (*agent.TaskExecutionResult, error)

// NewCustomAdapter ينشئ محول مخصص
func NewCustomAdapter(info *agent.AgentInfo, handler CustomHandler) *CustomAdapter {
	return &CustomAdapter{
		info:        info,
		handler:     handler,
		config:      make(map[string]interface{}),
		initialized: false,
	}
}

// NewCustomAgent ينشئ وكيل مخصص بسيط
func NewCustomAgent(name, provider, model string, handler CustomHandler) *CustomAdapter {
	info := &agent.AgentInfo{
		ID:         fmt.Sprintf("custom_%s", name),
		Name:       name,
		Type:       agent.AgentTypeCustom,
		Provider:   provider,
		Model:      model,
		AuthMethod: "custom",
		CreatedAt:  time.Now(),
	}
	return NewCustomAdapter(info, handler)
}

// NewSimpleAgent ينشئ وكيل بسيط بدون handler مخصص
// يستخدم للوكلاء التي يتم إدارتها عبر providers مباشرة
func NewSimpleAgent(agentID, name string, agentType agent.AgentType, provider, model string) *CustomAdapter {
	info := &agent.AgentInfo{
		ID:         agentID,
		Name:       name,
		Type:       agentType,
		Provider:   provider,
		Model:      model,
		AuthMethod: "provider",
		CreatedAt:  time.Now(),
	}

	// Handler افتراضي بسيط
	handler := func(ctx context.Context, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
		return &agent.TaskExecutionResult{
			Success: true,
			Output:  fmt.Sprintf("Agent %s received task: %s", name, task.Title),
		}, nil
	}

	adapter := NewCustomAdapter(info, handler)
	adapter.initialized = true
	return adapter
}

func (a *CustomAdapter) GetInfo() *agent.AgentInfo {
	return a.info
}

func (a *CustomAdapter) SendMessage(ctx context.Context, prompt string) (*agent.AgentResponse, error) {
	startTime := time.Now()

	if !a.initialized {
		return nil, fmt.Errorf("الوكيل غير مهيأ")
	}

	if a.handler == nil {
		return nil, fmt.Errorf("لم يتم تعيين دالة معالجة")
	}

	// تحويل الرسالة إلى مهمة
	task := &agent.AgentTask{
		ID:          fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Title:       "رسالة",
		Description: prompt,
	}

	result, err := a.handler(ctx, task)
	if err != nil {
		return nil, err
	}

	return &agent.AgentResponse{
		Content:  result.Output,
		Duration: time.Since(startTime),
	}, nil
}

func (a *CustomAdapter) ExecuteTask(ctx context.Context, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	if !a.initialized {
		return nil, fmt.Errorf("الوكيل غير مهيأ")
	}

	if a.handler == nil {
		return nil, fmt.Errorf("لم يتم تعيين دالة معالجة")
	}

	return a.handler(ctx, task)
}

func (a *CustomAdapter) GetCapabilities() []agent.AgentCapability {
	// يمكن للمستخدم تحديد القدرات عبر config
	if caps, ok := a.config["capabilities"]; ok {
		if capsList, ok := caps.([]agent.AgentCapability); ok {
			return capsList
		}
	}

	// قدرات افتراضية
	return []agent.AgentCapability{
		agent.CapabilityCodeGeneration,
		agent.CapabilityCodeReview,
	}
}

func (a *CustomAdapter) GetStatus() *agent.AgentStatus {
	return &agent.AgentStatus{
		IsAvailable:  a.initialized,
		LastSeen:     time.Now(),
		ResponseTime: 1 * time.Second,
		SuccessRate:  95.0,
	}
}

func (a *CustomAdapter) IsAvailable() bool {
	return a.initialized && a.handler != nil
}

func (a *CustomAdapter) Close() error {
	a.initialized = false
	return nil
}

// Initialize يهيئ الوكيل
func (a *CustomAdapter) Initialize(config map[string]interface{}) error {
	a.config = config
	a.initialized = true
	return nil
}

// SetHandler يضبط دالة المعالجة
func (a *CustomAdapter) SetHandler(handler CustomHandler) {
	a.handler = handler
}

// SetConfig يضبط إعداداً معيناً
func (a *CustomAdapter) SetConfig(key string, value interface{}) {
	a.config[key] = value
}

// GetConfig يحصل على إعداد معين
func (a *CustomAdapter) GetConfig(key string) (interface{}, bool) {
	value, ok := a.config[key]
	return value, ok
}

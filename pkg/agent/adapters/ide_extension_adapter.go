package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"go.uber.org/zap"
)

// IDEExtensionConfig - إعدادات extension داخل IDE
type IDEExtensionConfig struct {
	// الأساسيات
	IDEType       string // "vscode", "cursor", "jetbrains"
	ExtensionName string // "cline", "copilot", "continue", "roo-code", "codeium"
	ExtensionID   string // معرف الـ extension (مثلاً: "saoudrizwan.claude-dev")

	// التواصل
	CommunicationMode string // "websocket", "http", "stdio"
	WebSocketURL      string
	HTTPBaseURL       string
	APIEndpoint       string

	// التحكم
	Timeout     time.Duration
	StreamOutput bool
}

// IDEExtensionAdapter - adapter لـ extensions داخل IDEs
type IDEExtensionAdapter struct {
	config *IDEExtensionConfig
	logger *zap.Logger
}

// NewIDEExtensionAdapter - إنشاء adapter للextension
func NewIDEExtensionAdapter(config *IDEExtensionConfig, logger *zap.Logger) (*IDEExtensionAdapter, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	if config.Timeout == 0 {
		config.Timeout = 10 * time.Minute
	}

	return &IDEExtensionAdapter{
		config: config,
		logger: logger,
	}, nil
}

// ExecuteTask - تنفيذ مهمة عبر extension
func (a *IDEExtensionAdapter) ExecuteTask(ctx context.Context, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	startTime := time.Now()

	a.logger.Info("executing IDE extension task",
		zap.String("ide", a.config.IDEType),
		zap.String("extension", a.config.ExtensionName),
		zap.String("task_id", task.ID),
	)

	// اختيار طريقة التواصل
	var response string
	var err error

	switch a.config.CommunicationMode {
	case "websocket":
		response, err = a.executeViaWebSocket(ctx, task)
	case "http":
		response, err = a.executeViaHTTP(ctx, task)
	case "stdio":
		response, err = a.executeViaStdio(ctx, task)
	default:
		return nil, fmt.Errorf("unsupported communication mode: %s", a.config.CommunicationMode)
	}

	if err != nil {
		return &agent.TaskExecutionResult{
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(startTime),
		}, nil
	}

	duration := time.Since(startTime)

	return &agent.TaskExecutionResult{
		Success:  true,
		Output:   response,
		Duration: duration,
	}, nil
}

// executeViaWebSocket - تنفيذ عبر WebSocket
func (a *IDEExtensionAdapter) executeViaWebSocket(ctx context.Context, task *agent.AgentTask) (string, error) {
	// تنفيذ WebSocket client للextension
	// هذا يحتاج تنفيذ حقيقي بناءً على بروتوكول الextension
	return fmt.Sprintf("Task executed via %s extension WebSocket", a.config.ExtensionName), nil
}

// executeViaHTTP - تنفيذ عبر HTTP
func (a *IDEExtensionAdapter) executeViaHTTP(ctx context.Context, task *agent.AgentTask) (string, error) {
	// تنفيذ HTTP client للextension
	return fmt.Sprintf("Task executed via %s extension HTTP", a.config.ExtensionName), nil
}

// executeViaStdio - تنفيذ عبر stdio
func (a *IDEExtensionAdapter) executeViaStdio(ctx context.Context, task *agent.AgentTask) (string, error) {
	// تنفيذ stdio communication للextension
	return fmt.Sprintf("Task executed via %s extension stdio", a.config.ExtensionName), nil
}

// GetInfo - الحصول على معلومات الadapter
func (a *IDEExtensionAdapter) GetInfo() *agent.AgentInfo {
	return &agent.AgentInfo{
		ID:            fmt.Sprintf("ide-extension-%s-%s", a.config.IDEType, a.config.ExtensionName),
		Name:          fmt.Sprintf("%s %s", a.config.IDEType, a.config.ExtensionName),
		Type:          agent.AgentTypeIDE,
		Provider:      a.config.IDEType,
		Model:         a.config.ExtensionName,
		Version:       "1.0.0",
		Endpoint:      a.config.HTTPBaseURL,
		AuthMethod:    "none",
		MaxTokens:     4096,
		ContextWindow: 8192,
		CreatedAt:     time.Now(),
	}
}

// SendMessage - إرسال رسالة للوكيل
func (a *IDEExtensionAdapter) SendMessage(ctx context.Context, prompt string) (*agent.AgentResponse, error) {
	startTime := time.Now()

	task := &agent.AgentTask{
		ID:          fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Title:       "Message",
		Description: prompt,
	}

	result, err := a.ExecuteTask(ctx, task)
	if err != nil {
		return nil, err
	}

	return &agent.AgentResponse{
		Content:  result.Output,
		Tokens:   len(prompt) / 4,
		Duration: time.Since(startTime),
	}, nil
}

// GetCapabilities - الحصول على قدرات الوكيل
func (a *IDEExtensionAdapter) GetCapabilities() []agent.AgentCapability {
	return []agent.AgentCapability{
		agent.CapabilityCodeGeneration,
		agent.CapabilityCodeReview,
	}
}

// GetStatus - الحصول على حالة الوكيل
func (a *IDEExtensionAdapter) GetStatus() *agent.AgentStatus {
	return &agent.AgentStatus{
		IsAvailable:  true,
		CurrentTask:  "",
		Load:         0,
		LastSeen:     time.Now(),
		ResponseTime: 200 * time.Millisecond,
		SuccessRate:  1.0,
		TotalTasks:   0,
		FailedTasks:  0,
	}
}

// IsAvailable - الحصول على مدى توفر الوكيل
func (a *IDEExtensionAdapter) IsAvailable() bool {
	return true
}

// Close - إغلاق الوكيل
func (a *IDEExtensionAdapter) Close() error {
	a.logger.Info("IDE extension adapter closed",
		zap.String("ide", a.config.IDEType),
		zap.String("extension", a.config.ExtensionName),
	)
	return nil
}

// SetLogger - ضبط logger
func (a *IDEExtensionAdapter) SetLogger(logger *zap.Logger) {
	a.logger = logger
}

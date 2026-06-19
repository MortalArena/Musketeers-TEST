package adapters

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"go.uber.org/zap"
)

// DesktopAppAdapter محول لتطبيقات سطح المكتب
// يدعم: Claude Desktop, Codex App, Hermes, وغيرها
type DesktopAppAdapter struct {
	info              *agent.AgentInfo
	appName           string
	executable        string
	communicationMode string // "websocket", "http", "stdio"
	webSocketURL      string
	httpBaseURL       string
	logger            *zap.Logger
	available         bool
	running           bool
}

// DesktopAppConfig إعدادات تطبيق سطح المكتب
type DesktopAppConfig struct {
	Name              string
	Executable        string
	CommunicationMode string // "websocket", "http", "stdio"
	WebSocketURL      string
	HTTPBaseURL       string
	AutoStart         bool
}

// NewDesktopAppAdapter ينشئ محول تطبيق سطح مكتب جديد
func NewDesktopAppAdapter(config *DesktopAppConfig, logger *zap.Logger) (*DesktopAppAdapter, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	return &DesktopAppAdapter{
		info: &agent.AgentInfo{
			ID:            fmt.Sprintf("desktop_%s", config.Name),
			Name:          fmt.Sprintf("%s Desktop App", config.Name),
			Type:          agent.AgentTypeCustom,
			Provider:      "desktop",
			Model:         config.Name,
			Version:       "1.0.0",
			Endpoint:      config.HTTPBaseURL,
			AuthMethod:    "none",
			MaxTokens:     4096,
			ContextWindow: 8192,
			CreatedAt:     time.Now(),
		},
		appName:           config.Name,
		executable:        config.Executable,
		communicationMode: config.CommunicationMode,
		webSocketURL:      config.WebSocketURL,
		httpBaseURL:       config.HTTPBaseURL,
		logger:            logger,
		available:         true,
		running:           false,
	}, nil
}

// GetInfo يعيد معلومات الوكيل
func (da *DesktopAppAdapter) GetInfo() *agent.AgentInfo {
	return da.info
}

// SendMessage يرسل رسالة للوكيل
func (da *DesktopAppAdapter) SendMessage(ctx context.Context, prompt string) (*agent.AgentResponse, error) {
	startTime := time.Now()

	if !da.running {
		return nil, fmt.Errorf("التطبيق غير يعمل")
	}

	// تنفيذ حسب طريقة التواصل
	var response string
	var err error

	switch da.communicationMode {
	case "websocket":
		response, err = da.sendViaWebSocket(ctx, prompt)
	case "http":
		response, err = da.sendViaHTTP(ctx, prompt)
	case "stdio":
		response, err = da.sendViaStdio(ctx, prompt)
	default:
		return nil, fmt.Errorf("طريقة تواصل غير مدعومة: %s", da.communicationMode)
	}

	if err != nil {
		return nil, err
	}

	duration := time.Since(startTime)

	da.logger.Info("Desktop app message sent",
		zap.String("app_name", da.appName),
		zap.Int("prompt_length", len(prompt)),
		zap.Duration("duration", duration),
	)

	return &agent.AgentResponse{
		Content:  response,
		Tokens:   len(prompt) / 4,
		Duration: duration,
	}, nil
}

// sendViaWebSocket يرسل عبر WebSocket
func (da *DesktopAppAdapter) sendViaWebSocket(ctx context.Context, prompt string) (string, error) {
	// في التنفيذ الحقيقي: WebSocket client
	return fmt.Sprintf("WebSocket response from %s: %s", da.appName, prompt), nil
}

// sendViaHTTP يرسل عبر HTTP
func (da *DesktopAppAdapter) sendViaHTTP(ctx context.Context, prompt string) (string, error) {
	// في التنفيذ الحقيقي: HTTP client
	return fmt.Sprintf("HTTP response from %s: %s", da.appName, prompt), nil
}

// sendViaStdio يرسل عبر stdio
func (da *DesktopAppAdapter) sendViaStdio(ctx context.Context, prompt string) (string, error) {
	// في التنفيذ الحقيقي: stdio communication
	return fmt.Sprintf("Stdio response from %s: %s", da.appName, prompt), nil
}

// ExecuteTask ينفذ مهمة
func (da *DesktopAppAdapter) ExecuteTask(ctx context.Context, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	startTime := time.Now()

	// تجهيز prompt من المهمة
	prompt := fmt.Sprintf("Task: %s\nDescription: %s", task.Title, task.Description)
	if task.Context != "" {
		prompt += fmt.Sprintf("\nContext: %s", task.Context)
	}

	// إرسال الرسالة
	response, err := da.SendMessage(ctx, prompt)
	if err != nil {
		return &agent.TaskExecutionResult{
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(startTime),
		}, nil
	}

	duration := time.Since(startTime)

	da.logger.Info("Desktop app task executed",
		zap.String("task_id", task.ID),
		zap.String("task_title", task.Title),
		zap.Bool("success", true),
		zap.Duration("duration", duration),
	)

	return &agent.TaskExecutionResult{
		Success:  true,
		Output:   response.Content,
		Duration: duration,
		Metrics: map[string]interface{}{
			"tokens": response.Tokens,
		},
	}, nil
}

// GetCapabilities يعيد قدرات الوكيل
func (da *DesktopAppAdapter) GetCapabilities() []agent.AgentCapability {
	return []agent.AgentCapability{
		agent.CapabilityCodeGeneration,
		agent.CapabilityCodeReview,
		agent.CapabilityBrowserControl,
	}
}

// GetStatus يعيد حالة الوكيل
func (da *DesktopAppAdapter) GetStatus() *agent.AgentStatus {
	return &agent.AgentStatus{
		IsAvailable:  da.available,
		CurrentTask:  "",
		Load:         0,
		LastSeen:     time.Now(),
		ResponseTime: 300 * time.Millisecond,
		SuccessRate:  1.0,
		TotalTasks:   0,
		FailedTasks:  0,
	}
}

// IsAvailable يعيد ما إذا كان الوكيل متاحاً
func (da *DesktopAppAdapter) IsAvailable() bool {
	return da.available
}

// Close يغلق الوكيل
func (da *DesktopAppAdapter) Close() error {
	if da.running {
		da.Stop()
	}
	da.available = false
	da.logger.Info("Desktop app adapter closed",
		zap.String("app_name", da.appName),
	)
	return nil
}

// Start يبدأ التطبيق
func (da *DesktopAppAdapter) Start() error {
	if da.running {
		return fmt.Errorf("التطبيق يعمل بالفعل")
	}

	if da.executable == "" {
		return fmt.Errorf("لم يتم تحديد المسار التنفيذي")
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", da.executable)
	case "windows":
		cmd = exec.Command("start", "", da.executable)
	case "linux":
		cmd = exec.Command(da.executable)
	default:
		return fmt.Errorf("نظام التشغيل غير مدعوم: %s", runtime.GOOS)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("فشل بدء التطبيق: %w", err)
	}

	da.running = true
	da.logger.Info("Desktop app started",
		zap.String("app_name", da.appName),
		zap.String("executable", da.executable),
	)

	return nil
}

// Stop يوقف التطبيق
func (da *DesktopAppAdapter) Stop() error {
	if !da.running {
		return fmt.Errorf("التطبيق غير يعمل")
	}

	// في التنفيذ الحقيقي: إيقاف العملية
	da.running = false
	da.logger.Info("Desktop app stopped",
		zap.String("app_name", da.appName),
	)

	return nil
}

// SetLogger يضبط logger
func (da *DesktopAppAdapter) SetLogger(logger *zap.Logger) {
	da.logger = logger
}

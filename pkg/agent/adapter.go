package agent

import (
	"context"
	"time"
)

// AgentType نوع الوكيل
type AgentType string

const (
	AgentTypeAPI     AgentType = "api"     // REST API (Claude, GPT, Gemini)
	AgentTypeCLI     AgentType = "cli"     // Command Line (Claude Code, Cline, Aider)
	AgentTypeIDE     AgentType = "ide"     // IDE Extension (Cursor, VS Code)
	AgentTypeLocal   AgentType = "local"   // Local Server (Ollama, LM Studio)
	AgentTypeBrowser AgentType = "browser" // Browser Automation
	AgentTypeCustom  AgentType = "custom"  // Custom Agent
)

// AgentCapability قدرة الوكيل
type AgentCapability string

const (
	CapabilityCodeGeneration AgentCapability = "code_generation"
	CapabilityCodeReview     AgentCapability = "code_review"
	CapabilityTesting        AgentCapability = "testing"
	CapabilityDocumentation  AgentCapability = "documentation"
	CapabilityDesign         AgentCapability = "design"
	CapabilityAnalysis       AgentCapability = "analysis"
	CapabilityFileOperations AgentCapability = "file_operations"
	CapabilityTerminalAccess AgentCapability = "terminal_access"
)

// UnifiedAgent واجهة وكيل موحدة - كل الوكلاء يطبقونها
type UnifiedAgent interface {
	GetInfo() *AgentInfo
	SendMessage(ctx context.Context, prompt string) (*AgentResponse, error)
	ExecuteTask(ctx context.Context, task *AgentTask) (*TaskExecutionResult, error)
	GetCapabilities() []AgentCapability
	GetStatus() *AgentStatus
	IsAvailable() bool
	Close() error
}

// AgentInfo معلومات الوكيل
type AgentInfo struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Type          AgentType `json:"type"`
	Provider      string    `json:"provider"` // claude, openai, ollama, etc.
	Model         string    `json:"model"`
	Version       string    `json:"version"`
	Endpoint      string    `json:"endpoint"`
	AuthMethod    string    `json:"auth_method"` // api_key, oauth, none
	MaxTokens     int       `json:"max_tokens"`
	ContextWindow int       `json:"context_window"`
	CreatedAt     time.Time `json:"created_at"`
}

// AgentResponse رد الوكيل
type AgentResponse struct {
	Content  string                 `json:"content"`
	Tokens   int                    `json:"tokens"`
	Duration time.Duration          `json:"duration"`
	Metadata map[string]interface{} `json:"metadata"`
}

// AgentTask مهمة للوكيل
type AgentTask struct {
	ID             string                 `json:"id"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	Context        string                 `json:"context"`
	Inputs         map[string]interface{} `json:"inputs"`
	Constraints    []string               `json:"constraints"`
	ExpectedOutput string                 `json:"expected_output"`
	Timeout        time.Duration          `json:"timeout"`
}

// TaskExecutionResult نتيجة تنفيذ المهمة
type TaskExecutionResult struct {
	Success   bool                   `json:"success"`
	Output    string                 `json:"output"`
	Artifacts []string               `json:"artifacts"` // ملفات ناتجة
	Metrics   map[string]interface{} `json:"metrics"`
	Error     string                 `json:"error,omitempty"`
	Duration  time.Duration          `json:"duration"`
}

// AgentStatus حالة الوكيل
type AgentStatus struct {
	IsAvailable  bool          `json:"is_available"`
	CurrentTask  string        `json:"current_task,omitempty"`
	Load         int           `json:"load"` // 0-100
	LastSeen     time.Time     `json:"last_seen"`
	ResponseTime time.Duration `json:"response_time"`
	SuccessRate  float64       `json:"success_rate"` // 0.0 - 1.0
	TotalTasks   int           `json:"total_tasks"`
	FailedTasks  int           `json:"failed_tasks"`
}

package providers

import (
	"context"
	"time"
)

// ProviderType represents the type of AI provider
type ProviderType string

const (
	// Official Providers (22 providers with official APIs)
	ProviderOpenAI     ProviderType = "openai"
	ProviderAnthropic  ProviderType = "anthropic"
	ProviderGoogle     ProviderType = "google"
	ProviderDeepSeek   ProviderType = "deepseek"
	ProviderXAI        ProviderType = "xai"
	ProviderMistral    ProviderType = "mistral"
	ProviderQwen       ProviderType = "qwen"
	ProviderMoonshot   ProviderType = "moonshot"
	ProviderNVIDIA     ProviderType = "nvidia"
	ProviderXiaomi     ProviderType = "xiaomi"
	ProviderZAI        ProviderType = "zai"
	ProviderTencent    ProviderType = "tencent"
	ProviderStepFun    ProviderType = "stepfun"
	ProviderPoolside   ProviderType = "poolside"
	ProviderRecraft    ProviderType = "recraft"
	ProviderSourceful  ProviderType = "sourceful"
	ProviderOpenRouter ProviderType = "openrouter"
	ProviderCohere     ProviderType = "cohere"
	ProviderGroq       ProviderType = "groq"
	ProviderTogetherAI ProviderType = "togetherai"
	ProviderPerplexity ProviderType = "perplexity"
	ProviderMinimax    ProviderType = "minimax"

	// Local Providers
	ProviderOllama ProviderType = "ollama"

	// Custom Provider
	ProviderCustom ProviderType = "custom"
)

// ModelCapability represents what a model can do
type ModelCapability string

const (
	CapabilityText          ModelCapability = "text"
	CapabilityCode          ModelCapability = "code"
	CapabilityVision        ModelCapability = "vision"
	CapabilityAudio         ModelCapability = "audio"
	CapabilityVideo         ModelCapability = "video"
	CapabilityImage         ModelCapability = "image"
	CapabilityEmbeddings    ModelCapability = "embeddings"
	CapabilityStreaming     ModelCapability = "streaming"
	CapabilityFunction      ModelCapability = "function"
	CapabilityReasoning     ModelCapability = "reasoning"
	CapabilityLongContext   ModelCapability = "long_context"
	CapabilityTranscription ModelCapability = "transcription"
	CapabilityTTS           ModelCapability = "tts"
	CapabilityRerank        ModelCapability = "rerank"
	CapabilitySearch        ModelCapability = "search"
)

// MessageRole represents the role of a message
type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
)

// Provider is the interface that all AI providers must implement
type Provider interface {
	// Type returns the provider type
	Type() ProviderType

	// Name returns the provider name
	Name() string

	// Capabilities returns the provider's capabilities
	Capabilities() ProviderCapabilities

	// Initialize initializes the provider with configuration
	Initialize(ctx context.Context, config ProviderConfig) error

	// Close closes the provider and cleans up resources
	Close() error

	// Ping checks if the provider is accessible
	Ping(ctx context.Context) error

	// Status returns the current status of the provider
	Status() ProviderStatus

	// IsAvailable checks if the provider is currently available
	IsAvailable() bool

	// ListModels returns all available models from this provider
	ListModels(ctx context.Context) ([]ModelInfo, error)

	// GetModel returns information about a specific model
	GetModel(ctx context.Context, modelID string) (*ModelInfo, error)

	// Complete performs a non-streaming completion
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// StreamComplete performs a streaming completion
	StreamComplete(ctx context.Context, req *CompletionRequest, callback StreamingCallback) error
}

// ProviderConfig holds configuration for a provider
type ProviderConfig struct {
	APIKey  string
	BaseURL string
	Timeout time.Duration
	Extra   map[string]interface{}
}

// ProviderCapabilities describes what a provider supports
type ProviderCapabilities struct {
	SupportsChat          bool
	SupportsStreaming     bool
	SupportsVision        bool
	SupportsAudio         bool
	SupportsVideo         bool
	SupportsImage         bool
	SupportsEmbeddings    bool
	SupportsFunctions     bool
	SupportsJSON          bool
	SupportsReasoning     bool
	SupportsLongContext   bool
	SupportsTranscription bool
	SupportsTTS           bool
	SupportsRerank        bool
}

// ProviderStatus represents the current status of a provider
type ProviderStatus struct {
	Provider    ProviderType
	IsAvailable bool
	LastCheck   time.Time
	Error       string
	ModelsCount int
}

// ModelInfo contains information about a model
type ModelInfo struct {
	ID            string
	Name          string
	Provider      ProviderType
	Owner         string
	Description   string
	ContextLength int
	PriceInput    float64
	PriceOutput   float64
	Capabilities  []ModelCapability
	Categories    []string
	Tags          []string
	IsAvailable   bool
}

// CompletionRequest is a request for text completion
type CompletionRequest struct {
	Model          string
	Messages       []Message
	MaxTokens      int
	Temperature    float64
	TopP           float64
	Stop           []string
	Stream         bool
	Tools          []Tool
	ResponseFormat *ResponseFormat
	Metadata       map[string]interface{}
}

// Message represents a chat message
type Message struct {
	Role       MessageRole
	Content    string
	MultiModal []MultiModalPart
}

// MultiModalPart represents a part of a multimodal message
type MultiModalPart struct {
	Type string // "text", "image_url"
	Text string
	URL  string
}

// Tool represents a function/tool that can be called
type Tool struct {
	Type     string
	Function FunctionDefinition
}

// FunctionDefinition defines a function
type FunctionDefinition struct {
	Name        string
	Description string
	Parameters  interface{}
}

// ResponseFormat specifies the format of the response
type ResponseFormat struct {
	Type string // "json", "text"
}

// CompletionResponse is the response from a completion request
type CompletionResponse struct {
	ID           string
	Provider     ProviderType
	Model        string
	Content      string
	FinishReason string
	ToolCalls    []ToolCall
	Usage        TokenUsage
	Latency      time.Duration
	Metadata     map[string]interface{}
}

// ToolCall represents a call to a tool
type ToolCall struct {
	ID       string
	Type     string
	Function FunctionCall
}

// FunctionCall is a call to a function
type FunctionCall struct {
	Name      string
	Arguments string
}

// TokenUsage represents token usage statistics
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// StreamChunk represents a chunk of streaming response
type StreamChunk struct {
	ID           string
	Provider     ProviderType
	Model        string
	Delta        string
	FinishReason string
	Usage        *TokenUsage
}

// StreamingCallback is called for each chunk in streaming
type StreamingCallback func(chunk StreamChunk) error

// Errors
var (
	ErrAPIKeyMissing       = &ProviderError{Message: "API key is required"}
	ErrModelNotFound       = &ProviderError{Message: "Model not found"}
	ErrProviderUnavailable = &ProviderError{Message: "Provider is unavailable"}
)

// ProviderError represents an error from a provider
type ProviderError struct {
	Message   string
	Code      string
	Type      string
	Retryable bool
}

func (e *ProviderError) Error() string {
	return e.Message
}

func (e *ProviderError) IsRetryable() bool {
	return e.Retryable
}

// NewProviderError creates a new provider error
func NewProviderError(provider ProviderType, statusCode int, code, message string) *ProviderError {
	return &ProviderError{
		Message:   message,
		Code:      code,
		Type:      "provider_error",
		Retryable: statusCode == 429 || statusCode >= 500,
	}
}

// IsProviderError checks if an error is a ProviderError
func IsProviderError(err error) (*ProviderError, bool) {
	if pErr, ok := err.(*ProviderError); ok {
		return pErr, true
	}
	return nil, false
}

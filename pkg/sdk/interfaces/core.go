package interfaces

import (
	"context"
	"time"
)

// ========================================================================
// Core domain types — shared across interfaces, zero project imports
// ========================================================================

// AgentInfo represents agent metadata.
type AgentInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Provider  string    `json:"provider"`
	Model     string    `json:"model"`
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

// AgentResponse is the result of sending a prompt to an agent.
type AgentResponse struct {
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

// AgentStatus represents current agent health.
type AgentStatus struct {
	IsAvailable bool      `json:"is_available"`
	CurrentTask string    `json:"current_task"`
	Load        int       `json:"load"`
	LastSeen    time.Time `json:"last_seen"`
	SuccessRate float64   `json:"success_rate"`
}

// Task is a unit of work for an agent.
type Task struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Context     string                 `json:"context"`
	Inputs      map[string]interface{} `json:"inputs"`
	Timeout     time.Duration          `json:"timeout"`
}

// TaskResult is the outcome of a Task execution.
type TaskResult struct {
	Success  bool          `json:"success"`
	Output   string        `json:"output"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration"`
}

// AgentManifest is the public record of an agent for discovery.
type AgentManifest struct {
	ID           string            `json:"id"`
	DID          string            `json:"did"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Category     string            `json:"category"`
	Capabilities []AgentCapability `json:"capabilities"`
	Tags         []string          `json:"tags"`
}

// AgentCapability describes one thing an agent can do.
type AgentCapability struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// DomainRecord represents a resolved .mskt domain.
type DomainRecord struct {
	Name      string    `json:"name"`
	Owner     string    `json:"owner"`
	Addresses []string  `json:"addresses"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// WorkflowDef describes a named, multi-step workflow.
type WorkflowDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Steps       []WorkflowStep  `json:"steps"`
}

// WorkflowStep is one step in a workflow.
type WorkflowStep struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Capability string                 `json:"capability"`
	Input      map[string]interface{} `json:"input"`
	DependsOn  []string               `json:"depends_on"`
}

// WorkflowExecution is the runtime state of a running workflow.
type WorkflowExecution struct {
	ID        string                 `json:"id"`
	Workflow  string                 `json:"workflow"`
	State     string                 `json:"state"`
	StartedAt time.Time              `json:"started_at"`
	EndedAt   time.Time              `json:"ended_at"`
	Output    map[string]interface{} `json:"output"`
	Error     string                 `json:"error,omitempty"`
}

// A2AMessage is an agent-to-agent protocol message.
type A2AMessage struct {
	ID        string    `json:"id"`
	Source    string    `json:"source"`
	Target    string    `json:"target"`
	Type      string    `json:"type"`
	Payload   []byte    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

// AIRequest is a completion request to an AI provider.
type AIRequest struct {
	Model       string       `json:"model"`
	Messages    []AIMessage  `json:"messages"`
	MaxTokens   int          `json:"max_tokens"`
	Temperature float64      `json:"temperature"`
	Stream      bool         `json:"stream"`
}

// AIMessage is a single message in a chat conversation.
type AIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AIResponse is the response from an AI provider.
type AIResponse struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Content string   `json:"content"`
	Usage   *AIUsage `json:"usage,omitempty"`
}

// AIStreamChunk is one chunk of a streaming response.
type AIStreamChunk struct {
	Delta        string `json:"delta"`
	FinishReason string `json:"finish_reason"`
}

// AIModel describes a model offered by a provider.
type AIModel struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ContextLength int    `json:"context_length"`
	IsAvailable   bool   `json:"is_available"`
}

// AIUsage tracks token consumption.
type AIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// JournalEntry is one recorded event in a session journal.
type JournalEntry struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// ========================================================================
// 1.  NodeInterface — P2P node lifecycle and core operations
// ========================================================================
type NodeInterface interface {
	PublishIdentity(ctx context.Context) error
	ResolveDomain(ctx context.Context, name string) (*DomainRecord, error)
	Connect(ctx context.Context, addr string) error
	Close() error
}

// ========================================================================
// 2.  SessionInterface — session lifecycle and execution
// ========================================================================
type SessionInterface interface {
	ID() string
	Status() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Execute(ctx context.Context, task *Task) (*TaskResult, error)
	State() map[string]interface{}
}

// ========================================================================
// 3.  IdentityInterface — DIDs, signing, key management
// ========================================================================
type IdentityInterface interface {
	DID() string
	Sign(data []byte) ([]byte, error)
	Verify(data, sig []byte) error
	ResolvePublicKey(did string) ([]byte, error)
	Encrypt(plain []byte) ([]byte, error)
	Decrypt(cipher []byte) ([]byte, error)
}

// ========================================================================
// 4.  CommunicationInterface — channels, publish/subscribe
// ========================================================================
type CommunicationInterface interface {
	Publish(ctx context.Context, channelID string, msg []byte) error
	Subscribe(ctx context.Context, channelID string, handler MessageHandler) (Subscription, error)
}

// MessageHandler processes an incoming channel message.
type MessageHandler func(msg []byte)

// Subscription represents an active channel subscription.
type Subscription interface {
	ID() string
	Close() error
}

// ========================================================================
// 5.  AgentInterface — agent contract
// ========================================================================
type AgentInterface interface {
	Info() *AgentInfo
	SendMessage(ctx context.Context, prompt string) (*AgentResponse, error)
	ExecuteTask(ctx context.Context, task *Task) (*TaskResult, error)
	Capabilities() []string
	Status() *AgentStatus
	IsAvailable() bool
	Close() error
}

// ========================================================================
// 6.  WorkflowInterface — workflow registration and execution
// ========================================================================
type WorkflowInterface interface {
	Register(name string, wf *WorkflowDef) error
	Execute(ctx context.Context, workflowName string, input map[string]interface{}) (*WorkflowExecution, error)
	CancelExecution(id string) error
}

// ========================================================================
// 7.  StorageInterface — content-addressed block storage
// ========================================================================
type StorageInterface interface {
	Get(cid string) ([]byte, error)
	Put(cid string, data []byte, did string) error
	Size() int64
	List(prefix string) ([]string, error)
	Close() error
}

// ========================================================================
// 8.  SecurityInterface — encryption, auth, signing
// ========================================================================
type SecurityInterface interface {
	Encrypt(data []byte) ([]byte, error)
	Decrypt(data []byte) ([]byte, error)
	CheckAuth(principal, resource, action string) error
	Sign(data []byte) ([]byte, error)
	Verify(data, sig []byte) error
}

// ========================================================================
// 9.  A2AInterface — agent-to-agent protocol
// ========================================================================
type A2AInterface interface {
	Send(ctx context.Context, target string, msg *A2AMessage) error
	Receive(ctx context.Context) (*A2AMessage, error)
	RegisterHandler(handler func(ctx context.Context, msg *A2AMessage) (*A2AMessage, error)) error
}

// ========================================================================
// 10. EventBus — central event pub/sub
// ========================================================================
type EventBus interface {
	Publish(eventType string, payload interface{})
	Subscribe(eventType string, handler func(eventType string, payload interface{}))
	Unsubscribe(eventType string)
}

// ========================================================================
// 11. AIInterface — AI/LLM provider abstraction
// ========================================================================
type AIInterface interface {
	Complete(ctx context.Context, req *AIRequest) (*AIResponse, error)
	StreamComplete(ctx context.Context, req *AIRequest, cb func(chunk *AIStreamChunk)) error
	ListModels(ctx context.Context) ([]*AIModel, error)
	Name() string
	IsAvailable() bool
}

// ========================================================================
// 12. UIBridgeInterface — WebSocket / REST live updates
// ========================================================================
type UIBridgeInterface interface {
	Send(event string, data interface{}) error
	On(event string, handler func(data interface{}))
	Broadcast(event string, data interface{}) error
}

// ========================================================================
// 13. JournalInterface — session journal / history
// ========================================================================
type JournalInterface interface {
	Record(entry *JournalEntry) error
	Query(filter map[string]interface{}) ([]*JournalEntry, error)
	Recent(n int) ([]*JournalEntry, error)
}

// ========================================================================
// 14. SyncInterface — multi-device / CRDT sync
// ========================================================================
type SyncInterface interface {
	Sync(ctx context.Context, data []byte) error
	OnSync(handler func(data []byte))
	GetState(key string) ([]byte, error)
	SetState(key string, value []byte) error
}

// ========================================================================
// 15. DiscoveryInterface — peer and agent discovery
// ========================================================================
type DiscoveryInterface interface {
	Index(manifest *AgentManifest) error
	Search(query string) ([]*AgentManifest, error)
	Recommend(tags ...string) ([]*AgentManifest, error)
	FindPeers(ctx context.Context, topic string) ([]string, error)
}

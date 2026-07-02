# PHASE 1 - Repository Discovery Report

## Overview
This document contains the complete internal map of the Musketeers repository as of the discovery phase.

---

## 1. REST APIs (api/rest.go)

### Existing Endpoints

#### Core System
- `/api/identity` - Handle identity operations
- `/api/search` - Search functionality
- `/api/resolve` - Resolve operations
- `/api/content` - Content management (GET/PUT)
- `/api/health` - Health check

#### ACP (Agent Communication Protocol)
- `/api/acp/task` - Handle ACP tasks
- `/api/acp/tasks` - List supported ACP tasks

#### Domain
- `/api/domain/commit` - Domain commit operations

#### Channels (GossipSub)
- `/api/channels/create` - Create channel
- `/api/channels/leave` - Leave channel
- `/api/channels/join` - Join channel
- `/api/channels/publish` - Publish to channel
- `/api/channels/list` - List channels
- `/api/channels/messages` - Get channel messages

#### Sessions
- `/api/sessions` - Session management (GET/POST)
  - POST: Create session with name, owner_did, manager_agent_id, assistant_agents
  - GET: List all sessions
- `/api/sessions/{id}` - Session operations
  - GET: Get session details
  - PUT: Update session
  - DELETE: Delete session
  - POST with action=pause: Pause session
  - POST with action=resume: Resume session
  - POST with action=complete: Complete session
  - POST with action=register_human: Register human client
  - POST with action=register_agent: Register agent instance

#### Messages
- `/api/messages` - Message operations
- `/api/messages/{session_id}` - Messages by session

#### Tasks
- `/api/tasks` - Task operations
- `/api/tasks/{session_id}` - Tasks by session

#### Progress
- `/api/progress` - Progress tracking
- `/api/progress/{session_id}` - Progress by session

#### Memory
- `/api/memory` - Memory operations
- `/api/memory/{session_id}` - Memory by session

#### Knowledge
- `/api/knowledge` - Knowledge operations
- `/api/knowledge/{session_id}` - Knowledge by session
- `/api/knowledge/search` - Knowledge search

#### Skills
- `/api/skills` - Skills operations
- `/api/skills/{session_id}` - Skills by session

#### Artifacts
- `/api/artifacts` - Artifacts operations
- `/api/artifacts/{session_id}` - Artifacts by session

#### Bridges
- `/api/bridges` - Bridge operations
- `/api/bridges/{id}` - Bridge by ID

#### Agents
- `/api/agents` - Agent operations
- `/api/agents/{id}` - Agent by ID

#### MCP (Model Context Protocol)
- `/api/mcp/servers` - MCP servers
- `/api/mcp/servers/{id}` - MCP server by ID
- `/api/mcp/tools` - MCP tools
- `/api/mcp/tools/{id}` - MCP tool by ID

#### WebSocket
- `/api/ws` - WebSocket endpoint

#### Dashboard
- `/dashboard` - Dashboard HTML
- `/dashboard/` - Dashboard HTML

### Server Components
- SessionManager (sessioncore.UnifiedSessionManager)
- ChatManagers (map[string]*session.ChatManager)
- TaskManagers (map[string]*session.TaskManager)
- ProgressTrackers (map[string]*session.ProgressTracker)
- Memories (map[string]*session.CollectiveMemory)
- SkillsManagers (map[string]*session.SkillsManager)
- Artifacts (map[string][]Artifact)
- BridgeManager (session.SessionBridgeManager)
- MCPServers (map[string]*MCPServer)
- MCPTools (map[string]*MCPTool)
- EventBus (eventbus.EventBus)
- RateLimiter (security.RateLimiter)

---

## 2. WebSocket APIs (api/local_ws_bridge.go)

### WebSocket Bridge
- Local WebSocket bridge for real-time communication
- Handles streaming data
- Connects to backend services

---

## 3. Agent System (pkg/agent)

### Core Components

#### Agent Registry (pkg/agent/registry.go)
- AgentRegistry: Manages agent registration and tracking
- HumanClientStatus: Human client state management
- AgentMetadata: Agent metadata (type, provider, model, etc.)
- AgentStats: Agent statistics (tasks, tokens, success rate)
- Methods: Register, Unregister, Get, List, UpdateStats, GetStats

#### Unified Agent (pkg/agent/unified/unified_agent.go)
- UnifiedAgent: Main agent integrating all systems
- UnifiedSkillManager: Skill management
- UnifiedMemoryManager: Memory management
- SubagentManager: Subagent management
- AutomationManager: Automation operations
- SkillDirector: Skill direction
- MultiLayerValidator: Multi-layer validation
- Coordinator: Central coordination
- FlowManager: Flow management
- ErrorHandler: Error handling
- CollectiveSystem: Collective agent system
- SessionEventBus: Session event bus
- RealTimeMemorySync: Real-time memory sync
- RealTimeSkillSync: Real-time skill sync
- ProblemSolutionRegistry: Problem/solution registry
- LocalMemoryCache: Local memory cache
- DataCurator: Data curation
- TaskScheduler: Task scheduling
- AgentSyncManager: Agent sync management
- ProviderRegistry: Provider integration
- Router: Provider routing
- ToolExecutor: Tool execution
- ThinkingEngine: Deep AI thought process
- WiringLayer: Automatic adapter connection
- SessionContainer: Session container reference
- SessionManager: Session management
- AgentPool: Agent pool management
- Metrics: Performance monitoring

#### Subagents (pkg/agent/subagents/)
- SubagentManager: Manages specialized subagents
- Subagent types: Manager, Planner, Architect, Researcher, Coder, Reviewer, Memory, Executor, Validator, Observer

#### Thinking Engine (pkg/agent/thinking/)
- ThinkingEngine: Deep AI thought process (222KB - largest file)
- ContextReranker: Context ranking
- CodeIndexer: Code indexing
- Embeddings: Embedding operations
- JSONParser: JSON parsing
- SessionAdaptors: Session adaptation
- SystemPrompts: System prompt management
- TokenCounter: Token counting

#### Tools (pkg/agent/tools/)
- ToolExecutor: Tool execution (27KB)
- ToolRegistry: Tool registration
- FileLock: File locking
- Tool types: CLI, IDE, Browser, Desktop, Custom

#### Adapters (pkg/agent/adapters/)
- BrowserAdapter: Browser integration
- CLIAdapter: CLI integration
- CustomAdapter: Custom adapter
- DesktopAdapter: Desktop integration
- IDEAdapter: IDE integration
- IDEExtensionAdapter: IDE extension integration
- InstanceManager: Instance management
- MultiCLIAdapter: Multi-CLI integration
- MultiDesktopAdapter: Multi-desktop integration
- MultiIDEAdapter: Multi-IDE integration

#### Wiring (pkg/agent/wiring/)
- WiringLayer: Automatic adapter connection
- Adapters: Adapter management

#### Other Agent Components
- Automation (pkg/agent/automation/): AutomationManager
- Collaboration (pkg/agent/collaboration/): Workflow
- Direction (pkg/agent/direction/): SkillDirector
- Integration (pkg/agent/integration/): CollectiveAgentSystem
- Learning (pkg/agent/learning/): LearningEngine
- Memory (pkg/agent/memory/): CollectiveMemory
- Quality (pkg/agent/quality/): QualityChecker
- Skills (pkg/agent/skills/): SkillManager
- Tasks (pkg/agent/tasks/): Task management
- Tracking (pkg/agent/tracking/): Tracking
- Validation (pkg/agent/validation/): Validation

---

## 4. Session Manager (pkg/session/)

### Core Components

#### Session Container (pkg/session/container.go)
- SessionContainer: Main session container (44KB)
- Manages session lifecycle
- Integrates with all subsystems

#### Chat Manager (pkg/session/chat.go)
- ChatManager: Chat operations
- Message handling
- Conversation management

#### Task Manager (pkg/session/task_manager.go)
- TaskManager: Task operations (18KB)
- Task scheduling
- Task tracking

#### Progress Tracker (pkg/session/progress_tracker.go)
- ProgressTracker: Progress tracking (15KB)
- Progress updates
- Status monitoring

#### Memory (pkg/session/memory.go)
- CollectiveMemory: Collective memory (18KB)
- Memory operations
- Knowledge management

#### Skills (pkg/session/skills.go)
- SkillsManager: Skills management
- Skill operations

#### Tool Handlers (pkg/session/tool_handlers.go)
- ToolHandlers: Tool handling (30KB)
- Tool execution
- Tool management

#### Workflow (pkg/session/workflow.go)
- Workflow: Workflow management

#### Journal (pkg/session/journal.go)
- Journal: Session journal
- Event logging

#### Other Session Components
- Advanced (pkg/session/advanced/): Advanced features
- Connection (pkg/session/connection/): Connection management
- Sessions (pkg/session/sessions/): Session storage
- CapabilityVerifier: Capability verification
- FinalReviewer: Final review
- HandoffManager: Handoff management
- Retry: Retry logic
- SessionBridge: Session bridging
- SessionBridgeManager: Bridge management
- Performance tests

---

## 5. Provider Layer (pkg/providers/)

### Core Components

#### Provider Registry (pkg/providers/register.go)
- ProviderRegistry: Manages all providers
- Global registry instance
- Methods: Register, Get, List, ListByType, GetProviderByName

#### Router (pkg/providers/router.go)
- Router: Smart router for intelligent model selection
- RouterConfig: Router configuration
- UsageStats: Usage statistics
- Model caching
- Candidate ranking
- Retry logic
- Fallback support

#### Types (pkg/providers/types.go)
- ProviderType: 23 official providers + local + custom
  - Official: OpenAI, Anthropic, Google, DeepSeek, XAI, Mistral, Qwen, Moonshot, NVIDIA, Xiaomi, ZAI, Tencent, StepFun, Poolside, Recraft, Sourceful, OpenRouter, Cohere, Groq, TogetherAI, Perplexity, Minimax
  - Local: Ollama
  - Custom: Custom
- ModelCapability: Text, Code, Vision, Audio, Video, Image, Embeddings, Streaming, Function, Reasoning, LongContext, Transcription, TTS, Rerank, Search
- MessageRole: System, User, Assistant, Tool
- Provider interface: Type, Name, Capabilities, Initialize, Close, Ping, Status, IsAvailable, ListModels, GetModel, Complete, StreamComplete
- ProviderConfig: APIKey, BaseURL, Timeout, Extra
- ProviderCapabilities: Support flags
- ProviderStatus: Status information
- ModelInfo: Model information
- CompletionRequest: Request structure
- Message: Message structure
- Tool: Tool structure
- CompletionResponse: Response structure
- TokenUsage: Token statistics
- StreamChunk: Streaming chunk
- ProviderError: Error handling

#### API Key Manager (pkg/providers/api_key_manager.go)
- APIKeyManager: API key management
- Key storage
- Key validation

#### Model Catalog (pkg/providers/model_catalog.go)
- ModelCatalog: Model catalog
- Model information

#### Free Models Tracker (pkg/providers/free_models_tracker.go)
- FreeModelsTracker: Free model tracking
- Cost optimization

#### Free Router (pkg/providers/free_router.go)
- FreeRouter: Free model routing
- Cost-aware routing

#### Built-in Providers (pkg/providers/builtin/)
- OpenAI (openai/)
- Anthropic (anthropic/)
- Google (google/)
- DeepSeek (deepseek/)
- Groq (groq/)
- Mistral (mistral/)
- Ollama (ollama/)
- OpenRouter (openrouter/)
- Custom (custom/)
- And 14 more providers

---

## 6. Event Bus (pkg/eventbus/)

### Components
- EventBus: Main event bus
- DLQ: Dead letter queue
- Event publishing
- Event subscription
- Event filtering

---

## 7. Memory System (pkg/memory/)

### Components
- Cache (pkg/memory/cache/): Memory caching
- Core (pkg/memory/core/): Core memory operations
- Integration (pkg/memory/integration/): Memory integration
- Storage (pkg/memory/storage/): Memory storage
- Sync (pkg/memory/sync/): Memory synchronization
- Types (pkg/memory/types/): Memory types

---

## 8. Orchestrator (pkg/orchestrator/)

### Components

#### Core Orchestrator
- OrchestratorEngine: Main orchestrator engine (17KB)
- AgentLifecycle: Agent lifecycle management
- RoleAssigner: Role assignment
- SessionManager: Session management

#### Protocols
- A2AProtocol: Agent-to-agent protocol (18KB)
- MCPProtocol: MCP protocol (19KB)
- ChatConnector: Chat connection

#### Integration
- ExternalPlatforms: External platform integration (15KB)
- StorageConnector: Storage integration (10KB)
- SessionEventBroadcaster: Session event broadcasting (11KB)

#### Management
- DelegationManager: Delegation management
- FailureHandler: Failure handling
- FinalReviewer: Final review
- Aggregator: Aggregation

#### Email System
- EmailSystem: Email operations (19KB)
- Email mailbox integration

#### Logging
- ComprehensiveLogger: Comprehensive logging (11KB)

---

## 9. Node System (pkg/node/)

### Components
- Node: Main node (17KB)
- SessionLifecycle: Session lifecycle (28KB)
- SessionBridge: Session bridging (11KB)
- Direct: Direct operations (10KB)
- ChannelOps: Channel operations
- DomainOps: Domain operations
- SessionWiring: Session wiring (11KB)
- Validator: Validation
- ACP: ACP operations
- Subsystems (pkg/node/subsystems/): Node subsystems

---

## 10. Configuration System (pkg/config/)

### Components
- Config: Configuration management
- Config validation
- Config loading

---

## 11. Storage System (pkg/storage/)

### Components
- Erasure: Erasure coding
- Quota: Quota management
- Storage operations

---

## 12. Metrics System (pkg/metrics/)

### Components
- Metrics: Performance metrics
- Metric collection
- Metric reporting

---

## 13. Other Important Systems

### ACP (Agent Communication Protocol) (pkg/acp/)
- Handler: ACP handler
- Message: ACP message
- Tasks: ACP tasks
- Transport: ACP transport

### Agent Bridge (pkg/agent_bridge/)
- Client: Bridge client
- Server: Bridge server
- MultiplexedBridge: Multiplexed bridging
- SessionManager: Session management
- TaskProtocol: Task protocol
- Tools: Tool bridging
- Middleware: Bridge middleware

### Channel (pkg/channel/)
- Private: Private channels
- PubSub: PubSub channels

### Capability (pkg/capability/)
- Manager: Capability management
- Types: Capability types

### CEO (pkg/ceo/)
- Supervisor: CEO supervisor

### Common (pkg/common/)
- Common utilities

### Content (pkg/content/)
- Content management

### Crypto (pkg/crypto/)
- Cryptographic operations

### Delegation (pkg/delegation/)
- Delegation operations

### Discovery (pkg/discovery/)
- Service discovery

### Email (pkg/email/)
- Email operations

### Events (pkg/events/)
- Event management

### Gateway (pkg/gateway/)
- Gateway operations

### Hosting (pkg/hosting/)
- Hosting operations

### Identity (pkg/identity/)
- Identity management

### Integration (pkg/integration/)
- Integration operations

### Ledger (pkg/ledger/)
- Ledger operations

### Limits (pkg/limits/)
- Rate limiting

### Logger (pkg/logger/)
- Logging

### Mailbox (pkg/mailbox/)
- Mailbox operations

### Naming (pkg/naming/)
- Naming operations

### Network (pkg/network/)
- Network operations

### Notifications (pkg/notifications/)
- Notifications

### Plugins (pkg/plugins/)
- Plugin system

### Policy (pkg/policy/)
- Policy management

### Protocol (pkg/protocol/)
- Protocol operations

### Rate (pkg/rate/)
- Rate limiting

### Recovery (pkg/recovery/)
- Recovery operations

### Registry (pkg/registry/)
- Registry operations

### Runtime (pkg/runtime/)
- Runtime operations

### Sandbox (pkg/sandbox/)
- Sandbox operations

### SDK (pkg/sdk/)
- SDK operations

### Search (pkg/search/)
- Search operations

### Security (pkg/security/)
- Security operations

### Skills (pkg/skills/)
- Skills operations

### Timeout (pkg/timeout/)
- Timeout operations

### Upgrade (pkg/upgrade/)
- Upgrade operations

### Validation (pkg/validation/)
- Validation operations

### Vault (pkg/vault/)
- Vault operations

### Verification (pkg/verification/)
- Verification operations

### Workflow (pkg/workflow/)
- Workflow operations

---

## 14. Command Line Interfaces (cmd/)

### Main Commands
- cmd/main.go: Main entry point
- cmd/studio/main.go: Studio server
- cmd/agent/main.go: Agent server
- cmd/founder/main.go: Founder server
- cmd/gateway/main.go: Gateway server
- cmd/seed/main.go: Seed server

---

## 15. Dashboard (api/dashboard.go)

### Current State
- Simple Cursor-style UI
- Session management
- Agent registration
- Channel management
- Basic statistics
- Modal-based forms

---

## 16. Configuration Files

### Config
- config.example.yaml: Example configuration
- models.json: Model catalog

---

## 17. Documentation

### Engineering Docs
- Multiple markdown files documenting architecture, dependencies, integration, etc.
- Located in root directory and docs/engineering/

---

## 18. Summary Statistics

### Package Count
- Total packages: ~50+
- Agent system: 15+ sub-packages
- Providers: 23+ built-in providers
- Session system: 15+ components
- Orchestrator: 15+ components

### API Endpoints
- REST endpoints: 40+
- WebSocket endpoints: 1

### File Sizes (Notable)
- pkg/agent/thinking/thinking_engine.go: 222KB (largest)
- pkg/session/container.go: 44KB
- pkg/agent/tools/executor.go: 27KB
- pkg/agent/unified/unified_agent.go: 45KB (truncated in view)

### Provider Support
- Official providers: 23
- Local providers: 1 (Ollama)
- Custom providers: 1

---

## 19. Key Findings

### Strengths
1. Comprehensive agent system with multiple specialized agents
2. Extensive provider support (23+ providers)
3. Rich session management system
4. Advanced thinking engine
5. Multiple adapter types (IDE, CLI, Browser, Desktop)
6. Event-driven architecture
7. Memory management system
8. Tool execution system
9. MCP protocol support
10. WebSocket support for real-time communication

### Areas for Dashboard Integration
1. Provider management UI needed
2. Model selection UI needed
3. Agent orchestration visualization needed
4. Real-time metrics display needed
5. Log viewer needed
6. Event timeline needed
7. System graph needed
8. API explorer needed
9. Configuration UI needed
10. File explorer needed

### Existing Dashboard Capabilities
1. Session creation/management
2. Agent registration
3. Channel management
4. Basic statistics
5. Simple UI (Cursor-style)

### Missing Dashboard Capabilities
1. Provider configuration UI
2. Model assignment UI
3. Real-time agent orchestration view
4. Tool registry UI
5. Memory inspection UI
6. File explorer UI
7. System health UI
8. Observability UI
9. Log viewer UI
10. Event timeline UI
11. System graph UI
12. API explorer UI
13. Configuration UI
14. IDE integration UI
15. CLI integration UI

---

## Next Steps

Proceed to PHASE 2 - Capability Inventory to determine:
- What exists for each Dashboard feature
- What is partially implemented
- What is missing
- What is broken
- What is unused
- What is deprecated

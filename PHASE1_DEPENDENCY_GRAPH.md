# Phase 1: Dependency Graph

## Core Dependencies

```
cmd/studio/main.go
├── api/
│   ├── rest.go (REST API Server)
│   ├── dashboard.go (Dashboard HTML/JS)
│   ├── providers_runtime.go (Provider Runtime)
│   └── local_ws_bridge.go (WebSocket Bridge)
├── pkg/
│   ├── agent/
│   │   ├── registry.go (Agent Registry)
│   │   ├── adapters/ (CLI, IDE, Browser, Custom Adapters)
│   │   ├── unified/ (UnifiedAgent, AgentPool, SessionManager)
│   │   ├── thinking/ (ThinkingEngine)
│   │   ├── tools/ (ToolExecutor, ToolRegistry)
│   │   ├── subagents/ (SubagentManager)
│   │   ├── automation/ (AutomationManager)
│   │   ├── direction/ (SkillDirector)
│   │   ├── validation/ (MultiLayerValidator)
│   │   ├── skills/ (SkillManager)
│   │   ├── memory/ (CollectiveMemory)
│   │   ├── collaboration/ (Workflow)
│   │   └── learning/ (LearningEngine)
│   ├── orchestrator/
│   │   ├── orchestrator_engine.go (Orchestrator Engine)
│   │   ├── connector.go (Connector)
│   │   ├── email_system.go (Email System)
│   │   ├── storage_connector.go (Storage Connector)
│   │   ├── session_manager.go (Session Manager)
│   │   └── delegation_manager.go (Delegation Manager)
│   ├── providers/
│   │   ├── router.go (Smart Router)
│   │   ├── free_router.go (Free Router)
│   │   ├── builtin/ (23 LLM Providers)
│   │   └── api_key_manager.go (API Key Manager)
│   ├── session/
│   │   ├── container.go (Session Container)
│   │   ├── session_bridge.go (Session Bridge)
│   │   ├── session_bridge_manager.go (Bridge Manager)
│   │   ├── task_manager.go (Task Manager)
│   │   ├── workflow.go (Workflow)
│   │   ├── memory.go (Memory)
│   │   ├── skills.go (Skills)
│   │   ├── journal.go (Journal)
│   │   ├── progress_tracker.go (Progress Tracker)
│   │   ├── handoff_manager.go (Handoff Manager)
│   │   ├── final_reviewer.go (Final Reviewer)
│   │   ├── capability_verifier.go (Capability Verifier)
│   │   ├── aggregator.go (Aggregator)
│   │   ├── retry.go (Retry)
│   │   ├── placeholders.go (Placeholders)
│   │   ├── chat.go (Chat)
│   │   └── tool_handlers.go (Tool Handlers)
│   ├── node/
│   │   ├── node.go (P2P Node)
│   │   ├── session_lifecycle.go (Session Lifecycle)
│   │   ├── session_bridge.go (Session Bridge)
│   │   ├── session_wiring.go (Session Wiring)
│   │   ├── direct.go (Direct Connection)
│   │   ├── domain_ops.go (Domain Operations)
│   │   ├── channel_ops.go (Channel Operations)
│   │   ├── validator.go (Validator)
│   │   ├── config.go (Config)
│   │   ├── acp.go (ACP)
│   │   └── subsystems/ (Identity, Messaging, Network, Security, Storage)
│   ├── eventbus/
│   │   ├── bus.go (Event Bus)
│   │   └── dlq.go (Dead Letter Queue)
│   ├── ceo/
│   │   └── supervisor.go (CEO Supervisor)
│   ├── verification/
│   │   └── multi_stage_verifier.go (Multi-Stage Verifier)
│   ├── acp/
│   │   ├── router.go (ACP Router)
│   │   ├── tasks.go (ACP Tasks)
│   │   ├── approvals.go (Approvals)
│   │   ├── engine.go (Policy Engine)
│   │   └── types.go (Types)
│   ├── policy/
│   │   ├── engine.go (Policy Engine)
│   │   ├── approvals.go (Approvals)
│   │   └── types.go (Types)
│   ├── capability/
│   │   ├── manager.go (Capability Manager)
│   │   ├── github/ (GitHub Capability)
│   │   ├── gmail/ (Gmail Capability)
│   │   ├── messaging/ (Messaging Capability)
│   │   └── pipeline/ (Pipeline Capability)
│   ├── agent_bridge/
│   │   ├── multiplexed_bridge.go (Multiplexed Bridge)
│   │   ├── client.go (Client)
│   │   ├── server.go (Server)
│   │   ├── session_manager.go (Session Manager)
│   │   ├── task_protocol.go (Task Protocol)
│   │   ├── tools.go (Tools)
│   │   ├── middleware.go (Middleware)
│   │   └── protocol/ (Protocol)
│   ├── integration/
│   │   ├── agent_communication.go (Agent Communication)
│   │   ├── agent_session_integration.go (Agent Session Integration)
│   │   ├── instance_session_integration.go (Instance Session Integration)
│   │   ├── session_orchestrator.go (Session Orchestrator)
│   │   ├── task_routing.go (Task Routing)
│   │   ├── role_assignment.go (Role Assignment)
│   │   ├── webhook_router.go (Webhook Router)
│   │   └── sessions/ (Sessions)
│   ├── delegation/
│   │   ├── advanced.go (Advanced Delegation)
│   │   ├── advanced_test.go (Advanced Delegation Test)
│   │   └── integration.go (Integration)
│   ├── discovery/
│   │   ├── discovery.go (Discovery)
│   │   └── discovery_test.go (Discovery Test)
│   ├── hosting/
│   │   ├── hosting.go (Hosting)
│   │   ├── hosting_types.go (Hosting Types)
│   │   ├── integration.go (Integration)
│   │   ├── p2p_hosting_service.go (P2P Hosting Service)
│   │   ├── site_uploader.go (Site Uploader)
│   │   └── hosting_test.go (Hosting Test)
│   ├── email/
│   │   ├── email.go (Email)
│   │   ├── email_store.go (Email Store)
│   │   ├── email_types.go (Email Types)
│   │   ├── integration.go (Integration)
│   │   ├── p2p_email_service.go (P2P Email Service)
│   │   └── email_test.go (Email Test)
│   ├── analytics/
│   │   ├── integration.go (Integration)
│   │   └── core/ (Analytics Core)
│   ├── backup/
│   │   ├── integration.go (Integration)
│   │   └── core/ (Backup Core)
│   ├── notifications/
│   │   ├── integration.go (Integration)
│   │   └── core/ (Notifications Core)
│   ├── plugins/
│   │   ├── integration.go (Integration)
│   │   └── core/ (Plugins Core)
│   ├── upgrade/
│   │   ├── integration.go (Integration)
│   │   └── core/ (Upgrade Core)
│   ├── config/
│   │   └── config.go (Config)
│   ├── logger/
│   │   └── logger.go (Logger)
│   ├── limits/
│   │   └── limits.go (Limits)
│   ├── timeout/
│   │   └── timeout.go (Timeout)
│   ├── validation/
│   │   └── validator.go (Validator)
│   ├── ledger/
│   │   └── ledger.go (Ledger)
│   ├── sandbox/
│   │   └── executor.go (WASM Sandbox Executor)
│   ├── storage/
│   │   ├── quota.go (Quota Manager)
│   │   └── erasure.go (Erasure Coding)
│   ├── memory/
│   │   ├── core/ (Memory Core)
│   │   ├── cache/ (Memory Cache)
│   │   ├── storage/ (Memory Storage)
│   │   ├── types/ (Memory Types)
│   │   ├── integration/ (Memory Integration)
│   │   ├── sync/ (Memory Sync)
│   │   └── README.md (README)
│   ├── skills/
│   │   ├── core/ (Skills Core)
│   │   ├── direction/ (Skills Direction)
│   │   ├── evolution/ (Skills Evolution)
│   │   ├── sync/ (Skills Sync)
│   │   ├── types/ (Skills Types)
│   │   └── README.md (README)
│   ├── workflow/
│   │   ├── workflow.go (Workflow)
│   │   ├── engine.go (Workflow Engine)
│   │   ├── checkpoint.go (Checkpoint)
│   │   └── templates/ (Workflow Templates)
│   ├── security/
│   │   ├── ratelimit.go (Rate Limiter)
│   │   ├── tls.go (TLS)
│   │   └── core/ (Security Core)
│   ├── runtime/
│   │   ├── runtime.go (Runtime)
│   │   ├── context.go (Context)
│   │   ├── events/ (Runtime Events)
│   │   ├── knowledge/ (Runtime Knowledge)
│   │   ├── lifecycle/ (Runtime Lifecycle)
│   │   ├── observability/ (Runtime Observability)
│   │   ├── scheduler/ (Runtime Scheduler)
│   │   ├── state/ (Runtime State)
│   │   └── sandbox/ (Runtime Sandbox)
│   ├── network/domain/
│   │   ├── p2p_dns_resolver.go (P2P DNS Resolver)
│   │   ├── local_dns_proxy.go (Local DNS Proxy)
│   │   ├── http_proxy.go (HTTP Proxy)
│   │   └── system_proxy.go (System Proxy)
│   ├── crypto/
│   │   └── crypto.go (Crypto)
│   ├── identity/
│   │   └── identity.go (Identity)
│   ├── naming/
│   │   └── naming.go (Naming)
│   ├── protocol/
│   │   └── protocol.go (Protocol)
│   ├── metrics/
│   │   └── metrics.go (Metrics)
│   ├── cache/
│   │   └── cache.go (Cache)
│   ├── channel/
│   │   └── channel.go (Channel)
│   ├── common/
│   │   └── common.go (Common)
│   ├── mailbox/
│   │   └── mailbox.go (Mailbox)
│   ├── rate/
│   │   └── rate.go (Rate)
│   ├── recovery/
│   │   └── recovery.go (Recovery)
│   ├── registry/
│   │   └── registry.go (Registry)
│   ├── search/
│   │   └── search.go (Search)
│   └── vault/
│       └── keyprovider/ (Key Provider)
└── github.com/dgraph-io/badger/v4 (BadgerDB)
```

## Dependency Relationships

### High-Level Dependencies
```
main.go
├── Node (P2P Network)
├── EventBus (Event System)
├── BadgerDB (Persistence)
├── AgentRegistry (Agent Management)
├── UnifiedAgent (Agent Coordination)
├── OrchestratorEngine (Task Orchestration)
├── ProviderRegistry (LLM Providers)
├── SmartRouter (Model Selection)
├── CEOSupervisor (Health Monitoring)
├── SessionManager (Session Management)
├── SessionBridgeManager (Session Bridging)
├── REST API Server (HTTP API)
├── WebSocket Bridge (Real-time Communication)
└── Isolated Packages (Analytics, Backup, Delegation, etc.)
```

### Component Interdependencies
```
EventBus
├── Used by: All components
└── Subscribers: EmailManager, UnifiedAgent, OrchestratorEngine, etc.

AgentRegistry
├── Used by: OrchestratorEngine, UnifiedAgent, CEOSupervisor
└── Contains: CLI Adapter, IDE Adapter, Browser Adapter, Custom Adapter

ProviderRegistry
├── Used by: UnifiedAgent, SmartRouter, REST API
└── Contains: 23 LLM Providers (Mistral, OpenRouter, Qwen, etc.)

SessionContainer
├── Used by: UnifiedAgent, OrchestratorEngine
└── Manages: Session State, Memory, Skills, Workflow

UnifiedAgent
├── Uses: ProviderRegistry, SmartRouter, SessionContainer, EventBus
├── Contains: AgentPool, SessionManager, ThinkingEngine, ToolExecutor
└── Coordinates: All agents in the session

OrchestratorEngine
├── Uses: AgentRegistry, UnifiedAgent, Connector, PolicyEngine
├── Contains: CapabilityMatcher, RoleAssigner, Verifier
└── Coordinates: Task execution across agents

SmartRouter
├── Uses: ProviderRegistry
├── Contains: UsageTracker, ModelCache
└── Selects: Best model for each request
```

## External Dependencies
```
github.com/dgraph-io/badger/v4 (Database)
github.com/sirupsen/logrus (Logging)
go.uber.org/zap (Logging)
github.com/libp2p/go-libp2p (P2P)
github.com/libp2p/go-libp2p-pubsub (PubSub)
github.com/gorilla/websocket (WebSocket)
github.com/google/uuid (UUID)
github.com/MortalArena/Musketeers (Internal packages)
```

## Circular Dependencies
**None detected** - The architecture appears to be designed to avoid circular dependencies through interface-based design.

## Missing Dependencies
**None detected** - All imports in main.go resolve successfully.

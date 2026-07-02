# PHASE 3 - Connection Map Report

## Overview
This document provides the complete dependency graph showing how the Dashboard should connect to the backend systems through REST APIs, WebSocket, and services.

---

## 1. High-Level Architecture Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                         DASHBOARD                                │
│  (HTML/JS SPA - Cursor-style Engineering Console)              │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         │ HTTP/HTTPS
                         │ WebSocket
                         │
┌────────────────────────▼────────────────────────────────────────┐
│                      REST API LAYER                              │
│  (api/rest.go - HTTP Handlers & Middleware)                    │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         │ Service Calls
                         │
┌────────────────────────▼────────────────────────────────────────┐
│                    SERVICE LAYER                                 │
│  (Session Manager, Agent Registry, Provider Registry, etc.)    │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         │ Runtime Operations
                         │
┌────────────────────────▼────────────────────────────────────────┐
│                    RUNTIME LAYER                                 │
│  (UnifiedAgent, Orchestrator, ThinkingEngine, etc.)            │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         │ Agent Operations
                         │
┌────────────────────────▼────────────────────────────────────────┐
│                    AGENT SYSTEM                                 │
│  (Manager, Planner, Architect, Coder, Reviewer, etc.)          │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         │ Memory Operations
                         │
┌────────────────────────▼────────────────────────────────────────┐
│                    MEMORY SYSTEM                                │
│  (Working Memory, Long Memory, Vector DB, etc.)               │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         │ Tool Execution
                         │
┌────────────────────────▼────────────────────────────────────────┐
│                    TOOL SYSTEM                                  │
│  (ToolExecutor, ToolRegistry, Adapters)                       │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         │ Provider Calls
                         │
┌────────────────────────▼────────────────────────────────────────┐
│                   PROVIDER LAYER                                 │
│  (ProviderRegistry, Router, Individual Providers)             │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         │ Model Selection
                         │
┌────────────────────────▼────────────────────────────────────────┐
│                    MODEL SYSTEM                                  │
│  (ModelCatalog, ModelInfo, Model Assignment)                   │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         │ LLM API Calls
                         │
┌────────────────────────▼────────────────────────────────────────┐
│              EXTERNAL LLM PROVIDERS                              │
│  (OpenAI, Anthropic, Google, Ollama, etc.)                     │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         │ Responses
                         │
┌────────────────────────▼────────────────────────────────────────┐
│                    RESPONSES                                     │
│  (Completions, Streaming, Tool Results, etc.)                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. Detailed Connection Map by Subsystem

### 2.1 Provider Management Flow

```
Dashboard (Provider Page)
    │
    │ GET /api/providers
    │ POST /api/providers/{id}/config
    │ POST /api/providers/{id}/connect
    │ POST /api/providers/{id}/disconnect
    │ POST /api/providers/{id}/validate
    │ POST /api/providers/{id}/reload-models
    │
    ▼
REST Handler (handleProviders)
    │
    │ ProviderRegistry.List()
    │ ProviderRegistry.Get()
    │ Provider.Initialize()
    │ Provider.Close()
    │ Provider.Ping()
    │ Provider.ListModels()
    │
    ▼
ProviderRegistry (pkg/providers/register.go)
    │
    │ GlobalRegistry()
    │ GetProvider()
    │ RegisterProvider()
    │
    ▼
Individual Provider (pkg/providers/builtin/{provider}/)
    │
    │ Initialize()
    │ Complete() / StreamComplete()
    │ ListModels()
    │ GetModel()
    │ Ping()
    │ Status()
    │
    ▼
External LLM API (OpenAI, Anthropic, etc.)
```

**Status**: 
- Backend: ✓ Complete
- REST API: ✗ Missing (need to implement)
- Dashboard UI: ✗ Missing

---

### 2.2 Model Management Flow

```
Dashboard (Model Page)
    │
    │ GET /api/models
    │ GET /api/models/{provider}
    │ POST /api/models/assign
    │
    ▼
REST Handler (handleModels)
    │
    │ ProviderRegistry.List()
    │ Provider.ListModels()
    │ ModelCatalog.Get()
    │ ModelAssignment.Set()
    │
    ▼
ProviderRegistry
    │
    │ ListProviders()
    │ GetProvider()
    │
    ▼
Individual Provider
    │
    │ ListModels()
    │ GetModel()
    │
    ▼
ModelCatalog (pkg/providers/model_catalog.go)
    │
    │ GetAllModels()
    │ GetModelsByProvider()
    │ GetModelInfo()
    │
    ▼
ModelAssignment (NEW - need to implement)
    │
    │ SetManagerModel()
    │ SetPlannerModel()
    │ SetCoderModel()
    │ etc.
    │
    ▼
Runtime Configuration
```

**Status**: 
- Backend: PARTIAL (ModelCatalog exists, ModelAssignment missing)
- REST API: ✗ Missing (need to implement)
- Dashboard UI: ✗ Missing

---

### 2.3 Session Management Flow

```
Dashboard (Session Page)
    │
    │ GET /api/sessions
    │ POST /api/sessions
    │ GET /api/sessions/{id}
    │ PUT /api/sessions/{id}
    │ DELETE /api/sessions/{id}
    │ POST /api/sessions/{id}?action=pause
    │ POST /api/sessions/{id}?action=resume
    │ POST /api/sessions/{id}?action=complete
    │ POST /api/sessions/{id}?action=duplicate
    │ POST /api/sessions/{id}?action=archive
    │ POST /api/sessions/{id}?action=export
    │ POST /api/sessions/{id}?action=import
    │
    ▼
REST Handler (handleSessions, handleSessionByID)
    │
    │ SessionManager.CreateSession()
    │ SessionManager.GetSession()
    │ SessionManager.ListSessions()
    │ SessionManager.PauseSession()
    │ SessionManager.ResumeSession()
    │ SessionManager.CompleteSession()
    │ SessionManager.DuplicateSession() [MISSING]
    │ SessionManager.ArchiveSession() [MISSING]
    │ SessionManager.ExportSession() [MISSING]
    │ SessionManager.ImportSession() [MISSING]
    │
    ▼
SessionManager (pkg/session/core/session_manager.go)
    │
    │ CreateSession()
    │ GetSession()
    │ ListSessions()
    │ PauseSession()
    │ ResumeSession()
    │ CompleteSession()
    │
    ▼
SessionContainer (pkg/session/container.go)
    │
    │ Session lifecycle management
    │ Component initialization
    │ State management
    │
    ▼
Session Components
    ├── ChatManager
    ├── TaskManager
    ├── ProgressTracker
    ├── CollectiveMemory
    ├── SkillsManager
    └── Artifacts
```

**Status**: 
- Backend: PARTIAL (basic CRUD exists, advanced operations missing)
- REST API: PARTIAL (basic CRUD exists, advanced operations missing)
- Dashboard UI: PARTIAL (basic CRUD exists, advanced operations missing)

---

### 2.4 Agent Registration Flow

```
Dashboard (Agent Page)
    │
    │ GET /api/agents
    │ GET /api/agents/{id}
    │ POST /api/agents
    │ POST /api/agents/{id}/start
    │ POST /api/agents/{id}/stop
    │ POST /api/agents/{id}/restart
    │
    ▼
REST Handler (handleAgents, handleAgentByID)
    │
    │ AgentRegistry.Register()
    │ AgentRegistry.Get()
    │ AgentRegistry.List()
    │ AgentRegistry.Unregister()
    │ UnifiedAgent.Start()
    │ UnifiedAgent.Stop()
    │ UnifiedAgent.Restart()
    │
    ▼
AgentRegistry (pkg/agent/registry.go)
    │
    │ Register()
    │ Get()
    │ List()
    │ Unregister()
    │ UpdateStats()
    │
    ▼
UnifiedAgent (pkg/agent/unified/unified_agent.go)
    │
    │ Start()
    │ Stop()
    │ Restart()
    │ GetInfo()
    │
    ▼
Agent Components
    ├── UnifiedSkillManager
    ├── UnifiedMemoryManager
    ├── SubagentManager
    ├── AutomationManager
    ├── SkillDirector
    ├── MultiLayerValidator
    ├── Coordinator
    ├── FlowManager
    ├── ErrorHandler
    ├── CollectiveSystem
    ├── SessionEventBus
    ├── RealTimeMemorySync
    ├── RealTimeSkillSync
    ├── ProblemSolutionRegistry
    ├── LocalMemoryCache
    ├── DataCurator
    ├── TaskScheduler
    ├── AgentSyncManager
    ├── ProviderRegistry
    ├── Router
    ├── ToolExecutor
    ├── ThinkingEngine
    ├── WiringLayer
    ├── SessionContainer
    ├── SessionManager
    ├── AgentPool
    └── Metrics
```

**Status**: 
- Backend: ✓ Complete
- REST API: PARTIAL (basic list exists, detailed operations missing)
- Dashboard UI: PARTIAL (basic list exists, detailed operations missing)

---

### 2.5 Agent Orchestration Flow

```
Dashboard (Agent Orchestration View)
    │
    │ GET /api/agents/orchestration
    │ GET /api/agents/orchestration/{session_id}
    │ WebSocket /api/ws/agents
    │
    ▼
REST Handler (handleAgentOrchestration)
    │
    │ AgentPool.GetAll()
    │ AgentPool.GetBySession()
    │ AgentPool.GetState()
    │
    ▼
AgentPool (pkg/agent/unified/agent_pool.go)
    │
    │ GetAllAgents()
    │ GetAgentsBySession()
    │ GetAgentState()
    │ GetAgentTask()
    │ GetAgentProgress()
    │
    ▼
UnifiedAgent (per agent)
    │
    │ GetCurrentTask()
    │ GetCurrentStep()
    │ GetProvider()
    │ GetModel()
    │ GetLatency()
    │ GetContext()
    │ GetCurrentTool()
    │ GetMemoryUsage()
    │ GetTokens()
    │ GetExecutionTime()
    │
    ▼
Agent Components
    ├── SubagentManager (Manager, Planner, Architect, etc.)
    ├── ThinkingEngine (reasoning)
    ├── ToolExecutor (tool execution)
    ├── UnifiedMemoryManager (memory)
    └── ProviderRegistry (LLM calls)
```

**Status**: 
- Backend: ✓ Complete
- REST API: ✗ Missing (need to implement)
- WebSocket: ✗ Missing (need to implement)
- Dashboard UI: ✗ Missing

---

### 2.6 Chat Workspace Flow

```
Dashboard (Chat Workspace)
    │
    │ GET /api/messages/{session_id}
    │ POST /api/messages/{session_id}
    │ WebSocket /api/ws/chat/{session_id}
    │
    ▼
REST Handler (handleMessagesBySession)
    │
    │ ChatManager.GetMessages()
    │ ChatManager.SendMessage()
    │
    ▼
ChatManager (pkg/session/chat.go)
    │
    │ GetMessages()
    │ SendMessage()
    │ StreamMessage()
    │
    ▼
UnifiedAgent
    │
    │ ProcessMessage()
    │ StreamResponse()
    │
    ▼
ThinkingEngine
    │
    │ Think()
    │ Reason()
    │
    ▼
Provider (via Router)
    │
    │ Complete() / StreamComplete()
    │
    ▼
External LLM
```

**Status**: 
- Backend: ✓ Complete
- REST API: PARTIAL (basic endpoints exist, streaming missing)
- WebSocket: PARTIAL (basic WS exists, chat-specific missing)
- Dashboard UI: ✗ Missing (professional chat workspace missing)

---

### 2.7 Tool System Flow

```
Dashboard (Tools Page)
    │
    │ GET /api/tools
    │ GET /api/tools/{id}
    │ POST /api/tools/{id}/execute
    │ POST /api/tools/{id}/dry-run
    │ POST /api/tools/{id}/enable
    │ POST /api/tools/{id}/disable
    │
    ▼
REST Handler (handleTools)
    │
    │ ToolRegistry.List()
    │ ToolRegistry.Get()
    │ ToolExecutor.Execute()
    │ ToolExecutor.DryRun()
    │ ToolRegistry.Enable()
    │ ToolRegistry.Disable()
    │
    ▼
ToolRegistry (pkg/agent/tools/registry.go)
    │
    │ List()
    │ Get()
    │ Register()
    │ Enable()
    │ Disable()
    │
    ▼
ToolExecutor (pkg/agent/tools/executor.go)
    │
    │ Execute()
    │ DryRun()
    │ GetStats()
    │
    ▼
Tool Adapters
    ├── CLIAdapter
    ├── IDEAdapter
    ├── BrowserAdapter
    ├── DesktopAdapter
    └── CustomAdapter
```

**Status**: 
- Backend: ✓ Complete
- REST API: ✗ Missing (need to implement)
- Dashboard UI: ✗ Missing

---

### 2.8 Memory System Flow

```
Dashboard (Memory Page)
    │
    │ GET /api/memory/{session_id}
    │ POST /api/memory/{session_id}/search
    │ DELETE /api/memory/{session_id}/{key}
    │ POST /api/memory/{session_id}/export
    │ POST /api/memory/{session_id}/import
    │ POST /api/memory/{session_id}/rebuild-embeddings
    │
    ▼
REST Handler (handleMemoryBySession)
    │
    │ CollectiveMemory.GetAll()
    │ CollectiveMemory.Search()
    │ CollectiveMemory.Delete()
    │ CollectiveMemory.Export() [MISSING]
    │ CollectiveMemory.Import() [MISSING]
    │ CollectiveMemory.RebuildEmbeddings() [MISSING]
    │
    ▼
CollectiveMemory (pkg/session/memory.go)
    │
    │ GetAll()
    │ Search()
    │ Delete()
    │ Get()
    │ Set()
    │
    ▼
Memory Components
    ├── Working Memory
    ├── Short-Term Memory
    ├── Long-Term Memory
    ├── Session Memory
    ├── Shared Memory
    ├── Knowledge Store
    ├── Vector Database
    └── Embeddings
```

**Status**: 
- Backend: PARTIAL (basic operations exist, advanced operations missing)
- REST API: PARTIAL (basic endpoints exist, advanced operations missing)
- Dashboard UI: ✗ Missing

---

### 2.9 Event System Flow

```
Dashboard (Event Timeline)
    │
    │ WebSocket /api/ws/events
    │ GET /api/events (historical)
    │
    ▼
REST Handler (handleEvents)
    │
    │ EventBus.GetHistory()
    │
    ▼
WebSocket Handler (handleWebSocket)
    │
    │ EventBus.Subscribe()
    │ EventBus.StreamEvents()
    │
    ▼
EventBus (pkg/eventbus/bus.go)
    │
    │ Publish()
    │ Subscribe()
    │ GetHistory()
    │ Stream()
    │
    ▼
Event Sources
    ├── Session Events
    ├── Agent Events
    ├── Tool Events
    ├── Memory Events
    ├── Provider Events
    └── System Events
```

**Status**: 
- Backend: ✓ Complete
- REST API: ✗ Missing (historical events missing)
- WebSocket: PARTIAL (basic WS exists, event streaming missing)
- Dashboard UI: ✗ Missing

---

### 2.10 Logging System Flow

```
Dashboard (Log Viewer)
    │
    │ WebSocket /api/ws/logs
    │ GET /api/logs (historical)
    │
    ▼
REST Handler (handleLogs)
    │
    │ Logger.GetHistory()
    │ Logger.Filter()
    │
    ▼
WebSocket Handler (handleWebSocket)
    │
    │ Logger.Subscribe()
    │ Logger.StreamLogs()
    │
    ▼
Logger (pkg/logger/)
    │
    │ Log()
    │ GetHistory()
    │ Filter()
    │ Stream()
    │
    ▼
Log Sources
    ├── Application Logs
    ├── Agent Logs
    ├── Provider Logs
    ├── Tool Logs
    └── System Logs
```

**Status**: 
- Backend: PARTIAL (logging exists, streaming missing)
- REST API: ✗ Missing (log endpoints missing)
- WebSocket: ✗ Missing (log streaming missing)
- Dashboard UI: ✗ Missing

---

### 2.11 Metrics System Flow

```
Dashboard (Observability Page)
    │
    │ GET /api/metrics
    │ WebSocket /api/ws/metrics
    │
    ▼
REST Handler (handleMetrics)
    │
    │ Metrics.Get()
    │ Metrics.GetHistory()
    │
    ▼
WebSocket Handler (handleWebSocket)
    │
    │ Metrics.Subscribe()
    │ Metrics.Stream()
    │
    ▼
Metrics (pkg/metrics/metrics.go)
    │
    │ Record()
    │ Get()
    │ GetHistory()
    │ Stream()
    │
    ▼
Metric Sources
    ├── CPU
    ├── RAM
    ├── GPU
    ├── Latency
    ├── API Calls
    ├── Requests
    ├── Errors
    ├── Tokens
    ├── Streaming
    ├── Sessions
    ├── Workers
    ├── Queue
    └── WebSocket
```

**Status**: 
- Backend: PARTIAL (basic metrics exist, collection incomplete)
- REST API: ✗ Missing (metrics endpoints missing)
- WebSocket: ✗ Missing (metrics streaming missing)
- Dashboard UI: ✗ Missing

---

### 2.12 Configuration System Flow

```
Dashboard (Configuration Page)
    │
    │ GET /api/config
    │ PUT /api/config
    │ POST /api/config/validate
    │ POST /api/config/apply
    │ POST /api/config/rollback
    │
    ▼
REST Handler (handleConfig)
    │
    │ Config.Get()
    │ Config.Set()
    │ Config.Validate()
    │ Config.Apply()
    │ Config.Rollback()
    │
    ▼
Config (pkg/config/config.go)
    │
    │ Load()
    │ Save()
    │ Validate()
    │ Apply()
    │ Rollback()
    │
    ▼
Configuration File (config.yaml)
```

**Status**: 
- Backend: ✓ Complete (file-based)
- REST API: ✗ Missing (config endpoints missing)
- Dashboard UI: ✗ Missing

---

### 2.13 Storage System Flow

```
Dashboard (File Explorer)
    │
    │ GET /api/files
    │ GET /api/files/{path}
    │ POST /api/files/upload
    │ GET /api/files/{path}/download
    │ GET /api/files/{path}/preview
    │ GET /api/files/{path1}/compare/{path2}
    │
    ▼
REST Handler (handleFiles)
    │
    │ Storage.List()
    │ Storage.Get()
    │ Storage.Upload()
    │ Storage.Download()
    │ Storage.Preview()
    │ Storage.Compare()
    │
    ▼
Storage (pkg/storage/)
    │
    │ List()
    │ Get()
    │ Upload()
    │ Download()
    │ Preview()
    │ Compare()
    │
    ▼
File System
    ├── Workspace
    ├── Generated Files
    ├── Artifacts
    ├── Sessions
    ├── Logs
    └── Exports
```

**Status**: 
- Backend: PARTIAL (storage exists, file operations incomplete)
- REST API: ✗ Missing (file endpoints missing)
- Dashboard UI: ✗ Missing

---

### 2.14 Integration System Flow

```
Dashboard (Integrations Page)
    │
    │ GET /api/integrations
    │ GET /api/integrations/{type}
    │ POST /api/integrations/{type}/connect
    │ POST /api/integrations/{type}/disconnect
    │ GET /api/integrations/{type}/status
    │
    ▼
REST Handler (handleIntegrations)
    │
    │ AdapterManager.List()
    │ AdapterManager.Connect()
    │ AdapterManager.Disconnect()
    │ AdapterManager.GetStatus()
    │
    ▼
AdapterManager (pkg/agent/adapters/instance_manager.go)
    │
    │ List()
    │ Connect()
    │ Disconnect()
    │ GetStatus()
    │
    ▼
Adapters
    ├── IDEAdapter (VSCode, Cursor, Windsurf, Zed, JetBrains)
    ├── CLIAdapter (Claude Code, Gemini CLI, OpenAI CLI, etc.)
    ├── BrowserAdapter
    ├── DesktopAdapter
    └── CustomAdapter
```

**Status**: 
- Backend: ✓ Complete
- REST API: ✗ Missing (configuration endpoints missing)
- Dashboard UI: ✗ Missing

---

### 2.15 System Health Flow

```
Dashboard (System Health Page)
    │
    │ GET /api/health
    │ GET /api/health/backend
    │ GET /api/health/database
    │ GET /api/health/storage
    │ GET /api/health/memory
    │ GET /api/health/providers
    │ GET /api/health/agents
    │ GET /api/health/tools
    │ GET /api/health/api
    │ GET /api/health/websocket
    │ GET /api/health/synchronization
    │ GET /api/health/queue
    │ GET /api/health/scheduler
    │ GET /api/health/workers
    │
    ▼
REST Handler (handleHealth)
    │
    │ HealthChecker.CheckAll()
    │ HealthChecker.CheckBackend()
    │ HealthChecker.CheckDatabase()
    │ HealthChecker.CheckStorage()
    │ HealthChecker.CheckMemory()
    │ HealthChecker.CheckProviders()
    │ HealthChecker.CheckAgents()
    │ HealthChecker.CheckTools()
    │ HealthChecker.CheckAPI()
    │ HealthChecker.CheckWebSocket()
    │ HealthChecker.CheckSynchronization()
    │ HealthChecker.CheckQueue()
    │ HealthChecker.CheckScheduler()
    │ HealthChecker.CheckWorkers()
    │
    ▼
HealthChecker (NEW - need to implement)
    │
    │ CheckAll()
    │ CheckBackend()
    │ CheckDatabase()
    │ CheckStorage()
    │ CheckMemory()
    │ CheckProviders()
    │ CheckAgents()
    │ CheckTools()
    │ CheckAPI()
    │ CheckWebSocket()
    │ CheckSynchronization()
    │ CheckQueue()
    │ CheckScheduler()
    │ CheckWorkers()
    │
    ▼
Subsystems
    ├── Backend
    ├── Database (BadgerDB)
    ├── Storage
    ├── Memory
    ├── Providers
    ├── Agents
    ├── Tools
    ├── API
    ├── WebSocket
    ├── Synchronization
    ├── Queue
    ├── Scheduler
    └── Workers
```

**Status**: 
- Backend: PARTIAL (basic health endpoint exists, detailed checks missing)
- REST API: PARTIAL (basic health endpoint exists, detailed checks missing)
- Dashboard UI: ✗ Missing

---

### 2.16 API Explorer Flow

```
Dashboard (API Explorer)
    │
    │ POST /api/api-explorer/request
    │ GET /api/api-explorer/endpoints
    │
    ▼
REST Handler (handleAPIExplorer)
    │
    │ APIExplorer.ListEndpoints()
    │ APIExplorer.ExecuteRequest()
    │
    ▼
APIExplorer (NEW - need to implement)
    │
    │ ListEndpoints()
    │ ExecuteRequest()
    │ GetEndpointDocs()
    │ ValidateRequest()
    │
    ▼
REST API Registry
    │
    │ All registered endpoints
    │ Endpoint documentation
    │ Request/Response schemas
```

**Status**: 
- Backend: ✗ Missing (API Explorer not implemented)
- REST API: ✗ Missing (API Explorer endpoints missing)
- Dashboard UI: ✗ Missing

---

## 3. WebSocket Connection Map

### 3.1 WebSocket Endpoints Needed

```
Dashboard
    │
    ├── WebSocket /api/ws/chat/{session_id}
    │   │   Chat streaming
    │   │   Message streaming
    │   │   Tool call streaming
    │   │   Reasoning streaming
    │   │
    │   ▼
    │ ChatManager.StreamMessages()
    │ UnifiedAgent.StreamResponse()
    │
    ├── WebSocket /api/ws/agents
    │   │   Agent state updates
    │   │   Agent task updates
    │   │   Agent progress updates
    │   │
    │   ▼
    │ AgentPool.StreamAgentStates()
    │ UnifiedAgent.StreamState()
    │
    ├── WebSocket /api/ws/events
    │   │   Event streaming
    │   │   System events
    │   │   Agent events
    │   │   Tool events
    │   │
    │   ▼
    │ EventBus.StreamEvents()
    │
    ├── WebSocket /api/ws/logs
    │   │   Log streaming
    │   │   Application logs
    │   │   Agent logs
    │   │   Provider logs
    │   │
    │   ▼
    │ Logger.StreamLogs()
    │
    ├── WebSocket /api/ws/metrics
    │   │   Metrics streaming
    │   │   CPU, RAM, GPU
    │   │   Latency, API calls
    │   │   Tokens, errors
    │   │
    │   ▼
    │ Metrics.Stream()
    │
    └── WebSocket /api/ws/system
        │   System state updates
        │   Health updates
        │   Connection updates
        │
        ▼
    SystemMonitor.StreamState()
```

**Status**: 
- Backend: PARTIAL (basic WS exists, specific streams missing)
- Dashboard UI: ✗ Missing

---

## 4. Data Flow Diagrams

### 4.1 Session Creation Flow

```
User clicks "Create Session"
    │
    ▼
Dashboard POST /api/sessions
    │
    │ { name, owner_did, manager_agent_id, assistant_agents }
    │
    ▼
REST Handler handleSessions (POST)
    │
    ▼
SessionManager.CreateSession()
    │
    ▼
SessionContainer (new)
    │
    ├── Initialize ChatManager
    ├── Initialize TaskManager
    ├── Initialize ProgressTracker
    ├── Initialize CollectiveMemory
    ├── Initialize SkillsManager
    ├── Initialize Artifacts
    └── Initialize BridgeManager
    │
    ▼
Return Session ID
    │
    ▼
Dashboard displays new session
```

### 4.2 Agent Registration Flow

```
User clicks "Register Agent"
    │
    ▼
Dashboard POST /api/sessions/{id}?action=register_agent
    │
    │ { agent_id, instance_id, human_client_id, provider, model, role }
    │
    ▼
REST Handler handleSessionByID (POST, action=register_agent)
    │
    ▼
SessionManager.RegisterAgentInstance()
    │
    ▼
AgentRegistry.Register()
    │
    ▼
UnifiedAgent (new)
    │
    ├── Initialize UnifiedSkillManager
    ├── Initialize UnifiedMemoryManager
    ├── Initialize SubagentManager
    ├── Initialize AutomationManager
    ├── Initialize SkillDirector
    ├── Initialize MultiLayerValidator
    ├── Initialize Coordinator
    ├── Initialize FlowManager
    ├── Initialize ErrorHandler
    ├── Initialize CollectiveSystem
    ├── Initialize SessionEventBus
    ├── Initialize RealTimeMemorySync
    ├── Initialize RealTimeSkillSync
    ├── Initialize ProblemSolutionRegistry
    ├── Initialize LocalMemoryCache
    ├── Initialize DataCurator
    ├── Initialize TaskScheduler
    ├── Initialize AgentSyncManager
    ├── Initialize ProviderRegistry
    ├── Initialize Router
    ├── Initialize ToolExecutor
    ├── Initialize ThinkingEngine
    ├── Initialize WiringLayer
    ├── Initialize SessionContainer
    ├── Initialize SessionManager
    ├── Initialize AgentPool
    └── Initialize Metrics
    │
    ▼
Return success
    │
    ▼
Dashboard displays registered agent
```

### 4.3 Message Processing Flow

```
User sends message in Chat Workspace
    │
    ▼
Dashboard POST /api/messages/{session_id}
    │
    │ { content, role: "user" }
    │
    ▼
REST Handler handleMessagesBySession (POST)
    │
    ▼
ChatManager.SendMessage()
    │
    ▼
UnifiedAgent.ProcessMessage()
    │
    ▼
ThinkingEngine.Think()
    │
    ├── Analyze task
    ├── Plan approach
    ├── Select tools
    ├── Generate reasoning
    └── Create response
    │
    ▼
Router.Route() (if LLM needed)
    │
    ▼
Provider.Complete() / StreamComplete()
    │
    ▼
External LLM API
    │
    ▼
Response
    │
    ▼
ThinkingEngine.ProcessResponse()
    │
    ▼
ToolExecutor.Execute() (if tools needed)
    │
    ▼
Tool Adapters
    │
    ▼
Tool Results
    │
    ▼
UnifiedMemoryManager.UpdateMemory()
    │
    ▼
CollectiveMemory.Update()
    │
    ▼
ChatManager.AddMessage()
    │
    ▼
WebSocket /api/ws/chat/{session_id} (stream)
    │
    ▼
Dashboard displays streaming response
```

---

## 5. Critical Dependencies

### 5.1 Provider Management Dependencies
- **Required**: ProviderRegistry, Individual Providers, ModelCatalog
- **Missing**: REST API handlers, Dashboard UI
- **Priority**: HIGH

### 5.2 Model Assignment Dependencies
- **Required**: ProviderRegistry, ModelCatalog, NEW ModelAssignment service
- **Missing**: ModelAssignment service, REST API handlers, Dashboard UI
- **Priority**: HIGH

### 5.3 Agent Orchestration Dependencies
- **Required**: AgentPool, UnifiedAgent, SubagentManager
- **Missing**: REST API handlers, WebSocket handlers, Dashboard UI
- **Priority**: HIGH

### 5.4 Real-time Metrics Dependencies
- **Required**: Metrics service, NEW Metrics collection
- **Missing**: Complete Metrics collection, REST API handlers, WebSocket handlers, Dashboard UI
- **Priority**: HIGH

### 5.5 Log Streaming Dependencies
- **Required**: Logger service, NEW Log streaming
- **Missing**: Log streaming, REST API handlers, WebSocket handlers, Dashboard UI
- **Priority**: HIGH

### 5.6 Event Streaming Dependencies
- **Required**: EventBus, NEW Event streaming
- **Missing**: Event streaming, REST API handlers, WebSocket handlers, Dashboard UI
- **Priority**: HIGH

---

## 6. Integration Points

### 6.1 Dashboard ↔ REST API
- **Protocol**: HTTP/HTTPS
- **Format**: JSON
- **Authentication**: Token-based (mskt-*)
- **Rate Limiting**: security.RateLimiter

### 6.2 Dashboard ↔ WebSocket
- **Protocol**: WebSocket
- **Format**: JSON
- **Authentication**: Token-based (query param)
- **Streaming**: Real-time updates

### 6.3 REST API ↔ Services
- **Protocol**: Go function calls
- **Format**: Go structs
- **Error Handling**: Go error returns

### 6.4 Services ↔ Runtime
- **Protocol**: Go function calls
- **Format**: Go structs
- **Concurrency**: Goroutines, channels

### 6.5 Runtime ↔ Agents
- **Protocol**: Go function calls
- **Format**: Go structs
- **Communication**: Channels, EventBus

### 6.6 Agents ↔ Providers
- **Protocol**: Go function calls
- **Format**: Go structs
- **External**: HTTP to LLM APIs

---

## 7. Missing Components

### 7.1 REST API Handlers (Missing)
1. handleProviders
2. handleProviderByID
3. handleModels
4. handleModelAssignment
5. handleAgentOrchestration
6. handleTools
7. handleLogs
8. handleEvents
9. handleMetrics
10. handleConfig
11. handleFiles
12. handleIntegrations
13. handleHealthDetailed
14. handleAPIExplorer

### 7.2 WebSocket Handlers (Missing)
1. handleChatWebSocket
2. handleAgentWebSocket
3. handleEventWebSocket
4. handleLogWebSocket
5. handleMetricsWebSocket
6. handleSystemWebSocket

### 7.3 Services (Missing)
1. ModelAssignment
2. HealthChecker
3. APIExplorer
4. SystemMonitor
5. LogStreamer
6. EventStreamer
7. MetricsCollector

### 7.4 Dashboard UI (Missing)
1. Provider Management UI
2. Model Assignment UI
3. Agent Orchestration UI
4. Tool Registry UI
5. Memory Inspection UI
6. Log Viewer UI
7. Event Timeline UI
8. Metrics Dashboard UI
9. Configuration UI
10. File Explorer UI
11. Integration Management UI
12. System Health UI
13. API Explorer UI

---

## 8. Implementation Priority

### Phase 1: Critical APIs (Must Have)
1. Provider Management APIs
2. Model Assignment APIs
3. Agent Orchestration APIs
4. Real-time Metrics APIs
5. Log Streaming APIs
6. Event Streaming APIs

### Phase 2: Important APIs (Should Have)
1. Tool Registry APIs
2. Memory Inspection APIs
3. Configuration APIs
4. File Explorer APIs
5. Integration Management APIs
6. System Health APIs

### Phase 3: Nice to Have APIs (Could Have)
1. API Explorer APIs
2. Advanced Session Operations
3. Advanced Tool Operations
4. Advanced Memory Operations
5. Advanced Configuration

---

## Next Steps

Proceed to PHASE 4 - Gap Analysis to determine for every Dashboard feature what exists and what is missing in terms of Backend, API, Service, Handler, Event, WebSocket, Model, DTO, and Frontend.

# MUSKETEERS — Complete Architecture Map

## 1. HIGH-LEVEL PACKAGE MAP

```
cmd/studio/main.go          ← Entry point (Composition Root)
├── api/                    ← REST API + WebSocket + Dashboard
├── pkg/agent/             ← Agent definitions, registry, pool
│   ├── unified/           ← AgentPool, UnifiedAgent, coordinators
│   ├── thinking/          ← ThinkingEngine (LLM execution)
│   ├── adapters/          ← ProviderAdapter, CLIAdapter, etc.
│   ├── tools/             ← ToolRegistry, ToolExecutor
│   ├── autodiscovery/     ← CLI/IDE agent discovery
│   └── ...
├── pkg/orchestrator/     ← SessionManager, OrchestratorEngine
├── pkg/session/          ← SessionContainer, Memory, Workflow, Skills
├── pkg/providers/        ← ProviderRegistry, Router, ModelCatalog
├── pkg/runtime/          ← ApplicationRuntime, AgentRuntime
├── pkg/agent_bridge/     ← External agent bridge (TCP)
├── pkg/eventbus/         ← Pub/sub event bus
├── pkg/integration/      ← Glue: agents ↔ sessions, tasks, roles
├── pkg/workflow/         ← Sequential workflow engine
├── pkg/node/             ← P2P networking
├── pkg/identity/         ← DID-based identity
├── pkg/crypto/           ← Key generation, encryption
├── pkg/channel/          ← Encrypted agent channels
└── pkg/policy/           ← Security policy engine
```

## 2. DEPENDENCY GRAPH (Top-Down)

```
cmd/studio/main.go
  ├── pkg/runtime (ApplicationRuntime)
  │     ├── pkg/agent (AgentRegistry)
  │     ├── pkg/agent/unified (AgentPool, UnifiedAgent)
  │     ├── pkg/session (SessionContainer)
  │     ├── pkg/orchestrator (OrchestratorEngine)
  │     └── pkg/providers (ProviderRegistry)
  ├── pkg/orchestrator
  │     ├── pkg/agent (AgentRegistry)
  │     ├── pkg/agent/unified (AgentPool)
  │     ├── pkg/session (SessionContainer)
  │     ├── pkg/eventbus
  │     ├── pkg/agent_bridge
  │     ├── pkg/policy
  │     └── pkg/providers
  ├── pkg/session
  │     ├── pkg/agent (UnifiedAgent interface)
  │     ├── pkg/agent/tools
  │     ├── pkg/eventbus
  │     └── pkg/workflow (via internal structures)
  ├── pkg/agent/unified
  │     ├── pkg/agent (registry, interfaces)
  │     ├── pkg/agent/thinking
  │     ├── pkg/agent/tools
  │     └── pkg/providers
  ├── pkg/providers
  │     ├── pkg/providers/builtin/* (24 providers)
  │     └── pkg/lifecycle
  └── pkg/eventbus (stdlib only)
```

## 3. INITIALIZATION ORDER (main.go actual sequence)

```
1.  Parse flags (addr, data-dir, api-port, etc.)
2.  Create logrus logger
3.  Create root context
4.  Generate key pair (nrcrypto.GenerateKeyPair)
5.  Create identity record
6.  Create P2P node (node.New)
7.  Publish identity on DHT
8.  Create EventBus (eventbus.NewEventBus)
9.  Create QuotaManager
10. Open BadgerDB
11. Create ApplicationRuntime (pkgRuntime.NewApplicationRuntime)
12. Build() — creates AgentRegistry internally
13. Inject() — stub
14. Start()
15. Get AgentRegistry from ApplicationRuntime
16. Create ReservationManager + start cleanup
17. Create orchestrator.SessionManager
18. Wire SessionManager to AgentRegistry, EventBus, DB
19. Create SessionBridgeManager
20. Defer: appRuntime.Shutdown + Cancel
21. Create EmailManager + start
22. Create AutoDiscovery + LifecycleManager
23. Discover CLI/IDE agents
24. Create SessionConfig + SessionContainer
25. Start SessionContainer flush worker
26. Create UnifiedAgent (unified.NewUnifiedAgent)
27. Set Real SessionContainer on UnifiedAgent
28. Initialize UnifiedAgent
29. Create MultiplexedBridge
30. Create Connector
31. Create DelegationManager
32. Wire DelegationManager to AgentRegistry, EventBus
33. Create OrchestratorEngine
34. Wire logger, UnifiedAgent, SessionContainer, Connector,
    DelegationManager, AgentPool, SessionManager
35. Start OrchestratorEngine
36. Register discovered CLI agents in AgentRegistry
37. FOR each agent in AgentRegistry:
      → orchestratorEngine.RegisterAgent
      → unifiedAgent.RegisterAgent
      → unifiedAgent.RegisterAgentToPool
38. Create ProviderRegistry (builtin.NewRegistry)
39. Initialize providers (Cloudflare, Mistral, OpenRouter, Ollama)
40. Set ProviderRegistry on UnifiedAgent
41. Create ModelAgent IDs from providers
42. FOR each model agent:
      → agentRegistry.Register
      → orchestratorEngine.RegisterAgent
      → unifiedAgent.RegisterAgentToPool
43. Create Smart Router
44. Link Router to UnifiedAgent
45. Link default provider to AgentPool
46. [GOROUTINE] Execute test task
47. Create CEOSupervisor, A2AManager
48. Create integration layer (AgentSessionIntegration, etc.)
49. Initialize resource/memory/rate/connection limiters
50. Create verification components
51. Configure policy engine (AUDIT mode)
52. Create API Server + wire runtime
53. Create WebSocket Bridge
54. [GOROUTINE] Start API Server
55. ??? Create default session (currently commented/not executed)
56. [GOROUTINE] ThinkingEngine test
57. Wait for SIGINT/SIGTERM
58. Graceful shutdown
```

## 4. KEY OBSERVATIONS: DUPLICATE SYSTEMS

### 4a. Multiple Session Managers
| System | Location | Used By |
|--------|----------|---------|
| `orchestrator.SessionManager` | `pkg/orchestrator/session_manager.go` | main.go, Integration layer |
| `session.SessionContainer` | `pkg/session/container.go` | main.go, OrchestratorEngine |
| `core.UnifiedSessionManager` | `pkg/session/core/manager.go` | InstanceSessionIntegration |
| `advanced.AdvancedSessionManager` | `pkg/session/advanced/` | Not wired in main.go |

### 4b. Multiple Runtime Systems
| System | Location | Purpose |
|--------|----------|---------|
| `ApplicationRuntime` | `pkg/runtime/application_runtime.go` | App-level composition root |
| `AgentRuntime` (interface) | `pkg/runtime/runtime.go` | Per-agent runtime |
| Main.go direct init | `cmd/studio/main.go` | Actual composition root |

### 4c. Multiple Event Buses
| System | Location | Note |
|--------|----------|------|
| `eventbus.EventBus` | `pkg/eventbus/bus.go` | Main app bus (used everywhere) |
| `events.MemoryEventBus` | `pkg/runtime/events/bus.go` | Separate bus for AgentRuntime |
| Internal bus in OrchestratorEngine | `pkg/orchestrator/orchestrator_engine.go` | Created inside constructor |

### 4d. Multiple Agent Pools
| System | Location | Note |
|--------|----------|------|
| `unified.AgentPool` | `pkg/agent/unified/agent_pool.go` | Single source of truth |
| `agent.AgentRegistry` | `pkg/agent/registry.go` | Agent registry (different from pool) |
| ApplicationRuntime internal | `pkg/runtime/application_runtime.go` | Has field but unused |

### 4e. Multiple Lifecycle Systems
| System | Location | Used By |
|--------|----------|---------|
| `pkg/lifecycle.LifecycleMixin` | `pkg/lifecycle/` | ApplicationRuntime |
| `pkg/runtime/lifecycle.AgentLifecycle` | `pkg/runtime/lifecycle/` | AgentRuntimeImpl |
| `orchestrator.AgentLifecycleManager` | `pkg/orchestrator/agent_lifecycle.go` | OrchestratorEngine |

## 5. MISSING INTEGRATIONS

### 5a. SessionManagerAgentID never consumed
- `sessionManagerAgentID` is set at main.go (line ~823) but NEVER passed to `CreateSession`
- Session is created with hardcoded `"manager-default"` (or not created at all)
- Result: No session manager agent is ever linked to a session

### 5b. ApplicationRuntime is bypassed
- ApplicationRuntime creates its own AgentRegistry internally (Build())
- main.go gets this registry but NEVER uses it
- main.go creates a SECOND AgentRegistry through auto-discovery and model agents
- The ApplicationRuntime's internal registry remains empty

### 5c. ProviderRegistry injected AFTER Build()
- ApplicationRuntime.Build() validates ProviderRegistry is not nil
- BUT: providerRegistry was created BEFORE the fix
- The fix now injects properly but still sequential issue

### 5d. SessionContainer not linked to SessionManager sessions
- SessionContainer has UnifiedSessionState (cross-session agents)
- SessionManager has SessionInfo (per-session agents with ManagerAgentID)
- These two are NOT synchronized:
  - Agents entered via RegisterAgentFromUnified go into SessionContainer
  - But SessionManager.CreateSession doesn't read from SessionContainer

### 5e. AgentPool not linked to session membership
- Agents registered in AgentPool have no session_id field
- Session knows agent IDs (ManagerAgentID, AssistantAgents)
- But AgentPool doesn't know which session an agent belongs to

### 5f. Multiple provider registries
- `main.go` creates `providerRegistry` at step 38
- `ApplicationRuntime` may have another ProviderRegistry
- `unified_agent.go` has its own `NewProviderRegistry()` call at line 169
- Result: multiple registries with inconsistent state

### 5g. core.UnifiedSessionManager never wired
- Exists in `pkg/session/core/manager.go`
- Has RegisterAgentInstance with Provider/Model fields
- But is never instantiated or wired in main.go

## 6. EXISTING SYSTEMS THAT ARE DISCONNECTED

| System | Exists | Wired? | Impact |
|--------|--------|--------|--------|
| ThinkingEngine | ✅ | ✅ (per-agent in AgentPool) | Works for any pool agent |
| CollectiveMemory | ✅ | ✅ (via SessionContainer) | Works |
| WorkflowEngine | ✅ | ✅ (via SessionContainer) | Works |
| SkillsManager | ✅ | ✅ (via SessionContainer) | Works |
| ProgressTracker | ✅ | ✅ (via SessionContainer) | Works |
| AgentCapabilityVerifier | ✅ | ✅ (called by RegisterAgentFromUnified) | Works |
| SessionJournal | ✅ | ✅ (via SessionContainer) | Works |
| ChatManager | ✅ | ✅ (via SessionContainer) | Works |
| Core UnifiedSessionManager | ✅ | ❌ Never instantiated | Dead code |
| AdvancedSessionManager | ✅ | ❌ Never instantiated | Dead code |
| IdentitySystem | ✅ | ✅ | Works |
| P2P Node | ✅ | ✅ | Works |
| EventBus | ✅ | ✅ | Works |
| MultiplexedBridge | ✅ | ✅ | Works |
| A2AManager | ✅ | ✅ | Works |
| DelegationManager | ✅ | ✅ | Works |
| OrchestratorEngine | ✅ | ✅ (but disconnected from sessions) | Partial |
| AgentPool | ✅ | ✅ (has all agents) | Works |
| AgentRegistry | ✅ | ✅ | Works |
| ProviderRegistry | ✅ | ✅ (now working) | Fixed |
| ResourceLimiter | ✅ | ❌ Not used after creation | Dead |
| MemoryLimiter | ✅ | ❌ Not used after creation | Dead |
| RateLimiter | ✅ | ❌ Not used after creation | Dead |
| ConnLimiter | ✅ | ❌ Not used after creation | Dead |
| WASM Sandbox | ✅ | ❌ Not used | Dead |
| IndexedDiscovery | ✅ | ❌ Not used | Dead |
| HostingManager | ✅ | ❌ Not used | Dead |
| AnalyticsIntegrator | ✅ | ✅ (starts, subscribes) | Works |
| BackupIntegrator | ✅ | ✅ (starts) | Works |
| NotificationsIntegrator | ✅ | ✅ (starts) | Works |
| PluginsIntegrator | ✅ | ✅ (starts) | Works |
| UpgradeIntegrator | ✅ | ✅ (starts) | Works |
| Verification components | ✅ | ❌ (verifier created but not used) | Dead |
| Policy Engine | ✅ | ✅ (AUDIT mode) | Works (passive) |

## 7. RECOMMENDED INTEGRATION ORDER

### Phase 1: Session-Centric Runtime (Critical)
1. Fix `sessionManagerAgentID` → pass to `CreateSession`
2. Add model agents as `AssistantAgents` in session
3. Link `SessionContainer.UnifiedSessionState` with `SessionManager.SessionInfo`

### Phase 2: Complete the Model→Agent→Session Chain
4. Show Session ID, Manager Agent ID in `/api/runtime/status`
5. Show Provider+Model in `/api/runtime/agents`
6. Show ThinkingEngine per agent in `/api/runtime/agents`

### Phase 3: Remove Dead Systems
7. Remove or wire: limiters, sandbox, discovery, hosting, verifiers
8. Remove `core.UnifiedSessionManager` or wire it properly

### Phase 4: Remove Duplicate Systems
9. Consolidate: only ONE ProviderRegistry source
10. Consolidate: only ONE EventBus
11. Consolidate: only ONE lifecycle system

### Phase 5: External Agents (Bridge)
12. Wire bridge for external agents
13. Separate internal/external agent routing

## 8. SESSION LIFECYCLE (Intended vs Current)

```
INTENDED:
  User → CreateSession → SelectManagerAgent → BindRole
  → CreateInternalAgents → AttachExternalAgents
  → CreateSharedMemory → CreateWorkflow
  → StartRuntime → StartEventBus → Ready

CURRENT:
  main.go → init ALL components linearly
  → AgentRegistry populated (CLI + model agents)
  → AgentPool populated
  → SessionManager created
  → ??? Session may or may not be created
  → If created, manager is "manager-default" (not real agent)
  → Agents exist but not linked to any session
```

## 9. COMMUNICATION GRAPH

```
Request → REST API (/api/*)
  → api.ServerRuntime
    → OrchestratorEngine.ExecuteTask
      → SessionManager.GetSession (finds session)
        → AgentPool.GetAgent(managerAgentID)
          → ThinkingEngine.AnalyzeTask
            → ThinkingEngine.PlanTask
              → ThinkingEngine.ExecuteSteps
                → ToolExecutor.Execute
                  → EventBus events
                    → Memory record
                    → Skills update
                    → Progress update
                    → Journal append
                    → WebSocket broadcast
```

## 10. OWNERSHIP GRAPH

```
SessionContainer (session owner)
  ├── Memory (CollectiveMemory) — events, facts, workflows, strategies
  ├── Skills (SkillsManager) — per-agent skill trees
  ├── Workflow (WorkflowEngine) — 16-step workflow
  ├── Tasks (TaskManager) — priority queue per agent
  ├── Progress (ProgressTracker) — delays, risks, metrics
  ├── Chat (ChatManager) — message history
  ├── Journal (SessionJournal) — append-only event log
  ├── Handoff (HandoffManager) — inter-agent artifact handoff
  ├── Artifacts (ArtifactsStore) — stored artifacts
  └── CapabilityVerifier — probes agent capabilities

AgentPool (agent runtime owner)
  ├── AgentInstance (per agent)
  │   ├── ProviderAdapter — LLM provider connection
  │   ├── ThinkingEngine — Analyze/Plan/Execute/Verify
  │   └── ToolExecutor — Tool execution with registry
  └── DefaultProvider, DefaultModelID

SessionManager (session metadata owner)
  ├── SessionInfo (per session)
  │   ├── ManagerAgentID
  │   ├── AssistantAgents
  │   ├── RoleAssignments
  │   └── AgentInstances
  └── Persisted via BadgerDB
```

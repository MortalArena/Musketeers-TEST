# Phase 1: Startup Graph

## Complete Startup Sequence

```
main()
├── 1. Parse Command-Line Flags
│   ├── addr (default: 127.0.0.1:5000)
│   ├── data-dir (default: ./studio-data)
│   ├── bootstrap (default: "")
│   ├── founder-pub (default: "")
│   ├── verbose (default: false)
│   ├── tls-cert (default: "")
│   ├── tls-key (default: "")
│   └── api-port (default: 8081)
├── 2. Initialize Logger
│   ├── logrus.New()
│   ├── SetLevel (Debug/Info)
│   └── SetFormatter (JSON)
├── 3. Create Context
│   └── context.WithCancel()
├── 4. Generate Key Pair
│   └── nrcrypto.GenerateKeyPair()
├── 5. Create Identity Record
│   └── identity.NewIdentityRecord()
├── 6. Create Node Configuration
│   ├── node.DefaultConfig()
│   ├── Set DataDir
│   ├── Set StorageQuotaMB (2GB)
│   ├── Set FounderPubHex
│   ├── Set BootstrapPeers
│   └── Set MaxPutPerMinute (300)
├── 7. Create P2P Node
│   └── node.New()
├── 8. Publish Identity
│   └── n.PublishIdentity()
├── 9. Create QuotaManager
│   └── storage.NewQuotaManager()
├── 10. Create EventBus
│   └── pkgEventbus.NewEventBus()
├── 11. Create BadgerDB
│   ├── Get Process ID
│   ├── Create Unique DB Path (badger-pid-{pid})
│   ├── Retry Loop (3 attempts)
│   └── badger.Open()
├── 12. Create AgentRegistry
│   ├── pkgAgent.NewAgentRegistry()
│   └── SetLogger()
├── 13. Create ReservationManager
│   ├── pkgAgent.NewReservationManager()
│   └── StartCleanupScheduler()
├── 14. Create UnifiedSessionManager
│   └── core.NewUnifiedSessionManager()
├── 15. Create SessionBridgeManager
│   └── pkgSession.NewSessionBridgeManager()
├── 16. Create Example Sessions
│   ├── Session 1: Project A
│   ├── Session 2: Project B
│   └── Session 3: Project C
├── 17. Create Example Bridges
│   ├── Bridge 1-2
│   └── Bridge 2-3
├── 18. Create EmailManager
│   ├── orchestrator.NewEmailManager()
│   └── Start()
├── 19. Create EmailIntegrator
│   ├── pkgEmail.NewEmailIntegrator()
│   └── Wire EventBus Subscribers
├── 20. Register Default Agents
│   ├── CLI Adapter (claude)
│   ├── IDE Adapter (cursor)
│   ├── Browser Adapter (Computer Use)
│   └── Custom Adapter
├── 21. Create SessionContainer
│   ├── pkgSession.NewSessionContainer()
│   └── StartFlushWorker()
├── 22. Create UnifiedAgent
│   ├── unified.NewUnifiedAgent()
│   ├── SetRealSessionContainer()
│   └── Initialize()
├── 23. Create StorageConnector
│   └── orchestrator.NewStorageConnector()
├── 24. Create MultiplexedBridge
│   └── agent_bridge.NewMultiplexedBridge()
├── 25. Create Connector
│   ├── orchestrator.NewConnector()
│   └── Start()
├── 26. Create OrchestratorEngine
│   ├── orchestrator.NewOrchestratorEngine()
│   ├── SetLogger()
│   ├── SetUnifiedAgent()
│   ├── SetConnector()
│   └── Start()
├── 27. Register Agents in Unified System
│   ├── Register in UnifiedAgent
│   └── Register in AgentPool
├── 28. Create ProviderRegistry
│   └── builtin.NewRegistry()
├── 29. Initialize Providers
│   ├── Mistral AI
│   │   ├── Get API Key (env or fallback)
│   │   ├── Initialize()
│   │   └── Ping()
│   ├── OpenRouter
│   │   ├── Get API Key (env or fallback)
│   │   ├── Initialize()
│   │   └── Ping()
│   └── Qwen
│       ├── Get API Key (env or fallback)
│       ├── Initialize()
│       └── Ping()
├── 30. Link ProviderRegistry to UnifiedAgent
│   ├── SetProviderRegistry()
│   └── SetThinkingEngineProvider()
├── 31. Create SmartRouter
│   ├── providers.NewRouter()
│   └── SetRouter()
├── 32. Execute Test Task
│   ├── Create Test Task
│   ├── orchestratorEngine.ExecuteTask()
│   └── Log Result
├── 33. Create CEOSupervisor
│   ├── pkgCEO.NewCEOSupervisor()
│   └── Start()
├── 34. Initialize Isolated Packages
│   ├── Logger
│   ├── Config
│   ├── Limits
│   ├── Timeout
│   ├── Validation
│   ├── Ledger
│   ├── Sandbox
│   ├── Discovery
│   ├── Hosting
│   ├── Analytics
│   ├── Backup
│   ├── Delegation
│   ├── Notifications
│   ├── Plugins
│   └── Upgrade
├── 35. Initialize P2P Systems
│   ├── P2P Email Service
│   ├── P2P DNS Resolver
│   ├── Local DNS Proxy
│   ├── HTTP Proxy
│   ├── System Proxy
│   └── P2P Hosting Service
├── 36. Create Verification Components
│   ├── pkgVerification.NewMultiStageVerifier()
│   └── Register Verifiers
├── 37. Create ACP Handler
│   └── acp.NewRouter()
├── 38. Configure Policy Engine
│   ├── SetPolicyMode(Audit)
│   ├── Add Default Deny Rule
│   └── Add Allow Rules
├── 39. Create REST API Server
│   ├── api.NewServerWithTLS()
│   ├── Create API Key Manager
│   ├── UseRuntime()
│   │   ├── EventBus
│   │   ├── SessionManager
│   │   ├── BridgeManager
│   │   ├── ProviderRegistry
│   │   ├── APIKeyManager
│   │   └── OwnerDID
│   └── Generate API Token
├── 40. Create WebSocket Bridge
│   ├── api.NewWebSocketHandler()
│   └── Start()
├── 41. Start REST API Server
│   └── apiServer.Start() (goroutine)
└── 42. Wait for Shutdown Signal
    ├── signal.Notify()
    └── <-sigCh
```

## Startup Dependencies

### Critical Path (Must Complete Before Next Step)
```
Key Pair → Identity → Node → EventBus → DB → AgentRegistry → UnifiedAgent → Orchestrator → Providers → API Server
```

### Parallel Initialization (Can Run Concurrently)
```
EmailManager + EmailIntegrator
SessionContainer + UnifiedAgent
Provider Initialization (Mistral, OpenRouter, Qwen)
Isolated Packages (Analytics, Backup, Delegation, etc.)
P2P Systems (Email, DNS, HTTP, Hosting)
```

### Lazy Initialization (Can Be Deferred)
```
AgentPool (activates on first use)
ThinkingEngine (initializes on first task)
ToolExecutor (initializes on first tool call)
WebSocket Connections (on client connect)
```

## Startup Failures

### Fatal Failures (Stop Startup)
```
✗ Key Pair Generation
✗ Identity Record Creation
✗ Node Creation
✗ BadgerDB Open (after retries)
✗ SessionContainer Creation
✗ UnifiedAgent Initialization
✗ Connector Start
✗ CEOSupervisor Start
```

### Non-Fatal Failures (Continue with Warning)
```
⚠ Identity Publish
⚠ Session Creation (1, 2, 3)
⚠ Bridge Creation
⚠ EmailManager Start
⚠ Provider Initialization
⚠ Provider Ping
⚠ OrchestratorEngine Start
⚠ Isolated Package Initialization
⚠ P2P System Initialization
⚠ DNS Proxy Start
⚠ HTTP Proxy Start
```

## Startup Time Analysis

### Estimated Startup Times
```
Key Pair Generation: ~100ms
Identity Creation: ~50ms
Node Creation: ~500ms
EventBus Creation: ~10ms
BadgerDB Open: ~200ms
AgentRegistry Creation: ~10ms
SessionManager Creation: ~50ms
SessionContainer Creation: ~100ms
UnifiedAgent Creation: ~200ms
Provider Initialization: ~500ms per provider
OrchestratorEngine Start: ~100ms
CEOSupervisor Start: ~50ms
API Server Start: ~100ms
WebSocket Bridge Start: ~50ms
Isolated Packages: ~200ms total
P2P Systems: ~300ms total

Total Estimated: ~2.5-3 seconds
```

## Startup Health Checks

### Automatic Health Checks
```
✓ Node Connectivity (DHT)
✓ Provider Connectivity (Ping)
✓ Database Health (BadgerDB)
✓ Agent Health (CEOSupervisor)
✓ API Server Health (Port 8081)
✓ WebSocket Health (Port 8081)
```

### Manual Health Checks
```
→ Dashboard Access (http://localhost:8081/dashboard)
→ API Endpoints (/api/models, /api/sessions, /api/agents)
→ Agent Registration (AgentRegistry.ListAll())
→ Provider Availability (ProviderRegistry.List())
→ Session Status (SessionManager.ListSessions())
```

## Startup Configuration

### Environment Variables
```
MISTRAL_API_KEY (fallback: hardcoded)
OPENROUTER_API_KEY (fallback: hardcoded)
QWEN_API_KEY (fallback: hardcoded)
SMTP_HOST (default: smtp.gmail.com)
SMTP_USERNAME (required for email)
SMTP_PASSWORD (required for email)
```

### Configuration Files
```
config.yaml (if exists)
config.example.yaml (template)
```

### Command-Line Flags
```
-addr (default: 127.0.0.1:5000)
-data-dir(default: ./studio-data)
-bootstrap (default: "")
-founder-pub (default: "")
-verbose (default: false)
-tls-cert (default: "")
-tls-key (default: "")
-api-port (default: 8081)
```

## Startup Logging

### Critical Logs
```
✓ "Studio node created"
✓ "Event Bus created"
✓ "BadgerDB created"
✓ "Agent Registry created"
✓ "UnifiedAgent created"
✓ "UnifiedAgent initialized successfully"
✓ "Provider registry created"
✓ "Mistral provider initialized successfully"
✓ "OpenRouter provider initialized successfully"
✓ "Qwen provider initialized successfully"
✓ "CEO Supervisor started"
✓ "API Server started"
✓ "WebSocket Bridge created and started"
```

### Warning Logs (Expected)
```
⚠ "Failed to publish identity"
⚠ "Failed to create session X"
⚠ "Failed to create bridge X-Y"
⚠ "Failed to start EmailManager"
⚠ "Mistral provider ping failed"
⚠ "OpenRouter provider ping failed"
⚠ "Qwen provider ping failed"
⚠ "Failed to start orchestrator engine"
⚠ "Failed to start isolated package X"
⚠ "Failed to start P2P system X"
```

### Error Logs (Unexpected)
```
✗ "Failed to generate key pair"
✗ "Failed to create identity record"
✗ "Failed to create node"
✗ "Failed to open BadgerDB after retries"
✗ "Failed to create session container"
✗ "Failed to initialize unified agent"
✗ "Failed to start Connector"
✗ "Failed to start CEO supervisor"
✗ "API server failed to start"
```

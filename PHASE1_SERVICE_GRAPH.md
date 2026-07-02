# Phase 1: Service Graph

## Service Architecture

```
Service Layer
├── Core Services
│   ├── Node Service (P2P Network)
│   │   ├── DHT Service
│   │   ├── PubSub Service
│   │   ├── Discovery Service
│   │   └── Direct Connection Service
│   ├── EventBus Service (Event System)
│   │   ├── Event Publisher
│   │   ├── Event Subscriber
│   │   ├── Event Processor
│   │   └── Dead Letter Queue
│   ├── Database Service (Persistence)
│   │   ├── BadgerDB Service
│   │   ├── Quota Manager Service
│   │   └── Storage Connector Service
│   ├── Agent Service (Agent Management)
│   │   ├── Agent Registry Service
│   │   ├── Reservation Manager Service
│   │   ├── Agent Pool Service
│   │   └── Agent Lifecycle Service
│   ├── Session Service (Session Management)
│   │   ├── Session Manager Service
│   │   ├── Session Container Service
│   │   ├── Session Bridge Service
│   │   └── Session Bridge Manager Service
│   ├── Orchestrator Service (Task Orchestration)
│   │   ├── Orchestrator Engine Service
│   │   ├── Connector Service
│   │   ├── Role Assigner Service
│   │   ├── Capability Matcher Service
│   │   └── Delegation Manager Service
│   ├── Provider Service (LLM Management)
│   │   ├── Provider Registry Service
│   │   ├── Smart Router Service
│   │   ├── Free Router Service
│   │   ├── API Key Manager Service
│   │   └── Model Catalog Service
│   ├── UnifiedAgent Service (Agent Coordination)
│   │   ├── UnifiedAgent Service
│   │   ├── Session Manager Service
│   │   ├── Task Scheduler Service
│   │   ├── Flow Manager Service
│   │   ├── Coordinator Service
│   │   └── Error Handler Service
│   ├── CEO Service (Health Monitoring)
│   │   ├── CEO Supervisor Service
│   │   ├── Health Check Service
│   │   └── Alert Service
│   ├── Verification Service (Code Verification)
│   │   ├── Multi-Stage Verifier Service
│   │   ├── Syntax Verifier Service
│   │   ├── Semantics Verifier Service
│   │   ├── Security Verifier Service
│   │   ├── Performance Verifier Service
│   │   └── Integration Verifier Service
│   └── Policy Service (Access Control)
│       ├── Policy Engine Service
│       ├── ACP Router Service
│       ├── Approvals Service
│       └── Capability Manager Service
├── API Services
│   ├── REST API Service (HTTP API)
│   │   ├── Models Endpoint
│   │   ├── Sessions Endpoint
│   │   ├── Agents Endpoint
│   │   ├── Tasks Endpoint
│   │   ├── Artifacts Endpoint
│   │   ├── MCP Servers Endpoint
│   │   ├── MCP Tools Endpoint
│   │   ├── Provider Config Endpoint
│   │   └── Channels Endpoint
│   ├── WebSocket Service (Real-time API)
│   │   ├── WebSocket Handler
│   │   ├── WebSocket Bridge
│   │   ├── Event Subscription
│   │   └── Message Broadcasting
│   └── Dashboard Service (Web UI)
│       ├── Dashboard HTML
│       ├── Dashboard JavaScript
│       ├── Dashboard CSS
│       └── Dashboard Assets
├── Agent Services
│   ├── CLI Adapter Service (Command Line)
│   ├── IDE Adapter Service (Development Environment)
│   ├── Browser Adapter Service (Web Automation)
│   ├── Custom Adapter Service (Custom Tasks)
│   ├── Thinking Engine Service (AI Reasoning)
│   ├── Tool Executor Service (Tool Execution)
│   ├── Tool Registry Service (Tool Management)
│   ├── Subagent Manager Service (Subagent Coordination)
│   ├── Automation Manager Service (Task Automation)
│   ├── Skill Director Service (Skill Direction)
│   ├── Multi-Layer Validator Service (Validation)
│   ├── Skill Manager Service (Skill Management)
│   ├── Collective Memory Service (Shared Memory)
│   ├── Workflow Service (Agent Collaboration)
│   └── Learning Engine Service (Agent Learning)
├── Integration Services
│   ├── Agent Communication Service
│   ├── Agent Session Integration Service
│   ├── Instance Session Integration Service
│   ├── Session Orchestrator Service
│   ├── Task Routing Service
│   ├── Role Assignment Service
│   └── Webhook Router Service
├── P2P Services
│   ├── Email Service (P2P Email)
│   │   ├── Email Store Service
│   │   ├── P2P Email Service
│   │   └── Email Integrator Service
│   ├── DNS Service (P2P DNS)
│   │   ├── P2P DNS Resolver Service
│   │   ├── Local DNS Proxy Service
│   │   └── System Proxy Service
│   ├── HTTP Service (P2P HTTP)
│   │   ├── HTTP Proxy Service
│   │   └── System Proxy Service
│   └── Hosting Service (P2P Hosting)
│       ├── Hosting Manager Service
│       ├── P2P Hosting Service
│       └── Site Uploader Service
├── Isolated Services
│   ├── Analytics Service
│   │   ├── Analytics Integrator Service
│   │   └── Analytics Core Service
│   ├── Backup Service
│   │   ├── Backup Integrator Service
│   │   └── Backup Core Service
│   ├── Delegation Service
│   │   ├── Delegation Integrator Service
│   │   └── Advanced Delegation Service
│   ├── Notifications Service
│   │   ├── Notifications Integrator Service
│   │   └── Notifications Core Service
│   ├── Plugins Service
│   │   ├── Plugins Integrator Service
│   │   └── Plugins Core Service
│   └── Upgrade Service
│       ├── Upgrade Integrator Service
│       └── Upgrade Core Service
├── Support Services
│   ├── Config Service (Configuration)
│   ├── Logger Service (Logging)
│   ├── Limits Service (Resource Limits)
│   ├── Timeout Service (Timeout Management)
│   ├── Validation Service (Data Validation)
│   ├── Ledger Service (Cost Tracking)
│   ├── Sandbox Service (WASM Sandbox)
│   ├── Storage Service (Storage Management)
│   ├── Memory Service (Memory Management)
│   ├── Skills Service (Skill Management)
│   ├── Workflow Service (Workflow Management)
│   ├── Security Service (Security)
│   ├── Runtime Service (Runtime Management)
│   ├── Crypto Service (Cryptography)
│   ├── Identity Service (Identity Management)
│   ├── Naming Service (Naming)
│   ├── Protocol Service (Protocol)
│   ├── Metrics Service (Metrics)
│   ├── Cache Service (Caching)
│   ├── Channel Service (Channel)
│   ├── Common Service (Common)
│   ├── Mailbox Service (Mailbox)
│   ├── Rate Service (Rate Limiting)
│   ├── Recovery Service (Recovery)
│   ├── Registry Service (Registry)
│   ├── Search Service (Search)
│   └── Vault Service (Vault)
└── External Services
    ├── Mistral AI Service
    ├── OpenRouter Service
    ├── Qwen Service
    ├── Other 20 LLM Provider Services
    ├── SMTP Service (Email)
    └── External Tool Services
```

## Service Dependencies

### Service Dependency Graph
```
Node Service
├── Depends on: Crypto Service, Identity Service
└── Used by: All P2P Services

EventBus Service
├── Depends on: None
└── Used by: All Services

Database Service
├── Depends on: Storage Service, Limits Service
└── Used by: Session Service, Agent Service, Provider Service

Agent Service
├── Depends on: EventBus Service, Database Service
└── Used by: Orchestrator Service, UnifiedAgent Service, CEO Service

Session Service
├── Depends on: EventBus Service, Database Service, Agent Service
└── Used by: UnifiedAgent Service, Orchestrator Service, API Service

Orchestrator Service
├── Depends on: Agent Service, EventBus Service, Policy Service
└── Used by: API Service, UnifiedAgent Service

Provider Service
├── Depends on: EventBus Service, Database Service
└── Used by: UnifiedAgent Service, API Service, Smart Router Service

UnifiedAgent Service
├── Depends on: Provider Service, Session Service, EventBus Service, Agent Service
└── Used by: Orchestrator Service, API Service

CEO Service
├── Depends on: EventBus Service, Agent Service
└── Used by: None (monitoring only)

API Service
├── Depends on: Session Service, Provider Service, Agent Service, EventBus Service
└── Used by: External Clients

WebSocket Service
├── Depends on: EventBus Service, Session Service
└── Used by: Dashboard Service

Dashboard Service
├── Depends on: API Service, WebSocket Service
└── Used by: External Clients
```

## Service Communication

### Service Communication Patterns
```
Synchronous Communication
├── API Service → Orchestrator Service (Task Execution)
├── API Service → Provider Service (Model Listing)
├── API Service → Session Service (Session Management)
├── Orchestrator Service → Agent Service (Agent Execution)
├── UnifiedAgent Service → Provider Service (LLM Requests)
└── Agent Service → Tool Service (Tool Execution)

Asynchronous Communication
├── All Services → EventBus Service (Event Publishing)
├── EventBus Service → All Services (Event Subscription)
├── WebSocket Service → EventBus Service (Event Subscription)
├── CEO Service → EventBus Service (Health Events)
└── Integration Services → EventBus Service (Integration Events)

Streaming Communication
├── WebSocket Service → Dashboard Service (Real-time Updates)
├── Provider Service → UnifiedAgent Service (LLM Streaming)
└── Tool Service → Agent Service (Tool Output Streaming)
```

## Service Lifecycle

### Service States
```
Service States
├── Created (Service instantiated)
├── Initialized (Service initialized)
├── Started (Service running)
├── Paused (Service paused)
├── Stopped (Service stopped)
└── Failed (Service failed)
```

### Service Startup Order
```
Critical Path
1. EventBus Service
2. Database Service
3. Agent Service
4. Session Service
5. Provider Service
6. UnifiedAgent Service
7. Orchestrator Service
8. CEO Service
9. API Service
10. WebSocket Service

Parallel Startup
- Integration Services (Analytics, Backup, Delegation, etc.)
- P2P Services (Email, DNS, HTTP, Hosting)
- Support Services (Config, Logger, Limits, etc.)

Lazy Startup
- Agent Pool Service (on first agent use)
- Thinking Engine Service (on first task)
- Tool Executor Service (on first tool call)
```

## Service Health

### Health Check Endpoints
```
Service Health Checks
├── Node Service Health (DHT, PubSub, Discovery)
├── EventBus Service Health (Queue Size, Handler Count)
├── Database Service Health (Connection, Disk Usage)
├── Agent Service Health (Agent Count, Agent Health)
├── Session Service Health (Session Count, Session Health)
├── Orchestrator Service Health (Task Count, Task Health)
├── Provider Service Health (Provider Count, Provider Health)
├── UnifiedAgent Service Health (Agent Pool Health, Memory Health)
├── CEO Service Health (System Health, Agent Health)
├── API Service Health (Request Rate, Response Time)
└── WebSocket Service Health (Connection Count, Message Rate)
```

### Health Metrics
```
Health Metrics
├── Availability (Up/Down)
├── Response Time (ms)
├── Error Rate (%)
├── Throughput (requests/sec)
├── Resource Usage (CPU, Memory)
├── Queue Size (events, tasks)
└── Connection Count (active connections)
```

## Service Scaling

### Horizontal Scaling
```
Scalable Services
├── Session Service (Multiple Sessions)
├── Agent Service (Multiple Agents)
├── Provider Service (Multiple Providers)
├── API Service (Multiple API Servers)
├── WebSocket Service (Multiple Connections)
└── Integration Services (Multiple Integrators)
```

### Vertical Scaling
```
Resource Scaling
├── Memory Scaling (Agent Pool Parking, Session Persistence)
├── CPU Scaling (Goroutine Pooling, Async Operations)
├── I/O Scaling (Connection Pooling, Buffered Channels)
└── Network Scaling (Connection Pooling, Load Balancing)
```

## Service Configuration

### Service Configuration
```
Configuration Sources
├── Command-Line Flags
├── Environment Variables
├── Configuration Files (config.yaml)
├── Runtime Configuration
└── Default Configuration
```

### Service Parameters
```
Configurable Parameters
├── Node Parameters (addr, bootstrap, founder-pub)
├── Database Parameters (data-dir, quota)
├── API Parameters (api-port, tls-cert, tls-key)
├── Agent Parameters (max-agents, max-active-agents)
├── Provider Parameters (api-keys, timeouts)
├── Session Parameters (max-sessions, session-timeout)
├── EventBus Parameters (queue-size, handler-count)
├── Orchestrator Parameters (task-timeout, retry-count)
└── WebSocket Parameters (connection-limit, message-limit)
```

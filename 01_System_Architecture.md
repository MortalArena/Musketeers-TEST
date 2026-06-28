# Musketeers Backend - System Architecture

**Document Version:** 1.0  
**Date:** 2025-11-28  
**Phase:** 1.3 - Backend Audit Complete  
**Status:** Complete

---

## Executive Summary

Musketeers is a distributed, P2P-based multi-agent orchestration system built in Go. The system provides a comprehensive platform for managing AI agents, sessions, workflows, and decentralized communication through a libp2p-based network. The architecture is designed around a central orchestrator engine that coordinates multiple specialized subsystems including unified agents, session containers, capability matching, and real-time event distribution.

---

## 1. System Overview

### 1.1 Core Architecture Pattern

The Musketeers backend follows a **distributed orchestration pattern** with the following key characteristics:

- **P2P Network Foundation:** Built on libp2p for decentralized node communication
- **Event-Driven Architecture:** Central EventBus for decoupled component communication
- **Agent-Centric Design:** UnifiedAgent as the primary abstraction for AI agent management
- **Session-Based Isolation:** SessionContainer provides isolated execution environments
- **Capability-Based Matching:** Dynamic agent selection based on declared capabilities
- **Multi-Subsystem Integration:** 20+ specialized subsystems coordinated through OrchestratorEngine

### 1.2 Primary Entry Points

The system exposes 5 main entry points:

| Entry Point | Location | Purpose |
|-------------|----------|---------|
| `cmd/studio/main.go` | Central orchestrator | Manages all subsystems, REST API, WebSocket bridge |
| `cmd/agent/main.go` | Agent worker | Executes tasks, connects to Agent Bridge |
| `cmd/seed/main.go` | Bootstrap node | P2P network bootstrap seed |
| `cmd/founder/main.go` | Domain management | DHT-based domain registration/renewal |
| `cmd/gateway/main.go` | HTTP Gateway | External interface to P2P network |

---

## 2. Package Structure

### 2.1 Core Packages (`pkg/`)

The backend is organized into 28 packages under `pkg/`:

#### 2.1.1 Agent Management
- **`pkg/agent/`** - Core agent abstractions
  - `adapter.go` - AgentType, AgentCapability, UnifiedAgent interface
  - `registry.go` - AgentRegistry for agent registration and metadata
  - `unified/` - UnifiedAgent implementation with 20+ subsystems
    - `unified_agent.go` - Main UnifiedAgent struct
    - `session_manager.go` - Session lifecycle management
    - `thinking/` - ThinkingEngine with 16-phase workflow
    - `tools/` - Tool execution framework
    - `collaboration/` - Multi-agent coordination
    - `memory/` - Collective memory management
    - `skills/` - Skill tracking and synchronization

#### 2.1.2 Session Management
- **`pkg/session/`** - Session lifecycle and state
  - `container.go` - SessionContainer with memory, skills, workflow
  - `workflow.go` - 16-phase WorkflowEngine
  - `capability_verifier.go` - Agent capability probing
  - `chat_manager.go` - Chat message management
  - `task_manager.go` - Task execution tracking

#### 2.1.3 Orchestration
- **`pkg/orchestrator/`** - Central coordination
  - `orchestrator_engine.go` - OrchestratorEngine (580 lines)
  - Manages agent lifecycle, role assignment, capability matching
  - Integrates with UnifiedAgent and SessionContainer

#### 2.1.4 P2P Networking
- **`pkg/node/`** - P2P node implementation
  - `node.go` - Node struct with libp2p host, DHT, PubSub
  - `direct.go` - Direct messaging with NaCl encryption
  - Subsystems for network, storage, security, identity, messaging
- **`pkg/network/`** - Network utilities
  - `bootstrap.go` - BootstrapManager for P2P discovery
- **`pkg/protocol/`** - Protocol definitions
  - `messages.go` - ChannelMessage, DirectMessage, SiteManifest

#### 2.1.5 Cryptography & Security
- **`pkg/crypto/`** - Cryptographic primitives
  - `identity.go` - Ed25519 key pairs, DID generation, PoW mining
  - `sign.go` - Domain-separated signatures
  - `pow.go` - Proof-of-work with scrypt
- **`pkg/identity/`** - Identity management
  - `identity_record.go` - IdentityRecord with CRL support
- **`pkg/naming/`** - Domain naming
  - `dht.go` - DHT-based domain storage

#### 2.1.6 Communication
- **`pkg/channel/`** - Channel messaging
  - `public.go` - Public channels
  - `private.go` - Private encrypted channels with AES-GCM
- **`pkg/eventbus/`** - Event distribution
  - `bus.go` - EventBus with 10,000-event queue, Dead Letter Queue

#### 2.1.7 Data Storage
- **`pkg/storage/`** - Storage abstraction
  - `badger_storage.go` - BadgerDB implementation
  - `content/` - Content addressing and retrieval
- **`pkg/search/`** - Search functionality
  - `search.go` - Token bucket rate limiting

#### 2.1.8 AI Provider Integration
- **`pkg/providers/`** - AI provider abstraction
  - `types.go` - Provider interface, 23 provider types
  - Supports OpenAI, Anthropic, Google, DeepSeek, XAI, Mistral, Qwen, etc.
  - Local providers: Ollama

#### 2.1.9 Workflow Engine
- **`pkg/workflow/`** - Workflow definitions
  - `workflow.go` - Step types, Execution state

#### 2.1.10 Isolated Packages
The following packages are isolated and integrated via `cmd/studio/main.go`:
- `pkg/isolated/analytics/` - Analytics collection
- `pkg/isolated/backup/` - Backup management
- `pkg/isolated/delegation/` - Task delegation
- `pkg/isolated/discovery/` - Service discovery
- `pkg/isolated/hosting/` - Content hosting
- `pkg/isolated/notifications/` - Notification system
- `pkg/isolated/plugins/` - Plugin system
- `pkg/isolated/sandbox/` - Execution sandbox
- `pkg/isolated/upgrade/` - Upgrade management
- `pkg/isolated/validation/` - Validation services

#### 2.1.11 API Layer
- **`api/`** - HTTP/WebSocket API
  - `rest.go` - REST API server with session, chat, task, memory, skill, artifact endpoints
  - `local_ws_bridge.go` - WebSocket handler for real-time updates
  - `dashboard.go` - Embedded HTML dashboard (3653 lines)

---

## 3. Core Components

### 3.1 OrchestratorEngine

**Location:** `pkg/orchestrator/orchestrator_engine.go` (580 lines)

**Purpose:** Central coordination engine for agent lifecycle and task distribution

**Key Features:**
- Agent registration and deregistration
- Capability-based agent matching
- Task execution with verification
- Role assignment and lifecycle management
- Event bus integration
- Policy engine integration
- Statistics and metrics collection

**Key Methods:**
- `RegisterAgent(agent UnifiedAgent) error`
- `DeregisterAgent(agentID string) error`
- `ExecuteTask(task *AgentTask) (*TaskExecutionResult, error)`
- `MatchAgents(requiredCapabilities []AgentCapability) []UnifiedAgent`
- `SyncCapabilities() error`

**Integration Points:**
- References UnifiedAgent and SessionContainer
- Uses EventBus for decoupled communication
- Integrates with policy engine for decision making

---

### 3.2 UnifiedAgent

**Location:** `pkg/agent/unified/unified_agent.go`

**Purpose:** Comprehensive agent integrating 20+ subsystems

**Subsystems:**
1. SkillManager - Skill tracking and development
2. MemoryManager - Collective memory operations
3. SubagentManager - Subagent coordination
4. Automation - Automated task execution
5. Direction - Task direction and planning
6. Validation - Result validation
7. Coordination - Multi-agent coordination
8. FlowManager - Workflow flow control
9. ErrorHandler - Error handling and recovery
10. CollectiveSystems - Collective intelligence
11. SessionEventBus - Session-specific events
12. RealTimeSync - Real-time synchronization
13. ProblemSolutionRegistry - Problem-solution patterns
14. LocalMemoryCache - Local caching
15. DataCurator - Data curation
16. TaskScheduler - Task scheduling
17. SyncManager - Synchronization management
18. ProviderRegistry - AI provider registry
19. ToolExecutor - Tool execution
20. ThinkingEngine - 16-phase thinking process
21. WiringLayer - Component wiring
22. SessionContainer - Session state
23. SessionManager - Session lifecycle
24. AgentPool - Agent pool management
25. Metrics - Performance metrics

---

### 3.3 SessionContainer

**Location:** `pkg/session/container.go`

**Purpose:** Manages complete session state

**Components:**
- Metadata (ID, title, created_at, etc.)
- Memory (local and collective)
- Skills (session-specific skills)
- Workflow (16-phase workflow engine)
- Tasks (task tracking)
- Progress (progress tracking)
- Handoff (session handoff management)
- Aggregator (result aggregation)
- Reviewer (result review)
- ChatManager (chat messages)
- UnifiedSessionState (unified state)
- EventBus (session events)
- BadgerDB (persistence)
- CapabilityVerifier (capability verification)
- ContextReranker (context reranking - stored as interface{} to avoid import cycle)

---

### 3.4 EventBus

**Location:** `pkg/eventbus/bus.go` (205 lines)

**Purpose:** Central event distribution system

**Features:**
- 10,000-event queue to prevent goroutine leaks
- Dead Letter Queue for rejected events
- Wildcard handler support (`*`)
- Panic recovery with automatic restart
- Thread-safe with RWMutex
- Session-scoped events

**Event Structure:**
```go
type Event struct {
    Type      string
    Payload   interface{}
    Source    string
    Timestamp time.Time
    SessionID string
}
```

---

### 3.5 WorkflowEngine

**Location:** `pkg/session/workflow.go` (446 lines)

**Purpose:** Executes 16-phase workflow

**16 Phases:**
1. Understand Request
2. Analyze Context
3. Identify Tools
4. Plan Execution
5. Execute Tools
6. Verify Results
7. Handle Errors
8. Retry on Failure
9. Integrate Components
10. Sync State
11. Send Updates
12. Receive Responses
13. Analyze Final Results
14. Reflect and Learn
15. Save Lessons
16. Cleanup and Complete

**Key Features:**
- StepExecutor interface for actual execution
- Progress tracking (0-100%)
- Task management per phase
- State persistence
- Deadlock-safe mutex usage

---

### 3.6 AgentCapabilityVerifier

**Location:** `pkg/session/capability_verifier.go` (269 lines)

**Purpose:** Probes claimed agent capabilities with lightweight tasks

**Capabilities Tested:**
- Code Generation
- Code Review
- Testing
- Documentation
- Analysis
- Design
- File Operations
- Terminal Access
- Browser Control
- API Integration

**Features:**
- Cached verification results
- Configurable probe timeout (default 30s)
- Lightweight probe tasks per capability type
- VerificationReport with detailed probe results

---

### 3.7 P2P Node

**Location:** `pkg/node/node.go` (576 lines)

**Purpose:** P2P network node with libp2p

**Components:**
- NetworkSubsystem - libp2p host, DHT, PubSub
- StorageSubsystem - BadgerDB, content provider/fetcher
- SecuritySubsystem - NonceStore, CRL cache, rate limiter, validators
- IdentitySubsystem - KeyPair, IdentityRecord
- MessagingSubsystem - Message handling

**Protocols:**
- `/mskt/bitswap/1.0.0` - Bitswap protocol
- `/mskt/direct/1.0.0` - Direct messaging

---

### 3.8 BootstrapManager

**Location:** `pkg/network/bootstrap.go` (281 lines)

**Purpose:** Manages P2P network bootstrap

**Features:**
- Configurable bootstrap peers from environment
- Minimum connection threshold (default 5)
- Retry logic with configurable delay
- Periodic reconnection checks
- Statistics tracking

**Environment Variables:**
- `MUSKETEERS_BOOTSTRAP_PEERS` - Comma-separated multiaddrs
- `MUSKETEERS_MIN_CONNECTIONS` - Minimum connections

---

### 3.9 Cryptography

**Location:** `pkg/crypto/`

**Identity (`identity.go`):**
- Ed25519 key pair generation
- DID generation: `did:mskt:<base58(sha256(pub)[:16])>`
- Proof-of-work mining with scrypt
- PoW verification

**Signing (`sign.go`):**
- Domain-separated signatures (10 domain tags)
- Random nonce generation
- Hex encoding/decoding

**PoW (`pow.go`):**
- scrypt-based proof-of-work
- Configurable difficulty (18-24)
- Optimized mining function

---

### 3.10 REST API

**Location:** `api/rest.go`

**Endpoints:**
- Session management (create, get, list, delete)
- Chat (send, receive, history)
- Tasks (create, update, status)
- Progress tracking
- Memory (store, retrieve, search)
- Skills (list, register, sync)
- Artifacts (upload, download, list)
- MCP servers (list, register, tools)
- Event bus integration

**Features:**
- Local token authentication
- Rate limiting
- Session-scoped operations
- WebSocket bridge integration

---

### 3.11 WebSocket Bridge

**Location:** `api/local_ws_bridge.go`

**Purpose:** Real-time event streaming to clients

**Features:**
- Client connection management
- EventBus integration
- SessionContainer linking
- Origin checking (localhost only by default)
- Live updates for sessions, chats, tasks, progress

---

## 4. Data Flow

### 4.1 Task Execution Flow

```
User Request
    ↓
REST API / WebSocket
    ↓
SessionContainer
    ↓
WorkflowEngine (16 phases)
    ↓
OrchestratorEngine
    ↓
CapabilityMatcher
    ↓
UnifiedAgent (via ThinkingEngine)
    ↓
ToolExecutor
    ↓
ProviderRegistry → AI Provider
    ↓
Result → Verification → EventBus → Client
```

### 4.2 P2P Message Flow

```
Node A
    ↓
Encrypt (NaCl box / AES-GCM)
    ↓
Sign (Ed25519 with domain tag)
    ↓
libp2p Host
    ↓
DHT / PubSub
    ↓
Node B
    ↓
Verify Signature
    ↓
Decrypt
    ↓
NonceStore (replay prevention)
    ↓
Process Message
```

### 4.3 Event Flow

```
Component
    ↓
EventBus.Publish()
    ↓
Event Queue (10,000 capacity)
    ↓
processQueue() goroutine
    ↓
Handlers (including wildcard)
    ↓
Dead Letter Queue (if rejected)
```

---

## 5. Concurrency Model

### 5.1 Synchronization Primitives

- **sync.RWMutex** - Read-write locks for shared state
- **sync.Mutex** - Exclusive locks for critical sections
- **sync.WaitGroup** - Goroutine coordination
- **atomic.Bool** - Atomic boolean operations
- **Channels** - Goroutine communication

### 5.2 Goroutine Usage

- **EventBus.processQueue()** - Single goroutine for event processing
- **BootstrapManager** - Goroutines for parallel peer connections
- **WorkflowEngine** - Sequential step execution (no parallel goroutines during execution)
- **P2P Network** - libp2p manages its own goroutines

### 5.3 Deadlock Prevention

- WorkflowEngine uses internal `addTaskLocked()` to avoid re-entrant mutex calls
- EventBus uses defer recover() to prevent panic cascades
- NonceStore uses BadgerDB TTL for automatic cleanup

---

## 6. Security Architecture

### 6.1 Cryptographic Primitives

- **Ed25519** - Digital signatures and key pairs
- **Curve25519** - ECDH key exchange
- **AES-256-GCM** - Symmetric encryption for private channels
- **NaCl box** - Direct message encryption
- **scrypt** - Proof-of-work
- **SHA-256** - Hashing and DID generation

### 6.2 Domain Separation

10 domain tags prevent signature reuse:
- `NR-IDENTITY-V1|`
- `NR-REVOKE-V1|`
- `NR-DELEGATION-V1|`
- `NR-DOMAIN-FOUNDER-V1|`
- `NR-DOMAIN-OWNER-V1|`
- `NR-CHANNEL-MSG-V1|`
- `NR-SEARCH-V1|`
- `NR-DM-V1|`
- `NR-CHANNEL-CFG-V1|`
- `NR-ACP-V1|`

### 6.3 Replay Prevention

- NonceStore with TTL-based expiration
- BadgerDB automatic TTL cleanup
- Per-message nonce validation

### 6.4 Identity & Access Control

- DID-based identity: `did:mskt:<base58(sha256(pub)[:16])>`
- CRL (Certificate Revocation List) cache
- Founder public key validation
- Human client tracking

---

## 7. Storage Architecture

### 7.1 Primary Storage

- **BadgerDB** - Embedded key-value store
  - Location: `<data-dir>/badger`
  - Value log size: 16MB (reduced from 1GB for low disk space)
  - TTL support for automatic cleanup

### 7.2 Data Stored

- Identity records
- NonceStore (replay prevention)
- Session state
- Collective memory
- Skills data
- Workflow state
- Capability verification cache
- Domain records (DHT)

### 7.3 Content Addressing

- CID-based content storage
- Bitswap protocol for content distribution
- Provider records for content discovery

---

## 8. Network Architecture

### 8.1 P2P Stack

```
Application Layer
    ↓
Musketeers Protocols (/mskt/*)
    ↓
libp2p
    ↓
Transport (TCP, QUIC, WebTransport)
    ↓
Security (Noise, TLS)
    ↓
Multiplexing (Yamux)
```

### 8.2 Discovery

- DHT (Kademlia) for peer discovery
- Bootstrap peers for initial connection
- MDNS for local network discovery
- Periodic reconnection checks

### 8.3 Communication Patterns

- **PubSub** - Broadcast messaging
- **Direct** - 1:1 encrypted messaging
- **Bitswap** - Content exchange
- **DHT** - Distributed data storage

---

## 9. Configuration

### 9.1 Configuration File

**Location:** `config.example.yaml`

**Sections:**
- `server` - HTTP server settings (host, port, timeouts)
- `database` - Database configuration (type, host, port)
- `email` - SMTP settings
- `storage` - Storage configuration (type, path, quota)
- `network` - P2P network settings (listen addr, bootstrap peers)
- `security` - Encryption and TLS settings

### 9.2 Environment Variables

- `NR_POW_DIFFICULTY` - Proof-of-work difficulty (18-24)
- `MUSKETEERS_BOOTSTRAP_PEERS` - Bootstrap peer multiaddrs
- `MUSKETEERS_MIN_CONNECTIONS` - Minimum P2P connections
- `NR_REST_PORT` - REST API port
- `NR_FOUNDER_PUB` - Founder public key

---

## 10. Build & Deployment

### 10.1 Build System

**Makefile:**
```makefile
build:          # Build all binaries
test:           # Run tests
run-seed:       # Run seed node
run-agent:      # Run agent
clean:          # Clean build artifacts
docker:         # Build Docker image
```

### 10.2 Docker Configuration

**Dockerfile:**
- Multi-stage build
- Base: `golang:1.22-alpine`
- Runtime: `alpine:3.20`
- Builds: seed, agent, founder
- Exposes: 4001 (P2P), 8080 (REST)

**docker-compose.yml:**
- Seed service (port 4001)
- Agent service (ports 4002, 8080)
- Volume mounts for data persistence
- Depends on seed for bootstrap

---

## 11. Dependencies

### 11.1 Go Version

- **Go 1.25.3** (as specified in go.mod)

### 11.2 Direct Dependencies (32 packages)

**Core:**
- `filippo.io/edwards25519 v1.2.0` - Ed25519 cryptography
- `github.com/dgraph-io/badger/v4 v4.5.0` - Embedded KV store
- `github.com/libp2p/go-libp2p v0.36.2` - P2P networking
- `github.com/libp2p/go-libp2p-kad-dht v0.25.2` - Kademlia DHT
- `github.com/libp2p/go-libp2p-pubsub v0.12.0` - PubSub messaging

**Security:**
- `golang.org/x/crypto v0.46.0` - Cryptographic primitives
- `github.com/hashicorp/vault v1.21.4` - Secret management

**Logging:**
- `github.com/sirupsen/logrus v1.9.3` - Structured logging
- `go.uber.org/zap v1.27.0` - High-performance logging

**Networking:**
- `github.com/gorilla/websocket v1.5.4` - WebSocket support
- `github.com/multiformats/go-multiaddr v0.13.0` - Multiaddr

**Utilities:**
- `github.com/google/uuid v1.6.0` - UUID generation
- `github.com/robfig/cron/v3 v3.0.1` - Cron scheduling
- `github.com/tetratelabs/wazero v1.12.0` - WebAssembly runtime
- `github.com/tyler-smith/go-bip39 v1.1.0` - BIP39 mnemonic

**Monitoring:**
- `github.com/prometheus/client_golang v1.22.0` - Metrics
- `go.opentelemetry.io/otel v1.40.0` - OpenTelemetry

**Testing:**
- `github.com/stretchr/testify v1.11.1` - Test assertions

**Storage:**
- `github.com/klauspost/reedsolomon v1.14.0` - Reed-Solomon erasure coding

**Configuration:**
- `gopkg.in/yaml.v3 v3.0.1` - YAML parsing

### 11.3 Indirect Dependencies (120+ packages)

Major indirect dependencies include:
- libp2p ecosystem (40+ packages)
- Pion WebRTC stack (15+ packages)
- QUIC implementation (quic-go)
- Prometheus client libraries
- OpenTelemetry instrumentation
- Various crypto and math libraries

---

## 12. Integration Points

### 12.1 Internal Integration

**OrchestratorEngine ↔ UnifiedAgent:**
- OrchestratorEngine manages UnifiedAgent lifecycle
- UnifiedAgent registers with OrchestratorEngine
- Capability matching for task distribution

**OrchestratorEngine ↔ SessionContainer:**
- SessionContainer provides session context
- OrchestratorEngine executes tasks within sessions
- Event bus integration for state updates

**UnifiedAgent ↔ SessionContainer:**
- UnifiedAgent references SessionContainer for state
- SessionContainer persists agent state
- Shared event bus for real-time updates

**WorkflowEngine ↔ ThinkingEngine:**
- WorkflowEngine defines 16 phases
- ThinkingEngine implements StepExecutor interface
- Step-by-step execution with progress tracking

### 12.2 External Integration

**AI Providers:**
- 23 provider types via Provider interface
- OpenAI, Anthropic, Google, DeepSeek, XAI, Mistral, etc.
- Local providers: Ollama
- ProviderRegistry for provider management

**P2P Network:**
- libp2p for decentralized networking
- DHT for distributed data storage
- Bootstrap peers for network discovery

**Storage:**
- BadgerDB for embedded storage
- Content addressing via CID
- Bitswap for content distribution

---

## 13. Error Handling

### 13.1 Error Patterns

- Domain-specific errors in each package
- Wrapped errors with context
- Panic recovery in EventBus
- Retry logic in BootstrapManager
- Error aggregation in WorkflowEngine

### 13.2 Error Recovery

- EventBus: Automatic restart after panic
- WorkflowEngine: Phase-level error handling
- BootstrapManager: Retry with exponential backoff
- CapabilityVerifier: Partial verification support

---

## 14. Performance Considerations

### 14.1 Optimization Strategies

- **Event Queue:** 10,000 capacity to prevent blocking
- **BadgerDB:** 16MB value log for low disk space
- **Capability Verification:** Cached results
- **P2P Connection Pooling:** libp2p manages connections
- **Rate Limiting:** Token bucket for search operations

### 14.2 Resource Management

- Goroutine limits in EventBus
- TTL-based cleanup in NonceStore
- Connection limits in BootstrapManager
- Memory caching in various components

---

## 15. Known Limitations

### 15.1 Import Cycles

- SessionManager uses `interface{}` for OrchestratorEngine and ContextReranker to avoid import cycles
- This is documented in code comments

### 15.2 Bootstrap Peers

- Default bootstrap peers are placeholder values
- Must be configured via environment variable for production

### 15.3 Dashboard

- Embedded HTML dashboard (3653 lines) in Go file
- Not separated into frontend build process

---

## 16. Future Considerations

### 16.1 Scalability

- Consider horizontal scaling for OrchestratorEngine
- Evaluate distributed EventBus implementation
- P2P network scaling with more bootstrap peers

### 16.2 Security

- Consider adding TLS for all P2P connections
- Evaluate additional cryptographic primitives
- Implement more granular access control

### 16.3 Observability

- Expand OpenTelemetry instrumentation
- Add distributed tracing
- Enhance metrics collection

---

## 17. Conclusion

The Musketeers backend is a sophisticated distributed system with:

- **28 packages** organized into clear functional domains
- **5 entry points** for different operational modes
- **20+ subsystems** integrated through UnifiedAgent
- **16-phase workflow** for structured task execution
- **P2P networking** with libp2p for decentralization
- **Event-driven architecture** with robust EventBus
- **Capability-based matching** for dynamic agent selection
- **Comprehensive security** with domain-separated signatures
- **Embedded storage** with BadgerDB
- **23 AI provider integrations** for flexibility

The architecture is well-designed for its purpose, with clear separation of concerns, robust error handling, and extensive integration points for future expansion.

---

**Document End**

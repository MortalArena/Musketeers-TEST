# 🗡️ Musketeers

**"One for All, All for One"** — The Agent Operating System for the Decentralized Web

[![Go Report Card](https://goreportcard.com/badge/github.com/MortalArena/Musketeers)](https://goreportcard.com/report/github.com/MortalArena/Musketeers)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://go.dev/)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)]()

> **Musketeers** is a production-grade **Agent Operating System** that enables AI agents, IoT devices, and autonomous applications to discover, authenticate, communicate, and collaborate — all without centralized infrastructure. Built on battle-tested P2P primitives and modern cryptographic standards.

---

## 🌟 Overview

In a world where AI agents are becoming autonomous actors, **Musketeers** provides the missing layer: a decentralized operating system where agents can:

- 🆔 **Own their identity** via Self-Sovereign DIDs (`did:ia:...`)
- 🔐 **Communicate securely** with end-to-end encryption (NaCl box, AES-256-GCM)
- 🌐 **Discover each other** through a distributed hash table (Kademlia DHT)
- 📡 **Publish and subscribe** to channels via GossipSub
- 📦 **Store and retrieve** content using content-addressed storage (Bitswap-like)
- 🤝 **Collaborate in workflows** with multi-agent orchestration
- 🔑 **Manage capabilities** through a unified permission system (ABAC)
- 💼 **Integrate with external services** (GitHub, Gmail, Stripe, etc.) via a secure vault

Think of it as **"Linux + TCP/IP + AWS + Zapier"** — but for the age of autonomous agents.

---

## 🏗️ Architecture

Musketeers is built on a **6-layer architecture**, each layer composable and independently testable:

```
┌─────────────────────────────────────────────────────────┐
│  Layer 6: Economy (Token incentives, Payments)          │
├─────────────────────────────────────────────────────────┤
│  Layer 5: Applications (Marketplace, Bridge Bots)       │
├─────────────────────────────────────────────────────────┤
│  Layer 4: User Value (Agent Hub, Desktop/Mobile Apps)   │
├─────────────────────────────────────────────────────────┤
│  Layer 3: Network Upgrade (Domains .ia, ACP, Gateway)   │
├─────────────────────────────────────────────────────────┤
│  Layer 2: Protocols (Channels, Messaging, Bitswap)      │
├─────────────────────────────────────────────────────────┤
│  Layer 1: Infrastructure (DID, DHT, P2P, Cryptography)  │
└─────────────────────────────────────────────────────────┘
```

### Core Components

| Package | Description |
|---------|-------------|
| `pkg/runtime` | Agent Operating System — lifecycle, events, state, knowledge, scheduling |
| `pkg/policy` | ABAC (Attribute-Based Access Control) engine with multi-level approvals |
| `pkg/vault` | Encrypted secret storage with pluggable key providers (OS Keychain, HSM, KMS) |
| `pkg/capability` | Unified capability system with middleware pipeline |
| `pkg/workflow` | Graph-based workflow engine for multi-agent orchestration |
| `pkg/node` | Core P2P node built on libp2p, Kademlia DHT, and GossipSub |
| `pkg/channel` | Public (GossipSub) and private (AES-256-GCM) communication channels |
| `pkg/acp` | Agent Communication Protocol for task delegation |
| `pkg/content` | Content-addressed storage with Bitswap-like retrieval |
| `pkg/naming` | Decentralized naming system (`.ia` domains) with commit-reveal |
| `pkg/identity` | Self-sovereign identities with Ed25519 + BIP39 mnemonics |
| `pkg/crypto` | Cryptographic primitives (PoW, signatures, key derivation) |
| `pkg/registry` | Agent manifest registry (Kubernetes CRD-style) |
| `pkg/discovery` | Agent discovery with indexed search |
| `pkg/sdk` | Lightweight facade layer for external consumers |
| `pkg/gateway` | HTTP/HTTPS gateway for browser access to `.ia` sites |

---

## 🚀 Quick Start

### Prerequisites

- Go 1.21 or higher
- Git
- (Optional) Docker for containerized deployment

### Installation

```bash
# Clone the repository
git clone https://github.com/MortalArena/Musketeers.git
cd Musketeers

# Install dependencies
go mod download

# Build all executables
make build

# Or build individually
go build -o bin/agent ./cmd/agent
go build -o bin/seed ./cmd/seed
go build -o bin/founder ./cmd/founder
go build -o bin/gateway ./cmd/gateway
```

### Running a Local Network

**1. Start a seed (bootstrap) node:**

```bash
./bin/seed -port 4001
# Note the printed multiaddress, e.g.:
# /ip4/127.0.0.1/tcp/4001/p2p/12D3KooW...
```

**2. Start an agent node:**

```bash
./bin/agent -port 4002 -rest 8081 -init \
  -bootstrap "/ip4/127.0.0.1/tcp/4001/p2p/<SEED_PEER_ID>"
# Save the printed mnemonic (24 words) — this is your identity backup!
# Note the REST API token
```

**3. Start an HTTP gateway (optional):**

```bash
./bin/gateway -port 8090 -p2p-port 4010 \
  -bootstrap "/ip4/127.0.0.1/tcp/4001/p2p/<SEED_PEER_ID>"
```

### Using the REST API

```bash
# Health check
curl http://127.0.0.1:8081/api/health

# Get your identity
curl -H "Authorization: Bearer <TOKEN>" http://127.0.0.1:8081/api/identity

# Join a channel
curl -X POST -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"channel_id": "general"}' \
  http://127.0.0.1:8081/api/channels/join

# Publish a message
curl -X POST -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"channel_id": "general", "content": "Hello Musketeers!"}' \
  http://127.0.0.1:8081/api/channels/publish
```

### Web Dashboard

Open your browser to `http://127.0.0.1:8081/dashboard` for a visual interface.

---

## 🔐 Security Model

Musketeers implements defense-in-depth security at every layer:

### Cryptographic Primitives

| Function | Algorithm |
|----------|-----------|
| Signatures | Ed25519 |
| Symmetric Encryption | AES-256-GCM |
| Asymmetric Encryption | NaCl box (Curve25519 + XSalsa20-Poly1305) |
| Key Derivation | scrypt (N=131072, r=8, p=1) |
| Proof of Work | scrypt-based |
| Hashing | SHA-256, SHA-512, SHA3-256 |
| Key Exchange | ECDH on Curve25519 |

### Security Features

- ✅ **Domain Separation** — Each message type uses a unique prefix to prevent signature reuse
- ✅ **Nonce Store** — Prevents replay attacks with 1-hour TTL
- ✅ **Key Rotation** — Automatic key rotation when members are removed from private channels
- ✅ **Commit-Reveal** — Prevents front-running attacks on domain registration
- ✅ **Homograph Protection** — Rejects non-ASCII characters in domain names
- ✅ **Fail-Closed Revocation** — Assumes identity is revoked if CRL is unreachable
- ✅ **Memory Safety** — Bounded caches with automatic cleanup
- ✅ **Rate Limiting** — Token bucket algorithm on DHT operations
- ✅ **ABAC Policies** — Fine-grained attribute-based access control
- ✅ **Encrypted Vault** — Secrets encrypted at rest with AES-256-GCM
- ✅ **Pluggable Key Providers** — OS Keychain, TPM, HSM, AWS KMS, Hashicorp Vault

---

## 📖 Core Concepts

### 1. Self-Sovereign Identity (DID)

Every participant has a decentralized identifier:

```
did:ia:<base58(sha256(public_key)[:16])>
```

Identities are:
- Generated from Ed25519 keypairs
- Backed up with BIP39 24-word mnemonics
- Protected by Proof-of-Work (Sybil resistance)
- Revocable via CRL (Certificate Revocation List)

### 2. Decentralized Naming (.ia)

Register human-readable names:

```bash
# Register example.ia
./bin/founder -action register \
  -domain example.ia \
  -owner did:ia:... \
  -target did:ia:... \
  -expires 1735689600
```

Features:
- Commit-reveal registration (prevents front-running)
- Dual signatures (founder + owner)
- Automatic renewal
- Homograph attack protection

### 3. Communication Channels

**Public Channels (GossipSub):**
```go
node.JoinChannel(ctx, "general")
node.PublishChannelMessage(ctx, "general", "Hello world!")
```

**Private Channels (AES-256-GCM):**
```go
config, key, _ := channel.NewPrivateChannel("team", ownerDID, ownerPriv, members, admins)
encrypted, _ := channel.EncryptPrivateMessage("team", plaintext, key)
```

### 4. Direct Messaging

End-to-end encrypted 1:1 messaging with automatic chunking for large files:

```go
node.SendDirectToPeer(ctx, peerID, toDID, []byte("secret message"))
```

### 5. Agent Communication Protocol (ACP)

Delegate tasks between agents:

```go
response, _ := transport.SendTask(ctx, peerID, toDID, "translate", input, requestID)
```

Built-in tasks: `ping`, `echo`, `translate`, `execute` 

### 6. Content-Addressed Storage

Publish and retrieve content by hash:

```go
cid, _ := node.PublishContent(ctx, data)
data, _ := node.FetchContent(ctx, cid)
```

### 7. Capability System

Unified access to resources with ABAC policies:

```go
// Request GitHub access
cap, _ := client.Capabilities.RequestCapability(agentDID, "github", "read:repos")

// Execute a command
result, _ := cap.Execute(ctx, &github.ListReposCommand{Owner: "myorg"})
```

### 8. Workflow Engine

Graph-based orchestration for multi-agent tasks:

```go
graph := core.NewGraph("code-review", "Code Review Workflow")
graph.AddNode(&core.Node{ID: "analyze", Type: core.NodeTypeAgent, AgentID: "analyzer"})
graph.AddNode(&core.Node{ID: "review", Type: core.NodeTypeAgent, AgentID: "reviewer"})
graph.AddEdge(&core.Edge{From: "analyze", To: "review"})

execution, _ := executor.Execute(ctx, graph, state)
```

---

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run with race detection
go test -race ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./pkg/runtime/... -v
go test ./pkg/policy/... -v
go test ./pkg/workflow/... -v
```

---

## 📦 Docker

```bash
# Build Docker images
docker build -t musketeers-agent -f docker/Dockerfile.agent .
docker build -t musketeers-seed -f docker/Dockerfile.seed .
docker build -t musketeers-gateway -f docker/Dockerfile.gateway .

# Run with Docker Compose
docker-compose up -d
```

---

## 🗺️ Roadmap

### Phase 1: Foundation ✅ (Complete)
- [x] P2P networking with libp2p
- [x] Kademlia DHT
- [x] GossipSub pub/sub
- [x] Ed25519 identities
- [x] Encrypted channels
- [x] Content-addressed storage
- [x] Agent Communication Protocol

### Phase 2: Agent OS ✅ (Complete)
- [x] Runtime with lifecycle management
- [x] Event-driven architecture
- [x] State persistence
- [x] Knowledge systems (Working, Semantic, Episodic, Procedural)
- [x] ABAC policy engine
- [x] Encrypted vault
- [x] Capability pipeline
- [x] Workflow engine

### Phase 3: User Experience (In Progress)
- [ ] Desktop application (Tauri)
- [ ] Mobile application (React Native)
- [ ] Agent Hub with onboarding wizard
- [ ] Browser extension for `.ia` domains
- [ ] Bridge bots (WhatsApp, Telegram, Discord)

### Phase 4: Ecosystem (Planned)
- [ ] Agent Marketplace
- [ ] Storage economy with token incentives
- [ ] Payment rails (SOL, USDC)
- [ ] Enterprise plans
- [ ] 50 foundational `.ia` sites

---

## 🤝 Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) first.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Write tests for all new code (target: ≥70% coverage)
- Use structured logging via `pkg/runtime/observability` 
- Document all public APIs
- Run `golangci-lint` before committing

---

## 📄 License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

Musketeers builds on the shoulders of giants:

- **[libp2p](https://libp2p.io/)** — P2P networking stack
- **[IPFS](https://ipfs.tech/)** — Content-addressed storage concepts
- **[BadgerDB](https://github.com/dgraph-io/badger)** — Embedded key-value store
- **[OpenTelemetry](https://opentelemetry.io/)** — Observability standards
- **[Prometheus](https://prometheus.io/)** — Metrics collection
- **[Zap](https://github.com/uber-go/zap)** — Structured logging

---

## 📞 Contact

- **GitHub Issues:** [Report bugs or request features](https://github.com/MortalArena/Musketeers/issues)
- **Discussions:** [Join the conversation](https://github.com/MortalArena/Musketeers/discussions)
- **Twitter/X:** [@MusketeersOS](https://twitter.com/MusketeersOS)
- **Discord:** [Join our community](https://discord.gg/musketeers)

---

## 🌟 Star History

If Musketeers helps your project, consider giving us a star! ⭐

---

<p align="center">
  <b>"All for one, one for all"</b><br>
  Built with ❤️ by the Musketeers community
</p>

# PHASE 5 - Integration Blueprint Report

## Overview
This report analyzes every Dashboard feature to determine if the backend already supports it indirectly, another package already exposes it, an existing service can provide it, an existing interface already contains it, or if it only needs wiring/API handler/DTO/WebSocket subscription/frontend exposure.

**Key Finding**: 70-80% of the required backend functionality already exists. Most features only need API exposure, not new implementation.

---

## 1. Engineering Mapping Table

### 1.1 Provider Management Features

| Feature | Backend Exists | Needs API | Needs Handler | Needs DTO | Needs WebSocket | Needs UI | Needs Service | Needs Runtime | Needs New Logic | Priority | Files Involved | Dependencies | Risk | Estimated Effort |
|---------|---------------|----------|--------------|----------|----------------|---------|--------------|--------------|----------------|----------|----------------|--------------|------|------------------|
| Provider List Display | ✓ EXISTING | ✓ | ✓ | PARTIAL | ✓ | ✓ | NO | NO | NO | HIGH | pkg/providers/register.go | None | LOW | 2h |
| Provider Status Display | ✓ EXISTING | ✓ | ✓ | PARTIAL | ✓ | ✓ | NO | NO | NO | HIGH | pkg/providers/types.go | None | LOW | 2h |
| Provider Health Check | ✓ EXISTING | ✓ | ✓ | PARTIAL | ✓ | ✓ | NO | NO | NO | HIGH | pkg/providers/types.go | None | LOW | 2h |
| Provider Connection | ✓ EXISTING | ✓ | ✓ | PARTIAL | ✓ | ✓ | NO | NO | NO | HIGH | pkg/providers/types.go | None | LOW | 3h |
| Provider Disconnection | ✓ EXISTING | ✓ | ✓ | NO | ✓ | ✓ | NO | NO | NO | HIGH | pkg/providers/types.go | None | LOW | 2h |
| Provider Configuration | ✓ EXISTING | ✓ | ✓ | PARTIAL | NO | ✓ | NO | NO | NO | HIGH | pkg/providers/api_key_manager.go | None | LOW | 3h |
| Provider Models List | ✓ EXISTING | ✓ | ✓ | EXISTING | ✓ | ✓ | NO | NO | NO | HIGH | pkg/providers/model_catalog.go | None | LOW | 2h |
| Provider Capabilities Display | ✓ EXISTING | ✓ | ✓ | EXISTING | NO | ✓ | NO | NO | NO | MEDIUM | pkg/providers/types.go | None | LOW | 1h |

**Summary**: All provider management backend functionality EXISTS. Only needs API handlers and UI.

---

### 1.2 Model Management Features

| Feature | Backend Exists | Needs API | Needs Handler | Needs DTO | Needs WebSocket | Needs UI | Needs Service | Needs Runtime | Needs New Logic | Priority | Files Involved | Dependencies | Risk | Estimated Effort |
|---------|---------------|----------|--------------|----------|----------------|---------|--------------|--------------|----------------|----------|----------------|--------------|------|------------------|
| Model List Display | ✓ EXISTING | ✓ | ✓ | EXISTING | ✓ | ✓ | NO | NO | NO | HIGH | pkg/providers/model_catalog.go | None | LOW | 2h |
| Model Assignment to Roles | ✗ MISSING | ✓ | ✓ | NEW | ✓ | ✓ | ✓ NEW | PARTIAL | ✓ NEW | HIGH | NEW: pkg/providers/model_assignment.go | ModelCatalog, ProviderRegistry | MEDIUM | 8h |
| Model Capabilities Display | ✓ EXISTING | ✓ | ✓ | EXISTING | NO | ✓ | NO | NO | NO | MEDIUM | pkg/providers/types.go | None | LOW | 1h |

**Summary**: Model catalog EXISTS. Model assignment service needs NEW implementation.

---

### 1.3 Session Management Features

| Feature | Backend Exists | Needs API | Needs Handler | Needs DTO | Needs WebSocket | Needs UI | Needs Service | Needs Runtime | Needs New Logic | Priority | Files Involved | Dependencies | Risk | Estimated Effort |
|---------|---------------|----------|--------------|----------|----------------|---------|--------------|--------------|----------------|----------|----------------|--------------|------|------------------|
| Session List Display | ✓ EXISTING | PARTIAL | PARTIAL | EXISTING | ✓ | PARTIAL | NO | NO | NO | HIGH | pkg/agent/unified/session_manager.go | None | LOW | 2h |
| Session Creation | ✓ EXISTING | PARTIAL | PARTIAL | EXISTING | ✓ | PARTIAL | NO | NO | NO | HIGH | pkg/agent/unified/session_manager.go | None | LOW | 2h |
| Session Resume | ✓ EXISTING | PARTIAL | PARTIAL | EXISTING | ✓ | MISSING | NO | NO | NO | HIGH | pkg/agent/unified/session_manager.go | None | LOW | 2h |
| Session Pause | ✓ EXISTING | PARTIAL | PARTIAL | EXISTING | ✓ | MISSING | NO | NO | NO | HIGH | pkg/agent/unified/session_manager.go | None | LOW | 2h |
| Session Complete | ✓ EXISTING | PARTIAL | PARTIAL | EXISTING | ✓ | MISSING | NO | NO | NO | HIGH | pkg/agent/unified/session_manager.go | None | LOW | 2h |
| Session Duplicate | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | MEDIUM | pkg/agent/unified/session_manager.go | SessionContainer | MEDIUM | 6h |
| Session Rename | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | MEDIUM | pkg/agent/unified/session_manager.go | SessionContainer | MEDIUM | 4h |
| Session Archive | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | LOW | pkg/agent/unified/session_manager.go | SessionContainer | MEDIUM | 6h |
| Session Delete | ✓ EXISTING | PARTIAL | PARTIAL | EXISTING | ✓ | MISSING | NO | NO | NO | HIGH | pkg/agent/unified/session_manager.go | None | LOW | 2h |
| Session Export | ✗ MISSING | ✓ | ✓ | NEW | NO | MISSING | ✓ NEW | PARTIAL | ✓ NEW | LOW | pkg/agent/unified/session_manager.go | SessionContainer | MEDIUM | 8h |
| Session Import | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | LOW | pkg/agent/unified/session_manager.go | SessionContainer | MEDIUM | 8h |
| Session Messages Display | ✓ EXISTING | PARTIAL | PARTIAL | EXISTING | ✓ | MISSING | NO | NO | NO | HIGH | pkg/session/chat.go | None | LOW | 2h |
| Session Artifacts Display | ✓ EXISTING | PARTIAL | PARTIAL | EXISTING | ✓ | MISSING | NO | NO | NO | MEDIUM | api/rest.go (artifacts map) | None | LOW | 2h |
| Session Files Display | PARTIAL | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | MEDIUM | pkg/session/container.go | Storage | MEDIUM | 6h |
| Session Execution Tree Display | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | ✓ NEW | ✓ NEW | LOW | pkg/agent/unified/unified_agent.go | TaskManager, ProgressTracker | HIGH | 12h |
| Session Agent Timeline Display | PARTIAL | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | MEDIUM | pkg/agent/unified/unified_agent.go | SubagentManager | MEDIUM | 8h |
| Session Task Graph Display | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | ✓ NEW | ✓ NEW | LOW | pkg/session/task_manager.go | TaskManager | HIGH | 12h |
| Session Memory Display | ✓ EXISTING | PARTIAL | PARTIAL | EXISTING | ✓ | MISSING | NO | NO | NO | HIGH | pkg/session/memory.go | None | LOW | 2h |
| Session Context Display | PARTIAL | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | MEDIUM | pkg/session/container.go | CollectiveMemory | MEDIUM | 6h |
| Session Token Usage Display | PARTIAL | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | MEDIUM | pkg/providers/types.go | TokenUsage | MEDIUM | 4h |
| Session Execution Time Display | PARTIAL | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | MEDIUM | pkg/agent/registry.go | AgentStats | MEDIUM | 4h |

**Summary**: Core session operations EXIST. Advanced operations (duplicate, archive, export, import) need NEW methods. Visualization features (execution tree, task graph) need NEW tracking.

---

### 1.4 Agent Management Features

| Feature | Backend Exists | Needs API | Needs Handler | Needs DTO | Needs WebSocket | Needs UI | Needs Service | Needs Runtime | Needs New Logic | Priority | Files Involved | Dependencies | Risk | Estimated Effort |
|---------|---------------|----------|--------------|----------|----------------|---------|--------------|--------------|----------------|----------|----------------|--------------|------|------------------|
| Agent List Display | ✓ EXISTING | PARTIAL | PARTIAL | EXISTING | ✓ | PARTIAL | NO | NO | NO | HIGH | pkg/agent/registry.go | None | LOW | 2h |
| Agent Registration | ✓ EXISTING | PARTIAL | PARTIAL | EXISTING | ✓ | PARTIAL | NO | NO | NO | HIGH | pkg/agent/registry.go | None | LOW | 2h |
| Agent Status Display | ✓ EXISTING | ✓ | ✓ | EXISTING | ✓ | MISSING | NO | NO | NO | HIGH | pkg/agent/registry.go | None | LOW | 2h |
| Agent State Display | ✓ EXISTING | ✓ | ✓ | EXISTING | ✓ | MISSING | NO | NO | NO | HIGH | pkg/agent/registry.go | None | LOW | 2h |
| Agent Task Display | PARTIAL | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | HIGH | pkg/agent/unified/unified_agent.go | SubagentManager | MEDIUM | 6h |
| Agent Progress Display | ✓ EXISTING | PARTIAL | PARTIAL | EXISTING | ✓ | MISSING | NO | NO | NO | HIGH | pkg/session/progress_tracker.go | None | LOW | 2h |
| Agent Statistics Display | ✓ EXISTING | ✓ | ✓ | EXISTING | ✓ | MISSING | NO | NO | NO | MEDIUM | pkg/agent/registry.go | None | LOW | 2h |
| Agent Health Check | ✓ EXISTING | ✓ | ✓ | EXISTING | ✓ | MISSING | NO | NO | NO | HIGH | pkg/agent/registry.go | None | LOW | 2h |

**Summary**: Agent registry and statistics EXIST. Agent task tracking needs NEW logic.

---

### 1.5 Tool Management Features

| Feature | Backend Exists | Needs API | Needs Handler | Needs DTO | Needs WebSocket | Needs UI | Needs Service | Needs Runtime | Needs New Logic | Priority | Files Involved | Dependencies | Risk | Estimated Effort |
|---------|---------------|----------|--------------|----------|----------------|---------|--------------|--------------|----------------|----------|----------------|--------------|------|------------------|
| Tool List Display | ✓ EXISTING | ✓ | ✓ | EXISTING | ✓ | MISSING | NO | NO | NO | HIGH | pkg/agent/tools/registry.go | None | LOW | 2h |
| Tool Execution | ✓ EXISTING | ✓ | ✓ | EXISTING | ✓ | MISSING | NO | NO | NO | HIGH | pkg/agent/tools/executor.go | None | LOW | 3h |
| Tool Dry Run | ✓ EXISTING | ✓ | ✓ | NEW | ✓ | MISSING | NO | NO | NO | MEDIUM | pkg/agent/tools/executor.go | None | LOW | 2h |
| Tool Disable/Enable | ✓ EXISTING | ✓ | ✓ | NEW | ✓ | MISSING | NO | NO | NO | MEDIUM | pkg/agent/tools/registry.go | None | LOW | 2h |
| Tool Statistics Display | PARTIAL | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | MEDIUM | pkg/agent/tools/executor.go | ToolExecutor | MEDIUM | 4h |
| Tool Logs Display | PARTIAL | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | MEDIUM | pkg/logger/ | Logger | MEDIUM | 4h |

**Summary**: Tool registry and executor EXIST. Tool statistics and logs need NEW aggregation.

---

### 1.6 Memory Management Features

| Feature | Backend Exists | Needs API | Needs Handler | Needs DTO | Needs WebSocket | Needs UI | Needs Service | Needs Runtime | Needs New Logic | Priority | Files Involved | Dependencies | Risk | Estimated Effort |
|---------|---------------|----------|--------------|----------|----------------|---------|--------------|--------------|----------------|----------|----------------|--------------|------|------------------|
| Memory List Display | ✓ EXISTING | PARTIAL | PARTIAL | EXISTING | ✓ | MISSING | NO | NO | NO | HIGH | pkg/session/memory.go | None | LOW | 2h |
| Memory Search | ✓ EXISTING | PARTIAL | PARTIAL | EXISTING | ✓ | MISSING | NO | NO | NO | HIGH | pkg/session/memory.go | None | LOW | 2h |
| Memory Delete | ✓ EXISTING | PARTIAL | PARTIAL | EXISTING | ✓ | MISSING | NO | NO | NO | MEDIUM | pkg/session/memory.go | None | LOW | 2h |
| Memory Export | ✗ MISSING | ✓ | ✓ | NEW | NO | MISSING | ✓ NEW | PARTIAL | ✓ NEW | LOW | pkg/session/memory.go | CollectiveMemory | MEDIUM | 6h |
| Memory Import | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | LOW | pkg/session/memory.go | CollectiveMemory | MEDIUM | 6h |
| Memory Rebuild Embeddings | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | ✓ NEW | ✓ NEW | LOW | pkg/agent/thinking/embeddings.go | Embeddings | HIGH | 8h |
| Memory Inspection | ✓ EXISTING | PARTIAL | PARTIAL | EXISTING | ✓ | MISSING | NO | NO | NO | MEDIUM | pkg/session/memory.go | None | LOW | 2h |
| Memory Origin Tracing | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | ✓ NEW | ✓ NEW | LOW | pkg/session/memory.go | CollectiveMemory | HIGH | 10h |
| Memory Growth Graph | PARTIAL | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | MEDIUM | pkg/session/memory.go | CollectiveMemory | MEDIUM | 6h |

**Summary**: Core memory operations EXIST. Advanced operations (export, import, rebuild embeddings, origin tracing) need NEW implementation.

---

### 1.7 System Health Features

| Feature | Backend Exists | Needs API | Needs Handler | Needs DTO | Needs WebSocket | Needs UI | Needs Service | Needs Runtime | Needs New Logic | Priority | Files Involved | Dependencies | Risk | Estimated Effort |
|---------|---------------|----------|--------------|----------|----------------|---------|--------------|--------------|----------------|----------|----------------|--------------|------|------------------|
| Backend Health Check | ✓ EXISTING | PARTIAL | PARTIAL | EXISTING | ✓ | MISSING | NO | NO | NO | HIGH | api/rest.go (handleHealth) | None | LOW | 1h |
| Database Health Check | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | ✓ NEW | ✓ NEW | MEDIUM | NEW: pkg/health/database.go | BadgerDB | MEDIUM | 4h |
| Storage Health Check | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | ✓ NEW | ✓ NEW | MEDIUM | NEW: pkg/health/storage.go | pkg/storage | MEDIUM | 4h |
| Memory Health Check | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | ✓ NEW | ✓ NEW | MEDIUM | NEW: pkg/health/memory.go | pkg/memory | MEDIUM | 4h |
| Provider Connectivity Check | ✓ EXISTING | ✓ | ✓ | EXISTING | ✓ | MISSING | NO | NO | NO | HIGH | pkg/providers/types.go | None | LOW | 2h |
| Agent Runtime Health Check | ✓ EXISTING | ✓ | ✓ | EXISTING | ✓ | MISSING | NO | NO | NO | HIGH | pkg/agent/registry.go | None | LOW | 2h |
| Tool Registry Health Check | ✓ EXISTING | ✓ | ✓ | EXISTING | ✓ | MISSING | NO | NO | NO | MEDIUM | pkg/agent/tools/registry.go | None | LOW | 2h |
| API Status Check | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | ✓ NEW | ✓ NEW | MEDIUM | NEW: pkg/health/api.go | api/rest.go | MEDIUM | 4h |
| WebSocket Status Check | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | ✓ NEW | ✓ NEW | MEDIUM | NEW: pkg/health/websocket.go | api/local_ws_bridge.go | MEDIUM | 4h |
| Synchronization Health Check | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | ✓ NEW | ✓ NEW | LOW | NEW: pkg/health/sync.go | pkg/eventbus | MEDIUM | 4h |
| Execution Queue Health Check | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | ✓ NEW | ✓ NEW | LOW | NEW: pkg/health/queue.go | TaskManager | MEDIUM | 4h |
| Scheduler Health Check | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | ✓ NEW | ✓ NEW | LOW | NEW: pkg/health/scheduler.go | TaskScheduler | MEDIUM | 4h |
| Worker Status Check | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | ✓ NEW | ✓ NEW | LOW | NEW: pkg/health/worker.go | AgentPool | MEDIUM | 4h |

**Summary**: Basic health check EXISTS. Detailed subsystem health checks need NEW health service.

---

### 1.8 Observability Features

| Feature | Backend Exists | Needs API | Needs Handler | Needs DTO | Needs WebSocket | Needs UI | Needs Service | Needs Runtime | Needs New Logic | Priority | Files Involved | Dependencies | Risk | Estimated Effort |
|---------|---------------|----------|--------------|----------|----------------|---------|--------------|--------------|----------------|----------|----------------|--------------|------|------------------|
| CPU Metrics | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | ✓ NEW | ✓ NEW | MEDIUM | NEW: pkg/metrics/cpu.go | OS | MEDIUM | 6h |
| RAM Metrics | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | ✓ NEW | ✓ NEW | MEDIUM | NEW: pkg/metrics/ram.go | OS | MEDIUM | 6h |
| GPU Metrics | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | ✓ NEW | ✓ NEW | LOW | NEW: pkg/metrics/gpu.go | GPU libraries | HIGH | 8h |
| Latency Metrics | PARTIAL | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | HIGH | pkg/providers/types.go | None | LOW | 4h |
| API Call Metrics | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | ✓ NEW | ✓ NEW | MEDIUM | NEW: pkg/metrics/api.go | api/rest.go | MEDIUM | 6h |
| Request Metrics | PARTIAL | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | MEDIUM | security/rate_limiter.go | None | LOW | 4h |
| Error Metrics | PARTIAL | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | MEDIUM | pkg/logger/ | Logger | LOW | 4h |
| Token Metrics | PARTIAL | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | HIGH | pkg/providers/types.go | None | LOW | 4h |
| Streaming Metrics | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | ✓ NEW | ✓ NEW | MEDIUM | NEW: pkg/metrics/streaming.go | Provider | MEDIUM | 6h |
| Session Metrics | PARTIAL | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | HIGH | pkg/agent/unified/session_manager.go | None | LOW | 4h |
| Worker Metrics | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | ✓ NEW | ✓ NEW | MEDIUM | NEW: pkg/metrics/worker.go | AgentPool | MEDIUM | 6h |
| Queue Metrics | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | ✓ NEW | ✓ NEW | MEDIUM | NEW: pkg/metrics/queue.go | TaskManager | MEDIUM | 6h |
| WebSocket Metrics | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | ✓ NEW | ✓ NEW | MEDIUM | NEW: pkg/metrics/websocket.go | WebSocket | MEDIUM | 6h |

**Summary**: Basic metrics tracking EXISTS (latency, tokens, errors). System metrics (CPU, RAM, GPU) need NEW collection.

---

### 1.9 Logging Features

| Feature | Backend Exists | Needs API | Needs Handler | Needs DTO | Needs WebSocket | Needs UI | Needs Service | Needs Runtime | Needs New Logic | Priority | Files Involved | Dependencies | Risk | Estimated Effort |
|---------|---------------|----------|--------------|----------|----------------|---------|--------------|--------------|----------------|----------|----------------|--------------|------|------------------|
| Log Streaming | ✗ MISSING | ✓ | ✓ | EXISTING | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | HIGH | pkg/logger/ | Logger | MEDIUM | 6h |
| Log Filtering | ✓ EXISTING | ✓ | ✓ | NEW | ✓ | MISSING | NO | NO | NO | HIGH | pkg/logger/ | None | LOW | 2h |
| Log Search | ✓ EXISTING | ✓ | ✓ | NEW | ✓ | MISSING | NO | NO | NO | HIGH | pkg/logger/ | None | LOW | 2h |
| Log Export | ✗ MISSING | ✓ | ✓ | NEW | NO | MISSING | ✓ NEW | PARTIAL | ✓ NEW | MEDIUM | pkg/logger/ | Logger | MEDIUM | 4h |

**Summary**: Logger EXISTS. Log streaming needs NEW aggregation service.

---

### 1.10 Event Features

| Feature | Backend Exists | Needs API | Needs Handler | Needs DTO | Needs WebSocket | Needs UI | Needs Service | Needs Runtime | Needs New Logic | Priority | Files Involved | Dependencies | Risk | Estimated Effort |
|---------|---------------|----------|--------------|----------|----------------|---------|--------------|--------------|----------------|----------|----------------|--------------|------|------------------|
| Event Streaming | ✗ MISSING | ✓ | ✓ | EXISTING | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | HIGH | pkg/eventbus/bus.go | EventBus | MEDIUM | 6h |
| Event Timeline Display | ✗ MISSING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | MEDIUM | pkg/eventbus/bus.go | EventBus | MEDIUM | 6h |
| Event Filtering | ✓ EXISTING | ✓ | ✓ | NEW | ✓ | MISSING | NO | NO | NO | HIGH | pkg/eventbus/bus.go | None | LOW | 2h |
| Event Search | ✓ EXISTING | ✓ | ✓ | NEW | ✓ | MISSING | NO | NO | NO | HIGH | pkg/eventbus/bus.go | None | LOW | 2h |

**Summary**: EventBus EXISTS. Event streaming needs NEW aggregation service.

---

### 1.11 Configuration Features

| Feature | Backend Exists | Needs API | Needs Handler | Needs DTO | Needs WebSocket | Needs UI | Needs Service | Needs Runtime | Needs New Logic | Priority | Files Involved | Dependencies | Risk | Estimated Effort |
|---------|---------------|----------|--------------|----------|----------------|---------|--------------|--------------|----------------|----------|----------------|--------------|------|------------------|
| Configuration Display | ✓ EXISTING | ✓ | ✓ | EXISTING | NO | MISSING | NO | NO | NO | HIGH | pkg/config/config.go | None | LOW | 2h |
| Configuration Edit | ✓ EXISTING | ✓ | ✓ | EXISTING | NO | MISSING | NO | NO | NO | HIGH | pkg/config/config.go | None | LOW | 3h |
| Configuration Validation | ✓ EXISTING | ✓ | ✓ | EXISTING | NO | MISSING | NO | NO | NO | HIGH | pkg/config/config.go | None | LOW | 2h |
| Configuration Apply | ✓ EXISTING | ✓ | ✓ | EXISTING | NO | MISSING | NO | NO | NO | HIGH | pkg/config/config.go | None | LOW | 2h |
| Configuration Rollback | ✗ MISSING | ✓ | ✓ | NEW | NO | MISSING | ✓ NEW | PARTIAL | ✓ NEW | MEDIUM | pkg/config/config.go | Config | MEDIUM | 6h |

**Summary**: Config EXISTS. Rollback needs NEW implementation.

---

### 1.12 File Explorer Features

| Feature | Backend Exists | Needs API | Needs Handler | Needs DTO | Needs WebSocket | Needs UI | Needs Service | Needs Runtime | Needs New Logic | Priority | Files Involved | Dependencies | Risk | Estimated Effort |
|---------|---------------|----------|--------------|----------|----------------|---------|--------------|--------------|----------------|----------|----------------|--------------|------|------------------|
| File List Display | ✓ EXISTING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | MEDIUM | os package | None | LOW | 4h |
| File Preview | ✓ EXISTING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | MEDIUM | os package | None | LOW | 4h |
| File Download | ✓ EXISTING | ✓ | ✓ | NEW | NO | MISSING | ✓ NEW | PARTIAL | ✓ NEW | MEDIUM | os package | None | LOW | 2h |
| File Upload | ✓ EXISTING | ✓ | ✓ | NEW | ✓ | MISSING | ✓ NEW | PARTIAL | ✓ NEW | MEDIUM | os package | None | LOW | 3h |
| File Comparison | ✗ MISSING | ✓ | ✓ | NEW | NO | MISSING | ✓ NEW | ✓ NEW | ✓ NEW | LOW | NEW: pkg/storage/diff.go | Storage | HIGH | 10h |
| File Diff Viewer | ✗ MISSING | ✓ | ✓ | NEW | NO | MISSING | ✓ NEW | ✓ NEW | ✓ NEW | LOW | NEW: pkg/storage/diff.go | Storage | HIGH | 10h |

**Summary**: Basic file operations EXIST (via OS). Advanced operations (comparison, diff) need NEW implementation.

---

## 2. Dependency Graph

### 2.1 Provider Management Flow

```
Dashboard (Providers Page)
    ↓
REST Endpoint: GET /api/providers
    ↓
Handler: handleProviders (NEW)
    ↓
Service: ProviderRegistry (EXISTS - pkg/providers/register.go)
    ↓
Internal Runtime: Provider.List() (EXISTS)
    ↓
Provider Interface (EXISTS - pkg/providers/types.go)
    ↓
Individual Providers (EXISTS - pkg/providers/builtin/*)
```

**Status**: Backend EXISTS. Only needs Handler + DTO + UI.

---

### 2.2 Model Assignment Flow

```
Dashboard (Models Page)
    ↓
REST Endpoint: POST /api/models/assign
    ↓
Handler: handleModelAssignment (NEW)
    ↓
Service: ModelAssignment (NEW - needs creation)
    ↓
Internal Runtime: ModelAssignment.SetManagerModel() (NEW)
    ↓
ProviderRegistry (EXISTS - pkg/providers/register.go)
    ↓
ModelCatalog (EXISTS - pkg/providers/model_catalog.go)
    ↓
Router (EXISTS - pkg/providers/router.go)
```

**Status**: Backend PARTIAL. ModelAssignment service needs NEW implementation.

---

### 2.3 Agent Orchestration Flow

```
Dashboard (Agent Orchestration View)
    ↓
REST Endpoint: GET /api/agents/orchestration
    ↓
Handler: handleAgentOrchestration (NEW)
    ↓
Service: AgentPool (EXISTS - pkg/agent/unified/agent_pool.go)
    ↓
Internal Runtime: AgentPool.GetAllAgents() (EXISTS)
    ↓
UnifiedAgent (EXISTS - pkg/agent/unified/unified_agent.go)
    ↓
SubagentManager (EXISTS - pkg/agent/subagents/subagent_manager.go)
    ↓
ThinkingEngine (EXISTS - pkg/agent/thinking/thinking_engine.go)
    ↓
ToolExecutor (EXISTS - pkg/agent/tools/executor.go)
    ↓
ProviderRegistry (EXISTS - pkg/providers/register.go)
```

**Status**: Backend EXISTS. Only needs Handler + DTO + WebSocket + UI.

---

### 2.4 Session Management Flow

```
Dashboard (Session Center)
    ↓
REST Endpoint: GET/POST/PUT/DELETE /api/sessions
    ↓
Handler: handleSessions (PARTIAL - exists, needs enhancement)
    ↓
Service: SessionManager (EXISTS - pkg/agent/unified/session_manager.go)
    ↓
Internal Runtime: SessionManager.CreateSession() (EXISTS)
    ↓
SessionContainer (EXISTS - pkg/session/container.go)
    ↓
ChatManager (EXISTS - pkg/session/chat.go)
    ↓
TaskManager (EXISTS - pkg/session/task_manager.go)
    ↓
ProgressTracker (EXISTS - pkg/session/progress_tracker.go)
    ↓
CollectiveMemory (EXISTS - pkg/session/memory.go)
```

**Status**: Backend EXISTS. Only needs Handler enhancement + WebSocket + UI.

---

### 2.5 Tool Management Flow

```
Dashboard (Tools Page)
    ↓
REST Endpoint: GET /api/tools
    ↓
Handler: handleTools (NEW)
    ↓
Service: ToolRegistry (EXISTS - pkg/agent/tools/registry.go)
    ↓
Internal Runtime: ToolRegistry.List() (EXISTS)
    ↓
ToolExecutor (EXISTS - pkg/agent/tools/executor.go)
    ↓
Tool Adapters (EXISTS - pkg/agent/adapters/*)
```

**Status**: Backend EXISTS. Only needs Handler + DTO + UI.

---

### 2.6 Memory Management Flow

```
Dashboard (Memory Page)
    ↓
REST Endpoint: GET /api/memory/{session_id}
    ↓
Handler: handleMemoryBySession (PARTIAL - exists, needs enhancement)
    ↓
Service: CollectiveMemory (EXISTS - pkg/session/memory.go)
    ↓
Internal Runtime: CollectiveMemory.GetAll() (EXISTS)
    ↓
Memory Subsystems (EXISTS - pkg/memory/*)
```

**Status**: Backend EXISTS. Only needs Handler enhancement + WebSocket + UI.

---

### 2.7 System Health Flow

```
Dashboard (System Health Page)
    ↓
REST Endpoint: GET /api/health
    ↓
Handler: handleHealth (PARTIAL - exists, needs enhancement)
    ↓
Service: HealthChecker (NEW - needs creation)
    ↓
Internal Runtime: HealthChecker.CheckAll() (NEW)
    ↓
Subsystem Health Checks (NEW - needs creation)
    ├── Database Health (NEW)
    ├── Storage Health (NEW)
    ├── Memory Health (NEW)
    ├── Provider Connectivity (EXISTS)
    ├── Agent Runtime (EXISTS)
    ├── Tool Registry (EXISTS)
    ├── API Status (NEW)
    ├── WebSocket Status (NEW)
    └── Synchronization (NEW)
```

**Status**: Backend PARTIAL. HealthChecker service needs NEW implementation.

---

### 2.8 Observability Flow

```
Dashboard (Observability Page)
    ↓
REST Endpoint: GET /api/metrics
    ↓
Handler: handleMetrics (NEW)
    ↓
Service: Metrics (PARTIAL - pkg/metrics/metrics.go exists, needs expansion)
    ↓
Internal Runtime: Metrics.Get() (PARTIAL)
    ↓
Metric Collectors (NEW - needs creation)
    ├── CPU Collector (NEW)
    ├── RAM Collector (NEW)
    ├── GPU Collector (NEW)
    ├── Latency Collector (PARTIAL)
    ├── API Call Collector (NEW)
    ├── Request Collector (PARTIAL)
    ├── Error Collector (PARTIAL)
    ├── Token Collector (PARTIAL)
    ├── Streaming Collector (NEW)
    ├── Session Collector (PARTIAL)
    ├── Worker Collector (NEW)
    ├── Queue Collector (NEW)
    └── WebSocket Collector (NEW)
```

**Status**: Backend PARTIAL. Metrics service needs expansion with NEW collectors.

---

### 2.9 Log Streaming Flow

```
Dashboard (Log Viewer)
    ↓
WebSocket Endpoint: /api/ws/logs
    ↓
Handler: handleLogWebSocket (NEW)
    ↓
Service: LogStreamer (NEW - needs creation)
    ↓
Internal Runtime: LogStreamer.Stream() (NEW)
    ↓
Logger (EXISTS - pkg/logger/*)
```

**Status**: Backend EXISTS. LogStreamer service needs NEW implementation.

---

### 2.10 Event Streaming Flow

```
Dashboard (Event Timeline)
    ↓
WebSocket Endpoint: /api/ws/events
    ↓
Handler: handleEventWebSocket (NEW)
    ↓
Service: EventStreamer (NEW - needs creation)
    ↓
Internal Runtime: EventStreamer.Stream() (NEW)
    ↓
EventBus (EXISTS - pkg/eventbus/bus.go)
```

**Status**: Backend EXISTS. EventStreamer service needs NEW implementation.

---

## 3. Missing API Report

### 3.1 Provider APIs

| Endpoint | HTTP Method | Handler | Service | DTO | Response | Frontend Consumer | Status |
|----------|-------------|---------|---------|-----|----------|-------------------|--------|
| /api/providers | GET | handleProviders (NEW) | ProviderRegistry (EXISTS) | ProviderStatusDTO (NEW) | ProviderStatus[] | ProvidersPage | READY |
| /api/providers/{id} | GET | handleProviderByID (NEW) | ProviderRegistry (EXISTS) | ProviderDetailDTO (NEW) | ProviderDetail | ProvidersPage | READY |
| /api/providers/{id}/health | POST | handleProviderHealth (NEW) | Provider (EXISTS) | HealthCheckDTO (NEW) | HealthStatus | ProvidersPage | READY |
| /api/providers/{id}/connect | POST | handleProviderConnect (NEW) | Provider (EXISTS) | ProviderConfigDTO (EXISTS) | ConnectionStatus | ProvidersPage | READY |
| /api/providers/{id}/disconnect | POST | handleProviderDisconnect (NEW) | Provider (EXISTS) | None | Status | ProvidersPage | READY |
| /api/providers/{id}/config | PUT | handleProviderConfig (NEW) | APIKeyManager (EXISTS) | ProviderConfigDTO (EXISTS) | ConfigStatus | ProvidersPage | READY |
| /api/providers/{id}/models | GET | handleProviderModels (NEW) | Provider (EXISTS) | ModelInfoDTO (EXISTS) | ModelInfo[] | ProvidersPage | READY |

**Summary**: All provider APIs are READY - only handlers and DTOs needed.

---

### 3.2 Model APIs

| Endpoint | HTTP Method | Handler | Service | DTO | Response | Frontend Consumer | Status |
|----------|-------------|---------|---------|-----|----------|-------------------|--------|
| /api/models | GET | handleModels (NEW) | ModelCatalog (EXISTS) | ModelInfoDTO (EXISTS) | ModelInfo[] | ModelsPage | READY |
| /api/models/assign | POST | handleModelAssignment (NEW) | ModelAssignment (NEW) | ModelAssignmentDTO (NEW) | AssignmentStatus | ModelsPage | PARTIAL |
| /api/models/{provider} | GET | handleModelsByProvider (NEW) | ModelCatalog (EXISTS) | ModelInfoDTO (EXISTS) | ModelInfo[] | ModelsPage | READY |

**Summary**: Model list APIs are READY. Model assignment needs NEW service.

---

### 3.3 Agent APIs

| Endpoint | HTTP Method | Handler | Service | DTO | Response | Frontend Consumer | Status |
|----------|-------------|---------|---------|-----|----------|-------------------|--------|
| /api/agents/orchestration | GET | handleAgentOrchestration (NEW) | AgentPool (EXISTS) | AgentOrchestrationDTO (NEW) | AgentOrchestration | AgentOrchestrationView | READY |
| /api/agents/{id}/state | GET | handleAgentState (NEW) | UnifiedAgent (EXISTS) | AgentStateDTO (NEW) | AgentState | AgentOrchestrationView | READY |
| /api/agents/{id}/task | GET | handleAgentTask (NEW) | SubagentManager (EXISTS) | AgentTaskDTO (NEW) | AgentTask | AgentOrchestrationView | PARTIAL |

**Summary**: Agent APIs are READY. Agent task tracking needs NEW logic.

---

### 3.4 Tool APIs

| Endpoint | HTTP Method | Handler | Service | DTO | Response | Frontend Consumer | Status |
|----------|-------------|---------|---------|-----|----------|-------------------|--------|
| /api/tools | GET | handleTools (NEW) | ToolRegistry (EXISTS) | ToolInfoDTO (NEW) | ToolInfo[] | ToolsPage | READY |
| /api/tools/{id}/execute | POST | handleToolExecute (NEW) | ToolExecutor (EXISTS) | ToolExecutionDTO (NEW) | ToolResult | ToolsPage | READY |
| /api/tools/{id}/dry-run | POST | handleToolDryRun (NEW) | ToolExecutor (EXISTS) | ToolExecutionDTO (NEW) | ToolResult | ToolsPage | READY |
| /api/tools/{id}/enable | POST | handleToolEnable (NEW) | ToolRegistry (EXISTS) | None | Status | ToolsPage | READY |
| /api/tools/{id}/disable | POST | handleToolDisable (NEW) | ToolRegistry (EXISTS) | None | Status | ToolsPage | READY |

**Summary**: Tool APIs are READY - only handlers and DTOs needed.

---

### 3.5 Memory APIs

| Endpoint | HTTP Method | Handler | Service | DTO | Response | Frontend Consumer | Status |
|----------|-------------|---------|---------|-----|----------|-------------------|--------|
| /api/memory/{session_id} | GET | handleMemoryBySession (PARTIAL) | CollectiveMemory (EXISTS) | MemoryDTO (NEW) | MemoryData | MemoryPage | READY |
| /api/memory/{session_id}/search | POST | handleMemorySearch (PARTIAL) | CollectiveMemory (EXISTS) | SearchDTO (NEW) | SearchResult | MemoryPage | READY |
| /api/memory/{session_id}/export | POST | handleMemoryExport (NEW) | CollectiveMemory (EXISTS) | ExportDTO (NEW) | ExportData | MemoryPage | PARTIAL |
| /api/memory/{session_id}/import | POST | handleMemoryImport (NEW) | CollectiveMemory (EXISTS) | ImportDTO (NEW) | ImportStatus | MemoryPage | PARTIAL |

**Summary**: Core memory APIs are READY. Export/Import need NEW logic.

---

### 3.6 Health APIs

| Endpoint | HTTP Method | Handler | Service | DTO | Response | Frontend Consumer | Status |
|----------|-------------|---------|---------|-----|----------|-------------------|--------|
| /api/health | GET | handleHealth (PARTIAL) | HealthChecker (NEW) | HealthReportDTO (NEW) | HealthReport | SystemHealthPage | PARTIAL |
| /api/health/database | GET | handleDatabaseHealth (NEW) | DatabaseHealthChecker (NEW) | DatabaseHealthDTO (NEW) | DatabaseHealth | SystemHealthPage | PARTIAL |
| /api/health/storage | GET | handleStorageHealth (NEW) | StorageHealthChecker (NEW) | StorageHealthDTO (NEW) | StorageHealth | SystemHealthPage | PARTIAL |
| /api/health/memory | GET | handleMemoryHealth (NEW) | MemoryHealthChecker (NEW) | MemoryHealthDTO (NEW) | MemoryHealth | SystemHealthPage | PARTIAL |
| /api/health/providers | GET | handleProviderHealth (NEW) | ProviderRegistry (EXISTS) | ProviderHealthDTO (NEW) | ProviderHealth[] | SystemHealthPage | READY |
| /api/health/agents | GET | handleAgentHealth (NEW) | AgentRegistry (EXISTS) | AgentHealthDTO (NEW) | AgentHealth | SystemHealthPage | READY |
| /api/health/tools | GET | handleToolHealth (NEW) | ToolRegistry (EXISTS) | ToolHealthDTO (NEW) | ToolHealth | SystemHealthPage | READY |
| /api/health/api | GET | handleAPIHealth (NEW) | APIHealthChecker (NEW) | APIHealthDTO (NEW) | APIHealth | SystemHealthPage | PARTIAL |
| /api/health/websocket | GET | handleWebSocketHealth (NEW) | WebSocketHealthChecker (NEW) | WebSocketHealthDTO (NEW) | WebSocketHealth | SystemHealthPage | PARTIAL |

**Summary**: Provider/Agent/Tool health APIs are READY. Subsystem health APIs need NEW services.

---

### 3.7 Metrics APIs

| Endpoint | HTTP Method | Handler | Service | DTO | Response | Frontend Consumer | Status |
|----------|-------------|---------|---------|-----|----------|-------------------|--------|
| /api/metrics | GET | handleMetrics (NEW) | Metrics (PARTIAL) | MetricsDTO (NEW) | MetricsData | ObservabilityPage | PARTIAL |
| /api/metrics/cpu | GET | handleCPUMetrics (NEW) | CPUMetricsCollector (NEW) | CPUMetricsDTO (NEW) | CPUMetrics | ObservabilityPage | PARTIAL |
| /api/metrics/ram | GET | handleRAMMetrics (NEW) | RAMMetricsCollector (NEW) | RAMMetricsDTO (NEW) | RAMMetrics | ObservabilityPage | PARTIAL |
| /api/metrics/gpu | GET | handleGPUMetrics (NEW) | GPUMetricsCollector (NEW) | GPUMetricsDTO (NEW) | GPUMetrics | ObservabilityPage | PARTIAL |
| /api/metrics/tokens | GET | handleTokenMetrics (NEW) | TokenMetricsCollector (PARTIAL) | TokenMetricsDTO (NEW) | TokenMetrics | ObservabilityPage | READY |
| /api/metrics/latency | GET | handleLatencyMetrics (NEW) | LatencyMetricsCollector (PARTIAL) | LatencyMetricsDTO (NEW) | LatencyMetrics | ObservabilityPage | READY |

**Summary**: Token/Latency metrics APIs are READY. System metrics APIs need NEW collectors.

---

### 3.8 WebSocket Endpoints

| Endpoint | Handler | Service | DTO | Frontend Consumer | Status |
|----------|---------|---------|-----|-------------------|--------|
| /api/ws/chat/{session_id} | handleChatWebSocket (NEW) | ChatManager (EXISTS) | MessageDTO (EXISTS) | ChatWorkspace | READY |
| /api/ws/agents | handleAgentWebSocket (NEW) | AgentPool (EXISTS) | AgentStateDTO (NEW) | AgentOrchestrationView | READY |
| /api/ws/events | handleEventWebSocket (NEW) | EventStreamer (NEW) | EventDTO (EXISTS) | EventTimeline | PARTIAL |
| /api/ws/logs | handleLogWebSocket (NEW) | LogStreamer (NEW) | LogEntryDTO (EXISTS) | LogViewer | PARTIAL |
| /api/ws/metrics | handleMetricsWebSocket (NEW) | MetricsStreamer (NEW) | MetricsDTO (NEW) | ObservabilityPage | PARTIAL |

**Summary**: Chat/Agent WebSocket are READY. Event/Log/Metrics streaming need NEW services.

---

## 4. Dashboard Wiring Report

### 4.1 Providers Page

**Consumes**:
- ProviderRegistry (EXISTS - pkg/providers/register.go)
  - List()
  - Get()
  - GetProviderByName()
- APIKeyManager (EXISTS - pkg/providers/api_key_manager.go)
  - SetKey()
  - GetKey()
  - DeleteKey()
  - HasKey()
- ModelCatalog (EXISTS - pkg/providers/model_catalog.go)
  - GetModelsByProvider()
  - GetModel()
- Provider Interface (EXISTS - pkg/providers/types.go)
  - Initialize()
  - Close()
  - Ping()
  - Status()
  - ListModels()

**REST Endpoints Needed**:
- GET /api/providers
- GET /api/providers/{id}
- POST /api/providers/{id}/health
- POST /api/providers/{id}/connect
- POST /api/providers/{id}/disconnect
- PUT /api/providers/{id}/config
- GET /api/providers/{id}/models

**WebSocket Channels Needed**:
- ws://providers/status

**Status**: READY - All backend exists. Only needs API exposure.

---

### 4.2 Models Page

**Consumes**:
- ModelCatalog (EXISTS - pkg/providers/model_catalog.go)
  - ListAllModels()
  - GetModelsByProvider()
  - GetModelsByCapability()
  - SearchModels()
- ModelAssignment (NEW - needs creation)
  - SetManagerModel()
  - SetPlannerModel()
  - SetCoderModel()
  - SetReviewerModel()
  - SetReasoningModel()
  - SetEmbeddingModel()
  - GetModelAssignment()
- ProviderRegistry (EXISTS - pkg/providers/register.go)
  - List()
  - Get()
- Router (EXISTS - pkg/providers/router.go)
  - Route()

**REST Endpoints Needed**:
- GET /api/models
- GET /api/models/{provider}
- POST /api/models/assign

**WebSocket Channels Needed**:
- ws://models/status

**Status**: PARTIAL - ModelCatalog EXISTS, ModelAssignment needs NEW implementation.

---

### 4.3 Session Center

**Consumes**:
- SessionManager (EXISTS - pkg/agent/unified/session_manager.go)
  - CreateSession()
  - GetSession()
  - ListSessions()
  - PauseSession()
  - ResumeSession()
  - CompleteSession()
  - DeleteSession()
  - DuplicateSession() (NEW)
  - RenameSession() (NEW)
  - ArchiveSession() (NEW)
  - ExportSession() (NEW)
  - ImportSession() (NEW)
- SessionContainer (EXISTS - pkg/session/container.go)
  - GetChatManager()
  - GetTaskManager()
  - GetProgressTracker()
  - GetCollectiveMemory()
  - GetSkillsManager()
- ChatManager (EXISTS - pkg/session/chat.go)
  - GetMessages()
  - SendMessage()
- TaskManager (EXISTS - pkg/session/task_manager.go)
  - GetTasks()
  - GetTaskGraph() (NEW)
- ProgressTracker (EXISTS - pkg/session/progress_tracker.go)
  - GetProgress()
- CollectiveMemory (EXISTS - pkg/session/memory.go)
  - GetAll()
  - Search()

**REST Endpoints Needed**:
- GET /api/sessions (EXISTS - needs enhancement)
- POST /api/sessions (EXISTS - needs enhancement)
- GET /api/sessions/{id} (EXISTS - needs enhancement)
- PUT /api/sessions/{id} (EXISTS - needs enhancement)
- DELETE /api/sessions/{id} (EXISTS - needs enhancement)
- POST /api/sessions/{id}?action=pause (EXISTS - needs enhancement)
- POST /api/sessions/{id}?action=resume (EXISTS - needs enhancement)
- POST /api/sessions/{id}?action=complete (EXISTS - needs enhancement)
- POST /api/sessions/{id}?action=duplicate (NEW)
- POST /api/sessions/{id}?action=rename (NEW)
- POST /api/sessions/{id}?action=archive (NEW)
- POST /api/sessions/{id}?action=export (NEW)
- POST /api/sessions/{id}?action=import (NEW)

**WebSocket Channels Needed**:
- ws://sessions/status
- ws://sessions/{id}/messages
- ws://sessions/{id}/progress
- ws://sessions/{id}/tasks

**Status**: PARTIAL - Core operations EXIST, advanced operations need NEW methods.

---

### 4.4 Chat Workspace

**Consumes**:
- ChatManager (EXISTS - pkg/session/chat.go)
  - GetMessages()
  - SendMessage()
  - StreamMessage()
- UnifiedAgent (EXISTS - pkg/agent/unified/unified_agent.go)
  - ProcessMessage()
  - StreamResponse()
- ThinkingEngine (EXISTS - pkg/agent/thinking/thinking_engine.go)
  - Think()
  - Reason()
- Provider (EXISTS - pkg/providers/types.go)
  - Complete()
  - StreamComplete()
- ToolExecutor (EXISTS - pkg/agent/tools/executor.go)
  - Execute()
- ProgressTracker (EXISTS - pkg/session/progress_tracker.go)
  - GetProgress()

**REST Endpoints Needed**:
- GET /api/messages/{session_id} (EXISTS - needs enhancement)
- POST /api/messages/{session_id} (EXISTS - needs enhancement)
- POST /api/messages/{session_id}/cancel (NEW)
- POST /api/messages/{session_id}/retry (NEW)
- POST /api/messages/{session_id}/fork (NEW)
- POST /api/messages/{session_id}/continue (NEW)

**WebSocket Channels Needed**:
- ws://chat/{session_id} (EXISTS - needs enhancement)
- ws://chat/{session_id}/streaming (NEW)

**Status**: READY - All backend exists. Only needs API exposure and streaming.

---

### 4.5 Agent Orchestration View

**Consumes**:
- AgentPool (EXISTS - pkg/agent/unified/agent_pool.go)
  - GetAllAgents()
  - GetAgentsBySession()
  - GetAgentState()
  - GetAgentTask()
  - GetAgentProgress()
- UnifiedAgent (EXISTS - pkg/agent/unified/unified_agent.go)
  - GetCurrentTask()
  - GetCurrentStep()
  - GetProvider()
  - GetModel()
  - GetLatency()
  - GetContext()
  - GetCurrentTool()
  - GetMemoryUsage()
  - GetTokens()
  - GetExecutionTime()
- SubagentManager (EXISTS - pkg/agent/subagents/subagent_manager.go)
  - GetSubagent()
  - GetSubagentState()
- AgentRegistry (EXISTS - pkg/agent/registry.go)
  - GetMetadata()
  - GetStats()
  - HealthCheck()

**REST Endpoints Needed**:
- GET /api/agents/orchestration (NEW)
- GET /api/agents/orchestration/{session_id} (NEW)
- GET /api/agents/{id}/state (NEW)
- GET /api/agents/{id}/task (NEW)

**WebSocket Channels Needed**:
- ws://agents/state
- ws://agents/{id}/task
- ws://agents/{id}/progress

**Status**: READY - All backend exists. Only needs API exposure and WebSocket.

---

### 4.6 Tools Page

**Consumes**:
- ToolRegistry (EXISTS - pkg/agent/tools/registry.go)
  - List()
  - Get()
  - Enable()
  - Disable()
- ToolExecutor (EXISTS - pkg/agent/tools/executor.go)
  - Execute()
  - DryRun()
  - GetStats()
- Tool Adapters (EXISTS - pkg/agent/adapters/*)
  - Execute()

**REST Endpoints Needed**:
- GET /api/tools (NEW)
- GET /api/tools/{id} (NEW)
- POST /api/tools/{id}/execute (NEW)
- POST /api/tools/{id}/dry-run (NEW)
- POST /api/tools/{id}/enable (NEW)
- POST /api/tools/{id}/disable (NEW)

**WebSocket Channels Needed**:
- ws://tools/status
- ws://tools/{id}/execution

**Status**: READY - All backend exists. Only needs API exposure.

---

### 4.7 Memory Page

**Consumes**:
- CollectiveMemory (EXISTS - pkg/session/memory.go)
  - GetAll()
  - Get()
  - Set()
  - Search()
  - Delete()
  - Export() (NEW)
  - Import() (NEW)
- Embeddings (EXISTS - pkg/agent/thinking/embeddings.go)
  - RebuildEmbeddings() (NEW)
- Memory Subsystems (EXISTS - pkg/memory/*)
  - GetWorkingMemory()
  - GetShortTermMemory()
  - GetLongTermMemory()
  - GetSessionMemory()
  - GetSharedMemory()
  - GetKnowledgeStore()

**REST Endpoints Needed**:
- GET /api/memory/{session_id} (EXISTS - needs enhancement)
- POST /api/memory/{session_id}/search (EXISTS - needs enhancement)
- DELETE /api/memory/{session_id}/{key} (NEW)
- POST /api/memory/{session_id}/export (NEW)
- POST /api/memory/{session_id}/import (NEW)
- POST /api/memory/{session_id}/rebuild-embeddings (NEW)

**WebSocket Channels Needed**:
- ws://memory/{session_id}
- ws://memory/{session_id}/search

**Status**: PARTIAL - Core operations EXIST, advanced operations need NEW methods.

---

### 4.8 System Health Page

**Consumes**:
- HealthChecker (NEW - needs creation)
  - CheckAll()
  - CheckDatabase() (NEW)
  - CheckStorage() (NEW)
  - CheckMemory() (NEW)
  - CheckProviders() (EXISTS)
  - CheckAgents() (EXISTS)
  - CheckTools() (EXISTS)
  - CheckAPI() (NEW)
  - CheckWebSocket() (NEW)
  - CheckSynchronization() (NEW)
- ProviderRegistry (EXISTS - pkg/providers/register.go)
  - List()
- AgentRegistry (EXISTS - pkg/agent/registry.go)
  - HealthCheck()
- ToolRegistry (EXISTS - pkg/agent/tools/registry.go)
  - List()

**REST Endpoints Needed**:
- GET /api/health (EXISTS - needs enhancement)
- GET /api/health/database (NEW)
- GET /api/health/storage (NEW)
- GET /api/health/memory (NEW)
- GET /api/health/providers (NEW)
- GET /api/health/agents (NEW)
- GET /api/health/tools (NEW)
- GET /api/health/api (NEW)
- GET /api/health/websocket (NEW)

**WebSocket Channels Needed**:
- ws://health/status
- ws://health/{subsystem}

**Status**: PARTIAL - Basic health check EXISTS, detailed checks need NEW service.

---

### 4.9 Observability Page

**Consumes**:
- Metrics (PARTIAL - pkg/metrics/metrics.go)
  - Get() (EXISTS)
  - GetHistory() (NEW)
- Metrics Collectors (NEW - needs creation)
  - CPUMetricsCollector (NEW)
  - RAMMetricsCollector (NEW)
  - GPUMetricsCollector (NEW)
  - LatencyMetricsCollector (PARTIAL)
  - APICallMetricsCollector (NEW)
  - RequestMetricsCollector (PARTIAL)
  - ErrorMetricsCollector (PARTIAL)
  - TokenMetricsCollector (PARTIAL)
  - StreamingMetricsCollector (NEW)
  - SessionMetricsCollector (PARTIAL)
  - WorkerMetricsCollector (NEW)
  - QueueMetricsCollector (NEW)
  - WebSocketMetricsCollector (NEW)

**REST Endpoints Needed**:
- GET /api/metrics (NEW)
- GET /api/metrics/cpu (NEW)
- GET /api/metrics/ram (NEW)
- GET /api/metrics/gpu (NEW)
- GET /api/metrics/tokens (NEW)
- GET /api/metrics/latency (NEW)

**WebSocket Channels Needed**:
- ws://metrics/stream
- ws://metrics/{type}

**Status**: PARTIAL - Basic metrics EXISTS, system metrics need NEW collectors.

---

### 4.10 Log Viewer

**Consumes**:
- Logger (EXISTS - pkg/logger/*)
  - Log()
  - GetHistory() (NEW)
  - Filter() (NEW)
- LogStreamer (NEW - needs creation)
  - Stream() (NEW)

**REST Endpoints Needed**:
- GET /api/logs (NEW)
- GET /api/logs/filter (NEW)

**WebSocket Channels Needed**:
- ws://logs/stream

**Status**: PARTIAL - Logger EXISTS, streaming needs NEW service.

---

### 4.11 Event Timeline

**Consumes**:
- EventBus (EXISTS - pkg/eventbus/bus.go)
  - Publish()
  - Subscribe()
  - GetHistory() (NEW)
- EventStreamer (NEW - needs creation)
  - Stream() (NEW)

**REST Endpoints Needed**:
- GET /api/events (NEW)

**WebSocket Channels Needed**:
- ws://events/stream

**Status**: PARTIAL - EventBus EXISTS, streaming needs NEW service.

---

### 4.12 Configuration Page

**Consumes**:
- Config (EXISTS - pkg/config/config.go)
  - Load()
  - Save()
  - Validate()
  - Apply()
  - Rollback() (NEW)

**REST Endpoints Needed**:
- GET /api/config (NEW)
- PUT /api/config (NEW)
- POST /api/config/validate (NEW)
- POST /api/config/apply (NEW)
- POST /api/config/rollback (NEW)

**WebSocket Channels Needed**:
- None

**Status**: PARTIAL - Config EXISTS, rollback needs NEW implementation.

---

### 4.13 File Explorer

**Consumes**:
- OS Package (EXISTS)
  - ReadDir()
  - ReadFile()
  - WriteFile()
- Storage (PARTIAL - pkg/storage/*)
  - List() (NEW)
  - Get() (NEW)
  - Upload() (NEW)
  - Download() (NEW)
  - Compare() (NEW)
  - Diff() (NEW)

**REST Endpoints Needed**:
- GET /api/files (NEW)
- GET /api/files/{path} (NEW)
- POST /api/files/upload (NEW)
- GET /api/files/{path}/download (NEW)
- GET /api/files/{path}/preview (NEW)
- GET /api/files/{path1}/compare/{path2} (NEW)

**WebSocket Channels Needed**:
- None

**Status**: PARTIAL - OS operations EXIST, advanced operations need NEW service.

---

## 5. Engineering Readiness Report

### 5.1 Classification Summary

#### READY (Backend complete, only frontend needed)
- Provider Management (100%)
- Agent List Display (100%)
- Agent Registration (100%)
- Agent Status Display (100%)
- Agent Progress Display (100%)
- Agent Statistics Display (100%)
- Agent Health Check (100%)
- Tool List Display (100%)
- Tool Execution (100%)
- Tool Dry Run (100%)
- Tool Disable/Enable (100%)
- Memory List Display (100%)
- Memory Search (100%)
- Memory Delete (100%)
- Memory Inspection (100%)
- Session List Display (100%)
- Session Creation (100%)
- Session Resume (100%)
- Session Pause (100%)
- Session Complete (100%)
- Session Delete (100%)
- Session Messages Display (100%)
- Session Artifacts Display (100%)
- Session Memory Display (100%)
- Chat Workspace (100%)
- Provider Connectivity Check (100%)
- Agent Runtime Health Check (100%)
- Tool Registry Health Check (100%)
- Token Metrics (100%)
- Latency Metrics (100%)
- Configuration Display (100%)
- Configuration Edit (100%)
- Configuration Validation (100%)
- Configuration Apply (100%)

**Total READY**: 35 features (40%)

---

#### WIRE (Backend exists, only API exposure required)
- Provider Status Display (WIRE)
- Provider Health Check (WIRE)
- Provider Connection (WIRE)
- Provider Disconnection (WIRE)
- Provider Configuration (WIRE)
- Provider Models List (WIRE)
- Provider Capabilities Display (WIRE)
- Model List Display (WIRE)
- Model Capabilities Display (WIRE)
- Agent State Display (WIRE)
- Agent Orchestration View (WIRE)
- Session Handlers Enhancement (WIRE)
- Memory Handlers Enhancement (WIRE)
- Provider Health API (WIRE)
- Agent Health API (WIRE)
- Tool Health API (WIRE)
- Token Metrics API (WIRE)
- Latency Metrics API (WIRE)
- Chat WebSocket (WIRE)
- Agent WebSocket (WIRE)

**Total WIRE**: 19 features (22%)

---

#### PARTIAL (Backend partially exists, requires extension)
- Model Assignment (PARTIAL - needs ModelAssignment service)
- Session Duplicate (PARTIAL - needs method)
- Session Rename (PARTIAL - needs method)
- Session Archive (PARTIAL - needs method)
- Session Export (PARTIAL - needs method)
- Session Import (PARTIAL - needs method)
- Session Files Display (PARTIAL - needs tracking)
- Session Agent Timeline Display (PARTIAL - needs tracking)
- Session Context Display (PARTIAL - needs tracking)
- Session Token Usage Display (PARTIAL - needs aggregation)
- Session Execution Time Display (PARTIAL - needs aggregation)
- Agent Task Display (PARTIAL - needs tracking)
- Tool Statistics Display (PARTIAL - needs aggregation)
- Tool Logs Display (PARTIAL - needs aggregation)
- Memory Export (PARTIAL - needs method)
- Memory Import (PARTIAL - needs method)
- Memory Growth Graph (PARTIAL - needs tracking)
- System Health Check (PARTIAL - needs HealthChecker service)
- Database Health Check (PARTIAL - needs service)
- Storage Health Check (PARTIAL - needs service)
- Memory Health Check (PARTIAL - needs service)
- API Status Check (PARTIAL - needs service)
- WebSocket Status Check (PARTIAL - needs service)
- Synchronization Health Check (PARTIAL - needs service)
- Execution Queue Health Check (PARTIAL - needs service)
- Scheduler Health Check (PARTIAL - needs service)
- Worker Status Check (PARTIAL - needs service)
- Metrics Service (PARTIAL - needs expansion)
- CPU Metrics (PARTIAL - needs collector)
- RAM Metrics (PARTIAL - needs collector)
- GPU Metrics (PARTIAL - needs collector)
- API Call Metrics (PARTIAL - needs collector)
- Request Metrics (PARTIAL - needs collector)
- Error Metrics (PARTIAL - needs collector)
- Streaming Metrics (PARTIAL - needs collector)
- Session Metrics (PARTIAL - needs collector)
- Worker Metrics (PARTIAL - needs collector)
- Queue Metrics (PARTIAL - needs collector)
- WebSocket Metrics (PARTIAL - needs collector)
- Log Streaming (PARTIAL - needs LogStreamer service)
- Event Streaming (PARTIAL - needs EventStreamer service)
- Configuration Rollback (PARTIAL - needs method)
- File Explorer (PARTIAL - needs service)

**Total PARTIAL**: 42 features (48%)

---

#### NEW (Backend capability genuinely missing)
- Session Execution Tree Display (NEW - needs tracking service)
- Session Task Graph Display (NEW - needs tracking service)
- Memory Rebuild Embeddings (NEW - needs service)
- Memory Origin Tracing (NEW - needs service)
- File Comparison (NEW - needs service)
- File Diff Viewer (NEW - needs service)

**Total NEW**: 6 features (7%)

---

### 5.2 Overall Statistics

- **READY**: 35 features (40%)
- **WIRE**: 19 features (22%)
- **PARTIAL**: 42 features (48%)
- **NEW**: 6 features (7%)

**Total Features**: 87 features

**Key Insight**: 62% of features are READY or WIRE (backend complete or mostly complete). Only 7% require genuinely new backend implementation.

---

## 6. Implementation Priority

### Phase 1: READY Features (Low Risk, Fast Implementation)
**Estimated Effort**: 70 hours
**Risk**: LOW
**Priority**: HIGH

1. Provider Management APIs (8h)
2. Agent Orchestration APIs (6h)
3. Tool Management APIs (6h)
4. Memory Core APIs (4h)
5. Session Core APIs (6h)
6. Chat Workspace APIs (4h)
7. Health Core APIs (6h)
8. Metrics Core APIs (8h)
9. Configuration APIs (6h)
10. WebSocket Wiring (12h)
11. DTO Creation (10h)

---

### Phase 2: WIRE Features (Low Risk, Medium Implementation)
**Estimated Effort**: 40 hours
**Risk**: LOW
**Priority**: HIGH

1. Provider Status/Health APIs (4h)
2. Model List APIs (2h)
3. Agent State APIs (2h)
4. Session Handler Enhancements (6h)
5. Memory Handler Enhancements (4h)
6. Health API Enhancements (6h)
7. Metrics API Enhancements (8h)
8. WebSocket Enhancements (8h)

---

### Phase 3: PARTIAL Features (Medium Risk, Medium Implementation)
**Estimated Effort**: 120 hours
**Risk**: MEDIUM
**Priority**: MEDIUM

1. Model Assignment Service (8h)
2. Session Advanced Operations (24h)
3. Session Tracking Services (20h)
4. Tool Statistics/Logs (8h)
5. Memory Advanced Operations (12h)
6. Health Checker Service (20h)
7. Metrics Collectors (20h)
8. Log/Event Streamers (8h)

---

### Phase 4: NEW Features (High Risk, High Implementation)
**Estimated Effort**: 50 hours
**Risk**: HIGH
**Priority**: LOW

1. Session Execution Tree (12h)
2. Session Task Graph (12h)
3. Memory Rebuild Embeddings (8h)
4. Memory Origin Tracing (10h)
5. File Comparison (10h)
6. File Diff Viewer (8h)

---

## 7. Total Estimated Effort

- **Phase 1 (READY)**: 70 hours
- **Phase 2 (WIRE)**: 40 hours
- **Phase 3 (PARTIAL)**: 120 hours
- **Phase 4 (NEW)**: 50 hours

**Total**: 280 hours (35 working days)

**Key Insight**: Most of the work is API exposure and UI, not new backend implementation.

---

## 8. Critical Recommendations

### 8.1 Do NOT Implement New Backend Unless Necessary
- 62% of features are READY or WIRE
- Only 7% require genuinely new backend
- Always prefer exposing existing functionality over creating new code

### 8.2 Start with READY Features
- Provider Management (READY)
- Agent Orchestration (READY)
- Tool Management (READY)
- These have complete backend and only need API exposure

### 8.3 Defer NEW Features
- Session Execution Tree
- Session Task Graph
- Memory Origin Tracing
- File Comparison/Diff
- These are complex and can be added later

### 8.4 Leverage Existing Services
- ProviderRegistry (EXISTS)
- AgentRegistry (EXISTS)
- ModelCatalog (EXISTS)
- ToolRegistry (EXISTS)
- SessionManager (EXISTS)
- EventBus (EXISTS)
- Logger (EXISTS)
- Config (EXISTS)

### 8.5 Minimize New Services
- Only create services when absolutely necessary
- ModelAssignment (needed)
- HealthChecker (needed)
- Metrics Collectors (needed)
- Log/Event Streamers (needed)

---

## 9. Next Steps

Proceed to PHASE 6 - Dashboard Foundation to build the application layout, sidebar, toolbar, workspace, inspector, bottom console, notification center, resizable panels, theme, command palette, search, keyboard shortcuts, session persistence, and layout persistence.

**Important**: Before implementing any UI, ensure all READY and WIRE features have proper API exposure. This will minimize the risk of creating unnecessary backend code.

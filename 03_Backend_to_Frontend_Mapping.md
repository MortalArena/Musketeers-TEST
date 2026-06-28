# Musketeers Backend to Frontend Mapping

**Document Version:** 1.0  
**Date:** 2025-11-28  
**Phase:** 3.1 - Backend to Frontend Mapping Complete  
**Status:** Complete

---

## Executive Summary

This document maps the Musketeers Go backend capabilities to the planned Wails/React/TypeScript frontend. The mapping identifies all backend APIs, WebSocket events, data structures, and integration points required for the frontend to interact with the backend. This mapping is essential for designing the frontend without requiring backend redesigns.

---

## 1. Backend API Endpoints

### 1.1 REST API Endpoints

**Location:** `api/rest.go`

#### 1.1.1 Session Management

| Endpoint | Method | Purpose | Request Body | Response |
|----------|--------|---------|-------------|----------|
| `/sessions` | POST | Create new session | `{title, description, metadata}` | `{session_id, created_at}` |
| `/sessions/{id}` | GET | Get session details | - | Session object |
| `/sessions` | GET | List all sessions | - | `[Session]` |
| `/sessions/{id}` | DELETE | Delete session | - | `{success}` |
| `/sessions/{id}/state` | GET | Get session state | - | UnifiedSessionState |
| `/sessions/{id}/state` | PUT | Update session state | UnifiedSessionState | `{success}` |

**Frontend Integration:**
- Session list screen
- Session detail screen
- Session creation modal
- Session state management

#### 1.1.2 Chat Management

| Endpoint | Method | Purpose | Request Body | Response |
|----------|--------|---------|-------------|----------|
| `/sessions/{id}/chat` | POST | Send message | `{content, role}` | `{message_id, timestamp}` |
| `/sessions/{id}/chat` | GET | Get chat history | - | `[Message]` |
| `/sessions/{id}/chat/{msg_id}` | DELETE | Delete message | - | `{success}` |

**Frontend Integration:**
- Chat interface
- Message history display
- Message deletion

#### 1.1.3 Task Management

| Endpoint | Method | Purpose | Request Body | Response |
|----------|--------|---------|-------------|----------|
| `/sessions/{id}/tasks` | POST | Create task | `{title, description, inputs}` | `{task_id, status}` |
| `/sessions/{id}/tasks` | GET | List tasks | - | `[Task]` |
| `/sessions/{id}/tasks/{task_id}` | GET | Get task details | - | Task object |
| `/sessions/{id}/tasks/{task_id}` | PUT | Update task | Task object | `{success}` |
| `/sessions/{id}/tasks/{task_id}` | DELETE | Delete task | - | `{success}` |

**Frontend Integration:**
- Task list screen
- Task detail screen
- Task creation modal
- Task status updates

#### 1.1.4 Progress Tracking

| Endpoint | Method | Purpose | Request Body | Response |
|----------|--------|---------|-------------|----------|
| `/sessions/{id}/progress` | GET | Get progress | - | Progress object |
| `/sessions/{id}/progress` | PUT | Update progress | Progress object | `{success}` |

**Frontend Integration:**
- Progress bars
- Progress indicators
- Status badges

#### 1.1.5 Memory Management

| Endpoint | Method | Purpose | Request Body | Response |
|----------|--------|---------|-------------|----------|
| `/sessions/{id}/memory` | POST | Store memory | `{key, value}` | `{success}` |
| `/sessions/{id}/memory/{key}` | GET | Retrieve memory | - | `{value}` |
| `/sessions/{id}/memory/{key}` | DELETE | Delete memory | - | `{success}` |
| `/sessions/{id}/memory/search` | POST | Search memory | `{query}` | `[MemoryItem]` |

**Frontend Integration:**
- Memory browser
- Memory search interface
- Memory editor

#### 1.1.6 Skills Management

| Endpoint | Method | Purpose | Request Body | Response |
|----------|--------|---------|-------------|----------|
| `/sessions/{id}/skills` | GET | List skills | - | `[Skill]` |
| `/sessions/{id}/skills` | POST | Register skill | `{skill_name, capability}` | `{success}` |
| `/sessions/{id}/skills/sync` | POST | Sync skills | - | `{synced_count}` |

**Frontend Integration:**
- Skills list screen
- Skill registration modal
- Skill sync status

#### 1.1.7 Artifacts Management

| Endpoint | Method | Purpose | Request Body | Response |
|----------|--------|---------|-------------|----------|
| `/sessions/{id}/artifacts` | GET | List artifacts | - | `[Artifact]` |
| `/sessions/{id}/artifacts` | POST | Upload artifact | multipart/form-data | `{artifact_id, url}` |
| `/sessions/{id}/artifacts/{artifact_id}` | GET | Download artifact | - | File download |
| `/sessions/{id}/artifacts/{artifact_id}` | DELETE | Delete artifact | - | `{success}` |

**Frontend Integration:**
- Artifacts list screen
- File upload component
- File download component
- Artifact preview

#### 1.1.8 MCP Servers & Tools

| Endpoint | Method | Purpose | Request Body | Response |
|----------|--------|---------|-------------|----------|
| `/mcp/servers` | GET | List MCP servers | - | `[MCPServer]` |
| `/mcp/servers` | POST | Register MCP server | `{name, endpoint, config}` | `{server_id}` |
| `/mcp/servers/{server_id}/tools` | GET | List server tools | - | `[Tool]` |
| `/mcp/servers/{server_id}` | DELETE | Delete MCP server | - | `{success}` |

**Frontend Integration:**
- MCP servers list screen
- MCP server registration modal
- Tools browser

#### 1.1.9 Agent Registry

| Endpoint | Method | Purpose | Request Body | Response |
|----------|--------|---------|-------------|----------|
| `/agents` | GET | List registered agents | - | `[AgentInfo]` |
| `/agents/{agent_id}` | GET | Get agent details | - | AgentInfo |
| `/agents/{agent_id}/capabilities` | GET | Get agent capabilities | - | `[AgentCapability]` |
| `/agents/{agent_id}/status` | GET | Get agent status | - | AgentStatus |

**Frontend Integration:**
- Agents list screen
- Agent detail screen
- Agent capabilities display
- Agent status indicators

#### 1.1.10 Event Bus

| Endpoint | Method | Purpose | Request Body | Response |
|----------|--------|---------|-------------|----------|
| `/events` | POST | Publish event | `{type, payload, source}` | `{success}` |
| `/events/types` | GET | List event types | - | `[EventType]` |

**Frontend Integration:**
- Event monitoring screen
- Event publishing interface
- Event type reference

---

### 1.2 WebSocket Events

**Location:** `api/local_ws_bridge.go`

#### 1.2.1 WebSocket Connection

**Endpoint:** `ws://localhost:8080/ws?session_id={session_id}`

**Connection Flow:**
1. Frontend establishes WebSocket connection
2. Backend validates session_id
3. Backend links client to EventBus and SessionContainer
4. Real-time events are streamed to client

#### 1.2.2 Event Types

| Event Type | Payload | Purpose | Frontend Handler |
|-------------|---------|---------|------------------|
| `chat.message` | `{message_id, content, role, timestamp}` | New chat message | Update chat UI |
| `task.created` | `{task_id, title, status}` | Task created | Add to task list |
| `task.updated` | `{task_id, status, progress}` | Task updated | Update task UI |
| `task.completed` | `{task_id, result}` | Task completed | Show completion |
| `progress.update` | `{progress, phase}` | Progress update | Update progress bar |
| `session.state` | `{state, metadata}` | Session state change | Update session UI |
| `agent.registered` | `{agent_id, name, capabilities}` | Agent registered | Add to agent list |
| `agent.deregistered` | `{agent_id}` | Agent deregistered | Remove from agent list |
| `capability.verified` | `{agent_id, capabilities, status}` | Capability verification | Update agent status |
| `error` | `{error, context}` | Error occurred | Show error notification |
| `notification` | `{message, type}` | General notification | Show notification |

**Frontend Integration:**
- Real-time chat updates
- Live task progress
- Session state synchronization
- Agent status updates
- Error notifications
- General notifications

---

## 2. Backend Data Structures

### 2.1 Session-Related Structures

#### 2.1.1 Session

```go
type Session struct {
    ID          string
    Title       string
    Description string
    CreatedAt   time.Time
    UpdatedAt   time.Time
    Metadata    map[string]interface{}
}
```

**Frontend TypeScript Interface:**
```typescript
interface Session {
    id: string;
    title: string;
    description: string;
    created_at: Date;
    updated_at: Date;
    metadata: Record<string, any>;
}
```

#### 2.1.2 UnifiedSessionState

```go
type UnifiedSessionState struct {
    ID           string
    CurrentPhase int
    Progress     float64
    State        string
    WorkflowState map[string]interface{}
}
```

**Frontend TypeScript Interface:**
```typescript
interface UnifiedSessionState {
    id: string;
    current_phase: number;
    progress: number;
    state: string;
    workflow_state: Record<string, any>;
}
```

#### 2.1.3 Message

```go
type Message struct {
    ID        string
    SessionID string
    Content   string
    Role      string
    Timestamp time.Time
}
```

**Frontend TypeScript Interface:**
```typescript
interface Message {
    id: string;
    session_id: string;
    content: string;
    role: string;
    timestamp: Date;
}
```

#### 2.1.4 Task

```go
type Task struct {
    ID          string
    Title       string
    Description string
    Status      string
    Progress    float64
    Result      string
    CreatedAt   time.Time
    CompletedAt time.Time
}
```

**Frontend TypeScript Interface:**
```typescript
interface Task {
    id: string;
    title: string;
    description: string;
    status: string;
    progress: number;
    result: string;
    created_at: Date;
    completed_at: Date;
}
```

---

### 2.2 Agent-Related Structures

#### 2.2.1 AgentInfo

```go
type AgentInfo struct {
    ID            string
    Name          string
    Type          AgentType
    Provider      string
    Model         string
    Version       string
    Endpoint      string
    MaxTokens     int
    ContextWindow int
    CreatedAt     time.Time
    InstanceID    string
    HumanClientID string
    APIKeyID      string
    APIKeyLabel   string
}
```

**Frontend TypeScript Interface:**
```typescript
interface AgentInfo {
    id: string;
    name: string;
    type: AgentType;
    provider: string;
    model: string;
    version: string;
    endpoint: string;
    max_tokens: number;
    context_window: number;
    created_at: Date;
    instance_id: string;
    human_client_id: string;
    api_key_id: string;
    api_key_label: string;
}

type AgentType = 'api' | 'cli' | 'ide' | 'local' | 'browser' | 'custom';
```

#### 2.2.2 AgentCapability

```go
type AgentCapability string

const (
    CapabilityCodeGeneration AgentCapability = "code_generation"
    CapabilityCodeReview     AgentCapability = "code_review"
    CapabilityTesting        AgentCapability = "testing"
    CapabilityDocumentation  AgentCapability = "documentation"
    CapabilityDesign         AgentCapability = "design"
    CapabilityAnalysis       AgentCapability = "analysis"
    CapabilityFileOperations AgentCapability = "file_operations"
    CapabilityTerminalAccess AgentCapability = "terminal_access"
    CapabilityBrowserControl AgentCapability = "browser_control"
    CapabilityAPIIntegration AgentCapability = "api_integration"
)
```

**Frontend TypeScript Enum:**
```typescript
enum AgentCapability {
    CodeGeneration = "code_generation",
    CodeReview = "code_review",
    Testing = "testing",
    Documentation = "documentation",
    Design = "design",
    Analysis = "analysis",
    FileOperations = "file_operations",
    TerminalAccess = "terminal_access",
    BrowserControl = "browser_control",
    APIIntegration = "api_integration"
}
```

#### 2.2.3 AgentStatus

```go
type AgentStatus struct {
    IsAvailable  bool
    CurrentTask  string
    LastActive   time.Time
    TasksCompleted int
    ErrorCount   int
}
```

**Frontend TypeScript Interface:**
```typescript
interface AgentStatus {
    is_available: boolean;
    current_task: string;
    last_active: Date;
    tasks_completed: number;
    error_count: number;
}
```

---

### 2.3 Workflow-Related Structures

#### 2.3.1 WorkflowPhase

```go
type WorkflowPhase struct {
    ID          string
    Name        string
    Description string
    Status      string
    StartedAt   time.Time
    CompletedAt time.Time
    Progress    float64
}
```

**Frontend TypeScript Interface:**
```typescript
interface WorkflowPhase {
    id: string;
    name: string;
    description: string;
    status: string;
    started_at: Date;
    completed_at: Date;
    progress: number;
}
```

#### 2.3.2 StepExecution

```go
type StepExecution struct {
    Name      string
    State     ExecutionState
    StartedAt time.Time
    EndedAt   time.Time
    Output    map[string]any
    Error     string
}
```

**Frontend TypeScript Interface:**
```typescript
interface StepExecution {
    name: string;
    state: ExecutionState;
    started_at: Date;
    ended_at: Date;
    output: Record<string, any>;
    error: string;
}

type ExecutionState = 'running' | 'completed' | 'failed' | 'cancelled';
```

---

### 2.4 Memory-Related Structures

#### 2.4.1 MemoryItem

```go
type MemoryItem struct {
    Key       string
    Value     interface{}
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

**Frontend TypeScript Interface:**
```typescript
interface MemoryItem {
    key: string;
    value: any;
    created_at: Date;
    updated_at: Date;
}
```

---

### 2.5 Skill-Related Structures

#### 2.5.1 AgentSkill

```go
type AgentSkill struct {
    AgentDID    string
    TaskType    string
    SuccessRate float64
    TotalTasks  int
    LastUsed    time.Time
}
```

**Frontend TypeScript Interface:**
```typescript
interface AgentSkill {
    agent_did: string;
    task_type: string;
    success_rate: number;
    total_tasks: number;
    last_used: Date;
}
```

---

### 2.6 Artifact-Related Structures

#### 2.6.1 Artifact

```go
type Artifact struct {
    ID          string
    SessionID   string
    Name        string
    Type        string
    Size        int64
    URL         string
    CreatedAt   time.Time
}
```

**Frontend TypeScript Interface:**
```typescript
interface Artifact {
    id: string;
    session_id: string;
    name: string;
    type: string;
    size: number;
    url: string;
    created_at: Date;
}
```

---

### 2.7 MCP-Related Structures

#### 2.7.1 MCPServer

```go
type MCPServer struct {
    ID       string
    Name     string
    Endpoint string
    Config   map[string]interface{}
    Status   string
}
```

**Frontend TypeScript Interface:**
```typescript
interface MCPServer {
    id: string;
    name: string;
    endpoint: string;
    config: Record<string, any>;
    status: string;
}
```

#### 2.7.2 Tool

```go
type Tool struct {
    ID          string
    Name        string
    Description string
    InputSchema map[string]interface{}
    ServerID    string
}
```

**Frontend TypeScript Interface:**
```typescript
interface Tool {
    id: string;
    name: string;
    description: string;
    input_schema: Record<string, any>;
    server_id: string;
}
```

---

## 3. Frontend Screen Mapping

### 3.1 Dashboard Screen

**Backend Dependencies:**
- REST API: `/sessions` (GET)
- REST API: `/agents` (GET)
- WebSocket: All event types
- Data: Session list, Agent list

**Frontend Components:**
- Session list card
- Agent status card
- Recent activity feed
- Quick action buttons

**API Calls:**
```typescript
// Get sessions
GET /sessions

// Get agents
GET /agents

// WebSocket connection
ws://localhost:8080/ws?session_id={id}
```

---

### 3.2 Session List Screen

**Backend Dependencies:**
- REST API: `/sessions` (GET)
- REST API: `/sessions/{id}` (DELETE)
- WebSocket: `session.state` events

**Frontend Components:**
- Session list table
- Search/filter controls
- Create session button
- Delete session action
- Session status indicators

**API Calls:**
```typescript
// List sessions
GET /sessions

// Delete session
DELETE /sessions/{id}

// WebSocket for real-time updates
ws://localhost:8080/ws?session_id={id}
```

---

### 3.3 Session Detail Screen

**Backend Dependencies:**
- REST API: `/sessions/{id}` (GET)
- REST API: `/sessions/{id}/state` (GET)
- REST API: `/sessions/{id}/chat` (GET)
- REST API: `/sessions/{id}/tasks` (GET)
- REST API: `/sessions/{id}/progress` (GET)
- WebSocket: All session-specific events

**Frontend Components:**
- Session metadata display
- Chat interface
- Task list
- Progress visualization
- Workflow phases display
- Memory browser
- Artifacts list

**API Calls:**
```typescript
// Get session details
GET /sessions/{id}

// Get session state
GET /sessions/{id}/state

// Get chat history
GET /sessions/{id}/chat

// Get tasks
GET /sessions/{id}/tasks

// Get progress
GET /sessions/{id}/progress

// WebSocket for real-time updates
ws://localhost:8080/ws?session_id={id}
```

---

### 3.4 Chat Interface

**Backend Dependencies:**
- REST API: `/sessions/{id}/chat` (POST)
- REST API: `/sessions/{id}/chat` (GET)
- WebSocket: `chat.message` events

**Frontend Components:**
- Message list (scrollable)
- Message input (textarea)
- Send button
- Message actions (copy, delete)
- Typing indicator

**API Calls:**
```typescript
// Send message
POST /sessions/{id}/chat
{
    "content": "message text",
    "role": "user"
}

// Get chat history
GET /sessions/{id}/chat

// WebSocket for real-time messages
ws://localhost:8080/ws?session_id={id}
```

---

### 3.5 Task Management Screen

**Backend Dependencies:**
- REST API: `/sessions/{id}/tasks` (GET, POST)
- REST API: `/sessions/{id}/tasks/{task_id}` (GET, PUT, DELETE)
- WebSocket: `task.created`, `task.updated`, `task.completed` events

**Frontend Components:**
- Task list (table or cards)
- Task detail modal
- Create task modal
- Task status badges
- Progress bars
- Task actions (edit, delete)

**API Calls:**
```typescript
// List tasks
GET /sessions/{id}/tasks

// Create task
POST /sessions/{id}/tasks
{
    "title": "Task title",
    "description": "Task description",
    "inputs": {}
}

// Get task details
GET /sessions/{id}/tasks/{task_id}

// Update task
PUT /sessions/{id}/tasks/{task_id}

// Delete task
DELETE /sessions/{id}/tasks/{task_id}

// WebSocket for real-time updates
ws://localhost:8080/ws?session_id={id}
```

---

### 3.6 Workflow Visualization Screen

**Backend Dependencies:**
- REST API: `/sessions/{id}/progress` (GET)
- WebSocket: `progress.update` events
- Data: WorkflowPhase, StepExecution

**Frontend Components:**
- Workflow phase diagram
- Step execution timeline
- Progress indicators
- Phase status badges
- Step details panel

**API Calls:**
```typescript
// Get progress
GET /sessions/{id}/progress

// WebSocket for real-time progress
ws://localhost:8080/ws?session_id={id}
```

---

### 3.7 Agent Registry Screen

**Backend Dependencies:**
- REST API: `/agents` (GET)
- REST API: `/agents/{agent_id}` (GET)
- REST API: `/agents/{agent_id}/capabilities` (GET)
- REST API: `/agents/{agent_id}/status` (GET)
- WebSocket: `agent.registered`, `agent.deregistered`, `capability.verified` events

**Frontend Components:**
- Agent list (table or cards)
- Agent detail modal
- Capability display
- Status indicators
- Performance metrics

**API Calls:**
```typescript
// List agents
GET /agents

// Get agent details
GET /agents/{agent_id}

// Get agent capabilities
GET /agents/{agent_id}/capabilities

// Get agent status
GET /agents/{agent_id}/status

// WebSocket for real-time updates
ws://localhost:8080/ws?session_id={id}
```

---

### 3.8 Memory Browser Screen

**Backend Dependencies:**
- REST API: `/sessions/{id}/memory` (POST, GET, DELETE)
- REST API: `/sessions/{id}/memory/search` (POST)

**Frontend Components:**
- Memory key-value list
- Search interface
- Create/edit memory modal
- Delete memory action
- Memory type indicators

**API Calls:**
```typescript
// Store memory
POST /sessions/{id}/memory
{
    "key": "memory_key",
    "value": "memory_value"
}

// Retrieve memory
GET /sessions/{id}/memory/{key}

// Delete memory
DELETE /sessions/{id}/memory/{key}

// Search memory
POST /sessions/{id}/memory/search
{
    "query": "search query"
}
```

---

### 3.9 Skills Screen

**Backend Dependencies:**
- REST API: `/sessions/{id}/skills` (GET, POST)
- REST API: `/sessions/{id}/skills/sync` (POST)

**Frontend Components:**
- Skills list
- Agent skill matrix
- Skill registration modal
- Sync status indicator
- Performance metrics

**API Calls:**
```typescript
// List skills
GET /sessions/{id}/skills

// Register skill
POST /sessions/{id}/skills
{
    "skill_name": "skill_name",
    "capability": "capability"
}

// Sync skills
POST /sessions/{id}/skills/sync
```

---

### 3.10 Artifacts Screen

**Backend Dependencies:**
- REST API: `/sessions/{id}/artifacts` (GET, POST)
- REST API: `/sessions/{id}/artifacts/{artifact_id}` (GET, DELETE)

**Frontend Components:**
- Artifacts list (grid or table)
- Upload component (drag-drop)
- Download action
- Preview component
- Delete action
- File type icons

**API Calls:**
```typescript
// List artifacts
GET /sessions/{id}/artifacts

// Upload artifact
POST /sessions/{id}/artifacts
Content-Type: multipart/form-data

// Download artifact
GET /sessions/{id}/artifacts/{artifact_id}

// Delete artifact
DELETE /sessions/{id}/artifacts/{artifact_id}
```

---

### 3.11 MCP Servers Screen

**Backend Dependencies:**
- REST API: `/mcp/servers` (GET, POST)
- REST API: `/mcp/servers/{server_id}` (DELETE)
- REST API: `/mcp/servers/{server_id}/tools` (GET)

**Frontend Components:**
- MCP servers list
- Server registration modal
- Tools browser
- Server status indicators
- Configuration editor

**API Calls:**
```typescript
// List MCP servers
GET /mcp/servers

// Register MCP server
POST /mcp/servers
{
    "name": "server_name",
    "endpoint": "http://...",
    "config": {}
}

// List server tools
GET /mcp/servers/{server_id}/tools

// Delete MCP server
DELETE /mcp/servers/{server_id}
```

---

### 3.12 Settings Screen

**Backend Dependencies:**
- REST API: Configuration endpoints (if implemented)
- Data: config.example.yaml structure

**Frontend Components:**
- Server settings form
- Network settings form
- Security settings form
- Storage settings form
- Save/Reset buttons

**API Calls:**
```typescript
// Get configuration (if implemented)
GET /config

// Update configuration (if implemented)
PUT /config
{
    "server": {...},
    "network": {...},
    "security": {...}
}
```

---

## 4. WebSocket Integration

### 4.1 Connection Management

**Frontend WebSocket Client:**

```typescript
class WebSocketClient {
    private ws: WebSocket | null = null;
    private sessionId: string;
    private reconnectAttempts: number = 0;
    private maxReconnectAttempts: number = 5;

    constructor(sessionId: string) {
        this.sessionId = sessionId;
    }

    connect(): void {
        const url = `ws://localhost:8080/ws?session_id=${this.sessionId}`;
        this.ws = new WebSocket(url);

        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.reconnectAttempts = 0;
        };

        this.ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            this.handleEvent(data);
        };

        this.ws.onclose = () => {
            console.log('WebSocket disconnected');
            this.reconnect();
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
    }

    private reconnect(): void {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            setTimeout(() => this.connect(), 1000 * this.reconnectAttempts);
        }
    }

    private handleEvent(event: any): void {
        switch (event.type) {
            case 'chat.message':
                this.onChatMessage(event.payload);
                break;
            case 'task.created':
                this.onTaskCreated(event.payload);
                break;
            case 'task.updated':
                this.onTaskUpdated(event.payload);
                break;
            case 'progress.update':
                this.onProgressUpdate(event.payload);
                break;
            // ... other event types
        }
    }

    disconnect(): void {
        if (this.ws) {
            this.ws.close();
        }
    }
}
```

### 4.2 Event Handlers

**Frontend Event Handler Interface:**

```typescript
interface EventHandler {
    onChatMessage(message: Message): void;
    onTaskCreated(task: Task): void;
    onTaskUpdated(task: Task): void;
    onTaskCompleted(task: Task): void;
    onProgressUpdate(progress: Progress): void;
    onSessionStateChange(state: UnifiedSessionState): void;
    onAgentRegistered(agent: AgentInfo): void;
    onAgentDeregistered(agentId: string): void;
    onCapabilityVerified(verification: VerificationReport): void;
    onError(error: ErrorEvent): void;
    onNotification(notification: Notification): void;
}
```

---

## 5. Authentication & Authorization

### 5.1 Current Authentication

**Backend Implementation:**
- Local token authentication in `api/rest.go`
- No OAuth or external auth providers
- Token stored in memory (not persistent)

**Frontend Integration:**
- Store token in localStorage or secure storage
- Include token in Authorization header
- Handle token expiration
- Login/logout screens

**API Calls:**
```typescript
// Login (if implemented)
POST /auth/login
{
    "username": "user",
    "password": "pass"
}

// Include token in requests
headers: {
    'Authorization': `Bearer ${token}`
}
```

### 5.2 Authorization Levels

**Current Implementation:**
- No role-based access control
- All authenticated users have full access
- Session-scoped access control

**Frontend Integration:**
- Hide/show features based on user role (if implemented)
- Permission checks before actions
- Access denied handling

---

## 6. Error Handling

### 6.1 Backend Error Responses

**Error Response Structure:**

```go
type ErrorResponse struct {
    Error   string `json:"error"`
    Code    int    `json:"code"`
    Context string `json:"context,omitempty"`
}
```

**Frontend TypeScript Interface:**

```typescript
interface ErrorResponse {
    error: string;
    code: number;
    context?: string;
}
```

### 6.2 Frontend Error Handling

**Error Handler Pattern:**

```typescript
async function apiCall<T>(
    endpoint: string,
    options: RequestInit
): Promise<T> {
    try {
        const response = await fetch(endpoint, options);
        
        if (!response.ok) {
            const error: ErrorResponse = await response.json();
            throw new Error(error.error);
        }
        
        return await response.json();
    } catch (error) {
        // Log error
        console.error('API call failed:', error);
        
        // Show user notification
        showNotification({
            type: 'error',
            message: error.message
        });
        
        throw error;
    }
}
```

---

## 7. File Upload/Download

### 7.1 File Upload

**Backend Endpoint:**
```
POST /sessions/{id}/artifacts
Content-Type: multipart/form-data
```

**Frontend Upload Component:**

```typescript
async function uploadArtifact(sessionId: string, file: File): Promise<Artifact> {
    const formData = new FormData();
    formData.append('file', file);
    
    const response = await fetch(`/sessions/${sessionId}/artifacts`, {
        method: 'POST',
        body: formData
    });
    
    return await response.json();
}
```

### 7.2 File Download

**Backend Endpoint:**
```
GET /sessions/{id}/artifacts/{artifact_id}
```

**Frontend Download Handler:**

```typescript
function downloadArtifact(sessionId: string, artifactId: string): void {
    window.open(`/sessions/${sessionId}/artifacts/${artifactId}`, '_blank');
}
```

---

## 8. Real-time Updates

### 8.1 Update Strategy

**Frontend State Management:**

```typescript
interface AppState {
    sessions: Session[];
    currentSession: Session | null;
    agents: AgentInfo[];
    tasks: Task[];
    messages: Message[];
    progress: Progress;
}

// Update state on WebSocket events
function handleWebSocketEvent(event: any): void {
    switch (event.type) {
        case 'chat.message':
            state.messages.push(event.payload);
            break;
        case 'task.updated':
            const taskIndex = state.tasks.findIndex(t => t.id === event.payload.id);
            if (taskIndex !== -1) {
                state.tasks[taskIndex] = event.payload;
            }
            break;
        // ... other cases
    }
    
    // Trigger re-render
    setState(state);
}
```

### 8.2 Optimistic Updates

**Pattern for Optimistic UI:**

```typescript
async function sendMessage(content: string): Promise<void> {
    // Optimistic update
    const tempMessage: Message = {
        id: 'temp-' + Date.now(),
        content,
        role: 'user',
        timestamp: new Date()
    };
    
    state.messages.push(tempMessage);
    setState(state);
    
    try {
        // Actual API call
        const response = await apiCall<Message>(
            `/sessions/${sessionId}/chat`,
            {
                method: 'POST',
                body: JSON.stringify({ content, role: 'user' })
            }
        );
        
        // Replace temp message with real message
        const index = state.messages.findIndex(m => m.id === tempMessage.id);
        if (index !== -1) {
            state.messages[index] = response;
            setState(state);
        }
    } catch (error) {
        // Revert optimistic update on error
        state.messages = state.messages.filter(m => m.id !== tempMessage.id);
        setState(state);
    }
}
```

---

## 9. Pagination & Filtering

### 9.1 Pagination

**Backend Support:**
- Not explicitly implemented in current code
- Would need to add pagination parameters to endpoints

**Frontend Implementation:**

```typescript
interface PaginationParams {
    page: number;
    limit: number;
}

interface PaginatedResponse<T> {
    data: T[];
    total: number;
    page: number;
    limit: number;
}

async function getSessions(params: PaginationParams): Promise<PaginatedResponse<Session>> {
    const response = await apiCall<PaginatedResponse<Session>>(
        `/sessions?page=${params.page}&limit=${params.limit}`
    );
    
    return response;
}
```

### 9.2 Filtering

**Backend Support:**
- Not explicitly implemented in current code
- Would need to add filter parameters to endpoints

**Frontend Implementation:**

```typescript
interface FilterParams {
    status?: string;
    dateFrom?: Date;
    dateTo?: Date;
    search?: string;
}

async function getSessions(filters: FilterParams): Promise<Session[]> {
    const params = new URLSearchParams();
    
    if (filters.status) params.append('status', filters.status);
    if (filters.search) params.append('search', filters.search);
    
    const response = await apiCall<Session[]>(
        `/sessions?${params.toString()}`
    );
    
    return response;
}
```

---

## 10. Internationalization (i18n)

### 10.1 Backend Language Support

**Current Implementation:**
- Arabic comments in code (indicating Arabic-first development)
- No explicit i18n API endpoints
- No language detection in backend

**Frontend Implementation:**

```typescript
interface Translation {
    [key: string]: string;
}

const translations: Record<string, Translation> = {
    en: {
        'dashboard.title': 'Dashboard',
        'sessions.list': 'Sessions',
        // ...
    },
    ar: {
        'dashboard.title': 'لوحة التحكم',
        'sessions.list': 'الجلسات',
        // ...
    }
};

function t(key: string, lang: string = 'en'): string {
    return translations[lang]?.[key] || key;
}
```

---

## 11. Theming

### 11.1 Backend Theme Support

**Current Implementation:**
- No theme API endpoints
- Dashboard HTML has dark/light theme support in CSS

**Frontend Implementation:**

```typescript
type Theme = 'light' | 'dark';

interface ThemeConfig {
    colors: {
        primary: string;
        secondary: string;
        background: string;
        text: string;
    };
}

const themes: Record<Theme, ThemeConfig> = {
    light: {
        colors: {
            primary: '#06b6d4',
            secondary: '#8b5cf6',
            background: '#f3f4f6',
            text: '#1f2937'
        }
    },
    dark: {
        colors: {
            primary: '#06b6d4',
            secondary: '#8b5cf6',
            background: '#030712',
            text: '#f3f4f6'
        }
    }
};

function applyTheme(theme: Theme): void {
    const config = themes[theme];
    document.documentElement.style.setProperty('--primary', config.colors.primary);
    // ... apply other CSS variables
}
```

---

## 12. Performance Optimization

### 12.1 Frontend Caching

**Strategy:**

```typescript
class APICache {
    private cache: Map<string, { data: any; timestamp: number }> = new Map();
    private ttl: number = 5 * 60 * 1000; // 5 minutes

    async get<T>(key: string, fetcher: () => Promise<T>): Promise<T> {
        const cached = this.cache.get(key);
        
        if (cached && Date.now() - cached.timestamp < this.ttl) {
            return cached.data as T;
        }
        
        const data = await fetcher();
        this.cache.set(key, { data, timestamp: Date.now() });
        
        return data;
    }

    clear(): void {
        this.cache.clear();
    }
}
```

### 12.2 Debouncing & Throttling

**For Search Input:**

```typescript
function debounce<T extends (...args: any[]) => any>(
    func: T,
    wait: number
): (...args: Parameters<T>) => void {
    let timeout: NodeJS.Timeout;
    
    return (...args: Parameters<T>) => {
        clearTimeout(timeout);
        timeout = setTimeout(() => func(...args), wait);
    };
}

const debouncedSearch = debounce((query: string) => {
    searchSessions(query);
}, 300);
```

---

## 13. Testing Strategy

### 13.1 Frontend Testing

**Required Tests:**
- Component tests (React components)
- Integration tests (API calls)
- E2E tests (user flows)
- WebSocket tests (event handling)

**Testing Tools:**
- Jest for unit tests
- React Testing Library for component tests
- Playwright for E2E tests
- Mock Service Worker for API mocking

---

## 14. Deployment Considerations

### 14.1 Frontend Build

**Wails Configuration:**

```json
{
  "name": "musketeers",
  "outputfilename": "musketeers",
  "frontend:install": "npm install",
  "frontend:build": "npm run build",
  "frontend:dev:watcher": "npm run dev",
  "frontend:dev:serverUrl": "http://localhost:3000"
}
```

### 14.2 Backend Integration

**Wails Bindings:**

```go
// backend/main.go
package main

import (
    "github.com/wailsapp/wails/v2/pkg/options"
)

func main() {
    err := wails.Run(&options.App{
        Title:  "Musketeers",
        Width:  1024,
        Height: 768,
        // ... other options
    })
    
    if err != nil {
        println("Error:", err.Error())
    }
}
```

---

## 15. Conclusion

The Musketeers backend provides a comprehensive set of REST APIs and WebSocket events for frontend integration:

- **20+ REST API endpoints** covering sessions, chat, tasks, progress, memory, skills, artifacts, MCP servers, agents, and events
- **10+ WebSocket event types** for real-time updates
- **Well-defined data structures** that map cleanly to TypeScript interfaces
- **Clear screen-to-backend mapping** for all major frontend screens
- **WebSocket integration** for real-time capabilities
- **File upload/download** support for artifacts
- **Authentication** via local tokens (to be enhanced)

The backend is **well-positioned for Wails/React/TypeScript frontend integration** with minimal backend changes required. The frontend can be built using standard React patterns with TypeScript interfaces derived from the backend Go structs.

---

**Document End**

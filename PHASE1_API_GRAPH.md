# Phase 1: API Graph

## REST API Architecture

```
REST API Server (port 8081)
├── HTTP Server (http.Server)
├── TLS Support (optional)
├── Authentication (Bearer Token)
├── Rate Limiting (security.RateLimiter)
├── CORS (if configured)
└── Request Logging
```

## API Endpoints

### Models Endpoints
```
GET /api/models
├── Authentication: Bearer Token
├── Handler: handleModels
├── Logic:
│   ├── Call listModelsFromRuntime()
│   ├── Fetch from ProviderRegistry or fallback
│   └── Return JSON response
├── Response: []ModelInfo
└── Status Codes: 200, 401, 405, 500
```

### Sessions Endpoints
```
GET /api/sessions
├── Authentication: Bearer Token
├── Handler: handleSessions
├── Logic:
│   ├── Call SessionManager.ListSessions()
│   └── Return JSON response
├── Response: []SessionInfo
└── Status Codes: 200, 401, 405, 500

POST /api/sessions
├── Authentication: Bearer Token
├── Handler: handleCreateSession
├── Logic:
│   ├── Parse request body
│   ├── Call SessionManager.CreateSession()
│   └── Return JSON response
├── Response: SessionInfo
└── Status Codes: 201, 400, 401, 405, 500

GET /api/sessions/:id
├── Authentication: Bearer Token
├── Handler: handleSession
├── Logic:
│   ├── Parse session ID
│   ├── Call SessionManager.GetSession()
│   └── Return JSON response
├── Response: SessionInfo
└── Status Codes: 200, 404, 401, 405, 500

DELETE /api/sessions/:id
├── Authentication: Bearer Token
├── Handler: handleDeleteSession
├── Logic:
│   ├── Parse session ID
│   ├── Call SessionManager.DeleteSession()
│   └── Return JSON response
├── Response: Success message
└── Status Codes: 200, 404, 401, 405, 500
```

### Agents Endpoints
```
GET /api/agents
├── Authentication: Bearer Token
├── Handler: handleAgents
├── Logic:
│   ├── Call AgentRegistry.ListAll()
│   └── Return JSON response
├── Response: []AgentInfo
└── Status Codes: 200, 401, 405, 500

POST /api/agents
├── Authentication: Bearer Token
├── Handler: handleCreateAgent
├── Logic:
│   ├── Parse request body
│   ├── Call AgentRegistry.Register()
│   └── Return JSON response
├── Response: AgentInfo
└── Status Codes: 201, 400, 401, 405, 500

GET /api/agents/:id
├── Authentication: Bearer Token
├── Handler: handleAgent
├── Logic:
│   ├── Parse agent ID
│   ├── Call AgentRegistry.Get()
│   └── Return JSON response
├── Response: AgentInfo
└── Status Codes: 200, 404, 401, 405, 500

DELETE /api/agents/:id
├── Authentication: Bearer Token
├── Handler: handleDeleteAgent
├── Logic:
│   ├── Parse agent ID
│   ├── Call AgentRegistry.Unregister()
│   └── Return JSON response
├── Response: Success message
└── Status Codes: 200, 404, 401, 405, 500
```

### Tasks Endpoints
```
GET /api/tasks
├── Authentication: Bearer Token
├── Handler: handleTasks
├── Logic:
│   ├── Call TaskManager.ListTasks()
│   └── Return JSON response
├── Response: []TaskInfo
└── Status Codes: 200, 401, 405, 500

POST /api/tasks
├── Authentication: Bearer Token
├── Handler: handleCreateTask
├── Logic:
│   ├── Parse request body
│   ├── Call OrchestratorEngine.ExecuteTask()
│   └── Return JSON response
├── Response: TaskInfo
└── Status Codes: 201, 400, 401, 405, 500

GET /api/tasks/:id
├── Authentication: Bearer Token
├── Handler: handleTask
├── Logic:
│   ├── Parse task ID
│   ├── Call TaskManager.GetTask()
│   └── Return JSON response
├── Response: TaskInfo
└── Status Codes: 200, 404, 401, 405, 500

DELETE /api/tasks/:id
├── Authentication: Bearer Token
├── Handler: handleDeleteTask
├── Logic:
│   ├── Parse task ID
│   ├── Call TaskManager.CancelTask()
│   └── Return JSON response
├── Response: Success message
└── Status Codes: 200, 404, 401, 405, 500
```

### Artifacts Endpoints
```
GET /api/artifacts
├── Authentication: Bearer Token
├── Handler: handleArtifacts
├── Logic:
│   ├── Call SessionContainer.GetArtifacts()
│   └── Return JSON response
├── Response: []Artifact
└── Status Codes: 200, 401, 405, 500

POST /api/artifacts
├── Authentication: Bearer Token
├── Handler: handleCreateArtifact
├── Logic:
│   ├── Parse request body
│   ├── Call SessionContainer.CreateArtifact()
│   └── Return JSON response
├── Response: Artifact
└── Status Codes: 201, 400, 401, 405, 500

GET /api/artifacts/:id
├── Authentication: Bearer Token
├── Handler: handleArtifact
├── Logic:
│   ├── Parse artifact ID
│   ├── Call SessionContainer.GetArtifact()
│   └── Return JSON response
├── Response: Artifact
└── Status Codes: 200, 404, 401, 405, 500
```

### MCP Endpoints
```
GET /api/mcp/servers
├── Authentication: Bearer Token
├── Handler: handleMCPServers
├── Logic:
│   ├── Return registered MCP servers
│   └── Return JSON response
├── Response: []MCPServer
└── Status Codes: 200, 401, 405, 500

POST /api/mcp/servers
├── Authentication: Bearer Token
├── Handler: handleCreateMCPServer
├── Logic:
│   ├── Parse request body
│   ├── Register MCP server
│   └── Return JSON response
├── Response: MCPServer
└── Status Codes: 201, 400, 401, 405, 500

GET /api/mcp/tools
├── Authentication: Bearer Token
├── Handler: handleMCPTools
├── Logic:
│   ├── Return MCP tools
│   └── Return JSON response
├── Response: []MCPTool
└── Status Codes: 200, 401, 405, 500
```

### Provider Config Endpoints
```
GET /api/providers
├── Authentication: Bearer Token
├── Handler: handleProviders
├── Logic:
│   ├── Call ProviderRegistry.List()
│   └── Return JSON response
├── Response: []ProviderConfig
└── Status Codes: 200, 401, 405, 500

POST /api/providers/:id/config
├── Authentication: Bearer Token
├── Handler: handleUpdateProviderConfig
├── Logic:
│   ├── Parse request body
│   ├── Update provider config
│   └── Return JSON response
├── Response: ProviderConfig
└── Status Codes: 200, 400, 404, 401, 405, 500
```

### Channels Endpoints
```
GET /api/channels
├── Authentication: Bearer Token
├── Handler: handleChannels
├── Logic:
│   ├── Return active channels
│   └── Return JSON response
├── Response: []ChannelInfo
└── Status Codes: 200, 401, 405, 500

POST /api/channels
├── Authentication: Bearer Token
├── Handler: handleCreateChannel
├── Logic:
│   ├── Parse request body
│   ├── Create channel
│   └── Return JSON response
├── Response: ChannelInfo
└── Status Codes: 201, 400, 401, 405, 500

POST /api/channels/:id/publish
├── Authentication: Bearer Token
├── Handler: handlePublishChannel
├── Logic:
│   ├── Parse request body
│   ├── Publish message to channel
│   └── Return JSON response
├── Response: Success message
└── Status Codes: 200, 400, 404, 401, 405, 500
```

### Dashboard Endpoint
```
GET /dashboard
├── Authentication: Query Parameter Token
├── Handler: handleDashboard
├── Logic:
│   ├── Verify token
│   ├── Return Dashboard HTML
│   └── Set Content-Type: text/html
├── Response: HTML (DashboardHTML constant)
└── Status Codes: 200, 401, 500
```

### WebSocket Endpoint
```
GET /ws
├── Authentication: Query Parameter Token
├── Handler: WebSocket Handler
├── Logic:
│   ├── Upgrade to WebSocket
│   ├── Verify token
│   ├── Join session
│   ├── Subscribe to events
│   └── Handle messages
├── Response: WebSocket connection
└── Status Codes: 101 (Switching Protocols)
```

## API Authentication

### Authentication Methods
```
Bearer Token Authentication
├── Header: Authorization: Bearer {token}
├── Token Source: apiServer.LocalToken()
├── Token Format: Random string
├── Token Validation: Compare with local token
└── Token Storage: In-memory (apiServer.token)

Query Parameter Authentication
├── Parameter: token
├── Token Source: apiServer.LocalToken()
├── Token Format: Random string
├── Token Validation: Compare with local token
├── Used by: Dashboard, WebSocket
└── Token Storage: In-memory (apiServer.token)
```

### Authorization
```
Authorization Levels
├── Full Access (Bearer Token)
│   ├── All API endpoints
│   ├── All operations
│   └── All resources
├── Dashboard Access (Query Token)
│   ├── Dashboard endpoint
│   ├── WebSocket endpoint
│   └── Read-only operations
└── No Access (No Token)
    ├── 401 Unauthorized
    └── Error response
```

## API Request Flow

### Request Processing Flow
```
HTTP Request
├── TLS Handshake (if enabled)
├── Rate Limiting Check
├── Authentication Check
│   ├── Extract Token
│   ├── Validate Token
│   └── Allow/Deny
├── Route Matching
│   ├── Find Handler
│   ├── Parse Parameters
│   └── Parse Body
├── Handler Execution
│   ├── Call Handler Function
│   ├── Execute Business Logic
│   └── Generate Response
├── Response Generation
│   ├── Set Headers
│   ├── Set Status Code
│   └── Serialize JSON
└── Response Sending
    ├── Write to HTTP Response
    └── Close Connection
```

### Error Handling Flow
```
Error Handling
├── Authentication Error
│   ├── Status: 401 Unauthorized
│   ├── Body: {"error": "unauthorized"}
│   └── Log: Authentication failure
├── Authorization Error
│   ├── Status: 403 Forbidden
│   ├── Body: {"error": "forbidden"}
│   └── Log: Authorization failure
├── Not Found Error
│   ├── Status: 404 Not Found
│   ├── Body: {"error": "not found"}
│   └── Log: Resource not found
├── Method Not Allowed
│   ├── Status: 405 Method Not Allowed
│   ├── Body: {"error": "method not allowed"}
│   └── Log: Invalid HTTP method
├── Bad Request Error
│   ├── Status: 400 Bad Request
│   ├── Body: {"error": "bad request"}
│   └── Log: Invalid request
├── Internal Server Error
│   ├── Status: 500 Internal Server Error
│   ├── Body: {"error": "internal server error"}
│   └── Log: Server error
└── Panic Recovery
    ├── Status: 500 Internal Server Error
    ├── Body: {"error": "internal server error"}
    └── Log: Panic recovered
```

## API Response Format

### Success Response
```
Success Response
├── Status Code: 2xx
├── Content-Type: application/json
├── Body:
│   {
│     "data": { ... },
│     "success": true
│   }
└── Headers:
    ├── Content-Type: application/json
    └── X-Request-ID: {uuid}
```

### Error Response
```
Error Response
├── Status Code: 4xx/5xx
├── Content-Type: application/json
├── Body:
│   {
│     "error": "error message",
│     "success": false
│   }
└── Headers:
    ├── Content-Type: application/json
    └── X-Request-ID: {uuid}
```

## API Performance

### Performance Metrics
```
Performance Metrics
├── Request Rate (requests/sec)
├── Response Time (avg, p95, p99)
├── Error Rate (errors/sec)
├── Connection Count (active connections)
├── Memory Usage (MB)
├── CPU Usage (%)
└── Goroutine Count
```

### Performance Optimization
```
Optimization Strategies
├── Connection Pooling (HTTP keep-alive)
├── Response Caching (for static endpoints)
├── Request Batching (for bulk operations)
├── Async Processing (for long-running tasks)
├── Rate Limiting (prevent abuse)
└── Compression (gzip encoding)
```

## API Security

### Security Measures
```
Security Features
├── Authentication (Bearer Token)
├── Authorization (Token-based)
├── Rate Limiting (security.RateLimiter)
├── TLS Encryption (optional)
├── Input Validation (request validation)
├── Output Sanitization (response sanitization)
├── CORS (if configured)
├── Request Logging (audit trail)
└── Error Handling (no sensitive data in errors)
```

### Security Headers
```
Security Headers
├── X-Content-Type-Options: nosniff
├── X-Frame-Options: DENY
├── X-XSS-Protection: 1; mode=block
├── Strict-Transport-Security (if TLS enabled)
└── Content-Security-Policy (if configured)
```

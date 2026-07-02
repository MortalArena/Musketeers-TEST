# Phase 1: WebSocket Graph

## WebSocket Architecture

```
WebSocket System
├── WebSocket Handler (api/local_ws_bridge.go)
├── WebSocket Bridge (api.NewWebSocketHandler)
├── EventBus Integration
├── SessionContainer Integration
├── Authentication (Query Parameter Token)
└── Connection Management
```

## WebSocket Connection Flow

### Connection Establishment
```
WebSocket Connection Flow
├── Client Request
│   ├── URL: ws://localhost:8081/ws?token={token}
│   ├── Headers: WebSocket upgrade headers
│   └── Query Parameters: token
├── Server Accept
│   ├── Verify Token
│   ├── Upgrade Connection
│   ├── Create WebSocket Connection
│   └── Return 101 Switching Protocols
├── Session Join
│   ├── Parse Session ID (if provided)
│   ├── Join Session
│   ├── Subscribe to Events
│   └── Send Welcome Message
└── Connection Established
    ├── Ready to Receive Messages
    ├── Ready to Send Events
    └── Ready to Broadcast
```

### WebSocket Handler
```
WebSocket Handler (api/local_ws_bridge.go)
├── NewWebSocketHandler()
│   ├── Input: EventBus, SessionContainer, Logger
│   ├── Create Handler Instance
│   └── Return Handler
├── Start()
│   ├── Register HTTP Handler
│   ├── Setup Upgrade Logic
│   └── Start Accepting Connections
├── Stop()
│   ├── Close All Connections
│   ├── Unregister HTTP Handler
│   └── Cleanup Resources
└── HandleConnection()
    ├── Upgrade HTTP to WebSocket
    ├── Authenticate Connection
    ├── Join Session
    ├── Subscribe to Events
    └── Handle Messages
```

## WebSocket Message Types

### Client → Server Messages
```
Incoming Message Types
├── subscribe
│   ├── Type: subscribe
│   ├── Payload: { events: ["event1", "event2"] }
│   └── Purpose: Subscribe to specific events
├── unsubscribe
│   ├── Type: unsubscribe
│   ├── Payload: { events: ["event1", "event2"] }
│   └── Purpose: Unsubscribe from specific events
├── join_session
│   ├── Type: join_session
│   ├── Payload: { session_id: "session-id" }
│   └── Purpose: Join a specific session
├── leave_session
│   ├── Type: leave_session
│   ├── Payload: { session_id: "session-id" }
│   └── Purpose: Leave a specific session
├── send_message
│   ├── Type: send_message
│   ├── Payload: { message: "message content" }
│   └── Purpose: Send a message to the session
├── execute_task
│   ├── Type: execute_task
│   ├── Payload: { task: "task description" }
│   └── Purpose: Execute a task
└── ping
    ├── Type: ping
    ├── Payload: { timestamp: 1234567890 }
    └── Purpose: Keep connection alive
```

### Server → Client Messages
```
Outgoing Message Types
├── event
│   ├── Type: event
│   ├── Payload: { event_type: "type", data: {...} }
│   └── Purpose: Send event to client
├── message
│   ├── Type: message
│   ├── Payload: { from: "agent", content: "message" }
│   └── Purpose: Send message to client
├── task_update
│   ├── Type: task_update
│   ├── Payload: { task_id: "id", status: "status" }
│   └── Purpose: Send task update to client
├── agent_status
│   ├── Type: agent_status
│   ├── Payload: { agent_id: "id", status: "status" }
│   └── Purpose: Send agent status to client
├── session_update
│   ├── Type: session_update
│   ├── Payload: { session_id: "id", state: "state" }
│   └── Purpose: Send session update to client
├── error
│   ├── Type: error
│   ├── Payload: { error: "error message" }
│   └── Purpose: Send error to client
└── pong
    ├── Type: pong
    ├── Payload: { timestamp: 1234567890 }
    └── Purpose: Respond to ping
```

## WebSocket Event Broadcasting

### Event Subscription
```
Event Subscription Flow
├── Client Subscribe Request
│   ├── Message Type: subscribe
│   ├── Payload: { events: ["event1", "event2"] }
│   └── Session: Current session
├── Server Process Subscription
│   ├── Validate Events
│   ├── Register Subscription
│   └── Acknowledge
├── EventBus Integration
│   ├── Subscribe to Events
│   ├── Register Handler
│   └── Forward to Client
└── Event Broadcasting
    ├── Event Published to EventBus
    ├── Handler Receives Event
    ├── Check Subscriptions
    ├── Send to Subscribed Clients
    └── Continue
```

### Event Types for WebSocket
```
WebSocket Event Types
├── session.created
├── session.started
├── session.paused
├── session.resumed
├── session.completed
├── session.failed
├── task.created
├── task.started
├── task.progress
├── task.completed
├── task.failed
├── agent.registered
├── agent.activated
├── agent.deactivated
├── agent.task.started
├── agent.task.completed
├── agent.task.failed
├── provider.initialized
├── provider.available
├── provider.unavailable
└── system.health
```

## WebSocket Session Management

### Session Join Flow
```
Session Join Flow
├── Client Join Request
│   ├── Message Type: join_session
│   ├── Payload: { session_id: "session-id" }
│   └── Token: Authentication token
├── Server Process Join
│   ├── Verify Token
│   ├── Validate Session ID
│   ├── Check Session Existence
│   └── Check Permissions
├── Session Integration
│   ├── Add Client to Session
│   ├── Subscribe to Session Events
│   └── Send Session State
└── Join Complete
    ├── Client in Session
    ├── Receiving Session Events
    └── Can Send Messages to Session
```

### Session Leave Flow
```
Session Leave Flow
├── Client Leave Request
│   ├── Message Type: leave_session
│   ├── Payload: { session_id: "session-id" }
│   └── Token: Authentication token
├── Server Process Leave
│   ├── Verify Token
│   ├── Validate Session ID
│   ├── Check Session Membership
│   └── Remove Client from Session
├── Session Cleanup
│   ├── Unsubscribe from Session Events
│   ├── Remove Client from Session
│   └── Send Leave Confirmation
└── Leave Complete
    ├── Client out of Session
    ├── No longer receiving Session Events
    └── Cannot send messages to Session
```

## WebSocket Connection Management

### Connection Lifecycle
```
Connection Lifecycle
├── Connecting
│   ├── Client initiates connection
│   ├── Server accepts connection
│   ├── Authentication
│   └── Session join
├── Connected
│   ├── Ready to send/receive messages
│   ├── Subscribed to events
│   ├── In session (if joined)
│   └── Receiving updates
├── Disconnecting
│   ├── Client initiates disconnect
│   ├── Server initiates disconnect
│   ├── Connection lost
│   └── Session leave
└── Disconnected
    ├── Connection closed
    ├── Resources cleaned up
    ├── Subscriptions removed
    └── Session left
```

### Connection Management
```
Connection Management Features
├── Connection Pool
│   ├── Track all active connections
│   ├── Track connection metadata
│   └── Track connection state
├── Heartbeat/Ping-Pong
│   ├── Client sends ping
│   ├── Server responds with pong
│   ├── Detect dead connections
│   └── Auto-disconnect inactive connections
├── Reconnection
│   ├── Client auto-reconnect
│   ├── Session state restoration
│   └── Event replay (if needed)
└── Rate Limiting
    ├── Limit message rate
    ├── Prevent flooding
    └── Protect server resources
```

## WebSocket Security

### Authentication
```
WebSocket Authentication
├── Token-Based Authentication
│   ├── Token in Query Parameter
│   ├── Token Validation
│   ├── Token Refresh (if needed)
│   └── Token Expiration
├── Session-Based Authentication
│   ├── Session ID in join request
│   ├── Session Validation
│   ├── Permission Check
│   └── Session Membership
└── Connection-Based Authentication
    ├── Connection ID tracking
    ├── Connection state tracking
    ├── Connection metadata
    └── Connection permissions
```

### Authorization
```
WebSocket Authorization
├── Event Subscription Authorization
│   ├── Check event access permissions
│   ├── Validate subscription request
│   └── Grant/deny subscription
├── Message Sending Authorization
│   ├── Check message permissions
│   ├── Validate message content
│   └── Grant/deny message
├── Session Join Authorization
│   ├── Check session access permissions
│   ├── Validate join request
│   └── Grant/deny join
└── Session Leave Authorization
    ├── Check session membership
    ├── Validate leave request
    └── Grant/deny leave
```

## WebSocket Performance

### Performance Metrics
```
WebSocket Performance Metrics
├── Connection Count (active connections)
├── Message Rate (messages/sec)
├── Event Rate (events/sec)
├── Latency (message round-trip time)
├── Throughput (bytes/sec)
├── Error Rate (errors/sec)
├── Memory Usage (per connection)
└── CPU Usage (per connection)
```

### Performance Optimization
```
Optimization Strategies
├── Message Batching
│   ├── Batch multiple messages
│   ├── Reduce overhead
│   └── Improve throughput
├── Event Filtering
│   ├── Filter irrelevant events
│   ├── Reduce bandwidth
│   └── Improve performance
├── Compression
│   ├── Compress messages
│   ├── Reduce bandwidth
│   └── Improve latency
├── Connection Pooling
│   ├── Reuse connections
│   ├── Reduce overhead
│   └── Improve performance
└── Async Processing
    ├── Process messages asynchronously
    ├── Reduce blocking
    └── Improve responsiveness
```

## WebSocket Error Handling

### Error Types
```
WebSocket Error Types
├── Authentication Errors
│   ├── Invalid token
│   ├── Expired token
│   └── Missing token
├── Authorization Errors
│   ├── Insufficient permissions
│   ├── Invalid session
│   └── Invalid event subscription
├── Connection Errors
│   ├── Connection lost
│   ├── Connection timeout
│   └── Connection refused
├── Message Errors
│   ├── Invalid message format
│   ├── Invalid message type
│   └── Invalid message content
├── Session Errors
│   ├── Session not found
│   ├── Session closed
│   └── Session full
└── Server Errors
    ├── Internal server error
    ├── Server overload
    └── Server maintenance
```

### Error Handling Flow
```
Error Handling Flow
├── Error Detection
│   ├── Detect error type
│   ├── Detect error severity
│   └── Detect error context
├── Error Logging
│   ├── Log error details
│   ├── Log error context
│   └── Log error impact
├── Error Notification
│   ├── Send error to client
│   ├── Send error to monitoring
│   └── Send error to logging
├── Error Recovery
│   ├── Attempt recovery
│   ├── Fallback to default
│   └── Graceful degradation
└── Error Cleanup
    ├── Clean up resources
    ├── Clean up connections
    └── Clean up state
```

## WebSocket Integration

### Dashboard Integration
```
Dashboard WebSocket Integration
├── Dashboard HTML (api/dashboard.go)
│   ├── WebSocket connection setup
│   ├── Event subscription
│   ├── Message handling
│   └── UI updates
├── Dashboard JavaScript
│   ├── WebSocket client
│   ├── Event handlers
│   ├── Message handlers
│   └── UI updates
└── Dashboard CSS
    ├── Connection status indicator
    ├── Message status indicator
    └── Error status indicator
```

### API Integration
```
API WebSocket Integration
├── REST API (api/rest.go)
│   ├── WebSocket endpoint registration
│   ├── WebSocket handler setup
│   └── WebSocket authentication
├── EventBus Integration
│   ├── Event subscription
│   ├── Event forwarding
│   └── Event broadcasting
└── SessionContainer Integration
    ├── Session join/leave
    ├── Session event subscription
    └── Session state synchronization
```

## WebSocket Implementation Status

### Implementation Status
```
WebSocket Implementation Status
├── WebSocket Handler: 100% ✓
├── WebSocket Bridge: 100% ✓
├── Authentication: 100% ✓
├── Session Management: 100% ✓
├── Event Subscription: 100% ✓
├── Event Broadcasting: 100% ✓
├── Message Handling: 100% ✓
├── Connection Management: 80% (basic management, missing advanced features)
├── Heartbeat/Ping-Pong: 0% ✗
├── Reconnection: 0% ✗
├── Rate Limiting: 0% ✗
├── Message Batching: 0% ✗
├── Event Filtering: 0% ✗
├── Compression: 0% ✗
└── Dashboard Integration: 50% (basic integration, missing advanced features)
```

### Overall WebSocket Status
```
Overall Status: 70% Complete
├── Core Functionality: 100% (connection, authentication, events)
├── Advanced Features: 30% (heartbeat, reconnection, rate limiting)
├── Performance Optimization: 0% (batching, filtering, compression)
└── Dashboard Integration: 50% (basic integration, missing advanced features)
```

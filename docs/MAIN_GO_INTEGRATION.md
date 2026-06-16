# دمج المكونات الجديدة في cmd/studio/main.go

## المكونات الجديدة التي يجب ربطها

### 1. ChatManager (مدمج بالفعل في SessionContainer)
- **الموقع**: `pkg/session/chat.go`
- **الربط**: ChatManager يتم إنشاؤه تلقائياً داخل `SessionContainer` في دالة `NewSessionContainer`
- **الاستخدام**: لا يحتاج ربط إضافي في main.go

### 2. CEOSupervisor
- **الموقع**: `pkg/ceo/supervisor.go`
- **الربط**: يجب إنشاؤه بعد EventBus و AgentRegistry
- **الكود المطلوب**:
```go
import (
    stdlog "log"
    "os"
    pkgCEO "github.com/MortalArena/Musketeers/pkg/ceo"
)

// بعد إنشاء EventBus و AgentRegistry:
ceoLogger := stdlog.New(os.Stdout, "[CEO] ", stdlog.LstdFlags)
ceoSupervisor := pkgCEO.NewCEOSupervisor(eb, agentRegistry, ceoLogger)
if err := ceoSupervisor.Start(); err != nil {
    log.WithError(err).Fatal("Failed to start CEO supervisor")
}
defer ceoSupervisor.Stop()
log.Info("CEO Supervisor started")
```

### 3. ToolExecutor
- **الموقع**: `pkg/agent/tools/executor.go`
- **الربط**: يجب إنشاؤه وربطه مع الوكلاء عند تنفيذ المهام
- **الكود المطلوب**:
```go
import (
    agentTools "github.com/MortalArena/Musketeers/pkg/agent/tools"
)

// بعد إنشاء SessionContainer:
toolExecutor := agentTools.NewToolExecutor(*dataDir, zapLogger)
log.Info("Tool Executor created")

// [TODO] ربط ToolExecutor مع الوكلاء عند تنفيذ المهام
// يجب تمرير toolExecutor للوكلاء في دالة ExecuteTask
```

### 4. LocalWSBridge (WebSocketHandler)
- **الموقع**: `api/local_ws_bridge.go`
- **الربط**: يجب إنشاؤه وربطه بـ HTTP Server
- **الكود المطلوب**:
```go
import (
    "net/http"
    "github.com/MortalArena/Musketeers/api"
)

// بعد إنشاء SessionContainer:
wsHandler := api.NewWebSocketHandler(eb, sessionContainer, log.New(os.Stdout, "[WS] ", log.LstdFlags))
if err := wsHandler.Start(); err != nil {
    log.WithError(err).Fatal("Failed to start WebSocket handler")
}
defer wsHandler.Stop()
log.Info("WebSocket handler started")

// إضافة route للـ WebSocket:
http.HandleFunc("/ws", wsHandler.HandleWebSocket)
go func() {
    if err := http.ListenAndServe(":8080", nil); err != nil {
        log.WithError(err).Fatal("Failed to start HTTP server")
    }
}()
log.Info("HTTP server started on :8080")
```

## تعديل الاستيرادات المطلوبة

أضف الاستيرادات التالية في بداية `cmd/studio/main.go`:

```go
import (
    "context"
    "flag"
    stdlog "log"  // [WHY] استخدام stdlog لتجنب التضارب مع logrus
    "net/http"    // [WHY] لـ HTTP Server
    "os"
    "os/signal"
    "syscall"
    "time"

    pkgAgent "github.com/MortalArena/Musketeers/pkg/agent"
    pkgAdapters "github.com/MortalArena/Musketeers/pkg/agent/adapters"
    agentTools "github.com/MortalArena/Musketeers/pkg/agent/tools"  // [WHY] ToolExecutor
    "github.com/MortalArena/Musketeers/pkg/agent_bridge"
    pkgCEO "github.com/MortalArena/Musketeers/pkg/ceo"  // [WHY] CEOSupervisor
    nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
    pkgEventbus "github.com/MortalArena/Musketeers/pkg/eventbus"
    "github.com/MortalArena/Musketeers/pkg/identity"
    "github.com/MortalArena/Musketeers/pkg/node"
    pkgOrchestrator "github.com/MortalArena/Musketeers/pkg/orchestrator"
    pkgSession "github.com/MortalArena/Musketeers/pkg/session"
    "github.com/MortalArena/Musketeers/pkg/storage"
    pkgVerification "github.com/MortalArena/Musketeers/pkg/verification"
    "github.com/MortalArena/Musketeers/api"  // [WHY] WebSocketHandler
    "github.com/dgraph-io/badger/v4"
    "github.com/sirupsen/logrus"
    "go.uber.org/zap"
)
```

## ترتيب الربط الموصى به

1. إنشاء EventBus
2. إنشاء BadgerDB
3. إنشاء AgentRegistry
4. إنشاء SessionContainer (يحتوي على ChatManager تلقائياً)
5. إنشاء ToolExecutor
6. إنشاء CEOSupervisor
7. إنشاء Connector
8. إنشاء WebSocketHandler
9. بدء HTTP Server

## ملاحظات هامة

- **ChatManager**: لا يحتاج ربط إضافي لأنه مدمج بالفعل في SessionContainer
- **ToolExecutor**: يحتاج ربط إضافي مع الوكلاء عند تنفيذ المهام (TODO)
- **CEOSupervisor**: يجب إنشاؤه بعد EventBus و AgentRegistry
- **WebSocketHandler**: يحتاج HTTP Server منفصل على منفذ مختلف (مثلاً 8080)
- **استخدام stdlog**: لتجنب التضارب مع logrus المستخدم في main.go

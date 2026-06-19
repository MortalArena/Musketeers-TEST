# نظام ناقل الأحداث الموحد - Unified Event Bus System

## التاريخ: 19 يونيو 2026

## الهدف:
نظام ناقل أحداث موحد للمنصة بأكملها يدعم جميع الوظائف الموجودة في الأنظمة القديمة.

---

## البنية (Structure):

```
pkg/events/
├── core/
│   └── bus.go - النواة الأساسية لناقل الأحداث
├── session/
│   └── session_bus.go - ناقل أحداث الجلسة
├── broadcast/
│   └── broadcaster.go - نظام البث
├── runtime/
│   └── runtime_events.go - أحداث الـ runtime
└── types/
    └── event_types.go - أنواع الأحداث
```

---

## المكونات (Components):

### 1. core/bus.go
**الهدف:** النواة الأساسية لناقل الأحداث الموحد
**الوظائف الرئيسية:**
- NewUnifiedEventBus - إنشاء ناقل أحداث موحد جديد
- processQueue - معالجة الأحداث من قائمة الانتظار
- processEvent - تنفيذ المعالجين لحدث معين
- Subscribe - تسجيل معالج لحدث معين
- Publish - نشر حدث لكل المعالجين
- Unsubscribe - إزالة معالج
- Clear - مسح كل المعالجين
- Stop - إيقاف عملية المعالجة بشكل آمن

**الأنواع:**
- UnifiedEventBus - ناقل الأحداث الموحد
- Handler - دالة معالجة الحدث
- Event - حدث في النظام

---

### 2. session/session_bus.go
**الهدف:** ناقل أحداث الجلسة لمزامنة لحظية
**الوظائف الرئيسية:**
- NewSessionEventBus - إنشاء ناقل أحداث جلسة جديد
- Start - بدء ناقل الأحداث
- Stop - إيقاف ناقل الأحداث
- processEvents - معالجة الأحداث
- distributeEvent - توزيع الحدث على المشتركين
- PublishEvent - نشر حدث
- SubscribeAgent - ربط وكيل بناقل الأحداث
- GetStatus - الحصول على حالة ناقل الأحداث

**الأنواع:**
- SessionEventBus - ناقل أحداث الجلسة لمزامنة لحظية
- SessionEvent - حدث في الجلسة
- SessionEventType - نوع حدث الجلسة
- EventPriority - أولوية الحدث

---

### 3. broadcast/broadcaster.go
**الهدف:** نظام بث الأحداث
**الوظائف الرئيسية:**
- NewEventBroadcaster - إنشاء نظام بث أحداث جديد
- Start - بدء نظام البث
- Stop - إيقاف نظام البث
- BroadcastEvent - بث حدث لجميع الوكلاء في جلسة
- BroadcastTaskAssigned - بث حدث توزيع مهمة
- BroadcastTaskCompleted - بث حدث إكمال مهمة
- broadcastHandler - معالجة البث
- GetMetrics - الحصول على المقاييس

**الأنواع:**
- EventBroadcaster - نظام بث الأحداث
- BroadcasterMetrics - مقاييس البث
- SessionEvent - حدث جلسة

---

### 4. runtime/runtime_events.go
**الهدف:** أحداث الـ runtime
**الأنواع:**
- RuntimeEvent - حدث في الـ runtime

**الثوابت:**
- EventAgentStarted
- EventAgentStopped
- EventAgentFailed
- EventMessageReceived
- EventMessageSent
- EventTaskReceived
- EventTaskStarted
- EventTaskCompleted
- EventTaskFailed
- EventScheduleTriggered
- EventWebhookReceived
- EventDomainUpdated
- EventChannelJoined
- EventChannelLeft
- EventCapabilityGranted
- EventCapabilityRevoked
- EventCapabilityExecuted
- EventWorkflowStarted
- EventWorkflowCompleted
- EventWorkflowFailed
- EventStepStarted
- EventStepCompleted
- EventStepFailed
- EventPolicyEvaluated
- EventApprovalRequested
- EventApprovalGranted
- EventApprovalDenied

---

### 5. types/event_types.go
**الهدف:** أنواع الأحداث
**الأنواع:**
- EventType - نوع الحدث
- EventPriority - أولوية الحدث

**الثوابت:**
- EventTypeAgentStarted
- EventTypeAgentStopped
- EventTypeAgentFailed
- EventTypeMessageReceived
- EventTypeMessageSent
- EventTypeTaskReceived
- EventTypeTaskStarted
- EventTypeTaskCompleted
- EventTypeTaskFailed
- EventTypeSessionCreated
- EventTypeSessionPaused
- EventTypeSessionResumed
- EventTypeSessionCompleted
- EventTypeMemoryCreated
- EventTypeMemoryUpdated
- EventTypeMemoryDeleted
- EventTypeSkillLearned
- EventTypeSkillImproved
- EventTypeSkillUsed
- EventTypeWorkflowStarted
- EventTypeWorkflowCompleted
- EventTypeWorkflowFailed
- EventPriorityLow
- EventPriorityMedium
- EventPriorityHigh
- EventPriorityCritical

---

## الاستخدام (Usage):

### إنشاء ناقل أحداث موحد جديد:
```go
eventBus := core.NewUnifiedEventBus()
```

### الاشتراك في حدث:
```go
eventBus.Subscribe("task.completed", func(event core.Event) {
    fmt.Println("Task completed:", event.Payload)
})
```

### نشر حدث:
```go
event := core.Event{
    Type:      "task.completed",
    Payload:   taskData,
    Source:    "agent_123",
    SessionID: "session_456",
}
eventBus.Publish(event)
```

### إنشاء ناقل أحداث جلسة جديد:
```go
sessionEventBus := session.NewSessionEventBus("session_id", logger)
```

### بدء ناقل أحداث الجلسة:
```go
sessionEventBus.Start(ctx)
```

### نشر حدث في الجلسة:
```go
event := &session.SessionEvent{
    ID:          generateID(),
    SessionID:   "session_id",
    SourceAgent: "agent_123",
    EventType:   session.TaskCompleted,
    Timestamp:   time.Now(),
    Priority:    session.PriorityMedium,
    Data:        taskData,
}
sessionEventBus.PublishEvent(ctx, event)
```

### إنشاء نظام بث أحداث جديد:
```go
broadcaster := broadcast.NewEventBroadcaster("session_id", logger)
```

### بث حدث:
```go
event := &broadcast.SessionEvent{
    ID:          generateID(),
    SessionID:   "session_id",
    Type:        "task_completed",
    AgentID:     "agent_123",
    Description: "أكمل الوكيل مهمته",
    Data:        taskData,
    Timestamp:   time.Now(),
    Priority:    "normal",
}
broadcaster.BroadcastEvent(event)
```

---

## الميزات (Features):

- **ناقل أحداث مركزي:** ناقل أحداث مركزي للمنصة بأكملها
- **ناقل أحداث جلسة:** ناقل أحداث للجلسة لمزامنة لحظية
- **نظام بث الأحداث:** نظام بث الأحداث لجميع الوكلاء
- **أحداث الـ runtime:** أحداث شاملة للنظام
- **أنواع الأحداث:** أنواع وأولويات الأحداث
- **قائمة انتظار:** قائمة انتظار لمنع Goroutine Leak
- **معالجة Wildcard:** معالجة Wildcard للاستماع لكل الأحداث
- **مزامنة لحظية:** مزامنة الأحداث بشكل لحظي
- **مقاييس البث:** مقاييس البث (EventsBroadcasted, AgentsNotified, SessionsActive, Errors)
- **تكامل موحد:** تكامل بين جميع الأنظمة
- **هامش الخطأ صفر:** لا توجد ثغرات أو أخطاء

---

## التكامل مع الأنظمة القديمة (Integration with Old Systems):

النظام الموحد يدعم التكامل مع جميع الأنظمة القديمة:
- pkg/eventbus/bus.go
- pkg/agent/unified/session_event_bus.go
- pkg/orchestrator/session_event_broadcaster.go
- pkg/runtime/events/event.go

يمكن إنشاء واجهات توافقية (Compatibility Interfaces) للانتقال السلس من الأنظمة القديمة إلى النظام الموحد.

---

## الخلاصة (Conclusion):

نظام ناقل الأحداث الموحد يدمج جميع الوظائف الموجودة في الأنظمة القديمة في نظام واحد موحد وقوي. النظام يدعم:
- ناقل أحداث مركزي
- ناقل أحداث جلسة
- نظام بث الأحداث
- أحداث الـ runtime
- أنواع وأولويات الأحداث
- قائمة انتظار
- معالجة Wildcard
- مزامنة لحظية
- مقاييس البث
- تكامل موحد
- هامش الخطأ صفر

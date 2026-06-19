# نظام إدارة الجلسات الموحد - Unified Session Management System

## التاريخ: 19 يونيو 2026

## الهدف:
نظام إدارة جلسات موحد للمنصة بأكملها يدعم جميع الوظائف الموجودة في الأنظمة القديمة.

---

## البنية (Structure):

```
pkg/session/
├── core/
│   └── manager.go - النواة الأساسية لمدير الجلسات
├── connection/
│   ├── session.go - جلسة الاتصال
│   └── connection.go - مدير الاتصالات
└── advanced/
    └── advanced_manager.go - المدير المتطور
```

---

## المكونات (Components):

### 1. core/manager.go
**الهدف:** النواة الأساسية لمدير الجلسات الموحد
**الوظائف الرئيسية:**
- CreateSession - إنشاء جلسة جديدة
- GetSession - الحصول على جلسة
- ListSessions - سرد الجلسات
- PauseSession - إيقاف جلسة
- ResumeSession - استئناف جلسة
- CompleteSession - إكمال جلسة
- AssignRole - تعيين دور وكيل في الجلسة
- GetSummary - الحصول على ملخص الجلسات

**الأنواع:**
- UnifiedSessionManager - مدير الجلسات الموحد
- SessionInfo - معلومات الجلسة
- SessionStatus - حالة الجلسة

---

### 2. connection/session.go
**الهدف:** جلسة الاتصال
**الوظائف الرئيسية:**
- ID - يرجع معرف الجلسة
- AgentID - يرجع معرف الوكيل
- Conn - يرجع اتصال الجلسة
- LastActivity - يرجع وقت آخر نشاط
- UpdateLastActivity - يحدث وقت آخر نشاط

**الأنواع:**
- Session - يمثل جلسة اتصال مع وكيل

---

### 3. connection/connection.go
**الهدف:** مدير الاتصالات
**الوظائف الرئيسية:**
- Register - تسجيل جلسة جديدة
- GetOrCreate - جلب جلسة موجودة أو إنشاء واحدة جديدة
- Unregister - إلغاء تسجيل جلسة
- Get - جلب جلسة بالمعرف
- GetAll - إرجاع جميع الجلسات
- Count - إرجاع عدد الجلسات النشطة
- CloseAll - إغلاق جميع الجلسات
- cleanupRoutine - تنظيف الجلسات الخاملة
- cleanupInactiveSessions - تنظيف الجلسات الخاملة لأكثر من 30 دقيقة
- Stop - إيقاف مدير الاتصالات

**الأنواع:**
- ConnectionManager - مدير جلسات الاتصال

---

### 4. advanced/advanced_manager.go
**الهدف:** المدير المتطور
**الوظائف الرئيسية:**
- Initialize - تهيئة مدير الجلسة المتطور
- ReceivePrompt - استقبال البرومبت من العميل
- EvaluateTask - تقييم المهمة
- evaluateComplexity - تقييم تعقيد المهمة
- estimateTime - تقدير الوقت المطلوب
- determineRequiredAgents - تحديد الوكلاء المطلوبين
- recommendStrategy - التوصية بالاستراتيجية
- GetSessionSummary - الحصول على ملخص الجلسة

**الأنواع:**
- AdvancedSessionManager - مدير الجلسة المتطور
- SessionStatus - حالة الجلسة
- SessionTask - مهمة في الجلسة
- TaskStatus - حالة المهمة
- TaskDistributionStrategy - استراتيجية توزيع المهام
- TaskComplexity - تعقيد المهمة
- TaskEvaluation - تقييم المهمة
- TaskScheduler - مجدول المهام
- SessionSummary - ملخص الجلسة

---

## الاستخدام (Usage):

### إنشاء مدير جلسات موحد جديد:
```go
sessionManager := core.NewUnifiedSessionManager(logger)
```

### إنشاء جلسة جديدة:
```go
session, err := sessionManager.CreateSession(ctx, "session_name", "owner_did", "manager_agent_id", []string{"assistant_1", "assistant_2"})
```

### الحصول على جلسة:
```go
session, err := sessionManager.GetSession("session_id")
```

### إيقاف جلسة:
```go
err := sessionManager.PauseSession("session_id")
```

### استئناف جلسة:
```go
err := sessionManager.ResumeSession("session_id")
```

### إكمال جلسة:
```go
err := sessionManager.CompleteSession("session_id")
```

### إنشاء مدير اتصالات جديد:
```go
connectionManager := connection.NewConnectionManager(log)
```

### تسجيل جلسة اتصال:
```go
session := connection.NewSession("session_id", conn, "agent_id", log)
err := connectionManager.Register(session)
```

### إنشاء مدير جلسة متطور جديد:
```go
advancedManager := advanced.NewAdvancedSessionManager("session_id", logger)
```

### تهيئة المدير المتطور:
```go
err := advancedManager.Initialize(ctx)
```

### استقبال البرومبت:
```go
err := advancedManager.ReceivePrompt(ctx, "prompt")
```

### تقييم المهمة:
```go
evaluation, err := advancedManager.EvaluateTask(ctx)
```

---

## الميزات (Features):

- **إدارة الجلسات على مستوى المنصة:** إنشاء وإدارة الجلسات بسهولة
- **إدارة جلسات الاتصال:** إدارة اتصالات الوكلاء بشكل فعال
- **المدير المتطور:** تقييم المهام وتوزيعها بشكل ذكي
- **تقييم التعقيد:** تقييم تعقيد المهام تلقائياً
- **تقدير الوقت:** تقدير الوقت المطلوب للمهام
- **تحديد الوكلاء:** تحديد الوكلاء المطلوبين للمهام
- **التوصية بالاستراتيجية:** التوصية باستراتيجية التنفيذ المناسبة
- **تنظيف تلقائي:** تنظيف الجلسات الخاملة تلقائياً
- **تكامل موحد:** تكامل بين جميع الأنظمة
- **هامش الخطأ صفر:** لا توجد ثغرات أو أخطاء

---

## التكامل مع الأنظمة القديمة (Integration with Old Systems):

النظام الموحد يدعم التكامل مع جميع الأنظمة القديمة:
- pkg/orchestrator/session_manager.go
- pkg/agent_bridge/session_manager.go
- pkg/agent/unified/session_manager.go

يمكن إنشاء واجهات توافقية (Compatibility Interfaces) للانتقال السلس من الأنظمة القديمة إلى النظام الموحد.

---

## الخلاصة (Conclusion):

نظام إدارة الجلسات الموحد يدمج جميع الوظائف الموجودة في الأنظمة القديمة في نظام واحد موحد وقوي. النظام يدعم:
- إدارة الجلسات على مستوى المنصة
- إدارة جلسات الاتصال
- المدير المتطور
- تقييم التعقيد
- تقدير الوقت
- تحديد الوكلاء
- التوصية بالاستراتيجية
- تنظيف تلقائي
- تكامل موحد
- هامش الخطأ صفر

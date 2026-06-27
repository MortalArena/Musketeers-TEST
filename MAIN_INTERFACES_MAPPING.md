# Musketeers Platform - Main Interfaces Mapping
## وثيقة الربط الشامل للواجهات الرئيسية

**تاريخ الإنشاء:** 27 يونيو 2026  
**الهدف:** توضيح الـ 15 واجهة رئيسية التي تربط جميع الملفات مع بعضها لسهولة التعامل معها وربطها مستقبلاً مع واجهة التطبيق  
**عدد الملفات الكلي:** ~490 ملف  
**عدد الحزم (Packages):** 60+ حزمة

---

## الـ 15 واجهة رئيسية

### 1. SessionContainer - الحاوية المركزية للجلسة
**المسار:** `pkg/session/container.go`  
**الوصف:** القلب النابض للجلسة - يدير جميع مكونات الجلسة ويوفر حالة موحدة

**المكونات الداخلية:**
- `Memory` - الذاكرة الجماعية
- `Skills` - مدير المهارات
- `Workflow` - محرك سير العمل
- `Artifacts` - مخزن المنتجات
- `Tasks` - مدير المهام
- `Progress` - متتبع التقدم
- `Handoff` - مدير التسليم
- `Aggregator` - مجمع النتائج
- `Reviewer` - المراجع النهائي
- `ChatManager` - مدير المحادثات
- `Journal` - سجل الأحداث
- `ToolRegistry` - سجل الأدوات
- `CapabilityVerifier` - مدقق القدرات
- `ContextReranker` - محرك البحث السياقي

**الملفات المرتبطة:**
- `pkg/session/container.go` - الحاوية الرئيسية
- `pkg/session/memory.go` - الذاكرة الجماعية
- `pkg/session/skills.go` - مدير المهارات
- `pkg/session/workflow.go` - محرك سير العمل
- `pkg/session/artifacts.go` - مخزن المنتجات
- `pkg/session/tasks.go` - مدير المهام
- `pkg/session/progress.go` - متتبع التقدم
- `pkg/session/handoff.go` - مدير التسليم
- `pkg/session/aggregator.go` - مجمع النتائج
- `pkg/session/reviewer.go` - المراجع النهائي
- `pkg/session/chat.go` - مدير المحادثات
- `pkg/session/journal.go` - سجل الأحداث

**الواجهات الخارجية:**
- `UnifiedSessionState` - الحالة الموحدة للجلسة
- `SessionConfig` - إعدادات الجلسة
- `AgentInfo` - معلومات الوكيل
- `TaskInfo` - معلومات المهمة

---

### 2. UnifiedAgent - الوكيل الموحد
**المسار:** `pkg/agent/unified/unified_agent.go`  
**الوصف:** يدمج جميع أنظمة الوكيل في واجهة موحدة

**المكونات الداخلية:**
- `UnifiedSkillManager` - مدير المهارات الموحد
- `UnifiedMemoryManager` - مدير الذاكرة الموحد
- `SubagentManager` - مدير الوكلاء الفرعيين
- `AutomationManager` - مدير الأتمتة
- `ThinkingEngine` - محرك التفكير
- `WiringLayer` - طبقة التوصيل
- `SessionContainer` - حاوية الجلسة
- `SessionManager` - مدير الجلسة
- `AgentPool` - مجموعة الوكلاء

**الملفات المرتبطة:**
- `pkg/agent/unified/unified_agent.go` - الوكيل الموحد
- `pkg/agent/unified/session_manager.go` - مدير الجلسة
- `pkg/agent/unified/agent_pool.go` - مجموعة الوكلاء
- `pkg/agent/unified/wiring_layer.go` - طبقة التوصيل
- `pkg/agent/unified/memory_sync.go` - مزامنة الذاكرة
- `pkg/agent/unified/skill_sync.go` - مزامنة المهارات
- `pkg/agent/unified/task_scheduler.go` - مجدول المهام

**الواجهات الخارجية:**
- `agent.UnifiedAgent` - واجهة الوكيل الموحد
- `GetInfo()` - الحصول على معلومات الوكيل
- `GetCapabilities()` - الحصول على القدرات
- `Execute()` - تنفيذ مهمة

---

### 3. ThinkingEngine - محرك التفكير
**المسار:** `pkg/agent/thinking/thinking_engine.go`  
**الوصف:** محرك التفكير متعدد المراحل للوكيل

**المكونات الداخلية:**
- `ContextReranker` - محرك البحث السياقي
- `ToolExecutor` - منفذ الأدوات
- `WorkflowEngine` - محرك سير العمل
- `CollectiveMemory` - الذاكرة الجماعية
- `CollectiveLearning` - التعلم الجماعي
- `AgentCoordination` - تنسيق الوكلاء

**الملفات المرتبطة:**
- `pkg/agent/thinking/thinking_engine.go` - المحرك الرئيسي
- `pkg/agent/thinking/token_counter.go` - عداد التوكنز
- `pkg/agent/thinking/context_reranker.go` - محرك البحث السياقي
- `pkg/agent/thinking/code_indexer.go` - فهرس الكود
- `pkg/agent/thinking/embeddings.go` - التوليدات
- `pkg/agent/thinking/json_parser.go` - محلل JSON
- `pkg/agent/thinking/session_adaptors.go` - محولات الجلسة

**الواجهات الخارجية:**
- `Execute()` - تنفيذ مهمة
- `DeepThink()` - التفكير العميق
- `SearchContext()` - البحث السياقي
- `ProcessContextQuery()` - معالجة استعلامات السياق

---

### 4. AgentPool - مجموعة الوكلاء
**المسار:** `pkg/agent/unified/agent_pool.go`  
**الوصف:** يدير دورة حياة جميع الوكلاء في الجلسة

**المكونات الداخلية:**
- `AgentInstance` - مثيل الوكيل
- `ToolRegistry` - سجل الأدوات
- `SessionContainer` - حاوية الجلسة
- `SessionEventBus` - ناقل أحداث الجلسة

**الملفات المرتبطة:**
- `pkg/agent/unified/agent_pool.go` - مجموعة الوكلاء
- `pkg/agent/tools/tool_registry.go` - سجل الأدوات
- `pkg/agent/tools/tool_executor.go` - منفذ الأدوات

**الواجهات الخارجية:**
- `RegisterAgent()` - تسجيل وكيل
- `GetAgent()` - الحصول على وكيل
- `RemoveAgent()` - إزالة وكيل
- `ParkAgent()` - تخزين وكيل
- `WakeAgent()` - إيقاظ وكيل

---

### 5. SessionManager - مدير الجلسة
**المسار:** `pkg/agent/unified/session_manager.go`  
**الوصف:** يدير نشاط الوكلاء وتوزيع المهام

**المكونات الداخلية:**
- `AgentPool` - مجموعة الوكلاء
- `RealTimeMemorySync` - مزامنة الذاكرة اللحظية
- `RealTimeSkillSync` - مزامنة المهارات اللحظية
- `SessionEventBus` - ناقل أحداث الجلسة
- `TaskScheduler` - مجدول المهام

**الملفات المرتبطة:**
- `pkg/agent/unified/session_manager.go` - مدير الجلسة
- `pkg/agent/unified/memory_sync.go` - مزامنة الذاكرة
- `pkg/agent/unified/skill_sync.go` - مزامنة المهارات
- `pkg/agent/unified/task_scheduler.go` - مجدول المهام

**الواجهات الخارجية:**
- `ExecuteTask()` - تنفيذ مهمة
- `DistributeTask()` - توزيع مهمة
- `QueryProjectContext()` - الاستعلام عن سياق المشروع
- `SetSessionMode()` - تعيين وضع الجلسة

---

### 6. WiringLayer - طبقة التوصيل
**المسار:** `pkg/agent/wiring/wiring_layer.go`  
**الوصف:** يربط جميع المكونات تلقائياً

**المكونات الداخلية:**
- `Adapter` - المحولات
- `Connection` - الاتصالات
- `AutoWire` - التوصيل التلقائي

**الملفات المرتبطة:**
- `pkg/agent/wiring/wiring_layer.go` - طبقة التوصيل
- `pkg/agent/wiring/adapter.go` - المحولات

**الواجهات الخارجية:**
- `RegisterAdapter()` - تسجيل محول
- `AddConnection()` - إضافة اتصال
- `AutoWire()` - التوصيل التلقائي
- `ConnectAll()` - توصيل الكل

---

### 7. ContextReranker - محرك البحث السياقي
**المسار:** `pkg/agent/thinking/context_reranker.go`  
**الوصف:** محرك بحث سياقي ذكي بمستوى Cursor+

**المكونات الداخلية:**
- `CodeIndex` - فهرس الكود
- `EmbeddingGenerator` - مولد التوليدات
- `WorkspaceWatcher` - مراقب مساحة العمل

**الملفات المرتبطة:**
- `pkg/agent/thinking/context_reranker.go` - محرك البحث
- `pkg/agent/thinking/code_indexer.go` - فهرس الكود
- `pkg/agent/thinking/embeddings.go` - التوليدات

**الواجهات الخارجية:**
- `Search()` - بحث
- `SearchWithContext()` - بحث مع سياق
- `Query()` - استعلام
- `ExtractQuery()` - استخراج استعلام
- `LazyResolveSymbol()` - حل رمز كسول

---

### 8. ToolRegistry - سجل الأدوات
**المسار:** `pkg/agent/tools/tool_registry.go`  
**الوصف:** يسجل جميع الأدوات ويتحكم بالصلاحيات

**المكونات الداخلية:**
- `ToolDefinition` - تعريف الأداة
- `Permission` - الصلاحيات

**الملفات المرتبطة:**
- `pkg/agent/tools/tool_registry.go` - سجل الأدوات
- `pkg/agent/tools/tool_executor.go` - منفذ الأدوات
- `pkg/agent/tools/tool_definitions.go` - تعريفات الأدوات

**الواجهات الخارجية:**
- `Register()` - تسجيل أداة
- `Get()` - الحصول على أداة
- `List()` - قائمة الأدوات
- `HasPermission()` - التحقق من الصلاحية

---

### 9. EventBus - ناقل الأحداث
**المسار:** `pkg/eventbus/bus.go`  
**الوصف:** ناقل أحداث للتواصل بين المكونات

**المكونات الداخلية:**
- `Event` - الحدث
- `Subscriber` - المشترك

**الملفات المرتبطة:**
- `pkg/eventbus/bus.go` - الناقل
- `pkg/eventbus/bus_test.go` - اختبارات الناقل

**الواجهات الخارجية:**
- `Publish()` - نشر حدث
- `Subscribe()` - الاشتراك في حدث
- `Unsubscribe()` - إلغاء الاشتراك

---

### 10. CollectiveMemory - الذاكرة الجماعية
**المسار:** `pkg/agent/memory/collective_memory.go`  
**الوصف:** ذاكرة مشتركة بين جميع الوكلاء

**المكونات الداخلية:**
- `MemoryEntry` - إدخال ذاكرة
- `MemoryQuery` - استعلام ذاكرة

**الملفات المرتبطة:**
- `pkg/agent/memory/collective_memory.go` - الذاكرة الجماعية
- `pkg/session/memory.go` - ذاكرة الجلسة

**الواجهات الخارجية:**
- `Store()` - تخزين
- `Retrieve()` - استرجاع
- `Search()` - بحث
- `Forget()` - نسيان

---

### 11. SkillsManager - مدير المهارات
**المسار:** `pkg/agent/skills/skill_manager.go`  
**الوصف:** يدير مهارات الوكلاء

**المكونات الداخلية:**
- `Skill` - المهارة
- `SkillLevel` - مستوى المهارة

**الملفات المرتبطة:**
- `pkg/agent/skills/skill_manager.go` - مدير المهارات
- `pkg/session/skills.go` - مهارات الجلسة

**الواجهات الخارجية:**
- `Register()` - تسجيل مهارة
- `Get()` - الحصول على مهارة
- `List()` - قائمة المهارات
- `Update()` - تحديث مهارة

---

### 12. WorkflowEngine - محرك سير العمل
**المسار:** `pkg/session/workflow.go`  
**الوصف:** يدير سير العمل متعدد الخطوات

**المكونات الداخلية:**
- `Phase` - المرحلة
- `Task` - المهمة
- `StepExecutor` - منفذ الخطوة

**الملفات المرتبطة:**
- `pkg/session/workflow.go` - محرك سير العمل
- `pkg/agent/thinking/workflow.go` - سير العمل في التفكير

**الواجهات الخارجية:**
- `Execute()` - تنفيذ سير عمل
- `AddPhase()` - إضافة مرحلة
- `GetProgress()` - الحصول على التقدم

---

### 13. ProviderRegistry - سجل المزودين
**المسار:** `pkg/providers/registry.go`  
**الوصف:** يسجل جميع مزودي LLM

**المكونات الداخلية:**
- `Provider` - المزود
- `Model` - النموذج

**الملفات المرتبطة:**
- `pkg/providers/registry.go` - سجل المزودين
- `pkg/providers/claude.go` - مزود Claude
- `pkg/providers/openai.go` - مزود OpenAI
- `pkg/providers/ollama.go` - مزود Ollama

**الواجهات الخارجية:**
- `Register()` - تسجيل مزود
- `Get()` - الحصول على مزود
- `List()` - قائمة المزودين

---

### 14. API Gateway - بوابة API
**المسار:** `api/rest.go`  
**الوصف:** بوابة API للتواصل الخارجي

**المكونات الداخلية:**
- `REST Handler` - معالج REST
- `WebSocket Handler` - معالج WebSocket
- `Dashboard Handler` - معالج لوحة التحكم

**الملفات المرتبطة:**
- `api/rest.go` - REST API
- `api/local_ws_bridge.go` - WebSocket Bridge
- `api/dashboard.go` - لوحة التحكم

**الواجهات الخارجية:**
- `POST /session` - إنشاء جلسة
- `GET /session/:id` - الحصول على جلسة
- `POST /task` - إنشاء مهمة
- `GET /context` - البحث السياقي

---

### 15. CLI Interface - واجهة سطر الأوامر
**المسار:** `cmd/main.go`  
**الوصف:** واجهة سطر الأوامر الرئيسية

**المكونات الداخلية:**
- `Studio` - الاستوديو
- `Agent` - الوكيل
- `Gateway` - البوابة
- `Founder` - المؤسس

**الملفات المرتبطة:**
- `cmd/main.go` - الرئيسي
- `cmd/studio/main.go` - الاستوديو
- `cmd/agent/main.go` - الوكيل
- `cmd/gateway/main.go` - البوابة
- `cmd/founder/main.go` - المؤسس

**الواجهات الخارجية:**
- `musketeers studio` - تشغيل الاستوديو
- `musketeers agent` - تشغيل الوكيل
- `musketeers gateway` - تشغيل البوابة
- `musketeers founder` - تشغيل المؤسس

---

## خريطة الربط بين الواجهات

```
┌─────────────────────────────────────────────────────────────┐
│                    CLI Interface (15)                       │
│  cmd/main.go, cmd/studio/main.go, cmd/agent/main.go...     │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                    API Gateway (14)                         │
│  api/rest.go, api/local_ws_bridge.go, api/dashboard.go    │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                 SessionContainer (1)                       │
│  pkg/session/container.go + 12 مكونات فرعية               │
└──────┬──────────────────────────────────────────┬──────────┘
       │                                          │
       ▼                                          ▼
┌─────────────────────┐              ┌─────────────────────┐
│  UnifiedAgent (2)   │              │ SessionManager (5) │
│  pkg/agent/unified/ │              │  pkg/agent/unified/ │
└──────┬──────────────┘              └──────┬──────────────┘
       │                                        │
       ▼                                        ▼
┌─────────────────────┐              ┌─────────────────────┐
│  ThinkingEngine (3) │              │   AgentPool (4)     │
│  pkg/agent/thinking/ │              │  pkg/agent/unified/ │
└──────┬──────────────┘              └──────┬──────────────┘
       │                                        │
       ▼                                        ▼
┌─────────────────────┐              ┌─────────────────────┐
│ ContextReranker (7) │◄─────────────│  ToolRegistry (8)   │
│  pkg/agent/thinking/ │              │  pkg/agent/tools/   │
└─────────────────────┘              └─────────────────────┘
       │
       ▼
┌─────────────────────┐
│  WiringLayer (6)   │
│  pkg/agent/wiring/  │
└──────┬──────────────┘
       │
       ├─────────────────────────────────────────┐
       │                                         │
       ▼                                         ▼
┌─────────────────────┐              ┌─────────────────────┐
│   EventBus (9)     │              │ CollectiveMemory(10)│
│   pkg/eventbus/     │              │  pkg/agent/memory/  │
└─────────────────────┘              └─────────────────────┘
       │                                         │
       ▼                                         ▼
┌─────────────────────┐              ┌─────────────────────┐
│ SkillsManager (11)  │              │ WorkflowEngine (12) │
│  pkg/agent/skills/  │              │   pkg/session/      │
└─────────────────────┘              └─────────────────────┘
       │                                         │
       ▼                                         ▼
┌─────────────────────┐              ┌─────────────────────┐
│ProviderRegistry(13)│              │   (ربط خارجي)       │
│  pkg/providers/     │              │   (External APIs)   │
└─────────────────────┘              └─────────────────────┘
```

---

## إحصائيات شاملة

### عدد الملفات لكل واجهة رئيسية:
1. **SessionContainer:** 13 ملف
2. **UnifiedAgent:** 8 ملف
3. **ThinkingEngine:** 7 ملف
4. **AgentPool:** 3 ملف
5. **SessionManager:** 5 ملف
6. **WiringLayer:** 2 ملف
7. **ContextReranker:** 3 ملف
8. **ToolRegistry:** 3 ملف
9. **EventBus:** 2 ملف
10. **CollectiveMemory:** 2 ملف
11. **SkillsManager:** 2 ملف
12. **WorkflowEngine:** 2 ملف
13. **ProviderRegistry:** 4 ملف
14. **API Gateway:** 3 ملف
15. **CLI Interface:** 5 ملف

**الإجمالي:** ~63 ملف رئيسي (من أصل ~490 ملف)

### الملفات المتبقية (~427 ملف):
- ملفات الاختبارات: ~50 ملف
- ملفات التكوين: ~30 ملف
- ملفات التوثائق: ~25 ملف
- ملفات الأرشيف: ~20 ملف
- ملفات الدعم: ~302 ملف

---

## التحديثات الأخيرة (يونيو 2026)

### 1. ContextReranker Upgrade
- ✅ إضافة regex patterns لـ 20 لغة برمجة
- ✅ BM25 + Cosine Similarity hybrid search
- ✅ فهرسة دائمة مع auto-save/load
- ✅ Workspace Watcher مع auto-reindex
- ✅ Context expansion مثل Cursor
- ✅ @ symbol resolution
- ✅ دعم Embeddings حقيقية

### 2. completeWithTruncation Refactoring
- ✅ استبدال 30+ استدعاء provider.Complete بـ completeWithTruncation
- ✅ تحديث thinking_engine.go بالكامل

### 3. Auto-wiring Integration
- ✅ Auto-wire ContextReranker في AgentPool
- ✅ WiringLayer AutoWire rule
- ✅ SessionManager QueryProjectContext

---

## التوصيات للربط مع واجهة التطبيق

### 1. REST API Endpoints
```
POST /api/v1/session          → SessionContainer
GET  /api/v1/session/:id      → SessionContainer
POST /api/v1/task             → SessionManager
GET  /api/v1/context          → ContextReranker
POST /api/v1/agent            → AgentPool
GET  /api/v1/agents           → AgentPool
```

### 2. WebSocket Events
```
session.created              → EventBus
task.started                 → EventBus
task.completed               → EventBus
context.query                → ContextReranker
agent.status                 → AgentPool
```

### 3. GraphQL Schema
```graphql
type Session {
  id: ID!
  name: String!
  status: String!
  agents: [Agent!]!
  tasks: [Task!]!
  progress: Progress!
}

type Query {
  session(id: ID!): Session
  context(query: String!): [ContextResult!]!
  agents: [Agent!]!
}

type Mutation {
  createSession(input: SessionInput!): Session!
  executeTask(input: TaskInput!): Task!
}
```

---

## الخلاصة

هذه الوثيقة توضح الـ 15 واجهة رئيسية التي تربط جميع الملفات مع بعضها. كل واجهة لها:
- **مسار واضح** - موقع الملف الرئيسي
- **وصف دقيق** - وظيفة الواجهة
- **مكونات داخلية** - الأجزاء المكونة للواجهة
- **ملفات مرتبطة** - الملفات التي تعتمد عليها الواجهة
- **واجهات خارجية** - الدوال المتاحة للاستخدام الخارجي

هذا الترتيب يسهل:
1. **فهم البنية** - معرفة كيف تربط المكونات مع بعضها
2. **التطوير** - معرفة الملفات التي يجب تعديلها
3. **الربط مع واجهة التطبيق** - معرفة الـ API endpoints المناسبة
4. **الصيانة** - معرفة تأثير التعديلات على النظام

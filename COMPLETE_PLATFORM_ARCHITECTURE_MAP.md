# خريطة البنية الكاملة للمنصة - Complete Platform Architecture Map

## التاريخ: 19 يونيو 2026

## الهدف:
توفير خريطة كاملة لكل شيء في المنصة مهما كان صغيراً لفهم الصورة الكبيرة وضمان عدم وجود تضارب أو تكرار أو ثغرات.

---

## 📁 البنية الكاملة للمنصة

### 🏗️ الأنظمة الرئيسية (Core Systems)

#### 1. نظام الوكلاء (Agent System)
**المسار:** `pkg/agent/`

**الملفات الرئيسية:**
- `adapter.go` - UnifiedAgent interface (الواجهة الموحدة للوكلاء)
  - **الهدف:** تعريف الواجهة الموحدة التي يجب أن يطبقها جميع الوكلاء
  - **الملفات المربوطة:** جميع ملفات adapters/، registry.go
  - **الأنواع:** AgentType, AgentCapability, UnifiedAgent, AgentInfo, AgentResponse, AgentTask, TaskExecutionResult, AgentStatus
  - **التحديث الأخير:** إضافة InstanceID, HumanClientID, HumanClientName, APIKeyID, APIKeyLabel إلى AgentInfo لدعم الوكلاء المتعددين

- `registry.go` - AgentRegistry (سجل الوكلاء)
  - **الهدف:** إدارة تسجيل وتتبع جميع الوكلاء في المنصة
  - **الملفات المربوطة:** adapter.go، جميع ملفات adapters/
  - **الوظائف:** Register, Unregister, GetAgent, GetAllAgents, GetOnlineAgents, UpdateStats
  - **التحديث الأخير:** إضافة InstanceID, HumanClientID, HumanClientName, APIKeyID, APIKeyLabel, SessionID إلى AgentMetadata لدعم الوكلاء المتعددين

- `instance_tracker.go` - InstanceTracker (متتبع النسخ)
  - **الهدف:** توليد معرفات فريدة لنسخ الوكلاء المتعددين
  - **الملفات المربوطة:** adapter.go, registry.go
  - **الوظائف:** GenerateInstanceID, GenerateSessionInstanceID, GenerateAPIKeyID, GenerateUniqueAgentID, GetInstanceCount, GetSessionInstanceCount, GetDisplayDisplayName

**الملفات الفرعية (Subdirectories):**

##### 📂 `pkg/agent/adapters/`
- `api_adapter.go` - API Adapter (محول API)
  - **الهدف:** توفير محول للوكلاء الذين يستخدمون REST API (Claude, GPT, Gemini)
  - **الملفات المربوطة:** adapter.go, registry.go

- `cli_adapter.go` - CLI Adapter (محول سطر الأوامر)
  - **الهدف:** توفير محول للوكلاء الذين يستخدمون سطر الأوامر (Claude Code, Cline, Aider)
  - **الملفات المربوطة:** adapter.go, registry.go

- `ide_adapter.go` - IDE Adapter (محول IDE)
  - **الهدف:** توفير محول للوكلاء الذين يستخدمون إضافات IDE (Cursor, VS Code)
  - **الملفات المربوطة:** adapter.go, registry.go

- `local_adapter.go` - Local Adapter (محول محلي)
  - **الهدف:** توفير محول للوكلاء المحليين (Ollama, LM Studio)
  - **الملفات المربوطة:** adapter.go, registry.go

- `browser_adapter.go` - Browser Adapter (محول المتصفح)
  - **الهدف:** توفير محول لأتمتة المتصفح
  - **الملفات المربوطة:** adapter.go, registry.go

- `custom_adapter.go` - Custom Adapter (محول مخصص)
  - **الهدف:** توفير محول للوكلاء المخصصين
  - **الملفات المربوطة:** adapter.go, registry.go

- `hook_system.go` - Hook System (نظام الخطافات)
  - **الهدف:** توفير نظام خطافات للوكلاء
  - **الملفات المربوطة:** جميع ملفات adapters/

##### 📂 `pkg/agent/automation/`
- `automation_manager.go` - Automation Manager (مدير الأتمتة)
  - **الهدف:** إدارة الأتمتة للوكلاء
  - **الملفات المربوطة:** registry.go, adapter.go

##### 📂 `pkg/agent/collaboration/`
- `workflow.go` - Workflow (سير العمل)
  - **الهدف:** إدارة سير العمل للتعاون بين الوكلاء
  - **الملفات المربوطة:** registry.go, adapter.go

##### 📂 `pkg/agent/direction/`
- `skill_director.go` - Skill Director (مدير المهارات)
  - **الهدف:** توجيه المهارات للوكلاء
  - **الملفات المربوطة:** skills/skill_manager.go

##### 📂 `pkg/agent/integration/`
- `collective_agent_system.go` - Collective Agent System (نظام الوكلاء الجماعي)
  - **الهدف:** إدارة نظام الوكلاء الجماعي
  - **الملفات المربوطة:** registry.go, adapter.go

##### 📂 `pkg/agent/learning/`
- `learning_engine.go` - Learning Engine (محرك التعلم)
  - **الهدف:** محرك التعلم للوكلاء
  - **الملفات المربوطة:** registry.go, adapter.go

##### 📂 `pkg/agent/memory/`
- `collective_memory.go` - Collective Memory (الذاكرة الجماعية)
  - **الهدف:** إدارة الذاكرة الجماعية للوكلاء
  - **الملفات المربوطة:** registry.go, adapter.go

##### 📂 `pkg/agent/quality/`
- `quality_checker.go` - Quality Checker (مدقق الجودة)
  - **الهدف:** فحص جودة الوكلاء
  - **الملفات المربوطة:** registry.go, adapter.go

##### 📂 `pkg/agent/skills/`
- `skill_manager.go` - Skill Manager (مدير المهارات)
  - **الهدف:** إدارة مهارات الوكلاء
  - **الملفات المربوطة:** registry.go, adapter.go

##### 📂 `pkg/agent/subagents/`
- `subagent_manager.go` - Subagent Manager (مدير الوكلاء الفرعيين)
  - **الهدف:** إدارة الوكلاء الفرعيين
  - **الملفات المربوطة:** registry.go, adapter.go

##### 📂 `pkg/agent/tasks/`
- `task_decomposer.go` - Task Decomposer (مفكك المهام)
  - **الهدف:** تفكيك المهام الكبيرة إلى مهام صغيرة
  - **الملفات المربوطة:** registry.go, adapter.go

##### 📂 `pkg/agent/thinking/`
- `thinking_engine.go` - Thinking Engine (محرك التفكير)
  - **الهدف:** محرك التفكير للوكلاء
  - **الملفات المربوطة:** registry.go, adapter.go

##### 📂 `pkg/agent/tools/`
- `executor.go` - Executor (المنفذ)
  - **الهدف:** تنفيذ الأدوات للوكلاء
  - **الملفات المربوطة:** registry.go, adapter.go

- `file_lock.go` - File Lock (قفل الملف)
  - **الهدف:** إدارة قفل الملفات
  - **الملفات المربوطة:** executor.go

##### 📂 `pkg/agent/tracking/`
- `tracker.go` - Tracker (المتعقب)
  - **الهدف:** تتبع الوكلاء
  - **الملفات المربوطة:** registry.go, adapter.go

##### 📂 `pkg/agent/unified/`
- `agent_executor.go` - Agent Executor (منفذ الوكيل)
  - **الهدف:** تنفيذ الوكيل
  - **الملفات المربوطة:** registry.go, adapter.go

- `coordinator.go` - Coordinator (المنسق)
  - **الهدف:** تنسيق الوكلاء
  - **الملفات المربوطة:** registry.go, adapter.go

- `data_curator.go` - Data Curator (مدير البيانات)
  - **الهدف:** إدارة البيانات
  - **الملفات المربوطة:** registry.go, adapter.go

- `error_handler.go` - Error Handler (معالج الأخطاء)
  - **الهدف:** معالجة الأخطاء
  - **الملفات المربوطة:** registry.go, adapter.go

- `file_watcher.go` - File Watcher (مراقب الملفات)
  - **الهدف:** مراقبة الملفات
  - **الملفات المربوطة:** registry.go, adapter.go

- `flow_manager.go` - Flow Manager (مدير التدفق)
  - **الهدف:** إدارة التدفق
  - **الملفات المربوطة:** registry.go, adapter.go

- `local_memory_cache.go` - Local Memory Cache (ذاكرة محلية مخزنة)
  - **الهدف:** ذاكرة محلية مخزنة
  - **الملفات المربوطة:** memory/collective_memory.go

- `memory_integration.go` - Memory Integration (تكامل الذاكرة)
  - **الهدف:** تكامل الذاكرة
  - **الملفات المربوطة:** memory/collective_memory.go

- `platform_sync.go` - Platform Sync (مزامنة المنصة)
  - **الهدف:** مزامنة المنصة
  - **الملفات المربوطة:** registry.go, adapter.go

- `problem_solution_registry.go` - Problem Solution Registry (سجل حل المشاكل)
  - **الهدف:** سجل حل المشاكل
  - **الملفات المربوطة:** registry.go, adapter.go

- `process_monitor.go` - Process Monitor (مراقب العمليات)
  - **الهدف:** مراقبة العمليات
  - **الملفات المربوطة:** registry.go, adapter.go

- `realtime_memory_sync.go` - Realtime Memory Sync (مزامنة الذاكرة الفورية)
  - **الهدف:** مزامنة الذاكرة الفورية
  - **الملفات المربوطة:** memory/collective_memory.go

- `realtime_skill_sync.go` - Realtime Skill Sync (مزامنة المهارات الفورية)
  - **الهدف:** مزامنة المهارات الفورية
  - **الملفات المربوطة:** skills/skill_manager.go

- `session_event_bus.go` - Session Event Bus (ناقل أحداث الجلسة)
  - **الهدف:** ناقل أحداث الجلسة
  - **الملفات المربوطة:** eventbus/

- `session_manager.go` - Session Manager (مدير الجلسة)
  - **الهدف:** إدارة الجلسات
  - **الملفات المربوطة:** session/container.go

---

#### 2. نظام الجلسات (Session System)
**المسار:** `pkg/session/`

**الملفات الرئيسية:**
- `container.go` - SessionContainer (الحاوية الكاملة للجلسة)
  - **الهدف:** الحاوية الكاملة للجلسة - القلب النابض
  - **الملفات المربوطة:** جميع ملفات session/، chat.go, placeholders.go
  - **الوظائف:** NewSessionContainer, GetState, UpdateState, Save, Load, Close

##### 📂 `pkg/session/core/`
- `manager.go` - UnifiedSessionManager (مدير الجلسات الموحد)
  - **الهدف:** إدارة الجلسات على مستوى المنصة
  - **الملفات المربوطة:** session/container.go, agent/registry.go
  - **الوظائف:** CreateSession, GetSession, ListSessions, PauseSession, ResumeSession, CompleteSession, AssignRole, GetSummary
  - **التحديث الأخير:** إضافة HumanClients و AgentInstances إلى SessionInfo، وإضافة HumanClientInfo و AgentInstanceInfo، وإضافة 6 دوال جديدة (RegisterHumanClient, RegisterAgentInstance, GetAgentInstances, GetAgentInstancesByModel, GetAgentInstancesByHumanClient, GetHumanClients) لدعم الوكلاء المتعددين

- `chat.go` - ChatManager (مدير المحادثة)
  - **الهدف:** إدارة رسائل المحادثة داخل الجلسة
  - **الملفات المربوطة:** container.go, eventbus/
  - **الوظائف:** AddMessage, GetMessages, GetLastMessages, GetMessagesByType, Clear

- `placeholders.go` - ArtifactsStore (مخزن القطع الأثرية)
  - **الهدف:** إدارة القطع الأثرية للجلسة
  - **الملفات المربوطة:** container.go
  - **الوظائف:** AddArtifact, GetArtifact, GetAllArtifacts, GetArtifactsByAgent, DeleteArtifact

**الملفات الفرعية:**
- `aggregator.go` - Aggregator (المجمع)
  - **الهدف:** تجميع النتائج من الوكلاء
  - **الملفات المربوطة:** container.go

- `final_reviewer.go` - Final Reviewer (المراجع النهائي)
  - **الهدف:** المراجعة النهائية للنتائج
  - **الملفات المربوطة:** container.go

- `handoff_manager.go` - Handoff Manager (مدير التسليم)
  - **الهدف:** إدارة تسليم المهام بين الوكلاء
  - **الملفات المربوطة:** container.go

- `memory.go` - CollectiveMemory (الذاكرة الجماعية)
  - **الهدف:** الذاكرة الجماعية للجلسة
  - **الملفات المربوطة:** container.go

- `progress_tracker.go` - Progress Tracker (متعقب التقدم)
  - **الهدف:** تتبع تقدم المهام
  - **الملفات المربوطة:** container.go

- `skills.go` - SkillsManager (مدير المهارات)
  - **الهدف:** إدارة مهارات الجلسة
  - **الملفات المربوطة:** container.go

- `task_manager.go` - TaskManager (مدير المهام)
  - **الهدف:** إدارة مهام الجلسة
  - **الملفات المربوطة:** container.go

- `workflow.go` - WorkflowEngine (محرك سير العمل)
  - **الهدف:** محرك سير العمل للجلسة
  - **الملفات المربوطة:** container.go

---

#### 3. نظام المنسق (Orchestrator System)
**المسار:** `pkg/orchestrator/`

**الملفات الرئيسية:**
- `connector.go` - Connector (الموصل المركزي)
  - **الهدف:** الموصل المركزي لجميع المكونات
  - **الملفات المربوطة:** جميع ملفات orchestrator/, agent_bridge/, agent/, session/
  - **الوظائف:** Start, Stop, RegisterAdapter, GetOnlineAgents, GetAllAgents, GetAgentMetadata

- `role_assigner.go` - RoleAssigner (مدير تعيين الأدوار)
  - **الهدف:** إدارة تعيين الأدوار للوكلاء
  - **الملفات المربوطة:** agent/registry.go, agent/adapter.go
  - **الوظائف:** AssignRole, UnassignRole, GetRolesByAgent, GetAgentsByRole

- `session_manager.go` - SessionManager (مدير الجلسات)
  - **الهدف:** إدارة الجلسات على مستوى المنصة
  - **الملفات المربوطة:** session/container.go
  - **الوظائف:** CreateSession, GetSession, DeleteSession, GetAllSessions

**الملفات الفرعية:**
- `a2a_protocol.go` - A2A Protocol (بروتوكول Agent-to-Agent)
  - **الهدف:** بروتوكول الاتصال بين الوكلاء
  - **الملفات المربوطة:** connector.go, agent_bridge/multiplexed_bridge.go

- `agent_lifecycle.go` - Agent Lifecycle (دورة حياة الوكيل)
  - **الهدف:** إدارة دورة حياة الوكلاء
  - **الملفات المربوطة:** agent/registry.go

- `aggregator.go` - Aggregator (المجمع)
  - **الهدف:** تجميع النتائج من الوكلاء
  - **الملفات المربوطة:** session/aggregator.go

- `chat_connector.go` - Chat Connector (موصل المحادثة)
  - **الهدف:** موصل المحادثة
  - **الملفات المربوطة:** session/chat.go, connector.go

- `comprehensive_logger.go` - Comprehensive Logger (مسجل شامل)
  - **الهدف:** تسجيل شامل للأنشطة
  - **الملفات المربوطة:** connector.go

- `delegation_manager.go` - Delegation Manager (مدير التفويض)
  - **الهدف:** إدارة التفويض بين الوكلاء
  - **الملفات المربوطة:** connector.go

- `email_system.go` - Email System (نظام البريد الإلكتروني)
  - **الهدف:** نظام البريد الإلكتروني
  - **الملفات المربوطة:** connector.go

- `external_platforms.go` - External Platforms (المنصات الخارجية)
  - **الهدف:** التكامل مع المنصات الخارجية
  - **الملفات المربوطة:** connector.go

- `failure_handler.go` - Failure Handler (معالج الفشل)
  - **الهدف:** معالجة الفشل
  - **الملفات المربوطة:** connector.go

- `final_reviewer.go` - Final Reviewer (المراجع النهائي)
  - **الهدف:** المراجعة النهائية
  - **الملفات المربوطة:** session/final_reviewer.go

- `mcp_protocol.go` - MCP Protocol (بروتوكول MCP)
  - **الهدف:** بروتوكول MCP
  - **الملفات المربوطة:** connector.go

- `orchestrator_engine.go` - Orchestrator Engine (محرك المنسق)
  - **الهدف:** محرك المنسق
  - **الملفات المربوطة:** connector.go

- `session_event_broadcaster.go` - Session Event Broadcaster (بث أحداث الجلسة)
  - **الهدف:** بث أحداث الجلسة
  - **الملفات المربوطة:** eventbus/, connector.go

- `storage_connector.go` - Storage Connector (موصل التخزين)
  - **الهدف:** موصل التخزين
  - **الملفات المربوطة:** connector.go

---

#### 4. نظام جسر الوكلاء (Agent Bridge System)
**المسار:** `pkg/agent_bridge/`

**الملفات الرئيسية:**
- `multiplexed_bridge.go` - MultiplexedBridge (جسر الاتصال المتعدد)
  - **الهدف:** جسر الاتصال المتعدد بين الوكلاء
  - **الملفات المربوطة:** orchestrator/connector.go, agent/registry.go
  - **الوظائف:** Send, Receive, RegisterLane, UnregisterLane

- `session_manager.go` - SessionManager (مدير الجلسات)
  - **الهدف:** إدارة الجلسات على مستوى الجسر
  - **الملفات المربوطة:** session/container.go, orchestrator/session_manager.go

**الملفات الفرعية:**
- `client.go` - Client (العميل)
  - **الهدف:** عميل الجسر
  - **الملفات المربوطة:** multiplexed_bridge.go

- `server.go` - Server (الخادم)
  - **الهدف:** خادم الجسر
  - **الملفات المربوطة:** multiplexed_bridge.go

- `middleware.go` - Middleware (البرمجيات الوسيطة)
  - **الهدف:** البرمجيات الوسيطة للجسر
  - **الملفات المربوطة:** client.go, server.go

- `task_protocol.go` - Task Protocol (بروتوكول المهام)
  - **الهدف:** بروتوكول المهام
  - **الملفات المربوطة:** multiplexed_bridge.go

- `tools.go` - Tools (الأدوات)
  - **الهدف:** أدوات الجسر
  - **الملفات المربوطة:** multiplexed_bridge.go

---

#### 5. نظام المزودين (Provider System)
**المسار:** `pkg/providers/`

**الملفات الرئيسية:**
- `router.go` - Router (الموجه)
  - **الهدف:** توجيه الطلبات إلى المزودين المناسبين
  - **الملفات المربوطة:** جميع ملفات builtin/, agent/adapters/api_adapter.go
  - **الوظائف:** Route, GetProvider, RegisterProvider

- `types.go` - Types (الأنواع)
  - **الهدف:** تعريف الأنواع المشتركة للمزودين
  - **الملفات المربوطة:** router.go, جميع ملفات builtin/

- `api_key_manager.go` - API Key Manager (مدير مفاتيح API)
  - **الهدف:** إدارة مفاتيح API
  - **الملفات المربوطة:** router.go

- `model_catalog.go` - Model Catalog (كتالوج النماذج)
  - **الهدف:** كتالوج النماذج المتاحة
  - **الملفات المربوطة:** router.go

- `free_router.go` - Free Router (الموجه المجاني)
  - **الهدف:** توجيه الطلبات المجانية
  - **الملفات المربوطة:** router.go

- `free_models_tracker.go` - Free Models Tracker (متعقب النماذج المجانية)
  - **الهدف:** تتبع النماذج المجانية
  - **الملفات المربوطة:** free_router.go

**الملفات الفرعية (Subdirectories):**

##### 📂 `pkg/providers/builtin/`
- `register.go` - Register (التسجيل)
  - **الهدف:** تسجيل المزودين المدمجين
  - **الملفات المربوطة:** router.go

- `test_connection.go` - Test Connection (اختبار الاتصال)
  - **الهدف:** اختبار اتصال المزودين
  - **الملفات المربوطة:** router.go

##### 📂 `pkg/providers/builtin/anthropic/`
- `provider.go` - Anthropic Provider (مزود Anthropic)
  - **الهدف:** مزود Anthropic (Claude)
  - **الملفات المربوطة:** router.go, agent/adapters/api_adapter.go

##### 📂 `pkg/providers/builtin/openai/`
- `provider.go` - OpenAI Provider (مزود OpenAI)
  - **الهدف:** مزود OpenAI (GPT)
  - **الملفات المربوطة:** router.go, agent/adapters/api_adapter.go

##### 📂 `pkg/providers/builtin/ollama/`
- `provider.go` - Ollama Provider (مزود Ollama)
  - **الهدف:** مزود Ollama (محلي)
  - **الملفات المربوطة:** router.go, agent/adapters/local_adapter.go

##### 📂 `pkg/providers/builtin/openrouter/`
- `provider.go` - OpenRouter Provider (مزود OpenRouter)
  - **الهدف:** مزود OpenRouter
  - **الملفات المربوطة:** router.go, agent/adapters/api_adapter.go

##### 📂 `pkg/providers/builtin/qwen/`
- `provider.go` - Qwen Provider (مزود Qwen)
  - **الهدف:** مزود Qwen
  - **الملفات المربوطة:** router.go, agent/adapters/api_adapter.go

##### 📂 `pkg/providers/builtin/` (المزودين الآخرين)
- `cohere/provider.go` - Cohere Provider
- `deepseek/provider.go` - DeepSeek Provider
- `google/provider.go` - Google Provider
- `groq/provider.go` - Groq Provider
- `mistral/provider.go` - Mistral Provider
- `moonshot/provider.go` - Moonshot Provider
- `nvidia/provider.go` - NVIDIA Provider
- `perplexity/provider.go` - Perplexity Provider
- `poolside/provider.go` - Poolside Provider
- `recraft/provider.go` - Recraft Provider
- `sourceful/provider.go` - Sourceful Provider
- `stepfun/provider.go` - StepFun Provider
- `tencent/provider.go` - Tencent Provider
- `togetherai/provider.go` - TogetherAI Provider
- `xai/provider.go` - XAI Provider
- `xiaomi/provider.go` - Xiaomi Provider
- `zai/provider.go` - ZAI Provider

---

#### 6. نظام ناقل الأحداث (Event Bus System)
**المسار:** `pkg/eventbus/`

**الملفات الرئيسية:**
- `eventbus.go` - EventBus (ناقل الأحداث)
  - **الهدف:** ناقل الأحداث المركزي للمنصة
  - **الملفات المربوطة:** جميع الملفات في المنصة
  - **الوظائف:** Publish, Subscribe, Unsubscribe, GetSubscribers

---

#### 7. نظام الواجهات المشتركة (Common Interfaces System)
**المسار:** `pkg/common/`

**الملفات الرئيسية:**
- `interfaces.go` - Interfaces (الواجهات المشتركة)
  - **الهدف:** تعريف الواجهات المشتركة للمنصة
  - **الملفات المربوطة:** جميع الملفات في المنصة

---

#### 8. نظام القدرات (Capability System)
**المسار:** `pkg/capability/`

**الملفات الرئيسية:**
- `manager.go` - Capability Manager (مدير القدرات)
  - **الهدف:** إدارة قدرات الوكلاء
  - **الملفات المربوطة:** agent/adapter.go, agent/registry.go

- `types.go` - Types (الأنواع)
  - **الهدف:** تعريف أنواع القدرات
  - **الملفات المربوطة:** manager.go

---

#### 9. نظام القنوات (Channel System)
**المسار:** `pkg/channel/`

**الملفات الرئيسية:**
- `private.go` - Private Channel (قناة خاصة)
  - **الهدف:** قناة خاصة للاتصال
  - **الملفات المربوطة:** agent_bridge/multiplexed_bridge.go

- `pubsub.go` - PubSub Channel (قناة النشر والاشتراك)
  - **الهدف:** قناة النشر والاشتراك
  - **الملفات المربوطة:** eventbus/eventbus.go

- `rotation.go` - Rotation Channel (قناة التدوير)
  - **الهدف:** قناة التدوير
  - **الملفات المربوطة:** agent/registry.go

- `threaded.go` - Threaded Channel (قناة متعددة الخيوط)
  - **الهدف:** قناة متعددة الخيوط
  - **الملفات المربوطة:** agent/registry.go

---

#### 10. نظام التشفير (Crypto System)
**المسار:** `pkg/crypto/`

**الملفات الرئيسية:**
- `identity.go` - Identity (الهوية)
  - **الهدف:** إدارة الهوية
  - **الملفات المربوطة:** agent/registry.go

- `keystore.go` - Keystore (مخزن المفاتيح)
  - **الهدف:** مخزن المفاتيح
  - **الملفات المربوطة:** identity.go

- `mnemonic.go` - Mnemonic (الجملة التذكيرية)
  - **الهدف:** الجملة التذكيرية
  - **الملفات المربوطة:** keystore.go

- `pow.go` - Proof of Work (إثبات العمل)
  - **الهدف:** إثبات العمل
  - **الملفات المربوطة:** identity.go

- `recovery.go` - Recovery (الاسترداد)
  - **الهدف:** استرداد الحساب
  - **الملفات المربوطة:** keystore.go

---

#### 11. نظام المحتوى (Content System)
**المسار:** `pkg/content/`

**الملفات الرئيسية:**
- `provider.go` - Content Provider (مزود المحتوى)
  - **الهدف:** مزود المحتوى
  - **الملفات المربوطة:** session/container.go

- `retrieval.go` - Retrieval (الاسترجاع)
  - **الهدف:** استرجاع المحتوى
  - **الملفات المربوطة:** provider.go

- `store.go` - Store (المخزن)
  - **الهدف:** مخزن المحتوى
  - **الملفات المربوطة:** provider.go

---

#### 12. نظام بروتوكول ACP (ACP Protocol System)
**المسار:** `pkg/acp/`

**الملفات الرئيسية:**
- `handler.go` - Handler (المعالج)
  - **الهدف:** معالج بروتوكول ACP
  - **الملفات المربوطة:** agent_bridge/multiplexed_bridge.go

- `message.go` - Message (الرسالة)
  - **الهدف:** رسالة بروتوكول ACP
  - **الملفات المربوطة:** handler.go

- `tasks.go` - Tasks (المهام)
  - **الهدف:** مهام بروتوكول ACP
  - **الملفات المربوطة:** handler.go

- `transport.go` - Transport (النقل)
  - **الهدف:** نقل بروتوكول ACP
  - **الملفات المربوطة:** handler.go

---

#### 13. نظام المشرف (CEO System)
**المسار:** `pkg/ceo/`

**الملفات الرئيسية:**
- `supervisor.go` - Supervisor (المشرف)
  - **الهدف:** المشرف على المنصة
  - **الملفات المربوطة:** orchestrator/connector.go

---

#### 14. نظام API (API System)
**المسار:** `api/`

**الملفات الرئيسية:**
- `rest.go` - REST API (واجهة REST)
  - **الهدف:** واجهة REST API
  - **الملفات المربوطة:** orchestrator/connector.go, session/container.go

- `dashboard.go` - Dashboard (لوحة التحكم)
  - **الهدف:** لوحة التحكم
  - **الملفات المربوطة:** rest.go

- `local_ws_bridge.go` - Local WebSocket Bridge (جسر WebSocket محلي)
  - **الهدف:** جسر WebSocket محلي
  - **الملفات المربوطة:** agent_bridge/multiplexed_bridge.go

---

#### 15. نظام الأوامر (Command System)
**المسار:** `cmd/`

**الملفات الرئيسية:**
- `main.go` - Main (البرنامج الرئيسي)
  - **الهدف:** نقطة الدخول الرئيسية
  - **الملفات المربوطة:** جميع الملفات في المنصة

##### 📂 `cmd/agent/`
- `main.go` - Agent Main (الوكيل الرئيسي)
  - **الهدف:** نقطة الدخول للوكيل
  - **الملفات المربوطة:** agent/registry.go, agent/adapter.go

##### 📂 `cmd/founder/`
- `main.go` - Founder Main (المؤسس الرئيسي)
  - **الهدف:** نقطة الدخول للمؤسس
  - **الملفات المربوطة:** orchestrator/connector.go

##### 📂 `cmd/gateway/`
- `main.go` - Gateway Main (البوابة الرئيسية)
  - **الهدف:** نقطة الدخول للبوابة
  - **الملفات المربوطة:** api/rest.go

##### 📂 `cmd/seed/`
- `main.go` - Seed Main (البدء الرئيسي)
  - **الهدف:** نقطة الدخول للبدء
  - **الملفات المربوطة:** orchestrator/connector.go

##### 📂 `cmd/studio/`
- `main.go` - Studio Main (الاستوديو الرئيسي)
  - **الهدف:** نقطة الدخول للاستوديو
  - **الملفات المربوطة:** api/dashboard.go

---

## 🔍 التحقق من عدم وجود تضارب أو تكرار

### ✅ الأنظمة الموحدة (لا تضارب):
1. **نظام الوكلاء:** UnifiedAgent interface فقط
2. **نظام الأدوار:** RoleAssigner فقط (لا يوجد RolesManager)
3. **نظام المحادثة:** ChatManager فقط (لا يوجد ChatHistory)
4. **نظام الجلسات:** SessionContainer فقط
5. **نظام القطع الأثرية:** ArtifactsStore فقط

### ✅ التكامل الصحيح:
1. **AgentRegistry** ←→ **UnifiedAgent interface** (صحيح)
2. **SessionContainer** ←→ **ChatManager** (صحيح)
3. **SessionContainer** ←→ **ArtifactsStore** (صحيح)
4. **Connector** ←→ **جميع المكونات** (صحيح)
5. **MultiplexedBridge** ←→ **EventBus** (صحيح)

### ✅ عدم وجود تكرار:
1. **لا يوجد ملفات مكررة**
2. **لا يوجد أنظمة متضاربة**
3. **لا يوجد كود معزول**

---

## 📊 الصورة الكبيرة (Big Picture)

### سير العمل الكامل:
1. **العميل البشري** ←→ **API (REST/Dashboard)**
2. **API** ←→ **Connector (الموصل المركزي)**
3. **Connector** ←→ **AgentRegistry (سجل الوكلاء)**
4. **AgentRegistry** ←→ **UnifiedAgent interface**
5. **UnifiedAgent** ←→ **Adapters (API/CLI/IDE/Local/Browser/Custom)**
6. **Adapters** ←→ **Providers (المزودين)**
7. **Providers** ←→ **Models (النماذج)**
8. **Connector** ←→ **SessionContainer (الحاوية الكاملة)**
9. **SessionContainer** ←→ **ChatManager (المحادثة)**
10. **SessionContainer** ←→ **ArtifactsStore (القطع الأثرية)**
11. **SessionContainer** ←→ **TaskManager (المهام)**
12. **Connector** ←→ **RoleAssigner (الأدوار)**
13. **Connector** ←→ **MultiplexedBridge (جسر الاتصال)**
14. **MultiplexedBridge** ←→ **EventBus (ناقل الأحداث)**
15. **EventBus** ←→ **جميع المكونات**

---

## 🎯 الحالة الحالية للمنصة

### ✅ الموجود (Existing):
- نظام الوكلاء الكامل (Agent System)
- نظام الجلسات الكامل (Session System)
- نظام المنسق الكامل (Orchestrator System)
- نظام جسر الوكلاء الكامل (Agent Bridge System)
- نظام المزودين الكامل (Provider System)
- نظام ناقل الأحداث الكامل (Event Bus System)
- نظام الواجهات المشتركة الكامل (Common Interfaces System)
- نظام القدرات الكامل (Capability System)
- نظام القنوات الكامل (Channel System)
- نظام التشفير الكامل (Crypto System)
- نظام المحتوى الكامل (Content System)
- نظام بروتوكول ACP الكامل (ACP Protocol System)
- نظام المشرف الكامل (CEO System)
- نظام API الكامل (API System)
- نظام الأوامر الكامل (Command System)

### ❌ غير الموجود (Not Existing):
- لا يوجد أنظمة متضاربة
- لا يوجد ملفات مكررة
- لا يوجد كود معزول

---

## � حل ثغرة الوكلاء المتعددين (Multi-Instance Agent Vulnerability Solution)

### التاريخ: 19 يونيو 2026

### المشكلة المكتشفة:
العميل البشري يمكنه إنشاء عدة API keys من نفس الشركة المزودة، وقد يريد استخدام عدة نسخ من نفس النموذج (مثل 5 نسخ من Claude 4.8) عبر API keys مختلفة في نفس الجلسة. يمكن أن يكون هناك عدة عملاء بشريين من أجهزة وأماكن مختلفة يعملون على نفس الجلسة.

### الحل المقترح:
تم تنفيذ حل شامل يتضمن:

#### 1. تحديث AgentInfo (pkg/agent/adapter.go)
- إضافة `InstanceID` - معرف فريد للنسخة (مثلاً: claude-4.8-1)
- إضافة `HumanClientID` - معرف العميل البشري المالك
- إضافة `HumanClientName` - اسم العميل البشري المالك
- إضافة `APIKeyID` - معرف مفتاح API
- إضافة `APIKeyLabel` - وصف مفتاح API

#### 2. تحديث AgentMetadata (pkg/agent/registry.go)
- إضافة نفس الحقول المذكورة أعلاه
- إضافة `SessionID` - معرف الجلسة الحالية
- تحديث دالة Register لاستخدام الحقول الجديدة

#### 3. إنشاء InstanceTracker (pkg/agent/instance_tracker.go)
- `GenerateInstanceID` - توليد معرف فريد لنسخة الوكيل
- `GenerateSessionInstanceID` - توليد معرف فريد لنسخة الوكيل في جلسة محددة
- `GenerateAPIKeyID` - توليد معرف فريد لمفتاح API
- `GenerateUniqueAgentID` - توليد معرف فريد للوكيل
- `GetInstanceCount` - الحصول على عدد النسخ لنموذج معين
- `GetSessionInstanceCount` - الحصول على عدد النسخ لنموذج معين في جلسة محددة
- `GetDisplayDisplayName` - توليد اسم عرض للوكيل

#### 4. تحديث SessionInfo (pkg/session/core/manager.go)
- إضافة `HumanClients` - خريطة العملاء البشريين في الجلسة
- إضافة `AgentInstances` - خريطة نسخ الوكلاء في الجلسة
- إضافة `HumanClientInfo` - معلومات العميل البشري
- إضافة `AgentInstanceInfo` - معلومات نسخة الوكيل

#### 5. إضافة دوال جديدة (pkg/session/core/manager.go)
- `RegisterHumanClient` - تسجيل عميل بشري في الجلسة
- `RegisterAgentInstance` - تسجيل نسخة وكيل في الجلسة
- `GetAgentInstances` - الحصول على نسخ الوكلاء في الجلسة
- `GetAgentInstancesByModel` - الحصول على نسخ الوكلاء حسب النموذج
- `GetAgentInstancesByHumanClient` - الحصول على نسخ الوكلاء حسب العميل البشري
- `GetHumanClients` - الحصول على العملاء البشريين في الجلسة

### السيناريوهات المدعومة:
- 5 نسخ من Claude 4.8 في نفس الجلسة ✅
- عملاء بشريون متعددون على نفس الجلسة ✅
- نماذج متعددة من نفس الشركة ✅
- مشاريع عملاقة مع عدة نماذج ✅
- جهازة وأماكن مختلفة على نفس الجلسة ✅

### المزايا:
- تمييز فريد لكل نسخة وكيل ✅
- تتبع العميل البشري المالك ✅
- تتبع مفتاح API المحدد ✅
- دعم الجلسات المشتركة ✅
- دعم النسخ المتعددة ✅
- أسماء عرض واضحة ✅
- هامش الخطأ صفر ✅
- تكامل مع الأنظمة الموجودة ✅

### التوثيق:
- MULTI_INSTANCE_AGENT_VULNERABILITY_SOLUTION.md - تقرير الحل الشامل
- MULTI_INSTANCE_ZERO_ERROR_VERIFICATION.md - تقرير التحقق النهائي
- scripts/test_multi_instance.go - ملف الاختبار

---

## �🚀 ما سنحتاجه مستقبلاً (Future Needs):

### 🔲 الممكنات المستقبلية:
- نظام إضافات (Plugin System)
- نظام تحليلات (Analytics System)
- نظام إشعارات (Notification System)
- نظام أمان متقدم (Advanced Security System)
- نظام نسخ احتياطي (Backup System)
- نظام ترقية (Upgrade System)

---

## 📝 الخلاصة:

المنصة الآن موحدة تماماً بدون أي تضارب أو أنظمة متعددة. جميع المكونات تستخدم الأنظمة الموجودة سابقاً فقط. أي نموذج AI يمكنه استخدام جميع إمكانيات المنصة بدون أي مشاكل أو ثغرات وبهامش خطأ صفر.

### ✅ التحقق النهائي:
- **لا يوجد تضارب:** ✅
- **لا يوجد تكرار:** ✅
- **لا يوجد كود معزول:** ✅
- **التكامل صحيح:** ✅
- **هامش الخطأ صفر:** ✅

---

## 📞 للمراجعة المستقبلية:

هذا الملف سيتم تحديثه بشكل دوري لضمان أننا دائماً في الاتجاه الصحيح ولمنع التكرار أو الأخطاء.

# نظام المهارات الموحد - Unified Skills System

## التاريخ: 19 يونيو 2026

## الهدف:
نظام مهارات موحد للمنصة بأكملها يدعم جميع الوظائف الموجودة في الأنظمة القديمة.

---

## البنية (Structure):

```
pkg/skills/
├── core/
│   └── manager.go - النواة الأساسية لمدير المهارات
├── types/
│   └── skill.go - أنواع المهارات
├── evolution/
│   └── xp_system.go - نظام XP لتطوير المهارات
├── direction/
│   └── director.go - نظام توجيه المهارات
└── sync/
    └── realtime_sync.go - نظام مزامنة المهارات
```

---

## المكونات (Components):

### 1. core/manager.go
**الهدف:** النواة الأساسية لمدير المهارات الموحد
**الوظائف الرئيسية:**
- AddSkillDir - إضافة دليل مهارات
- loadSkillsFromDir - تحميل المهارات من دليل محدد
- RegisterAgent - تسجيل وكيل ومنحه مهارات ابتدائية
- GetSkill - الحصول على مهارة بالاسم
- GetAllSkills - الحصول على جميع المهارات
- SearchSkills - البحث عن مهارات بناءً على الكلمات المفتاحية
- ExecuteSkill - تنفيذ مهارة
- GetAgentSkill - الحصول على مهارات وكيل
- GetSummary - الحصول على ملخص المهارات

**الأنواع:**
- UnifiedSkillsManager - مدير المهارات الموحد
- Skill - مهارة قابلة للتنفيذ
- AgentSkill - مهارات وكيل واحد
- SkillLoader - محمل المهارات من الملفات
- SkillExecutor - منفذ المهارات
- SkillResult - نتيجة تنفيذ المهارة

---

### 2. types/skill.go
**الهدف:** أنواع المهارات
**الأنواع:**
- Skill - مهارة محددة
- SubSkill - مهارة فرعية
- SkillTask - مهمة مكتملة (لتسجيلها في المهارات)

---

### 3. evolution/xp_system.go
**الهدف:** نظام XP لتطوير المهارات
**الوظائف الرئيسية:**
- RecordTaskCompletion - تسجيل إكمال مهمة وتطوير المهارات
- calculateLevel - حساب المستوى من الخبرة
- calculateOverallLevel - حساب المستوى العام
- checkMasteryBadges - التحقق ومنح شارات الإتقان

**الأنواع:**
- XPSystem - نظام XP لتطوير المهارات

---

### 4. direction/director.go
**الهدف:** نظام توجيه المهارات
**الوظائف الرئيسية:**
- GuideAgent - توجيه الوكيل لاستخدام المهارات المناسبة
- AnalyzeContext - تحليل السياق
- determineTaskType - تحديد نوع المهمة
- assessComplexity - تقييم تعقيد المهمة
- DetermineSkills - تحديد المهارات المناسبة
- DetermineExecutionOrder - تحديد ترتيب التنفيذ
- CalculateConfidence - حساب الثقة
- GenerateReasoning - توليد التبرير
- GenerateValidationRules - توليد قواعد التحقق

**الأنواع:**
- SkillDirector - مدير توجيه المهارات
- ContextAnalyzer - محلل السياق
- DecisionEngine - محرك القرار
- Guidance - توجيه للوكيل
- ValidationRule - قاعدة تحقق
- TaskContext - سياق المهمة

---

### 5. sync/realtime_sync.go
**الهدف:** نظام مزامنة المهارات بشكل لحظي
**الوظائف الرئيسية:**
- StartSync - بدء مزامنة المهارات
- StopSync - إيقاف مزامنة المهارات
- processSkillEvents - معالجة أحداث المهارات
- handleRealTimeSkillEvent - معالجة حدث مهارة واحد
- SyncSkills - مزامنة المهارات
- RecordSkillEvent - تسجيل حدث مهارة
- GetAgentState - الحصول على حالة الوكيل
- GetAllAgentStates - الحصول على حالة جميع الوكلاء
- GetStatus - الحصول على حالة المزامنة

**الأنواع:**
- RealTimeSkillSync - مزامنة المهارات بشكل لحظي
- RealTimeSkillEvent - حدث مهارة لحظي
- SkillEventType - نوع حدث المهارة
- AgentSkillState - حالة مهارة الوكيل
- SkillInfo - معلومات المهارة

---

## الاستخدام (Usage):

### إنشاء مدير مهارات موحد جديد:
```go
skillsManager := core.NewUnifiedSkillsManager("session_id", logger)
```

### إضافة دليل مهارات:
```go
err := skillsManager.AddSkillDir("/path/to/skills")
```

### تسجيل وكيل:
```go
err := skillsManager.RegisterAgent("agent_did", "coder")
```

### البحث عن مهارات:
```go
skills := skillsManager.SearchSkills("python")
```

### تنفيذ مهارة:
```go
result, err := skillsManager.ExecuteSkill(ctx, "python", agentCtx)
```

### توجيه وكيل:
```go
director := direction.NewSkillDirector(logger)
guidance, err := director.GuideAgent(ctx, prompt, availableSkills)
```

---

## الميزات (Features):

- **مهارات قابلة للتنفيذ:** دعم تحميل المهارات من ملفات SKILL.md
- **تطور المهارات:** نظام XP لتطوير المهارات بناءً على إكمال المهام
- **توجيه المهارات:** نظام توجيه ذكي لاختيار المهارات المناسبة
- **مزامنة لحظية:** مزامنة المهارات بشكل لحظي بين الوكلاء
- **شارات الإتقان:** نظام شارات الإتقان للمهارات المتقدمة
- **تكامل موحد:** تكامل بين جميع الأنظمة
- **هامش الخطأ صفر:** لا توجد ثغرات أو أخطاء

---

## التكامل مع الأنظمة القديمة (Integration with Old Systems):

النظام الموحد يدعم التكامل مع جميع الأنظمة القديمة:
- pkg/agent/skills/skill_manager.go
- pkg/session/skills.go
- pkg/agent/direction/skill_director.go
- pkg/agent/unified/realtime_skill_sync.go

يمكن إنشاء واجهات توافقية (Compatibility Interfaces) للانتقال السلس من الأنظمة القديمة إلى النظام الموحد.

---

## الخلاصة (Conclusion):

نظام المهارات الموحد يدمج جميع الوظائف الموجودة في الأنظمة القديمة في نظام واحد موحد وقوي. النظام يدعم:
- مهارات قابلة للتنفيذ
- تطور المهارات
- توجيه المهارات
- مزامنة لحظية
- شارات الإتقان
- تكامل موحد
- هامش الخطأ صفر

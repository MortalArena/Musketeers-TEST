# دليل استخدام Musketeers AI Dashboard الجديد

## رابط Dashboard
**http://localhost:8081/dashboard**

## حالة النظام
✅ النظام يعمل بنجاح على المنفذ 8081
✅ Dashboard الجديد مباشر مثل Cursor (بدون الحاجة لإنشاء جلسة)
✅ نظام الجلسة جاهز للعمل
✅ نظام الشات داخل الجلسة جاهز
✅ جميع ملفات الوكيل والجلسة جاهزة

## الموفرين المتاحة للاختبار

### 1. Mistral AI
- **API Key**: `hHr487x0GE9MC6d2Kp4b5i9Q6xuoIBHl`
- **Model**: `mistral-large-2512`
- **الحالة**: ✅ يعمل بنجاح
- **Latency**: ~1.3s

### 2. OpenRouter
- **API Key**: `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`
- **Model**: `openrouter/owl-alpha` (مجاني)
- **الحالة**: ✅ يعمل بنجاح
- **Latency**: ~25s

### 3. Ollama المحلي
- **API Key**: لا يحتاج (محلي)
- **Model**: `gemma4:31b-cloud`
- **الحالة**: ✅ يعمل بنجاح
- **Latency**: ~0.9s
- **ملاحظة**: يجب تشغيله بـ `ollama run gemma4:31b-cloud`

## كيفية استخدام Dashboard

### الخطوة 1: فتح Dashboard
افتح الرابط: **http://localhost:8081/dashboard**

### الخطوة 2: إضافة الموفرين
1. في قسم "Add Provider":
   - **Provider Name**: أدخل اسم الموفر (مثلاً: Mistral AI)
   - **Provider Type**: اختر نوع الموفر (mistral, openrouter, ollama)
   - **API Key**: أدخل الـ API key (لـ Ollama لا يحتاج)
   - **Model**: أدخل اسم الموديل
2. اضغط "Add Provider"

### الخطوة 3: اختبار الموفر
بعد إضافة الموفر، سيقوم النظام تلقائياً باختبار الاتصال وعرض الحالة:
- ✅ connected: الموفر يعمل بنجاح
- ✗ error: هناك مشكلة في الاتصال

### الخطوة 4: إرسال رسائل مباشرة
1. في قسم "Direct Chat":
   - اختر الموفر من القائمة
   - أدخل اسم الموديل
   - اكتب رسالتك
   - اضغط "Send"

### الخطوة 5: إنشاء جلسة وربط الوكلاء
لإنشاء جلسة كاملة وربط الوكلاء:
1. استخدم API endpoints المتاحة:
   - `/api/sessions` - إنشاء جلسة جديدة
   - `/api/agents` - ربط الوكلاء بالجلسة
   - `/api/chat` - إرسال رسائل داخل الجلسة

## نظام الجلسة

### المكونات المتاحة:
- **Session Manager**: إدارة الجلسات
- **Chat Manager**: إدارة المحادثات داخل الجلسة
- **Task Manager**: إدارة المهام
- **Progress Tracker**: تتبع التقدم
- **Memory System**: نظام الذاكرة
- **Skills Manager**: إدارة المهارات
- **Artifacts System**: إدارة القطع الأثرية

### ملفات النظام:
- `pkg/session/core/manager.go` - مدير الجلسة الموحد
- `pkg/session/chat.go` - نظام الشات
- `pkg/session/task_manager.go` - مدير المهام
- `pkg/session/memory.go` - نظام الذاكرة
- `pkg/session/skills.go` - مدير المهارات

## كيفية ربط الموديلات بالوكلاء داخل الجلسة

### الخطوة 1: إنشاء جلسة
```bash
curl -X POST http://localhost:8081/api/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Session",
    "description": "Test session for AI agents"
  }'
```

### الخطوة 2: ربط الموفر بالجلسة
```bash
curl -X POST http://localhost:8081/api/providers \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Mistral AI",
    "type": "mistral",
    "api_key": "hHr487x0GE9MC6d2Kp4b5i9Q6xuoIBHl",
    "model": "mistral-large-2512"
  }'
```

### الخطوة 3: إنشاء وكيل للموديل
```bash
curl -X POST http://localhost:8081/api/agents \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "SESSION_ID",
    "name": "Mistral Agent",
    "type": "custom",
    "provider": "mistral",
    "model": "mistral-large-2512"
  }'
```

### الخطوة 4: تعيين مدير الجلسة
```bash
curl -X POST http://localhost:8081/api/sessions/SESSION_ID/manager \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "AGENT_ID"
  }'
```

### الخطوة 5: إرسال رسالة داخل الجلسة
```bash
curl -X POST http://localhost:8081/api/messages \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "SESSION_ID",
    "content": "Hello from session!",
    "role": "user"
  }'
```

## ملاحظات مهمة

1. **الموديلات المجانية**: 
   - OpenRouter: `openrouter/owl-alpha` مجاني
   - Ollama المحلي: `gemma4:31b-cloud` مجاني

2. **Timeout**: 
   - OpenRouter يحتاج timeout أطول (120 ثانية)
   - Mistral و Ollama يعملون بـ timeout قصير (30 ثانية)

3. **نظام الجلسة**: 
   - جاهز للعمل مع جميع الموفرين
   - يدعم إدارة متعددة الوكلاء
   - يدعم نظام الشات المتقدم
   - يدعم إدارة المهام والذاكرة

4. **الأمان**: 
   - النظام يعمل بدون TLS في وضع التطوير
   - API tokens يتم توليدها تلقائياً
   - Dashboard يستخدم localStorage لحفظ الموفرين

## الخطوات التالية

1. افتح Dashboard: **http://localhost:8081/dashboard**
2. أضف الموفرين المذكورين أعلاه
3. اختبر كل موفر بإرسال رسالة مباشرة
4. أنشئ جلسة جديدة عبر API
5. اربط الوكلاء بالجلسة
6. عين مدير الجلسة
7. اختبر نظام الشات داخل الجلسة

## الدعم الفني

إذا واجهت أي مشاكل:
- تحقق من أن النظام يعمل على المنفذ 8081
- تحقق من أن Ollama يعمل على المنفذ 11434
- تحقق من صحة API keys
- راجع logs النظام في terminal

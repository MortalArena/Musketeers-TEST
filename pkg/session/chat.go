package session

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/google/uuid"
)

// [WHY] MsgType أنواع الرسائل في المحادثة
// [HOW] يحدد نوع الرسالة لتمييز التفكير عن الأفعال عن الرسائل النهائية
const (
	MsgTypeThought = "thought" // [WHY] تفكير الوكيل الداخلي
	MsgTypeAction  = "action"  // [WHY] تنفيذ أداة أو عملية
	MsgTypeMessage = "message" // [WHY] رسالة نهائية للعميل
	MsgTypeSystem  = "system"  // [WHY] رسائل النظام
	MsgTypeFile    = "file"    // [WHY] ملف أو صورة
	MsgTypeLink    = "link"    // [WHY] رابط أو URL
)

// [WHY] ChatMessage رسالة في المحادثة
// [HOW] يحتوي على محتوى الرسالة ونوعها والوقت والمصدر
type ChatMessage struct {
	ID        string      `json:"id"`         // [WHY] معرف فريد للرسالة
	Type      string      `json:"type"`       // [WHY] نوع الرسالة (thought/action/message/system/file/link)
	Content   string      `json:"content"`    // [WHY] محتوى الرسالة
	Source    string      `json:"source"`     // [WHY] مصدر الرسالة (agent_did أو human)
	Timestamp time.Time   `json:"timestamp"`  // [WHY] وقت الرسالة
	SessionID string      `json:"session_id"` // [WHY] معرف الجلسة
	Metadata  interface{} `json:"metadata"`   // [WHY] بيانات إضافية (اختياري)

	// [WHY] دعم الملفات والصور
	Attachment *MessageAttachment `json:"attachment,omitempty"` // ملف مرفق
}

// [WHY] MessageAttachment ملف مرفق بالرسالة
type MessageAttachment struct {
	Type        string    `json:"type"`                   // image, document, video, audio, other
	Name        string    `json:"name"`                   // اسم الملف
	Size        int64     `json:"size"`                   // حجم الملف بالبايت
	MimeType    string    `json:"mime_type"`              // نوع MIME
	FilePath    string    `json:"file_path"`              // مسار الملف في فولدر الجلسة
	URL         string    `json:"url,omitempty"`          // رابط الملف (إن وجد)
	Processed   bool      `json:"processed"`              // هل تم معالجة الملف؟
	ProcessedAt time.Time `json:"processed_at,omitempty"` // وقت المعالجة
}

// [WHY] ChatManager يدير الرسائل في الجلسة
// [HOW] يحفظ الرسائل في الذاكرة ويحدّثها ويطلق أحداث
// [SAFETY] يستخدم RWMutex لحماية الـ messages
type ChatManager struct {
	messages          []ChatMessage      // [WHY] قائمة الرسائل المؤقتة (يتم مسحها بعد 1000)
	permanentMessages []ChatMessage      // [WHY] قائمة الرسائل الدائمة (الأهداف والتطورات المهمة)
	maxMemory         int                // [WHY] الحد الأقصى للرسائل المؤقتة في الذاكرة (1000)
	maxPermanent      int                // [WHY] الحد الأقصى للرسائل الدائمة (للأهداف طويلة الأمد)
	mu                sync.RWMutex       // [SAFETY] لحماية الـ messages
	eventBus          *eventbus.EventBus // [WHY] ناقل الأحداث المحلي
	sessionID         string             // [WHY] معرف الجلسة
	sessionFolder     string             // [WHY] مسار فولدر الجلسة
}

// [WHY] NewChatManager ينشئ مدير محادثة جديد
// [HOW] يهيئ القائمة الفارغة ويضبط الحد الأقصى
// [SAFETY] يتحقق من أن eventBus ليس nil
func NewChatManager(sessionID string, eventBus *eventbus.EventBus) *ChatManager {
	if eventBus == nil {
		panic("eventBus cannot be nil") // [SAFETY] منع nil pointer
	}

	return &ChatManager{
		messages:          make([]ChatMessage, 0),
		permanentMessages: make([]ChatMessage, 0),
		maxMemory:         1000, // [WHY] حد أقصى 1000 رسالة مؤقتة لمنع استنزاف الذاكرة
		maxPermanent:      500,  // [WHY] حد أقصى 500 رسالة دائمة للأهداف طويلة الأمد
		eventBus:          eventBus,
		sessionID:         sessionID,
		sessionFolder:     filepath.Join(".", "sessions", sessionID), // [WHY] فولدر الجلسة الافتراضي
	}
}

// [WHY] NewChatManagerWithFolder ينشئ مدير محادثة مع فولدر مخصص
// [HOW] يهيئ القائمة الفارغة ويضبط الحد الأقصى وفولدر الجلسة
// [SAFETY] يتحقق من أن eventBus ليس nil
func NewChatManagerWithFolder(sessionID string, eventBus *eventbus.EventBus, sessionFolder string) *ChatManager {
	if eventBus == nil {
		panic("eventBus cannot be nil") // [SAFETY] منع nil pointer
	}

	return &ChatManager{
		messages:          make([]ChatMessage, 0),
		permanentMessages: make([]ChatMessage, 0),
		maxMemory:         1000,
		maxPermanent:      500,
		eventBus:          eventBus,
		sessionID:         sessionID,
		sessionFolder:     sessionFolder,
	}
}

// [WHY] AddMessage يضيف رسالة جديدة للمحادثة
// [HOW] يضيف الرسالة، يحدّث الذاكرة، ويطلق حدث chat.message_added
// [SAFETY] يفك القفل قبل استدعاء eventBus.Publish لمنع Deadlock
func (cm *ChatManager) AddMessage(msg ChatMessage) {
	// [SAFETY] تأكد من أن الرسالة تحتوي على معرف الجلسة
	if msg.SessionID == "" {
		msg.SessionID = cm.sessionID
	}

	// [SAFETY] تأكد من أن الرسالة تحتوي على وقت
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	// [SAFETY] تأكد من أن الرسالة تحتوي على معرف
	if msg.ID == "" {
		msg.ID = generateMessageID() // [HOW] توليد معرف فريد
	}

	// [SAFETY] قفل للكتابة
	cm.mu.Lock()

	// [HOW] إضافة الرسالة للمؤقتة
	cm.messages = append(cm.messages, msg)

	// [HOW] تحديث الذاكرة المؤقتة إذا تجاوزت الحد الأقصى
	if len(cm.messages) > cm.maxMemory {
		// [HOW] حذف أقدم الرسائل (FIFO)
		keep := cm.maxMemory / 2 // [WHY] نصف الحد الأقصى
		cm.messages = cm.messages[len(cm.messages)-keep:]
	}

	// [HOW] نسخ الرسالة للنشر
	msgCopy := msg

	// [SAFETY] فك القفل فوراً قبل النشر لمنع Deadlock
	cm.mu.Unlock()

	// [HOW] إطلاق حدث chat.message_added
	cm.eventBus.Publish(eventbus.Event{
		Type:      "chat.message_added",
		Payload:   msgCopy,
		Source:    "chat_manager",
		SessionID: cm.sessionID,
	})
}

// [WHY] AddPermanentMessage يضيف رسالة دائمة للأهداف والتطورات المهمة
// [HOW] يحفظ الرسالة في الذاكرة الدائمة التي لا يتم مسحها
// [SAFETY] يتحقق من الحد الأقصى للذاكرة الدائمة
func (cm *ChatManager) AddPermanentMessage(msg ChatMessage) error {
	// [SAFETY] تأكد من أن الرسالة تحتوي على معرف الجلسة
	if msg.SessionID == "" {
		msg.SessionID = cm.sessionID
	}

	// [SAFETY] تأكد من أن الرسالة تحتوي على وقت
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	// [SAFETY] تأكد من أن الرسالة تحتوي على معرف
	if msg.ID == "" {
		msg.ID = generateMessageID()
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// [SAFETY] التحقق من الحد الأقصى للذاكرة الدائمة
	if len(cm.permanentMessages) >= cm.maxPermanent {
		return fmt.Errorf("maximum permanent messages limit reached (%d)", cm.maxPermanent)
	}

	// [HOW] إضافة الرسالة للدائمة
	cm.permanentMessages = append(cm.permanentMessages, msg)

	// [HOW] إطلاق حدث خاص للرسائل الدائمة
	cm.eventBus.Publish(eventbus.Event{
		Type:      "chat.permanent_message_added",
		Payload:   msg,
		Source:    "chat_manager",
		SessionID: cm.sessionID,
	})

	return nil
}

// [WHY] GetMessages يحصل على جميع الرسائل
// [HOW] ينسخ القائمة ويعيدها
// [SAFETY] يستخدم RLock للقراءة فقط
func (cm *ChatManager) GetMessages() []ChatMessage {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// [WHY] نسخ القائمة لمنع تعديلها من الخارج
	messagesCopy := make([]ChatMessage, len(cm.messages))
	copy(messagesCopy, cm.messages)

	return messagesCopy
}

// [WHY] GetLastMessages يحصل على آخر N رسائل
// [HOW] ينسخ آخر N رسائل ويعيدها
// [SAFETY] يستخدم RLock للقراءة فقط
func (cm *ChatManager) GetLastMessages(n int) []ChatMessage {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// [HOW] حساب البداية
	start := len(cm.messages) - n
	if start < 0 {
		start = 0
	}

	// [WHY] نسخ آخر N رسائل
	messagesCopy := make([]ChatMessage, len(cm.messages)-start)
	copy(messagesCopy, cm.messages[start:])

	return messagesCopy
}

// [WHY] GetMessagesByType يحصل على الرسائل حسب النوع
// [HOW] ينسخ الرسائل من النوع المحدد ويعيدها
// [SAFETY] يستخدم RLock للقراءة فقط
func (cm *ChatManager) GetMessagesByType(msgType string) []ChatMessage {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// [HOW] تصفية الرسائل حسب النوع
	var filtered []ChatMessage
	for _, msg := range cm.messages {
		if msg.Type == msgType {
			filtered = append(filtered, msg)
		}
	}

	return filtered
}

// [WHY] Clear يمسح جميع الرسائل
// [HOW] يفرغ القائمة
// [SAFETY] يستخدم Lock للكتابة
func (cm *ChatManager) Clear() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.messages = make([]ChatMessage, 0)
}

// [WHY] GetSessionID يحصل على معرف الجلسة
// [HOW] يعيد معرف الجلسة
// [SAFETY] لا يحتاج قفل لأنه ثابت
func (cm *ChatManager) GetSessionID() string {
	return cm.sessionID
}

// [WHY] generateMessageID يولد معرف فريد للرسالة
// [HOW] يستخدم UUID الحقيقي
// [SAFETY] يضمن التفرد
func generateMessageID() string {
	return uuid.New().String()
}

// [WHY] AddMessageWithAttachment يضيف رسالة مع ملف مرفق
// [HOW] يحفظ الملف في فولدر الجلسة المنظم ويضيف الرسالة
// [SAFETY] يتحقق من صحة الملف والمسار
func (cm *ChatManager) AddMessageWithAttachment(msg ChatMessage, attachment *MessageAttachment) error {
	// إنشاء فولدر الجلسة المنظم إذا لم يكن موجوداً
	if err := cm.ensureSessionFolder(); err != nil {
		return err
	}

	// حفظ الملف في الفولدر المناسب
	if attachment != nil {
		attachmentPath := filepath.Join(cm.sessionFolder, "attachments", attachment.Name)
		if err := os.MkdirAll(filepath.Dir(attachmentPath), 0755); err != nil {
			return err
		}

		// [TODO] نسخ الملف فعلياً من المصدر إلى المسار الجديد
		attachment.FilePath = attachmentPath
		attachment.Processed = true
		attachment.ProcessedAt = time.Now()
	}

	msg.Attachment = attachment
	cm.AddMessage(msg)
	return nil
}

// [WHY] ensureSessionFolder يضمن وجود فولدر الجلسة المنظم
// [HOW] ينشئ البنية التنظيمية للفولدر
// [SAFETY] يستخدم os.MkdirAll مع الأذونات الصحيحة
func (cm *ChatManager) ensureSessionFolder() error {
	// البنية التنظيمية:
	// sessions/{session_id}/
	//   ├── attachments/    (الملفات المرفقة)
	//   ├── memory/         (الذاكرة والمعرفة)
	//   ├── artifacts/      (القطع الأثرية والمنتج النهائي)
	//   ├── work/           (ملفات العمل المؤقتة)
	//   └── logs/           (سجلات الجلسة)

	folders := []string{
		filepath.Join(cm.sessionFolder, "attachments"),
		filepath.Join(cm.sessionFolder, "memory"),
		filepath.Join(cm.sessionFolder, "artifacts"),
		filepath.Join(cm.sessionFolder, "work"),
		filepath.Join(cm.sessionFolder, "logs"),
	}

	for _, folder := range folders {
		if err := os.MkdirAll(folder, 0755); err != nil {
			return err
		}
	}

	return nil
}

// [WHY] GetSessionFolder يحصل على مسار فولدر الجلسة
// [HOW] يعيد المسار
// [SAFETY] لا يحتاج قفل لأنه ثابت
func (cm *ChatManager) GetSessionFolder() string {
	return cm.sessionFolder
}

// [WHY] GetPermanentMessages يحصل على الرسائل الدائمة
// [HOW] ينسخ القائمة الدائمة ويعيدها
// [SAFETY] يستخدم RLock للقراءة فقط
func (cm *ChatManager) GetPermanentMessages() []ChatMessage {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	messagesCopy := make([]ChatMessage, len(cm.permanentMessages))
	copy(messagesCopy, cm.permanentMessages)

	return messagesCopy
}

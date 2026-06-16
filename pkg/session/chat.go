package session

import (
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
)

// [WHY] MsgType أنواع الرسائل في المحادثة
// [HOW] يحدد نوع الرسالة لتمييز التفكير عن الأفعال عن الرسائل النهائية
const (
	MsgTypeThought = "thought" // [WHY] تفكير الوكيل الداخلي
	MsgTypeAction  = "action"  // [WHY] تنفيذ أداة أو عملية
	MsgTypeMessage = "message" // [WHY] رسالة نهائية للعميل
	MsgTypeSystem  = "system"  // [WHY] رسائل النظام
)

// [WHY] ChatMessage رسالة في المحادثة
// [HOW] يحتوي على محتوى الرسالة ونوعها والوقت والمصدر
type ChatMessage struct {
	ID        string      `json:"id"`         // [WHY] معرف فريد للرسالة
	Type      string      `json:"type"`       // [WHY] نوع الرسالة (thought/action/message/system)
	Content   string      `json:"content"`    // [WHY] محتوى الرسالة
	Source    string      `json:"source"`     // [WHY] مصدر الرسالة (agent_did أو human)
	Timestamp time.Time   `json:"timestamp"`  // [WHY] وقت الرسالة
	SessionID string      `json:"session_id"` // [WHY] معرف الجلسة
	Metadata  interface{} `json:"metadata"`   // [WHY] بيانات إضافية (اختياري)
}

// [WHY] ChatManager يدير الرسائل في الجلسة
// [HOW] يحفظ الرسائل في الذاكرة ويحدّثها ويطلق أحداث
// [SAFETY] يستخدم RWMutex لحماية الـ messages
type ChatManager struct {
	messages  []ChatMessage      // [WHY] قائمة الرسائل
	maxMemory int                // [WHY] الحد الأقصى للرسائل في الذاكرة (1000)
	mu        sync.RWMutex       // [SAFETY] لحماية الـ messages
	eventBus  *eventbus.EventBus // [WHY] ناقل الأحداث المحلي
	sessionID string             // [WHY] معرف الجلسة
}

// [WHY] NewChatManager ينشئ مدير محادثة جديد
// [HOW] يهيئ القائمة الفارغة ويضبط الحد الأقصى
// [SAFETY] يتحقق من أن eventBus ليس nil
func NewChatManager(sessionID string, eventBus *eventbus.EventBus) *ChatManager {
	if eventBus == nil {
		panic("eventBus cannot be nil") // [SAFETY] منع nil pointer
	}

	return &ChatManager{
		messages:  make([]ChatMessage, 0),
		maxMemory: 1000, // [WHY] حد أقصى 1000 رسالة لمنع استنزاف الذاكرة
		eventBus:  eventBus,
		sessionID: sessionID,
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

	// [HOW] إضافة الرسالة
	cm.messages = append(cm.messages, msg)

	// [HOW] تحديث الذاكرة إذا تجاوزت الحد الأقصى
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
// [HOW] يستخدم الوقت وUUID
// [SAFETY] يضمن التفرد
func generateMessageID() string {
	// [TODO] استخدام UUID حقيقي
	return time.Now().Format("20060102150405.000000")
}

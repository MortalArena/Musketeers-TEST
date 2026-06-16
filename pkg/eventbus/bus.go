package eventbus

import (
	"sync"
	"time"
)

// [WHY] EventBus ناقل الأحداث المركزي - يربط كل المكونات
// [HOW] يستخدم قائمة انتظار لمنع استنزاف الذاكرة من goroutines
// [SAFETY] يستخدم RWMutex لحماية الـ handlers و eventQueue
type EventBus struct {
	handlers   map[string][]Handler
	mu         sync.RWMutex
	eventQueue chan Event   // [WHY] قائمة انتظار للأحداث لمنع Goroutine Leak
	running    bool         // [WHY] لمعرفة ما إذا كانت عملية المعالجة تعمل
	queueMu    sync.RWMutex // [SAFETY] لحماية حالة running
}

// Handler دالة معالجة الحدث
type Handler func(event Event)

// Event حدث في النظام
type Event struct {
	Type      string      `json:"type"`       // نوع الحدث
	Payload   interface{} `json:"payload"`    // البيانات
	Source    string      `json:"source"`     // المصدر
	Timestamp time.Time   `json:"timestamp"`  // الوقت
	SessionID string      `json:"session_id"` // الجلسة (اختياري)
}

// NewEventBus ينشئ ناقل أحداث جديد
// [WHY] يهيئ قائمة الانتظار ويبدأ عملية المعالجة الخلفية
// [HOW] ينشئ قناة بسعة 10000 ويبدأ goroutine واحدة لمعالجة الأحداث
// [SAFETY] يستخدم defer recover() لمنع تعطل النظام من panic
func NewEventBus() *EventBus {
	eb := &EventBus{
		handlers:   make(map[string][]Handler),
		eventQueue: make(chan Event, 10000), // [WHY] سعة 10000 لمنع الحظر تحت الحمل
		running:    true,
	}

	// [HOW] ابدأ عملية المعالجة الخلفية في goroutine واحدة
	go eb.processQueue()

	return eb
}

// [WHY] processQueue يعالج الأحداث من قائمة الانتظار في goroutine واحدة
// [HOW] يقرأ من eventQueue بشكل مستمر، ويطبق recover()، وينفذ المعالجات
// [SAFETY] يستخدم defer recover() لمنع تعطل النظام من panic
func (eb *EventBus) processQueue() {
	// [SAFETY] استرد من أي panic لمنع تعطل النظام
	defer func() {
		if r := recover(); r != nil {
			// [TODO] تسجيل الخطأ في logger
			// [HOW] أعد تشغيل عملية المعالجة إذا حدث panic
			go eb.processQueue()
		}
	}()

	for event := range eb.eventQueue {
		// [HOW] معالجة الحدث
		eb.processEvent(event)
	}
}

// [WHY] processEvent ينفذ المعالجين لحدث معين
// [HOW] ينسخ قائمة المعالجين وينفذهم
// [SAFETY] يستخدم RLock للقراءة فقط
func (eb *EventBus) processEvent(event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	// [HOW] معالجةWildcard (*) - يستمع لكل الأحداث
	if handlers, ok := eb.handlers["*"]; ok {
		// [WHY] نسخ قائمة المعالجين لمنع تعديلها أثناء التنفيذ
		handlersCopy := make([]Handler, len(handlers))
		copy(handlersCopy, handlers)

		for _, handler := range handlersCopy {
			// [SAFETY] استرد من panic في كل معالج
			func() {
				defer func() {
					if r := recover(); r != nil {
						// [TODO] تسجيل الخطأ في logger
					}
				}()
				handler(event)
			}()
		}
	}

	// [HOW] معالجة النوع المحدد
	if handlers, ok := eb.handlers[event.Type]; ok {
		// [WHY] نسخ قائمة المعالجين لمنع تعديلها أثناء التنفيذ
		handlersCopy := make([]Handler, len(handlers))
		copy(handlersCopy, handlers)

		for _, handler := range handlersCopy {
			// [SAFETY] استرد من panic في كل معالج
			func() {
				defer func() {
					if r := recover(); r != nil {
						// [TODO] تسجيل الخطأ في logger
					}
				}()
				handler(event)
			}()
		}
	}
}

// Subscribe يسجل معالجاً لحدث معين
func (eb *EventBus) Subscribe(eventType string, handler Handler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

// Publish ينشر حدثاً لكل المعالجين
// [WHY] يستخدم قائمة انتظار لمنع Goroutine Leak تحت الحمل
// [HOW] يضع الحدث في eventQueue باستخدام select مع default لمنع الحظر
// [SAFETY] لا يحظر أبداً حتى لو كانت القائمة ممتلئة
func (eb *EventBus) Publish(event Event) {
	eb.queueMu.RLock()
	running := eb.running
	eb.queueMu.RUnlock()

	if !running {
		return // [SAFETY] لا تنشر إذا كانت عملية المعالجة متوقفة
	}

	event.Timestamp = time.Now()

	// [HOW] وضع الحدث في القائمة الانتظار دون حظر
	select {
	case eb.eventQueue <- event:
		// [OK] الحدث تم وضعه في القائمة
	default:
		// [SAFETY] القائمة ممتلئة، تجاهل الحدث لمنع الحظر
		// [TODO] تسجيل تحذير في logger
	}
}

// Unsubscribe يزيل معالجاً
func (eb *EventBus) Unsubscribe(eventType string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	delete(eb.handlers, eventType)
}

// Clear يمسح كل المعالجين
func (eb *EventBus) Clear() {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers = make(map[string][]Handler)
}

// Stop يوقف عملية المعالجة بشكل آمن
// [WHY] لإيقاف EventBus بشكل صحيح عند إغلاق النظام
// [HOW] يوقف وضع الأحداث الجديدة ويغلق القناة
// [SAFETY] يستخدم queueMu لحماية حالة running
func (eb *EventBus) Stop() {
	eb.queueMu.Lock()
	defer eb.queueMu.Unlock()

	if !eb.running {
		return // [SAFETY] لا تتوقف إذا كانت متوقفة بالفعل
	}

	eb.running = false
	close(eb.eventQueue) // [HOW] إغلاق القناة لإيقاف processQueue
}

package eventbus

import (
	"sync"
	"time"
)

// EventBus ناقل الأحداث المركزي - يربط كل المكونات
type EventBus struct {
	handlers map[string][]Handler
	mu       sync.RWMutex
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
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[string][]Handler),
	}
}

// Subscribe يسجل معالجاً لحدث معين
func (eb *EventBus) Subscribe(eventType string, handler Handler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

// Publish ينشر حدثاً لكل المعالجين
func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	event.Timestamp = time.Now()

	// معالجةWildcard (*) - يستمع لكل الأحداث
	if handlers, ok := eb.handlers["*"]; ok {
		for _, handler := range handlers {
			go handler(event)
		}
	}

	// معالجة النوع المحدد
	if handlers, ok := eb.handlers[event.Type]; ok {
		for _, handler := range handlers {
			go handler(event)
		}
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

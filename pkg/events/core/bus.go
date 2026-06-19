package core

import (
	"sync"
	"time"
)

// UnifiedEventBus ناقل الأحداث الموحد
type UnifiedEventBus struct {
	handlers   map[string][]Handler
	mu         sync.RWMutex
	eventQueue chan Event
	running    bool
	queueMu    sync.RWMutex
}

// Handler دالة معالجة الحدث
type Handler func(event Event)

// Event حدث في النظام
type Event struct {
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload"`
	Source    string      `json:"source"`
	Timestamp time.Time   `json:"timestamp"`
	SessionID string      `json:"session_id"`
}

// NewUnifiedEventBus ينشئ ناقل أحداث موحد جديد
func NewUnifiedEventBus() *UnifiedEventBus {
	eb := &UnifiedEventBus{
		handlers:   make(map[string][]Handler),
		eventQueue: make(chan Event, 10000),
		running:    true,
	}

	go eb.processQueue()

	return eb
}

// processQueue يعالج الأحداث من قائمة الانتظار
func (eb *UnifiedEventBus) processQueue() {
	defer func() {
		if r := recover(); r != nil {
			go eb.processQueue()
		}
	}()

	for event := range eb.eventQueue {
		eb.processEvent(event)
	}
}

// processEvent ينفذ المعالجين لحدث معين
func (eb *UnifiedEventBus) processEvent(event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	// معالجة Wildcard (*) - يستمع لكل الأحداث
	if handlers, ok := eb.handlers["*"]; ok {
		handlersCopy := make([]Handler, len(handlers))
		copy(handlersCopy, handlers)

		for _, handler := range handlersCopy {
			func() {
				defer func() {
					if r := recover(); r != nil {
						// تسجيل الخطأ
					}
				}()
				handler(event)
			}()
		}
	}

	// معالجة النوع المحدد
	if handlers, ok := eb.handlers[event.Type]; ok {
		handlersCopy := make([]Handler, len(handlers))
		copy(handlersCopy, handlers)

		for _, handler := range handlersCopy {
			func() {
				defer func() {
					if r := recover(); r != nil {
						// تسجيل الخطأ
					}
				}()
				handler(event)
			}()
		}
	}
}

// Subscribe يسجل معالجاً لحدث معين
func (eb *UnifiedEventBus) Subscribe(eventType string, handler Handler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

// Publish ينشر حدثاً لكل المعالجين
func (eb *UnifiedEventBus) Publish(event Event) {
	eb.queueMu.RLock()
	running := eb.running
	eb.queueMu.RUnlock()

	if !running {
		return
	}

	event.Timestamp = time.Now()

	select {
	case eb.eventQueue <- event:
	default:
		// القائمة ممتلئة، تجاهل الحدث لمنع الحظر
	}
}

// Unsubscribe يزيل معالجاً
func (eb *UnifiedEventBus) Unsubscribe(eventType string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	delete(eb.handlers, eventType)
}

// Clear يمسح كل المعالجين
func (eb *UnifiedEventBus) Clear() {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers = make(map[string][]Handler)
}

// Stop يوقف عملية المعالجة بشكل آمن
func (eb *UnifiedEventBus) Stop() {
	eb.queueMu.Lock()
	defer eb.queueMu.Unlock()

	if !eb.running {
		return
	}

	eb.running = false
	close(eb.eventQueue)
}

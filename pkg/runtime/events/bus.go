package events

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type EventBus interface {
	Publish(event Event) error
	Subscribe(eventType string, handler EventHandler) (Subscription, error)
	Replay(from time.Time, handler EventHandler) error
	Close() error
}

type Subscription interface {
	ID() string
	Unsubscribe() error
}

type EventHandler func(event Event) error

type MemoryEventBus struct {
	mu         sync.RWMutex
	handlers   map[string][]eventHandlerEntry
	history    []Event
	maxHistory int
	closed     bool
	wg         sync.WaitGroup
}

type eventHandlerEntry struct {
	id      string
	handler EventHandler
}

func NewMemoryEventBus(maxHistory int) *MemoryEventBus {
	if maxHistory <= 0 {
		maxHistory = 10000
	}
	return &MemoryEventBus{
		handlers:   make(map[string][]eventHandlerEntry),
		history:    make([]Event, 0, maxHistory),
		maxHistory: maxHistory,
	}
}

func (b *MemoryEventBus) Publish(event Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return fmt.Errorf("event bus is closed")
	}
	if event.ID == "" {
		event.ID = generateEventID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.Data == nil {
		event.Data = map[string]any{}
	}
	b.history = append(b.history, event)
	if len(b.history) > b.maxHistory {
		b.history = b.history[len(b.history)-b.maxHistory:]
	}
	for _, entry := range append([]eventHandlerEntry(nil), b.handlers[event.Type]...) {
		b.wg.Add(1)
		go func(handler EventHandler) {
			defer b.wg.Done()
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("panic in event handler: %v\n", r)
				}
			}()
			callHandler(handler, event)
		}(entry.handler)
	}
	for _, entry := range append([]eventHandlerEntry(nil), b.handlers["*"]...) {
		b.wg.Add(1)
		go func(handler EventHandler) {
			defer b.wg.Done()
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("panic in wildcard event handler: %v\n", r)
				}
			}()
			callHandler(handler, event)
		}(entry.handler)
	}
	return nil
}

func (b *MemoryEventBus) Subscribe(eventType string, handler EventHandler) (Subscription, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, fmt.Errorf("event bus is closed")
	}
	if handler == nil {
		return nil, fmt.Errorf("handler is nil")
	}
	id := generateEventID()
	entry := eventHandlerEntry{id: id, handler: handler}
	b.handlers[eventType] = append(b.handlers[eventType], entry)
	return &memorySubscription{id: id, eventType: eventType, bus: b}, nil
}

func (b *MemoryEventBus) Replay(from time.Time, handler EventHandler) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if handler == nil {
		return fmt.Errorf("handler is nil")
	}
	for _, event := range b.history {
		if event.Timestamp.After(from) || event.Timestamp.Equal(from) {
			if err := handler(event); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *MemoryEventBus) Close() error {
	b.mu.Lock()
	b.closed = true
	b.handlers = make(map[string][]eventHandlerEntry)
	b.mu.Unlock()
	b.wg.Wait()
	return nil
}

func (b *MemoryEventBus) unsubscribe(eventType, id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	handlers := b.handlers[eventType]
	for i, entry := range handlers {
		if entry.id == id {
			b.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			return
		}
	}
}

func callHandler(handler EventHandler, event Event) {
	if err := handler(event); err != nil {
		fmt.Printf("Error handling event %s: %v\n", event.ID, err)
	}
}

type memorySubscription struct {
	id        string
	eventType string
	bus       *MemoryEventBus
}

func (s *memorySubscription) ID() string { return s.id }

func (s *memorySubscription) Unsubscribe() error {
	s.bus.unsubscribe(s.eventType, s.id)
	return nil
}

var eventSeq uint64

func generateEventID() string {
	return fmt.Sprintf("event-%d", atomic.AddUint64(&eventSeq, 1))
}

package adapters

import (
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/MortalArena/Musketeers/pkg/sdk/interfaces"
)

type EventBusAdapter struct {
	bus *eventbus.EventBus
}

func NewEventBusAdapter(bus *eventbus.EventBus) *EventBusAdapter {
	return &EventBusAdapter{bus: bus}
}

func (a *EventBusAdapter) Publish(eventType string, payload interface{}) {
	a.bus.Publish(eventbus.Event{Type: eventType, Payload: payload})
}

func (a *EventBusAdapter) Subscribe(eventType string, handler func(eventType string, payload interface{})) {
	a.bus.Subscribe(eventType, func(e eventbus.Event) {
		handler(e.Type, e.Payload)
	})
}

func (a *EventBusAdapter) Unsubscribe(eventType string) {
	a.bus.Unsubscribe(eventType)
}

var _ interfaces.EventBus = (*EventBusAdapter)(nil)

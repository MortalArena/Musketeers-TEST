package adapters

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/MortalArena/Musketeers/pkg/sdk/interfaces"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type UIBridgeAdapter struct {
	bus      *eventbus.EventBus
	clients  map[*websocket.Conn]bool
	mu       sync.RWMutex
	handlers map[string][]func(data interface{})
}

func NewUIBridgeAdapter(bus *eventbus.EventBus) *UIBridgeAdapter {
	a := &UIBridgeAdapter{
		bus:      bus,
		clients:  make(map[*websocket.Conn]bool),
		handlers: make(map[string][]func(data interface{})),
	}
	return a
}

func (a *UIBridgeAdapter) Send(event string, data interface{}) error {
	msg, _ := json.Marshal(map[string]interface{}{
		"event": event,
		"data":  data,
	})
	a.mu.RLock()
	defer a.mu.RUnlock()
	for conn := range a.clients {
		conn.WriteMessage(websocket.TextMessage, msg)
	}
	return nil
}

func (a *UIBridgeAdapter) On(event string, handler func(data interface{})) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.handlers[event] = append(a.handlers[event], handler)
}

func (a *UIBridgeAdapter) Broadcast(event string, data interface{}) error {
	return a.Send(event, data)
}

func (a *UIBridgeAdapter) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	a.mu.Lock()
	a.clients[conn] = true
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		delete(a.clients, conn)
		a.mu.Unlock()
		conn.Close()
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var envelope struct {
			Event string          `json:"event"`
			Data  json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(msg, &envelope); err != nil {
			continue
		}
		a.mu.RLock()
		handlers := a.handlers[envelope.Event]
		a.mu.RUnlock()
		for _, h := range handlers {
			h(envelope.Data)
		}
	}
}

var _ interfaces.UIBridgeInterface = (*UIBridgeAdapter)(nil)

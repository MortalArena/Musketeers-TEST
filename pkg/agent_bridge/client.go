package agent_bridge

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/agent_bridge/protocol"
	"github.com/sirupsen/logrus"
)

// Client عميل للاتصال بـ Agent Bridge
type Client struct {
	addr       string
	conn       net.Conn
	sessionID  string
	log        *logrus.Logger
	mu         sync.RWMutex
	connected  bool
}

// NewClient ينشئ عميل جديد
func NewClient(addr string, log *logrus.Logger) *Client {
	return &Client{
		addr: addr,
		log:  log,
	}
}

// Connect يتصل بالخادم
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return fmt.Errorf("already connected")
	}

	conn, err := net.Dial("tcp", c.addr)
	if err != nil {
		return fmt.Errorf("failed to connect to bridge: %w", err)
	}

	c.conn = conn
	c.connected = true
	c.log.WithField("addr", c.addr).Info("Connected to Agent Bridge")

	return nil
}

// Disconnect يفصل الاتصال
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			c.log.WithError(err).Error("Failed to close connection")
		}
	}

	c.connected = false
	c.log.Info("Disconnected from Agent Bridge")

	return nil
}

// SendTaskRequest يرسل طلب مهمة
func (c *Client) SendTaskRequest(ctx context.Context, req *TaskRequest) (*TaskResponse, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	tp := NewTaskProtocol()
	msg, err := tp.EncodeTaskRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode task request: %w", err)
	}

	if err := protocol.WriteMessage(c.conn, msg); err != nil {
		return nil, fmt.Errorf("failed to send task request: %w", err)
	}

	// قراءة الاستجابة
	respMsg, err := protocol.ReadMessage(c.conn)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	resp, err := tp.DecodeTaskResponse(respMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return resp, nil
}

// IsConnected يرجع حالة الاتصال
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

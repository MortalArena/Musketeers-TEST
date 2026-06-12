package messaging

import (
	"context"
	"fmt"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/capability"
	"github.com/MortalArena/Musketeers/pkg/policy"
)

type Sender interface {
	Send(ctx context.Context, to, channel, message string) error
}

type Joiner interface {
	Join(ctx context.Context, channel string) error
}

type MessagingCapability struct {
	Sender Sender
	Joiner Joiner
}

func NewMessagingCapability(sender Sender, joiner Joiner) *MessagingCapability {
	return &MessagingCapability{Sender: sender, Joiner: joiner}
}

func (c *MessagingCapability) Name() string { return "messaging" }

func (c *MessagingCapability) Execute(ctx context.Context, principal policy.Principal, cmd capability.Command) (*capability.Result, error) {
	switch v := cmd.(type) {
	case SendMessageCommand:
		return c.sendMessage(ctx, v)
	case JoinChannelCommand:
		return c.joinChannel(ctx, v)
	default:
		return nil, fmt.Errorf("unsupported messaging command: %s", cmd.Name())
	}
}

type SendMessageCommand struct {
	To      string `json:"to"`
	Channel string `json:"channel,omitempty"`
	Message string `json:"message"`
}

func (SendMessageCommand) Name() string { return "messaging.send_message" }
func (c SendMessageCommand) Args() map[string]any {
	return map[string]any{"to": c.To, "channel": c.Channel, "message": c.Message}
}

type JoinChannelCommand struct {
	Channel string `json:"channel"`
}

func (JoinChannelCommand) Name() string { return "messaging.join_channel" }
func (c JoinChannelCommand) Args() map[string]any {
	return map[string]any{"channel": c.Channel}
}

func (c *MessagingCapability) sendMessage(ctx context.Context, cmd SendMessageCommand) (*capability.Result, error) {
	if cmd.To == "" && cmd.Channel == "" {
		return nil, fmt.Errorf("to or channel is required")
	}
	if cmd.Message == "" {
		return nil, fmt.Errorf("message is required")
	}
	if c.Sender == nil {
		return nil, fmt.Errorf("sender is not configured")
	}
	if err := c.Sender.Send(ctx, cmd.To, cmd.Channel, cmd.Message); err != nil {
		return nil, err
	}
	return capability.NewResult(cmd.Name(), map[string]any{"sent": true}), nil
}

func (c *MessagingCapability) joinChannel(ctx context.Context, cmd JoinChannelCommand) (*capability.Result, error) {
	if cmd.Channel == "" {
		return nil, fmt.Errorf("channel is required")
	}
	if c.Joiner == nil {
		return nil, fmt.Errorf("joiner is not configured")
	}
	if err := c.Joiner.Join(ctx, cmd.Channel); err != nil {
		return nil, err
	}
	return capability.NewResult(cmd.Name(), map[string]any{"joined": cmd.Channel}), nil
}

type MemoryMessenger struct {
	mu       sync.RWMutex
	messages []SentMessage
	channels map[string]bool
}

type SentMessage struct {
	To      string
	Channel string
	Message string
}

func NewMemoryMessenger() *MemoryMessenger {
	return &MemoryMessenger{channels: make(map[string]bool)}
}

func (m *MemoryMessenger) Send(ctx context.Context, to, channel, message string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, SentMessage{To: to, Channel: channel, Message: message})
	return nil
}

func (m *MemoryMessenger) Join(ctx context.Context, channel string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels[channel] = true
	return nil
}

func (m *MemoryMessenger) Messages() []SentMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]SentMessage(nil), m.messages...)
}

func (m *MemoryMessenger) Channels() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	channels := make([]string, 0, len(m.channels))
	for channel := range m.channels {
		channels = append(channels, channel)
	}
	return channels
}

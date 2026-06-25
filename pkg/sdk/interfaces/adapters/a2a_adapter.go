package adapters

import (
	"context"
	"encoding/json"

	"github.com/MortalArena/Musketeers/pkg/node"
	"github.com/MortalArena/Musketeers/pkg/sdk/interfaces"
	"github.com/libp2p/go-libp2p/core/peer"
)

type A2AAdapter struct {
	node *node.Node
}

func NewA2AAdapter(n *node.Node) *A2AAdapter {
	return &A2AAdapter{node: n}
}

func (a *A2AAdapter) Send(ctx context.Context, target string, msg *interfaces.A2AMessage) error {
	pid, err := peer.Decode(target)
	if err != nil {
		return err
	}
	var input interface{}
	if len(msg.Payload) > 0 {
		json.Unmarshal(msg.Payload, &input)
	}
	_, err = a.node.SendACPTask(ctx, pid, msg.Target, msg.Type, input)
	return err
}

func (a *A2AAdapter) Receive(ctx context.Context) (*interfaces.A2AMessage, error) {
	return nil, nil
}

func (a *A2AAdapter) RegisterHandler(handler func(ctx context.Context, msg *interfaces.A2AMessage) (*interfaces.A2AMessage, error)) error {
	a.node.RegisterACPTask("task", func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
		in := &interfaces.A2AMessage{
			Payload: input,
		}
		out, err := handler(ctx, in)
		if err != nil {
			return nil, err
		}
		return out.Payload, nil
	})
	return nil
}

var _ interfaces.A2AInterface = (*A2AAdapter)(nil)

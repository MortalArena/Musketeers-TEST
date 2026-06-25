package adapters

import (
	"context"
	"encoding/json"

	"github.com/MortalArena/Musketeers/pkg/channel"
	"github.com/MortalArena/Musketeers/pkg/sdk/interfaces"
)

type CommAdapter struct {
	manager channel.ChannelManager
}

func NewCommAdapter(manager channel.ChannelManager) *CommAdapter {
	return &CommAdapter{manager: manager}
}

func (a *CommAdapter) Publish(ctx context.Context, channelID string, msg []byte) error {
	return a.manager.Publish(ctx, channelID, msg)
}

func (a *CommAdapter) Subscribe(ctx context.Context, channelID string, handler interfaces.MessageHandler) (interfaces.Subscription, error) {
	rawSub, err := a.manager.Subscribe(ctx, channelID, func(data []byte) {
		handler(data)
	})
	if err != nil {
		return nil, err
	}
	return &commSubscription{sub: rawSub}, nil
}

type commSubscription struct {
	sub interface{}
}

func (s *commSubscription) ID() string {
	return ""
}

func (s *commSubscription) Close() error {
	return nil
}

func marshalMsg(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}

var _ interfaces.CommunicationInterface = (*CommAdapter)(nil)

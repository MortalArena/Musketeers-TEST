package node

import (
	"context"
	"crypto/ed25519"
	"encoding/json"

	"github.com/MortalArena/Musketeers/pkg/channel"
)

// PublishChannelConfig ينشر إعدادات قناة خاصة على DHT
func (n *Node) PublishChannelConfig(ctx context.Context, cfg *channel.ChannelConfig) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	key := "/nr/channel-config/" + cfg.ID
	return n.dht().PutValue(ctx, key, data)
}

// GetChannelConfig يجلب إعدادات قناة
func (n *Node) GetChannelConfig(ctx context.Context, channelID string) (*channel.ChannelConfig, error) {
	val, err := n.dht().GetValue(ctx, "/nr/channel-config/"+channelID)
	if err != nil {
		return nil, err
	}
	var cfg channel.ChannelConfig
	if err := json.Unmarshal(val, &cfg); err != nil {
		return nil, err
	}
	ownerPub, err := n.ResolvePublicKey(cfg.Owner)
	if err != nil {
		return nil, err
	}
	if err := cfg.VerifyConfigV2(ownerPub); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// RemoveChannelMember يزيل عضواً مع تدوير المفتاح
func (n *Node) RemoveChannelMember(ctx context.Context, cfg *channel.ChannelConfig, memberDID string, memberPubs map[string]ed25519.PublicKey) ([]byte, error) {
	newKey, err := cfg.RemoveMember(memberDID, n.keyPair().DID, n.keyPair().Private, memberPubs)
	if err != nil {
		return nil, err
	}
	if err := n.PublishChannelConfig(ctx, cfg); err != nil {
		return nil, err
	}
	return newKey, nil
}

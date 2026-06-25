package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/MortalArena/Musketeers/pkg/node"
	"github.com/MortalArena/Musketeers/pkg/sdk/interfaces"
	"github.com/libp2p/go-libp2p/core/peer"
)

type NodeAdapter struct {
	node *node.Node
}

func NewNodeAdapter(n *node.Node) *NodeAdapter {
	return &NodeAdapter{node: n}
}

func (a *NodeAdapter) PublishIdentity(ctx context.Context) error {
	return a.node.PublishIdentity(ctx)
}

func (a *NodeAdapter) ResolveDomain(ctx context.Context, name string) (*interfaces.DomainRecord, error) {
	rec, err := a.node.ResolveDomain(ctx, name)
	if err != nil {
		return nil, err
	}
	return &interfaces.DomainRecord{
		Name:      rec.Name,
		Owner:     rec.Owner,
		Addresses: []string{rec.Target},
		CreatedAt: time.Unix(rec.UpdatedAt, 0),
		ExpiresAt: time.Unix(rec.ExpiresAt, 0),
	}, nil
}

func (a *NodeAdapter) Connect(ctx context.Context, addr string) error {
	addrInfo, err := peer.AddrInfoFromString(addr)
	if err != nil {
		return fmt.Errorf("invalid peer address: %w", err)
	}
	return a.node.Host().Connect(ctx, *addrInfo)
}

func (a *NodeAdapter) Close() error {
	return a.node.Host().Close()
}

var _ interfaces.NodeInterface = (*NodeAdapter)(nil)

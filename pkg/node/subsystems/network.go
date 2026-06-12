package subsystems

import (
	"context"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

type NetworkSubsystem struct {
	host host.Host
	dht  *dht.IpfsDHT
	ps   *pubsub.PubSub
}

func NewNetworkSubsystem(h host.Host, kad *dht.IpfsDHT, ps *pubsub.PubSub) *NetworkSubsystem {
	return &NetworkSubsystem{host: h, dht: kad, ps: ps}
}

func (s *NetworkSubsystem) Host() host.Host        { return s.host }
func (s *NetworkSubsystem) DHT() *dht.IpfsDHT      { return s.dht }
func (s *NetworkSubsystem) PubSub() *pubsub.PubSub { return s.ps }
func (s *NetworkSubsystem) Close(ctx context.Context) error {
	if s.host == nil {
		return nil
	}
	return s.host.Close()
}

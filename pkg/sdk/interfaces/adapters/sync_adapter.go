package adapters

import (
	"context"
	"fmt"

	"github.com/MortalArena/Musketeers/pkg/sdk"
	"github.com/MortalArena/Musketeers/pkg/sdk/interfaces"
)

type SyncAdapter struct {
	manager *sdk.CRDTSyncManager
	handler func(data []byte)
}

func NewSyncAdapter(manager *sdk.CRDTSyncManager) *SyncAdapter {
	a := &SyncAdapter{manager: manager}
	ctx := context.Background()
	manager.Subscribe(ctx, "sync-adapter", func(update []byte, senderDID string) {
		if a.handler != nil {
			a.handler(update)
		}
	})
	return a
}

func (a *SyncAdapter) Sync(ctx context.Context, data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty sync data")
	}
	return a.manager.BroadcastUpdate(ctx, data)
}

func (a *SyncAdapter) OnSync(handler func(data []byte)) {
	a.handler = handler
}

func (a *SyncAdapter) GetState(key string) ([]byte, error) {
	return nil, fmt.Errorf("key-value state not supported by CRDT sync adapter")
}

func (a *SyncAdapter) SetState(key string, value []byte) error {
	return a.Sync(context.Background(), value)
}

var _ interfaces.SyncInterface = (*SyncAdapter)(nil)
package adapters

import (
	"github.com/MortalArena/Musketeers/pkg/content"
	"github.com/MortalArena/Musketeers/pkg/sdk/interfaces"
)

type StorageAdapter struct {
	store content.BlockStore
}

func NewStorageAdapter(store content.BlockStore) *StorageAdapter {
	return &StorageAdapter{store: store}
}

func (a *StorageAdapter) Get(cid string) ([]byte, error) {
	return a.store.Get(cid)
}

func (a *StorageAdapter) Put(cid string, data []byte, did string) error {
	return a.store.Put(cid, data, did)
}

func (a *StorageAdapter) Size() int64 {
	return a.store.Size()
}

func (a *StorageAdapter) List(prefix string) ([]string, error) {
	return a.store.ListKeys(prefix)
}

func (a *StorageAdapter) Close() error {
	return nil
}

var _ interfaces.StorageInterface = (*StorageAdapter)(nil)

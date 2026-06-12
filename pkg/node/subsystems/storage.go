package subsystems

import (
	"github.com/dgraph-io/badger/v4"
	"github.com/MortalArena/Musketeers/pkg/content"
)

type StorageSubsystem struct {
	db         *badger.DB
	blockStore content.BlockStore
	provider   *content.ProviderManager
	fetcher    *content.Fetcher
}

func NewStorageSubsystem(db *badger.DB, blockStore content.BlockStore, provider *content.ProviderManager, fetcher *content.Fetcher) *StorageSubsystem {
	return &StorageSubsystem{db: db, blockStore: blockStore, provider: provider, fetcher: fetcher}
}

func (s *StorageSubsystem) DB() *badger.DB                     { return s.db }
func (s *StorageSubsystem) BlockStore() content.BlockStore     { return s.blockStore }
func (s *StorageSubsystem) Provider() *content.ProviderManager { return s.provider }
func (s *StorageSubsystem) Fetcher() *content.Fetcher          { return s.fetcher }
func (s *StorageSubsystem) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

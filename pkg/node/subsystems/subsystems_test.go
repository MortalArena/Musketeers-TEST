package subsystems

import (
	"testing"
	"time"

	"github.com/MortalArena/Musketeers/pkg/content"
	"github.com/MortalArena/Musketeers/pkg/identity"
	"github.com/MortalArena/Musketeers/pkg/search"
	"github.com/MortalArena/Musketeers/pkg/storage"
	"github.com/dgraph-io/badger/v4"
	"github.com/sirupsen/logrus"
)

func TestStorageSubsystem(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions(t.TempDir()).WithLogger(nil))
	if err != nil {
		t.Fatalf("badger.Open returned error: %v", err)
	}
	defer db.Close()
	store := content.NewBadgerBlockStore(db, storage.NewQuotaManager())
	log := logrus.New()
	provider := content.NewProviderManager(nil, nil, store, log)
	fetcher := content.NewFetcher(nil, provider, store, log)
	storage := NewStorageSubsystem(db, store, provider, fetcher)
	if storage.DB() != db || storage.BlockStore() != store || storage.Provider() != provider || storage.Fetcher() != fetcher {
		t.Fatal("unexpected storage subsystem accessors")
	}
}

func TestSecuritySubsystem(t *testing.T) {
	crl := identity.NewCRLCache(24 * time.Hour)
	security := NewSecuritySubsystem(nil, crl, nil, search.NewTokenBucket(1, 2))
	if security.CRL() != crl {
		t.Fatal("unexpected crl")
	}
}

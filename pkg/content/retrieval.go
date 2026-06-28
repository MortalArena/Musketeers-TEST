package content

import (
	"context"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sirupsen/logrus"
)

// Fetcher fetches content from network
type Fetcher struct {
	host     host.Host
	provider *ProviderManager
	store    BlockStore
	log      *logrus.Entry
}

// NewFetcher creates fetcher
func NewFetcher(h host.Host, pm *ProviderManager, store BlockStore, log *logrus.Logger) *Fetcher {
	return &Fetcher{
		host:     h,
		provider: pm,
		store:    store,
		log:      log.WithField("component", "fetcher"),
	}
}

// FetchContent fetches content by CID
func (f *Fetcher) FetchContent(ctx context.Context, cid string, did string) ([]byte, error) {
	// 1. Local search
	data, err := f.store.Get(cid)
	if err == nil {
		return data, nil
	}

	// 2. Search for providers
	providers, err := f.provider.FindProviders(ctx, cid)
	if err != nil {
		return nil, fmt.Errorf("no providers found: %w", err)
	}
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers for %s", cid)
	}

	// 3. Parallel fetch from multiple providers — first valid response wins
	type result struct {
		data []byte
		err  error
	}
	resultCh := make(chan result, len(providers))
	var wg sync.WaitGroup

	for _, pid := range providers {
		wg.Add(1)
		go func(p peer.ID) {
			defer func() {
				if r := recover(); r != nil {
					f.log.WithField("panic", r).Error("content retrieval worker panicked")
				}
			}()
			defer wg.Done()
			data, err := RequestBlock(ctx, f.host, p, cid)
			resultCh <- result{data: data, err: err}
		}(pid)
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				f.log.WithField("panic", r).Error("content retrieval closer panicked")
			}
		}()
		wg.Wait()
		close(resultCh)
	}()

	var lastErr error
	for res := range resultCh {
		if res.err == nil {
			// 4. Local storage and re-provisioning
			if putErr := f.store.Put(cid, res.data, did); putErr != nil {
				f.log.WithError(putErr).Warn("failed to store block locally")
			}
			if provErr := f.provider.AddProvider(ctx, cid); provErr != nil {
				f.log.WithError(provErr).Debug("failed to re-provision")
			}
			return res.data, nil
		}
		lastErr = res.err
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to fetch content: %w", lastErr)
	}
	return nil, fmt.Errorf("failed to fetch content from all providers")
}

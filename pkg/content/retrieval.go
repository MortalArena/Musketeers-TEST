package content

import (
	"context"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sirupsen/logrus"
)

// Fetcher يجلب المحتوى من الشبكة
type Fetcher struct {
	host     host.Host
	provider *ProviderManager
	store    BlockStore
	log      *logrus.Entry
}

// NewFetcher ينشئ fetcher
func NewFetcher(h host.Host, pm *ProviderManager, store BlockStore, log *logrus.Logger) *Fetcher {
	return &Fetcher{
		host:     h,
		provider: pm,
		store:    store,
		log:      log.WithField("component", "fetcher"),
	}
}

// FetchContent يجلب محتوى بالـ CID
func (f *Fetcher) FetchContent(ctx context.Context, cid string, did string) ([]byte, error) {
	// 1. بحث محلي
	data, err := f.store.Get(cid)
	if err == nil {
		return data, nil
	}

	// 2. بحث عن موفرين
	providers, err := f.provider.FindProviders(ctx, cid)
	if err != nil {
		return nil, fmt.Errorf("لم يُعثر على موفرين: %w", err)
	}
	if len(providers) == 0 {
		return nil, fmt.Errorf("لا يوجد موفرون لـ %s", cid)
	}

	// 3. جلب متوازٍ من عدة موفرين — أول استجابة صالحة تفوز
	type result struct {
		data []byte
		err  error
	}
	resultCh := make(chan result, len(providers))
	var wg sync.WaitGroup

	for _, pid := range providers {
		wg.Add(1)
		go func(p peer.ID) {
			defer wg.Done()
			data, err := RequestBlock(ctx, f.host, p, cid)
			resultCh <- result{data: data, err: err}
		}(pid)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	var lastErr error
	for res := range resultCh {
		if res.err == nil {
			// 4. تخزين محلي وإعادة توفير
			if putErr := f.store.Put(cid, res.data, did); putErr != nil {
				f.log.WithError(putErr).Warn("فشل تخزين الكتلة محلياً")
			}
			if provErr := f.provider.AddProvider(ctx, cid); provErr != nil {
				f.log.WithError(provErr).Debug("فشل إعادة التوفير")
			}
			return res.data, nil
		}
		lastErr = res.err
	}

	if lastErr != nil {
		return nil, fmt.Errorf("فشل جلب المحتوى: %w", lastErr)
	}
	return nil, fmt.Errorf("فشل جلب المحتوى من جميع الموفرين")
}

package network

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

// DefaultBootstrapPeers العقد الافتراضية للشبكة
// يتم تحميلها من متغيرات البيئة MUSKETEERS_BOOTSTRAP_PEERS إن وجد
// المتغير يجب أن يحتوي على multiaddrs مفصولة بفواصل
//
// مثال:
//
//	MUSKETEERS_BOOTSTRAP_PEERS="/dns4/seed1.example.com/tcp/4001/p2p/QmX...,/dns4/seed2.example.com/tcp/4001/p2p/QmY..."
var DefaultBootstrapPeers = loadBootstrapPeers()

// loadBootstrapPeers يحمّل bootstrap peers من متغير البيئة أو يستخدم القيم الافتراضية الوهمية
func loadBootstrapPeers() []string {
	envPeers := os.Getenv("MUSKETEERS_BOOTSTRAP_PEERS")
	if envPeers != "" {
		peers := strings.Split(envPeers, ",")
		// تنظيف المسافات
		cleaned := make([]string, 0, len(peers))
		for _, p := range peers {
			p = strings.TrimSpace(p)
			if p != "" {
				cleaned = append(cleaned, p)
			}
		}
		if len(cleaned) > 0 {
			return cleaned
		}
	}

	// القيم الافتراضية الوهمية — يجب استبدالها عبر متغير البيئة قبل الإنتاج
	return []string{
		"/dns4/seed1.musketeers.network/tcp/4001/p2p/12D3KooWPrimarySeed1PeerID",
		"/dns4/seed2.musketeers.network/tcp/4001/p2p/12D3KooWPrimarySeed2PeerID",
		"/dns4/seed3.musketeers.network/tcp/4001/p2p/12D3KooWPrimarySeed3PeerID",

		"/dns4/seed-us.musketeers.network/tcp/4001/p2p/12D3KooWBackupUSPeerID",
		"/dns4/seed-eu.musketeers.network/tcp/4001/p2p/12D3KooWBackupEUPeerID",
		"/dns4/seed-asia.musketeers.network/tcp/4001/p2p/12D3KooWBackupAsiaPeerID",

		"/dns4/community1.musketeers.network/tcp/4001/p2p/12D3KooWCommunity1PeerID",
		"/dns4/community2.musketeers.network/tcp/4001/p2p/12D3KooWCommunity2PeerID",
	}
}

// BootstrapManager يدير عملية bootstrap
type BootstrapManager struct {
	host           host.Host
	logger         *logrus.Logger
	peers          []string
	minConnections int
	maxRetries     int
	retryDelay     time.Duration
	mu             sync.RWMutex
	connectedPeers map[peer.ID]bool
	stopChan       chan struct{}
}

// BootstrapConfig إعدادات bootstrap
type BootstrapConfig struct {
	Peers            []string
	MinConnections   int
	MaxRetries       int
	RetryDelay       time.Duration
	PeriodicInterval time.Duration
	EnablePeriodic   bool
}

// DefaultBootstrapConfig الإعدادات الافتراضية
func DefaultBootstrapConfig() *BootstrapConfig {
	cfg := &BootstrapConfig{
		Peers:            DefaultBootstrapPeers,
		MinConnections:   5,
		MaxRetries:       3,
		RetryDelay:       5 * time.Second,
		PeriodicInterval: 5 * time.Minute,
		EnablePeriodic:   true,
	}

	// متغير البيئة MUSKETEERS_MIN_CONNECTIONS
	if v := os.Getenv("MUSKETEERS_MIN_CONNECTIONS"); v != "" {
		if n, err := fmt.Sscanf(v, "%d", &cfg.MinConnections); err == nil && n > 0 {
			// keep parsed value
		}
	}

	return cfg
}

// NewBootstrapManager ينشئ manager جديد
func NewBootstrapManager(h host.Host, logger *logrus.Logger, cfg *BootstrapConfig) *BootstrapManager {
	if cfg == nil {
		cfg = DefaultBootstrapConfig()
	}

	return &BootstrapManager{
		host:           h,
		logger:         logger,
		peers:          cfg.Peers,
		minConnections: cfg.MinConnections,
		maxRetries:     cfg.MaxRetries,
		retryDelay:     cfg.RetryDelay,
		connectedPeers: make(map[peer.ID]bool),
		stopChan:       make(chan struct{}),
	}
}

// Bootstrap يقوم بالاتصال بالعقد الأولية
func (bm *BootstrapManager) Bootstrap(ctx context.Context) error {
	bm.logger.Info("بدء عملية bootstrap",
		bm.logger.WithField("num_peers", len(bm.peers)),
		bm.logger.WithField("min_connections", bm.minConnections))

	var wg sync.WaitGroup
	successChan := make(chan peer.ID, len(bm.peers))
	errorChan := make(chan error, len(bm.peers))

	for _, peerAddr := range bm.peers {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()

			for retry := 0; retry < bm.maxRetries; retry++ {
				peerID, err := bm.connectToPeer(ctx, addr)
				if err == nil {
					successChan <- peerID
					return
				}

				bm.logger.Debug("محاولة فاشلة",
					bm.logger.WithField("addr", addr),
					bm.logger.WithField("retry", retry+1),
					bm.logger.WithError(err))

				if retry < bm.maxRetries-1 {
					time.Sleep(bm.retryDelay)
				}
			}
			errorChan <- fmt.Errorf("فشل الاتصال بـ %s بعد %d محاولات", addr, bm.maxRetries)
		}(peerAddr)
	}

	go func() {
		wg.Wait()
		close(successChan)
		close(errorChan)
	}()

	successCount := 0
	for peerID := range successChan {
		bm.mu.Lock()
		bm.connectedPeers[peerID] = true
		bm.mu.Unlock()
		successCount++
		bm.logger.Info("✅ تم الاتصال", bm.logger.WithField("peer", peerID.String()[:16]))
	}

	for err := range errorChan {
		bm.logger.Warn("⚠️ فشل bootstrap", bm.logger.WithError(err))
	}

	if successCount == 0 {
		return fmt.Errorf("فشل الاتصال بأي عقدة bootstrap")
	}

	bm.logger.Info("✅ اكتمل bootstrap", bm.logger.WithField("connected", successCount))
	return nil
}

// connectToPeer يتصل بعقدة واحدة
func (bm *BootstrapManager) connectToPeer(ctx context.Context, addr string) (peer.ID, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return "", fmt.Errorf("عنوان غير صالح: %w", err)
	}

	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return "", fmt.Errorf("فشل استخراج معلومات العقدة: %w", err)
	}

	if bm.host.Network().Connectedness(info.ID) == 1 {
		return info.ID, nil
	}

	if err := bm.host.Connect(ctx, *info); err != nil {
		return "", fmt.Errorf("فشل الاتصال: %w", err)
	}

	return info.ID, nil
}

// StartPeriodicBootstrap يبدأ bootstrap الدوري
func (bm *BootstrapManager) StartPeriodicBootstrap(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-bm.stopChan:
				return
			case <-ticker.C:
				bm.checkAndReconnect(ctx)
			}
		}
	}()
}

// checkAndReconnect يتحقق من الاتصالات ويعيد الاتصال إذا لزم
func (bm *BootstrapManager) checkAndReconnect(ctx context.Context) {
	currentPeers := bm.host.Network().Peers()

	if len(currentPeers) < bm.minConnections {
		bm.logger.Warn("⚠️ عدد العقد منخفض",
			bm.logger.WithField("current", len(currentPeers)),
			bm.logger.WithField("min", bm.minConnections))

		if err := bm.Bootstrap(ctx); err != nil {
			bm.logger.Error("فشل إعادة bootstrap", bm.logger.WithError(err))
		}
	}
}

// Stop يوقف bootstrap
func (bm *BootstrapManager) Stop() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	select {
	case <-bm.stopChan:
		// Already closed
		return
	default:
		close(bm.stopChan)
	}
}

// GetConnectedPeers يعيد العقد المتصلة
func (bm *BootstrapManager) GetConnectedPeers() []peer.ID {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	peers := make([]peer.ID, 0, len(bm.connectedPeers))
	for p := range bm.connectedPeers {
		peers = append(peers, p)
	}
	return peers
}

// Stats يعرض إحصائيات bootstrap
func (bm *BootstrapManager) Stats() map[string]interface{} {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	return map[string]interface{}{
		"total_peers":     len(bm.peers),
		"connected_peers": len(bm.connectedPeers),
		"network_peers":   len(bm.host.Network().Peers()),
		"min_connections": bm.minConnections,
	}
}

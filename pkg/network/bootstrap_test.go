package network

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/sirupsen/logrus"
)

func TestBootstrapManager(t *testing.T) {
	// إنشاء host للاختبار
	h, err := libp2p.New()
	if err != nil {
		t.Fatalf("فشل إنشاء host: %v", err)
	}
	defer h.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// إنشاء manager
	cfg := &BootstrapConfig{
		Peers:            []string{}, // فارغ للاختبار
		MinConnections:   1,
		MaxRetries:       2,
		RetryDelay:       100 * time.Millisecond,
		PeriodicInterval: 1 * time.Second,
	}

	bm := NewBootstrapManager(h, logger, cfg)

	// اختبار Stats
	stats := bm.Stats()
	if stats["total_peers"].(int) != 0 {
		t.Error("يجب أن يكون 0 peers")
	}
}

func TestBootstrapWithMockPeers(t *testing.T) {
	// إنشاء 3 hosts
	hosts := make([]host.Host, 3)
	for i := 0; i < 3; i++ {
		h, err := libp2p.New()
		if err != nil {
			t.Fatalf("فشل إنشاء host %d: %v", i, err)
		}
		defer h.Close()
		hosts[i] = h
	}

	// بناء bootstrap peers من hosts
	peers := make([]string, 2)
	for i := 1; i < 3; i++ {
		addrs := hosts[i].Addrs()
		if len(addrs) > 0 {
			peers[i-1] = fmt.Sprintf("%s/p2p/%s", addrs[0], hosts[i].ID())
		}
	}

	// اختبار bootstrap
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	cfg := &BootstrapConfig{
		Peers:          peers,
		MinConnections: 2,
		MaxRetries:     3,
		RetryDelay:     100 * time.Millisecond,
	}

	bm := NewBootstrapManager(hosts[0], logger, cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := bm.Bootstrap(ctx)
	if err != nil {
		t.Errorf("فشل bootstrap: %v", err)
	}

	// التحقق من الاتصالات
	connected := hosts[0].Network().Peers()
	if len(connected) < 2 {
		t.Errorf("يجب الاتصال بـ 2 peers، تم %d", len(connected))
	}
}

func TestDefaultBootstrapConfig(t *testing.T) {
	cfg := DefaultBootstrapConfig()

	if cfg.MinConnections != 5 {
		t.Errorf("MinConnections يجب أن يكون 5، حصلت على %d", cfg.MinConnections)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries يجب أن يكون 3، حصلت على %d", cfg.MaxRetries)
	}
	if len(cfg.Peers) == 0 {
		t.Error("يجب أن يكون هناك peers افتراضية")
	}
}

func TestGetConnectedPeers(t *testing.T) {
	h, err := libp2p.New()
	if err != nil {
		t.Fatalf("فشل إنشاء host: %v", err)
	}
	defer h.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	cfg := DefaultBootstrapConfig()
	bm := NewBootstrapManager(h, logger, cfg)

	peers := bm.GetConnectedPeers()
	if len(peers) != 0 {
		t.Error("يجب أن يكون 0 peers متصلة")
	}
}

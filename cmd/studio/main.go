package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent_bridge"
	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
	"github.com/MortalArena/Musketeers/pkg/identity"
	"github.com/MortalArena/Musketeers/pkg/node"
	"github.com/MortalArena/Musketeers/pkg/storage"
	"github.com/sirupsen/logrus"
)

var (
	addr       = flag.String("addr", "127.0.0.1:5000", "Studio server address")
	agentAddr  = flag.String("agent-addr", "127.0.0.1:5001", "Agent bridge address")
	dataDir    = flag.String("data-dir", "./studio-data", "Data directory")
	bootstrap  = flag.String("bootstrap", "", "Bootstrap peer multiaddr")
	founderPub = flag.String("founder-pub", "", "Founder public key hex")
	verbose    = flag.Bool("verbose", false, "Verbose logging")
)

func main() {
	flag.Parse()

	log := logrus.New()
	if *verbose {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}
	log.SetFormatter(&logrus.JSONFormatter{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// إنشاء مفاتيح
	kp, err := nrcrypto.GenerateKeyPair()
	if err != nil {
		log.WithError(err).Fatal("Failed to generate key pair")
	}

	// إنشاء سجل هوية
	idRec := &identity.IdentityRecord{
		DID:       kp.DID,
		CreatedAt: time.Now().Unix(),
	}

	// إنشاء عقدة
	cfg := &node.Config{
		DataDir:        *dataDir,
		ListenPort:     4001,
		StorageQuotaMB: 2048, // 2GB
		FounderPubHex:  *founderPub,
		BootstrapPeers: parseBootstrap(*bootstrap),
	}

	n, err := node.New(ctx, cfg, kp, idRec)
	if err != nil {
		log.WithError(err).Fatal("Failed to create node")
	}
	defer n.Close()

	log.WithField("did", kp.DID).Info("Studio node created")

	// نشر الهوية على DHT
	if err := n.PublishIdentity(ctx); err != nil {
		log.WithError(err).Warn("Failed to publish identity")
	}

	// إنشاء QuotaManager
	qm := storage.NewQuotaManager()
	qm.SetLimit(kp.DID, 2*1024*1024*1024) // 2GB لـ Studio

	// إنشاء مدير الجلسات
	sessionMgr := agent_bridge.NewSessionManager(log)

	// إنشاء الجسر المتعدد
	multiplexedBrg := agent_bridge.NewMultiplexedBridge(log)

	// إنشاء خادم الجسر
	bridgeServer := agent_bridge.NewServer(*agentAddr, sessionMgr, multiplexedBrg, log)
	if err := bridgeServer.Start(ctx); err != nil {
		log.WithError(err).Fatal("Failed to start bridge server")
	}
	defer bridgeServer.Stop()

	log.WithField("addr", *agentAddr).Info("Agent bridge server started")

	// بدء واجهة Studio
	log.WithField("addr", *addr).Info("Studio starting...")

	// في التنفيذ الحالي، سنبدأ فقط الخدمات الأساسية
	// في المستقبل، سيتم إضافة واجهة ويب/CLI كاملة

	// انتظار إشارة الإنهاء
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Info("Studio shutting down...")
}

// parseBootstrap يحلل عناوين bootstrap
func parseBootstrap(bootstrap string) []string {
	if bootstrap == "" {
		return nil
	}
	return []string{bootstrap}
}

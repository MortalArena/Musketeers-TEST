package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/MortalArena/Musketeers/pkg/agent_bridge"
	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
	"github.com/sirupsen/logrus"
)

func main() {
	bridgeAddr := flag.String("bridge", "127.0.0.1:5001", "Agent Bridge address")
	flag.Parse()

	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ✅ إنشاء مفاتيح للوكيل
	kp, err := nrcrypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("فشل توليد المفاتيح: %v", err)
	}

	// ✅ الاتصال بـ Agent Bridge (لا ينشئ عقدة جديدة!)
	client := agent_bridge.NewClient(*bridgeAddr, log)
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("فشل الاتصال بالجسر: %v", err)
	}
	defer client.Disconnect()

	log.WithFields(logrus.Fields{
		"did":       kp.DID,
		"bridge":    *bridgeAddr,
		"connected": client.IsConnected(),
	}).Info("Agent متصل بـ Studio Bridge")

	// في التنفيذ الحالي، الوكيل ينتظر فقط
	// في المستقبل، سيتم إضافة منطق تنفيذ المهام

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Info("إيقاف Agent...")
}

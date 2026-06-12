package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
	"github.com/MortalArena/Musketeers/pkg/identity"
	"github.com/MortalArena/Musketeers/pkg/node"
	"github.com/sirupsen/logrus"
)

func main() {
	port := flag.Int("port", 4001, "منفذ الاستماع")
	dataDir := flag.String("data", "./seed-data", "مجلد البيانات")
	flag.Parse()

	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	cfg := node.DefaultConfig()
	cfg.ListenPort = *port
	cfg.DataDir = *dataDir
	cfg.EnableMDNS = true

	kp, err := nrcrypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("فشل توليد المفاتيح: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	powCtx, powCancel := context.WithTimeout(ctx, 5*time.Minute)
	idRec, err := identity.NewIdentityRecord(powCtx, kp, []string{"bootstrap"}, nrcrypto.DefaultIdentityTTL)
	powCancel()
	if err != nil {
		log.Fatalf("فشل إنشاء سجل الهوية: %v", err)
	}

	n, err := node.New(ctx, cfg, kp, idRec)
	if err != nil {
		log.Fatalf("فشل إنشاء عقدة البذرة: %v", err)
	}
	defer n.Close()

	if err := n.PublishIdentity(ctx); err != nil {
		log.WithError(err).Warn("فشل نشر الهوية")
	}

	addrs := n.Addrs()
	log.Info("عقدة البذرة جاهزة — استخدم أحد العناوين التالية للـ bootstrap:")
	for _, addr := range addrs {
		fmt.Println(addr)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Info("إيقاف عقدة البذرة...")
}

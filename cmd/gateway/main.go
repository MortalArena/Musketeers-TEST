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
	"github.com/MortalArena/Musketeers/pkg/gateway"
	"github.com/MortalArena/Musketeers/pkg/identity"
	"github.com/MortalArena/Musketeers/pkg/node"
	"github.com/sirupsen/logrus"
)

func main() {
	port := flag.Int("port", 8090, "منفذ HTTP Gateway")
	dataDir := flag.String("data", "./gateway-data", "مجلد البيانات")
	bootstrap := flag.String("bootstrap", "", "عنوان bootstrap peer")
	founderPub := flag.String("founder-pub", "", "مفتاح المؤسس العام (hex)")
	listenPort := flag.Int("p2p-port", 4010, "منفذ libp2p")
	tls := flag.Bool("tls", false, "تمكين TLS (HTTPS)")
	cert := flag.String("cert", "", "مسار ملف شهادة TLS (اختياري)")
	key := flag.String("key", "", "مسار ملف مفتاح TLS (اختياري)")
	flag.Parse()

	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	cfg := node.DefaultConfig()
	cfg.DataDir = *dataDir
	cfg.ListenPort = *listenPort
	cfg.EnableMDNS = false
	if *bootstrap != "" {
		cfg.BootstrapPeers = append(cfg.BootstrapPeers, *bootstrap)
	}
	if *founderPub != "" {
		cfg.FounderPubHex = *founderPub
	}

	kp, err := nrcrypto.GenerateKeyPair()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	powCtx, powCancel := context.WithTimeout(ctx, 5*time.Minute)
	idRec, err := identity.NewIdentityRecord(powCtx, kp, []string{"gateway"}, nrcrypto.DefaultIdentityTTL)
	powCancel()
	if err != nil {
		log.Fatalf("فشل إنشاء الهوية: %v", err)
	}

	n, err := node.New(ctx, cfg, kp, idRec)
	if err != nil {
		log.Fatal(err)
	}
	defer n.Close()

	gw := gateway.NewServer(n, *port, log)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.WithField("panic", r).Error("gateway server goroutine panicked")
			}
		}()
		var err error
		if *tls {
			err = gw.StartTLS(*cert, *key)
		} else {
			err = gw.Start()
		}
		if err != nil {
			log.WithError(err).Error("Gateway توقف")
		}
	}()

	scheme := "http"
	if *tls {
		scheme = "https"
	}
	log.WithFields(logrus.Fields{
		"gateway": fmt.Sprintf("%s://127.0.0.1:%d", scheme, *port),
		"usage":   "/d/example.ia/",
	}).Info("HTTP Gateway جاهزة")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Info("إيقاف Gateway...")
}

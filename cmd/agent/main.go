package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/MortalArena/Musketeers/api"
	"github.com/MortalArena/Musketeers/pkg/agent_bridge"
	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
	"github.com/MortalArena/Musketeers/pkg/identity"
	"github.com/MortalArena/Musketeers/pkg/naming"
	"github.com/MortalArena/Musketeers/pkg/node"
	"github.com/MortalArena/Musketeers/pkg/storage"
	"github.com/sirupsen/logrus"
)

func main() {
	mnemonic := flag.String("mnemonic", "", "عبارة تذكيرية BIP39 (24 كلمة)")
	passphrase := flag.String("passphrase", "", "عبارة مرور اختيارية")
	dataDir := flag.String("data", "", "مجلد البيانات")
	port := flag.Int("port", 0, "منفذ الاستماع")
	restPort := flag.Int("rest", 0, "منفذ REST API (0 = تعطيل)")
	bridgePort := flag.Int("bridge", 5001, "منفذ Agent Bridge")
	bootstrap := flag.String("bootstrap", "", "عنوان bootstrap peer")
	initKeystore := flag.Bool("init", false, "إنشاء keystore جديد مع mnemonic")
	commitDomain := flag.String("commit-domain", "", "نشر التزام لنطاق (commit-reveal)")
	commitSecret := flag.String("commit-secret", "", "السر للتزام (فارغ = توليد تلقائي)")
	flag.Parse()

	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	cfg := node.LoadFromEnv()
	if *dataDir != "" {
		cfg.DataDir = *dataDir
	}
	if *port > 0 {
		cfg.ListenPort = *port
	}
	if *restPort > 0 {
		cfg.RESTPort = *restPort
	}
	if *bootstrap != "" {
		cfg.BootstrapPeers = append(cfg.BootstrapPeers, *bootstrap)
	}

	kp, savedMnemonic, err := loadIdentity(cfg.DataDir, *mnemonic, *passphrase, *initKeystore, log)
	if err != nil {
		log.Fatalf("فشل تحميل الهوية: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	powCtx, powCancel := context.WithTimeout(ctx, 5*time.Minute)
	idRec, err := identity.NewIdentityRecord(powCtx, kp, []string{"acp/v1", "lang/json"}, nrcrypto.DefaultIdentityTTL)
	powCancel()
	if err != nil {
		log.Fatalf("فشل إنشاء سجل الهوية: %v", err)
	}

	n, err := node.New(ctx, cfg, kp, idRec)
	if err != nil {
		log.Fatalf("فشل إنشاء العقدة: %v", err)
	}
	defer n.Close()

	if err := n.PublishIdentity(ctx); err != nil {
		log.WithError(err).Warn("فشل نشر الهوية على DHT")
	}

	// commit-reveal للنطاق
	if *commitDomain != "" {
		secret := *commitSecret
		if secret == "" {
			secret, err = naming.GenerateSecret()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("السر (احفظه): %s\n", secret)
		}
		commit, err := n.PublishDomainCommit(ctx, *commitDomain, kp.DID, secret)
		if err != nil {
			log.Fatalf("فشل نشر التزام: %v", err)
		}
		fmt.Printf("التزام: %s\nانتظر %v ثم اطلب التسجيل من المؤسس\n", commit.Commitment, "60s")
	}

	log.WithFields(logrus.Fields{
		"did":   kp.DID,
		"peer":  n.Host().ID().String(),
		"addrs": n.Addrs(),
		"acp":   n.SupportedACPTasks(),
	}).Info("عقدة Musketeers جاهزة")

	if savedMnemonic != "" {
		log.Warn("احفظ عبارتك التذكيرية في مكان آمن — لن تُعرض مرة أخرى")
	}

	if cfg.RESTPort > 0 {
		srv := api.NewServer(n, cfg.RESTPort, log)
		log.WithField("token", srv.LocalToken()).Info("REST API token (محلي فقط)")
		go func() {
			if err := srv.Start(); err != nil {
				log.WithError(err).Error("REST API توقف")
			}
		}()
	}

	// ✅ إعداد Agent Bridge
	qm := storage.NewQuotaManager()
	qm.SetLimit(kp.DID, 2*1024*1024*1024) // 2GB لكل وكيل

	sessionMgr := agent_bridge.NewSessionManager(log)
	multiplexedBrg := agent_bridge.NewMultiplexedBridge(log)
	bridgeAddr := fmt.Sprintf("127.0.0.1:%d", *bridgePort)
	bridgeServer := agent_bridge.NewServer(bridgeAddr, sessionMgr, multiplexedBrg, log)

	if err := bridgeServer.Start(ctx); err != nil {
		log.WithError(err).Fatal("فشل بدء Agent Bridge")
	}
	defer bridgeServer.Stop()

	log.WithField("addr", bridgeAddr).Info("Agent Bridge بدأ")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Info("إيقاف العقدة...")
}

func loadIdentity(dataDir, mnemonic, passphrase string, init bool, log *logrus.Logger) (*nrcrypto.KeyPair, string, error) {
	ksPath := nrcrypto.KeystorePath(dataDir)

	if mnemonic != "" {
		priv, err := nrcrypto.IdentityFromMnemonic(mnemonic, passphrase)
		if err != nil {
			return nil, "", err
		}
		kp := nrcrypto.KeyPairFromPrivate(priv)
		if init || !nrcrypto.KeystoreExists(dataDir) {
			pass := passphrase
			if pass == "" {
				pass = promptPassphrase("أدخل عبارة مرور لحماية keystore: ")
			}
			if err := nrcrypto.SaveKeystore(ksPath, pass, kp, mnemonic); err != nil {
				return nil, "", err
			}
			log.Info("تم حفظ keystore مشفّر")
		}
		return kp, "", nil
	}

	if nrcrypto.KeystoreExists(dataDir) {
		pass := passphrase
		if pass == "" {
			pass = promptPassphrase("عبارة مرور keystore: ")
		}
		kp, _, err := nrcrypto.LoadKeystore(ksPath, pass)
		return kp, "", err
	}

	if init {
		m, err := nrcrypto.GenerateMnemonic()
		if err != nil {
			return nil, "", err
		}
		fmt.Printf("عبارة تذكيرية (احفظها): %s\n", m)
		priv, err := nrcrypto.IdentityFromMnemonic(m, passphrase)
		if err != nil {
			return nil, "", err
		}
		kp := nrcrypto.KeyPairFromPrivate(priv)
		pass := passphrase
		if pass == "" {
			pass = promptPassphrase("أدخل عبارة مرور لحماية keystore: ")
		}
		if err := nrcrypto.SaveKeystore(ksPath, pass, kp, m); err != nil {
			return nil, "", err
		}
		return kp, m, nil
	}

	kp, err := nrcrypto.GenerateKeyPair()
	if err != nil {
		return nil, "", err
	}
	log.Warn("تم توليد مفاتيح مؤقتة — استخدم -init لإنشاء keystore دائم")
	return kp, "", nil
}

func promptPassphrase(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

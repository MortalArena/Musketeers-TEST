package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
	"github.com/MortalArena/Musketeers/pkg/identity"
	"github.com/MortalArena/Musketeers/pkg/naming"
	"github.com/MortalArena/Musketeers/pkg/node"
	"github.com/sirupsen/logrus"
)

func main() {
	action := flag.String("action", "register", "الإجراء: register, reveal-register, renew")
	domain := flag.String("domain", "", "اسم النطاق (مثال: example.ia)")
	owner := flag.String("owner", "", "DID المالك")
	target := flag.String("target", "", "الهدف (DID أو CID)")
	secret := flag.String("secret", "", "سر commit-reveal")
	commitment := flag.String("commitment", "", "hash التزام (hex)")
	expires := flag.Int64("expires", 0, "تاريخ انتهاء Unix (0 = سنة من الآن)")
	founderKeyHex := flag.String("founder-key", "", "مفتاح المؤسس الخاص (hex)")
	dataDir := flag.String("data", "./founder-data", "مجلد البيانات")
	flag.Parse()

	log := logrus.New()

	if *founderKeyHex == "" {
		log.Fatal("مفتاح المؤسس مطلوب: -founder-key")
	}
	keyBytes, err := hex.DecodeString(*founderKeyHex)
	if err != nil || len(keyBytes) != ed25519.PrivateKeySize {
		log.Fatal("مفتاح المؤسس غير صالح")
	}
	founderPriv := ed25519.PrivateKey(keyBytes)
	founderPub := founderPriv.Public().(ed25519.PublicKey)

	cfg := node.DefaultConfig()
	cfg.DataDir = *dataDir
	cfg.FounderPubHex = hex.EncodeToString(founderPub)

	kp := nrcrypto.KeyPairFromPrivate(founderPriv)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	powCtx, powCancel := context.WithTimeout(ctx, 5*time.Minute)
	idRec, err := identity.NewIdentityRecord(powCtx, kp, []string{"founder"}, nrcrypto.DefaultIdentityTTL)
	powCancel()
	if err != nil {
		log.Fatalf("فشل إنشاء هوية المؤسس: %v", err)
	}

	n, err := node.New(ctx, cfg, kp, idRec)
	if err != nil {
		log.Fatalf("فشل إنشاء عقدة المؤسس: %v", err)
	}
	defer n.Close()

	exp := *expires
	if exp == 0 {
		exp = time.Now().Add(365 * 24 * time.Hour).Unix()
	}

	switch *action {
	case "register":
		if *domain == "" || *owner == "" {
			log.Fatal("النطاق و DID المالك مطلوبان")
		}
		rec, err := naming.NewDomainRecord(*domain, *owner, *target, "did", exp, founderPriv)
		if err != nil {
			log.Fatalf("فشل إنشاء سجل النطاق: %v", err)
		}
		data, err := json.Marshal(rec)
		if err != nil {
			log.Fatal(err)
		}
		if err := n.DHT().PutValue(ctx, rec.DHTKey(), data); err != nil {
			log.Fatalf("فشل نشر النطاق: %v", err)
		}
		fmt.Printf("تم تسجيل %s للمالك %s\n", rec.Name, rec.Owner)

	case "reveal-register":
		// تسجيل آمن عبر commit-reveal
		if *domain == "" || *owner == "" || *secret == "" {
			log.Fatal("النطاق والمالك والسر مطلوبان")
		}
		rec, err := n.RegisterDomainReveal(ctx, *domain, *owner, *secret, *target, "did", exp, founderPriv)
		if err != nil {
			log.Fatalf("فشل reveal-register: %v", err)
		}
		fmt.Printf("تم تسجيل %s عبر commit-reveal للمالك %s\n", rec.Name, rec.Owner)

	case "verify-commit":
		if *commitment == "" {
			log.Fatal("hash التزام مطلوب")
		}
		commit, err := n.GetDomainCommit(ctx, *commitment)
		if err != nil {
			log.Fatalf("التزام غير موجود: %v", err)
		}
		out, _ := json.MarshalIndent(commit, "", "  ")
		fmt.Println(string(out))

	case "renew":
		if *domain == "" {
			log.Fatal("اسم النطاق مطلوب")
		}
		rec, err := n.ResolveDomain(ctx, *domain)
		if err != nil {
			log.Fatalf("فشل حل النطاق: %v", err)
		}
		newExp := time.Now().Add(365 * 24 * time.Hour).Unix()
		renewed, err := naming.NewDomainRecord(rec.Name, rec.Owner, rec.Target, rec.Type, newExp, founderPriv)
		if err != nil {
			log.Fatal(err)
		}
		renewed.Version = rec.Version
		renewed.OwnerSig = rec.OwnerSig
		renewed.ManifestCID = rec.ManifestCID
		renewed.Providers = rec.Providers
		data, _ := json.Marshal(renewed)
		if err := n.DHT().PutValue(ctx, renewed.DHTKey(), data); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("تم تجديد %s حتى %d\n", renewed.Name, renewed.ExpiresAt)

	default:
		fmt.Fprintf(os.Stderr, "إجراء غير معروف: %s\n", *action)
		os.Exit(1)
	}
}

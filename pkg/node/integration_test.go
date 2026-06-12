package node_test

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
	"github.com/MortalArena/Musketeers/pkg/identity"
	"github.com/MortalArena/Musketeers/pkg/naming"
	"github.com/MortalArena/Musketeers/pkg/node"
	"github.com/MortalArena/Musketeers/pkg/protocol"
)

func init() {
	os.Setenv("NR_POW_DIFFICULTY", "1")
}

func parseAddrInfo(addr string) (*peer.AddrInfo, error) {
	ma, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return nil, err
	}
	return peer.AddrInfoFromP2pAddr(ma)
}

func startNode(t *testing.T, ctx context.Context, port int, dataDir string, founderPubHex string, caps []string) (*node.Node, *nrcrypto.KeyPair) {
	t.Helper()
	kp, err := nrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	cfg := node.DefaultConfig()
	cfg.ListenPort = port
	cfg.DataDir = dataDir
	cfg.EnableMDNS = false
	cfg.FounderPubHex = founderPubHex

	powCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	idRec, err := identity.NewIdentityRecord(powCtx, kp, caps, 3600)
	cancel()
	if err != nil {
		t.Skipf("PoW skipped: %v", err)
	}

	n, err := node.New(ctx, cfg, kp, idRec)
	if err != nil {
		t.Fatal(err)
	}
	_ = n.PublishIdentity(ctx)
	return n, kp
}

func TestTwoNodesACP(t *testing.T) {
	if testing.Short() {
		t.Skip("تخطي اختبار التكامل في الوضع السريع")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	tmp := t.TempDir()
	n1, kp1 := startNode(t, ctx, 14101, tmp+"/n1", "", []string{"acp/v1"})
	defer n1.Close()
	n2, _ := startNode(t, ctx, 14102, tmp+"/n2", "", []string{"acp/v1"})
	defer n2.Close()

	addrs := n1.Addrs()
	info, err := parseAddrInfo(addrs[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := n2.Host().Connect(ctx, *info); err != nil {
		t.Fatalf("connect failed: %v", err)
	}

	time.Sleep(2 * time.Second)
	_ = n1.PublishIdentity(ctx)

	resp, err := n2.SendACPTask(ctx, n1.Host().ID(), kp1.DID, "ping", nil)
	if err != nil {
		t.Fatalf("ACP ping failed: %v", err)
	}
	if resp.Intent != "task.response" {
		t.Fatalf("unexpected intent: %s", resp.Intent)
	}
}

func TestCommitRevealIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("تخطي اختبار التكامل في الوضع السريع")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	_, founderPriv, _ := ed25519.GenerateKey(nil)
	founderPubHex := hex.EncodeToString(founderPriv.Public().(ed25519.PublicKey))

	tmp := t.TempDir()
	nOwner, kpOwner := startNode(t, ctx, 14201, tmp+"/owner", founderPubHex, []string{"acp/v1"})
	defer nOwner.Close()
	nFounder, _ := startNode(t, ctx, 14202, tmp+"/founder", founderPubHex, []string{"founder"})
	defer nFounder.Close()

	// ربط العقدتين
	info, err := parseAddrInfo(nOwner.Addrs()[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := nFounder.Host().Connect(ctx, *info); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	time.Sleep(2 * time.Second)

	domain := "testsite.ia"
	secret, err := naming.GenerateSecret()
	if err != nil {
		t.Fatal(err)
	}

	// تزام بوقت قديم (تجاوز فترة الانتظار)
	committedAt := time.Now().Add(-2 * naming.MinRevealDelay).Unix()
	commit, err := naming.CommitRecordWithTime(domain, kpOwner.DID, secret, committedAt)
	if err != nil {
		t.Fatal(err)
	}
	if err := nOwner.PutDomainCommit(ctx, commit); err != nil {
		t.Fatalf("put commit failed: %v", err)
	}

	// المؤسس يسجّل عبر reveal
	rec, err := nFounder.RegisterDomainReveal(ctx, domain, kpOwner.DID, secret, "did:ia:target", "did",
		time.Now().Add(365*24*time.Hour).Unix(), founderPriv)
	if err != nil {
		t.Fatalf("reveal-register failed: %v", err)
	}
	if rec.Name != domain {
		t.Fatalf("expected %s, got %s", domain, rec.Name)
	}

	// المالك يحل النطاق
	resolved, err := nOwner.ResolveDomain(ctx, domain)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if resolved.Owner != kpOwner.DID {
		t.Fatalf("owner mismatch: %s", resolved.Owner)
	}
}

func TestGatewaySiteFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("تخطي اختبار التكامل في الوضع السريع")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	_, founderPriv, _ := ed25519.GenerateKey(nil)
	founderPubHex := hex.EncodeToString(founderPriv.Public().(ed25519.PublicKey))

	tmp := t.TempDir()
	nOwner, kp := startNode(t, ctx, 14301, tmp+"/owner", founderPubHex, []string{"hosting"})
	defer nOwner.Close()
	nPeer, _ := startNode(t, ctx, 14302, tmp+"/peer", founderPubHex, []string{"bootstrap"})
	defer nPeer.Close()

	info, _ := parseAddrInfo(nOwner.Addrs()[0])
	if err := nPeer.Host().Connect(ctx, *info); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)

	html := []byte("<html><body><h1>Hello .ia</h1></body></html>")
	htmlCID, err := nOwner.PublishContent(ctx, html)
	if err != nil {
		t.Fatal(err)
	}

	manifest := protocol.SiteManifest{
		Version: 1,
		Title:   "Test",
		Files:   map[string]string{"index.html": htmlCID},
	}
	manifestData, _ := json.Marshal(manifest)
	manifestCID, err := nOwner.PublishContent(ctx, manifestData)
	if err != nil {
		t.Fatal(err)
	}

	domain := "hello.ia"
	exp := time.Now().Add(365 * 24 * time.Hour).Unix()
	drec, err := naming.NewDomainRecord(domain, kp.DID, "cid:"+manifestCID, "cid", exp, founderPriv)
	if err != nil {
		t.Fatal(err)
	}
	drec.ManifestCID = manifestCID
	if err := drec.SignOwner(kp.Private); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(drec)
	if err := nOwner.DHT().PutValue(ctx, drec.DHTKey(), data); err != nil {
		t.Fatal(err)
	}

	resolved, err := nOwner.ResolveDomain(ctx, domain)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.ManifestCID != manifestCID {
		t.Fatalf("manifest cid mismatch")
	}

	fetched, err := nOwner.FetchContent(ctx, resolved.ManifestCID)
	if err != nil {
		t.Fatal(err)
	}
	var m protocol.SiteManifest
	json.Unmarshal(fetched, &m)
	if m.Files["index.html"] != htmlCID {
		t.Fatal("manifest file cid mismatch")
	}
}

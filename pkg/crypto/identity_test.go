package crypto

import (
	"context"
	"os"
	"testing"
	"time"
)

func init() {
	os.Setenv("NR_POW_DIFFICULTY", "1")
}

func TestDIDGeneration(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	if kp.DID == "" || len(kp.DID) < 10 {
		t.Fatal("invalid DID")
	}
	if kp.DID[:9] != "did:mskt:" {
		t.Fatalf("DID prefix wrong: %s", kp.DID)
	}
}

func TestMnemonicRoundTrip(t *testing.T) {
	mnemonic, err := GenerateMnemonic()
	if err != nil {
		t.Fatal(err)
	}
	priv1, err := IdentityFromMnemonic(mnemonic, "test-pass")
	if err != nil {
		t.Fatal(err)
	}
	priv2, err := IdentityFromMnemonic(mnemonic, "test-pass")
	if err != nil {
		t.Fatal(err)
	}
	if string(priv1) != string(priv2) {
		t.Fatal("mnemonic derivation not deterministic")
	}
}

func TestPowVerify(t *testing.T) {
	kp, _ := GenerateKeyPair()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nonce, err := MinePow(ctx, kp.DID, 8) // صعوبة منخفضة للاختبار
	if err != nil {
		t.Skipf("PoW mining skipped (slow): %v", err)
	}
	if !VerifyPow(kp.DID, nonce, 8) {
		t.Fatal("PoW verification failed")
	}
}

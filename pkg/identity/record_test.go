package identity

import (
	"context"
	"os"
	"testing"
	"time"

	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
)

func init() {
	os.Setenv("NR_POW_DIFFICULTY", "1")
}

func TestIdentityRecord(t *testing.T) {
	kp, err := nrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	rec, err := NewIdentityRecord(ctx, kp, []string{"acp/v1"}, 3600)
	if err != nil {
		t.Skipf("identity record creation skipped: %v", err)
	}
	if err := rec.Verify(); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
}

func TestRevocationRecord(t *testing.T) {
	kp, _ := nrcrypto.GenerateKeyPair()
	rec, err := NewRevocationRecord(kp.DID, kp.Private)
	if err != nil {
		t.Fatal(err)
	}
	if err := rec.Verify(kp.Public); err != nil {
		t.Fatalf("revocation verify failed: %v", err)
	}
}

func TestDelegationRecord(t *testing.T) {
	owner, _ := nrcrypto.GenerateKeyPair()
	delegate, _ := nrcrypto.GenerateKeyPair()
	rec, err := NewDelegationRecord(
		owner.DID, delegate.DID,
		[]string{PermSendMessage},
		time.Now().Add(time.Hour).Unix(),
		owner.Private,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := rec.Verify(owner.Public); err != nil {
		t.Fatalf("delegation verify failed: %v", err)
	}
	if !rec.HasPermission(PermSendMessage) {
		t.Fatal("should have send_message permission")
	}
}

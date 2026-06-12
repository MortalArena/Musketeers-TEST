package channel

import (
	"crypto/ed25519"
	"testing"

	"github.com/MortalArena/Musketeers/pkg/protocol"
)

func TestPrivateChannelLifecycle(t *testing.T) {
	ownerPub, ownerPriv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	m1Pub, m1Priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	m2Pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	ownerDID := "did:ia:owner"
	m1DID := "did:ia:member1"
	m2DID := "did:ia:member2"

	// 1. Create channel
	cfg, aesKey, err := NewPrivateChannel("chan-1", ownerDID, ownerPriv, []string{}, nil)
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	// 2. Decrypt shared key for owner
	decryptedKey, err := DecryptSharedKey(cfg.SharedKey, ownerPriv)
	if err != nil {
		t.Fatalf("failed to decrypt shared key for owner: %v", err)
	}
	if string(decryptedKey) != string(aesKey) {
		t.Fatal("decrypted AES key mismatch for owner")
	}

	// 3. Add member 2
	err = cfg.AddMember(m2DID, m2Pub, ownerDID, ownerPriv, aesKey)
	if err != nil {
		t.Fatalf("failed to add member: %v", err)
	}

	// 4. Verify member key encryption and decryption for member 1 (added during creation)
	// Wait, NewPrivateChannel does not automatically encrypt for members in NewPrivateChannel,
	// let's check: does NewPrivateChannel call AddMember internally? No, NewPrivateChannel just
	// sets Members, but MemberKeys is empty except if we call AddMember.
	// Let's add member 1 explicitly as well.
	err = cfg.AddMember(m1DID, m1Pub, ownerDID, ownerPriv, aesKey)
	if err != nil {
		t.Fatalf("failed to add member 1: %v", err)
	}

	m1EncKey, ok := cfg.MemberKeyFor(m1DID)
	if !ok {
		t.Fatal("member 1 key not found")
	}
	m1DecryptedKey, err := DecryptSharedKey(m1EncKey, m1Priv)
	if err != nil {
		t.Fatalf("failed to decrypt member 1 key: %v", err)
	}
	if string(m1DecryptedKey) != string(aesKey) {
		t.Fatal("decrypted AES key mismatch for member 1")
	}

	// 5. Encrypt and decrypt a message
	plainText := &protocol.PrivatePlaintext{
		From:    ownerDID,
		Content: "Welcome to the private channel!",
	}
	encMsg, err := EncryptPrivateMessage("chan-1", plainText, aesKey)
	if err != nil {
		t.Fatalf("failed to encrypt message: %v", err)
	}

	decryptedMsg, err := DecryptPrivateMessage("chan-1", encMsg, m1DecryptedKey)
	if err != nil {
		t.Fatalf("failed to decrypt message for member 1: %v", err)
	}
	if decryptedMsg.Content != plainText.Content {
		t.Errorf("decrypted message mismatch: expected %q, got %q", plainText.Content, decryptedMsg.Content)
	}

	// 6. Verify Signature
	if err := cfg.Verify(ownerPub); err != nil {
		t.Fatalf("failed to verify channel config signature: %v", err)
	}
}

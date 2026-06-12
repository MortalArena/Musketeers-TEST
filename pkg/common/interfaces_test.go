package common

import (
	"crypto/ed25519"
	"testing"
)

func TestKeyResolverContract(t *testing.T) {
	var resolver KeyResolver = mockResolver{}
	pub, err := resolver.ResolvePublicKey("did:ia:test")
	if err != nil {
		t.Fatalf("ResolvePublicKey returned error: %v", err)
	}
	if len(pub) != ed25519.PublicKeySize {
		t.Fatalf("unexpected public key size: %d", len(pub))
	}
}

type mockResolver struct{}

func (mockResolver) ResolvePublicKey(string) (ed25519.PublicKey, error) {
	return make(ed25519.PublicKey, ed25519.PublicKeySize), nil
}

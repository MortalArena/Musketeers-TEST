package adapters

import (
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"

	"github.com/MortalArena/Musketeers/pkg/node"
	"github.com/MortalArena/Musketeers/pkg/sdk/interfaces"
	"github.com/MortalArena/Musketeers/pkg/vault/encryption"
)

type IdentityAdapter struct {
	node *node.Node
}

func NewIdentityAdapter(n *node.Node) *IdentityAdapter {
	return &IdentityAdapter{node: n}
}

func (a *IdentityAdapter) DID() string {
	return a.node.KeyPair().DID
}

func (a *IdentityAdapter) Sign(data []byte) ([]byte, error) {
	return ed25519.Sign(a.node.KeyPair().Private, data), nil
}

func (a *IdentityAdapter) Verify(data, sig []byte) error {
	if !ed25519.Verify(a.node.KeyPair().Public, data, sig) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

func (a *IdentityAdapter) ResolvePublicKey(did string) ([]byte, error) {
	pub, err := a.node.ResolvePublicKey(did)
	if err != nil {
		return nil, err
	}
	return []byte(pub), nil
}

func (a *IdentityAdapter) Encrypt(plain []byte) ([]byte, error) {
	key := sha256.Sum256(a.node.KeyPair().Private)
	return encryption.Encrypt(plain, key[:])
}

func (a *IdentityAdapter) Decrypt(cipher []byte) ([]byte, error) {
	key := sha256.Sum256(a.node.KeyPair().Private)
	return encryption.Decrypt(cipher, key[:])
}

var _ interfaces.IdentityInterface = (*IdentityAdapter)(nil)

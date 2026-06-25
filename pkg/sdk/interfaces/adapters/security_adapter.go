package adapters

import (
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"

	"github.com/MortalArena/Musketeers/pkg/node"
	"github.com/MortalArena/Musketeers/pkg/policy"
	"github.com/MortalArena/Musketeers/pkg/sdk/interfaces"
	"github.com/MortalArena/Musketeers/pkg/vault/encryption"
)

type SecurityAdapter struct {
	node   *node.Node
	policy *policy.Engine
}

func NewSecurityAdapter(n *node.Node, pe *policy.Engine) *SecurityAdapter {
	return &SecurityAdapter{node: n, policy: pe}
}

func (a *SecurityAdapter) Encrypt(data []byte) ([]byte, error) {
	key := sha256.Sum256(a.node.KeyPair().Private)
	return encryption.Encrypt(data, key[:])
}

func (a *SecurityAdapter) Decrypt(data []byte) ([]byte, error) {
	key := sha256.Sum256(a.node.KeyPair().Private)
	return encryption.Decrypt(data, key[:])
}

func (a *SecurityAdapter) CheckAuth(principal, resource, action string) error {
	req := policy.Request{
		Principal: policy.Principal{DID: principal},
		Resource:  policy.Resource{Type: resource, Action: action},
	}
	result, err := a.policy.Evaluate(req)
	if err != nil {
		return err
	}
	if result.Effect != policy.EffectAllow {
		return fmt.Errorf("access denied: %s/%s for %s", resource, action, principal)
	}
	return nil
}

func (a *SecurityAdapter) Sign(data []byte) ([]byte, error) {
	return ed25519.Sign(a.node.KeyPair().Private, data), nil
}

func (a *SecurityAdapter) Verify(data, sig []byte) error {
	if !ed25519.Verify(a.node.KeyPair().Public, data, sig) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

var _ interfaces.SecurityInterface = (*SecurityAdapter)(nil)

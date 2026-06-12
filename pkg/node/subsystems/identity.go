package subsystems

import (
	"crypto/ed25519"

	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
	"github.com/MortalArena/Musketeers/pkg/identity"
)

type IdentitySubsystem struct {
	keyPair  *nrcrypto.KeyPair
	identity *identity.IdentityRecord
	keyCache map[string]ed25519.PublicKey
}

func NewIdentitySubsystem(keyPair *nrcrypto.KeyPair, identity *identity.IdentityRecord) *IdentitySubsystem {
	return &IdentitySubsystem{keyPair: keyPair, identity: identity, keyCache: make(map[string]ed25519.PublicKey)}
}

func (s *IdentitySubsystem) KeyPair() *nrcrypto.KeyPair             { return s.keyPair }
func (s *IdentitySubsystem) Identity() *identity.IdentityRecord     { return s.identity }
func (s *IdentitySubsystem) KeyCache() map[string]ed25519.PublicKey { return s.keyCache }

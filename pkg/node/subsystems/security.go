package subsystems

import (
	"crypto/ed25519"

	"github.com/MortalArena/Musketeers/pkg/identity"
	"github.com/MortalArena/Musketeers/pkg/search"
)

type SecuritySubsystem struct {
	nonceStore  any
	crl         *identity.CRLCache
	validators  any
	rateLimiter *search.TokenBucket
	founderPub  ed25519.PublicKey
}

func NewSecuritySubsystem(nonceStore any, crl *identity.CRLCache, validators any, rateLimiter *search.TokenBucket) *SecuritySubsystem {
	return &SecuritySubsystem{nonceStore: nonceStore, crl: crl, validators: validators, rateLimiter: rateLimiter}
}

func (s *SecuritySubsystem) NonceStore() any                     { return s.nonceStore }
func (s *SecuritySubsystem) CRL() *identity.CRLCache             { return s.crl }
func (s *SecuritySubsystem) Validators() any                     { return s.validators }
func (s *SecuritySubsystem) RateLimiter() *search.TokenBucket    { return s.rateLimiter }
func (s *SecuritySubsystem) FounderPublicKey() ed25519.PublicKey { return s.founderPub }

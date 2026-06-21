package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
)

// Domain separation tags — prevent signature reuse across different types
const (
	DomainIdentity      = "NR-IDENTITY-V1|"
	DomainRevocation    = "NR-REVOKE-V1|"
	DomainDelegation    = "NR-DELEGATION-V1|"
	DomainDomainFounder = "NR-DOMAIN-FOUNDER-V1|"
	DomainDomainOwner   = "NR-DOMAIN-OWNER-V1|"
	DomainChannelMsg    = "NR-CHANNEL-MSG-V1|"
	DomainSearch        = "NR-SEARCH-V1|"
	DomainDirectMsg     = "NR-DM-V1|"
	DomainChannelConfig = "NR-CHANNEL-CFG-V1|"
	DomainACP           = "NR-ACP-V1|"
)

var (
	ErrInvalidSignature = errors.New("invalid signature")
	ErrInvalidKey       = errors.New("invalid key")
)

// RandomNonce generates random 16-byte nonce
func RandomNonce() ([]byte, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	return nonce, nil
}

// SignPayload signs payload with domain separation tag
func SignPayload(priv ed25519.PrivateKey, domain, payload string) ([]byte, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return nil, ErrInvalidKey
	}
	msg := []byte(domain + payload)
	sig := ed25519.Sign(priv, msg)
	return sig, nil
}

// VerifyPayload verifies signature
func VerifyPayload(pub ed25519.PublicKey, domain, payload string, sig []byte) error {
	if len(pub) != ed25519.PublicKeySize {
		return ErrInvalidKey
	}
	msg := []byte(domain + payload)
	if !ed25519.Verify(pub, msg, sig) {
		return ErrInvalidSignature
	}
	return nil
}

// SignPayloadHex signs and returns signature as hex
func SignPayloadHex(priv ed25519.PrivateKey, domain, payload string) (string, error) {
	sig, err := SignPayload(priv, domain, payload)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(sig), nil
}

// VerifyPayloadHex verifies hex signature
func VerifyPayloadHex(pub ed25519.PublicKey, domain, payload, sigHex string) error {
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return fmt.Errorf("invalid signature format: %w", err)
	}
	return VerifyPayload(pub, domain, payload, sig)
}

// PubKeyFromHex converts public key from hex
func PubKeyFromHex(hexKey string) (ed25519.PublicKey, error) {
	raw, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, ErrInvalidKey
	}
	return ed25519.PublicKey(raw), nil
}

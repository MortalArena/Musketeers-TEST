package common

import "crypto/ed25519"

// KeyResolver resolves a public Ed25519 key from a DID.
type KeyResolver interface {
	ResolvePublicKey(did string) (ed25519.PublicKey, error)
}

// DIDProvider exposes a decentralized identifier.
type DIDProvider interface {
	DID() string
}

// Signer signs raw bytes.
type Signer interface {
	Sign(data []byte) ([]byte, error)
}

// Verifier verifies raw bytes against a signature.
type Verifier interface {
	Verify(data, signature []byte) error
}

// Encryptor encrypts plaintext bytes.
type Encryptor interface {
	Encrypt(plaintext []byte) ([]byte, error)
}

// Decryptor decrypts ciphertext bytes.
type Decryptor interface {
	Decrypt(ciphertext []byte) ([]byte, error)
}

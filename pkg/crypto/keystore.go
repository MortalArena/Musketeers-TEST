package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/scrypt"
)

const (
	keystoreVersion = 1
	scryptN         = 1 << 17 // 131072
	scryptR         = 8
	scryptP         = 1
	scryptKeyLen    = 32
	saltLen         = 32
)

// KeystoreFile تنسيق ملف المفتاح المشفّر
type KeystoreFile struct {
	Version    int    `json:"version"`
	DID        string `json:"did"`
	Salt       string `json:"salt"`       // hex
	Nonce      string `json:"nonce"`      // hex — AES-GCM nonce
	Ciphertext string `json:"ciphertext"` // hex — مفتاح خاص مشفّر
	Mnemonic   string `json:"mnemonic,omitempty"` // مشفّر داخل ciphertext إن وُجد
}

// keystorePlaintext البيانات قبل التشفير
type keystorePlaintext struct {
	PrivateKey string `json:"private_key"` // hex
	Mnemonic   string `json:"mnemonic,omitempty"`
}

// SaveKeystore يحفظ المفتاح الخاص مشفّراً على القرص
func SaveKeystore(path, passphrase string, kp *KeyPair, mnemonic string) error {
	plain := keystorePlaintext{
		PrivateKey: hex.EncodeToString(kp.Private),
		Mnemonic:   mnemonic,
	}
	plainJSON, err := json.Marshal(plain)
	if err != nil {
		return err
	}

	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return err
	}

	passphrase = NormalizePassphrase(passphrase)
	derived, err := scrypt.Key([]byte(passphrase), salt, scryptN, scryptR, scryptP, scryptKeyLen)
	if err != nil {
		return fmt.Errorf("فشل scrypt: %w", err)
	}

	block, err := aes.NewCipher(derived)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}

	ciphertext := gcm.Seal(nil, nonce, plainJSON, []byte("Musketeers-keystore-v1"))

	ks := KeystoreFile{
		Version:    keystoreVersion,
		DID:        kp.DID,
		Salt:       hex.EncodeToString(salt),
		Nonce:      hex.EncodeToString(nonce),
		Ciphertext: hex.EncodeToString(ciphertext),
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(ks, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// LoadKeystore يحمّل المفتاح الخاص من ملف مشفّر
func LoadKeystore(path, passphrase string) (*KeyPair, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("فشل قراءة keystore: %w", err)
	}

	var ks KeystoreFile
	if err := json.Unmarshal(data, &ks); err != nil {
		return nil, "", err
	}
	if ks.Version != keystoreVersion {
		return nil, "", fmt.Errorf("إصدار keystore غير مدعوم: %d", ks.Version)
	}

	salt, err := hex.DecodeString(ks.Salt)
	if err != nil {
		return nil, "", err
	}
	nonce, err := hex.DecodeString(ks.Nonce)
	if err != nil {
		return nil, "", err
	}
	ciphertext, err := hex.DecodeString(ks.Ciphertext)
	if err != nil {
		return nil, "", err
	}

	passphrase = NormalizePassphrase(passphrase)
	derived, err := scrypt.Key([]byte(passphrase), salt, scryptN, scryptR, scryptP, scryptKeyLen)
	if err != nil {
		return nil, "", err
	}

	block, err := aes.NewCipher(derived)
	if err != nil {
		return nil, "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, "", err
	}

	plainJSON, err := gcm.Open(nil, nonce, ciphertext, []byte("Musketeers-keystore-v1"))
	if err != nil {
		return nil, "", fmt.Errorf("عبارة مرور خاطئة أو ملف تالف")
	}

	var plain keystorePlaintext
	if err := json.Unmarshal(plainJSON, &plain); err != nil {
		return nil, "", err
	}

	privBytes, err := hex.DecodeString(plain.PrivateKey)
	if err != nil || len(privBytes) != ed25519.PrivateKeySize {
		return nil, "", fmt.Errorf("مفتاح خاص تالف في keystore")
	}

	kp := KeyPairFromPrivate(ed25519.PrivateKey(privBytes))
	if ks.DID != "" && kp.DID != ks.DID {
		return nil, "", fmt.Errorf("DID لا يطابق الملف")
	}
	return kp, plain.Mnemonic, nil
}

// KeystorePath يبني مسار keystore الافتراضي
func KeystorePath(dataDir string) string {
	return filepath.Join(dataDir, "identity.key")
}

// KeystoreExists يتحقق من وجود keystore
func KeystoreExists(dataDir string) bool {
	_, err := os.Stat(KeystorePath(dataDir))
	return err == nil
}

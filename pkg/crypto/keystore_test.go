package crypto

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestKeystoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "identity.key")

	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	mnemonic, _ := GenerateMnemonic()

	if err := SaveKeystore(path, "test-pass-phrase", kp, mnemonic); err != nil {
		t.Fatal(err)
	}

	loaded, loadedMnemonic, err := LoadKeystore(path, "test-pass-phrase")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DID != kp.DID {
		t.Fatal("DID mismatch")
	}
	if string(loaded.Private) != string(kp.Private) {
		t.Fatal("private key mismatch")
	}
	if loadedMnemonic != mnemonic {
		t.Fatal("mnemonic mismatch")
	}

	// عبارة مرور خاطئة
	if _, _, err := LoadKeystore(path, "wrong"); err == nil {
		t.Fatal("should fail with wrong passphrase")
	}

	// على Unix نتحقق من صلاحيات الملف
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("keystore permissions should be 0600, got %o", info.Mode().Perm())
		}
	}
}

// [SAFETY] Test argon2id key derivation (using public API)
func TestArgon2idKeyDerivation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "identity-argon2.key")

	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	mnemonic, _ := GenerateMnemonic()

	// Test encryption with argon2id (via SaveKeystore)
	if err := SaveKeystore(path, "test-passphrase", kp, mnemonic); err != nil {
		t.Fatalf("SaveKeystore failed: %v", err)
	}

	// Test decryption with correct passphrase (via LoadKeystore)
	loaded, loadedMnemonic, err := LoadKeystore(path, "test-passphrase")
	if err != nil {
		t.Fatalf("LoadKeystore failed: %v", err)
	}

	if loaded.DID != kp.DID {
		t.Fatal("DID mismatch after decryption")
	}
	if string(loaded.Private) != string(kp.Private) {
		t.Fatal("private key mismatch after decryption")
	}
	if loadedMnemonic != mnemonic {
		t.Fatal("mnemonic mismatch after decryption")
	}

	// Test decryption with wrong passphrase
	_, _, err = LoadKeystore(path, "wrong-passphrase")
	if err == nil {
		t.Fatal("should fail with wrong passphrase")
	}
}

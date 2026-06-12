package encryption

import "testing"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	plaintext := []byte("secret")
	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt returned error: %v", err)
	}
	if string(ciphertext) == string(plaintext) {
		t.Fatal("ciphertext equals plaintext")
	}
	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt returned error: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("unexpected plaintext: %s", decrypted)
	}
}

func TestDecryptRejectsWrongKey(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	wrong := []byte("abcdef0123456789abcdef0123456789")
	ciphertext, err := Encrypt([]byte("secret"), key)
	if err != nil {
		t.Fatalf("Encrypt returned error: %v", err)
	}
	if _, err := Decrypt(ciphertext, wrong); err == nil {
		t.Fatal("expected wrong-key error")
	}
}

func TestNormalizeKeyRejectsInvalidLength(t *testing.T) {
	if _, err := NormalizeKey([]byte("short")); err == nil {
		t.Fatal("expected invalid key length error")
	}
}

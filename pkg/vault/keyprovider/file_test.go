package keyprovider

import "testing"

func TestFileKeyProviderRoundTrip(t *testing.T) {
	provider := NewFileKeyProvider(t.TempDir())
	if err := provider.Store("master", []byte("0123456789abcdef0123456789abcdef")); err != nil {
		t.Fatalf("Store returned error: %v", err)
	}
	key, err := provider.Load("master")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if string(key) != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("unexpected key: %s", key)
	}
	if err := provider.Delete("master"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if _, err := provider.Load("master"); err == nil {
		t.Fatal("expected missing key error")
	}
}

func TestFileKeyProviderRejectsInvalidKey(t *testing.T) {
	provider := NewFileKeyProvider(t.TempDir())
	if err := provider.StoreKey("master", []byte("short")); err == nil {
		t.Fatal("expected invalid key error")
	}
}

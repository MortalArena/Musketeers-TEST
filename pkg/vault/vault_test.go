package vault

import (
	"testing"

	"github.com/MortalArena/Musketeers/pkg/vault/keyprovider"
)

func TestVaultStoreRetrieveDeleteList(t *testing.T) {
	provider := keyprovider.NewFileKeyProvider(t.TempDir())
	v := New(provider)
	if err := v.Store("github-token", []byte("token-value"), map[string]string{"service": "github"}); err != nil {
		t.Fatalf("Store returned error: %v", err)
	}
	value, err := v.Retrieve("github-token")
	if err != nil {
		t.Fatalf("Retrieve returned error: %v", err)
	}
	if string(value) != "token-value" {
		t.Fatalf("unexpected value: %s", value)
	}
	names, err := v.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(names) != 1 || names[0] != "github-token" {
		t.Fatalf("unexpected names: %v", names)
	}
	if err := v.Delete("github-token"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if _, err := v.Retrieve("github-token"); err == nil {
		t.Fatal("expected missing secret error")
	}
}

func TestVaultRejectsEmptyName(t *testing.T) {
	v := New(keyprovider.NewFileKeyProvider(t.TempDir()))
	if err := v.Store("", []byte("secret"), nil); err == nil {
		t.Fatal("expected empty name error")
	}
}

package keyprovider

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type FileKeyProvider struct {
	dir string
}

func NewFileKeyProvider(dir string) *FileKeyProvider {
	return &FileKeyProvider{dir: dir}
}

func (p *FileKeyProvider) Store(name string, key []byte) error {
	if err := validateName(name); err != nil {
		return err
	}
	if err := os.MkdirAll(p.dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(p.path(name), []byte(encodeKey(key)), 0600)
}

func (p *FileKeyProvider) StoreKey(name string, key []byte) error {
	if _, err := NormalizeKey(key); err != nil {
		return err
	}
	return p.Store(name, key)
}

func (p *FileKeyProvider) Load(name string) ([]byte, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p.path(name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("key not found: %s", name)
		}
		return nil, err
	}
	return decodeKey(strings.TrimSpace(string(data)))
}

func (p *FileKeyProvider) Delete(name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	if err := os.Remove(p.path(name)); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("key not found: %s", name)
		}
		return err
	}
	return nil
}

func (p *FileKeyProvider) List() ([]string, error) {
	entries, err := os.ReadDir(p.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".key") {
			continue
		}
		names = append(names, strings.TrimSuffix(entry.Name(), ".key"))
	}
	sort.Strings(names)
	return names, nil
}

func (p *FileKeyProvider) path(name string) string {
	return filepath.Join(p.dir, name+".key")
}

func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("key name is required")
	}
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return fmt.Errorf("invalid key name: %s", name)
	}
	return nil
}

func NormalizeKey(key []byte) ([]byte, error) {
	switch len(key) {
	case 16, 24, 32:
		return append([]byte(nil), key...), nil
	default:
		return nil, fmt.Errorf("key length must be 16, 24, or 32 bytes")
	}
}

type OSKeychainProvider struct {
	fallback KeyProvider
}

func NewOSKeychainProvider(fallback KeyProvider) *OSKeychainProvider {
	return &OSKeychainProvider{fallback: fallback}
}

func (p *OSKeychainProvider) Store(name string, key []byte) error {
	if p.fallback != nil {
		if provider, ok := p.fallback.(*FileKeyProvider); ok {
			return provider.StoreKey(name, key)
		}
		return p.fallback.Store(name, key)
	}
	return fmt.Errorf("os keychain provider is not implemented on this platform")
}

func (p *OSKeychainProvider) Load(name string) ([]byte, error) {
	if p.fallback != nil {
		return p.fallback.Load(name)
	}
	return nil, fmt.Errorf("os keychain provider is not implemented on this platform")
}

func (p *OSKeychainProvider) Delete(name string) error {
	if p.fallback != nil {
		return p.fallback.Delete(name)
	}
	return fmt.Errorf("os keychain provider is not implemented on this platform")
}

func (p *OSKeychainProvider) List() ([]string, error) {
	if p.fallback != nil {
		return p.fallback.List()
	}
	return []string{}, nil
}

package keyprovider

import "encoding/hex"

type KeyProvider interface {
	Store(name string, key []byte) error
	Load(name string) ([]byte, error)
	Delete(name string) error
	List() ([]string, error)
}

func encodeKey(key []byte) string {
	return hex.EncodeToString(key)
}

func decodeKey(value string) ([]byte, error) {
	return hex.DecodeString(value)
}

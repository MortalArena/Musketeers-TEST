package crypto

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/mr-tron/base58"
	"golang.org/x/crypto/scrypt"
)

const (
	DefaultPowDifficulty = 18              // N = 1<<18
	DefaultIdentityTTL   = 365 * 24 * 3600 // سنة واحدة بالثواني
)

// KeyPair زوج مفاتيح Ed25519 مع DID
type KeyPair struct {
	Private ed25519.PrivateKey
	Public  ed25519.PublicKey
	DID     string
}

// GenerateKeyPair يولّد زوج مفاتيح جديد مع DID
func GenerateKeyPair() (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("فشل توليد المفاتيح: %w", err)
	}
	return &KeyPair{
		Private: priv,
		Public:  pub,
		DID:     DIDFromPublicKey(pub),
	}, nil
}

// KeyPairFromPrivate ينشئ KeyPair من مفتاح خاص
func KeyPairFromPrivate(priv ed25519.PrivateKey) *KeyPair {
	pub := priv.Public().(ed25519.PublicKey)
	return &KeyPair{
		Private: priv,
		Public:  pub,
		DID:     DIDFromPublicKey(pub),
	}
}

// DIDFromPublicKey يحسب DID من المفتاح العام: did:mskt:<base58(sha256(pub)[:16])>
func DIDFromPublicKey(pub ed25519.PublicKey) string {
	h := sha256.Sum256(pub)
	return "did:mskt:" + base58.Encode(h[:16])
}

// PowDifficulty يقرأ صعوبة PoW من البيئة
func PowDifficulty() int {
	if v := os.Getenv("NR_POW_DIFFICULTY"); v != "" {
		if d, err := strconv.Atoi(v); err == nil && d >= 1 && d <= 20 {
			return d
		}
	}
	return DefaultPowDifficulty
}

// MinePow يعدّن PoW باستخدام scrypt — أول بايت من الناتج == 0x00
func MinePow(ctx context.Context, did string, difficulty int) ([]byte, error) {
	N := uint64(1) << uint(difficulty)
	salt := []byte(did)
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}

	type result struct {
		nonce []byte
	}
	resultCh := make(chan result, 1)
	var found atomic.Bool
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			localNonce := make([]byte, 16)
			copy(localNonce, nonce)
			// كل worker يبدأ من offset مختلف
			localNonce[0] = byte((int(localNonce[0]) + workerID*17) % 256)

			counter := uint64(0)
			for {
				if found.Load() {
					return
				}
				select {
				case <-ctx.Done():
					return
				default:
				}

				input := append([]byte(did), localNonce...)
				out, err := scrypt.Key(input, salt, int(N), 8, 1, 32)
				if err != nil {
					return
				}
				if out[0] == 0x00 {
					if found.CompareAndSwap(false, true) {
						resultCh <- result{nonce: append([]byte(nil), localNonce...)}
					}
					return
				}

				counter++
				// زيادة العداد في آخر بايت
				for i := len(localNonce) - 1; i >= 0; i-- {
					localNonce[i]++
					if localNonce[i] != 0 {
						break
					}
				}
			}
		}(w)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res, ok := <-resultCh:
		if !ok {
			return nil, fmt.Errorf("فشل تعدين PoW")
		}
		return res.nonce, nil
	}
}

// VerifyPow يتحقق من صحة PoW
func VerifyPow(did string, nonce []byte, difficulty int) bool {
	N := uint64(1) << uint(difficulty)
	salt := []byte(did)
	input := append([]byte(did), nonce...)
	out, err := scrypt.Key(input, salt, int(N), 8, 1, 32)
	if err != nil {
		return false
	}
	return out[0] == 0x00
}

// PublicKeyHex يرجع المفتاح العام كـ hex
func PublicKeyHex(pub ed25519.PublicKey) string {
	return fmt.Sprintf("%x", pub)
}

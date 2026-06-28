package crypto

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/scrypt"
)

const (
	// [SAFETY] Low PoW difficulty for fast identity verification
	// [WHY] We're proving identity, not mining crypto - needs to be fast (< 1 second)
	// [HOW] Difficulty 1 provides basic proof while remaining fast
	DefaultPowDifficulty = 1 // Fast mining for identity verification
	MinPowDifficulty     = 1 // Minimum difficulty
	MaxPowDifficulty     = 4 // Maximum for special cases

	// [SAFETY] Balanced scrypt parameters for speed and security
	ScryptN = 1 << 15 // 32,768 - balanced for speed
	ScryptR = 8
	ScryptP = 1
	KeyLen  = 32

	// Fixed salt for the network
	PowSalt = "musketeers-pow-v1"
)

// DynamicDifficultyAdjuster dynamically adjusts difficulty
type DynamicDifficultyAdjuster struct {
	mu                sync.RWMutex
	currentDifficulty int32
	targetBlockTime   time.Duration
	history           []BlockInfo
	maxHistory        int
	lastAdjustment    time.Time
}

type BlockInfo struct {
	Timestamp time.Time
	Duration  time.Duration
}

func NewDynamicDifficultyAdjuster() *DynamicDifficultyAdjuster {
	return &DynamicDifficultyAdjuster{
		currentDifficulty: DefaultPowDifficulty,
		targetBlockTime:   10 * time.Minute,
		history:           make([]BlockInfo, 0, 1000),
		maxHistory:        1000,
		lastAdjustment:    time.Now(),
	}
}

// RecordBlock records a new block
func (dda *DynamicDifficultyAdjuster) RecordBlock(duration time.Duration) {
	dda.mu.Lock()
	defer dda.mu.Unlock()

	dda.history = append(dda.history, BlockInfo{
		Timestamp: time.Now(),
		Duration:  duration,
	})

	// Maintain size
	if len(dda.history) > dda.maxHistory {
		dda.history = dda.history[1:]
	}

	// Adjust every 100 blocks or 10 minutes
	if len(dda.history) >= 100 || time.Since(dda.lastAdjustment) >= 10*time.Minute {
		dda.adjust()
	}
}

// adjust adjusts difficulty - disabled to ensure constant difficulty
func (dda *DynamicDifficultyAdjuster) adjust() {
	// Disable dynamic adjustment to ensure constant difficulty of 1
	// Prevent weak devices from stopping
}

// GetDifficulty returns the current difficulty
func (dda *DynamicDifficultyAdjuster) GetDifficulty() int {
	return int(atomic.LoadInt32(&dda.currentDifficulty))
}

// PoWResult mining result
type PoWResult struct {
	Nonce      string
	Hash       string
	Difficulty int
	Duration   time.Duration
	Attempts   int64
}

// MineIdentity mines a new identity
func MineIdentity(ctx context.Context, did string, difficulty int) (*PoWResult, error) {
	startTime := time.Now()
	workers := runtime.NumCPU()
	if workers > 16 {
		workers = 16 // Maximum limit
	}

	resultChan := make(chan *PoWResult, 1)
	errorChan := make(chan error, workers)
	var attempts int64

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer func() {
				if r := recover(); r != nil {
					_ = r
				}
			}()
			defer wg.Done()

			nonce := make([]byte, 32)
			if _, err := rand.Read(nonce); err != nil {
				errorChan <- err
				return
			}

			// Each worker starts from a different point
			start := int64(workerID) * 10000000

			for i := start; i < start+10000000; i++ {
				select {
				case <-ctx.Done():
					return
				default:
				}

				atomic.AddInt64(&attempts, 1)

				// Update nonce
				nonce[0] = byte(i & 0xff)
				nonce[1] = byte((i >> 8) & 0xff)
				nonce[2] = byte((i >> 16) & 0xff)
				nonce[3] = byte((i >> 24) & 0xff)
				nonce[4] = byte(workerID)

				// Calculate hash
				hash, err := scrypt.Key(
					[]byte(did+hex.EncodeToString(nonce)),
					[]byte(PowSalt),
					ScryptN, ScryptR, ScryptP, KeyLen,
				)
				if err != nil {
					errorChan <- err
					return
				}

				// Check difficulty
				if checkDifficulty(hash, difficulty) {
					result := &PoWResult{
						Nonce:      hex.EncodeToString(nonce),
						Hash:       hex.EncodeToString(hash),
						Difficulty: difficulty,
						Duration:   time.Since(startTime),
						Attempts:   atomic.LoadInt64(&attempts),
					}

					select {
					case resultChan <- result:
						cancel() // Stop other workers
					default:
					}
					return
				}
			}
		}(w)
	}

	// Wait for result
	go func() {
		defer func() {
			if r := recover(); r != nil {
				_ = r
			}
		}()
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	select {
	case result := <-resultChan:
		if result == nil {
			return nil, fmt.Errorf("mining failed")
		}
		return result, nil

	case err := <-errorChan:
		return nil, err

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// checkDifficulty checks the difficulty
func checkDifficulty(hash []byte, difficulty int) bool {
	// [SAFETY] Validate hash length
	if len(hash) < KeyLen {
		return false
	}

	requiredZeros := difficulty / 8
	requiredBits := difficulty % 8

	// Check full zeros
	for i := 0; i < requiredZeros; i++ {
		if i >= len(hash) || hash[i] != 0 {
			return false
		}
	}

	// Check additional bits
	if requiredBits > 0 && requiredZeros < len(hash) {
		mask := byte(0xff << (8 - requiredBits))
		if hash[requiredZeros]&mask != 0 {
			return false
		}
	}

	return true
}

// VerifyPoW verifies PoW validity
func VerifyPoW(did, nonce string, difficulty int) (bool, error) {
	hash, err := scrypt.Key(
		[]byte(did+nonce),
		[]byte(PowSalt),
		ScryptN, ScryptR, ScryptP, KeyLen,
	)
	if err != nil {
		return false, err
	}

	return checkDifficulty(hash, difficulty), nil
}

// EstimateMiningTime calculates expected mining time
func EstimateMiningTime(difficulty int, hashRate float64) time.Duration {
	// Expected attempts = 2^difficulty
	expectedAttempts := float64(uint64(1) << uint(difficulty))
	seconds := expectedAttempts / hashRate
	return time.Duration(seconds) * time.Second
}

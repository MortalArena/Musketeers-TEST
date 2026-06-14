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
	// ✅ صعوبة منخفضة جداً للأجهزة الضعيفة جداً
	DefaultPowDifficulty = 3 // N = 1<<3 (سريع جداً للأجهزة الضعيفة جداً)
	MinPowDifficulty     = 2 // الحد الأدنى للأجهزة الضعيفة جداً
	MaxPowDifficulty     = 8 // الحد الأقصى للأجهزة المتوسطة

	// معاملات scrypt المحسّنة
	ScryptN = 1 << 22 // 4,194,304
	ScryptR = 8
	ScryptP = 1
	KeyLen  = 32

	// Salt ثابت للشبكة
	PowSalt = "musketeers-pow-v1"
)

// DynamicDifficultyAdjuster يضبط الصعوبة ديناميكياً
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

// RecordBlock يسجل كتلة جديدة
func (dda *DynamicDifficultyAdjuster) RecordBlock(duration time.Duration) {
	dda.mu.Lock()
	defer dda.mu.Unlock()

	dda.history = append(dda.history, BlockInfo{
		Timestamp: time.Now(),
		Duration:  duration,
	})

	// الحفاظ على الحجم
	if len(dda.history) > dda.maxHistory {
		dda.history = dda.history[1:]
	}

	// الضبط كل 100 كتلة أو 10 دقائق
	if len(dda.history) >= 100 || time.Since(dda.lastAdjustment) >= 10*time.Minute {
		dda.adjust()
	}
}

// adjust يضبط الصعوبة
func (dda *DynamicDifficultyAdjuster) adjust() {
	if len(dda.history) < 10 {
		return
	}

	// حساب المتوسط
	var total time.Duration
	for _, block := range dda.history {
		total += block.Duration
	}
	avg := total / time.Duration(len(dda.history))

	ratio := float64(avg) / float64(dda.targetBlockTime)

	current := atomic.LoadInt32(&dda.currentDifficulty)
	newDifficulty := current

	// خوارزمية الضبط
	if ratio > 2.0 {
		newDifficulty = current - 2 // بطيء جداً
	} else if ratio > 1.5 {
		newDifficulty = current - 1 // بطيء
	} else if ratio < 0.5 {
		newDifficulty = current + 2 // سريع جداً
	} else if ratio < 0.75 {
		newDifficulty = current + 1 // سريع
	}

	// تطبيق الحدود
	if newDifficulty < MinPowDifficulty {
		newDifficulty = MinPowDifficulty
	}
	if newDifficulty > MaxPowDifficulty {
		newDifficulty = MaxPowDifficulty
	}

	atomic.StoreInt32(&dda.currentDifficulty, newDifficulty)
	dda.lastAdjustment = time.Now()
}

// GetDifficulty يعيد الصعوبة الحالية
func (dda *DynamicDifficultyAdjuster) GetDifficulty() int {
	return int(atomic.LoadInt32(&dda.currentDifficulty))
}

// PoWResult نتيجة التعدين
type PoWResult struct {
	Nonce      string
	Hash       string
	Difficulty int
	Duration   time.Duration
	Attempts   int64
}

// MineIdentity يقوم بتعدين هوية جديدة
func MineIdentity(ctx context.Context, did string, difficulty int) (*PoWResult, error) {
	startTime := time.Now()
	workers := runtime.NumCPU()
	if workers > 16 {
		workers = 16 // حد أقصى
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
			defer wg.Done()

			nonce := make([]byte, 32)
			if _, err := rand.Read(nonce); err != nil {
				errorChan <- err
				return
			}

			// كل worker يبدأ من نقطة مختلفة
			start := int64(workerID) * 10000000

			for i := start; i < start+10000000; i++ {
				select {
				case <-ctx.Done():
					return
				default:
				}

				atomic.AddInt64(&attempts, 1)

				// تحديث nonce
				nonce[0] = byte(i & 0xff)
				nonce[1] = byte((i >> 8) & 0xff)
				nonce[2] = byte((i >> 16) & 0xff)
				nonce[3] = byte((i >> 24) & 0xff)
				nonce[4] = byte(workerID)

				// حساب hash
				hash, err := scrypt.Key(
					[]byte(did+hex.EncodeToString(nonce)),
					[]byte(PowSalt),
					ScryptN, ScryptR, ScryptP, KeyLen,
				)
				if err != nil {
					errorChan <- err
					return
				}

				// التحقق من الصعوبة
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
						cancel() // إيقاف العمال الآخرين
					default:
					}
					return
				}
			}
		}(w)
	}

	// انتظار النتيجة
	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	select {
	case result := <-resultChan:
		if result == nil {
			return nil, fmt.Errorf("فشل التعدين")
		}
		return result, nil

	case err := <-errorChan:
		return nil, err

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// checkDifficulty يتحقق من الصعوبة
func checkDifficulty(hash []byte, difficulty int) bool {
	requiredZeros := difficulty / 8
	requiredBits := difficulty % 8

	// التحقق من الأصفار الكاملة
	for i := 0; i < requiredZeros; i++ {
		if i >= len(hash) || hash[i] != 0 {
			return false
		}
	}

	// التحقق من البتات الإضافية
	if requiredBits > 0 && requiredZeros < len(hash) {
		mask := byte(0xff << (8 - requiredBits))
		if hash[requiredZeros]&mask != 0 {
			return false
		}
	}

	return true
}

// VerifyPoW يتحقق من صحة PoW
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

// EstimateMiningTime يحسب الوقت المتوقع للتعدين
func EstimateMiningTime(difficulty int, hashRate float64) time.Duration {
	// عدد المحاولات المتوقعة = 2^difficulty
	expectedAttempts := float64(uint64(1) << uint(difficulty))
	seconds := expectedAttempts / hashRate
	return time.Duration(seconds) * time.Second
}

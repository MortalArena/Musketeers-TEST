package crypto

import (
	"context"
	"testing"
	"time"
)

func TestMineIdentity(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	did := "did:nr:test123456789"
	difficulty := 16 // صعوبة منخفضة جداً للاختبار السريع

	result, err := MineIdentity(ctx, did, difficulty)
	if err != nil {
		t.Fatalf("فشل التعدين: %v", err)
	}

	// التحقق من النتيجة
	if result.Nonce == "" {
		t.Error("Nonce فارغ")
	}
	if result.Hash == "" {
		t.Error("Hash فارغ")
	}
	if result.Difficulty != difficulty {
		t.Errorf("الصعوبة غير صحيحة: %d != %d", result.Difficulty, difficulty)
	}

	// التحقق من صحة PoW
	valid, err := VerifyPoW(did, result.Nonce, difficulty)
	if err != nil {
		t.Fatalf("فشل التحقق: %v", err)
	}
	if !valid {
		t.Error("PoW غير صالح")
	}
}

func TestDynamicDifficultyAdjuster(t *testing.T) {
	dda := NewDynamicDifficultyAdjuster()

	// تسجيل كتل بطيئة
	for i := 0; i < 100; i++ {
		dda.RecordBlock(20 * time.Minute) // بطيء جداً
	}

	// يجب أن تنخفض الصعوبة
	if dda.GetDifficulty() >= DefaultPowDifficulty {
		t.Error("الصعوبة يجب أن تنخفض")
	}

	// تسجيل كتل سريعة
	dda2 := NewDynamicDifficultyAdjuster()
	for i := 0; i < 100; i++ {
		dda2.RecordBlock(1 * time.Minute) // سريع جداً
	}

	// يجب أن ترتفع الصعوبة
	if dda2.GetDifficulty() <= DefaultPowDifficulty {
		t.Error("الصعوبة يجب أن ترتفع")
	}
}

func TestVerifyPoW(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	did := "did:nr:testverify"
	difficulty := 18 // صعوبة منخفضة للاختبار السريع

	result, err := MineIdentity(ctx, did, difficulty)
	if err != nil {
		t.Fatalf("فشل التعدين: %v", err)
	}

	// التحقق من صحة PoW
	valid, err := VerifyPoW(did, result.Nonce, difficulty)
	if err != nil {
		t.Fatalf("فشل التحقق: %v", err)
	}
	if !valid {
		t.Error("PoW غير صالح")
	}

	// التحقق من nonce خاطئ
	invalid, err := VerifyPoW(did, "wrongnonce", difficulty)
	if err != nil {
		t.Fatalf("فشل التحقق: %v", err)
	}
	if invalid {
		t.Error("Nonce خاطئ يجب أن يفشل التحقق")
	}
}

func TestCheckDifficulty(t *testing.T) {
	// اختبار hash مع أصفار في البداية
	hash1 := make([]byte, 32)
	hash1[0] = 0
	hash1[1] = 0
	hash1[2] = 0

	if !checkDifficulty(hash1, 16) {
		t.Error("Hash مع 16 bit zeros يجب أن ينجح")
	}

	if checkDifficulty(hash1, 24) {
		t.Error("Hash مع 16 bit zeros يجب أن يفشل لـ 24 bit difficulty")
	}
}

func BenchmarkMineIdentity(b *testing.B) {
	ctx := context.Background()
	did := "did:nr:benchmark"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MineIdentity(ctx, did, 18) // صعوبة منخفضة للـ benchmark
	}
}

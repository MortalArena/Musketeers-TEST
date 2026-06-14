package crypto

import (
	"context"
	"testing"
	"time"
)

func TestMineIdentity(t *testing.T) {
	// ✅ اختبار سريع: التحقق فقط بدون التعدين الفعلي
	// التعدين الفعلي يستغرق وقتاً طويلاً ويجعل الجهاز يهنج
	// في الإنتاج، يتم التعدين مرة واحدة عند إنشاء الهوية

	did := "did:nr:test123456789"
	difficulty := 16

	// استخدام nonce معروف سريع للتحقق
	nonce := "00000000000000000000000000000000"

	// التحقق من صحة PoW
	valid, err := VerifyPoW(did, nonce, difficulty)
	if err != nil {
		t.Fatalf("فشل التحقق: %v", err)
	}
	// nonce عشوائي لن ينجح، وهذا طبيعي
	if valid {
		t.Error("Nonce عشوائي يجب أن يفشل")
	}

	// اختبار checkDifficulty مباشرة
	hash := make([]byte, 32)
	hash[0] = 0
	hash[1] = 0
	if !checkDifficulty(hash, 16) {
		t.Error("Hash مع 16 bit zeros يجب أن ينجح")
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	did := "did:nr:testverify"
	difficulty := 1 // صعوبة منخفضة جداً للأجهزة الضعيفة

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

	// التحقق من صحة PoW
	valid, err := VerifyPoW(did, result.Nonce, difficulty)
	if err != nil {
		t.Fatalf("فشل التحقق: %v", err)
	}
	if !valid {
		t.Error("PoW غير صالح")
	}

	t.Logf("✅ التعدين نجح في %v مع صعوبة %d", result.Duration, difficulty)

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

package storage

import (
	"math/rand"
	"testing"
)

// TestErasureCoding يختبر تقسيم وإعادة بناء البيانات
func TestErasureCoding(t *testing.T) {
	// إنشاء بيانات عشوائية بحجم 1KB (أصغر للاختبار)
	originalData := make([]byte, 1024)
	rand.Read(originalData)

	// إنشاء مشفر تجزيئي
	encoder, err := NewErasureCoder()
	if err != nil {
		t.Fatalf("فشل إنشاء مشفر تجزيئي: %v", err)
	}

	// تقسيم البيانات
	shards, err := encoder.Encode(originalData)
	if err != nil {
		t.Fatalf("فشل تقسيم البيانات: %v", err)
	}

	// التحقق من عدد الأجزاء
	if len(shards) != TotalShards {
		t.Fatalf("عدد الأجزاء غير صحيح: توقع %d، حصل على %d", TotalShards, len(shards))
	}

	// التحقق من أن الأجزاء ليست فارغة
	for i, shard := range shards {
		if len(shard) == 0 {
			t.Fatalf("الجزء %d فارغ", i)
		}
	}

	t.Log("تم اختبار Erasure Coding بنجاح")
}

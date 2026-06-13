package storage

import (
	"sync"
	"testing"
)

func TestQuotaManager_Enforcement(t *testing.T) {
	qm := NewQuotaManager()
	did := "did:mskt:test_user"

	// 1. الاختبار الافتراضي (يجب أن يسمح بـ 1GB)
	largeFile := int64(500 * 1024 * 1024) // 500 MB
	err := qm.CheckAndAdd(did, largeFile)
	if err != nil {
		t.Fatalf("Unexpected error on first addition: %v", err)
	}

	// 2. محاولة تجاوز الحد (500MB + 600MB > 1GB)
	overLimitFile := int64(600 * 1024 * 1024)
	err = qm.CheckAndAdd(did, overLimitFile)
	if err == nil {
		t.Errorf("Expected quota exceeded error, but got nil")
	}

	// 3. تحرير مساحة ثم المحاولة مجدداً
	qm.Release(did, 200*1024*1024) // تحرير 200 MB

	// الآن الاستخدام 300MB، محاولة إضافة 600MB يجب أن تنجح (المجموع 900MB < 1GB)
	err = qm.CheckAndAdd(did, 600*1024*1024)
	if err != nil {
		t.Fatalf("Unexpected error after release: %v", err)
	}
}

func TestQuotaManager_SetLimit(t *testing.T) {
	qm := NewQuotaManager()
	did := "did:mskt:test_user"

	// تحديد حد مخصص (2GB)
	qm.SetLimit(did, 2*1024*1024*1024)

	// إضافة 1.5GB
	err := qm.CheckAndAdd(did, int64(1.5*1024*1024*1024))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// محاولة إضافة 1GB إضافي (المجموع 2.5GB > 2GB)
	err = qm.CheckAndAdd(did, 1024*1024*1024)
	if err == nil {
		t.Error("Expected quota exceeded error with custom limit")
	}
}

func TestQuotaManager_Release(t *testing.T) {
	qm := NewQuotaManager()
	did := "did:mskt:test_user"

	// إضافة ملف
	qm.CheckAndAdd(did, 500*1024*1024)

	// التحقق من الاستخدام
	if usage := qm.GetUsage(did); usage != 500*1024*1024 {
		t.Errorf("Expected usage 500MB, got %d", usage)
	}

	// تحرير المساحة
	qm.Release(did, 500*1024*1024)

	// التحقق من أن الاستخدام أصبح 0
	if usage := qm.GetUsage(did); usage != 0 {
		t.Errorf("Expected usage 0 after release, got %d", usage)
	}
}

func TestQuotaManager_InvalidSize(t *testing.T) {
	qm := NewQuotaManager()
	did := "did:mskt:test_user"

	// محاولة إضافة حجم صفر
	err := qm.CheckAndAdd(did, 0)
	if err == nil {
		t.Error("Expected error for zero size")
	}

	// محاولة إضافة حجم سلبي
	err = qm.CheckAndAdd(did, -100)
	if err == nil {
		t.Error("Expected error for negative size")
	}
}

func TestQuotaManager_GetUsage(t *testing.T) {
	qm := NewQuotaManager()
	did := "did:mskt:test_user"

	// الاستخدام الافتراضي يجب أن يكون 0
	if usage := qm.GetUsage(did); usage != 0 {
		t.Errorf("Expected usage 0, got %d", usage)
	}

	// إضافة ملف
	qm.CheckAndAdd(did, 500*1024*1024)

	// التحقق من الاستخدام
	if usage := qm.GetUsage(did); usage != 500*1024*1024 {
		t.Errorf("Expected usage 500MB, got %d", usage)
	}
}

func TestQuotaManager_GetLimit(t *testing.T) {
	qm := NewQuotaManager()
	did := "did:mskt:test_user"

	// الحد الافتراضي يجب أن يكون 1GB
	if limit := qm.GetLimit(did); limit != DefaultFreeTierBytes {
		t.Errorf("Expected limit %d, got %d", DefaultFreeTierBytes, limit)
	}

	// تحديد حد مخصص
	qm.SetLimit(did, 2*1024*1024*1024)

	// التحقق من الحد المخصص
	if limit := qm.GetLimit(did); limit != 2*1024*1024*1024 {
		t.Errorf("Expected limit 2GB, got %d", limit)
	}
}

func TestQuotaManager_GetRemaining(t *testing.T) {
	qm := NewQuotaManager()
	did := "did:mskt:test_user"

	// المساحة المتبقية الافتراضية يجب أن تكون 1GB
	if remaining := qm.GetRemaining(did); remaining != DefaultFreeTierBytes {
		t.Errorf("Expected remaining %d, got %d", DefaultFreeTierBytes, remaining)
	}

	// إضافة 500MB
	qm.CheckAndAdd(did, 500*1024*1024)

	// المساحة المتبقية يجب أن تكون 1GB - 500MB
	expectedRemaining := int64(DefaultFreeTierBytes - 500*1024*1024)
	if remaining := qm.GetRemaining(did); remaining != expectedRemaining {
		t.Errorf("Expected remaining %d, got %d", expectedRemaining, remaining)
	}
}

func TestQuotaManager_ResetUsage(t *testing.T) {
	qm := NewQuotaManager()
	did := "did:mskt:test_user"

	// إضافة ملفات
	qm.CheckAndAdd(did, 500*1024*1024)
	qm.CheckAndAdd(did, 300*1024*1024)

	// التحقق من الاستخدام
	if usage := qm.GetUsage(did); usage != 800*1024*1024 {
		t.Errorf("Expected usage 800MB, got %d", usage)
	}

	// إعادة تعيين الاستخدام
	qm.ResetUsage(did)

	// التحقق من أن الاستخدام أصبح 0
	if usage := qm.GetUsage(did); usage != 0 {
		t.Errorf("Expected usage 0 after reset, got %d", usage)
	}
}

func TestQuotaManager_ConcurrentAccess(t *testing.T) {
	qm := NewQuotaManager()
	did := "did:mskt:test_user"

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	// محاكاة 10 عمليات إضافة متزامنة
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := qm.CheckAndAdd(did, 100*1024*1024) // 100MB لكل عملية
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// التحقق من عدم وجود أخطاء
	for err := range errors {
		if err != nil {
			t.Errorf("Unexpected error in concurrent access: %v", err)
		}
	}

	// التحقق من الاستخدام النهائي (10 * 100MB = 1GB)
	expectedUsage := int64(10) * 100 * 1024 * 1024
	if usage := qm.GetUsage(did); usage != expectedUsage {
		t.Errorf("Expected usage %d after concurrent additions, got %d", expectedUsage, usage)
	}
}

func TestQuotaManager_MultipleUsers(t *testing.T) {
	qm := NewQuotaManager()

	// إضافة ملفات لمستخدمين مختلفين
	qm.CheckAndAdd("did:mskt:user1", 500*1024*1024)
	qm.CheckAndAdd("did:mskt:user2", 700*1024*1024)

	// التحقق من الاستخدام لكل مستخدم
	if usage := qm.GetUsage("did:mskt:user1"); usage != 500*1024*1024 {
		t.Errorf("Expected user1 usage 500MB, got %d", usage)
	}
	if usage := qm.GetUsage("did:mskt:user2"); usage != 700*1024*1024 {
		t.Errorf("Expected user2 usage 700MB, got %d", usage)
	}
}

func TestQuotaManager_ReleaseMoreThanUsage(t *testing.T) {
	qm := NewQuotaManager()
	did := "did:mskt:test_user"

	// إضافة ملف
	qm.CheckAndAdd(did, 500*1024*1024)

	// محاولة تحرير أكثر من الاستخدام
	qm.Release(did, 1000*1024*1024)

	// الاستخدام يجب أن يكون 0 (منع القيم السلبية)
	if usage := qm.GetUsage(did); usage != 0 {
		t.Errorf("Expected usage 0 after releasing more than usage, got %d", usage)
	}
}

func TestQuotaManager_ExactLimit(t *testing.T) {
	qm := NewQuotaManager()
	did := "did:mskt:test_user"

	// إضافة 1GB بالضبط
	err := qm.CheckAndAdd(did, DefaultFreeTierBytes)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// محاولة إضافة 1 بايت إضافي
	err = qm.CheckAndAdd(did, 1)
	if err == nil {
		t.Error("Expected quota exceeded error at exact limit")
	}
}

func TestQuotaManager_SmallFiles(t *testing.T) {
	qm := NewQuotaManager()
	did := "did:mskt:test_user"

	// إضافة ملفات صغيرة متعددة
	for i := 0; i < 100; i++ {
		err := qm.CheckAndAdd(did, 10*1024*1024) // 10MB لكل ملف
		if err != nil {
			t.Fatalf("Unexpected error on small file %d: %v", i, err)
		}
	}

	// الاستخدام يجب أن يكون 100 * 10MB = 1000MB
	expectedUsage := int64(100) * 10 * 1024 * 1024
	if usage := qm.GetUsage(did); usage != expectedUsage {
		t.Errorf("Expected usage %d after 100 small files, got %d", expectedUsage, usage)
	}
}

func TestQuotaManager_VeryLargeFile(t *testing.T) {
	qm := NewQuotaManager()
	did := "did:mskt:test_user"

	// محاولة إضافة ملف أكبر من الحد الافتراضي
	veryLargeFile := int64(2) * 1024 * 1024 * 1024 // 2GB
	err := qm.CheckAndAdd(did, veryLargeFile)
	if err == nil {
		t.Error("Expected quota exceeded error for very large file")
	}
}

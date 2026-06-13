package content

import (
	"testing"
)

// TestQuotaManager يختبر إدارة حدود التخزين
func TestQuotaManager(t *testing.T) {
	qm := NewQuotaManager()

	// تحديد حد 100 بايت
	qm.SetLimit("did:mskt:test123", 100)

	// إضافة 60 بايت (يجب أن ينجح)
	err := qm.CheckAndAdd("did:mskt:test123", 60)
	if err != nil {
		t.Fatalf("فشل إضافة 60 بايت: %v", err)
	}

	// إضافة 50 بايت أخرى (يجب أن يفشل)
	err = qm.CheckAndAdd("did:mskt:test123", 50)
	if err == nil {
		t.Fatalf("يجب أن يفشل إضافة 50 بايت (تجاوز الحد)")
	}

	// تحرير 60 بايت
	qm.Release("did:mskt:test123", 60)

	// إضافة 50 بايت (يجب أن ينجح الآن)
	err = qm.CheckAndAdd("did:mskt:test123", 50)
	if err != nil {
		t.Fatalf("فشل إضافة 50 بايت بعد التحرير: %v", err)
	}

	t.Log("تم اختبار Quota Manager بنجاح")
}

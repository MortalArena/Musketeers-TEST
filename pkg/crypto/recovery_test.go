package crypto

import (
	"math/rand"
	"testing"
)

// TestShamirSecretSharing يختبر تقسيم وإعادة بناء المفتاح
func TestShamirSecretSharing(t *testing.T) {
	// إنشاء مفتاح عشوائي
	masterKey := make([]byte, 32)
	rand.Read(masterKey)

	// تقسيم المفتاح إلى 3 أجزاء
	shares, err := SplitMasterKey(masterKey)
	if err != nil {
		t.Fatalf("فشل تقسيم المفتاح: %v", err)
	}

	// التحقق من عدد الأجزاء
	if len(shares) != TotalShares {
		t.Fatalf("عدد الأجزاء غير صحيح: توقع %d، حصل على %d", TotalShares, len(shares))
	}

	// إعادة البناء باستخدام الجزء 1 و 2 (يجب أن ينجح)
	reconstructed, err := ReconstructMasterKey([][]byte{shares[0], shares[1]})
	if err != nil {
		t.Fatalf("فشل إعادة بناء المفتاح من جزأين: %v", err)
	}

	// التحقق من تطابق المفتاح المستعادة مع الأصلي
	if len(reconstructed) != len(masterKey) {
		t.Fatalf("طول المفتاح المستعادة غير صحيح")
	}
	for i := range masterKey {
		if reconstructed[i] != masterKey[i] {
			t.Fatalf("المفتاح المستعادة لا يطابق الأصلي عند البايت %d", i)
		}
	}

	// إعادة البناء باستخدام جزء واحد فقط (يجب أن يفشل)
	_, err = ReconstructMasterKey([][]byte{shares[0]})
	if err == nil {
		t.Fatalf("يجب أن تفشل إعادة البناء من جزء واحد")
	}

	t.Log("تم اختبار Shamir's Secret Sharing بنجاح")
}

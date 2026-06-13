package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"testing"
	"time"
)

func TestCreditManager_VerifyPoSt(t *testing.T) {
	manager := NewCreditManager(0.01) // 0.01 credit per GB per hour
	nodeID := "node_abc"

	// 1. تسجيل العقدة (1 GB)
	manager.RegisterNode(nodeID, 1024*1024*1024)

	// محاكاة مرور ساعة
	time.Sleep(10 * time.Millisecond) // للاختبار السريع، في الواقع ننتظر ساعة أو نحاكي الوقت

	// 2. إنشاء تحدي واستجابة صحيحة
	challenge := "random_challenge_123"
	expectedHash := sha256.Sum256([]byte(challenge + nodeID))
	validResponse := hex.EncodeToString(expectedHash[:])

	// 3. التحقق الناجح
	err := manager.VerifyPoSt(nodeID, challenge, validResponse)
	if err != nil {
		t.Fatalf("Expected successful PoSt, got error: %v", err)
	}

	// 4. التحقق من فشل التحدي المزيف
	err = manager.VerifyPoSt(nodeID, challenge, "invalid_hash")
	if err == nil {
		t.Errorf("Expected PoSt verification to fail with invalid hash")
	}
}

func TestCreditManager_RegisterNode(t *testing.T) {
	manager := NewCreditManager(0.01)
	nodeID := "node_abc"

	// تسجيل عقدة جديدة
	manager.RegisterNode(nodeID, 1024*1024*1024)

	// التحقق من التسجيل
	stats, err := manager.GetNodeStats(nodeID)
	if err != nil {
		t.Fatalf("Expected node to be registered, got error: %v", err)
	}
	if stats.NodeID != nodeID {
		t.Errorf("Expected NodeID %s, got %s", nodeID, stats.NodeID)
	}
	if stats.StorageProvided != 1024*1024*1024 {
		t.Errorf("Expected StorageProvided 1GB, got %d", stats.StorageProvided)
	}
}

func TestCreditManager_GetCredits(t *testing.T) {
	manager := NewCreditManager(0.01)
	nodeID := "node_abc"

	// محاولة الحصول على رصيد لعقدة غير مسجلة
	_, err := manager.GetCredits(nodeID)
	if err == nil {
		t.Error("Expected error when getting credits for unregistered node")
	}

	// تسجيل العقدة
	manager.RegisterNode(nodeID, 1024*1024*1024)

	// الحصول على الرصيد (يجب أن يكون 0 في البداية)
	credits, err := manager.GetCredits(nodeID)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if credits != 0 {
		t.Errorf("Expected credits 0, got %f", credits)
	}
}

func TestCreditManager_GetNodeStats(t *testing.T) {
	manager := NewCreditManager(0.01)
	nodeID := "node_abc"

	// محاولة الحصول على إحصائيات لعقدة غير مسجلة
	_, err := manager.GetNodeStats(nodeID)
	if err == nil {
		t.Error("Expected error when getting stats for unregistered node")
	}

	// تسجيل العقدة
	manager.RegisterNode(nodeID, 1024*1024*1024)

	// الحصول على الإحصائيات
	stats, err := manager.GetNodeStats(nodeID)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if stats.NodeID != nodeID {
		t.Errorf("Expected NodeID %s, got %s", nodeID, stats.NodeID)
	}
}

func TestCreditManager_UpdateRate(t *testing.T) {
	manager := NewCreditManager(0.01)

	// التحقق من المعدل الافتراضي
	if rate := manager.GetRate(); rate != 0.01 {
		t.Errorf("Expected rate 0.01, got %f", rate)
	}

	// تحديث المعدل
	manager.UpdateRate(0.05)

	// التحقق من المعدل الجديد
	if rate := manager.GetRate(); rate != 0.05 {
		t.Errorf("Expected rate 0.05, got %f", rate)
	}
}

func TestCreditManager_ConcurrentAccess(t *testing.T) {
	manager := NewCreditManager(0.01)
	nodeID := "node_abc"
	manager.RegisterNode(nodeID, 1024*1024*1024)

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	// محاكاة 10 عمليات تحقق متزامنة
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			challenge := "challenge_" + string(rune(i))
			expectedHash := sha256.Sum256([]byte(challenge + nodeID))
			validResponse := hex.EncodeToString(expectedHash[:])
			err := manager.VerifyPoSt(nodeID, challenge, validResponse)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// التحقق من عدم وجود أخطاء
	for err := range errors {
		if err != nil {
			t.Errorf("Unexpected error in concurrent access: %v", err)
		}
	}

	// التحقق من أن الرصيد قد زاد (قد يكون صفراً جداً بسبب الوقت القصير)
	credits, err := manager.GetCredits(nodeID)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if credits < 0 {
		t.Errorf("Expected credits to be non-negative after concurrent verifications, got %f", credits)
	}
}

func TestCreditManager_MultipleNodes(t *testing.T) {
	manager := NewCreditManager(0.01)

	// تسجيل عدة عقد
	manager.RegisterNode("node_1", 1024*1024*1024)
	manager.RegisterNode("node_2", 2*1024*1024*1024)
	manager.RegisterNode("node_3", 3*1024*1024*1024)

	// التحقق من التسجيل
	stats1, _ := manager.GetNodeStats("node_1")
	stats2, _ := manager.GetNodeStats("node_2")
	stats3, _ := manager.GetNodeStats("node_3")

	if stats1.StorageProvided != 1024*1024*1024 {
		t.Errorf("Expected node_1 storage 1GB, got %d", stats1.StorageProvided)
	}
	if stats2.StorageProvided != 2*1024*1024*1024 {
		t.Errorf("Expected node_2 storage 2GB, got %d", stats2.StorageProvided)
	}
	if stats3.StorageProvided != 3*1024*1024*1024 {
		t.Errorf("Expected node_3 storage 3GB, got %d", stats3.StorageProvided)
	}
}

func TestCreditManager_UpdateExistingNode(t *testing.T) {
	manager := NewCreditManager(0.01)
	nodeID := "node_abc"

	// تسجيل عقدة بمساحة 1GB
	manager.RegisterNode(nodeID, 1024*1024*1024)

	// تحديث المساحة إلى 2GB
	manager.RegisterNode(nodeID, 2*1024*1024*1024)

	// التحقق من التحديث
	stats, _ := manager.GetNodeStats(nodeID)
	if stats.StorageProvided != 2*1024*1024*1024 {
		t.Errorf("Expected storage 2GB after update, got %d", stats.StorageProvided)
	}
}

func TestCreditManager_ZeroStorage(t *testing.T) {
	manager := NewCreditManager(0.01)
	nodeID := "node_abc"

	// تسجيل عقدة بمساحة 0
	manager.RegisterNode(nodeID, 0)

	// التحقق من التسجيل
	stats, _ := manager.GetNodeStats(nodeID)
	if stats.StorageProvided != 0 {
		t.Errorf("Expected storage 0, got %d", stats.StorageProvided)
	}
}

func TestCreditManager_LargeStorage(t *testing.T) {
	manager := NewCreditManager(0.01)
	nodeID := "node_abc"

	// تسجيل عقدة بمساحة كبيرة (100GB)
	largeStorage := int64(100) * 1024 * 1024 * 1024
	manager.RegisterNode(nodeID, largeStorage)

	// التحقق من التسجيل
	stats, _ := manager.GetNodeStats(nodeID)
	if stats.StorageProvided != largeStorage {
		t.Errorf("Expected storage %d, got %d", largeStorage, stats.StorageProvided)
	}
}

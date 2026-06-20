package crypto

import (
	"testing"
	"time"
)

func TestIdentityLimiterHumanIdentities(t *testing.T) {
	limiter := NewIdentityLimiter()

	// اختبار إنشاء هويات بشرية متعددة من عقود مختلفة
	for i := 0; i < 8; i++ {
		nodeID := "node-" + string(rune('a'+i))
		err := limiter.CanCreateIdentity(nodeID, IdentityTypeHuman)
		if err != nil {
			t.Fatalf("Failed to create human identity %d: %v", i, err)
		}
		limiter.RecordIdentityCreation(nodeID, IdentityTypeHuman)
	}

	// يجب أن يفشل إنشاء هوية بشرية جديدة
	err := limiter.CanCreateIdentity("node-8", IdentityTypeHuman)
	if err == nil {
		t.Error("Should fail when human identity limit is reached")
	}

	// التحقق من العدد
	count := limiter.GetIdentityCount(IdentityTypeHuman)
	if count != 8 {
		t.Errorf("Expected 8 human identities, got %d", count)
	}
}

func TestIdentityLimiterAgentIdentities(t *testing.T) {
	limiter := NewIdentityLimiter()

	// اختبار إنشاء هويات وكلاء متعددة من عقود مختلفة
	for i := 0; i < 128; i++ {
		nodeID := "agent-node-" + string(rune('a'+i%26)) + "-" + string(rune('0'+i%10))
		err := limiter.CanCreateIdentity(nodeID, IdentityTypeAgent)
		if err != nil {
			t.Fatalf("Failed to create agent identity %d: %v", i, err)
		}
		limiter.RecordIdentityCreation(nodeID, IdentityTypeAgent)
	}

	// يجب أن يفشل إنشاء هوية وكيل جديدة
	err := limiter.CanCreateIdentity("agent-node-128", IdentityTypeAgent)
	if err == nil {
		t.Error("Should fail when agent identity limit is reached")
	}

	// التحقق من العدد
	count := limiter.GetIdentityCount(IdentityTypeAgent)
	if count != 128 {
		t.Errorf("Expected 128 agent identities, got %d", count)
	}
}

func TestIdentityLimiterCooldown(t *testing.T) {
	limiter := NewIdentityLimiter()

	// تعيين حدود منخفضة للاختبار
	limiter.SetLimits(2, 2)

	nodeID := "node-3"

	// إنشاء هوية بشرية
	err := limiter.CanCreateIdentity(nodeID, IdentityTypeHuman)
	if err != nil {
		t.Fatalf("Failed to create first human identity: %v", err)
	}
	limiter.RecordIdentityCreation(nodeID, IdentityTypeHuman)

	// محاولة إنشاء هوية أخرى فوراً (يجب أن تفشل بسبب cooldown)
	err = limiter.CanCreateIdentity(nodeID, IdentityTypeHuman)
	if err == nil {
		t.Error("Should fail due to cooldown")
	}

	// الانتظار حتى انتهاء cooldown
	time.Sleep(6 * time.Minute) // humanCooldown is 5 minutes

	// الآن يجب أن تنجح
	err = limiter.CanCreateIdentity(nodeID, IdentityTypeHuman)
	if err != nil {
		t.Errorf("Should succeed after cooldown: %v", err)
	}
}

func TestIdentityLimiterSeparateTypes(t *testing.T) {
	limiter := NewIdentityLimiter()

	// إنشاء هويات بشرية من عقود مختلفة
	for i := 0; i < 8; i++ {
		nodeID := "human-node-" + string(rune('a'+i))
		err := limiter.CanCreateIdentity(nodeID, IdentityTypeHuman)
		if err != nil {
			t.Fatalf("Failed to create human identity %d: %v", i, err)
		}
		limiter.RecordIdentityCreation(nodeID, IdentityTypeHuman)
	}

	// يجب أن يفشل إنشاء هوية بشرية جديدة
	err := limiter.CanCreateIdentity("human-node-8", IdentityTypeHuman)
	if err == nil {
		t.Error("Should fail when human identity limit is reached")
	}

	// لكن يجب أن تنجح إنشاء هوية وكيل (أنواع منفصلة)
	err = limiter.CanCreateIdentity("agent-node-1", IdentityTypeAgent)
	if err != nil {
		t.Errorf("Should succeed for agent identity: %v", err)
	}
}

func TestIdentityLimiterMultipleNodes(t *testing.T) {
	limiter := NewIdentityLimiter()

	// إنشاء هويات من عقود مختلفة
	for i := 0; i < 8; i++ {
		nodeID := "node-" + string(rune('a'+i))
		err := limiter.CanCreateIdentity(nodeID, IdentityTypeHuman)
		if err != nil {
			t.Fatalf("Failed to create human identity for node %s: %v", nodeID, err)
		}
		limiter.RecordIdentityCreation(nodeID, IdentityTypeHuman)
	}

	// التحقق من العدد الإجمالي
	count := limiter.GetIdentityCount(IdentityTypeHuman)
	if count != 8 {
		t.Errorf("Expected 8 human identities across nodes, got %d", count)
	}
}

func TestIdentityLimiterClear(t *testing.T) {
	limiter := NewIdentityLimiter()

	// إنشاء بعض الهويات من عقود مختلفة
	for i := 0; i < 5; i++ {
		nodeID := "clear-node-" + string(rune('a'+i))
		err := limiter.CanCreateIdentity(nodeID, IdentityTypeHuman)
		if err != nil {
			t.Fatalf("Failed to create identity: %v", err)
		}
		limiter.RecordIdentityCreation(nodeID, IdentityTypeHuman)
	}

	// مسح السجلات
	limiter.Clear()

	// التحقق من أن العدد صفر
	count := limiter.GetIdentityCount(IdentityTypeHuman)
	if count != 0 {
		t.Errorf("Expected 0 identities after clear, got %d", count)
	}

	// يجب أن تنجح إنشاء هوية جديدة
	err := limiter.CanCreateIdentity("clear-node-new", IdentityTypeHuman)
	if err != nil {
		t.Errorf("Should succeed after clear: %v", err)
	}
}

func TestIdentityLimiterGetLimits(t *testing.T) {
	limiter := NewIdentityLimiter()

	maxHuman, maxAgent := limiter.GetLimits()

	if maxHuman != 8 {
		t.Errorf("Expected max human limit 8, got %d", maxHuman)
	}
	if maxAgent != 128 {
		t.Errorf("Expected max agent limit 128, got %d", maxAgent)
	}
}

func TestIdentityLimiterSetLimits(t *testing.T) {
	limiter := NewIdentityLimiter()

	// تعيين حدود جديدة
	limiter.SetLimits(16, 128)

	maxHuman, maxAgent := limiter.GetLimits()

	if maxHuman != 16 {
		t.Errorf("Expected max human limit 16, got %d", maxHuman)
	}
	if maxAgent != 128 {
		t.Errorf("Expected max agent limit 128, got %d", maxAgent)
	}
}

package ledger

import (
	"sync"
	"testing"
)

func TestCostTracker_CheckAndDeduct(t *testing.T) {
	tracker := NewCostTracker()
	workflowID := "wf_123"

	// 1. تحديد ميزانية
	tracker.SetBudget(workflowID, 1.00) // 1 دولار

	// 2. خصم ناجح
	err := tracker.CheckAndDeduct(workflowID, 0.40)
	if err != nil {
		t.Fatalf("Unexpected error on valid deduction: %v", err)
	}
	if remaining := tracker.GetRemaining(workflowID); remaining != 0.60 {
		t.Errorf("Expected remaining 0.60, got %f", remaining)
	}

	// 3. محاولة خصم تتجاوز الرصيد (يجب أن تفشل)
	err = tracker.CheckAndDeduct(workflowID, 0.70)
	if err != ErrInsufficientFunds {
		t.Errorf("Expected ErrInsufficientFunds, got %v", err)
	}

	// 4. التأكد من أن الرصيد لم يتأثر بالمحاولة الفاشلة
	if remaining := tracker.GetRemaining(workflowID); remaining != 0.60 {
		t.Errorf("Expected remaining to stay 0.60, got %f", remaining)
	}
}

func TestCostTracker_SetBudget(t *testing.T) {
	tracker := NewCostTracker()
	workflowID := "wf_123"

	// تحديد ميزانية
	tracker.SetBudget(workflowID, 100.00)
	if remaining := tracker.GetRemaining(workflowID); remaining != 100.00 {
		t.Errorf("Expected remaining 100.00, got %f", remaining)
	}

	// تحديث الميزانية
	tracker.SetBudget(workflowID, 200.00)
	if remaining := tracker.GetRemaining(workflowID); remaining != 200.00 {
		t.Errorf("Expected remaining 200.00, got %f", remaining)
	}
}

func TestCostTracker_NoBudgetSet(t *testing.T) {
	tracker := NewCostTracker()
	workflowID := "wf_123"

	// محاولة خصم بدون تحديد ميزانية
	err := tracker.CheckAndDeduct(workflowID, 0.40)
	if err == nil {
		t.Error("Expected error when no budget is set")
	}
}

func TestCostTracker_GetRemaining(t *testing.T) {
	tracker := NewCostTracker()
	workflowID := "wf_123"

	// الحصول على الرصيد بدون تحديد ميزانية
	remaining := tracker.GetRemaining(workflowID)
	if remaining != 0 {
		t.Errorf("Expected remaining 0, got %f", remaining)
	}

	// تحديد ميزانية والحصول على الرصيد
	tracker.SetBudget(workflowID, 50.00)
	remaining = tracker.GetRemaining(workflowID)
	if remaining != 50.00 {
		t.Errorf("Expected remaining 50.00, got %f", remaining)
	}
}

func TestCostTracker_ConcurrentAccess(t *testing.T) {
	tracker := NewCostTracker()
	workflowID := "wf_123"
	tracker.SetBudget(workflowID, 100.00)

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	// محاكاة 10 عمليات خصم متزامنة
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer func() { recover() }()
			defer wg.Done()
			err := tracker.CheckAndDeduct(workflowID, 5.00)
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

	// التحقق من الرصيد النهائي (100 - 10*5 = 50)
	remaining := tracker.GetRemaining(workflowID)
	if remaining != 50.00 {
		t.Errorf("Expected remaining 50.00 after concurrent deductions, got %f", remaining)
	}
}

func TestCostTracker_MultipleWorkflows(t *testing.T) {
	tracker := NewCostTracker()

	// تحديد ميزانيات لعدة workflows
	tracker.SetBudget("wf_1", 100.00)
	tracker.SetBudget("wf_2", 200.00)
	tracker.SetBudget("wf_3", 300.00)

	// خصم من كل workflow
	tracker.CheckAndDeduct("wf_1", 10.00)
	tracker.CheckAndDeduct("wf_2", 20.00)
	tracker.CheckAndDeduct("wf_3", 30.00)

	// التحقق من الرصيد المتبقي لكل workflow
	if remaining := tracker.GetRemaining("wf_1"); remaining != 90.00 {
		t.Errorf("Expected wf_1 remaining 90.00, got %f", remaining)
	}
	if remaining := tracker.GetRemaining("wf_2"); remaining != 180.00 {
		t.Errorf("Expected wf_2 remaining 180.00, got %f", remaining)
	}
	if remaining := tracker.GetRemaining("wf_3"); remaining != 270.00 {
		t.Errorf("Expected wf_3 remaining 270.00, got %f", remaining)
	}
}

func TestCostTracker_ZeroDeduction(t *testing.T) {
	tracker := NewCostTracker()
	workflowID := "wf_123"
	tracker.SetBudget(workflowID, 100.00)

	// خصم صفر
	err := tracker.CheckAndDeduct(workflowID, 0.00)
	if err != nil {
		t.Fatalf("Unexpected error on zero deduction: %v", err)
	}
	if remaining := tracker.GetRemaining(workflowID); remaining != 100.00 {
		t.Errorf("Expected remaining 100.00, got %f", remaining)
	}
}

func TestCostTracker_ExactBudget(t *testing.T) {
	tracker := NewCostTracker()
	workflowID := "wf_123"
	tracker.SetBudget(workflowID, 100.00)

	// خصم المبلغ بالضبط
	err := tracker.CheckAndDeduct(workflowID, 100.00)
	if err != nil {
		t.Fatalf("Unexpected error on exact deduction: %v", err)
	}
	if remaining := tracker.GetRemaining(workflowID); remaining != 0.00 {
		t.Errorf("Expected remaining 0.00, got %f", remaining)
	}
}

func TestCostTracker_NegativeCost(t *testing.T) {
	tracker := NewCostTracker()
	workflowID := "wf_123"
	tracker.SetBudget(workflowID, 100.00)

	// محاولة خصم مبلغ سلبي (يجب أن ينجح لأن الكود لا يتحقق من القيم السالبة)
	err := tracker.CheckAndDeduct(workflowID, -10.00)
	if err != nil {
		t.Fatalf("Unexpected error on negative deduction: %v", err)
	}
	// الرصيد يجب أن يزيد
	if remaining := tracker.GetRemaining(workflowID); remaining != 110.00 {
		t.Errorf("Expected remaining 110.00, got %f", remaining)
	}
}

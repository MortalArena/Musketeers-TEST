package ledger

import (
	"fmt"
	"sync"
)

var ErrInsufficientFunds = fmt.Errorf("insufficient funds: budget exceeded")

// CostTracker يدير الميزانيات الصارمة لكل وكيل أو مهمة
type CostTracker struct {
	mu      sync.RWMutex
	budgets map[string]float64 // workflow_id أو agent_id -> الميزانية المتبقية بالدولار
}

// NewCostTracker ينشئ متتبع تكاليف جديد
func NewCostTracker() *CostTracker {
	return &CostTracker{
		budgets: make(map[string]float64),
	}
}

// SetBudget يحدد ميزانية جديدة أو يحدث موجودة
func (c *CostTracker) SetBudget(id string, amount float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.budgets[id] = amount
}

// CheckAndDeduct يتحقق من الرصيد ويخصم المبلغ بشكل ذري (Atomic)
func (c *CostTracker) CheckAndDeduct(id string, cost float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	currentBudget, exists := c.budgets[id]
	if !exists {
		return fmt.Errorf("no budget set for id: %s", id)
	}

	if currentBudget < cost {
		return ErrInsufficientFunds
	}

	c.budgets[id] = currentBudget - cost
	return nil
}

// GetRemaining يعود بالميزانية المتبقية
func (c *CostTracker) GetRemaining(id string) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.budgets[id]
}

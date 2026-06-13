package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// NodeStats إحصائيات العقدة
type NodeStats struct {
	NodeID          string
	StorageProvided int64   // بالبايت
	UptimeHours     float64
	CreditsEarned   float64
	LastChallenge   time.Time
}

// CreditManager يدير محاسبة المكافآت والتحديات
type CreditManager struct {
	mu    sync.RWMutex
	nodes map[string]*NodeStats
	rate  float64 // Credits per GB per hour
}

// NewCreditManager ينشئ مدير ائتمان جديد
func NewCreditManager(rewardRate float64) *CreditManager {
	return &CreditManager{
		nodes: make(map[string]*NodeStats),
		rate:  rewardRate,
	}
}

// RegisterNode يسجل عقدة جديدة أو يحدث بياناتها
func (c *CreditManager) RegisterNode(nodeID string, storageBytes int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.nodes[nodeID]; !exists {
		c.nodes[nodeID] = &NodeStats{NodeID: nodeID}
	}
	c.nodes[nodeID].StorageProvided = storageBytes
	c.nodes[nodeID].LastChallenge = time.Now()
}

// VerifyPoSt يتحقق من تحدي إثبات التخزين ويمنح المكافأة
func (c *CreditManager) VerifyPoSt(nodeID string, challengeData string, responseHash string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	stats, exists := c.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node not registered")
	}

	// محاكاة التحقق من التحدي: Hash(ChallengeData + NodeID) يجب أن يساوي ResponseHash
	expectedHash := sha256.Sum256([]byte(challengeData + nodeID))
	if hex.EncodeToString(expectedHash[:]) != responseHash {
		// عقوبة بسيطة أو تجاهل
		return fmt.Errorf("PoSt verification failed")
	}

	// حساب المكافأة (مبسطة: ساعات التشغيل * المساحة * المعدل)
	hoursSinceLastChallenge := time.Since(stats.LastChallenge).Hours()
	reward := (float64(stats.StorageProvided) / (1024 * 1024 * 1024)) * hoursSinceLastChallenge * c.rate

	stats.CreditsEarned += reward
	stats.LastChallenge = time.Now()

	return nil
}

// GetCredits يعود برصيد العقدة
func (c *CreditManager) GetCredits(nodeID string) (float64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats, exists := c.nodes[nodeID]
	if !exists {
		return 0, fmt.Errorf("node not found")
	}
	return stats.CreditsEarned, nil
}

// GetNodeStats يعود بإحصائيات العقدة
func (c *CreditManager) GetNodeStats(nodeID string) (*NodeStats, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats, exists := c.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node not found")
	}
	return stats, nil
}

// UpdateRate يحدث معدل المكافأة
func (c *CreditManager) UpdateRate(newRate float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rate = newRate
}

// GetRate يعود بمعدل المكافأة الحالي
func (c *CreditManager) GetRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.rate
}

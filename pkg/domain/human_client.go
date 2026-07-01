package domain

import "time"

// HumanClient Domain Model - الكيان الأساسي للعميل البشري
type HumanClient struct {
	ID        string
	Name      string
	Status    HumanClientStatus
	JoinedAt  time.Time
	LastSeen  time.Time
	SessionID string
}

// HumanClientStatus Value Object
type HumanClientStatus string

const (
	HumanClientStatusOnline  HumanClientStatus = "online"
	HumanClientStatusOffline HumanClientStatus = "offline"
	HumanClientStatusAway    HumanClientStatus = "away"
	HumanClientStatusBusy    HumanClientStatus = "busy"
)

// IsValid يتحقق من صحة HumanClient
func (hc *HumanClient) IsValid() bool {
	if hc.ID == "" {
		return false
	}
	if hc.Name == "" {
		return false
	}
	if hc.Status == "" {
		return false
	}
	return true
}

// IsOnline يتحقق مما إذا كان العميل متصلاً
func (hc *HumanClient) IsOnline() bool {
	return hc.Status == HumanClientStatusOnline
}

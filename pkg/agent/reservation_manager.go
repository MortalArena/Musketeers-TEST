package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ReservationManager مدير حجز الوكلاء المحليين
// [WHY] يمنع تضارب الوكلاء المحليين عند ربطهم بجلسات متعددة
// [HOW] يحجز الوكلاء المحليين لجلسات محددة مع timeout
// [SAFETY] يستخدم RWMutex لحماية الحالة
type ReservationManager struct {
	reservations map[string]*AgentReservation // agentID -> reservation
	logger       *zap.Logger
	mu           sync.RWMutex
}

// AgentReservation حجز وكيل
type AgentReservation struct {
	AgentID        string    // معرف الوكيل
	ReservedBy     string    // الجلسة المحجوزة حالياً
	ReservedAt     time.Time // وقت الحجز
	Timeout        time.Duration // مدة الحجز
	ExpiresAt      time.Time // وقت انتهاء الحجز
	RequestQueue   []*ReservationRequest // قائمة انتظار الطلبات
	mu             sync.RWMutex
}

// ReservationRequest طلب حجز
type ReservationRequest struct {
	SessionID    string
	RequestTime  time.Time
	ResponseChan chan *ReservationResponse
}

// ReservationResponse رد حجز
type ReservationResponse struct {
	Granted      bool
	ReservedAt   time.Time
	ExpiresAt    time.Time
}

// NewReservationManager ينشئ مدير حجز جديد
func NewReservationManager(logger *zap.Logger) *ReservationManager {
	return &ReservationManager{
		reservations: make(map[string]*AgentReservation),
		logger:       logger,
	}
}

// ReserveAgent يحجز وكيل محلي لجلسة محددة
func (rm *ReservationManager) ReserveAgent(ctx context.Context, agentID, sessionID string, timeout time.Duration) (*ReservationResponse, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// التحقق من وجود الحجز
	reservation, exists := rm.reservations[agentID]
	if !exists {
		// إنشاء حجز جديد
		reservation = &AgentReservation{
			AgentID:      agentID,
			ReservedBy:   sessionID,
			ReservedAt:   time.Now(),
			Timeout:      timeout,
			ExpiresAt:    time.Now().Add(timeout),
			RequestQueue: make([]*ReservationRequest, 0),
		}
		rm.reservations[agentID] = reservation
		
		rm.logger.Info("تم حجز وكيل جديد",
			zap.String("agent_id", agentID),
			zap.String("session_id", sessionID),
			zap.Duration("timeout", timeout),
		)
		
		return &ReservationResponse{
			Granted:    true,
			ReservedAt: reservation.ReservedAt,
			ExpiresAt:  reservation.ExpiresAt,
		}, nil
	}

	// التحقق من انتهاء الحجز الحالي
	if time.Now().After(reservation.ExpiresAt) {
		// الحجز منتهي، يمكن حجز الوكيل
		reservation.ReservedBy = sessionID
		reservation.ReservedAt = time.Now()
		reservation.ExpiresAt = time.Now().Add(timeout)
		
		rm.logger.Info("تم حجز وكيل منتهي الحجز",
			zap.String("agent_id", agentID),
			zap.String("session_id", sessionID),
		)
		
		return &ReservationResponse{
			Granted:    true,
			ReservedAt: reservation.ReservedAt,
			ExpiresAt:  reservation.ExpiresAt,
		}, nil
	}

	// الوكيل محجوز حالياً
	if reservation.ReservedBy == sessionID {
		// نفس الجلسة، تمديد الحجز
		reservation.ExpiresAt = time.Now().Add(timeout)
		
		rm.logger.Info("تم تمديد حجز الوكيل",
			zap.String("agent_id", agentID),
			zap.String("session_id", sessionID),
		)
		
		return &ReservationResponse{
			Granted:    true,
			ReservedAt: reservation.ReservedAt,
			ExpiresAt:  reservation.ExpiresAt,
		}, nil
	}

	// الوكيل محجوز لجلسة أخرى
	rm.logger.Info("الوكيل محجوز لجلسة أخرى",
		zap.String("agent_id", agentID),
		zap.String("reserved_by", reservation.ReservedBy),
		zap.String("requested_by", sessionID),
	)

	return &ReservationResponse{
		Granted: false,
	}, fmt.Errorf("agent %s is reserved by session %s until %s", agentID, reservation.ReservedBy, reservation.ExpiresAt)
}

// ReleaseAgent يطلق حجز وكيل
func (rm *ReservationManager) ReleaseAgent(agentID, sessionID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	reservation, exists := rm.reservations[agentID]
	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}

	if reservation.ReservedBy != sessionID {
		return fmt.Errorf("agent %s is reserved by session %s, not %s", agentID, reservation.ReservedBy, sessionID)
	}

	// إطلاق الحجز
	delete(rm.reservations, agentID)

	rm.logger.Info("تم إطلاق حجز الوكيل",
		zap.String("agent_id", agentID),
		zap.String("session_id", sessionID),
	)

	return nil
}

// GetReservation يحصل على حجز وكيل
func (rm *ReservationManager) GetReservation(agentID string) (*AgentReservation, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	reservation, exists := rm.reservations[agentID]
	if !exists {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	return reservation, nil
}

// IsAgentAvailable يتحقق من توفر وكيل
func (rm *ReservationManager) IsAgentAvailable(agentID string) bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	reservation, exists := rm.reservations[agentID]
	if !exists {
		return true
	}

	// التحقق من انتهاء الحجز
	return time.Now().After(reservation.ExpiresAt)
}

// GetStats يحصل على إحصائيات المدير
func (rm *ReservationManager) GetStats() map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	totalReservations := len(rm.reservations)
	expiredReservations := 0
	activeReservations := 0

	for _, reservation := range rm.reservations {
		if time.Now().After(reservation.ExpiresAt) {
			expiredReservations++
		} else {
			activeReservations++
		}
	}

	return map[string]interface{}{
		"total_reservations":   totalReservations,
		"active_reservations":  activeReservations,
		"expired_reservations": expiredReservations,
	}
}

// CleanupExpiredReservations ينظف الحجوز المنتهية
func (rm *ReservationManager) CleanupExpiredReservations() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for agentID, reservation := range rm.reservations {
		if time.Now().After(reservation.ExpiresAt) {
			delete(rm.reservations, agentID)
			rm.logger.Info("تم تنظيف حجز منتهي",
				zap.String("agent_id", agentID),
			)
		}
	}
}

// StartCleanupScheduler يبدأ مجدول تنظيف دوري
func (rm *ReservationManager) StartCleanupScheduler(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			rm.CleanupExpiredReservations()
		}
	}()
	
	rm.logger.Info("بدء مجدول تنظيف الحجوز المنتهية",
		zap.Duration("interval", interval),
	)
}

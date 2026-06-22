package session

import (
	"context"
	"fmt"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

// SessionBridgeManager مدير جسور الجلسات
// [WHY] يدير عدة جسور لربط جلسات متعددة
// [HOW] يخزن الجسور في map ويسمح بإنشاء وإيقاف وإدارة الجسور
// [SAFETY] يستخدم RWMutex لحماية الحالة
type SessionBridgeManager struct {
	bridges  map[string]*SessionBridge // bridgeID -> bridge
	sessions map[string][]string       // sessionID -> []bridgeIDs
	eventBus *eventbus.EventBus
	logger   *zap.Logger
	mu       sync.RWMutex
}

// NewSessionBridgeManager ينشئ مدير جسور جديد
func NewSessionBridgeManager(eventBus *eventbus.EventBus, logger *zap.Logger) *SessionBridgeManager {
	return &SessionBridgeManager{
		bridges:  make(map[string]*SessionBridge),
		sessions: make(map[string][]string),
		eventBus: eventBus,
		logger:   logger,
	}
}

// CreateBridge ينشئ جسر جديد بين جلستين
func (sbm *SessionBridgeManager) CreateBridge(ctx context.Context, config *BridgeConfig) (*SessionBridge, error) {
	sbm.mu.Lock()
	defer sbm.mu.Unlock()

	// التحقق من عدم وجود الجسر بالفعل
	if _, exists := sbm.bridges[config.BridgeID]; exists {
		return nil, fmt.Errorf("bridge already exists: %s", config.BridgeID)
	}

	// إنشاء الجسر
	bridge := NewSessionBridge(config, sbm.eventBus, sbm.logger)

	// بدء الجسر
	if err := bridge.Start(); err != nil {
		return nil, fmt.Errorf("failed to start bridge: %w", err)
	}

	// تخزين الجسر
	sbm.bridges[config.BridgeID] = bridge

	// إضافة الجسر إلى الجلسات
	sbm.sessions[config.SourceID] = append(sbm.sessions[config.SourceID], config.BridgeID)
	sbm.sessions[config.TargetID] = append(sbm.sessions[config.TargetID], config.BridgeID)

	sbm.logger.Info("تم إنشاء جسر جلسة جديد",
		zap.String("bridge_id", config.BridgeID),
		zap.String("source", config.SourceID),
		zap.String("target", config.TargetID),
		zap.String("type", string(config.BridgeType)),
	)

	return bridge, nil
}

// GetBridge يحصل على جسر محدد
func (sbm *SessionBridgeManager) GetBridge(bridgeID string) (*SessionBridge, error) {
	sbm.mu.RLock()
	defer sbm.mu.RUnlock()

	bridge, exists := sbm.bridges[bridgeID]
	if !exists {
		return nil, fmt.Errorf("bridge not found: %s", bridgeID)
	}

	return bridge, nil
}

// GetBridgesBySession يحصل على جميع الجسور لجلسة محددة
func (sbm *SessionBridgeManager) GetBridgesBySession(sessionID string) []*SessionBridge {
	sbm.mu.RLock()
	defer sbm.mu.RUnlock()

	bridgeIDs, exists := sbm.sessions[sessionID]
	if !exists {
		return []*SessionBridge{}
	}

	bridges := make([]*SessionBridge, 0, len(bridgeIDs))
	for _, bridgeID := range bridgeIDs {
		if bridge, exists := sbm.bridges[bridgeID]; exists {
			bridges = append(bridges, bridge)
		}
	}

	return bridges
}

// GetAllBridges يحصل على جميع الجسور
func (sbm *SessionBridgeManager) GetAllBridges() []*SessionBridge {
	sbm.mu.RLock()
	defer sbm.mu.RUnlock()

	bridges := make([]*SessionBridge, 0, len(sbm.bridges))
	for _, bridge := range sbm.bridges {
		bridges = append(bridges, bridge)
	}

	return bridges
}

// StopBridge يوقف جسر محدد
func (sbm *SessionBridgeManager) StopBridge(bridgeID string) error {
	sbm.mu.Lock()
	defer sbm.mu.Unlock()

	bridge, exists := sbm.bridges[bridgeID]
	if !exists {
		return fmt.Errorf("bridge not found: %s", bridgeID)
	}

	// إيقاف الجسر
	if err := bridge.Stop(); err != nil {
		return fmt.Errorf("failed to stop bridge: %w", err)
	}

	// إزالة الجسر من الجلسات
	sbm.removeBridgeFromSessions(bridgeID)

	// حذف الجسر
	delete(sbm.bridges, bridgeID)

	sbm.logger.Info("تم إيقاف جسر الجلسة", zap.String("bridge_id", bridgeID))

	return nil
}

// removeBridgeFromSessions يزيل الجسر من الجلسات
func (sbm *SessionBridgeManager) removeBridgeFromSessions(bridgeID string) {
	for sessionID, bridgeIDs := range sbm.sessions {
		for i, id := range bridgeIDs {
			if id == bridgeID {
				sbm.sessions[sessionID] = append(bridgeIDs[:i], bridgeIDs[i+1:]...)
				break
			}
		}
	}
}

// StopAllBridges يوقف جميع الجسور
func (sbm *SessionBridgeManager) StopAllBridges() error {
	sbm.mu.Lock()
	defer sbm.mu.Unlock()

	for bridgeID, bridge := range sbm.bridges {
		if err := bridge.Stop(); err != nil {
			sbm.logger.Error("فشل إيقاف جسر",
				zap.String("bridge_id", bridgeID),
				zap.Error(err),
			)
		}
	}

	sbm.bridges = make(map[string]*SessionBridge)
	sbm.sessions = make(map[string][]string)

	sbm.logger.Info("تم إيقاف جميع جسور الجلسات")

	return nil
}

// GetStats يحصل على إحصائيات المدير
func (sbm *SessionBridgeManager) GetStats() map[string]interface{} {
	sbm.mu.RLock()
	defer sbm.mu.RUnlock()

	totalMessages := int64(0)
	totalBytes := int64(0)
	activeBridges := 0

	for _, bridge := range sbm.bridges {
		stats := bridge.GetStats()
		if messages, ok := stats["messages_sent"].(int64); ok {
			totalMessages += messages
		}
		if bytes, ok := stats["bytes_transferred"].(int64); ok {
			totalBytes += bytes
		}
		if status, ok := stats["status"].(BridgeStatus); ok && status == BridgeStatusActive {
			activeBridges++
		}
	}

	return map[string]interface{}{
		"total_bridges":       len(sbm.bridges),
		"active_bridges":      activeBridges,
		"total_sessions":      len(sbm.sessions),
		"total_messages":      totalMessages,
		"total_bytes":         totalBytes,
		"bridges_per_session": sbm.getBridgesPerSession(),
	}
}

// getBridgesPerSession يحصل على متوسط عدد الجسور لكل جلسة
func (sbm *SessionBridgeManager) getBridgesPerSession() float64 {
	if len(sbm.sessions) == 0 {
		return 0
	}

	totalBridges := 0
	for _, bridgeIDs := range sbm.sessions {
		totalBridges += len(bridgeIDs)
	}

	return float64(totalBridges) / float64(len(sbm.sessions))
}

// SendMessageAcrossBridges يرسل رسالة عبر جميع الجسور لجلسة محددة
func (sbm *SessionBridgeManager) SendMessageAcrossBridges(ctx context.Context, sessionID string, msg *BridgeMessage) error {
	bridges := sbm.GetBridgesBySession(sessionID)

	if len(bridges) == 0 {
		return fmt.Errorf("no bridges found for session: %s", sessionID)
	}

	// إرسال الرسالة عبر جميع الجسور
	for _, bridge := range bridges {
		if err := bridge.SendMessage(ctx, msg); err != nil {
			sbm.logger.Error("فشل إرسال رسالة عبر جسر",
				zap.String("bridge_id", bridge.bridgeID),
				zap.Error(err),
			)
		}
	}

	return nil
}

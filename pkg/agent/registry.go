package agent

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AgentRegistry سجل الوكلاء - يدير تسجيل وتتبع الوكلاء
type AgentRegistry struct {
	agents   map[string]UnifiedAgent   // agentID -> agent
	metadata map[string]*AgentMetadata // agentID -> metadata
	stats    map[string]*AgentStats    // agentID -> stats

	// حالة العميل البشري
	humanClient *HumanClientStatus // حالة العميل البشري

	mu     sync.RWMutex
	logger *zap.Logger
}

// HumanClientStatus حالة العميل البشري
type HumanClientStatus struct {
	UserID      string                 `json:"user_id"`
	Name        string                 `json:"name"`
	Status      string                 `json:"status"` // online, offline, busy, away
	LastSeen    time.Time              `json:"last_seen"`
	Preferences map[string]interface{} `json:"preferences"`
	AllowOnline bool                   `json:"allow_online"` // خيار للعميل للاختار بين أونلاين وأوفلاين
}

// AgentMetadata بيانات وصفية للوكيل
type AgentMetadata struct {
	AgentID       string                 `json:"agent_id"`
	Name          string                 `json:"name"`
	Type          AgentType              `json:"type"`
	Provider      string                 `json:"provider"`
	Model         string                 `json:"model"`
	Version       string                 `json:"version"`
	Endpoint      string                 `json:"endpoint"`
	AuthMethod    string                 `json:"auth_method"`
	MaxTokens     int                    `json:"max_tokens"`
	ContextWindow int                    `json:"context_window"`
	RegisteredAt  time.Time              `json:"registered_at"`
	LastSeen      time.Time              `json:"last_seen"`
	Tags          []string               `json:"tags"`
	Config        map[string]interface{} `json:"config"`
	// معلومات التتبع المتعدد
	InstanceID      string `json:"instance_id"`       // معرف فريد للنسخة (مثلاً: claude-4.8-1, claude-4.8-2)
	HumanClientID   string `json:"human_client_id"`   // معرف العميل البشري المالك
	HumanClientName string `json:"human_client_name"` // اسم العميل البشري المالك
	APIKeyID        string `json:"api_key_id"`        // معرف مفتاح API (للتمييز بين مفاتيح متعددة)
	APIKeyLabel     string `json:"api_key_label"`     // وصف مفتاح API (مثلاً: "Production Key #1")
	SessionID       string `json:"session_id"`        // معرف الجلسة الحالية
}

// AgentStats إحصائيات الوكيل
type AgentStats struct {
	AgentID         string        `json:"agent_id"`
	TotalTasks      int           `json:"total_tasks"`
	CompletedTasks  int           `json:"completed_tasks"`
	FailedTasks     int           `json:"failed_tasks"`
	TotalTokens     int           `json:"total_tokens"`
	TotalDuration   time.Duration `json:"total_duration"`
	AvgResponseTime time.Duration `json:"avg_response_time"`
	SuccessRate     float64       `json:"success_rate"`
	LastUsed        time.Time     `json:"last_used"`
}

// NewAgentRegistry ينشئ سجل وكلاء جديد
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		agents:   make(map[string]UnifiedAgent),
		metadata: make(map[string]*AgentMetadata),
		stats:    make(map[string]*AgentStats),
		logger:   zap.NewNop(), // سيتم استبداله بـ logger حقيقي
	}
}

// SetLogger يضبط logger
func (ar *AgentRegistry) SetLogger(logger *zap.Logger) {
	ar.mu.Lock()
	defer ar.mu.Unlock()
	ar.logger = logger
}

// Register يسجل وكيل جديد
func (ar *AgentRegistry) Register(agent UnifiedAgent, metadata *AgentMetadata) error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	info := agent.GetInfo()
	agentID := info.ID

	// التحقق من عدم وجود الوكيل بالفعل
	if _, exists := ar.agents[agentID]; exists {
		return fmt.Errorf("agent already registered: %s", agentID)
	}

	// تسجيل الوكيل
	ar.agents[agentID] = agent

	// تعيين البيانات الوصفية
	if metadata == nil {
		metadata = &AgentMetadata{
			AgentID:       agentID,
			Name:          info.Name,
			Type:          info.Type,
			Provider:      info.Provider,
			Model:         info.Model,
			Version:       info.Version,
			Endpoint:      info.Endpoint,
			AuthMethod:    info.AuthMethod,
			MaxTokens:     info.MaxTokens,
			ContextWindow: info.ContextWindow,
			RegisteredAt:  time.Now(),
			LastSeen:      time.Now(),
			Tags:          []string{},
			Config:        make(map[string]interface{}),
			// معلومات التتبع المتعدد من AgentInfo
			InstanceID:      info.InstanceID,
			HumanClientID:   info.HumanClientID,
			HumanClientName: info.HumanClientName,
			APIKeyID:        info.APIKeyID,
			APIKeyLabel:     info.APIKeyLabel,
		}
	} else {
		metadata.AgentID = agentID
		metadata.RegisteredAt = time.Now()
		metadata.LastSeen = time.Now()
		// تحديث معلومات التتبع المتعدد إذا لم تكن موجودة
		if metadata.InstanceID == "" {
			metadata.InstanceID = info.InstanceID
		}
		if metadata.HumanClientID == "" {
			metadata.HumanClientID = info.HumanClientID
		}
		if metadata.HumanClientName == "" {
			metadata.HumanClientName = info.HumanClientName
		}
		if metadata.APIKeyID == "" {
			metadata.APIKeyID = info.APIKeyID
		}
		if metadata.APIKeyLabel == "" {
			metadata.APIKeyLabel = info.APIKeyLabel
		}
	}

	ar.metadata[agentID] = metadata

	// تهيئة الإحصائيات
	ar.stats[agentID] = &AgentStats{
		AgentID:         agentID,
		TotalTasks:      0,
		CompletedTasks:  0,
		FailedTasks:     0,
		TotalTokens:     0,
		TotalDuration:   0,
		AvgResponseTime: 0,
		SuccessRate:     1.0,
		LastUsed:        time.Now(),
	}

	ar.logger.Info("Agent registered",
		zap.String("agent_id", agentID),
		zap.String("name", metadata.Name),
		zap.String("type", string(metadata.Type)),
	)

	return nil
}

// Unregister يلغي تسجيل وكيل
func (ar *AgentRegistry) Unregister(agentID string) error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	agent, exists := ar.agents[agentID]
	if !exists {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	// إغلاق الوكيل
	if err := agent.Close(); err != nil {
		ar.logger.Error("Failed to close agent",
			zap.String("agent_id", agentID),
			zap.Error(err),
		)
	}

	// إزالة من السجل
	delete(ar.agents, agentID)
	delete(ar.metadata, agentID)
	delete(ar.stats, agentID)

	ar.logger.Info("Agent unregistered",
		zap.String("agent_id", agentID),
	)

	return nil
}

// Get يحصل على وكيل
func (ar *AgentRegistry) Get(agentID string) (UnifiedAgent, error) {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	agent, exists := ar.agents[agentID]
	if !exists {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	return agent, nil
}

// GetMetadata يحصل على البيانات الوصفية لوكيل
func (ar *AgentRegistry) GetMetadata(agentID string) (*AgentMetadata, error) {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	metadata, exists := ar.metadata[agentID]
	if !exists {
		return nil, fmt.Errorf("agent metadata not found: %s", agentID)
	}

	// إنشاء نسخة لتجنب التعديل الخارجي
	metadataCopy := *metadata
	return &metadataCopy, nil
}

// GetStats يحصل على إحصائيات وكيل
func (ar *AgentRegistry) GetStats(agentID string) (*AgentStats, error) {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	stats, exists := ar.stats[agentID]
	if !exists {
		return nil, fmt.Errorf("agent stats not found: %s", agentID)
	}

	// إنشاء نسخة لتجنب التعديل الخارجي
	statsCopy := *stats
	return &statsCopy, nil
}

// ListAll يسرد جميع الوكلاء
func (ar *AgentRegistry) ListAll() []UnifiedAgent {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	agents := make([]UnifiedAgent, 0, len(ar.agents))
	for _, agent := range ar.agents {
		agents = append(agents, agent)
	}

	return agents
}

// ListByType يسرد الوكلاء حسب النوع
func (ar *AgentRegistry) ListByType(agentType AgentType) []UnifiedAgent {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	agents := make([]UnifiedAgent, 0)
	for _, agent := range ar.agents {
		if agent.GetInfo().Type == agentType {
			agents = append(agents, agent)
		}
	}

	return agents
}

// ListByCapability يسرد الوكلاء حسب القدرة
func (ar *AgentRegistry) ListByCapability(capability AgentCapability) []UnifiedAgent {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	agents := make([]UnifiedAgent, 0)
	for _, agent := range ar.agents {
		capabilities := agent.GetCapabilities()
		for _, cap := range capabilities {
			if cap == capability {
				agents = append(agents, agent)
				break
			}
		}
	}

	return agents
}

// ListAvailable يسرد الوكلاء المتاحين
func (ar *AgentRegistry) ListAvailable() []UnifiedAgent {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	agents := make([]UnifiedAgent, 0)
	for _, agent := range ar.agents {
		if agent.IsAvailable() {
			agents = append(agents, agent)
		}
	}

	return agents
}

// UpdateStats يحدث إحصائيات وكيل
func (ar *AgentRegistry) UpdateStats(agentID string, taskCompleted bool, tokensUsed int, duration time.Duration) error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	stats, exists := ar.stats[agentID]
	if !exists {
		return fmt.Errorf("agent stats not found: %s", agentID)
	}

	stats.TotalTasks++
	stats.TotalTokens += tokensUsed
	stats.TotalDuration += duration
	stats.LastUsed = time.Now()

	if taskCompleted {
		stats.CompletedTasks++
	} else {
		stats.FailedTasks++
	}

	// تحديث معدل النجاح
	if stats.TotalTasks > 0 {
		stats.SuccessRate = float64(stats.CompletedTasks) / float64(stats.TotalTasks)
	}

	// تحديث متوسط وقت الاستجابة
	if stats.TotalTasks > 0 {
		stats.AvgResponseTime = stats.TotalDuration / time.Duration(stats.TotalTasks)
	}

	// تحديث LastSeen في البيانات الوصفية
	if metadata, exists := ar.metadata[agentID]; exists {
		metadata.LastSeen = time.Now()
	}

	return nil
}

// UpdateMetadata يحدث البيانات الوصفية لوكيل
func (ar *AgentRegistry) UpdateMetadata(agentID string, metadata *AgentMetadata) error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	_, exists := ar.agents[agentID]
	if !exists {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	metadata.AgentID = agentID
	ar.metadata[agentID] = metadata

	ar.logger.Info("Agent metadata updated",
		zap.String("agent_id", agentID),
	)

	return nil
}

// GetCount يحصل على عدد الوكلاء
func (ar *AgentRegistry) GetCount() int {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	return len(ar.agents)
}

// GetAvailableCount يحصل على عدد الوكلاء المتاحين
func (ar *AgentRegistry) GetAvailableCount() int {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	count := 0
	for _, agent := range ar.agents {
		if agent.IsAvailable() {
			count++
		}
	}

	return count
}

// GetByProvider يسرد الوكلاء حسب المزود
func (ar *AgentRegistry) GetByProvider(provider string) []UnifiedAgent {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	agents := make([]UnifiedAgent, 0)
	for _, agent := range ar.agents {
		if agent.GetInfo().Provider == provider {
			agents = append(agents, agent)
		}
	}

	return agents
}

// GetByModel يسرد الوكلاء حسب النموذج
func (ar *AgentRegistry) GetByModel(model string) []UnifiedAgent {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	agents := make([]UnifiedAgent, 0)
	for _, agent := range ar.agents {
		if agent.GetInfo().Model == model {
			agents = append(agents, agent)
		}
	}

	return agents
}

// FindBestAgent يجد أفضل وكيل لمهمة معينة
func (ar *AgentRegistry) FindBestAgent(requiredCapabilities []AgentCapability) (UnifiedAgent, error) {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	var bestAgent UnifiedAgent
	bestScore := 0.0

	for _, agent := range ar.agents {
		if !agent.IsAvailable() {
			continue
		}

		// حساب النتيجة بناءً على القدرات والإحصائيات
		score := ar.calculateAgentScore(agent, requiredCapabilities)

		if score > bestScore {
			bestScore = score
			bestAgent = agent
		}
	}

	if bestAgent == nil {
		return nil, fmt.Errorf("no suitable agent found")
	}

	return bestAgent, nil
}

// calculateAgentScore يحسب نتيجة الوكيل
func (ar *AgentRegistry) calculateAgentScore(agent UnifiedAgent, requiredCapabilities []AgentCapability) float64 {
	score := 0.0

	// التحقق من القدرات
	capabilities := agent.GetCapabilities()
	capabilityMatch := 0
	for _, required := range requiredCapabilities {
		for _, cap := range capabilities {
			if cap == required {
				capabilityMatch++
				break
			}
		}
	}

	if len(requiredCapabilities) > 0 {
		score += float64(capabilityMatch) / float64(len(requiredCapabilities)) * 0.5
	} else {
		score += 0.5 // إذا لم تكن هناك متطلبات، نعطي نتيجة متوسطة
	}

	// التحقق من الإحصائيات
	info := agent.GetInfo()
	stats, exists := ar.stats[info.ID]
	if exists {
		// معدل النجاح
		score += stats.SuccessRate * 0.3

		// معدل الاستجابة (أقل وقت استجابة = نتيجة أعلى)
		if stats.AvgResponseTime > 0 {
			// تحويل إلى ثواني
			responseTimeSeconds := stats.AvgResponseTime.Seconds()
			// نتيجة أعلى لوقت استجابة أقل (حد أقصى 10 ثواني)
			responseScore := max(0, 1.0-responseTimeSeconds/10.0)
			score += responseScore * 0.2
		}
	}

	return score
}

// Save يحفظ حالة السجل
func (ar *AgentRegistry) Save() ([]byte, error) {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	data := struct {
		Metadata map[string]*AgentMetadata `json:"metadata"`
		Stats    map[string]*AgentStats    `json:"stats"`
	}{
		Metadata: ar.metadata,
		Stats:    ar.stats,
	}

	return json.Marshal(data)
}

// Load يحمل حالة السجل
func (ar *AgentRegistry) Load(data []byte) error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	var loaded struct {
		Metadata map[string]*AgentMetadata `json:"metadata"`
		Stats    map[string]*AgentStats    `json:"stats"`
	}

	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}

	ar.metadata = loaded.Metadata
	ar.stats = loaded.Stats

	return nil
}

// CleanupInactive ينظف الوكلاء غير النشطين
func (ar *AgentRegistry) CleanupInactive(inactiveThreshold time.Duration) []string {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	cutoff := time.Now().Add(-inactiveThreshold)
	removed := make([]string, 0)

	for agentID, metadata := range ar.metadata {
		if metadata.LastSeen.Before(cutoff) {
			// إغلاق الوكيل
			if agent, exists := ar.agents[agentID]; exists {
				if err := agent.Close(); err != nil {
					ar.logger.Error("Failed to close inactive agent",
						zap.String("agent_id", agentID),
						zap.Error(err),
					)
				}
			}

			// إزالة من السجل
			delete(ar.agents, agentID)
			delete(ar.metadata, agentID)
			delete(ar.stats, agentID)

			removed = append(removed, agentID)

			ar.logger.Info("Removed inactive agent",
				zap.String("agent_id", agentID),
				zap.Duration("inactive_duration", time.Since(metadata.LastSeen)),
			)
		}
	}

	return removed
}

// HealthReport تقرير الصحة
type HealthReport struct {
	Timestamp         time.Time                     `json:"timestamp"`
	TotalAgents       int                           `json:"total_agents"`
	AvailableAgents   int                           `json:"available_agents"`
	UnavailableAgents int                           `json:"unavailable_agents"`
	AgentDetails      map[string]*AgentHealthDetail `json:"agent_details"`
}

// AgentHealthDetail تفاصيل صحة وكيل
type AgentHealthDetail struct {
	Status       *AgentStatus      `json:"status"`
	Capabilities []AgentCapability `json:"capabilities"`
}

// HealthCheck يفحص صحة الوكلاء
func (ar *AgentRegistry) HealthCheck() *HealthReport {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	report := &HealthReport{
		Timestamp:         time.Now(),
		TotalAgents:       len(ar.agents),
		AvailableAgents:   0,
		UnavailableAgents: 0,
		AgentDetails:      make(map[string]*AgentHealthDetail),
	}

	for id, agent := range ar.agents {
		status := agent.GetStatus()
		detail := &AgentHealthDetail{
			Status:       status,
			Capabilities: agent.GetCapabilities(),
		}

		if status.IsAvailable {
			report.AvailableAgents++
		} else {
			report.UnavailableAgents++
		}

		report.AgentDetails[id] = detail
	}

	return report
}

// ============================================================
// Human Client Status - حالة العميل البشري
// ============================================================

// RegisterHumanClient يسجل عميل بشري جديد
func (ar *AgentRegistry) RegisterHumanClient(userID, name string, allowOnline bool) error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	ar.humanClient = &HumanClientStatus{
		UserID:      userID,
		Name:        name,
		Status:      "online",
		LastSeen:    time.Now(),
		Preferences: make(map[string]interface{}),
		AllowOnline: allowOnline,
	}

	ar.logger.Info("تم تسجيل عميل بشري جديد",
		zap.String("user_id", userID),
		zap.String("name", name),
		zap.Bool("allow_online", allowOnline),
	)

	return nil
}

// UpdateHumanClientStatus يحدث حالة العميل البشري
func (ar *AgentRegistry) UpdateHumanClientStatus(status string) error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	if ar.humanClient == nil {
		return fmt.Errorf("العميل البشري غير مسجل")
	}

	// إذا كان العميل لا يريد أن يكون أونلاين، نحترم اختياره
	if !ar.humanClient.AllowOnline && status == "online" {
		ar.logger.Warn("العميل البشري لا يريد أن يكون أونلاين",
			zap.String("user_id", ar.humanClient.UserID),
		)
		return fmt.Errorf("العميل البشري لا يريد أن يكون أونلاين")
	}

	ar.humanClient.Status = status
	ar.humanClient.LastSeen = time.Now()

	ar.logger.Info("تم تحديث حالة العميل البشري",
		zap.String("user_id", ar.humanClient.UserID),
		zap.String("status", status),
	)

	return nil
}

// GetHumanClientStatus يحصل على حالة العميل البشري
func (ar *AgentRegistry) GetHumanClientStatus() (*HumanClientStatus, error) {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	if ar.humanClient == nil {
		return nil, fmt.Errorf("العميل البشري غير مسجل")
	}

	// إنشاء نسخة لتجنب التعديل الخارجي
	statusCopy := *ar.humanClient
	return &statusCopy, nil
}

// SetHumanClientOnlinePreference يضبط تفضيل العميل البشري للأونلاين
func (ar *AgentRegistry) SetHumanClientOnlinePreference(allowOnline bool) error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	if ar.humanClient == nil {
		return fmt.Errorf("العميل البشري غير مسجل")
	}

	ar.humanClient.AllowOnline = allowOnline

	// إذا كان العميل لا يريد أن يكون أونلاين، نغير حالته إلى offline
	if !allowOnline && ar.humanClient.Status == "online" {
		ar.humanClient.Status = "offline"
	}

	ar.logger.Info("تم تحديث تفضيل العميل البشري للأونلاين",
		zap.String("user_id", ar.humanClient.UserID),
		zap.Bool("allow_online", allowOnline),
	)

	return nil
}

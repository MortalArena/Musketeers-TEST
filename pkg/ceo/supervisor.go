package ceo

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
)

// [WHY] CEOSupervisor المشرف على الشبكة - يراقب صحة النظام بأكمله
// [HOW] يسجل نفسه كوكيل admin ويشغل HealthCheck دورياً
// [SAFETY] يستخدم RWMutex لحماية الحالة
type CEOSupervisor struct {
	// المكونات الأساسية
	eventBus      *eventbus.EventBus
	agentRegistry *agent.AgentRegistry

	// حالة المشرف
	did     string       // [WHY] معرف الوكيل (CEO)
	name    string       // [WHY] اسم الوكيل
	running bool         // [WHY] هل المشرف يعمل
	mu      sync.RWMutex // [SAFETY] لحماية الحالة

	// Lifecycle - دورة الحياة
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Logger
	logger *log.Logger
}

// [WHY] NewCEOSupervisor ينشئ مشرف CEO جديد
// [HOW] يهيئ المكونات ويسجل نفسه كوكيل admin
// [SAFETY] يتحقق من أن eventBus و agentRegistry ليسا nil
func NewCEOSupervisor(eventBus *eventbus.EventBus, agentRegistry *agent.AgentRegistry, logger *log.Logger) *CEOSupervisor {
	if eventBus == nil {
		panic("eventBus cannot be nil") // [SAFETY] منع nil pointer
	}
	if agentRegistry == nil {
		panic("agentRegistry cannot be nil") // [SAFETY] منع nil pointer
	}

	ctx, cancel := context.WithCancel(context.Background())

	supervisor := &CEOSupervisor{
		eventBus:      eventBus,
		agentRegistry: agentRegistry,
		did:           "ceo_supervisor",
		name:          "CEO Supervisor",
		running:       true,
		ctx:           ctx,
		cancel:        cancel,
		logger:        logger,
	}

	// [HOW] تسجيل المشرف كوكيل admin في AgentRegistry
	err := supervisor.registerAsAgent()
	if err != nil {
		logger.Printf("فشل تسجيل المشرف كوكيل: %v", err)
	}

	return supervisor
}

// [WHY] registerAsAgent يسجل المشرف كوكيل admin
// [HOW] ينشئ SimpleAgent ويسجله في AgentRegistry
// [SAFETY] يستخدم recover() لمنع تعطل النظام
func (s *CEOSupervisor) registerAsAgent() error {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Printf("panic في تسجيل المشرف: %v", r)
		}
	}()

	// [HOW] إنشاء SimpleAgent للمشرف
	ceoAgent := NewCEOSupervisorAgent(s.did, s.name)

	// [HOW] إنشاء البيانات الوصفية
	metadata := &agent.AgentMetadata{
		AgentID:       s.did,
		Name:          s.name,
		Type:          agent.AgentTypeCustom,
		Provider:      "internal",
		Model:         "supervisor",
		Version:       "1.0.0",
		Endpoint:      "internal",
		AuthMethod:    "none",
		MaxTokens:     0,
		ContextWindow: 0,
		RegisteredAt:  time.Now(),
		LastSeen:      time.Now(),
		Tags:          []string{"admin", "supervisor"},
		Config:        make(map[string]interface{}),
	}

	// [HOW] تسجيل الوكيل
	err := s.agentRegistry.Register(ceoAgent, metadata)
	if err != nil {
		return err
	}

	s.logger.Printf("تم تسجيل المشرف كوكيل: %s", s.did)
	return nil
}

// [WHY] Start يبدأ المشرف
// [HOW] يشترك في EventBus ويبدأ HealthCheck دوري
// [SAFETY] يستخدم WaitGroup لانتظار goroutines
func (s *CEOSupervisor) Start() error {
	s.logger.Println("بدء CEOSupervisor")

	// [HOW] الاشتراك في كل الأحداث للمراقبة
	s.eventBus.Subscribe("*", s.handleAllEvents)

	// [HOW] بدء HealthCheck دوري
	s.wg.Add(1)
	go s.healthCheckLoop()

	s.logger.Println("تم بدء CEOSupervisor بنجاح")
	return nil
}

// [WHY] Stop يوقف المشرف
// [HOW] يلغي الاشتراك ويوقف goroutines
// [SAFETY] يستخدم WaitGroup لانتظار goroutines
func (s *CEOSupervisor) Stop() error {
	s.logger.Println("إيقاف CEOSupervisor")

	// [HOW] إيقاف التشغيل
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()

	// [HOW] إلغاء context
	s.cancel()

	// [HOW] انتظار goroutines
	s.wg.Wait()

	// [HOW] إلغاء التسجيل من AgentRegistry
	err := s.agentRegistry.Unregister(s.did)
	if err != nil {
		s.logger.Printf("فشل إلغاء تسجيل المشرف: %v", err)
	}

	s.logger.Println("تم إيقاف CEOSupervisor بنجاح")
	return nil
}

// [WHY] handleAllEvents يعالج كل الأحداث
// [HOW] يسجل الأحداث المهمة للمراقبة
// [SAFETY] يستخدم recover() لمنع تعطل النظام
func (s *CEOSupervisor) handleAllEvents(event eventbus.Event) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Printf("panic في معالج الأحداث: %v", r)
		}
	}()

	// [HOW] تسجيل الأحداث المهمة
	switch event.Type {
	case "agent.error", "bridge.error", "system.error":
		s.logger.Printf("حدث خطأ في النظام: %s", event.Type)
	case "agent.connected", "agent.disconnected":
		s.logger.Printf("تغير حالة وكيل: %s", event.Type)
	}
}

// [WHY] healthCheckLoop يعمل HealthCheck دوري كل 30 ثانية
// [HOW] يستدعي agentRegistry.HealthCheck() وينشر تنبيهات
// [SAFETY] يستخدم Ticker للتنفيذ الدوري
func (s *CEOSupervisor) healthCheckLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// [HOW] فحص صحة النظام
			s.checkSystemHealth()
		}
	}
}

// [WHY] checkSystemHealth يفحص صحة النظام
// [HOW] يستدعي HealthCheck وينشر تنبيهات إذا لزم الأمر
// [SAFETY] يستخدم recover() لمنع تعطل النظام
func (s *CEOSupervisor) checkSystemHealth() {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Printf("panic في فحص الصحة: %v", r)
		}
	}()

	// [HOW] الحصول على تقرير الصحة
	healthReport := s.agentRegistry.HealthCheck()

	// [HOW] حساب نسبة الصحة
	totalAgents := healthReport.TotalAgents
	healthScore := 0
	if totalAgents > 0 {
		healthScore = (healthReport.AvailableAgents * 100) / totalAgents
	}

	// [HOW] فحص النتيجة
	if healthScore < 50 {
		// [SAFETY] النتيجة منخفضة، نشر تنبيه
		s.publishHealthAlert("low_score", healthReport, healthScore)
	}

	// [HOW] فحص عدد الوكلاء غير المتاحين
	if healthReport.UnavailableAgents > 0 {
		// [SAFETY] هناك وكلاء غير متاحين، نشر تنبيه
		s.publishHealthAlert("agent_unavailable", healthReport, healthScore)
	}

	// [HOW] تسجيل التقرير
	s.logger.Printf("تقرير الصحة: النتيجة=%d%%, المتاح=%d, غير متاح=%d, الإجمالي=%d",
		healthScore,
		healthReport.AvailableAgents,
		healthReport.UnavailableAgents,
		healthReport.TotalAgents,
	)
}

// [WHY] publishHealthAlert ينشر تنبيه صحة
// [HOW] ينشر حدث ceo.health_alert مع التفاصيل
// [SAFETY] لا يحظر لأنه يستخدم EventBus
func (s *CEOSupervisor) publishHealthAlert(alertType string, report *agent.HealthReport, healthScore int) {
	alert := map[string]interface{}{
		"type":        alertType,
		"score":       healthScore,
		"available":   report.AvailableAgents,
		"unavailable": report.UnavailableAgents,
		"total":       report.TotalAgents,
		"timestamp":   time.Now(),
	}

	s.eventBus.Publish(eventbus.Event{
		Type:      "ceo.health_alert",
		Payload:   alert,
		Source:    "ceo_supervisor",
		Timestamp: time.Now(),
	})

	s.logger.Printf("تم نشر تنبيه صحة: %s", alertType)
}

// [WHY] GetDID يحصل على معرف المشرف
// [HOW] يعيد معرف الوكيل
// [SAFETY] لا يحتاج قفل لأنه ثابت
func (s *CEOSupervisor) GetDID() string {
	return s.did
}

// [WHY] GetName يحصل على اسم المشرف
// [HOW] يعيد اسم الوكيل
// [SAFETY] لا يحتاج قفل لأنه ثابت
func (s *CEOSupervisor) GetName() string {
	return s.name
}

// [WHY] IsRunning يحصل على حالة التشغيل
// [HOW] يعيد ما إذا كان المشرف يعمل
// [SAFETY] يستخدم RLock للقراءة فقط
func (s *CEOSupervisor) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// ============================================================
// CEOSupervisorAgent - وكيل بسيط للمشرف
// ============================================================

// ceoSupervisorAgent وكيل بسيط للمشرف
type ceoSupervisorAgent struct {
	did  string
	name string
}

// NewCEOSupervisorAgent ينشئ وكيل مشرف جديد
func NewCEOSupervisorAgent(did, name string) agent.UnifiedAgent {
	return &ceoSupervisorAgent{
		did:  did,
		name: name,
	}
}

// GetInfo يحصل على معلومات الوكيل
func (a *ceoSupervisorAgent) GetInfo() *agent.AgentInfo {
	return &agent.AgentInfo{
		ID:       a.did,
		Name:     a.name,
		Type:     agent.AgentTypeCustom,
		Provider: "internal",
		Model:    "supervisor",
	}
}

// SendMessage يرسل رسالة للوكيل
func (a *ceoSupervisorAgent) SendMessage(ctx context.Context, prompt string) (*agent.AgentResponse, error) {
	return &agent.AgentResponse{
		Content:  "CEO Supervisor: System is healthy",
		Tokens:   0,
		Duration: 0,
	}, nil
}

// ExecuteTask ينفذ مهمة
func (a *ceoSupervisorAgent) ExecuteTask(ctx context.Context, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	return &agent.TaskExecutionResult{
		Success: true,
		Output:  "CEO Supervisor: Task acknowledged",
	}, nil
}

// GetCapabilities يحصل على قدرات الوكيل
func (a *ceoSupervisorAgent) GetCapabilities() []agent.AgentCapability {
	return []agent.AgentCapability{
		agent.CapabilityAnalysis,
		agent.CapabilityDocumentation,
	}
}

// GetStatus يحصل على حالة الوكيل
func (a *ceoSupervisorAgent) GetStatus() *agent.AgentStatus {
	return &agent.AgentStatus{
		IsAvailable:  true,
		Load:         0,
		LastSeen:     time.Now(),
		ResponseTime: 0,
		SuccessRate:  1.0,
		TotalTasks:   0,
	}
}

// IsAvailable يتحقق من توفر الوكيل
func (a *ceoSupervisorAgent) IsAvailable() bool {
	return true
}

// Close يغلق الوكيل
func (a *ceoSupervisorAgent) Close() error {
	return nil
}

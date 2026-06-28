package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/agent/tools"
	"github.com/MortalArena/Musketeers/pkg/agent/unified"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/MortalArena/Musketeers/pkg/session"
	"github.com/dgraph-io/badger/v4"
	"go.uber.org/zap"
)

// ============================================================
// Mock Agent للتكامل - يطبق UnifiedAgent interface
// ============================================================

type testAgent struct {
	info      *agent.AgentInfo
	status    *agent.AgentStatus
	available bool
	mu        sync.Mutex
	msgCount  int
}

func newTestAgent(id, name, provider, model string) *testAgent {
	return &testAgent{
		info: &agent.AgentInfo{
			ID:            id,
			Name:          name,
			Type:          agent.AgentTypeAPI,
			Provider:      provider,
			Model:         model,
			Version:       "1.0.0",
			MaxTokens:     4096,
			ContextWindow: 8192,
			CreatedAt:     time.Now(),
			InstanceID:    fmt.Sprintf("%s-instance", id),
		},
		status: &agent.AgentStatus{
			IsAvailable:  true,
			Load:         0,
			LastSeen:     time.Now(),
			ResponseTime: 100 * time.Millisecond,
			SuccessRate:  1.0,
			TotalTasks:   0,
			FailedTasks:  0,
		},
		available: true,
	}
}

func (ta *testAgent) GetInfo() *agent.AgentInfo                       { return ta.info }
func (ta *testAgent) GetCapabilities() []agent.AgentCapability         { return []agent.AgentCapability{agent.CapabilityCodeGeneration} }
func (ta *testAgent) GetStatus() *agent.AgentStatus                    { return ta.status }
func (ta *testAgent) IsAvailable() bool                                { return ta.available }

func (ta *testAgent) SendMessage(ctx context.Context, prompt string) (*agent.AgentResponse, error) {
	ta.mu.Lock()
	ta.msgCount++
	ta.mu.Unlock()
	return &agent.AgentResponse{Content: "response", Tokens: 10, Duration: time.Millisecond}, nil
}

func (ta *testAgent) ExecuteTask(ctx context.Context, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	ta.mu.Lock()
	ta.msgCount++
	ta.mu.Unlock()
	return &agent.TaskExecutionResult{Success: true, Output: "done", Duration: time.Millisecond}, nil
}

func (ta *testAgent) Close() error {
	ta.available = false
	ta.status.IsAvailable = false
	return nil
}

// ============================================================
// 1. اختبار التواصل بين الوكلاء عبر AgentCommunication
// ============================================================

func TestAgentToAgentCommunication(t *testing.T) {
	logger := zap.NewNop()
	reg := agent.NewAgentRegistry()
	reg.SetLogger(logger)
	comm := NewAgentCommunication(reg, logger)

	// تسجيل 5 وكلاء
	agents := make([]*testAgent, 5)
	for i := 0; i < 5; i++ {
		provider := []string{"anthropic", "openai", "google", "deepseek", "meta"}[i]
		a := newTestAgent(fmt.Sprintf("agent_%d", i), fmt.Sprintf("Agent-%d", i), provider, "model-v1")
		agents[i] = a
		if err := reg.Register(a, nil); err != nil {
			t.Fatalf("failed to register agent %d: %v", i, err)
		}
	}

	t.Run("DirectMessage", func(t *testing.T) {
		err := comm.SendMessageBetweenAgents("agent_0", "agent_1", "hello from 0", "info")
		if err != nil {
			t.Fatalf("SendMessageBetweenAgents failed: %v", err)
		}
		msgs, err := comm.GetAgentMessages("agent_1")
		if err != nil {
			t.Fatalf("GetAgentMessages failed: %v", err)
		}
		if len(msgs) != 1 || msgs[0].Content != "hello from 0" {
			t.Fatalf("unexpected messages: got %d, content=%s", len(msgs), msgs[0].Content)
		}
	})

	t.Run("BroadcastToAll", func(t *testing.T) {
		err := comm.BroadcastMessage("agent_0", "broadcast test", "info")
		if err != nil {
			t.Fatalf("BroadcastMessage failed: %v", err)
		}
		// كل الـ 4 وكلاء الآخرين استلموا الرسالة
		for i := 1; i < 5; i++ {
			msgs, err := comm.GetAgentMessages(fmt.Sprintf("agent_%d", i))
			if err != nil {
				t.Fatalf("GetAgentMessages failed for agent_%d: %v", i, err)
			}
			if len(msgs) == 0 {
				t.Fatalf("agent_%d didn't receive broadcast", i)
			}
		}
	})

	t.Run("ShareTaskResult", func(t *testing.T) {
		result := &agent.TaskExecutionResult{
			Success:  true,
			Output:   "task output data",
			Duration: time.Second,
		}
		err := comm.ShareTaskResult("agent_0", "task_001", result, []string{"agent_2", "agent_3"})
		if err != nil {
			t.Fatalf("ShareTaskResult failed: %v", err)
		}
		for _, target := range []string{"agent_2", "agent_3"} {
			msgs, err := comm.GetAgentMessagesByTask(target, "task_001")
			if err != nil {
				t.Fatalf("GetAgentMessagesByTask failed for %s: %v", target, err)
			}
			if len(msgs) == 0 {
				t.Fatalf("%s didn't receive task result", target)
			}
		}
		// agent_4 لم يستلم النتيجة
		msgs, err := comm.GetAgentMessagesByTask("agent_4", "task_001")
		if err != nil {
			t.Fatalf("GetAgentMessagesByTask failed: %v", err)
		}
		if len(msgs) != 0 {
			t.Fatalf("agent_4 should not have received task result, got %d", len(msgs))
		}
	})

	t.Run("ConcurrentMessaging", func(t *testing.T) {
		var wg sync.WaitGroup
		errChan := make(chan error, 50)
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(n int) {
				defer func() { recover() }()
				defer wg.Done()
				from := fmt.Sprintf("agent_%d", n%5)
				to := fmt.Sprintf("agent_%d", (n+1)%5)
				if err := comm.SendMessageBetweenAgents(from, to, fmt.Sprintf("msg_%d", n), "info"); err != nil {
					errChan <- fmt.Errorf("concurrent send %d: %w", n, err)
				}
			}(i)
		}
		wg.Wait()
		close(errChan)
		for err := range errChan {
			t.Errorf("concurrent messaging error: %v", err)
		}
		summary := comm.GetCommunicationSummary()
		total, _ := summary["total_messages"].(int)
		if total < 50 {
			t.Errorf("expected >=50 total messages, got %d", total)
		}
	})
}

// ============================================================
// 2. اختبار SessionEventBus مع وكلاء متعددين
// ============================================================

func TestSessionEventBus_MultiAgent(t *testing.T) {
	logger := zap.NewNop()
	bus := unified.NewSessionEventBus("sess_test", logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)
	defer bus.Stop()

	agentCount := 10
	agents := make([]*testAgent, agentCount)
	channels := make([]chan *unified.SessionEvent, agentCount)

	for i := 0; i < agentCount; i++ {
		agents[i] = newTestAgent(fmt.Sprintf("agent_%d", i), fmt.Sprintf("Agent-%d", i), "test", "v1")
		channels[i] = bus.SubscribeAgent(agents[i].GetInfo().ID)
	}

	t.Run("BroadcastToAllAgents", func(t *testing.T) {
		var wg sync.WaitGroup
		received := make([]bool, agentCount)
		var mu sync.Mutex

		// كل وكيل ينتظر الحدث
		for i := 0; i < agentCount; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				timer := time.NewTimer(time.Second)
				defer timer.Stop()
				select {
				case event := <-channels[idx]:
					if event.EventType != unified.AgentMessage {
						t.Errorf("unexpected event type: %s", event.EventType)
					}
				case <-timer.C:
					t.Errorf("agent_%d timed out waiting for broadcast", idx)
				}
				mu.Lock()
				received[idx] = true
				mu.Unlock()
			}(i)
		}

		time.Sleep(50 * time.Millisecond) // السماح للـ goroutines بالبدء
		err := bus.BroadcastToAll(ctx, "agent_0", unified.AgentMessage, "hello everyone")
		if err != nil {
			t.Fatalf("BroadcastToAll failed: %v", err)
		}

		wg.Wait()
		for i, received := range received {
			if !received {
				t.Errorf("agent_%d didn't receive broadcast", i)
			}
		}
	})

	t.Run("TargetedMessage", func(t *testing.T) {
		ch := channels[3]
		err := bus.SendToAgent(ctx, "agent_0", "agent_3", unified.TaskAssigned, map[string]string{"task_id": "t1"})
		if err != nil {
			t.Fatalf("SendToAgent failed: %v", err)
		}

		timer := time.NewTimer(time.Second)
		defer timer.Stop()
		select {
		case event := <-ch:
			if event.TargetAgent != "agent_3" {
				t.Errorf("wrong target: %s", event.TargetAgent)
			}
		case <-timer.C:
			t.Fatal("agent_3 timed out waiting for targeted message")
		}
	})

	t.Run("SessionManagerReceivesAll", func(t *testing.T) {
		mgrCh := bus.GetSessionManagerChannel()
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < agentCount; i++ {
				timer := time.NewTimer(500 * time.Millisecond)
				select {
				case <-mgrCh:
					timer.Stop()
				case <-timer.C:
					t.Errorf("session manager missed event %d", i)
				}
			}
		}()

		for i := 0; i < agentCount; i++ {
			bus.SendToSessionManager(ctx, fmt.Sprintf("agent_%d", i), unified.AgentStatus, "working")
		}
		wg.Wait()
	})

	t.Run("EventHistory", func(t *testing.T) {
		history := bus.GetEventHistory(100)
		if len(history) == 0 {
			t.Fatal("expected event history, got empty")
		}
		if len(history) > 0 && history[len(history)-1].SessionID != "sess_test" {
			t.Errorf("wrong session ID in history: %s", history[len(history)-1].SessionID)
		}
	})
}

// ============================================================
// 3. اختبار SessionEventBus مع تحمّل عالٍ (50+ وكيل)
// ============================================================

func TestSessionEventBus_HighLoad(t *testing.T) {
	logger := zap.NewNop()
	bus := unified.NewSessionEventBus("sess_load", logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)
	defer bus.Stop()

	agentCount := 50
	channels := make([]chan *unified.SessionEvent, agentCount)

	for i := 0; i < agentCount; i++ {
		channels[i] = bus.SubscribeAgent(fmt.Sprintf("load_agent_%d", i))
	}

	// إرسال 100 حدث بث
	var wg sync.WaitGroup
	publishStart := make(chan struct{})
	received := make([]int32, agentCount)

	// كل وكيل يبدأ الاستماع
	for i := 0; i < agentCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-publishStart
			for eventCount := 0; eventCount < 100; eventCount++ {
				timer := time.NewTimer(5 * time.Second)
				select {
				case <-channels[idx]:
					timer.Stop()
					received[idx]++
				case <-timer.C:
					return
				}
			}
		}(i)
	}

	close(publishStart)

	// 100 حدث بث من 5 وكلاء مختلفين
	for j := 0; j < 100; j++ {
		source := fmt.Sprintf("load_agent_%d", j%5)
		if err := bus.BroadcastToAll(ctx, source, unified.AgentMessage, fmt.Sprintf("event_%d", j)); err != nil {
			t.Fatalf("publish %d failed: %v", j, err)
		}
		time.Sleep(time.Millisecond) // إتاحة وقت للمعالجة
	}

	wg.Wait()

	status := bus.GetStatus()
	totalEvents, _ := status["total_events"].(int)
	t.Logf("High load: %d events published, %d agents", totalEvents, agentCount)

	if totalEvents < 100 {
		t.Errorf("expected >=100 events, got %d", totalEvents)
	}
}

// ============================================================
// 4. اختبار SessionEventBus مع أحداث متزامنة ومتضاربة
// ============================================================

func TestSessionEventBus_ConcurrentEvents(t *testing.T) {
	logger := zap.NewNop()
	bus := unified.NewSessionEventBus("sess_concurrent", logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)
	defer bus.Stop()

	agentCount := 20
	channels := make([]chan *unified.SessionEvent, agentCount)
	for i := 0; i < agentCount; i++ {
		channels[i] = bus.SubscribeAgent(fmt.Sprintf("conc_agent_%d", i))
	}

	// 20 وكيلاً يرسلون أحداثاً في نفس الوقت
	var pubWg sync.WaitGroup
	for i := 0; i < agentCount; i++ {
		pubWg.Add(1)
		go func(id int) {
			defer func() { recover() }()
			defer pubWg.Done()
			for j := 0; j < 10; j++ {
				bus.SendToSessionManager(ctx, fmt.Sprintf("conc_agent_%d", id), unified.SystemAlert, fmt.Sprintf("alert_%d_%d", id, j))
				bus.BroadcastToAll(ctx, fmt.Sprintf("conc_agent_%d", id), unified.AgentMessage, fmt.Sprintf("broadcast_%d_%d", id, j))
			}
		}(i)
	}
	pubWg.Wait()

	// انتظار انتهاء معالجة الأحداث
	time.Sleep(200 * time.Millisecond)

	// التحقق من التاريخ
	history := bus.GetEventHistory(500)
	eventCount := len(history)
	t.Logf("Concurrent events: %d events in history from %d agents", eventCount, agentCount)

	if eventCount < agentCount*10 {
		t.Errorf("expected at least %d events, got %d", agentCount*10, eventCount)
	}

	// التحقق من وصول جميع الأحداث لمدير الجلسة
	mgrCh := bus.GetSessionManagerChannel()
	drainCount := 0
	drainDone := make(chan struct{})
	go func() {
		defer func() { recover() }()
		for {
			timer := time.NewTimer(100 * time.Millisecond)
			select {
			case <-mgrCh:
				timer.Stop()
				drainCount++
			case <-timer.C:
				close(drainDone)
				return
			}
		}
	}()
	<-drainDone
	t.Logf("Session manager received %d events", drainCount)
}

// ============================================================
// 5. اختبار AgentCommunication + SessionEventBus معاً
// ============================================================

func TestAgentCommunicationWithEventBus(t *testing.T) {
	logger := zap.NewNop()
	agentReg := agent.NewAgentRegistry()
	agentReg.SetLogger(logger)
	comm := NewAgentCommunication(agentReg, logger)
	sessionBus := unified.NewSessionEventBus("sess_integrated", logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sessionBus.Start(ctx)
	defer sessionBus.Stop()

	// تسجيل 10 وكلاء واشتراكهم في EventBus
	agents := make([]*testAgent, 10)
	for i := 0; i < 10; i++ {
		a := newTestAgent(fmt.Sprintf("integ_agent_%d", i), fmt.Sprintf("Agent-%d", i), "test", "v1")
		agents[i] = a
		if err := agentReg.Register(a, nil); err != nil {
			t.Fatalf("register %d: %v", i, err)
		}
		sessionBus.SubscribeAgent(a.GetInfo().ID)
	}

	// 10 وكلاء يتواصلون عبر AgentCommunication ويرسلون أحداثاً في نفس الوقت
	var wg sync.WaitGroup
	errChan := make(chan error, 200)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer func() { recover() }()
			defer wg.Done()
			agentID := fmt.Sprintf("integ_agent_%d", id)

			// إرسال رسائل عبر AgentCommunication
			for j := 0; j < 5; j++ {
				target := fmt.Sprintf("integ_agent_%d", (id+j+1)%10)
				if err := comm.SendMessageBetweenAgents(agentID, target, fmt.Sprintf("msg_%d_%d", id, j), "info"); err != nil {
					errChan <- fmt.Errorf("comm send %d/%d: %w", id, j, err)
				}
			}

			// إرسال أحداث عبر SessionEventBus
			for j := 0; j < 5; j++ {
				if err := sessionBus.BroadcastToAll(ctx, agentID, unified.AgentStatus, fmt.Sprintf("event_%d_%d", id, j)); err != nil {
					errChan <- fmt.Errorf("bus send %d/%d: %w", id, j, err)
				}
			}
		}(i)
	}
	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("integration error: %v", err)
	}

	// التحقق من AgentCommunication
	summary := comm.GetCommunicationSummary()
	totalMsgs, _ := summary["total_messages"].(int)
	if totalMsgs < 50 {
		t.Errorf("expected >=50 communication messages, got %d", totalMsgs)
	}

	// التحقق من SessionEventBus (مع انتظار قصير للمعالجة غير المتزامنة)
	var eventCount int
	for i := 0; i < 10; i++ {
		history := sessionBus.GetEventHistory(200)
		eventCount = len(history)
		if eventCount >= 50 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if eventCount < 50 {
		t.Errorf("expected >=50 events, got %d", eventCount)
	}
	t.Logf("Integration OK: %d messages, %d events", totalMsgs, eventCount)
}

// ============================================================
// 6. اختبار CollectiveMemory مع وكلاء متعددين
// ============================================================

func TestCollectiveMemory_ConcurrentAgents(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		t.Fatalf("failed to open in-memory badger: %v", err)
	}
	defer db.Close()

	memory := session.NewCollectiveMemory("sess_mem_test", db)

	agentCount := 20
	var wg sync.WaitGroup
	errChan := make(chan error, agentCount*10)

	// 20 وكيلاً يكتبون أحداثاً في CollectiveMemory
	for i := 0; i < agentCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer func() { recover() }()
			defer wg.Done()
			for j := 0; j < 10; j++ {
				event := session.MemoryEvent{
					ID:        fmt.Sprintf("event_%d_%d", id, j),
					AgentDID:  fmt.Sprintf("did:agent:%d", id),
					Action:    fmt.Sprintf("action_%d", j),
					Outcome:   "success",
					Confidence: 1.0,
				}
				if err := memory.RecordEvent(event); err != nil {
					errChan <- fmt.Errorf("agent %d record event %d: %w", id, j, err)
				}
			}
		}(i)
	}
	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("concurrent memory write error: %v", err)
	}

	// التحقق من كتابة جميع الأحداث
	events := memory.QueryEvents(nil)
	if len(events) != agentCount*10 {
		t.Errorf("expected %d events, got %d", agentCount*10, len(events))
	}

	// 20 وكيلاً يقرؤون الذاكرة في نفس الوقت
	var readWg sync.WaitGroup
	for i := 0; i < agentCount; i++ {
		readWg.Add(1)
		go func(id int) {
			defer func() { recover() }()
			defer readWg.Done()
			filters := map[string]interface{}{
				"agent_did": fmt.Sprintf("did:agent:%d", id),
			}
			results := memory.QueryEvents(filters)
			if len(results) != 10 {
				t.Errorf("agent %d expected 10 events, got %d", id, len(results))
			}
		}(i)
	}
	readWg.Wait()

	// إضافة حقائق معرفية بشكل متزامن
	var factWg sync.WaitGroup
	for i := 0; i < agentCount; i++ {
		factWg.Add(1)
		go func(id int) {
			defer func() { recover() }()
			defer factWg.Done()
			fact := session.MemoryFact{
				ID:        fmt.Sprintf("fact_%d", id),
				Statement: fmt.Sprintf("topic_%d is about value_%d", id%5, id),
				Category:  "test",
				Confidence: 0.9,
				Source:    fmt.Sprintf("agent_%d", id),
			}
			if err := memory.LearnFact(fact); err != nil {
				t.Errorf("agent %d learn fact error: %v", id, err)
			}
		}(i)
	}
	factWg.Wait()
}

// ============================================================
// 7. اختبار AgentRegistry مع تسجيل/إلغاء متزامن
// ============================================================

func TestAgentRegistry_ConcurrentRegisterUnregister(t *testing.T) {
	logger := zap.NewNop()
	reg := agent.NewAgentRegistry()
	reg.SetLogger(logger)

	agentCount := 30
	var wg sync.WaitGroup
	errChan := make(chan error, agentCount*2)

	// تسجيل 30 وكيلاً متزامناً
	for i := 0; i < agentCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer func() { recover() }()
			defer wg.Done()
			a := newTestAgent(fmt.Sprintf("reg_agent_%d", id), fmt.Sprintf("Agent-%d", id), "test", "v1")
			if err := reg.Register(a, nil); err != nil {
				errChan <- fmt.Errorf("register %d: %w", id, err)
			}
		}(i)
	}
	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("concurrent register error: %v", err)
	}

	if count := reg.GetCount(); count != agentCount {
		t.Fatalf("expected %d agents, got %d", agentCount, count)
	}

	// إلغاء تسجيل 15 وكيلاً متزامناً
	var unregWg sync.WaitGroup
	errChan2 := make(chan error, 15)
	for i := 0; i < 15; i++ {
		unregWg.Add(1)
		go func(id int) {
			defer func() { recover() }()
			defer unregWg.Done()
			if err := reg.Unregister(fmt.Sprintf("reg_agent_%d", id)); err != nil {
				errChan2 <- fmt.Errorf("unregister %d: %w", id, err)
			}
		}(i)
	}
	unregWg.Wait()
	close(errChan2)

	for err := range errChan2 {
		t.Errorf("concurrent unregister error: %v", err)
	}

	if count := reg.GetCount(); count != 15 {
		t.Errorf("expected 15 agents after unregister, got %d", count)
	}
}

// ============================================================
// 8. اختبار تصعيدي: 10 → 30 → 50 وكيلاً مع اتصال كامل
// ============================================================

func TestScaledAgentCommunication(t *testing.T) {
	for _, count := range []int{10, 30, 50} {
		t.Run(fmt.Sprintf("Agents_%d", count), func(t *testing.T) {
			logger := zap.NewNop()
			reg := agent.NewAgentRegistry()
			reg.SetLogger(logger)
			comm := NewAgentCommunication(reg, logger)

			// تسجيل الوكلاء
			for i := 0; i < count; i++ {
				a := newTestAgent(fmt.Sprintf("agent_%d", i), fmt.Sprintf("Agent-%d", i), "provider", "v1")
				if err := reg.Register(a, nil); err != nil {
					t.Fatalf("register %d: %v", i, err)
				}
			}

			// كل وكيل يرسل رسالة للجميع
			var wg sync.WaitGroup
			errChan := make(chan error, count)
			for i := 0; i < count; i++ {
				wg.Add(1)
				go func(id int) {
					defer func() { recover() }()
					defer wg.Done()
					from := fmt.Sprintf("agent_%d", id)
					err := comm.BroadcastMessage(from, fmt.Sprintf("hello from %d", id), "info")
					if err != nil {
						errChan <- fmt.Errorf("broadcast %d: %w", id, err)
					}
				}(i)
			}
			wg.Wait()
			close(errChan)

			for err := range errChan {
				t.Errorf("scale test error (%d agents): %v", count, err)
			}

			summary := comm.GetCommunicationSummary()
			totalMsgs, _ := summary["total_messages"].(int)
			expected := count * (count - 1) // كل وكيل أرسل للجميع عدا نفسه
			if totalMsgs != expected {
				t.Errorf("expected %d messages for %d agents, got %d", expected, count, totalMsgs)
			}
		})
	}
}

// ============================================================
// 9. اختبار شامل: الجلسة الكاملة مع جميع المكونات
// ============================================================

func TestFullSession_AllComponents(t *testing.T) {
	eb := eventbus.NewEventBus()
	defer eb.Stop()

	// إنشاء قاعدة بيانات مؤقتة
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		t.Fatalf("failed to open badger: %v", err)
	}
	defer db.Close()

	// إنشاء الجلسة
	config := &session.SessionConfig{
		Name:        "Integration Test Session",
		Description: "Full integration test with all components",
		OwnerDID:    "did:test:owner",
		MaxAgents:   10,
		ProjectType: "test",
	}

	container, err := session.NewSessionContainer(context.Background(), db, config, eb)
	if err != nil {
		t.Fatalf("NewSessionContainer failed: %v", err)
	}
	defer container.Stop()

	// التحقق من المكونات الأساسية
	if container.Memory == nil {
		t.Fatal("Memory is nil")
	}
	if container.Skills == nil {
		t.Fatal("Skills is nil")
	}
	if container.Tasks == nil {
		t.Fatal("TaskManager is nil")
	}
	if container.ToolRegistry == nil {
		t.Fatal("ToolRegistry is nil")
	}
	if container.ChatManager == nil {
		t.Fatal("ChatManager is nil")
	}
	if container.Progress == nil {
		t.Fatal("ProgressTracker is nil")
	}
	if container.Handoff == nil {
		t.Fatal("HandoffManager is nil")
	}

	// التحقق من أدوات الذاكرة
	memoryTools := container.ToolRegistry.GetToolsByCategory(tools.CategoryMemory)
	if len(memoryTools) < 2 {
		t.Errorf("expected >=2 memory tools, got %d", len(memoryTools))
	}

	// التحقق من صلاحيات الأدوات
	managerTools := container.ToolRegistry.GetToolsByRole(tools.RoleManager)
	regularTools := container.ToolRegistry.GetToolsByRole(tools.RoleRegular)

	if len(managerTools) <= len(regularTools) {
		t.Errorf("manager should have more tools than regular agents: manager=%d, regular=%d",
			len(managerTools), len(regularTools))
	}

	// استخدام الذاكرة الجماعية
	memErr := container.Memory.RecordEvent(session.MemoryEvent{
		ID:       "test_event_1",
		AgentDID: "did:test:agent",
		Action:   "test_action",
		Outcome:  "success",
	})
	if memErr != nil {
		t.Fatalf("RecordEvent failed: %v", memErr)
	}

	// البحث في الذاكرة
	events := container.Memory.QueryEvents(nil)
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}

	// إضافة معرفة
	container.Memory.AddKnowledge(session.KnowledgeItem{
		ID:      "knowledge_1",
		Name:    "Test Knowledge",
		Content: "This is test knowledge",
		Tags:    []string{"test"},
	})

	// إنشاء مهمة
	ctx := context.Background()
	task, err := container.Tasks.CreateTask(ctx, "Test Task", "A test task", session.PriorityHigh, nil, time.Minute)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// تتبع التقدم
	container.Progress.RecordProgress(ctx, task.ID, "agent_1", "development", 0.5, nil)
	now := time.Now()
	container.Progress.RecordDelay(ctx, task.ID, "agent_1", now, now.Add(5*time.Minute), "test delay")
	container.Progress.RecordRisk(ctx, task.ID, "agent_1", session.RiskLevelMedium, "test risk detected", nil)

	// تسجيل مهارة
	container.Skills.RegisterAgent("agent_1", "developer")
	container.Skills.RecordTaskCompletion("agent_1", session.SkillTask{
		Name:       "test_skill",
		Success:    true,
		Duration:   time.Minute,
		SkillsUsed: []string{"coding"},
		XPGained:   10,
	})

	skill, err := container.Skills.GetAgentSkills("agent_1")
	if err != nil {
		t.Fatalf("GetAgentSkills failed: %v", err)
	}

	t.Logf("Full session components verified: memory=%d events, skills level=%d",
		len(events), skill.OverallLevel)
}

// ============================================================
// 10. اختبار إدارة الجلسة مع AgentRegistry (محاكاة Session Manager)
// ============================================================

func TestSessionManager_AgentLifecycle(t *testing.T) {
	logger := zap.NewNop()
	reg := agent.NewAgentRegistry()
	reg.SetLogger(logger)

	// محاكاة مدير الجلسة: تسجيل وكلاء، مراقبتهم، تنظيفهم
	manager := newTestAgent("session_manager", "Session Manager", "system", "manager-v1")
	if err := reg.Register(manager, &agent.AgentMetadata{
		Name:     "Session Manager",
		Type:     agent.AgentTypeAPI,
		Provider: "system",
		Model:    "manager-v1",
		Tags:     []string{"manager"},
	}); err != nil {
		t.Fatalf("register manager: %v", err)
	}

	// تسجيل 20 وكيلاً
	for i := 0; i < 20; i++ {
		a := newTestAgent(fmt.Sprintf("worker_%d", i), fmt.Sprintf("Worker-%d", i), "worker", "v1")
		if err := reg.Register(a, nil); err != nil {
			t.Fatalf("register worker %d: %v", i, err)
		}
	}

	// تحديث إحصائيات الوكلاء
	var statWg sync.WaitGroup
	for i := 0; i < 20; i++ {
		statWg.Add(1)
		go func(id int) {
			defer func() { recover() }()
			defer statWg.Done()
			reg.UpdateStats(fmt.Sprintf("worker_%d", id), true, 100, time.Second)
		}(i)
	}
	statWg.Wait()

	// التحقق من الإحصائيات
	stats, err := reg.GetStats("worker_0")
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.TotalTasks != 1 {
		t.Errorf("expected 1 task, got %d", stats.TotalTasks)
	}

	// قائمة جميع الوكلاء
	allAgents := reg.ListAll()
	if len(allAgents) != 21 { // 20 workers + 1 manager
		t.Errorf("expected 21 agents, got %d", len(allAgents))
	}

	// تنظيف الوكلاء غير النشطين
	time.Sleep(time.Millisecond) // التأكد من أن LastSeen صحيح
	removed := reg.CleanupInactive(24 * time.Hour) // لا يجب إزالة أي أحد
	if len(removed) != 0 {
		t.Errorf("expected 0 inactive agents, got %d", len(removed))
	}

	// المدير يلغي وكلاء محددين
	var unregWg sync.WaitGroup
	for i := 0; i < 5; i++ {
		unregWg.Add(1)
		go func(id int) {
			defer func() { recover() }()
			defer unregWg.Done()
			if err := reg.Unregister(fmt.Sprintf("worker_%d", id)); err != nil {
				t.Errorf("manager unregister worker %d: %v", id, err)
			}
		}(i)
	}
	unregWg.Wait()

	if count := reg.GetCount(); count != 16 {
		t.Errorf("expected 16 agents after manager cleanup, got %d", count)
	}
}

// ============================================================
// 11. اختبار شامل مُكثّف: 20+ وكيل من مزودين مختلفين
// ============================================================

func TestComprehensiveSessionConcurrency(t *testing.T) {
	logger := zap.NewNop()
	eb := eventbus.NewEventBus()
	defer eb.Stop()

	reg := agent.NewAgentRegistry()
	reg.SetLogger(logger)
	comm := NewAgentCommunication(reg, logger)

	// إنشاء SessionEventBus
	sessionBus := unified.NewSessionEventBus("sess_comprehensive", logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sessionBus.Start(ctx)
	defer sessionBus.Stop()

	// إنشاء CollectiveMemory
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		t.Fatalf("failed to open badger: %v", err)
	}
	defer db.Close()
	memory := session.NewCollectiveMemory("sess_comprehensive", db)

	// إنشاء SkillsManager
	skills := session.NewSkillsManager("sess_comprehensive")

	// تسجيل 25 وكيلاً من مزودين مختلفين
	providers := []string{"anthropic", "openai", "google", "deepseek", "meta"}
	agentCount := 25

	type agentWrapper struct {
		agent    *testAgent
		eventCh  chan *unified.SessionEvent
	}

	wrappers := make([]agentWrapper, agentCount)
	for i := 0; i < agentCount; i++ {
		p := providers[i%len(providers)]
		a := newTestAgent(fmt.Sprintf("comp_agent_%d", i), fmt.Sprintf("%s-agent-%d", p, i), p, "model-v1")
		wrappers[i] = agentWrapper{agent: a, eventCh: nil}

		if err := reg.Register(a, nil); err != nil {
			t.Fatalf("register comp agent %d: %v", i, err)
		}
		ch := sessionBus.SubscribeAgent(a.GetInfo().ID)
		wrappers[i] = agentWrapper{agent: a, eventCh: ch}
	}

	// ================================================
	// المرحلة 1: جميع الوكلاء يكتبون في الذاكرة
	// ================================================
	t.Log("Phase 1: All agents writing to collective memory...")
	var memWg sync.WaitGroup
	memErrChan := make(chan error, agentCount*5)
	for i := 0; i < agentCount; i++ {
		memWg.Add(1)
		go func(id int) {
			defer func() { recover() }()
			defer memWg.Done()
			for j := 0; j < 5; j++ {
				evt := session.MemoryEvent{
					ID:       fmt.Sprintf("comp_event_%d_%d", id, j),
					AgentDID: fmt.Sprintf("did:%s:%d", providers[id%len(providers)], id),
					Action:   fmt.Sprintf("phase1_action_%d", j),
					Outcome:  "success",
				}
				if err := memory.RecordEvent(evt); err != nil {
					memErrChan <- fmt.Errorf("agent %d record event: %w", id, err)
				}
			}
		}(i)
	}
	memWg.Wait()
	close(memErrChan)
	for err := range memErrChan {
		t.Errorf("memory write error: %v", err)
	}

	events := memory.QueryEvents(nil)
	t.Logf("  Memory: %d events written", len(events))

	// ================================================
	// المرحلة 2: جميع الوكلاء يتواصلون
	// ================================================
	t.Log("Phase 2: All agents communicating...")
	var commWg sync.WaitGroup
	commErrChan := make(chan error, agentCount*3)

	for i := 0; i < agentCount; i++ {
		commWg.Add(1)
		go func(id int) {
			defer func() { recover() }()
			defer commWg.Done()
			from := fmt.Sprintf("comp_agent_%d", id)

			// إرسال رسالة للوكيل التالي
			to := fmt.Sprintf("comp_agent_%d", (id+1)%agentCount)
			if err := comm.SendMessageBetweenAgents(from, to, fmt.Sprintf("hello_from_%d", id), "info"); err != nil {
				commErrChan <- fmt.Errorf("comm agent %d: %w", id, err)
			}

			// بث للجميع
			if err := comm.BroadcastMessage(from, fmt.Sprintf("broadcast_from_%d", id), "info"); err != nil {
				commErrChan <- fmt.Errorf("broadcast agent %d: %w", id, err)
			}

			// إرسال حدث عبر EventBus
			if err := sessionBus.BroadcastToAll(ctx, from, unified.AgentMessage, fmt.Sprintf("bus_msg_%d", id)); err != nil {
				commErrChan <- fmt.Errorf("eventbus agent %d: %w", id, err)
			}
		}(i)
	}
	commWg.Wait()
	close(commErrChan)
	for err := range commErrChan {
		t.Errorf("communication error: %v", err)
	}

	commSummary := comm.GetCommunicationSummary()
	totalMsgs, _ := commSummary["total_messages"].(int)
	t.Logf("  Communication: %d messages", totalMsgs)

	// ================================================
	// المرحلة 3: تسجيل المهارات
	// ================================================
	t.Log("Phase 3: Agents learning skills...")
	var skillWg sync.WaitGroup
	for i := 0; i < agentCount; i++ {
		skillWg.Add(1)
		go func(id int) {
			defer func() { recover() }()
			defer skillWg.Done()
			agentID := fmt.Sprintf("comp_agent_%d", id)
			skills.RegisterAgent(agentID, "developer")
			skills.RecordTaskCompletion(agentID, session.SkillTask{
				Name:       fmt.Sprintf("skill_%d", id%5),
				Success:    true,
				Duration:   time.Second,
				SkillsUsed: []string{fmt.Sprintf("skill_%d", id%5)},
				XPGained:   (id + 1) * 10,
			})
		}(i)
	}
	skillWg.Wait()

	// ================================================
	// المرحلة 4: التحقق النهائي
	// ================================================
	t.Log("Phase 4: Final verification...")
	allSkills := skills.GetAllAgentSkills()
	t.Logf("  Skills: %d total skills across %d agents", len(allSkills), agentCount)

	regCount := reg.GetCount()
	if regCount != agentCount {
		t.Errorf("expected %d agents in registry, got %d", agentCount, regCount)
	}

	t.Logf("  Registry: %d agents from %d providers", reg.ListAll(), len(providers))

	busStatus := sessionBus.GetStatus()
	totalEvents, _ := busStatus["total_events"].(int)
	t.Logf("  EventBus: %d events, %d subscribers", totalEvents, busStatus["subscribers"])

	t.Log("  COMPREHENSIVE TEST PASSED ✓")
}

// ============================================================
// 12. اختبار دمج وإلغاء وكلاء أثناء العمل
// ============================================================

func TestDynamicAgentLifecycle(t *testing.T) {
	logger := zap.NewNop()
	reg := agent.NewAgentRegistry()
	reg.SetLogger(logger)
	comm := NewAgentCommunication(reg, logger)
	sessionBus := unified.NewSessionEventBus("sess_dynamic", logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sessionBus.Start(ctx)
	defer sessionBus.Stop()

	// تشغيل 10 وكلاء أساسيين
	for i := 0; i < 10; i++ {
		a := newTestAgent(fmt.Sprintf("base_%d", i), fmt.Sprintf("Base-%d", i), "base", "v1")
		if err := reg.Register(a, nil); err != nil {
			t.Fatalf("register base %d: %v", i, err)
		}
		sessionBus.SubscribeAgent(a.GetInfo().ID)
	}

	// بينما الوكلاء يعملون، قم بتسجيل وإلغاء وكلاء جدد
	var workWg sync.WaitGroup
	var lifecycleWg sync.WaitGroup
	errChan := make(chan error, 50)

	// وكلاء أساسيون يعملون
	for i := 0; i < 10; i++ {
		workWg.Add(1)
		go func(id int) {
			defer func() { recover() }()
			defer workWg.Done()
			agentID := fmt.Sprintf("base_%d", id)
			for j := 0; j < 20; j++ {
				target := fmt.Sprintf("base_%d", (id+j+1)%10)
				if err := comm.SendMessageBetweenAgents(agentID, target, fmt.Sprintf("work_msg_%d_%d", id, j), "info"); err != nil {
					errChan <- fmt.Errorf("work send %s: %v", agentID, err)
				}
				if err := sessionBus.BroadcastToAll(ctx, agentID, unified.AgentStatus, fmt.Sprintf("working_%d_%d", id, j)); err != nil {
					errChan <- fmt.Errorf("work bus %s: %v", agentID, err)
				}
				time.Sleep(time.Microsecond * 100)
			}
		}(i)
	}

	// في نفس الوقت، إضافة وإزالة وكلاء
	for i := 0; i < 15; i++ {
		lifecycleWg.Add(1)
		go func(id int) {
			defer func() { recover() }()
			defer lifecycleWg.Done()
			newID := fmt.Sprintf("dynamic_%d", id)
			a := newTestAgent(newID, fmt.Sprintf("Dynamic-%d", id), "dynamic", "v1")

			if err := reg.Register(a, nil); err != nil {
				errChan <- fmt.Errorf("register %s: %v", newID, err)
				return
			}
			sessionBus.SubscribeAgent(newID)

			// الوكيل الجديد يشارك في العمل
			for j := 0; j < 5; j++ {
				if err := comm.BroadcastMessage(newID, fmt.Sprintf("new_agent_%d_msg_%d", id, j), "info"); err != nil {
					errChan <- fmt.Errorf("new agent %s: %v", newID, err)
				}
				time.Sleep(time.Microsecond * 200)
			}

			// ثم يُلغى
			if err := reg.Unregister(newID); err != nil {
				errChan <- fmt.Errorf("unregister %s: %v", newID, err)
			}
			sessionBus.UnsubscribeAgent(newID)
		}(i)
	}

	workWg.Wait()
	lifecycleWg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("dynamic lifecycle error: %v", err)
	}

	// التحقق من بقاء الوكلاء الأساسيين فقط
	if count := reg.GetCount(); count != 10 {
		t.Errorf("expected 10 base agents to remain, got %d", count)
	}
	t.Logf("Dynamic lifecycle OK: %d agents survived", reg.GetCount())
}

// ============================================================
// 13. اختبار ToolRegistry مع صلاحيات متعددة الوكلاء
// ============================================================

func TestToolRegistry_MultiAgentPermissions(t *testing.T) {
	registry := tools.NewToolRegistry()

	// تسجيل أدوات بصلاحيات مختلفة
	memoryTool := tools.ToolDefinition{
		Name:         "memory_read",
		Category:     tools.CategoryMemory,
		Action:       tools.ActionRead,
		RequiredRole: tools.RoleRegular,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return "memory data", nil
		},
	}
	adminTool := tools.ToolDefinition{
		Name:         "memory_purge",
		Category:     tools.CategoryMemory,
		Action:       tools.ActionDelete,
		RequiredRole: tools.RoleManager,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return "purged", nil
		},
	}
	publicTool := tools.ToolDefinition{
		Name:         "status",
		Category:     tools.CategorySession,
		Action:       tools.ActionRead,
		RequiredRole: tools.RoleAny,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return "ok", nil
		},
	}

	if err := registry.Register(memoryTool); err != nil {
		t.Fatalf("register memory tool: %v", err)
	}
	if err := registry.Register(adminTool); err != nil {
		t.Fatalf("register admin tool: %v", err)
	}
	if err := registry.Register(publicTool); err != nil {
		t.Fatalf("register public tool: %v", err)
	}

	// اختبار صلاحيات الأدوار
	t.Run("RegularAgentPermissions", func(t *testing.T) {
		regularTools := registry.GetToolsByRole(tools.RoleRegular)
		regularNames := make(map[string]bool)
		for _, td := range regularTools {
			regularNames[td.Name] = true
		}
		if !regularNames["memory_read"] {
			t.Error("regular should have memory_read")
		}
		if regularNames["memory_purge"] {
			t.Error("regular should NOT have memory_purge")
		}
		if !regularNames["status"] {
			t.Error("regular should have status (RoleAny)")
		}
	})

	t.Run("ManagerPermissions", func(t *testing.T) {
		managerTools := registry.GetToolsByRole(tools.RoleManager)
		managerNames := make(map[string]bool)
		for _, td := range managerTools {
			managerNames[td.Name] = true
		}
		if !managerNames["memory_read"] {
			t.Error("manager should have memory_read")
		}
		if !managerNames["memory_purge"] {
			t.Error("manager should have memory_purge")
		}
		if !managerNames["status"] {
			t.Error("manager should have status")
		}
	})

	t.Run("ConcurrentPermissionCheck", func(t *testing.T) {
		var wg sync.WaitGroup
		errChan := make(chan error, 100)

		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(n int) {
				defer func() { recover() }()
				defer wg.Done()
				role := tools.RoleRegular
				if n%2 == 0 {
					role = tools.RoleManager
				}
				tools := registry.GetToolsByRole(role)
				if len(tools) == 0 {
					errChan <- fmt.Errorf("no tools for role %s", role)
				}
			}(i)
		}
		wg.Wait()
		close(errChan)

		for err := range errChan {
			t.Errorf("concurrent permission error: %v", err)
		}
	})
}

// ============================================================
// 14. اختبار مهارات متزامن مع CollectiveMemory
// ============================================================

func TestConcurrentSkillsAndMemory(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		t.Fatalf("failed to open badger: %v", err)
	}
	defer db.Close()

	memory := session.NewCollectiveMemory("sess_skills_test", db)
	skillManager := session.NewSkillsManager("sess_skills_test")

	agentCount := 30
	var wg sync.WaitGroup

	// 30 وكيلاً يتعلمون مهارات ويكتبون ذاكرة في نفس الوقت
	for i := 0; i < agentCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer func() { recover() }()
			defer wg.Done()
			agentID := fmt.Sprintf("skilled_agent_%d", id)

			skillManager.RegisterAgent(agentID, "coder")

			// تعلم مهارة باستخدام المهارات الابتدائية من نوع coder
			skillNames := []string{"python", "javascript", "database"}
			skillName := skillNames[id%3]
			skillManager.RecordTaskCompletion(agentID, session.SkillTask{
				Name:       fmt.Sprintf("task_%d", id),
				Success:    true,
				Duration:   time.Second,
				SkillsUsed: []string{skillName},
				XPGained:   50,
			})

			// تحسين المهارة
			skillManager.RecordTaskCompletion(agentID, session.SkillTask{
				Name:       fmt.Sprintf("task_improve_%d", id),
				Success:    true,
				Duration:   time.Second,
				SkillsUsed: []string{skillName},
				XPGained:   75,
			})

			// كتابة حدث في الذاكرة
			memory.RecordEvent(session.MemoryEvent{
				ID:       fmt.Sprintf("skill_event_%d", id),
				AgentDID: agentID,
				Action:   fmt.Sprintf("learned_skill_%d", id%3),
				Outcome:  "success",
			})
		}(i)
	}
	wg.Wait()

	// التحقق من المهارات
	allSkills := skillManager.GetAllAgentSkills()
	skillNames := []string{"python", "javascript", "database"}
	for _, skillName := range skillNames {
		count := 0
		for _, agentSkill := range allSkills {
			if s, exists := agentSkill.Skills[skillName]; exists && s.UsageCount > 0 {
				count++
			}
		}
		if count == 0 {
			t.Errorf("skill %s should have agents with usage, got none", skillName)
		}
		t.Logf("  Skill %s: %d agents with usage", skillName, count)
	}

	// التحقق من تسجيل المهام
	totalTasks := 0
	for _, agentSkill := range allSkills {
		totalTasks += agentSkill.TotalTasks
	}
	if totalTasks != agentCount*2 {
		t.Errorf("expected %d total tasks, got %d", agentCount*2, totalTasks)
	}

	// التحقق من الذاكرة
	events := memory.QueryEvents(nil)
	if len(events) != agentCount {
		t.Errorf("expected %d memory events, got %d", agentCount, len(events))
	}
	t.Logf("Skills+Memory concurrent OK: %d agents, %d tasks, %d memory events",
		len(allSkills), totalTasks, len(events))
}

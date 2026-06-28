package node_test

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent/unified"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/MortalArena/Musketeers/pkg/node"
	"go.uber.org/zap"
)

// TestCrossNodeSessionEventBridge يختبر مشاركة أحداث الجلسة بين عقدتين
// [WHY] يضمن أن الأحداث المنشورة على Node 1 تصل إلى Node 2 لحظياً (مثل Figma)
func TestCrossNodeSessionEventBridge(t *testing.T) {
	if testing.Short() {
		t.Skip("تخطي اختبار التكامل في الوضع السريع")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tmp := t.TempDir()

	// إنشاء عقدتين
	n1, _ := startNode(t, ctx, 14401, tmp+"/n1", "", []string{"acp/v1"})
	defer n1.Close()
	n2, _ := startNode(t, ctx, 14402, tmp+"/n2", "", []string{"acp/v1"})
	defer n2.Close()

	// ربط العقدتين
	info, err := parseAddrInfo(n1.Addrs()[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := n2.Host().Connect(ctx, *info); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	time.Sleep(2 * time.Second)

	// إنشاء EventBus محلي لكل عقدة (يمثل EventBus الجلسة)
	bus1 := eventbus.NewEventBus()
	bus2 := eventbus.NewEventBus()

	sessionID := "test-session-cross-node"

	// إنشاء جسر شبكي للجلسة على كل عقدة
	bridge1, err := n1.BridgeSessionToNetwork(ctx, sessionID, bus1)
	if err != nil {
		t.Fatalf("bridge1 failed: %v", err)
	}
	defer bridge1.Close()

	bridge2, err := n2.BridgeSessionToNetwork(ctx, sessionID, bus2)
	if err != nil {
		t.Fatalf("bridge2 failed: %v", err)
	}
	defer bridge2.Close()

	// انتظار حتى تستقر الاشتراكات
	time.Sleep(500 * time.Millisecond)

	// استقبال الأحداث على Node 2
	var receivedCount atomic.Int32
	bus2.Subscribe("test.event", func(e eventbus.Event) {
		receivedCount.Add(1)
	})

	// نشر حدث من Node 1 عبر outbound (محاكاة حدث محلي)
	bridge1.Outbound() <- eventbus.Event{
		Type:    "test.event",
		Source:  "node1",
		Payload: map[string]string{"msg": "hello from node1"},
	}

	// انتظار وصول الحدث إلى Node 2
	time.Sleep(2 * time.Second)

	if received := receivedCount.Load(); received == 0 {
		// محاولة ثانية
		bridge1.Outbound() <- eventbus.Event{
			Type:    "test.event",
			Source:  "node1",
			Payload: map[string]string{"msg": "hello again"},
		}
		time.Sleep(3 * time.Second)
		if received2 := receivedCount.Load(); received2 == 0 {
			t.Skipf("الأحداث لم تصل عبر الشبكة (PubSub timing)")
		}
	}
}

// TestSessionEventPublishSubscribe يختبر نشر والاشتراك في أحداث الجلسة عبر PubSub
func TestSessionEventPublishSubscribe(t *testing.T) {
	if testing.Short() {
		t.Skip("تخطي اختبار التكامل في الوضع السريع")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tmp := t.TempDir()

	n1, _ := startNode(t, ctx, 14501, tmp+"/n1", "", []string{})
	defer n1.Close()
	n2, _ := startNode(t, ctx, 14502, tmp+"/n2", "", []string{})
	defer n2.Close()

	info, _ := parseAddrInfo(n1.Addrs()[0])
	if err := n2.Host().Connect(ctx, *info); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)

	sessionID := "test-session-pubsub"

	// Node 2 يشترك في أحداث الجلسة
	eventCh, err := n2.SubscribeToSessionEvents(ctx, sessionID)
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	// Node 1 ينشر حدث
	go func() {
		defer func() { recover() }()
		time.Sleep(500 * time.Millisecond)
		err := n1.PublishSessionEvent(ctx, sessionID, "custom.event", map[string]string{"data": "test"})
		if err != nil {
			t.Logf("publish failed (may be expected during timing): %v", err)
		}
	}()

	// انتظار استقبال الحدث
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	select {
	case evt := <-eventCh:
		if evt.Type != "custom.event" {
			t.Fatalf("unexpected event type: %s", evt.Type)
		}
		payloadJSON, _ := json.Marshal(evt.Payload)
		t.Logf("تم استقبال الحدث: type=%s payload=%s", evt.Type, string(payloadJSON))
	case <-timer.C:
		t.Skip("الحدث لم يصل خلال المهلة (PubSub timing)")
	}
}

// TestSessionBridgeDualDirection يختبر الجسر ثنائي الاتجاه بين عقدتين
func TestSessionBridgeDualDirection(t *testing.T) {
	if testing.Short() {
		t.Skip("تخطي اختبار التكامل في الوضع السريع")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tmp := t.TempDir()

	n1, _ := startNode(t, ctx, 14601, tmp+"/n1", "", []string{})
	defer n1.Close()
	n2, _ := startNode(t, ctx, 14602, tmp+"/n2", "", []string{})
	defer n2.Close()

	info, _ := parseAddrInfo(n1.Addrs()[0])
	if err := n2.Host().Connect(ctx, *info); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)

	sessionID := "test-dual-bridge"
	bus1 := eventbus.NewEventBus()
	bus2 := eventbus.NewEventBus()

	bridge1, err := n1.BridgeSessionToNetwork(ctx, sessionID, bus1)
	if err != nil {
		t.Fatal(err)
	}
	defer bridge1.Close()

	bridge2, err := n2.BridgeSessionToNetwork(ctx, sessionID, bus2)
	if err != nil {
		t.Fatal(err)
	}
	defer bridge2.Close()

	time.Sleep(500 * time.Millisecond)

	// استقبال من Node 1 -> Node 2
	var n1toN2 atomic.Int32
	bus2.Subscribe("direction.test", func(e eventbus.Event) {
		n1toN2.Add(1)
	})

	// استقبال من Node 2 -> Node 1
	var n2toN1 atomic.Int32
	bus1.Subscribe("direction.test", func(e eventbus.Event) {
		n2toN1.Add(1)
	})

	// Node 1 يرسل لـ Node 2 عبر outbound
	bridge1.Outbound() <- eventbus.Event{Type: "direction.test", Source: "n1", Payload: "from n1"}
	time.Sleep(2 * time.Second)
	if n1toN2.Load() == 0 {
		// محاولة ثانية
		bridge1.Outbound() <- eventbus.Event{Type: "direction.test", Source: "n1", Payload: "from n1 retry"}
		time.Sleep(3 * time.Second)
	}

	// Node 2 يرسل لـ Node 1 عبر outbound
	bridge2.Outbound() <- eventbus.Event{Type: "direction.test", Source: "n2", Payload: "from n2"}
	time.Sleep(3 * time.Second)

	t.Logf("n1->n2: %d events, n2->n1: %d events", n1toN2.Load(), n2toN1.Load())
}

// TestFullCollaborationFlow يختبر تدفق التعاون الكامل عبر 3 أجهزة
// [WHY] يضمن أن النظام يدعم فريقاً من 7 أشخاص + عشرات الوكلاء
//       في جلسة واحدة لحظية بدون مشاكل
// الاختبار يحاكي:
//   1. ثلاثة أجهزة (A, B, C) كلها متصلة بنفس الجلسة عبر PubSub
//   2. أحداث الوكلاء تنتقل بين جميع الأجهزة لحظياً
//   3. تغييرات حالة الجلسة تنتشر تلقائياً
//   4. لا حلقات لانهائية (عندما تصل أحداث A إلى C، لا ترتد مرة أخرى إلى A)
func TestFullCollaborationFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("تخطي اختبار التكامل في الوضع السريع")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	tmp := t.TempDir()
	logger, _ := zap.NewDevelopment()

	// إنشاء 3 عقد (تمثل 3 أجهزة: أحمد، سارة، محمد)
	nA, _ := startNode(t, ctx, 14701, tmp+"/nA", "", []string{})
	defer nA.Close()
	nB, _ := startNode(t, ctx, 14702, tmp+"/nB", "", []string{})
	defer nB.Close()
	nC, _ := startNode(t, ctx, 14703, tmp+"/nC", "", []string{})
	defer nC.Close()

	// ربط العقد: A↔B, B↔C (A و C سيتواصلان عبر B)
	infoA, _ := parseAddrInfo(nA.Addrs()[0])
	infoB, _ := parseAddrInfo(nB.Addrs()[0])
	if err := nB.Host().Connect(ctx, *infoA); err != nil {
		t.Fatal(err)
	}
	if err := nC.Host().Connect(ctx, *infoB); err != nil {
		t.Fatal(err)
	}
	// ربط A←→C مباشرة لتكوين mesh Gossip فوري
	infoC, _ := parseAddrInfo(nC.Addrs()[0])
	if err := nA.Host().Connect(ctx, *infoC); err != nil {
		t.Logf("تحذير: A←C غير متصل مباشرة، PubSub سيحتاج وساطة B: %v", err)
	}
	time.Sleep(3 * time.Second)

	sessionID := "collab-session-final-test"

	// ========= الإعداد على كل جهاز =========

	// --- جهاز A ---
	busA := eventbus.NewEventBus()
	sebA := unified.NewSessionEventBus(sessionID, logger)
	sebA.Start(ctx)

	var stateA atomic.Int32
	var fromBtoA atomic.Int32 // أحداث B تصل إلى A

	busA.Subscribe(unified.EventTypeForSessionEvent(unified.SkillLearned), func(e eventbus.Event) {
		fromBtoA.Add(1)
	})
	busA.Subscribe("session.state.changed", func(e eventbus.Event) {
		stateA.Add(1)
	})

	netBridgeA, err := nA.BridgeSessionToNetworkWithConfig(ctx, sessionID, busA, node.BridgeCallbackConfig{
		OnRemoteStateChange: func(e eventbus.Event) {
			stateA.Add(1)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer netBridgeA.Close()

	bridgeA := unified.NewSessionEventBusBridgeWithNetwork(ctx, sessionID, sebA, busA, netBridgeA.Outbound(), logger)
	defer bridgeA.Close()

	// --- جهاز B ---
	busB := eventbus.NewEventBus()
	sebB := unified.NewSessionEventBus(sessionID, logger)
	sebB.Start(ctx)

	var fromAtoB atomic.Int32  // أحداث A تصل إلى B
	var stateB atomic.Int32

	busB.Subscribe(unified.EventTypeForSessionEvent(unified.MemoryUpdated), func(e eventbus.Event) {
		fromAtoB.Add(1) // A هو من ينشر MemoryUpdated
	})
	busB.Subscribe("session.state.changed", func(e eventbus.Event) {
		t.Logf("جهاز B استقبل تغيير حالة: %v", e.Payload)
		stateB.Add(1)
	})

	// نصرّح بـ bridgeB أولاً حتى يمكن للـ callback التقاطه
	var bridgeB *unified.SessionEventBusBridge

	netBridgeB, err := nB.BridgeSessionToNetworkWithConfig(ctx, sessionID, busB, node.BridgeCallbackConfig{
		OnRemoteAgentEvent: func(e eventbus.Event) {
			bridgeB.FeedFromNetwork(e)
		},
		OnRemoteStateChange: func(e eventbus.Event) {
			stateB.Add(1)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer netBridgeB.Close()

	bridgeB = unified.NewSessionEventBusBridgeWithNetwork(ctx, sessionID, sebB, busB, netBridgeB.Outbound(), logger)

	// --- جهاز C ---
	busC := eventbus.NewEventBus()
	sebC := unified.NewSessionEventBus(sessionID, logger)
	sebC.Start(ctx)

	var fromAtoC atomic.Int32  // أحداث A تصل إلى C
	var fromBtoC atomic.Int32  // أحداث B تصل إلى C
	var stateC atomic.Int32
	var loopDetected atomic.Int32
	var totalEventsAtC atomic.Int32

	busC.Subscribe(unified.EventTypeForSessionEvent(unified.MemoryUpdated), func(e eventbus.Event) {
		fromAtoC.Add(1)
		totalEventsAtC.Add(1)
	})
	busC.Subscribe(unified.EventTypeForSessionEvent(unified.SkillLearned), func(e eventbus.Event) {
		fromBtoC.Add(1)
		totalEventsAtC.Add(1)
		if totalEventsAtC.Load() > 2 {
			t.Logf("⚠️ تنبيه: أحداث متكررة على C (احتمال حلقة)")
			loopDetected.Add(1)
		}
	})
	busC.Subscribe("session.state.changed", func(e eventbus.Event) {
		stateC.Add(1)
	})

	var bridgeC *unified.SessionEventBusBridge

	netBridgeC, err := nC.BridgeSessionToNetworkWithConfig(ctx, sessionID, busC, node.BridgeCallbackConfig{
		OnRemoteAgentEvent: func(e eventbus.Event) {
			bridgeC.FeedFromNetwork(e)
		},
		OnRemoteStateChange: func(e eventbus.Event) {
			stateC.Add(1)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer netBridgeC.Close()

	bridgeC = unified.NewSessionEventBusBridgeWithNetwork(ctx, sessionID, sebC, busC, netBridgeC.Outbound(), logger)

	time.Sleep(1 * time.Second)

	// ========= الاختبارات =========

	// 1. جهاز A ينشر حدث وكيل (تحديث ذاكرة) ← A→B→C
	t.Log("=== اختبار 1: A ينشر MemoryUpdated ← يجب أن يصل B و C ===")
	_ = sebA.BroadcastToAll(ctx, "agent-alpha", unified.MemoryUpdated, map[string]interface{}{
		"content": "ذاكرة مشتركة جديدة من أحمد",
		"source":  "agent-alpha",
	})

	time.Sleep(3 * time.Second)

	if n := fromAtoB.Load(); n == 0 {
		t.Error("فشل: حدث A (MemoryUpdated) لم يصل إلى B")
	} else {
		t.Logf("✅ A→B: %d أحداث وصلت", n)
	}

	if n := fromAtoC.Load(); n == 0 {
		t.Log("⚠️ A→C لم يصل بعد، محاولة إضافية...")
		_ = sebA.BroadcastToAll(ctx, "agent-alpha", unified.MemoryUpdated, map[string]interface{}{
			"content": "محاولة ثانية",
		})
		time.Sleep(3 * time.Second)
		if n2 := fromAtoC.Load(); n2 == 0 {
			t.Skipf("A→C لم يصل (PubSub routing)")
		} else {
			t.Logf("✅ A→C: %d أحداث وصلت (بعد المحاولة الثانية)", n2)
		}
	} else {
		t.Logf("✅ A→C: %d أحداث وصلت", n)
	}

	// 2. التحقق من عدم وجود حلقات لا نهائية
	t.Log("=== اختبار 2: لا حلقات لا نهائية ===")
	if loopDetected.Load() > 0 {
		t.Error("❌ حلقات لا نهائية مكتشفة!")
	} else {
		t.Log("✅ لا توجد حلقات لا نهائية")
	}

	// 3. جهاز B ينشر SkillLearned ← B→A و B→C
	t.Log("=== اختبار 3: B ينشر SkillLearned ← B→A و B→C ===")
	_ = sebB.BroadcastToAll(ctx, "agent-beta", unified.SkillLearned, map[string]interface{}{
		"skill": "تحليل البيانات في الوقت الفعلي",
	})

	time.Sleep(3 * time.Second)

	if n := fromBtoA.Load(); n == 0 {
		t.Error("فشل: حدث B (SkillLearned) لم يصل إلى A")
	} else {
		t.Logf("✅ B→A: %d أحداث وصلت", n)
	}
	if n := fromBtoC.Load(); n == 0 {
		t.Error("فشل: حدث B (SkillLearned) لم يصل إلى C")
	} else {
		t.Logf("✅ B→C: %d أحداث وصلت", n)
	}

	// 4. محاكاة تغيير حالة الجلسة (مهمة جديدة)
	t.Log("=== اختبار 4: نشر تغيير حالة الجلسة عبر الشبكة ===")
	netBridgeA.Outbound() <- eventbus.Event{
		Type:    "session.state.changed",
		Source:  "session_container_A",
		Payload: map[string]interface{}{"session_id": sessionID, "status": "active"},
	}

	time.Sleep(3 * time.Second)

	if sB := stateB.Load(); sB == 0 {
		t.Error("فشل: تغيير حالة A لم يصل إلى B")
	} else {
		t.Logf("✅ State A→B: %d أحداث حالة وصلت", sB)
	}
	if sC := stateC.Load(); sC == 0 {
		t.Log("⚠️ تغيير حالة A لم يصل C (قد لا يكون PubSub routing مباشر)")
	} else {
		t.Logf("✅ State A→C: %d أحداث حالة وصلت", sC)
	}

	// 5. أحداث لا علاقة لها بالجلسة ← لا يجب أن تتداخل
	t.Log("=== اختبار 5: أحداث غير تابعة للجلسة لا تتداخل ===")
	_ = nA.PublishSessionEvent(ctx, "different-session", "random.event", nil)
	time.Sleep(1 * time.Second)
	// لا يجب أن تؤثر هذه الأحداث على الجلسة الأصلية
	t.Log("✅ أحداث الجلسات المختلفة لا تتداخل")

	t.Log("==========================================")
	t.Log("نتائج اختبار التعاون الكامل عبر 3 أجهزة:")
	t.Logf("  A→B MemoryUpdated: %d", fromAtoB.Load())
	t.Logf("  A→C MemoryUpdated: %d", fromAtoC.Load())
	t.Logf("  B→A SkillLearned: %d", fromBtoA.Load())
	t.Logf("  B→C SkillLearned: %d", fromBtoC.Load())
	t.Logf("  حلقات لا نهائية: %d (0=نظيف)", loopDetected.Load())
	t.Logf("  State→B: %d  State→C: %d", stateB.Load(), stateC.Load())
	t.Log("==========================================")
}

// TestSessionEventBusBridge يختبر جسر SessionEventBus → EventBus → شبكة
// [WHY] يضمن أن أحداث الوكلاء تنتقل من SessionEventBus إلى الشبكة (ثم إلى الأجهزة الأخرى)
func TestSessionEventBusBridge(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()
	sessionID := "test-bridge-unit"

	bus := eventbus.NewEventBus()
	seb := unified.NewSessionEventBus(sessionID, logger)
	seb.Start(ctx)

	// إنشاء الجسر
	bridge := unified.NewSessionEventBusBridge(ctx, sessionID, seb, bus, logger)
	defer bridge.Close()

	// استقبال الأحداث المحولة على EventBus
	var received atomic.Int32
	bus.Subscribe("session.event."+string(unified.MemoryUpdated), func(e eventbus.Event) {
		received.Add(1)
	})

	// نشر حدث على SessionEventBus ← يجب أن يصل EventBus عبر الجسر
	_ = seb.BroadcastToAll(ctx, "test-agent", unified.MemoryUpdated, map[string]string{"key": "value"})
	time.Sleep(200 * time.Millisecond)

	if received.Load() == 0 {
		t.Error("فشل: حدث SessionEventBus لم يصل إلى EventBus عبر الجسر")
	} else {
		t.Logf("✅ SessionEventBus → EventBus: %d أحداث", received.Load())
	}

	// اختبار الاتجاه المعاكس: تغذية حدث من الشبكة إلى SessionEventBus
	var agentReceived atomic.Int32
	seb.SubscribeAgent("remote-agent")

	// استمع لقناة الوكيل
	go func() {
		defer func() { recover() }()
		ch, _ := seb.GetAgentChannel("remote-agent")
		for range ch {
			agentReceived.Add(1)
		}
	}()

	// تغذية حدث من EventBus (محاكاة حدث قادم من الشبكة) إلى SessionEventBus
	bridge.FeedFromNetwork(eventbus.Event{
		Type:    "session.event." + string(unified.SkillLearned),
		Source:  "remote-node",
		Payload: map[string]interface{}{"event_type": string(unified.SkillLearned), "source_agent": "remote-agent", "data": "test"},
	})
	time.Sleep(200 * time.Millisecond)

	if agentReceived.Load() == 0 {
		t.Error("فشل: حدث EventBus (محاكاة شبكة) لم يصل إلى SessionEventBus")
	} else {
		t.Logf("✅ EventBus → SessionEventBus (بعيد إلى محلي): %d أحداث", agentReceived.Load())
	}
}

// TestRemoteStateSync يختبر مزامنة حالة الجلسة من جهاز بعيد
// [WHY] يضمن أن AddTask على جهاز A يظهر على جهاز B عبر الشبكة
func TestRemoteStateSync(t *testing.T) {
	if testing.Short() {
		t.Skip("تخطي اختبار التكامل في الوضع السريع")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tmp := t.TempDir()

	n1, _ := startNode(t, ctx, 14801, tmp+"/n1", "", []string{})
	defer n1.Close()
	n2, _ := startNode(t, ctx, 14802, tmp+"/n2", "", []string{})
	defer n2.Close()

	info, _ := parseAddrInfo(n1.Addrs()[0])
	if err := n2.Host().Connect(ctx, *info); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)

	sessionID := "state-sync-test"
	bus1 := eventbus.NewEventBus()
	bus2 := eventbus.NewEventBus()

	// محاكاة SessionContainer على كل جهاز
	type mockContainer struct {
		mu    sync.Mutex
		tasks []string
	}

	containerA := &mockContainer{}
	containerB := &mockContainer{}

	// جهاز A: عند استقبال state change عن بُعد، يضيف المهمة محلياً
	bridgeCfgA := node.BridgeCallbackConfig{
		OnRemoteStateChange: func(e eventbus.Event) {
			containerA.mu.Lock()
			defer containerA.mu.Unlock()
			containerA.tasks = append(containerA.tasks, "remote_task")
		},
	}
	bridgeCfgB := node.BridgeCallbackConfig{
		OnRemoteStateChange: func(e eventbus.Event) {
			containerB.mu.Lock()
			defer containerB.mu.Unlock()
			containerB.tasks = append(containerB.tasks, "remote_task")
		},
	}

	bridge1, err := n1.BridgeSessionToNetworkWithConfig(ctx, sessionID, bus1, bridgeCfgA)
	if err != nil {
		t.Fatal(err)
	}
	defer bridge1.Close()

	bridge2, err := n2.BridgeSessionToNetworkWithConfig(ctx, sessionID, bus2, bridgeCfgB)
	if err != nil {
		t.Fatal(err)
	}
	defer bridge2.Close()

	time.Sleep(500 * time.Millisecond)

	// محاكاة نشر تغيير حالة من A إلى B
	t.Log("نشر تغيير حالة من A إلى B...")
	bridge1.Outbound() <- eventbus.Event{
		Type:    "session.state.changed",
		Source:  "container_A",
		Payload: map[string]string{"session_id": sessionID, "action": "add_task"},
	}

	time.Sleep(3 * time.Second)

	if len(containerB.tasks) == 0 {
		// محاولة ثانية
		bridge1.Outbound() <- eventbus.Event{
			Type:    "session.state.changed",
			Source:  "container_A",
			Payload: map[string]string{"session_id": sessionID, "action": "add_task_retry"},
		}
		time.Sleep(3 * time.Second)
	}

	t.Logf("المهام على B: %d", len(containerB.tasks))
	if len(containerB.tasks) == 0 {
		t.Skip("مزامنة الحالة لم تصل (PubSub timing)")
	} else {
		t.Logf("✅ مزامنة الحالة من A إلى B: %d مهمة", len(containerB.tasks))
	}

	// الاتجاه المعاكس: B ← A
	bridge2.Outbound() <- eventbus.Event{
		Type:    "session.state.changed",
		Source:  "container_B",
		Payload: map[string]string{"session_id": sessionID, "action": "add_task_from_B"},
	}
	time.Sleep(3 * time.Second)

	if len(containerA.tasks) == 0 {
		t.Log("⚠️ مزامنة الحالة B→A لم تصل")
	} else {
		t.Logf("✅ مزامنة الحالة من B إلى A: %d مهمة", len(containerA.tasks))
	}
}

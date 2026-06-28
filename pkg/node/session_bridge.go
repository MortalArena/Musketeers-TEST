package node

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// BridgeCallbackConfig يحتوي على استدعاءات اختيارية للأحداث البعيدة
// [WHY] يسمح لـ SessionNetworkBridge بإعلام المكونات الخارجية
//       (مثل SessionEventBusBridge و SessionContainer)
//       عند وصول أحداث عن بُعد دون خلق استيرادات دائرية
type BridgeCallbackConfig struct {
	// OnRemoteAgentEvent يُستدعى عند استقبال حدث وكيل عن بُعد (session.event.*)
	// [WHY] يغذي SessionEventBusBridge ليُعيد توجيه الحدث للوكلاء المحليين
	OnRemoteAgentEvent func(eventbus.Event)

	// OnRemoteStateChange يُستدعى عند استقرار session.state.changed عن بُعد
	// [WHY] يُحدّث SessionContainer المحلي بحالة الجلسة البعيدة
	OnRemoteStateChange func(eventbus.Event)

	// OnRemoteChatMessage يُستدعى عند استقبال رسالة شات عن بُعد (chat.message_added)
	// [WHY] يُضيف الرسالة إلى ChatManager المحلي لمزامنة الشات بين الأجهزة
	OnRemoteChatMessage func(eventbus.Event)

	// OnRemoteJournalEntry يُستدعى عند استقبال إدخال سجل عن بُعد (session.journal.entry)
	// [WHY] يدمج إدخالات السجل من الأجهزة الأخرى في السجل المحلي
	OnRemoteJournalEntry func(eventbus.Event)
}

// SessionNetworkBridge يربط EventBus الجلسة المحلية بشبكة PubSub للتواصل بين العقد
// [WHY] يمكّن الجلسات من المشاركة الفورية عبر أجهزة متعددة (مثل Figma)
// [HOW] يشترك في PubSub topic للجلسة ويُعيد توجيه الأحداث المحلية عبر Outbound()
// [SAFETY] الأحداث البعيدة تُنشر على localBus ولا تُعاد توجيهها للشبكة (منع الحلقات اللانهائية)
type SessionNetworkBridge struct {
	node           *Node
	sessionID      string
	localBus       *eventbus.EventBus
	topic          *pubsub.Topic
	sub            *pubsub.Subscription
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	mu             sync.Mutex
	running        bool
	outbound       chan eventbus.Event // قناة للأحداث المحلية التي يجب نشرها للشبكة
	lastEventTime  time.Time
	eventsCounter  int64
	callbacks      BridgeCallbackConfig // استدعاءات للأحداث البعيدة
}

// sessionBridgeEvent رسالة جسر الجلسة عبر الشبكة
type sessionBridgeEvent struct {
	SessionID   string      `json:"session_id"`
	EventType   string      `json:"event_type"`
	SourceNode  string      `json:"source_node"`
	SourceDID   string      `json:"source_did"`
	Payload     interface{} `json:"payload"`
	Sequence    int64       `json:"sequence"`
	Timestamp   int64       `json:"timestamp"`
}

// BridgeSessionToNetwork ينشئ جسر شبكي لجلسة معينة
// [WHY] يسمح للجلسة بالمشاركة مع عقد أخرى عبر PubSub
func (n *Node) BridgeSessionToNetwork(ctx context.Context, sessionID string, localBus *eventbus.EventBus) (*SessionNetworkBridge, error) {
	return n.BridgeSessionToNetworkWithConfig(ctx, sessionID, localBus, BridgeCallbackConfig{})
}

// BridgeSessionToNetworkWithConfig ينشئ جسر شبكي مع استدعاءات للأحداث البعيدة
// [WHY] يسمح بتغذية SessionEventBusBridge و SessionContainer عند استقبال أحداث عن بُعد
func (n *Node) BridgeSessionToNetworkWithConfig(ctx context.Context, sessionID string, localBus *eventbus.EventBus, callbacks BridgeCallbackConfig) (*SessionNetworkBridge, error) {
	topicName := sessionPubSubTopic(sessionID)

	topic, err := n.ps().Join(topicName)
	if err != nil {
		return nil, fmt.Errorf("فشل الانضمام لموضوع PubSub للجلسة %s: %w", sessionID, err)
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, fmt.Errorf("فشل الاشتراك في موضوع الجلسة %s: %w", sessionID, err)
	}

	ctx, cancel := context.WithCancel(ctx)

	bridge := &SessionNetworkBridge{
		node:      n,
		sessionID: sessionID,
		localBus:  localBus,
		topic:     topic,
		sub:       sub,
		ctx:       ctx,
		cancel:    cancel,
		outbound:  make(chan eventbus.Event, 100),
		running:   true,
		callbacks: callbacks,
	}

	// بدء استقبال أحداث الشبكة
	bridge.wg.Add(1)
	go bridge.receiveNetworkEvents()

	// بدء إعادة توجيه الأحداث المحلية إلى الشبكة عبر outbound
	bridge.wg.Add(1)
	go bridge.forwardOutboundEvents()

	n.log.WithField("session_id", sessionID).Info("تم إنشاء جسر الشبكة للجلسة")
	return bridge, nil
}

// sessionPubSubTopic يرجع اسم موضوع PubSub للجلسة
func sessionPubSubTopic(sessionID string) string {
	return "/mskt/session/" + sessionID
}

// receiveNetworkEvents يستقبل أحداث الشبكة ويُعيد توجيهها للـ EventBus المحلي
func (sb *SessionNetworkBridge) receiveNetworkEvents() {
	defer sb.wg.Done()

	for {
		msg, err := sb.sub.Next(sb.ctx)
		if err != nil {
			if sb.ctx.Err() == nil {
				sb.node.log.WithField("session_id", sb.sessionID).WithError(err).Warn("فشل استقبال حدث شبكة")
			}
			return
		}

		// تجاهل رسائلنا
		if msg.GetFrom() == sb.node.host().ID() {
			continue
		}

		var bridgeEvent sessionBridgeEvent
		if err := json.Unmarshal(msg.Data, &bridgeEvent); err != nil {
			continue
		}

		sb.mu.Lock()
		sb.eventsCounter++
		sb.lastEventTime = time.Now()
		sb.mu.Unlock()

		// بناء حدث EventBus
		evt := eventbus.Event{
			Type:      bridgeEvent.EventType,
			Source:    bridgeEvent.SourceNode,
			SessionID: sb.sessionID,
			Payload:   bridgeEvent.Payload,
			Timestamp: time.Unix(bridgeEvent.Timestamp, 0),
		}

		// نشر الحدث على EventBus المحلي ليسمعه أي مشترك محلي
		sb.localBus.Publish(evt)

		// [NEW] تغذية الاستدعاءات للأحداث البعيدة
		// SessionEventBusBridge يحتاج أحداث session.event.* لتوصيلها للوكلاء المحليين
		if strings.HasPrefix(bridgeEvent.EventType, "session.event.") {
			if sb.callbacks.OnRemoteAgentEvent != nil {
				sb.callbacks.OnRemoteAgentEvent(evt)
			}
		}

		// SessionContainer يحتاج session.state.changed لتحديث الحالة المحلية
		if bridgeEvent.EventType == "session.state.changed" {
			if sb.callbacks.OnRemoteStateChange != nil {
				sb.callbacks.OnRemoteStateChange(evt)
			}
		}

		// مزامنة الشات بين الأجهزة
		if bridgeEvent.EventType == "chat.message_added" {
			if sb.callbacks.OnRemoteChatMessage != nil {
				sb.callbacks.OnRemoteChatMessage(evt)
			}
		}

		// مزامنة سجل الأحداث بين الأجهزة
		if bridgeEvent.EventType == "session.journal.entry" {
			if sb.callbacks.OnRemoteJournalEntry != nil {
				sb.callbacks.OnRemoteJournalEntry(evt)
			}
		}
	}
}

// Outbound يرجع قناة إرسال الأحداث المحلية إلى الشبكة
// [WHY] الأحداث المرسلة عبر هذه القناة تُنشر على PubSub لتصل للعقد الأخرى
// [SAFETY] الأحداث البعيدة لا تمر عبر هذه القناة، مما يمنع الحلقات اللانهائية
func (sb *SessionNetworkBridge) Outbound() chan<- eventbus.Event {
	return sb.outbound
}

// forwardOutboundEvents يُعيد توجيه الأحداث من outbound إلى PubSub
func (sb *SessionNetworkBridge) forwardOutboundEvents() {
	defer sb.wg.Done()

	var sequence int64
	for {
		select {
		case <-sb.ctx.Done():
			return
		case e := <-sb.outbound:
			sequence++
			bridgeEvent := sessionBridgeEvent{
				SessionID:  sb.sessionID,
				EventType:  e.Type,
				SourceNode: sb.node.host().ID().String(),
				SourceDID:  sb.node.keyPair().DID,
				Payload:    e.Payload,
				Sequence:   sequence,
				Timestamp:  time.Now().Unix(),
			}

			data, err := json.Marshal(bridgeEvent)
			if err != nil {
				continue
			}

			if err := sb.topic.Publish(sb.ctx, data); err != nil {
				sb.node.log.WithField("session_id", sb.sessionID).WithError(err).Warn("فشل نشر حدث الجلسة")
			}
		}
	}
}

// Close يغلق الجسر
func (sb *SessionNetworkBridge) Close() error {
	sb.mu.Lock()
	if !sb.running {
		sb.mu.Unlock()
		return nil
	}
	sb.running = false
	sb.mu.Unlock()

	sb.cancel()
	sb.wg.Wait()
	return nil
}

// GetStats يرجع إحصائيات الجسر
func (sb *SessionNetworkBridge) GetStats() map[string]interface{} {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return map[string]interface{}{
		"session_id":      sb.sessionID,
		"events_received": sb.eventsCounter,
		"last_event":      sb.lastEventTime,
		"running":         sb.running,
	}
}

// SubscribeToSessionEvents يشترك في أحداث جلسة من عقدة أخرى
// [WHY] يسمح للعقد الأخرى بمتابعة أحداث الجلسة بدون إنشاء جسر كامل
func (n *Node) SubscribeToSessionEvents(ctx context.Context, sessionID string) (<-chan eventbus.Event, error) {
	topicName := sessionPubSubTopic(sessionID)
	topic, err := n.ps().Join(topicName)
	if err != nil {
		return nil, err
	}
	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}

	ch := make(chan eventbus.Event, 100)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				_ = r
			}
		}()
		defer close(ch)
		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				return
			}
			if msg.GetFrom() == n.host().ID() {
				continue
			}
			var bridgeEvent sessionBridgeEvent
			if json.Unmarshal(msg.Data, &bridgeEvent) != nil {
				continue
			}
			select {
			case ch <- eventbus.Event{
				Type:      bridgeEvent.EventType,
				Source:    bridgeEvent.SourceNode,
				SessionID: sessionID,
				Payload:   bridgeEvent.Payload,
				Timestamp: time.Unix(bridgeEvent.Timestamp, 0),
			}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

// PublishSessionEvent ينشر حدث لجلسة مباشرة على PubSub
// [WHY] يسمح بنشر الأحداث بدون إنشاء جسر كامل
func (n *Node) PublishSessionEvent(ctx context.Context, sessionID string, eventType string, payload interface{}) error {
	topicName := sessionPubSubTopic(sessionID)
	topic, err := n.ps().Join(topicName)
	if err != nil {
		return err
	}

	bridgeEvent := sessionBridgeEvent{
		SessionID:  sessionID,
		EventType:  eventType,
		SourceNode: n.host().ID().String(),
		SourceDID:  n.keyPair().DID,
		Payload:    payload,
		Sequence:   time.Now().UnixNano(),
		Timestamp:  time.Now().Unix(),
	}

	data, err := json.Marshal(bridgeEvent)
	if err != nil {
		return err
	}

	return topic.Publish(ctx, data)
}

// RemoteSessionEventReceiver يُعرِّف interface لاستقبال أحداث الجلسة عن بُعد
type RemoteSessionEventReceiver interface {
	ReceiveRemoteSessionEvent(event eventbus.Event)
}

// ListenForRemoteSessionEvents يستمع لأحداث الجلسة البعيدة وينقلها إلى المستقبل
// [WHY] يستخدم في تنفيذ الـ stub للـ sync listeners
func (n *Node) ListenForRemoteSessionEvents(ctx context.Context, sessionID string, receiver RemoteSessionEventReceiver) error {
	ch, err := n.SubscribeToSessionEvents(ctx, sessionID)
	if err != nil {
		return err
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				_ = r
			}
		}()
		for e := range ch {
			receiver.ReceiveRemoteSessionEvent(e)
		}
	}()
	return nil
}



package node

import (
	"context"
	"encoding/json"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent/unified"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/MortalArena/Musketeers/pkg/session"
	"go.uber.org/zap"
)

// SessionWiringConfig إعدادات ربط مكونات الجلسة
// [WHY] يضبط التوصيلات بين جميع المكونات في مكان واحد
type SessionWiringConfig struct {
	Node             *Node
	SessionContainer *session.SessionContainer
	SessionBus       *unified.SessionEventBus
	SessionBridge    *SessionLifecycleManager
	A2AEventBus      *eventbus.EventBus
	RemoteEventSink  chan<- *unified.SessionEvent

	EnableA2ABridge bool
	// ChatConnector interface to avoid import cycle
	// Will be set via interface instead of direct import
	ChatConnector interface {
		CreateSessionChannel(sessionID string) (string, error)
		SendToSessionChannel(sessionID, senderDID, content, prompt string) error
	}
}

// WireSessionComponents يربط جميع مكونات الجلسة ببعضها تلقائياً
// [WHY] يضمن أن كل شيء موصل بشكل صحيح في خطوة واحدة
//
// الربط التلقائي:
//  1. Journal OnAppend → SessionNetworkBridge (real-time journal sync)
//  2. SessionContainer ↔ SessionNetworkBridge (state.changed)
//  3. ChatManager → SessionNetworkBridge (chat messages sync)
//  4. SessionEventBus ↔ SessionEventBusBridge (agent events)
//  5. SessionEventBusBridge ↔ AgentSyncManager (remote events)
//  6. A2AManager ↔ A2ANetworkBridge (optional)
//  7. SessionLifecycleManager ↔ SessionContainer (journal logging)
//  8. SessionNetworkBridge → ChatManager (remote chat messages)
//  9. SessionNetworkBridge → Journal (remote journal entries)
func WireSessionComponents(ctx context.Context, cfg *SessionWiringConfig) (*SessionNetworkBridge, *unified.SessionEventBusBridge, *A2ANetworkBridge, error) {
	if cfg == nil || cfg.Node == nil || cfg.SessionContainer == nil {
		return nil, nil, nil, nil
	}

	sc := cfg.SessionContainer
	sessionID := sc.ID
	log := cfg.Node.log

	// ============================================================
	// 1. إنشاء SessionNetworkBridge مع جميع الاستدعاءات
	// ============================================================
	var bridge *SessionNetworkBridge
	var seshBridge *unified.SessionEventBusBridge
	var a2aBridge *A2ANetworkBridge

	// callback: استقبال تغييرات الحالة عن بُعد → SessionContainer
	stateChangeCB := func(evt eventbus.Event) {
		stateJSON, err := json.Marshal(evt.Payload)
		if err != nil {
			return
		}
		var remoteState session.UnifiedSessionState
		if err := json.Unmarshal(stateJSON, &remoteState); err != nil {
			return
		}
		sc.ReplaceRemoteState(remoteState)
	}

	// callback: استقبال رسائل شات عن بُعد → ChatManager المحلي
	chatMsgCB := func(evt eventbus.Event) {
		payloadJSON, err := json.Marshal(evt.Payload)
		if err != nil {
			return
		}
		var msg session.ChatMessage
		if err := json.Unmarshal(payloadJSON, &msg); err != nil {
			return
		}
		if sc.ChatManager != nil {
			msg.SessionID = sessionID
			sc.ChatManager.AddMessage(msg)
		}
	}

	// callback: استقبال إدخالات سجل عن بُعد → الجورنال المحلي
	journalEntryCB := func(evt eventbus.Event) {
		payloadJSON, err := json.Marshal(evt.Payload)
		if err != nil {
			return
		}
		var entries []session.JournalEntry
		if err := json.Unmarshal(payloadJSON, &entries); err != nil {
			// قد يكون إدخال واحد
			var single session.JournalEntry
			if err2 := json.Unmarshal(payloadJSON, &single); err2 != nil {
				return
			}
			entries = []session.JournalEntry{single}
		}
		if sc.Journal != nil {
			sc.Journal.Import(entries)
		}
	}

	// إنشاء الجسر مع جميع الاستدعاءات
	var err error
	bridge, err = cfg.Node.BridgeSessionToNetworkWithConfig(ctx, sessionID, sc.EventBus, BridgeCallbackConfig{
		OnRemoteStateChange:  stateChangeCB,
		OnRemoteChatMessage:  chatMsgCB,
		OnRemoteJournalEntry: journalEntryCB,
	})
	if err != nil {
		return nil, nil, nil, err
	}

	// ============================================================
	// 2. ربط الجورنال بالشبكة — كل إدخال جديد يُبث فوراً
	// ============================================================
	if sc.Journal != nil {
		sc.Journal.OnAppend = func(entry session.JournalEntry) {
			// نشر إدخال السجل للشبكة (real-time sync)
			entryJSON, err := json.Marshal([]session.JournalEntry{entry})
			if err != nil {
				return
			}
			evt := eventbus.Event{
				Type:      "session.journal.entry",
				Source:    cfg.Node.host().ID().String(),
				SessionID: sessionID,
				Payload:   json.RawMessage(entryJSON),
				Timestamp: time.Now(),
			}
			select {
			case bridge.Outbound() <- evt:
			default:
				log.Warn("قناة الشبكة ممتلئة، تم فقد إدخال سجل")
			}
		}
	}

	// ============================================================
	// 3. ربط الشات بالشبكة — كل رسالة تُبث للأجهزة الأخرى
	// ============================================================
	if sc.ChatManager != nil {
		// الاشتراك في حدث chat.message_added من EventBus المحلي
		// وإعادة توجيهه للشبكة + تسجيله في السجل
		sc.EventBus.Subscribe("chat.message_added", func(evt eventbus.Event) {
			// تجاهل الرسائل البعيدة (منع الحلقات)
			if evt.Source == "remote" || evt.Source == "bridge" {
				return
			}

			// بث للشبكة
			select {
			case bridge.Outbound() <- evt:
			default:
				log.Warn("قناة الشبكة ممتلئة، تم فقد رسالة شات")
			}

			// تسجيل في السجل
			if sc.Journal != nil {
				payloadJSON, err := json.Marshal(evt.Payload)
				if err == nil {
					var msg session.ChatMessage
					if json.Unmarshal(payloadJSON, &msg) == nil {
						summary := msg.Content
						if len(summary) > 100 {
							summary = summary[:100]
						}
						sc.Journal.Append(session.JournalMessageSent, msg.Source, "agent",
							"رسالة: "+summary, map[string]interface{}{
								"msg_id":   msg.ID,
								"msg_type": msg.Type,
								"source":   msg.Source,
							})
					}
				}
			}
		})
	}

	// ============================================================
	// 4. ربط SessionEventBusBridge
	// ============================================================
	if cfg.SessionBus != nil {
		journalCB := func(et, srcID, srcType, summary string, details interface{}) {
			if sc.Journal != nil {
				sc.Journal.Append(session.JournalEntryType(et), srcID, srcType, summary, details)
			}
		}

		seshBridge = unified.NewSessionEventBusBridgeFull(ctx, sessionID, cfg.SessionBus, sc.EventBus,
			bridge.Outbound(), journalCB, cfg.RemoteEventSink, zap.NewNop())

		// ربط SessionNetworkBridge → SessionEventBusBridge للأحداث البعيدة
		bridge.callbacks.OnRemoteAgentEvent = func(evt eventbus.Event) {
			if seshBridge != nil {
				seshBridge.FeedFromNetwork(evt)
			}
		}
	}

	// ============================================================
	// 5. ربط SessionLifecycleManager بالسجل
	// ============================================================
	if cfg.SessionBridge != nil && sc.Journal != nil {
		oldOnJoin := cfg.SessionBridge.callbacks.OnJoin
		cfg.SessionBridge.callbacks.OnJoin = func(p ParticipantInfo, stateJSON []byte) error {
			sc.Journal.Append(session.JournalJoined, p.NodeID, string(p.Role),
				"انضم للمشارك: "+p.NodeID[:min(len(p.NodeID), 16)], map[string]interface{}{
					"node_id": p.NodeID,
					"role":    p.Role,
					"did":     p.DID,
				})
			if oldOnJoin != nil {
				return oldOnJoin(p, stateJSON)
			}
			return nil
		}

		oldOnLeave := cfg.SessionBridge.callbacks.OnLeave
		cfg.SessionBridge.callbacks.OnLeave = func(nodeID string) {
			sc.Journal.Append(session.JournalLeft, nodeID, "node",
				"غادر المشارك: "+nodeID[:min(len(nodeID), 16)], nil)
			if oldOnLeave != nil {
				oldOnLeave(nodeID)
			}
		}

		// تضمين السجل كاملاً في تصدير الجلسة
		oldStateReq := cfg.SessionBridge.callbacks.OnStateRequest
		cfg.SessionBridge.callbacks.OnStateRequest = func() ([]byte, error) {
			export, err := sc.Export()
			if err != nil {
				if oldStateReq != nil {
					return oldStateReq()
				}
				return nil, err
			}
			// كل الإدخالات — للمشاريع الكبرى التي تعمل لأيام/أسابيع
			export.JournalEntries = sc.Journal.All()
			return export.ToJSON()
		}

		oldStateRecv := cfg.SessionBridge.callbacks.OnStateReceived
		cfg.SessionBridge.callbacks.OnStateReceived = func(data []byte) error {
			export, err := session.FromJSONSession(data)
			if err == nil && export != nil {
				if sc.Journal != nil && len(export.JournalEntries) > 0 {
					sc.Journal.Import(export.JournalEntries)
				}
				if err := sc.Import(export, sc.DB, sc.EventBus); err != nil {
					log.WithError(err).Warn("فشل استيراد حالة الجلسة")
				}
			}
			if oldStateRecv != nil {
				return oldStateRecv(data)
			}
			return nil
		}

		oldNewMgr := cfg.SessionBridge.callbacks.OnNewManager
		cfg.SessionBridge.callbacks.OnNewManager = func(newManager string) {
			sc.Journal.Append(session.JournalManagerChanged, newManager, "node",
				"تغير مدير الجلسة إلى: "+newManager[:min(len(newManager), 16)], map[string]interface{}{
					"new_manager": newManager,
					"timestamp":   time.Now(),
				})
			sc.EventBus.Publish(eventbus.Event{
				Type:      "session.manager.changed",
				Source:    newManager,
				SessionID: sessionID,
			})
			if oldNewMgr != nil {
				oldNewMgr(newManager)
			}
		}
	}

	// ============================================================
	// 6. (اختياري) ربط A2AManager بـ A2ANetworkBridge
	// ============================================================
	if cfg.EnableA2ABridge && cfg.A2AEventBus != nil {
		a2aBridge, err = cfg.Node.BridgeA2AToNetwork(ctx, sessionID, cfg.A2AEventBus)
		if err != nil {
			log.WithError(err).Warn("فشل إنشاء جسر A2A")
		}
	}

	// ============================================================
	// 7. ربط ChatConnector — ربط القنوات والشات بالجلسة
	// ============================================================
	if cfg.ChatConnector != nil {
		// إنشاء قناة الجلسة في ChatConnector
		channelID, err := cfg.ChatConnector.CreateSessionChannel(sessionID)
		if err != nil {
			log.WithError(err).Warn("فشل إنشاء قناة جلسة في ChatConnector")
		} else {
			log.WithField("channel_id", channelID).Info("تم إنشاء قناة جلسة في ChatConnector")
		}

		// ربط رسائل الشات الجديدة بـ ChatConnector
		if sc.ChatManager != nil {
			sc.EventBus.Subscribe("chat.message_added", func(evt eventbus.Event) {
				payloadJSON, err := json.Marshal(evt.Payload)
				if err != nil {
					return
				}
				var msg session.ChatMessage
				if err := json.Unmarshal(payloadJSON, &msg); err != nil {
					return
				}
				var prompt string
			if md, ok := msg.Metadata.(map[string]interface{}); ok {
				if p, ok := md["prompt"].(string); ok {
					prompt = p
				}
			}
			_ = cfg.ChatConnector.SendToSessionChannel(sessionID, msg.Source, msg.Content, prompt)
			})
		}
	}

	log.WithField("session_id", sessionID).Info("✅ تم ربط جميع مكونات الجلسة")
	return bridge, seshBridge, a2aBridge, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

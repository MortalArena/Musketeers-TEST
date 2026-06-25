package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/MortalArena/Musketeers/pkg/session"
)

// [WHY] WebSocketHandler يدير اتصالات WebSocket للعملاء
// [HOW] يربط العملاء بـ EventBus ويرسل لهم التحديثات الحية
// [SAFETY] يستخدم RWMutex لحماية خريطة العملاء
type WebSocketHandler struct {
	// المكونات الأساسية
	eventBus  *eventbus.EventBus
	container *session.SessionContainer

	// إدارة العملاء
	clients   map[string]*Client // [WHY] خريطة العمليل (client_id -> Client)
	clientsMu sync.RWMutex       // [SAFETY] لحماية خريطة العملاء

	// إعدادات WebSocket
	upgrader websocket.Upgrader // [WHY] لترقية HTTP إلى WebSocket

	// Lifecycle - دورة الحياة
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Logger
	logger *log.Logger
}

// [WHY] Client يمثل عميل WebSocket متصل
// [HOW] يحتوي على الاتصال والقناة والبيانات
type Client struct {
	ID         string            // [WHY] معرف العميل
	SessionID  string            // [WHY] معرف الجلسة
	Conn       *websocket.Conn   // [WHY] اتصال WebSocket
	Send       chan []byte       // [WHY] قناة للإرسال للعميل
	Handler    *WebSocketHandler // [WHY] المرجع للمعالج
	Subscribed bool              // [WHY] هل العميل مشترك في EventBus
}

// [WHY] NewWebSocketHandler ينشئ معالج WebSocket جديد
// [HOW] يهيئ المكونات وإعدادات WebSocket
// [SAFETY] يتحقق من أن eventBus و container ليسا nil
func NewWebSocketHandler(eventBus *eventbus.EventBus, container *session.SessionContainer, logger *log.Logger) *WebSocketHandler {
	if eventBus == nil {
		panic("eventBus cannot be nil") // [SAFETY] منع nil pointer
	}
	if container == nil {
		panic("container cannot be nil") // [SAFETY] منع nil pointer
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &WebSocketHandler{
		eventBus:  eventBus,
		container: container,
		clients:   make(map[string]*Client),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// [SAFETY] السماح بـ localhost فقط للأمان
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true // السماح بالاتصالات بدون Origin
				}

				// السماح بـ localhost فقط
				allowedOrigins := []string{
					"http://localhost",
					"http://localhost:8080",
					"http://localhost:3000",
					"http://127.0.0.1",
					"http://127.0.0.1:8080",
					"http://127.0.0.1:3000",
				}

				for _, allowed := range allowedOrigins {
					if origin == allowed {
						return true
					}
				}

				return false
			},
		},
		ctx:    ctx,
		cancel: cancel,
		logger: logger,
	}
}

// VerifyWebSocketDelegationCallback يُستدعى للتحقق من صحة اتصال WebSocket
// يرجع true إذا كان DID هو مالك الجلسة أو الـ token صحيح
func (wh *WebSocketHandler) VerifyWebSocketDelegationCallback(did string, token string) bool {
	if did == "" && token == "" {
		return false
	}
	// إذا تم تقديم DID، تحقق من أنه مالك الجلسة
	if did != "" {
		if err := wh.container.VerifyOwner(did); err == nil {
			return true
		}
	}
	// إذا تم تقديم token، تحقق منه
	// في النسخة الحالية، نقبل فقط token = "owner" كدليل بسيط
	// في المستقبل: تحقق بالتوقيع الرقمي
	if token == "owner" || token == wh.container.OwnerDID {
		return true
	}
	return false
}

// [WHY] Start يبدأ معالج WebSocket
// [HOW] يبدأ goroutine لتنظيف العملاء غير النشطين
// [SAFETY] يستخدم WaitGroup لانتظار goroutines
func (wh *WebSocketHandler) Start() error {
	wh.logger.Println("بدء WebSocketHandler")

	// [HOW] بدء goroutine لتنظيف العملاء غير النشطين
	wh.wg.Add(1)
	go wh.cleanupInactiveClients()

	wh.logger.Println("تم بدء WebSocketHandler بنجاح")
	return nil
}

// [WHY] Stop يوقف معالج WebSocket
// [HOW] يغلق جميع اتصالات العملاء ويوقف goroutines
// [SAFETY] يستخدم WaitGroup لانتظار goroutines
func (wh *WebSocketHandler) Stop() error {
	wh.logger.Println("إيقاف WebSocketHandler")

	// [HOW] إلغاء context
	wh.cancel()

	// [HOW] إغلاق جميع العملاء
	wh.clientsMu.Lock()
	for _, client := range wh.clients {
		client.Conn.Close()
		close(client.Send)
	}
	wh.clients = make(map[string]*Client)
	wh.clientsMu.Unlock()

	// [HOW] انتظار goroutines
	wh.wg.Wait()

	wh.logger.Println("تم إيقاف WebSocketHandler بنجاح")
	return nil
}

// [WHY] HandleWebSocket يعالج اتصالات WebSocket
// [HOW] يرقّب HTTP إلى WebSocket ويسجّل العميل بعد التحقق من الهوية
// [SAFETY] يتحقق من هوية المتصل قبل السماح بالاتصال
func (wh *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// [HOW] استخراج المعاملات
	sessionID := r.URL.Query().Get("session_id")
	clientID := r.URL.Query().Get("client_id")
	callerDID := r.URL.Query().Get("did")
	callerToken := r.URL.Query().Get("token")

	if sessionID == "" || clientID == "" {
		wh.logger.Println("معاملات session_id أو client_id مفقودة")
		http.Error(w, "معاملات session_id أو client_id مطلوبة", http.StatusBadRequest)
		return
	}

	// [SAFETY] التحقق من هوية المتصل
	if !wh.VerifyWebSocketDelegationCallback(callerDID, callerToken) {
		wh.logger.Printf("رفض اتصال WebSocket: client=%s لم يقدم هوية صالحة", clientID)
		http.Error(w, "هوية غير صالحة — يجب تقديم did أو token", http.StatusUnauthorized)
		return
	}

	wh.logger.Printf("تم التحقق من هوية العميل %s: did=%s", clientID, callerDID)

	// [HOW] ترقية HTTP إلى WebSocket
	conn, err := wh.upgrader.Upgrade(w, r, nil)
	if err != nil {
		wh.logger.Printf("فشل ترقية WebSocket: %v", err)
		return
	}

	// [HOW] إنشاء العميل
	client := &Client{
		ID:        clientID,
		SessionID: sessionID,
		Conn:      conn,
		Send:      make(chan []byte, 256), // [WHY] قناة مخزنة لمنع الحظر
		Handler:   wh,
	}

	// [HOW] تسجيل العميل
	wh.clientsMu.Lock()
	wh.clients[clientID] = client
	wh.clientsMu.Unlock()

	wh.logger.Printf("تم تسجيل عميل جديد: %s (session: %s)", clientID, sessionID)

	// [HOW] إرسال مصالحة الحالة فوراً
	wh.sendStateReconciliation(client)

	// [HOW] الاشتراك في EventBus
	wh.subscribeClient(client)

	// [HOW] بدء goroutines للقراءة والكتابة
	wh.wg.Add(2)
	go client.readPump()
	go client.writePump()
}

// [WHY] sendStateReconciliation يرسل مصالحة الحالة للعميل
// [HOW] يرسل الحالة الموحدة + آخر 50 رسالة + كل إدخالات السجل
// [SAFETY] لا يحظر لأنه يستخدم قناة Send المخزنة
func (wh *WebSocketHandler) sendStateReconciliation(client *Client) {
	state := wh.container.GetUnifiedState()

	stateMsg := map[string]interface{}{
		"type":  "state_reconciliation",
		"state": state,
	}

	// آخر 50 رسالة شات
	if wh.container.ChatManager != nil {
		messages := wh.container.ChatManager.GetLastMessages(50)
		stateMsg["messages"] = messages
	}

	// كل إدخالات السجل
	if wh.container.Journal != nil {
		entries := wh.container.Journal.All()
		stateMsg["journal"] = entries
		stateMsg["journal_count"] = len(entries)
	}

	data, err := json.Marshal(stateMsg)
	if err != nil {
		wh.logger.Printf("فشل تحويل مصالحة الحالة: %v", err)
		return
	}

	select {
	case client.Send <- data:
	default:
	}
}

// sendToClient ترسل رسالة JSON لعميل معين (مع فلترة SessionID)
func (wh *WebSocketHandler) sendToClient(client *Client, msgType string, payload interface{}) {
	data, err := json.Marshal(map[string]interface{}{
		"type":    msgType,
		"payload": payload,
	})
	if err != nil {
		return
	}
	select {
	case client.Send <- data:
	default:
	}
}

// [WHY] subscribeClient يشترك العميل في EventBus
// [HOW] يسجل معالجات لكل أنواع الأحداث المهمة
// [SAFETY] يفك القفل قبل النشر لمنع Deadlock
func (wh *WebSocketHandler) subscribeClient(client *Client) {
	// 1. تغييرات حالة الجلسة
	wh.eventBus.Subscribe("session.state.changed", func(event eventbus.Event) {
		if event.SessionID == client.SessionID {
			wh.sendToClient(client, "state_changed", event.Payload)
		}
	})

	// 2. رسائل الشات
	wh.eventBus.Subscribe("chat.message_added", func(event eventbus.Event) {
		if event.SessionID == client.SessionID {
			wh.sendToClient(client, "message_added", event.Payload)
		}
	})

	// 3. إدخالات السجل الجديدة (real-time history)
	wh.eventBus.Subscribe("session.journal.entry", func(event eventbus.Event) {
		if event.SessionID == client.SessionID {
			wh.sendToClient(client, "journal_entry", event.Payload)
		}
	})

	// 4. أحداث الوكلاء — تُنشر كـ session.agent_event من SessionEventBusBridge
	wh.eventBus.Subscribe("session.agent_event", func(event eventbus.Event) {
		if event.SessionID == client.SessionID {
			wh.sendToClient(client, "agent_event", event.Payload)
		}
	})

	// 5. انضمام/مغادرة مشاركين
	wh.eventBus.Subscribe("session.participant.offline", func(event eventbus.Event) {
		if event.SessionID == client.SessionID {
			wh.sendToClient(client, "participant_offline", event.Payload)
		}
	})
	wh.eventBus.Subscribe("session.joined", func(event eventbus.Event) {
		if event.SessionID == client.SessionID {
			wh.sendToClient(client, "participant_joined", event.Payload)
		}
	})

	// 6. تغيير مدير الجلسة (failover)
	wh.eventBus.Subscribe("session.manager.changed", func(event eventbus.Event) {
		if event.SessionID == client.SessionID {
			wh.sendToClient(client, "manager_changed", event.Payload)
		}
	})

	// 7. رسائل القنوات (channel messages)
	wh.eventBus.Subscribe("chat.message", func(event eventbus.Event) {
		if event.SessionID == client.SessionID {
			wh.sendToClient(client, "channel_message", event.Payload)
		}
	})

	client.Subscribed = true
	wh.logger.Printf("تم اشتراك العميل %s في EventBus (8 أنواع أحداث)", client.ID)
}

// [WHY] unsubscribeClient يلغي اشتراك العميل من EventBus
// [HOW] يزيل المعالجات ويغلق القناة
// [SAFETY] يستخدم Lock لحماية خريطة العملاء
func (wh *WebSocketHandler) unsubscribeClient(client *Client) {
	// إلغاء الاشتراك من EventBus
	if client.Subscribed {
		wh.eventBus.Unsubscribe(client.ID)
		client.Subscribed = false
	}

	// [HOW] إغلاق القناة
	close(client.Send)

	// [HOW] إزالة العميل من الخريطة
	wh.clientsMu.Lock()
	delete(wh.clients, client.ID)
	wh.clientsMu.Unlock()

	wh.logger.Printf("تم إلغاء اشتراك العميل %s", client.ID)
}

// [WHY] cleanupInactiveClients ينظف العملاء غير النشطين
// [HOW] يفحص كل 30 ثانية ويزيل العملاء المنقطعين
// [SAFETY] يستخدم Ticker لتنفيذ دوري
func (wh *WebSocketHandler) cleanupInactiveClients() {
	defer wh.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-wh.ctx.Done():
			return
		case <-ticker.C:
			// [HOW] فحص جميع العملاء
			wh.clientsMu.Lock()
			for clientID, client := range wh.clients {
				// [HOW] إرسال Ping
				err := client.Conn.WriteMessage(websocket.PingMessage, nil)
				if err != nil {
					// [SAFETY] العميل منقطع، إزالته
					wh.logger.Printf("العميل %s منقطع، إزالته", clientID)
					client.Conn.Close()
					close(client.Send)
					delete(wh.clients, clientID)
				}
			}
			wh.clientsMu.Unlock()
		}
	}
}

// [WHY] readPump يقرأ الرسائل من العميل
// [HOW] يقرأ باستمرار من WebSocket ويعالج الرسائل
// [SAFETY] يستخدم defer لإغلاق الاتصال
func (c *Client) readPump() {
	defer c.Handler.wg.Done()
	defer c.Conn.Close()
	defer c.Handler.unsubscribeClient(c)

	// [HOW] تعيين مهلة القراءة
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		// [HOW] قراءة رسالة
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Handler.logger.Printf("خطأ في قراءة WebSocket: %v", err)
			}
			break
		}

		// [HOW] معالجة الرسالة من العميل
		c.Handler.logger.Printf("رسالة من العميل %s: %s", c.ID, string(message))

		// [HOW] تحليل الرسالة وإرسالها إلى EventBus
		var clientMsg map[string]interface{}
		if err := json.Unmarshal(message, &clientMsg); err == nil {
			// [HOW] إرسال الرسالة إلى EventBus
			c.Handler.eventBus.Publish(eventbus.Event{
				Type:      "client.message",
				Payload:   clientMsg,
				Source:    c.ID,
				SessionID: c.SessionID,
				Timestamp: time.Now(),
			})
		}
	}
}

// [WHY] writePump يكتب الرسائل للعميل
// [HOW] يقرأ من قناة Send ويكتب إلى WebSocket
// [SAFETY] يستخدم defer لإغلاق الاتصال
func (c *Client) writePump() {
	defer c.Handler.wg.Done()
	defer c.Conn.Close()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-c.Send:
			// [HOW] كتابة الرسالة
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// [SAFETY] القناة مغلقة
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// [HOW] إرسال الرسالة
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

			// [HOW] إرسال الرسائل المتراكمة
			n := len(c.Send)
			for i := 0; i < n; i++ {
				if err := c.Conn.WriteMessage(websocket.TextMessage, <-c.Send); err != nil {
					return
				}
			}

		case <-ticker.C:
			// [HOW] إرسال Ping
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

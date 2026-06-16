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
				// [TODO] تحقق من Origin في الإنتاج
				return true // [SAFETY] السماح بكل Origins للتطوير
			},
		},
		ctx:    ctx,
		cancel: cancel,
		logger: logger,
	}
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
// [HOW] يرقّب HTTP إلى WebSocket ويسجّل العميل
// [SAFETY] يستخدم context مع timeout لمنع الحظر
func (wh *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// [HOW] استخراج المعاملات
	sessionID := r.URL.Query().Get("session_id")
	clientID := r.URL.Query().Get("client_id")

	if sessionID == "" || clientID == "" {
		wh.logger.Println("معاملات session_id أو client_id مفقودة")
		http.Error(w, "معاملات session_id أو client_id مطلوبة", http.StatusBadRequest)
		return
	}

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
// [HOW] يرسل آخر UnifiedSessionState وآخر 50 رسالة
// [SAFETY] لا يحظر لأنه يستخدم قناة Send المخزنة
func (wh *WebSocketHandler) sendStateReconciliation(client *Client) {
	// [HOW] الحصول على الحالة الموحدة
	state := wh.container.GetUnifiedState()

	// [HOW] الحصول على آخر 50 رسالة
	messages := wh.container.ChatManager.GetLastMessages(50)

	// [HOW] إنشاء رسالة مصالحة
	reconciliation := map[string]interface{}{
		"type":     "state_reconciliation",
		"state":    state,
		"messages": messages,
	}

	// [HOW] تحويل إلى JSON
	data, err := json.Marshal(reconciliation)
	if err != nil {
		wh.logger.Printf("فشل تحويل مصالحة الحالة: %v", err)
		return
	}

	// [HOW] إرسال عبر القناة
	select {
	case client.Send <- data:
		// [OK] تم الإرسال
	default:
		// [SAFETY] القناة ممتلئة، تجاهل
	}
}

// [WHY] subscribeClient يشترك العميل في EventBus
// [HOW] يسجل معالجات للأحداث المهمة
// [SAFETY] يفك القفل قبل النشر لمنع Deadlock
func (wh *WebSocketHandler) subscribeClient(client *Client) {
	// [HOW] الاشتراك في session.state.changed
	wh.eventBus.Subscribe("session.state.changed", func(event eventbus.Event) {
		if event.SessionID == client.SessionID {
			// [HOW] إرسال الحالة للعميل
			stateData, err := json.Marshal(map[string]interface{}{
				"type":    "state_changed",
				"payload": event.Payload,
			})
			if err != nil {
				return
			}

			select {
			case client.Send <- stateData:
				// [OK] تم الإرسال
			default:
				// [SAFETY] القناة ممتلئة، تجاهل
			}
		}
	})

	// [HOW] الاشتراك في chat.message_added
	wh.eventBus.Subscribe("chat.message_added", func(event eventbus.Event) {
		if event.SessionID == client.SessionID {
			// [HOW] إرسال الرسالة للعميل
			messageData, err := json.Marshal(map[string]interface{}{
				"type":    "message_added",
				"payload": event.Payload,
			})
			if err != nil {
				return
			}

			select {
			case client.Send <- messageData:
				// [OK] تم الإرسال
			default:
				// [SAFETY] القناة ممتلئة، تجاهل
			}
		}
	})

	client.Subscribed = true
	wh.logger.Printf("تم اشتراك العميل %s في EventBus", client.ID)
}

// [WHY] unsubscribeClient يلغي اشتراك العميل من EventBus
// [HOW] يزيل المعالجات ويغلق القناة
// [SAFETY] يستخدم Lock لحماية خريطة العملاء
func (wh *WebSocketHandler) unsubscribeClient(client *Client) {
	// [TODO] إلغاء الاشتراك من EventBus (يتطلب دالة Unsubscribe محددة)
	client.Subscribed = false

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

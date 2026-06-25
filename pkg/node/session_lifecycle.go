package node

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/sirupsen/logrus"
)

// ============================================================
// Session Lifecycle Management — Join, Heartbeat, Election, Failover
// ============================================================

// CtrlMsgType نوع رسالة التحكم في الجلسة
type CtrlMsgType string

const (
	// الانضمام والمغادرة
	CtrlJoinRequest  CtrlMsgType = "join_req"
	CtrlJoinResponse CtrlMsgType = "join_resp"
	CtrlLeave        CtrlMsgType = "leave"
	// ضربات القلب
	CtrlHeartbeat CtrlMsgType = "hb"
	// انتخاب القائد
	CtrlElectStart    CtrlMsgType = "elect_start"
	CtrlElectAnnounce CtrlMsgType = "elect_announce"
	CtrlNewManager    CtrlMsgType = "new_mgr"
	// استعادة
	CtrlStateSnapshot CtrlMsgType = "state_snap"
	CtrlTaskReassign  CtrlMsgType = "task_reass"
	// حالة الوكيل
	CtrlAgentStatus CtrlMsgType = "agent_st"
)

// ParticipantRole دور المشارك في الجلسة
type ParticipantRole string

const (
	RoleManager   ParticipantRole = "manager"
	RoleBackup    ParticipantRole = "backup"
	RoleAssistant ParticipantRole = "assistant"
	RoleHuman     ParticipantRole = "human"
)

// SessionCtrlMsg رسالة تحكم الجلسة
type SessionCtrlMsg struct {
	Type       CtrlMsgType `json:"type"`
	SessionID  string      `json:"sid"`
	NodeID     string      `json:"nid"`
	DID        string      `json:"did"`
	Role       string      `json:"role"`
	DeviceName string      `json:"dev"`
	AgentID    string      `json:"aid,omitempty"`
	Timestamp  int64       `json:"ts"`
	// Join
	DelegationToken string `json:"dtok,omitempty"`
	HumanUserID     string `json:"huid,omitempty"`
	HumanName       string `json:"hname,omitempty"`
	// Election
	BackupPriority int    `json:"bpri,omitempty"`
	DelegationSig  string `json:"dsig,omitempty"`
	// State
	StatePayload string `json:"sp,omitempty"`
	TaskMapping  string `json:"tm,omitempty"`
	// Heartbeat
	AliveAgents []string `json:"aa,omitempty"`
	ActiveTasks int      `json:"at,omitempty"`
}

// ParticipantInfo معلومات مشارك في الجلسة
type ParticipantInfo struct {
	NodeID      string
	DID         string
	Role        ParticipantRole
	DeviceName  string
	AgentID     string
	LastSeen    time.Time
	IsOnline    bool
	BackupOrder int // ترتيب الاحتياط (0 = ليس احتياطياً)
}

// SessionLifecycleCallback استدعاءات لربط lifecycle مع الجلسة الفعلية
type SessionLifecycleCallback struct {
	// OnJoin يُستدعى عند انضمام مشارك جديد
	OnJoin func(participant ParticipantInfo, stateJSON []byte) error
	// OnLeave يُستدعى عند مغادرة مشارك
	OnLeave func(nodeID string)
	// OnNewManager يُستدعى عند تغيير المدير
	OnNewManager func(newManagerNodeID string)
	// OnStateRequest يُستدعى لطلب حالة الجلسة الحالية (للتصدير)
	OnStateRequest func() ([]byte, error)
	// OnStateReceived يُستدعى عند استقبال حالة من مشارك آخر
	OnStateReceived func(data []byte) error
	// OnTaskReassign يُستدعى عند إعادة توزيع المهام
	OnTaskReassign func(mappingJSON string) error
	// Electable يتحقق مما إذا كان هذا الجهاز مؤهلاً للانتخاب
	Electable func(backupPriority int) bool
	// OnElectionComplete يُستدعى عندما ينتهي الانتخاب دون فائز
	// يُعطي قائمة المرشحين للعميل البشري ليختار المدير الجديد
	OnElectionComplete func(candidates []string)
	// VerifyDelegation يُستدعى للتحقق من تفويض
	// يرجع true إذا كان التفويض (token) يسمح بالعملية (action) للعقدة (nodeDID)
	VerifyDelegation func(delegationToken string, action string, performerDID string) bool
}

// SessionLifecycleManager يدير دورة حياة الجلسة عبر الشبكة
type SessionLifecycleManager struct {
	node      *Node
	sessionID string
	localBus  *eventbus.EventBus
	callbacks SessionLifecycleCallback

	// PubSub
	topic  *pubsub.Topic
	sub    *pubsub.Subscription
	ctx    context.Context
	cancel context.CancelFunc

	// المشاركون
	mu           sync.RWMutex
	participants map[string]*ParticipantInfo
	myRole       ParticipantRole
	myNodeID     string
	myDID        string
	managerNode  string // Node ID of current manager

	// Heartbeat
	hbTicker      *time.Ticker
	hbCheckTicker *time.Ticker
	lastHB        map[string]time.Time

	// Election
	inElection         bool
	electionDeadline   time.Time
	electionMu         sync.Mutex
	electionCandidates []string // المرشحون في الانتخاب الحالي

	// Failover
	backupManagers []BackupEntry
	isNewManager   bool

	wg     sync.WaitGroup
	log    *logrus.Logger
	closed bool
}

// BackupEntry إدخال وكيل احتياطي
type BackupEntry struct {
	NodeID        string
	DID           string
	AgentID       string
	Priority      int
	DelegationDID string // DID of who delegated backup authority
}

const (
	heartbeatInterval = 3 * time.Second
	heartbeatTimeout  = 12 * time.Second
	electionBaseDelay = 1 * time.Second
	electionMaxDelay  = 5 * time.Second
	electionWaitTime  = 8 * time.Second
	hbCheckInterval   = 4 * time.Second
)

// NewSessionLifecycleManager ينشئ مدير دورة حياة لجلسة
func NewSessionLifecycleManager(
	node *Node,
	sessionID string,
	localBus *eventbus.EventBus,
	myRole ParticipantRole,
	callbacks SessionLifecycleCallback,
) (*SessionLifecycleManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	topicName := fmt.Sprintf("/mskt/session/%s/ctrl", sessionID)
	topic, err := node.ps().Join(topicName)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("join ctrl topic: %w", err)
	}

	sub, err := topic.Subscribe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("subscribe ctrl: %w", err)
	}

	kp := node.keyPair()
	mgr := &SessionLifecycleManager{
		node:         node,
		sessionID:    sessionID,
		localBus:     localBus,
		callbacks:    callbacks,
		topic:        topic,
		sub:          sub,
		ctx:          ctx,
		cancel:       cancel,
		participants: make(map[string]*ParticipantInfo),
		myRole:       myRole,
		myNodeID:     node.host().ID().String(),
		myDID:        kp.DID,
		lastHB:       make(map[string]time.Time),
		log:          node.log,
	}

	mgr.wg.Add(1)
	go mgr.receiveCtrlMessages()

	// Heartbeat sender (managers and backups always send)
	if myRole == RoleManager || myRole == RoleBackup {
		mgr.hbTicker = time.NewTicker(heartbeatInterval)
		mgr.wg.Add(1)
		go mgr.sendHeartbeats()
	}

	// Heartbeat checker (all participants)
	mgr.hbCheckTicker = time.NewTicker(hbCheckInterval)
	mgr.wg.Add(1)
	go mgr.checkHeartbeats()

	// Register self
	mgr.mu.Lock()
	mgr.participants[mgr.myNodeID] = &ParticipantInfo{
		NodeID:   mgr.myNodeID,
		DID:      mgr.myDID,
		Role:     myRole,
		LastSeen: time.Now(),
		IsOnline: true,
	}
	if myRole == RoleManager {
		mgr.managerNode = mgr.myNodeID
	}
	mgr.mu.Unlock()

	node.log.WithField("session_id", sessionID).Info("مدير دورة حياة الجلسة نشط")
	return mgr, nil
}

// Close يغلق المدير
func (lm *SessionLifecycleManager) Close() {
	lm.mu.Lock()
	if lm.closed {
		lm.mu.Unlock()
		return
	}
	lm.closed = true
	lm.mu.Unlock()

	// إرسال رسالة مغادرة
	lm.sendCtrlMsg(CtrlLeave, map[string]interface{}{
		"role": string(lm.myRole),
	})

	if lm.hbTicker != nil {
		lm.hbTicker.Stop()
	}
	lm.hbCheckTicker.Stop()
	lm.cancel()
	lm.wg.Wait()
}

// ============================================================
// Join Protocol
// ============================================================

// RequestJoin يرسل طلب انضمام للجلسة
func (lm *SessionLifecycleManager) RequestJoin(ctx context.Context, delegationToken, humanUserID, humanName, agentID string) error {
	return lm.sendCtrlMsg(CtrlJoinRequest, map[string]interface{}{
		"dtok":  delegationToken,
		"huid":  humanUserID,
		"hname": humanName,
		"aid":   agentID,
	})
}

// HandleJoinRequest يعالج طلب انضمام (يُستدعى من OnJoin callback)
func (lm *SessionLifecycleManager) HandleJoinRequest(req SessionCtrlMsg) error {
	lm.mu.RLock()
	managerNode := lm.managerNode
	lm.mu.RUnlock()

	if lm.myNodeID != managerNode {
		return nil // فقط مدير الجلسة يرد على طلبات الانضمام
	}

	// طلب حالة الجلسة
	if lm.callbacks.OnStateRequest == nil {
		return nil
	}
	stateJSON, err := lm.callbacks.OnStateRequest()
	if err != nil {
		return err
	}

	return lm.sendCtrlMsg(CtrlJoinResponse, map[string]interface{}{
		"sp":    string(stateJSON),
		"huid":  req.HumanUserID,
		"hname": req.HumanName,
	})
}

// handleJoinResponse يعالج رد الانضمام ويحمّل حالة الجلسة
func (lm *SessionLifecycleManager) handleJoinResponse(msg SessionCtrlMsg) error {
	if msg.StatePayload == "" || lm.callbacks.OnStateReceived == nil {
		return nil
	}

	// إضافة المشارك الذي أرسل الرد (المدير)
	lm.mu.Lock()
	if _, exists := lm.participants[msg.NodeID]; !exists {
		lm.participants[msg.NodeID] = &ParticipantInfo{
			NodeID:     msg.NodeID,
			DID:        msg.DID,
			Role:       RoleManager,
			DeviceName: msg.DeviceName,
			LastSeen:   time.Now(),
			IsOnline:   true,
		}
		lm.managerNode = msg.NodeID
	}
	lm.mu.Unlock()

	// استيراد حالة الجلسة
	if err := lm.callbacks.OnStateReceived([]byte(msg.StatePayload)); err != nil {
		return err
	}

	// إعلام المحليين
	lm.localBus.Publish(eventbus.Event{
		Type:      "session.joined",
		Source:    lm.myNodeID,
		SessionID: lm.sessionID,
	})

	lm.log.WithField("session_id", lm.sessionID).Info("انضممت للجلسة بنجاح")
	return nil
}

// ============================================================
// Heartbeat
// ============================================================

func (lm *SessionLifecycleManager) sendHeartbeats() {
	defer lm.wg.Done()
	for {
		select {
		case <-lm.ctx.Done():
			return
		case <-lm.hbTicker.C:
			lm.sendCtrlMsg(CtrlHeartbeat, map[string]interface{}{
				"role": string(lm.myRole),
			})
		}
	}
}

func (lm *SessionLifecycleManager) checkHeartbeats() {
	defer lm.wg.Done()
	for {
		select {
		case <-lm.ctx.Done():
			return
		case <-lm.hbCheckTicker.C:
			lm.detectStaleParticipants()
		}
	}
}

func (lm *SessionLifecycleManager) detectStaleParticipants() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	now := time.Now()
	var staleNodes []string

	for nodeID, p := range lm.participants {
		if nodeID == lm.myNodeID {
			continue
		}
		if !p.IsOnline {
			continue
		}
		if now.Sub(p.LastSeen) > heartbeatTimeout {
			staleNodes = append(staleNodes, nodeID)
		}
	}

	if len(staleNodes) == 0 {
		return
	}

	for _, nodeID := range staleNodes {
		p := lm.participants[nodeID]
		p.IsOnline = false
		lm.log.WithFields(logrus.Fields{
			"session_id": lm.sessionID,
			"node_id":    nodeID,
			"role":       string(p.Role),
		}).Warn("مشارك غير متصل — تجاوز مهلة ضربات القلب")

		if lm.callbacks.OnLeave != nil {
			go lm.callbacks.OnLeave(nodeID)
		}

		lm.localBus.Publish(eventbus.Event{
			Type:      "session.participant.offline",
			Source:    nodeID,
			SessionID: lm.sessionID,
			Payload:   map[string]string{"node_id": nodeID, "role": string(p.Role)},
		})
	}

	// إذا كان المدير غير متصل → ابدأ الانتخاب
	if lm.managerNode != "" {
		if p, exists := lm.participants[lm.managerNode]; exists && !p.IsOnline {
			go lm.startElection()
		}
	}
}

// ============================================================
// Leader Election
// ============================================================

func (lm *SessionLifecycleManager) startElection() {
	lm.electionMu.Lock()
	if lm.inElection {
		lm.electionMu.Unlock()
		return
	}
	lm.inElection = true
	lm.electionDeadline = time.Now().Add(electionWaitTime)
	lm.electionCandidates = nil
	lm.electionMu.Unlock()

	// الإعلان عن بداية الانتخاب
	_ = lm.sendCtrlMsg(CtrlElectStart, nil)
	lm.log.WithField("session_id", lm.sessionID).Warn("بدء انتخاب مدير جلسة جديد")

	// البحث عن نفسي في قائمة الاحتياط
	myPriority := 0
	lm.mu.RLock()
	for _, b := range lm.backupManagers {
		if b.NodeID == lm.myNodeID {
			myPriority = b.Priority
			break
		}
	}
	lm.mu.RUnlock()

	// إذا لم أكن احتياطياً → أنتظر فقط
	if myPriority == 0 && lm.myRole != RoleBackup {
		lm.waitForElectionResult()
		return
	}

	// إذا كنت احتياطياً → انتظر randomised backoff حسب الأولوية
	delay := electionBaseDelay + time.Duration(float64(electionMaxDelay-electionBaseDelay)*rand.Float64())
	// الأولوية الأعلى (رقم أقل) تنتظر أقل
	priorityFactor := time.Duration(myPriority) * 500 * time.Millisecond
	if priorityFactor > delay {
		delay = priorityFactor
	}

	lm.log.WithFields(logrus.Fields{
		"session_id": lm.sessionID,
		"priority":   myPriority,
		"delay":      delay,
	}).Info("انتظار قبل الترشح للانتخاب")

	select {
	case <-lm.ctx.Done():
		lm.electionMu.Lock()
		lm.inElection = false
		lm.electionMu.Unlock()
		return
	case <-time.After(delay):
	}

	// التحقق: هل انتهى الانتخاب بالفعل (مدير جديد أُعلن)؟
	lm.electionMu.Lock()
	if !lm.inElection {
		lm.electionMu.Unlock()
		return
	}
	lm.electionMu.Unlock()

	// الترشح
	if lm.callbacks.Electable != nil && !lm.callbacks.Electable(myPriority) {
		lm.waitForElectionResult()
		return
	}

	// إعلان الترشح
	_ = lm.sendCtrlMsg(CtrlElectAnnounce, map[string]interface{}{
		"bpri": myPriority,
	})
	lm.log.WithField("session_id", lm.sessionID).Warn("ترشحت لقيادة الجلسة")
}

func (lm *SessionLifecycleManager) waitForElectionResult() {
	// انتظر حتى deadline الانتخاب
	deadline := time.Now().Add(electionWaitTime)
	select {
	case <-lm.ctx.Done():
		return
	case <-time.After(time.Until(deadline)):
	}

	lm.electionMu.Lock()
	if !lm.inElection {
		lm.electionMu.Unlock()
		return
	}
	lm.electionMu.Unlock()

	// لم يصلنا إعلان مدير جديد → نطلب من العميل البشري اختيار مدير
	lm.electionMu.Lock()
	candidates := make([]string, len(lm.electionCandidates))
	copy(candidates, lm.electionCandidates)
	lm.inElection = false
	lm.electionMu.Unlock()

	if lm.callbacks.OnElectionComplete != nil {
		go lm.callbacks.OnElectionComplete(candidates)
	}

	lm.log.WithFields(logrus.Fields{
		"session_id": lm.sessionID,
		"candidates": candidates,
	}).Warn("انتخاب فشل — في انتظار اختيار العميل البشري للمدير الجديد")
}

// announceAsNewManager يعلن عن نفسي كمدير جديد
func (lm *SessionLifecycleManager) announceAsNewManager(delegationToken string) {
	lm.mu.Lock()
	lm.myRole = RoleManager
	lm.managerNode = lm.myNodeID
	lm.isNewManager = true
	lm.mu.Unlock()

	extras := map[string]interface{}{
		"old_manager": lm.managerNode,
	}
	if delegationToken != "" {
		extras["dtok"] = delegationToken
	}
	_ = lm.sendCtrlMsg(CtrlNewManager, extras)

	// بدء إرسال heartbeats إذا لم يكن قد بدأ
	if lm.hbTicker == nil {
		lm.hbTicker = time.NewTicker(heartbeatInterval)
		lm.wg.Add(1)
		go lm.sendHeartbeats()
	}

	lm.log.WithField("session_id", lm.sessionID).Warn("أصبحت مدير الجلسة الجديد")
}

// AuthorizeNewManager يُستدعى من العميل البشري لتفويض مدير جلسة جديد
// token هو التفويض الموقّع من مالك الجلسة (البشر)
// nodeID هو معرف العقدة المفوّضة (قد تكون هذه العقدة أو غيرها)
func (lm *SessionLifecycleManager) AuthorizeNewManager(nodeID string, delegationToken string) {
	lm.mu.Lock()
	lm.mu.Unlock()

	if nodeID == lm.myNodeID {
		// أنا المفوّض — أعلن نفسي
		lm.announceAsNewManager(delegationToken)
		return
	}

	// عقدة أخرى مفوّضة — أرسل لها الأمر
	_ = lm.sendCtrlMsg(CtrlNewManager, map[string]interface{}{
		"target": nodeID,
		"dtok":   delegationToken,
	})

	lm.log.WithFields(logrus.Fields{
		"session_id": lm.sessionID,
		"target":     nodeID,
	}).Warn("تم تفويض عقدة أخرى كمدير جلسة")
}

// handleElectAnnounce يعالج إعلان ترشح
func (lm *SessionLifecycleManager) handleElectAnnounce(msg SessionCtrlMsg) {
	lm.electionMu.Lock()
	defer lm.electionMu.Unlock()
	if !lm.inElection {
		return
	}

	// تسجيل المرشح في القائمة
	for _, c := range lm.electionCandidates {
		if c == msg.NodeID {
			return // مكرر
		}
	}
	lm.electionCandidates = append(lm.electionCandidates, msg.NodeID)

	lm.localBus.Publish(eventbus.Event{
		Type:      "session.election_candidate",
		Source:    msg.NodeID,
		SessionID: lm.sessionID,
	})

	lm.log.WithFields(logrus.Fields{
		"session_id": lm.sessionID,
		"candidate":  msg.NodeID,
		"total":      len(lm.electionCandidates),
	}).Info("تسجيل مرشح لقيادة الجلسة")
}

// ============================================================
// Failover
// ============================================================

// SetBackupManagers يضبط قائمة وكلاء الاحتياط
func (lm *SessionLifecycleManager) SetBackupManagers(backups []BackupEntry) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.backupManagers = backups

	// ترتيب حسب الأولوية
	sort.Slice(lm.backupManagers, func(i, j int) bool {
		return lm.backupManagers[i].Priority < lm.backupManagers[j].Priority
	})
}

// RequestStateSnapshot يطلب لقطة حالة من المدير الحالي
func (lm *SessionLifecycleManager) RequestStateSnapshot(ctx context.Context) ([]byte, error) {
	// إرسال طلب لقطة حالة
	if err := lm.sendCtrlMsg(CtrlStateSnapshot, nil); err != nil {
		return nil, err
	}

	// انتظار الرد (في التطبيق الحقيقي، هذا يحتاج قناة رد)
	// هنا نستخدم callback OnStateReceived
	lm.log.WithField("session_id", lm.sessionID).Info("طلب لقطة حالة")
	return nil, nil
}

// ReassignTasks يعيد توزيع مهام الوكيل المنقطع
func (lm *SessionLifecycleManager) ReassignTasks(failedNodeID string, taskMappingJSON string) error {
	lm.mu.RLock()
	isManager := lm.myRole == RoleManager
	lm.mu.RUnlock()

	if !isManager {
		return nil // فقط المدير يعيد التوزيع
	}

	if err := lm.sendCtrlMsg(CtrlTaskReassign, map[string]interface{}{
		"failed_node": failedNodeID,
		"tm":          taskMappingJSON,
	}); err != nil {
		return err
	}

	if lm.callbacks.OnTaskReassign != nil {
		return lm.callbacks.OnTaskReassign(taskMappingJSON)
	}

	return nil
}

// ============================================================
// رسائل التحكم (PubSub)
// ============================================================

func (lm *SessionLifecycleManager) receiveCtrlMessages() {
	defer lm.wg.Done()
	for {
		msg, err := lm.sub.Next(lm.ctx)
		if err != nil {
			if lm.ctx.Err() == nil {
				lm.log.WithField("session_id", lm.sessionID).WithError(err).Warn("خطأ في استقبال رسائل التحكم")
			}
			return
		}

		// تجاهل رسائلي
		if msg.GetFrom() == lm.node.host().ID() {
			continue
		}

		var ctrlMsg SessionCtrlMsg
		if err := json.Unmarshal(msg.Data, &ctrlMsg); err != nil {
			continue
		}

		if ctrlMsg.SessionID != lm.sessionID {
			continue
		}

		lm.handleCtrlMessage(ctrlMsg)
	}
}

func (lm *SessionLifecycleManager) handleCtrlMessage(msg SessionCtrlMsg) {
	// تحديث وقت آخر ظهور للمشارك
	lm.mu.Lock()
	if p, exists := lm.participants[msg.NodeID]; exists {
		p.LastSeen = time.Now()
		p.IsOnline = true
	} else {
		lm.participants[msg.NodeID] = &ParticipantInfo{
			NodeID:     msg.NodeID,
			DID:        msg.DID,
			Role:       ParticipantRole(msg.Role),
			DeviceName: msg.DeviceName,
			AgentID:    msg.AgentID,
			LastSeen:   time.Now(),
			IsOnline:   true,
		}

		// مشارك جديد — إعلام
		if lm.callbacks.OnJoin != nil {
			go lm.callbacks.OnJoin(*lm.participants[msg.NodeID], nil)
		}
	}
	lm.mu.Unlock()

	switch msg.Type {
	case CtrlJoinRequest:
		_ = lm.HandleJoinRequest(msg)

	case CtrlJoinResponse:
		_ = lm.handleJoinResponse(msg)

	case CtrlHeartbeat:
		// تم التحديث بالفعل أعلاه

	case CtrlLeave:
		lm.mu.Lock()
		lm.participants[msg.NodeID] = &ParticipantInfo{
			NodeID:   msg.NodeID,
			IsOnline: false,
			LastSeen: time.Now(),
		}
		lm.mu.Unlock()
		if lm.callbacks.OnLeave != nil {
			go lm.callbacks.OnLeave(msg.NodeID)
		}

	case CtrlElectStart:
		lm.log.WithField("session_id", lm.sessionID).Info("بدء انتخاب مدير جلسة جديد (مستلم)")

	case CtrlElectAnnounce:
		lm.handleElectAnnounce(msg)

	case CtrlNewManager:
		// قبول فقط إذا كان التفويض صحيحاً أو نحن من أرسلناه
		if msg.NodeID == lm.myNodeID {
			// نحن من أرسلنا — نعرف أنه صحيح
		} else if lm.callbacks.VerifyDelegation != nil && msg.DelegationToken != "" {
			if !lm.callbacks.VerifyDelegation(msg.DelegationToken, "session.manager", msg.DID) {
				lm.log.WithField("session_id", lm.sessionID).Warn("رفض CtrlNewManager: تفويض غير صحيح")
				break
			}
		} else {
			lm.log.WithField("session_id", lm.sessionID).Warn("رفض CtrlNewManager: لا يوجد تفويض")
			break
		}
		lm.mu.Lock()
		lm.managerNode = msg.NodeID
		if p, exists := lm.participants[msg.NodeID]; exists {
			p.Role = RoleManager
		}
		lm.mu.Unlock()
		if lm.callbacks.OnNewManager != nil {
			go lm.callbacks.OnNewManager(msg.NodeID)
		}

	case CtrlStateSnapshot:
		// طلب لقطة حالة — المدير فقط يرد
		if lm.callbacks.OnStateRequest != nil && lm.myRole == RoleManager {
			stateJSON, err := lm.callbacks.OnStateRequest()
			if err == nil {
				_ = lm.sendCtrlMsg(CtrlJoinResponse, map[string]interface{}{
					"sp": string(stateJSON),
				})
			}
		}

	case CtrlTaskReassign:
		if lm.callbacks.OnTaskReassign != nil && msg.TaskMapping != "" {
			_ = lm.callbacks.OnTaskReassign(msg.TaskMapping)
		}

	case CtrlAgentStatus:
		lm.mu.Lock()
		if p, exists := lm.participants[msg.NodeID]; exists {
			p.AgentID = msg.AgentID
		}
		lm.mu.Unlock()
	}
}

func (lm *SessionLifecycleManager) sendCtrlMsg(msgType CtrlMsgType, extra map[string]interface{}) error {
	msg := SessionCtrlMsg{
		Type:      msgType,
		SessionID: lm.sessionID,
		NodeID:    lm.myNodeID,
		DID:       lm.myDID,
		Role:      string(lm.myRole),
		Timestamp: time.Now().Unix(),
	}

	// دمج extra fields
	if extra != nil {
		if v, ok := extra["dtok"]; ok {
			msg.DelegationToken, _ = v.(string)
		}
		if v, ok := extra["huid"]; ok {
			msg.HumanUserID, _ = v.(string)
		}
		if v, ok := extra["hname"]; ok {
			msg.HumanName, _ = v.(string)
		}
		if v, ok := extra["aid"]; ok {
			msg.AgentID, _ = v.(string)
		}
		if v, ok := extra["bpri"]; ok {
			msg.BackupPriority, _ = v.(int)
		}
		if v, ok := extra["dsig"]; ok {
			msg.DelegationSig, _ = v.(string)
		}
		if v, ok := extra["sp"]; ok {
			msg.StatePayload, _ = v.(string)
		}
		if v, ok := extra["tm"]; ok {
			msg.TaskMapping, _ = v.(string)
		}
		if v, ok := extra["role"]; ok {
			msg.Role, _ = v.(string)
		}
		if v, ok := extra["dev"]; ok {
			msg.DeviceName, _ = v.(string)
		}
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return lm.topic.Publish(lm.ctx, data)
}

// ============================================================
// Queries
// ============================================================

// GetParticipants يرجع قائمة المشاركين النشطين
func (lm *SessionLifecycleManager) GetParticipants() []ParticipantInfo {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	result := make([]ParticipantInfo, 0, len(lm.participants))
	for _, p := range lm.participants {
		if p.IsOnline || time.Since(p.LastSeen) < heartbeatTimeout {
			result = append(result, *p)
		}
	}
	return result
}

// GetOnlineParticipants يرجع المشاركين المتصلين فقط
func (lm *SessionLifecycleManager) GetOnlineParticipants() []ParticipantInfo {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	result := make([]ParticipantInfo, 0)
	for _, p := range lm.participants {
		if p.IsOnline {
			result = append(result, *p)
		}
	}
	return result
}

// GetManagerNode يرجع Node ID للمدير الحالي
func (lm *SessionLifecycleManager) GetManagerNode() string {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.managerNode
}

// GetMyRole يرجع دوري في الجلسة
func (lm *SessionLifecycleManager) GetMyRole() ParticipantRole {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.myRole
}

// IsManager يتحقق مما إذا كنت المدير
func (lm *SessionLifecycleManager) IsManager() bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.myNodeID == lm.managerNode
}

// Stats يرجع إحصائيات
func (lm *SessionLifecycleManager) Stats() map[string]interface{} {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	online := 0
	offline := 0
	for _, p := range lm.participants {
		if p.IsOnline {
			online++
		} else {
			offline++
		}
	}

	return map[string]interface{}{
		"session_id":     lm.sessionID,
		"my_role":        string(lm.myRole),
		"manager_node":   lm.managerNode,
		"participants":   len(lm.participants),
		"online":         online,
		"offline":        offline,
		"backup_count":   len(lm.backupManagers),
		"is_new_manager": lm.isNewManager,
	}
}

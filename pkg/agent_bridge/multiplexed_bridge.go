package agent_bridge

import (
	"fmt"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/agent_bridge/protocol"
	"github.com/sirupsen/logrus"
)

// LaneType نوع المسار في الجسر المتعدد
type LaneType int

const (
	LaneEmergency LaneType = iota // مسار الطوارئ
	LaneChat                      // مسار المحادثة
	LaneWorkflow                  // مسار سير العمل
	LaneFileUpload                // مسار رفع الملفات
	LaneFileDownload              // مسار تنزيل الملفات
)

// String يرجع تمثيل نصي لنوع المسار
func (lt LaneType) String() string {
	switch lt {
	case LaneEmergency:
		return "emergency"
	case LaneChat:
		return "chat"
	case LaneWorkflow:
		return "workflow"
	case LaneFileUpload:
		return "file_upload"
	case LaneFileDownload:
		return "file_download"
	default:
		return "unknown"
	}
}

// Lane يمثل مساراً في الجسر المتعدد
type Lane struct {
	laneType LaneType
	queue    chan *protocol.Message
	mu       sync.Mutex
}

// NewLane ينشئ مساراً جديداً
func NewLane(laneType LaneType, bufferSize int) *Lane {
	return &Lane{
		laneType: laneType,
		queue:    make(chan *protocol.Message, bufferSize),
	}
}

// Enqueue يضيف رسالة إلى قائمة الانتظار
func (l *Lane) Enqueue(msg *protocol.Message) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	select {
	case l.queue <- msg:
		return nil
	default:
		return fmt.Errorf("lane %s queue full", l.laneType.String())
	}
}

// Dequeue يستخرج رسالة من قائمة الانتظار
func (l *Lane) Dequeue() *protocol.Message {
	return <-l.queue
}

// MultiplexedBridge جسر متعدد المسارات للوكلاء
type MultiplexedBridge struct {
	lanes map[LaneType]*Lane
	mu    sync.RWMutex
	log   *logrus.Logger
}

// NewMultiplexedBridge ينشئ جسراً متعدداً جديداً
func NewMultiplexedBridge(log *logrus.Logger) *MultiplexedBridge {
	mb := &MultiplexedBridge{
		lanes: make(map[LaneType]*Lane),
		log:   log,
	}

	// إنشاء المسارات الخمسة
	mb.lanes[LaneEmergency] = NewLane(LaneEmergency, 100)    // أولوية عالية
	mb.lanes[LaneChat] = NewLane(LaneChat, 1000)              // متوسط
	mb.lanes[LaneWorkflow] = NewLane(LaneWorkflow, 500)      // متوسط
	mb.lanes[LaneFileUpload] = NewLane(LaneFileUpload, 200)  // منخفض
	mb.lanes[LaneFileDownload] = NewLane(LaneFileDownload, 200) // منخفض

	return mb
}

// Send يرسل رسالة عبر مسار محدد
func (mb *MultiplexedBridge) Send(laneType LaneType, msg *protocol.Message) error {
	mb.mu.RLock()
	lane, exists := mb.lanes[laneType]
	mb.mu.RUnlock()

	if !exists {
		return fmt.Errorf("lane %s does not exist", laneType.String())
	}

	if err := lane.Enqueue(msg); err != nil {
		return fmt.Errorf("failed to enqueue message: %w", err)
	}

	mb.log.WithFields(logrus.Fields{
		"lane": laneType.String(),
		"type": msg.Type,
	}).Debug("Message enqueued")

	return nil
}

// Receive يستقبل رسالة من مسار محدد
func (mb *MultiplexedBridge) Receive(laneType LaneType) (*protocol.Message, error) {
	mb.mu.RLock()
	lane, exists := mb.lanes[laneType]
	mb.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("lane %s does not exist", laneType.String())
	}

	msg := lane.Dequeue()
	mb.log.WithFields(logrus.Fields{
		"lane": laneType.String(),
		"type": msg.Type,
	}).Debug("Message dequeued")

	return msg, nil
}

// HandleTaskRequest يعالج طلب مهمة
func (mb *MultiplexedBridge) HandleTaskRequest(session *Session, msg *protocol.Message) error {
	// إرسال طلب المهمة عبر مسار سير العمل
	if err := mb.Send(LaneWorkflow, msg); err != nil {
		return err
	}

	// تحديث نشاط الجلسة
	session.UpdateLastActivity()

	return nil
}

// HandleTaskResponse يعالج استجابة مهمة
func (mb *MultiplexedBridge) HandleTaskResponse(session *Session, msg *protocol.Message) error {
	// إرسال الاستجابة عبر المسار المناسب بناءً على نوع المهمة
	// في التنفيذ الحالي، نستخدم مسار سير العمل
	if err := mb.Send(LaneWorkflow, msg); err != nil {
		return err
	}

	// تحديث نشاط الجلسة
	session.UpdateLastActivity()

	return nil
}

// GetLaneSize يرجع حجم قائمة انتظار مسار معين
func (mb *MultiplexedBridge) GetLaneSize(laneType LaneType) int {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	if lane, exists := mb.lanes[laneType]; exists {
		return len(lane.queue)
	}
	return 0
}

// GetAllLaneSizes يرجع أحجام جميع المسارات
func (mb *MultiplexedBridge) GetAllLaneSizes() map[LaneType]int {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	sizes := make(map[LaneType]int)
	for laneType, lane := range mb.lanes {
		sizes[laneType] = len(lane.queue)
	}
	return sizes
}

// Close يغلق الجسر المتعدد
func (mb *MultiplexedBridge) Close() {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	for laneType, lane := range mb.lanes {
		close(lane.queue)
		delete(mb.lanes, laneType)
		mb.log.WithField("lane", laneType.String()).Debug("Lane closed")
	}
}

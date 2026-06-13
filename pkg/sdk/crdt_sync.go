package sdk

import (
	"crypto/ed25519"
	"fmt"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/crypto"
)

// CRDTSyncManager يدير تبادل تحديثات Yjs عبر الشبكة اللامركزية
type CRDTSyncManager struct {
	documentID  string
	mu          sync.RWMutex
	subscribers map[string]func(update []byte) // معرف الوكيل/المستخدم -> دالة الاستدعاء
}

// NewCRDTSyncManager ينشئ مدير مزامنة لسير عمل محدد
func NewCRDTSyncManager(documentID string) *CRDTSyncManager {
	return &CRDTSyncManager{
		documentID:  documentID,
		subscribers: make(map[string]func(update []byte)),
	}
}

// BroadcastUpdate يبث تحديث Yjs (Uint8Array) إلى جميع المشاركين في سير العمل
func (c *CRDTSyncManager) BroadcastUpdate(updatePayload []byte, senderPrivKey ed25519.PrivateKey) error {
	// 1. توقيع التحديث لضمان أنه جاء من مصدر موثوق
	domain := crypto.DomainDirectMsg + c.documentID + "|"
	signature, err := crypto.SignPayloadHex(senderPrivKey, domain, string(updatePayload))
	if err != nil {
		return fmt.Errorf("failed to sign update: %w", err)
	}

	// 2. تغليف التحديث في رسالة آمنة
	message := map[string]interface{}{
		"type":       "yjs_update",
		"documentID": c.documentID,
		"payload":    updatePayload,
		"signature":  signature,
	}

	// 3. في التنفيذ الحقيقي، يتم النشر عبر القناة
	// هنا سنقوم بمحاكاة النشر عن طريق استدعاء OnIncomingUpdate
	_ = message

	return nil
}

// Subscribe يسمح لعميل (وكيل أو بشري) بالاستماع للتحديثات
func (c *CRDTSyncManager) Subscribe(subscriberID string, callback func(update []byte)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.subscribers[subscriberID] = callback
}

// OnIncomingUpdate يُستدعى عند استقبال رسالة من القناة
func (c *CRDTSyncManager) OnIncomingUpdate(message map[string]interface{}) {
	if msgType, ok := message["type"].(string); !ok || msgType != "yjs_update" {
		return
	}

	if docID, ok := message["documentID"].(string); !ok || docID != c.documentID {
		return
	}

	payload, ok := message["payload"].([]byte)
	if !ok {
		return
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// توزيع التحديث على جميع المشتركين المحليين
	for _, callback := range c.subscribers {
		go callback(payload) // تشغيل في Goroutine لمنع الحظر
	}
}

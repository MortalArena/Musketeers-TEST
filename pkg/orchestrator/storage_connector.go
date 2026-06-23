package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/MortalArena/Musketeers/pkg/storage"
	"go.uber.org/zap"
)

// ============================================================
// Storage Connector - ربط نظام التخزين ب Connector
// ============================================================

// StorageConnector يربط نظام التخزين ب Connector لمنع عزلة الملفات
type StorageConnector struct {
	// المكونات الأساسية
	eventBus *eventbus.EventBus
	quotaMgr *storage.QuotaManager

	// الملفات
	files map[string]*StorageFile
	mu    sync.RWMutex

	// Channels للتواصل الداخلي
	storageToEventBus chan *StorageEvent
	eventBusToStorage chan eventbus.Event

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Counter for unique file IDs
	fileCounter int64

	// Logger
	logger *zap.Logger

	// Metrics
	metrics *StorageMetrics
}

// StorageMetrics مقاييس التخزين
type StorageMetrics struct {
	FilesStored    int64
	FilesRetrieved int64
	FilesDeleted   int64
	BytesStored    int64
	BytesRetrieved int64
	Errors         int64
	LastActivity   time.Time
}

// StorageFile ملف تخزين
type StorageFile struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Size      int64                  `json:"size"`
	Type      string                 `json:"type"`
	Content   []byte                 `json:"content,omitempty"`
	OwnerDID  string                 `json:"owner_did"`
	SessionID string                 `json:"session_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// StorageEvent حدث تخزين
type StorageEvent struct {
	Type      string                 `json:"type"` // store, retrieve, delete, list
	FileID    string                 `json:"file_id,omitempty"`
	File      *StorageFile           `json:"file,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewStorageConnector ينشئ StorageConnector جديد
func NewStorageConnector(eventBus *eventbus.EventBus, quotaMgr *storage.QuotaManager, logger *zap.Logger) *StorageConnector {
	ctx, cancel := context.WithCancel(context.Background())

	return &StorageConnector{
		eventBus:          eventBus,
		quotaMgr:          quotaMgr,
		files:             make(map[string]*StorageFile),
		storageToEventBus: make(chan *StorageEvent, 1000),
		eventBusToStorage: make(chan eventbus.Event, 1000),
		ctx:               ctx,
		cancel:            cancel,
		logger:            logger,
		metrics:           &StorageMetrics{},
	}
}

// Start يبدأ StorageConnector
func (sc *StorageConnector) Start() error {
	sc.logger.Info("بدء StorageConnector")

	// الاشتراك في أحداث Event Bus
	sc.subscribeToEventBus()

	// بدء معالج التخزين
	sc.wg.Add(1)
	go sc.storageHandler()

	// بدء معالج Event Bus
	sc.wg.Add(1)
	go sc.eventBusHandler()

	sc.logger.Info("تم بدء StorageConnector بنجاح")
	return nil
}

// Stop يوقف StorageConnector
func (sc *StorageConnector) Stop() error {
	sc.logger.Info("إيقاف StorageConnector")

	sc.cancel()
	sc.wg.Wait()

	close(sc.storageToEventBus)
	close(sc.eventBusToStorage)

	sc.logger.Info("تم إيقاف StorageConnector بنجاح")
	return nil
}

// ============================================================
// إدارة الملفات
// ============================================================

// StoreFile يخزن ملف
func (sc *StorageConnector) StoreFile(file *StorageFile) error {
	// التحقق من الحصة
	if sc.quotaMgr != nil {
		if err := sc.quotaMgr.CheckAndAdd(file.OwnerDID, file.Size); err != nil {
			return fmt.Errorf("فشل التحقق من الحصة: %w", err)
		}
	}

	// [FIX] تأكد من أن الملف له ID فريد باستخدام counter
	sc.mu.Lock()
	sc.fileCounter++
	file.ID = fmt.Sprintf("file_%d_%d", time.Now().UnixNano(), sc.fileCounter)
	file.CreatedAt = time.Now()
	file.UpdatedAt = time.Now()
	sc.files[file.ID] = file
	sc.logger.Debug("تم تخزين ملف",
		zap.String("file_id", file.ID),
		zap.String("owner_did", file.OwnerDID),
		zap.Int("total_files", len(sc.files)),
	)
	sc.mu.Unlock()

	// إرسال حدث
	event := &StorageEvent{
		Type:      "store",
		FileID:    file.ID,
		File:      file,
		Timestamp: time.Now(),
	}
	sc.storageToEventBus <- event

	sc.mu.Lock()
	sc.metrics.FilesStored++
	sc.metrics.BytesStored += file.Size
	sc.metrics.LastActivity = time.Now()
	sc.mu.Unlock()

	sc.logger.Info("تم تخزين ملف",
		zap.String("file_id", file.ID),
		zap.String("name", file.Name),
		zap.Int64("size", file.Size),
	)

	return nil
}

// RetrieveFile يسترجع ملف
func (sc *StorageConnector) RetrieveFile(fileID string) (*StorageFile, error) {
	sc.mu.RLock()
	file, exists := sc.files[fileID]
	sc.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("الملف %s غير موجود", fileID)
	}

	// إرسال حدث
	event := &StorageEvent{
		Type:      "retrieve",
		FileID:    fileID,
		File:      file,
		Timestamp: time.Now(),
	}
	sc.storageToEventBus <- event

	sc.mu.Lock()
	sc.metrics.FilesRetrieved++
	sc.metrics.BytesRetrieved += file.Size
	sc.metrics.LastActivity = time.Now()
	sc.mu.Unlock()

	return file, nil
}

// DeleteFile يحذف ملف
func (sc *StorageConnector) DeleteFile(fileID string) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	file, exists := sc.files[fileID]
	if !exists {
		return fmt.Errorf("الملف %s غير موجود", fileID)
	}

	// تحديث الحصة
	if sc.quotaMgr != nil {
		sc.quotaMgr.Release(file.OwnerDID, file.Size)
	}

	delete(sc.files, fileID)

	// إرسال حدث
	event := &StorageEvent{
		Type:      "delete",
		FileID:    fileID,
		Timestamp: time.Now(),
	}
	sc.storageToEventBus <- event

	sc.metrics.FilesDeleted++
	sc.metrics.LastActivity = time.Now()

	sc.logger.Info("تم حذف ملف",
		zap.String("file_id", fileID),
	)

	return nil
}

// ListFiles يرجع قائمة الملفات
func (sc *StorageConnector) ListFiles(ownerDID string, sessionID string) []*StorageFile {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	var files []*StorageFile
	for _, file := range sc.files {
		sc.logger.Debug("فحص ملف",
			zap.String("file_id", file.ID),
			zap.String("owner_did", file.OwnerDID),
			zap.String("search_owner", ownerDID),
			zap.String("session_id", file.SessionID),
			zap.String("search_session", sessionID),
		)
		if ownerDID != "" && file.OwnerDID != ownerDID {
			continue
		}
		if sessionID != "" && file.SessionID != sessionID {
			continue
		}
		files = append(files, file)
	}

	sc.logger.Debug("نتيجة ListFiles",
		zap.String("owner_did", ownerDID),
		zap.String("session_id", sessionID),
		zap.Int("total_files", len(sc.files)),
		zap.Int("matched_files", len(files)),
	)

	return files
}

// ============================================================
// معالجة الرسائل
// ============================================================

// subscribeToEventBus يرتبط بأحداث Event Bus
func (sc *StorageConnector) subscribeToEventBus() {
	sc.eventBus.Subscribe("storage.store", sc.handleStorageStore)
	sc.eventBus.Subscribe("storage.retrieve", sc.handleStorageRetrieve)
	sc.eventBus.Subscribe("storage.delete", sc.handleStorageDelete)
	sc.eventBus.Subscribe("storage.list", sc.handleStorageList)
}

// storageHandler يعالج رسائل التخزين
func (sc *StorageConnector) storageHandler() {
	defer sc.wg.Done()

	for {
		select {
		case <-sc.ctx.Done():
			return
		case event := <-sc.storageToEventBus:
			sc.processStorageEvent(event)
		}
	}
}

// processStorageEvent يعالج حدث تخزين
func (sc *StorageConnector) processStorageEvent(event *StorageEvent) {
	// تحويل الحدث إلى حدث Event Bus
	busEvent := eventbus.Event{
		Type:      "storage.event",
		Payload:   event,
		Timestamp: event.Timestamp,
	}

	// نشر الحدث
	sc.eventBus.Publish(busEvent)

	sc.logger.Debug("تم معالجة حدث تخزين",
		zap.String("type", event.Type),
		zap.String("file_id", event.FileID),
	)
}

// eventBusHandler يعالج أحداث Event Bus
func (sc *StorageConnector) eventBusHandler() {
	defer sc.wg.Done()

	for {
		select {
		case <-sc.ctx.Done():
			return
		case event := <-sc.eventBusToStorage:
			sc.processEventBusEvent(event)
		}
	}
}

// processEventBusEvent يعالج حدث Event Bus
func (sc *StorageConnector) processEventBusEvent(event eventbus.Event) {
	sc.logger.Debug("تم معالجة حدث Event Bus",
		zap.String("event_type", event.Type),
	)
}

// handleStorageStore يعالج تخزين
func (sc *StorageConnector) handleStorageStore(event eventbus.Event) {
	sc.logger.Debug("استقبال طلب تخزين")
}

// handleStorageRetrieve يعالج استرجاع
func (sc *StorageConnector) handleStorageRetrieve(event eventbus.Event) {
	sc.logger.Debug("استقبال طلب استرجاع")
}

// handleStorageDelete يعالج حذف
func (sc *StorageConnector) handleStorageDelete(event eventbus.Event) {
	sc.logger.Debug("استقبال طلب حذف")
}

// handleStorageList يعالج قائمة
func (sc *StorageConnector) handleStorageList(event eventbus.Event) {
	sc.logger.Debug("استقبال طلب قائمة")
}

// ============================================================
// المقاييس
// ============================================================

// GetMetrics يحصل على المقاييس
func (sc *StorageConnector) GetMetrics() *StorageMetrics {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return &StorageMetrics{
		FilesStored:    sc.metrics.FilesStored,
		FilesRetrieved: sc.metrics.FilesRetrieved,
		FilesDeleted:   sc.metrics.FilesDeleted,
		BytesStored:    sc.metrics.BytesStored,
		BytesRetrieved: sc.metrics.BytesRetrieved,
		Errors:         sc.metrics.Errors,
		LastActivity:   sc.metrics.LastActivity,
	}
}

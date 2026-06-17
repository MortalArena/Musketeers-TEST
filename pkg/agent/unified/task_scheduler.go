package unified

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Task مهمة في النظام
type Task struct {
	ID        string
	Type      string
	Priority  EventPriority
	Execute   func(ctx context.Context) error
	Cancel    context.CancelFunc
	CreatedAt time.Time
	Timeout   time.Duration
}

// TaskScheduler مجدول مهام يضمن عدم التعارض
type TaskScheduler struct {
	sessionID string
	logger    *zap.Logger
	tasks     chan *Task
	running   bool
	mu        sync.RWMutex
	wg        sync.WaitGroup
}

// NewTaskScheduler ينشئ مجدول مهام جديد
func NewTaskScheduler(sessionID string, logger *zap.Logger) *TaskScheduler {
	return &TaskScheduler{
		sessionID: sessionID,
		logger:    logger,
		tasks:     make(chan *Task, 1000), // قائمة انتظار كبيرة
		running:   false,
	}
}

// Start يبدأ المجدول
func (ts *TaskScheduler) Start(ctx context.Context) {
	ts.mu.Lock()
	if ts.running {
		ts.mu.Unlock()
		return
	}
	ts.running = true
	ts.mu.Unlock()

	ts.wg.Add(1)
	go ts.run(ctx)

	ts.logger.Info("تم بدء مجدول المهام")
}

// Stop يوقف المجدول
func (ts *TaskScheduler) Stop() {
	ts.mu.Lock()
	if !ts.running {
		ts.mu.Unlock()
		return
	}
	ts.running = false
	ts.mu.Unlock()

	close(ts.tasks)
	ts.wg.Wait()

	ts.logger.Info("تم إيقاف مجدول المهام")
}

// run يدير المهام بشكل دوري
func (ts *TaskScheduler) run(ctx context.Context) {
	defer ts.wg.Done()

	for {
		select {
		case <-ctx.Done():
			ts.logger.Info("تم إيقاف مجدول المهام بسبب إلغاء السياق")
			return
		case task, ok := <-ts.tasks:
			if !ok {
				ts.logger.Info("تم إغلاق قائمة المهام")
				return
			}

			ts.executeTask(ctx, task)
		}
	}
}

// executeTask ينفذ مهمة واحدة
func (ts *TaskScheduler) executeTask(ctx context.Context, task *Task) {
	startTime := time.Now()

	ts.logger.Info("بدء تنفيذ مهمة",
		zap.String("task_id", task.ID),
		zap.String("task_type", task.Type),
		zap.String("priority", string(task.Priority)),
	)

	// إنشاء سياق مع timeout
	taskCtx := ctx
	if task.Timeout > 0 {
		var cancel context.CancelFunc
		taskCtx, cancel = context.WithTimeout(ctx, task.Timeout)
		defer cancel()
	}

	// تنفيذ المهمة
	err := task.Execute(taskCtx)

	duration := time.Since(startTime)

	if err != nil {
		ts.logger.Error("فشل تنفيذ مهمة",
			zap.String("task_id", task.ID),
			zap.String("task_type", task.Type),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
	} else {
		ts.logger.Info("تم تنفيذ مهمة بنجاح",
			zap.String("task_id", task.ID),
			zap.String("task_type", task.Type),
			zap.Duration("duration", duration),
		)
	}
}

// SubmitTask يقدم مهمة للتنفيذ
func (ts *TaskScheduler) SubmitTask(task *Task) error {
	ts.mu.RLock()
	if !ts.running {
		ts.mu.RUnlock()
		return fmt.Errorf("مجدول المهام غير نشط")
	}
	ts.mu.RUnlock()

	select {
	case ts.tasks <- task:
		ts.logger.Debug("تم تقديم مهمة",
			zap.String("task_id", task.ID),
			zap.String("task_type", task.Type),
			zap.String("priority", string(task.Priority)),
		)
		return nil
	default:
		return fmt.Errorf("قائمة المهام ممتلئة")
	}
}

// GetQueueLength يحصل على طول قائمة الانتظار
func (ts *TaskScheduler) GetQueueLength() int {
	return len(ts.tasks)
}

// GetStatus يحصل على حالة المجدول
func (ts *TaskScheduler) GetStatus() map[string]interface{} {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	return map[string]interface{}{
		"running":      ts.running,
		"queue_length": len(ts.tasks),
		"session_id":   ts.sessionID,
	}
}

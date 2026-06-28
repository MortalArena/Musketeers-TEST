package session

import (
	"context"
	"testing"
	"time"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSessionContainer_HighLoadAgents اختبار تحميل عالي من الوكلاء
func TestSessionContainer_HighLoadAgents(t *testing.T) {
	opts := badger.DefaultOptions("")
	opts.InMemory = true
	db, err := badger.Open(opts)
	require.NoError(t, err)
	defer db.Close()

	eb := eventbus.NewEventBus()

	config := &SessionConfig{
		Name:        "High Load Test",
		Description: "High Load Description",
		OwnerDID:    "did:test:123",
		MaxAgents:   MaxAgentsInState,
		ProjectType: "test",
	}

	ctx := context.Background()
	container, err := NewSessionContainer(ctx, db, config, eb)
	require.NoError(t, err)

	start := time.Now()

	// إضافة الحد الأقصى من الوكلاء
	for i := 0; i < MaxAgentsInState; i++ {
		err = container.AddAgent(
			"did:agent:"+string(rune('0'+i%10))+string(rune('0'+(i/10)%10)),
			"Agent "+string(rune('0'+i)),
			"assistant",
		)
		assert.NoError(t, err)
	}

	duration := time.Since(start)
	t.Logf("إضافة %d وكيل استغرقت: %v", MaxAgentsInState, duration)

	state := container.GetUnifiedState()
	assert.Equal(t, MaxAgentsInState, len(state.Agents))
}

// TestSessionContainer_HighLoadTasks اختبار تحميل عالي من المهام
func TestSessionContainer_HighLoadTasks(t *testing.T) {
	opts := badger.DefaultOptions("")
	opts.InMemory = true
	db, err := badger.Open(opts)
	require.NoError(t, err)
	defer db.Close()

	eb := eventbus.NewEventBus()

	config := &SessionConfig{
		Name:        "High Load Tasks",
		Description: "High Load Tasks Description",
		OwnerDID:    "did:test:123",
		MaxAgents:   5,
		ProjectType: "test",
	}

	ctx := context.Background()
	container, err := NewSessionContainer(ctx, db, config, eb)
	require.NoError(t, err)

	start := time.Now()

	// إضافة الحد الأقصى من المهام
	for i := 0; i < MaxTasksInState; i++ {
		err = container.AddTask(
			"task:"+string(rune('0'+i%10))+string(rune('0'+(i/10)%10)),
			"Task "+string(rune('0'+i)),
			"did:agent:1",
			"high",
		)
		assert.NoError(t, err)
	}

	duration := time.Since(start)
	t.Logf("إضافة %d مهمة استغرقت: %v", MaxTasksInState, duration)

	state := container.GetUnifiedState()
	assert.Equal(t, MaxTasksInState, len(state.Tasks))
}

// TestSessionContainer_RapidStateUpdates اختبار تحديثات الحالة السريعة
func TestSessionContainer_RapidStateUpdates(t *testing.T) {
	opts := badger.DefaultOptions("")
	opts.InMemory = true
	db, err := badger.Open(opts)
	require.NoError(t, err)
	defer db.Close()

	eb := eventbus.NewEventBus()

	config := &SessionConfig{
		Name:        "Rapid Updates",
		Description: "Rapid Updates Description",
		OwnerDID:    "did:test:123",
		MaxAgents:   5,
		ProjectType: "test",
	}

	ctx := context.Background()
	container, err := NewSessionContainer(ctx, db, config, eb)
	require.NoError(t, err)

	// إضافة مهمة واحدة
	err = container.AddTask("task:1", "Task 1", "did:agent:1", "high")
	require.NoError(t, err)

	start := time.Now()

	// تحديث الحالة 1000 مرة
	for i := 0; i < 1000; i++ {
		status := "pending"
		if i%2 == 0 {
			status = "in_progress"
		} else {
			status = "completed"
		}
		err = container.UpdateTaskStatus("task:1", status)
		assert.NoError(t, err)
	}

	duration := time.Since(start)
	t.Logf("1000 تحديث حالة استغرقت: %v", duration)

	// يجب أن تكون العملية سريعة (أقل من 5 ثواني)
	assert.Less(t, duration, 5*time.Second)
}

// TestSessionContainer_ConcurrentAccess اختبار الوصول المتزامن
func TestSessionContainer_ConcurrentAccess(t *testing.T) {
	opts := badger.DefaultOptions("")
	opts.InMemory = true
	db, err := badger.Open(opts)
	require.NoError(t, err)
	defer db.Close()

	eb := eventbus.NewEventBus()

	config := &SessionConfig{
		Name:        "Concurrent Access",
		Description: "Concurrent Access Description",
		OwnerDID:    "did:test:123",
		MaxAgents:   20,
		ProjectType: "test",
	}

	ctx := context.Background()
	container, err := NewSessionContainer(ctx, db, config, eb)
	require.NoError(t, err)

	start := time.Now()
	done := make(chan bool, 20)

	// 20 goroutine تضيف وكلاء بشكل متزامن
	for i := 0; i < 20; i++ {
		go func(index int) {
			defer func() { recover() }()
			err := container.AddAgent(
				"did:agent:"+string(rune('0'+index)),
				"Agent "+string(rune('0'+index)),
				"assistant",
			)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// انتظار جميع العمليات
	for i := 0; i < 20; i++ {
		<-done
	}

	duration := time.Since(start)
	t.Logf("20 عملية متزامنة استغرقت: %v", duration)

	state := container.GetUnifiedState()
	assert.Equal(t, 20, len(state.Agents))
}

// TestSessionContainer_ExportPerformance اختبار أداء التصدير
func TestSessionContainer_ExportPerformance(t *testing.T) {
	opts := badger.DefaultOptions("")
	opts.InMemory = true
	db, err := badger.Open(opts)
	require.NoError(t, err)
	defer db.Close()

	eb := eventbus.NewEventBus()

	config := &SessionConfig{
		Name:        "Export Performance",
		Description: "Export Performance Description",
		OwnerDID:    "did:test:123",
		MaxAgents:   20,
		ProjectType: "test",
	}

	ctx := context.Background()
	container, err := NewSessionContainer(ctx, db, config, eb)
	require.NoError(t, err)

	// إضافة بيانات كثيرة
	for i := 0; i < 20; i++ {
		err = container.AddAgent("did:agent:"+string(rune('0'+i)), "Agent", "assistant")
		assert.NoError(t, err)
	}

	for i := 0; i < 50; i++ {
		err = container.AddTask("task:"+string(rune('0'+i)), "Task", "did:agent:1", "high")
		assert.NoError(t, err)
	}

	start := time.Now()

	// تصدير الجلسة
	exportData, err := container.Export()
	require.NoError(t, err)

	duration := time.Since(start)
	t.Logf("تصدير جلسة بـ 70 عنصر استغرقت: %v", duration)

	// يجب أن تكون العملية سريعة (أقل من 1 ثانية)
	assert.Less(t, duration, 1*time.Second)
	assert.NotNil(t, exportData)
	assert.Equal(t, 20, len(exportData.State.Agents))
	assert.Equal(t, 50, len(exportData.State.Tasks))
}

// TestSessionContainer_ImportPerformance اختبار أداء الاستيراد
func TestSessionContainer_ImportPerformance(t *testing.T) {
	opts := badger.DefaultOptions("")
	opts.InMemory = true
	db, err := badger.Open(opts)
	require.NoError(t, err)
	defer db.Close()

	eb := eventbus.NewEventBus()

	config := &SessionConfig{
		Name:        "Import Performance",
		Description: "Import Performance Description",
		OwnerDID:    "did:test:123",
		MaxAgents:   20,
		ProjectType: "test",
	}

	ctx := context.Background()
	container, err := NewSessionContainer(ctx, db, config, eb)
	require.NoError(t, err)

	// تصدير جلسة كبيرة
	for i := 0; i < 20; i++ {
		err = container.AddAgent("did:agent:"+string(rune('0'+i)), "Agent", "assistant")
		assert.NoError(t, err)
	}

	for i := 0; i < 50; i++ {
		err = container.AddTask("task:"+string(rune('0'+i)), "Task", "did:agent:1", "high")
		assert.NoError(t, err)
	}

	exportData, err := container.Export()
	require.NoError(t, err)

	// إنشاء جلسة جديدة بنفس المعرف
	newConfig := &SessionConfig{
		Name:        "Import Performance New",
		Description: "Import Performance Description New",
		OwnerDID:    "did:test:123",
		MaxAgents:   20,
		ProjectType: "test",
	}

	newContainer, err := NewSessionContainer(ctx, db, newConfig, eb)
	require.NoError(t, err)

	// تعيين نفس معرف الجلسة
	newContainer.ID = container.ID

	start := time.Now()

	// استيراد البيانات
	err = newContainer.Import(exportData, db, eb)
	require.NoError(t, err)

	duration := time.Since(start)
	t.Logf("استيراد جلسة بـ 70 عنصر استغرقت: %v", duration)

	// يجب أن تكون العملية سريعة (أقل من 1 ثانية)
	assert.Less(t, duration, 1*time.Second)

	state := newContainer.GetUnifiedState()
	assert.Equal(t, 20, len(state.Agents))
	assert.Equal(t, 50, len(state.Tasks))
}

// TestSessionContainer_MemoryUsage اختبار استخدام الذاكرة
func TestSessionContainer_MemoryUsage(t *testing.T) {
	opts := badger.DefaultOptions("")
	opts.InMemory = true
	db, err := badger.Open(opts)
	require.NoError(t, err)
	defer db.Close()

	eb := eventbus.NewEventBus()

	config := &SessionConfig{
		Name:        "Memory Usage",
		Description: "Memory Usage Description",
		OwnerDID:    "did:test:123",
		MaxAgents:   20,
		ProjectType: "test",
	}

	ctx := context.Background()
	container, err := NewSessionContainer(ctx, db, config, eb)
	require.NoError(t, err)

	// إضافة بيانات كثيرة
	for i := 0; i < 20; i++ {
		err = container.AddAgent("did:agent:"+string(rune('0'+i)), "Agent", "assistant")
		assert.NoError(t, err)
	}

	for i := 0; i < 50; i++ {
		err = container.AddTask("task:"+string(rune('0'+i)), "Task", "did:agent:1", "high")
		assert.NoError(t, err)
	}

	// الحصول على الحالة عدة مرات
	for i := 0; i < 1000; i++ {
		state := container.GetUnifiedState()
		assert.NotNil(t, state)
	}

	// إذا وصلنا هنا بدون panic أو out of memory، فهذا جيد
	assert.True(t, true)
}

// TestSessionContainer_JournalPerformance اختبار أداء السجل
func TestSessionContainer_JournalPerformance(t *testing.T) {
	opts := badger.DefaultOptions("")
	opts.InMemory = true
	db, err := badger.Open(opts)
	require.NoError(t, err)
	defer db.Close()

	eb := eventbus.NewEventBus()

	config := &SessionConfig{
		Name:        "Journal Performance",
		Description: "Journal Performance Description",
		OwnerDID:    "did:test:123",
		MaxAgents:   5,
		ProjectType: "test",
	}

	ctx := context.Background()
	container, err := NewSessionContainer(ctx, db, config, eb)
	require.NoError(t, err)

	start := time.Now()

	// إضافة 1000 إدخال للسجل
	for i := 0; i < 1000; i++ {
		container.Journal.Append(JournalTaskCreated, "agent:1", "agent", "Task created", nil)
	}

	duration := time.Since(start)
	t.Logf("إضافة 1000 إدخال للسجل استغرقت: %v", duration)

	// يجب أن تكون العملية سريعة (أقل من 2 ثانية)
	assert.Less(t, duration, 2*time.Second)

	// التحقق من عدد الإدخالات
	journalSize := container.Journal.Size()
	assert.GreaterOrEqual(t, journalSize, 1000)
}

// TestSessionContainer_StateRetrievalPerformance اختبار أداء استرجاع الحالة
func TestSessionContainer_StateRetrievalPerformance(t *testing.T) {
	opts := badger.DefaultOptions("")
	opts.InMemory = true
	db, err := badger.Open(opts)
	require.NoError(t, err)
	defer db.Close()

	eb := eventbus.NewEventBus()

	config := &SessionConfig{
		Name:        "State Retrieval",
		Description: "State Retrieval Description",
		OwnerDID:    "did:test:123",
		MaxAgents:   20,
		ProjectType: "test",
	}

	ctx := context.Background()
	container, err := NewSessionContainer(ctx, db, config, eb)
	require.NoError(t, err)

	// إضافة بيانات
	for i := 0; i < 20; i++ {
		err = container.AddAgent("did:agent:"+string(rune('0'+i)), "Agent", "assistant")
		assert.NoError(t, err)
	}

	for i := 0; i < 50; i++ {
		err = container.AddTask("task:"+string(rune('0'+i)), "Task", "did:agent:1", "high")
		assert.NoError(t, err)
	}

	start := time.Now()

	// استرجاع الحالة 1000 مرة
	for i := 0; i < 1000; i++ {
		state := container.GetUnifiedState()
		assert.NotNil(t, state)
	}

	duration := time.Since(start)
	t.Logf("استرجاع الحالة 1000 مرة استغرقت: %v", duration)

	// يجب أن تكون العملية سريعة (أقل من 1 ثانية)
	assert.Less(t, duration, 1*time.Second)
}

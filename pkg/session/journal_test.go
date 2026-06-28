package session

import (
	"os"
	"testing"
	"time"
)

func TestSessionJournalAppend(t *testing.T) {
	j := NewSessionJournal("test-session-1")
	j.autoSave = false

	e := j.Append(JournalSessionCreated, "system", "system", "تم إنشاء الجلسة", nil)
	if e.ID == "" {
		t.Error("يجب أن يكون للإدخال ID")
	}
	if e.Type != JournalSessionCreated {
		t.Errorf("نوع الإدخال: %s", e.Type)
	}
	if j.Size() != 1 {
		t.Errorf("الحجم: %d", j.Size())
	}
}

func TestSessionJournalMultipleEntries(t *testing.T) {
	j := NewSessionJournal("test-session-2")
	j.autoSave = false

	j.Append(JournalSessionCreated, "system", "system", "إنشاء الجلسة", nil)
	j.Append(JournalAgentAdded, "agent-planner", "agent", "إضافة المخطط", map[string]string{"role": "planner"})
	j.Append(JournalTaskCreated, "agent-planner", "agent", "إنشاء مهمة", map[string]string{"task_id": "t1"})
	j.Append(JournalTaskCompleted, "agent-coder", "agent", "إكمال مهمة", map[string]string{"task_id": "t1"})
	j.Append(JournalHumanJoined, "user-1", "human", "انضمام مستخدم", nil)

	if j.Size() != 5 {
		t.Errorf("الحجم: %d", j.Size())
	}
}

func TestSessionJournalQuery(t *testing.T) {
	j := NewSessionJournal("test-session-3")
	j.autoSave = false

	j.Append(JournalTaskCreated, "p1", "agent", "مهمة 1", nil)
	j.Append(JournalTaskCreated, "p1", "agent", "مهمة 2", nil)
	j.Append(JournalTaskCompleted, "c1", "agent", "مهمة 1 كاملة", nil)
	j.Append(JournalTaskCreated, "p1", "agent", "مهمة 3", nil)

	tasks := j.Query(JournalTaskCreated, 0)
	if len(tasks) != 3 {
		t.Errorf("مهام منشأة: %d", len(tasks))
	}

	completed := j.Query(JournalTaskCompleted, 0)
	if len(completed) != 1 {
		t.Errorf("مهام مكتملة: %d", len(completed))
	}
}

func TestSessionJournalQueryLimit(t *testing.T) {
	j := NewSessionJournal("test-session-4")
	j.autoSave = false

	for i := 0; i < 10; i++ {
		j.Append(JournalEventLogged, "src", "agent", "حدث", nil)
	}

	last3 := j.LastN(3)
	if len(last3) != 3 {
		t.Errorf("آخر 3: %d", len(last3))
	}
}

func TestSessionJournalExportImport(t *testing.T) {
	j1 := NewSessionJournal("test-session-5")
	j1.autoSave = false

	j1.Append(JournalSessionCreated, "system", "system", "إنشاء", nil)
	j1.Append(JournalAgentAdded, "a1", "agent", "إضافة وكيل", nil)

	exported := j1.Export()
	if len(exported) != 2 {
		t.Errorf("تصدير: %d", len(exported))
	}

	j2 := NewSessionJournal("test-session-5")
	j2.autoSave = false
	j2.Import(exported)

	if j2.Size() != 2 {
		t.Errorf("استيراد: %d", j2.Size())
	}

	// Duplicate import should be idempotent
	j2.Import(exported)
	if j2.Size() != 2 {
		t.Errorf("استيراد مكرر: %d", j2.Size())
	}
}

func TestSessionJournalQueryBySource(t *testing.T) {
	j := NewSessionJournal("test-session-6")
	j.autoSave = false

	j.Append(JournalAgentAdded, "agent-alpha", "agent", "إضافة alpha", nil)
	j.Append(JournalAgentAdded, "agent-beta", "agent", "إضافة beta", nil)
	j.Append(JournalTaskCreated, "agent-alpha", "agent", "مهمة alpha", nil)

	alphaEntries := j.QueryBySource("agent-alpha", 0)
	if len(alphaEntries) != 2 {
		t.Errorf("إدخالات alpha: %d", len(alphaEntries))
	}
}

func TestSessionJournalConcurrency(t *testing.T) {
	j := NewSessionJournal("test-session-conc")
	j.autoSave = false

	done := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		go func(n int) {
			defer func() { recover() }()
			j.Append(JournalEventLogged, "src", "agent", "حدث متزامن", nil)
			done <- true
		}(i)
	}

	for i := 0; i < 20; i++ {
		<-done
	}

	if j.Size() != 20 {
		t.Errorf("الإدخالات المتزامنة: %d", j.Size())
	}
}

func TestSessionJournalSummaryHTML(t *testing.T) {
	j := NewSessionJournal("test-html")
	j.autoSave = false

	j.Append(JournalSessionCreated, "system", "system", "إنشاء الجلسة", nil)
	j.Append(JournalAgentAdded, "agent-1", "agent", "وكيل 1", nil)

	html := j.SummaryHTML()
	if len(html) == 0 {
		t.Error("HTML فارغ")
	}
	if j.Size() != 2 {
		t.Errorf("الحجم بعد HTML: %d", j.Size())
	}
}

func TestSessionJournalLoadFromDisk(t *testing.T) {
	j := NewSessionJournal("test-disk-load")
	j.filePath = "test_journal.jsonl"
	defer func() {
		os.Remove(j.filePath)
	}()

	j.autoSave = true
	j.Append(JournalSessionCreated, "system", "system", "اختبار الحفظ", nil)
	j.Append(JournalAgentAdded, "a1", "agent", "وكيل اختبار", nil)

	j2 := NewSessionJournal("test-disk-load")
	j2.filePath = j.filePath
	j2.autoSave = false

	if err := j2.LoadFromDisk(); err != nil {
		t.Fatalf("خطأ في تحميل السجل: %v", err)
	}

	if j2.Size() != 2 {
		t.Errorf("إدخالات محملة: %d", j2.Size())
	}
}

func TestSessionJournalEmpty(t *testing.T) {
	j := NewSessionJournal("test-empty")
	j.autoSave = false

	if j.Size() != 0 {
		t.Errorf("الحجم: %d", j.Size())
	}

	all := j.All()
	if len(all) != 0 {
		t.Errorf("كل الإدخالات: %d", len(all))
	}

	last5 := j.LastN(5)
	if len(last5) != 0 {
		t.Errorf("آخر 5: %d", len(last5))
	}
}

func TestSessionJournalTimestamps(t *testing.T) {
	j := NewSessionJournal("test-ts")
	j.autoSave = false

	before := time.Now()
	j.Append(JournalSessionCreated, "system", "system", "وقت الإنشاء", nil)
	after := time.Now()

	entries := j.All()
	if len(entries) != 1 {
		t.Fatalf("إدخالات: %d", len(entries))
	}

	if entries[0].Timestamp.Before(before) || entries[0].Timestamp.After(after) {
		t.Error("الطابع الزمني خارج النطاق المتوقع")
	}
}

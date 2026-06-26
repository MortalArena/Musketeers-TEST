package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// JournalEntryType نوع إدخال في سجل الجلسة
type JournalEntryType string

const (
	JournalSessionCreated   JournalEntryType = "session.created"
	JournalSessionPaused    JournalEntryType = "session.paused"
	JournalSessionResumed   JournalEntryType = "session.resumed"
	JournalSessionCompleted JournalEntryType = "session.completed"

	JournalAgentAdded   JournalEntryType = "agent.added"
	JournalAgentRemoved JournalEntryType = "agent.removed"
	JournalAgentStatus  JournalEntryType = "agent.status"

	JournalTaskCreated   JournalEntryType = "task.created"
	JournalTaskUpdated   JournalEntryType = "task.updated"
	JournalTaskCompleted JournalEntryType = "task.completed"
	JournalTaskFailed    JournalEntryType = "task.failed"

	JournalStateChanged JournalEntryType = "state.changed"
	JournalEventLogged  JournalEntryType = "event.logged"

	JournalHumanJoined JournalEntryType = "human.joined"
	JournalHumanLeft   JournalEntryType = "human.left"

	JournalManagerChanged JournalEntryType = "manager.changed"
	JournalElectionStart  JournalEntryType = "election.started"
	JournalElectionDone   JournalEntryType = "election.completed"

	JournalMessageSent JournalEntryType = "message.sent"

	JournalMemoryUpdated JournalEntryType = "memory.updated"
	JournalSkillLearned  JournalEntryType = "skill.learned"

	JournalExported JournalEntryType = "session.exported"
	JournalImported JournalEntryType = "session.imported"
	JournalJoined   JournalEntryType = "session.joined"
	JournalLeft     JournalEntryType = "session.left"

	JournalCapabilityVerification JournalEntryType = "capability.verification"
	JournalAgentCapabilities      JournalEntryType = "agent.capabilities"
)

// JournalEntry إدخال واحد في سجل الجلسة
type JournalEntry struct {
	ID         string           `json:"id"`
	Timestamp  time.Time        `json:"timestamp"`
	Type       JournalEntryType `json:"type"`
	SourceID   string           `json:"source_id"`
	SourceType string           `json:"source_type"` // "agent", "human", "system", "node"
	Summary    string           `json:"summary"`
	Details    json.RawMessage  `json:"details,omitempty"`
	SessionID  string           `json:"session_id"`
}

// SessionJournal سجل أحداث الجلسة — append-only
// [WHY] يسجل كل ما يحدث في الجلسة بتفصيل كامل
//
//	كل جهاز ينضم يحصل على التاريخ الكامل
//	كل مستخدم يعيد فتح الجلسة يرى كل ما حدث
//	كل إدخال جديد يُنشر للشبكة عبر OnAppend (real-time sync)
type SessionJournal struct {
	mu        sync.RWMutex
	entries   []JournalEntry
	sessionID string
	filePath  string
	autoSave  bool

	// OnAppend يُستدعى عند كل إدخال جديد للبث للشبكة (real-time sync)
	// [WHY] يضمن أن كل الأجهزة ترى الإدخالات الجديدة فوراً
	OnAppend func(entry JournalEntry)
}

// NewSessionJournal ينشئ سجل أحداث جديد
func NewSessionJournal(sessionID string) *SessionJournal {
	return &SessionJournal{
		entries:   make([]JournalEntry, 0),
		sessionID: sessionID,
		filePath:  filepath.Join(".", "sessions", sessionID, "journal.jsonl"),
		autoSave:  true,
	}
}

// NewSessionJournalWithPath ينشئ سجل أحداث بمسار حفظ مخصص
func NewSessionJournalWithPath(sessionID, basePath string) *SessionJournal {
	return &SessionJournal{
		entries:   make([]JournalEntry, 0),
		sessionID: sessionID,
		filePath:  filepath.Join(basePath, "journal.jsonl"),
		autoSave:  true,
	}
}

// Append يُضيف إدخالاً جديداً للسجل (thread-safe)
func (j *SessionJournal) Append(entryType JournalEntryType, sourceID, sourceType, summary string, details interface{}) JournalEntry {
	return j.append(entryType, sourceID, sourceType, summary, details, true)
}

func (j *SessionJournal) append(entryType JournalEntryType, sourceID, sourceType, summary string, details interface{}, doSave bool) JournalEntry {
	var detailsJSON json.RawMessage
	if details != nil {
		switch d := details.(type) {
		case []byte:
			detailsJSON = d
		case string:
			detailsJSON = json.RawMessage(d)
		default:
			var err error
			if detailsJSON, err = json.Marshal(d); err != nil {
				// [SAFETY] إذا فشل marshal، نستخدم نص بسيط
				detailsJSON = json.RawMessage(fmt.Sprintf(`{"error":"failed to marshal details: %v"}`, err))
			}
		}
	}

	entry := JournalEntry{
		ID:         uuid.New().String(),
		Timestamp:  time.Now(),
		Type:       entryType,
		SourceID:   sourceID,
		SourceType: sourceType,
		Summary:    summary,
		Details:    detailsJSON,
		SessionID:  j.sessionID,
	}

	j.mu.Lock()
	j.entries = append(j.entries, entry)
	// نسخ OnAppend خارج القفل لمنع deadlock
	onAppend := j.OnAppend
	j.mu.Unlock()

	if doSave && j.autoSave {
		if err := j.saveEntry(entry); err != nil {
			// [SAFETY] تسجيل الخطأ ولكن لا نمنع إضافة الإدخال
			// في التطبيق الحقيقي، يمكن استخدام logger هنا
		}
	}

	// البث للشبكة (real-time sync)
	if onAppend != nil {
		onAppend(entry)
	}

	return entry
}

// saveEntry يحفظ إدخالاً واحداً على القرص (JSON Lines)
func (j *SessionJournal) saveEntry(entry JournalEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}
	dir := filepath.Dir(j.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	f, err := os.OpenFile(j.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}
	if _, err := f.Write([]byte("\n")); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}
	return nil
}

// All يرجع جميع الإدخالات
func (j *SessionJournal) All() []JournalEntry {
	j.mu.RLock()
	defer j.mu.RUnlock()

	result := make([]JournalEntry, len(j.entries))
	copy(result, j.entries)
	return result
}

// Query يبحث في السجل حسب النوع والمصدر
func (j *SessionJournal) Query(filterType JournalEntryType, limit int) []JournalEntry {
	j.mu.RLock()
	defer j.mu.RUnlock()

	var result []JournalEntry
	for i := len(j.entries) - 1; i >= 0 && (limit <= 0 || len(result) < limit); i-- {
		if j.entries[i].Type == filterType {
			result = append(result, j.entries[i])
		}
	}
	// reverse to chronological
	for i, k := 0, len(result)-1; i < k; i, k = i+1, k-1 {
		result[i], result[k] = result[k], result[i]
	}
	return result
}

// QueryBySource يبحث حسب المصدر
func (j *SessionJournal) QueryBySource(sourceID string, limit int) []JournalEntry {
	j.mu.RLock()
	defer j.mu.RUnlock()

	var result []JournalEntry
	for i := len(j.entries) - 1; i >= 0 && (limit <= 0 || len(result) < limit); i-- {
		if j.entries[i].SourceID == sourceID {
			result = append(result, j.entries[i])
		}
	}
	for i, k := 0, len(result)-1; i < k; i, k = i+1, k-1 {
		result[i], result[k] = result[k], result[i]
	}
	return result
}

// Export يُصدّر جميع الإدخالات لنقلها إلى جهاز آخر
func (j *SessionJournal) Export() []JournalEntry {
	return j.All()
}

// Import يستورد إدخالات من جهاز آخر (دمج)
// يحفظ الإدخالات الجديدة على القرص أيضاً
func (j *SessionJournal) Import(entries []JournalEntry) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	existingIDs := make(map[string]bool, len(j.entries))
	for _, e := range j.entries {
		existingIDs[e.ID] = true
	}

	for _, e := range entries {
		if !existingIDs[e.ID] {
			e.SessionID = j.sessionID
			j.entries = append(j.entries, e)
			existingIDs[e.ID] = true
			if j.autoSave {
				if err := j.saveEntry(e); err != nil {
					// [SAFETY] تسجيل الخطأ ولكن نكمل الاستيراد
					// في التطبيق الحقيقي، يمكن استخدام logger هنا
				}
			}
		}
	}
	return nil
}

// Size يرجع عدد الإدخالات
func (j *SessionJournal) Size() int {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return len(j.entries)
}

// LastN يرجع آخر N إدخال
func (j *SessionJournal) LastN(n int) []JournalEntry {
	j.mu.RLock()
	defer j.mu.RUnlock()

	if n <= 0 || n > len(j.entries) {
		n = len(j.entries)
	}
	start := len(j.entries) - n
	result := make([]JournalEntry, n)
	copy(result, j.entries[start:])
	return result
}

// SummaryHTML يُولّد تقرير HTML للسجل
func (j *SessionJournal) SummaryHTML() string {
	j.mu.RLock()
	defer j.mu.RUnlock()

	html := fmt.Sprintf(`<html><head><meta charset="utf-8"><title>سجل الجلسة %s</title>
<style>body{font-family:sans-serif;margin:20px}.entry{margin:8px 0;padding:8px;border-left:3px solid #007bff;background:#f8f9fa}
.timestamp{color:#6c757d;font-size:0.85em}.type{font-weight:bold;color:#007bff}</style></head><body>
<h1>سجل الجلسة: %s</h1><p>إجمالي %d إدخال</p><div id="journal">`,
		j.sessionID[:8], j.sessionID, len(j.entries))

	for _, e := range j.entries {
		html += fmt.Sprintf(`<div class="entry">
<div class="timestamp">%s</div>
<div class="type">[%s]</div>
<div>%s</div>
<div style="font-size:0.85em;color:#495057">المصدر: %s (%s)</div>
</div>`, e.Timestamp.Format("2006-01-02 15:04:05"), e.Type, e.Summary, e.SourceID, e.SourceType)
	}

	html += `</div></body></html>`
	return html
}

// LoadFromDisk يحمّل السجل من القرص
func (j *SessionJournal) LoadFromDisk() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	f, err := os.Open(j.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to open journal file: %w", err)
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	for decoder.More() {
		var entry JournalEntry
		if err := decoder.Decode(&entry); err != nil {
			// [SAFETY] تسجيل الخطأ ولكن نكمل تحميل الإدخالات الصالحة
			// في التطبيق الحقيقي، يمكن استخدام logger هنا
			continue
		}
		entry.SessionID = j.sessionID
		j.entries = append(j.entries, entry)
	}
	return nil
}

// GetStats يرجع إحصائيات السجل
func (j *SessionJournal) GetStats() map[string]interface{} {
	j.mu.RLock()
	defer j.mu.RUnlock()

	typeCounts := make(map[string]int)
	for _, e := range j.entries {
		typeCounts[string(e.Type)]++
	}

	return map[string]interface{}{
		"total_entries": len(j.entries),
		"first_entry":   j.entries[0].Timestamp,
		"last_entry":    j.entries[len(j.entries)-1].Timestamp,
		"types":         typeCounts,
		"file_path":     j.filePath,
	}
}

// String implements fmt.Stringer
func (j *SessionJournal) String() string {
	return fmt.Sprintf("SessionJournal[%s: %d entries]", j.sessionID[:8], j.Size())
}

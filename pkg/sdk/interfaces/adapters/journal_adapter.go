package adapters

import (
	"encoding/json"

	"github.com/MortalArena/Musketeers/pkg/sdk/interfaces"
	"github.com/MortalArena/Musketeers/pkg/session"
)

type JournalAdapter struct {
	journal *session.SessionJournal
}

func NewJournalAdapter(j *session.SessionJournal) *JournalAdapter {
	return &JournalAdapter{journal: j}
}

func (a *JournalAdapter) Record(entry *interfaces.JournalEntry) error {
	sourceType := entry.Source
	if sourceType == "" {
		sourceType = "system"
	}
	summary := ""
	if data, ok := entry.Data["summary"]; ok {
		summary, _ = data.(string)
	}
	var entryType session.JournalEntryType
	if entry.Type != "" {
		entryType = session.JournalEntryType(entry.Type)
	} else {
		entryType = session.JournalEventLogged
	}
	a.journal.Append(entryType, entry.Source, sourceType, summary, entry.Data)
	return nil
}

func (a *JournalAdapter) Query(filter map[string]interface{}) ([]*interfaces.JournalEntry, error) {
	if filter == nil {
		all := a.journal.All()
		return convertJournalEntries(all), nil
	}
	typeStr, _ := filter["type"].(string)
	limit := 0
	if l, ok := filter["limit"].(float64); ok {
		limit = int(l)
	}
	source, _ := filter["source"].(string)
	var results []session.JournalEntry
	if source != "" {
		results = a.journal.QueryBySource(source, limit)
	} else if typeStr != "" {
		results = a.journal.Query(session.JournalEntryType(typeStr), limit)
	} else {
		all := a.journal.All()
		if limit > 0 && len(all) > limit {
			all = all[len(all)-limit:]
		}
		results = all
	}
	return convertJournalEntries(results), nil
}

func (a *JournalAdapter) Recent(n int) ([]*interfaces.JournalEntry, error) {
	entries := a.journal.LastN(n)
	return convertJournalEntries(entries), nil
}

func convertJournalEntries(entries []session.JournalEntry) []*interfaces.JournalEntry {
	result := make([]*interfaces.JournalEntry, len(entries))
	for i, e := range entries {
		details := make(map[string]interface{})
		if e.Details != nil {
			if err := json.Unmarshal(e.Details, &details); err != nil {
				details["raw"] = string(e.Details)
			}
		}
		details["summary"] = e.Summary
		result[i] = &interfaces.JournalEntry{
			ID:        e.ID,
			Type:      string(e.Type),
			Source:    e.SourceID,
			Data:      details,
			Timestamp: e.Timestamp,
		}
	}
	return result
}

var _ interfaces.JournalInterface = (*JournalAdapter)(nil)

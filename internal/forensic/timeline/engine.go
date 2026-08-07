// Package timeline implements the Unified MACB Timeline Engine.
// Correlates timestamps from Registry, EVTX, and $MFT sources into a single chronological pipeline.
package timeline

import (
	"sort"
	"time"
)

// EventSource identifies where the timeline event originated.
type EventSource string

const (
	SourceRegistry EventSource = "registry"
	SourceEVTX     EventSource = "evtx"
	SourceMFT      EventSource = "mft"
)

// EventType identifies the type of event.
type EventType string

const (
	EventFileCreated    EventType = "file_created"
	EventFileModified   EventType = "file_modified"
	EventFileAccessed   EventType = "file_accessed"
	EventFileChanged    EventType = "file_changed"
	EventLogon          EventType = "logon"
	EventLogoff         EventType = "logoff"
	EventProcessCreated EventType = "process_created"
	EventServiceCreated EventType = "service_created"
	EventRegistryChange EventType = "registry_change"
	EventUserCreated    EventType = "user_created"
	EventAuditLogClear  EventType = "audit_log_clear"
)

// TimelineEntry represents a single event in the unified timeline.
type TimelineEntry struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	Source      EventSource            `json:"source"`
	EventType   EventType              `json:"event_type"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Path        string                 `json:"path,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// Timeline represents the unified MACB timeline.
type Timeline struct {
	Entries    []TimelineEntry `json:"entries"`
	StartTime  time.Time       `json:"start_time"`
	EndTime    time.Time       `json:"end_time"`
	Statistics TimelineStats   `json:"statistics"`
}

// TimelineStats contains summary statistics.
type TimelineStats struct {
	SourceCounts   map[EventSource]int `json:"source_counts"`
	TypeCounts     map[EventType]int   `json:"type_counts"`
	HourlyActivity map[int]int         `json:"hourly_activity"`
	DailyActivity  map[string]int      `json:"daily_activity"`
}

// NewTimeline creates a new empty timeline.
func NewTimeline() *Timeline {
	return &Timeline{
		Entries: make([]TimelineEntry, 0),
		Statistics: TimelineStats{
			SourceCounts:   make(map[EventSource]int),
			TypeCounts:     make(map[EventType]int),
			HourlyActivity: make(map[int]int),
			DailyActivity:  make(map[string]int),
		},
	}
}

// AddEntry adds a timeline entry and updates statistics.
func (t *Timeline) AddEntry(entry TimelineEntry) {
	t.Entries = append(t.Entries, entry)

	if t.StartTime.IsZero() || entry.Timestamp.Before(t.StartTime) {
		t.StartTime = entry.Timestamp
	}
	if t.EndTime.IsZero() || entry.Timestamp.After(t.EndTime) {
		t.EndTime = entry.Timestamp
	}

	t.Statistics.SourceCounts[entry.Source]++
	t.Statistics.TypeCounts[entry.EventType]++
	t.Statistics.HourlyActivity[entry.Timestamp.Hour()]++
	t.Statistics.DailyActivity[entry.Timestamp.Format("2006-01-02")]++
}

// AddMFTEntry adds a file system event from MFT.
func (t *Timeline) AddMFTEntry(filename string, modTime, accessTime, changeTime, birthTime time.Time, eventType EventType) {
	entry := TimelineEntry{
		ID:          "mft:" + filename,
		Timestamp:   modTime,
		Source:      SourceMFT,
		EventType:   eventType,
		Title:       eventTypeTitle(eventType),
		Description: filename,
		Path:        filename,
		Data: map[string]interface{}{
			"accessed":  accessTime.Format(time.RFC3339),
			"changed":   changeTime.Format(time.RFC3339),
			"born":      birthTime.Format(time.RFC3339),
		},
	}
	t.AddEntry(entry)
}

// AddEVTXEntry adds an event log entry from EVTX.
func (t *Timeline) AddEVTXEntry(eventID uint16, timestamp time.Time, description string, provider string) {
	eventType := EventLogon
	switch {
	case eventID == 4624 || eventID == 4625:
		eventType = EventLogon
	case eventID == 4634:
		eventType = EventLogoff
	case eventID == 4688:
		eventType = EventProcessCreated
	case eventID == 7045:
		eventType = EventServiceCreated
	case eventID == 1102:
		eventType = EventAuditLogClear
	case eventID == 4720:
		eventType = EventUserCreated
	}

	entry := TimelineEntry{
		ID:          "evtx:" + string(rune(eventID)),
		Timestamp:   timestamp,
		Source:      SourceEVTX,
		EventType:   eventType,
		Title:       eventTypeTitle(eventType),
		Description: description,
		Data: map[string]interface{}{
			"event_id": eventID,
			"provider": provider,
		},
	}
	t.AddEntry(entry)
}

// AddRegistryEntry adds a registry event.
func (t *Timeline) AddRegistryEntry(keyPath string, timestamp time.Time, eventType EventType) {
	entry := TimelineEntry{
		ID:          "reg:" + keyPath,
		Timestamp:   timestamp,
		Source:      SourceRegistry,
		EventType:   eventType,
		Title:       eventTypeTitle(eventType),
		Description: keyPath,
		Path:        keyPath,
	}
	t.AddEntry(entry)
}

// Sort sorts the timeline by timestamp.
func (t *Timeline) Sort() {
	sort.Slice(t.Entries, func(i, j int) bool {
		return t.Entries[i].Timestamp.Before(t.Entries[j].Timestamp)
	})
}

// FilterByTimeRange filters entries to a specific time range.
func (t *Timeline) FilterByTimeRange(start, end time.Time) *Timeline {
	result := NewTimeline()
	for _, entry := range t.Entries {
		if (entry.Timestamp.After(start) || entry.Timestamp.Equal(start)) &&
			(entry.Timestamp.Before(end) || entry.Timestamp.Equal(end)) {
			result.AddEntry(entry)
		}
	}
	return result
}

// FilterBySource filters entries by event source.
func (t *Timeline) FilterBySource(source EventSource) *Timeline {
	result := NewTimeline()
	for _, entry := range t.Entries {
		if entry.Source == source {
			result.AddEntry(entry)
		}
	}
	return result
}

// GetActivityByHour returns activity counts by hour of day.
func (t *Timeline) GetActivityByHour() map[int]int {
	return t.Statistics.HourlyActivity
}

// GetSourceBreakdown returns event counts by source.
func (t *Timeline) GetSourceBreakdown() map[EventSource]int {
	return t.Statistics.SourceCounts
}

func eventTypeTitle(eventType EventType) string {
	switch eventType {
	case EventFileCreated:
		return "File Created"
	case EventFileModified:
		return "File Modified"
	case EventFileAccessed:
		return "File Accessed"
	case EventFileChanged:
		return "File Changed"
	case EventLogon:
		return "User Logon"
	case EventLogoff:
		return "User Logoff"
	case EventProcessCreated:
		return "Process Created"
	case EventServiceCreated:
		return "Service Installed"
	case EventRegistryChange:
		return "Registry Modified"
	case EventUserCreated:
		return "User Account Created"
	case EventAuditLogClear:
		return "Audit Log Cleared"
	default:
		return "Unknown Event"
	}
}

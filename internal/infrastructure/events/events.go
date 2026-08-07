// Package events implements the event system for the Virtual Disk Intelligence Platform.
// All components communicate through events, never directly.
package events

import (
	"sync"
)

// Event represents a system event.
type Event struct {
	Type    EventType
	Payload interface{}
}

// EventType represents the type of event.
type EventType string

const (
	// Disk events
	EventDiskOpened       EventType = "disk.opened"
	EventDiskClosed       EventType = "disk.closed"
	EventDiskError        EventType = "disk.error"
	EventDiskValidating   EventType = "disk.validating"
	EventDiskValidated    EventType = "disk.validated"
	EventDiskDetecting    EventType = "disk.detecting"
	EventDiskDetected     EventType = "disk.detected"

	// Partition events
	EventPartitionScanning   EventType = "partition.scanning"
	EventPartitionFound      EventType = "partition.found"
	EventPartitionSelected   EventType = "partition.selected"
	EventPartitionError      EventType = "partition.error"

	// Filesystem events
	EventFilesystemDetecting EventType = "filesystem.detecting"
	EventFilesystemDetected  EventType = "filesystem.detected"
	EventFilesystemError     EventType = "filesystem.error"

	// File events
	EventFolderLoading   EventType = "folder.loading"
	EventFolderLoaded    EventType = "folder.loaded"
	EventFileLoading     EventType = "file.loading"
	EventFileLoaded      EventType = "file.loaded"
	EventFileError       EventType = "file.error"

	// Preview events
	EventPreviewGenerating EventType = "preview.generating"
	EventPreviewReady      EventType = "preview.ready"
	EventPreviewError      EventType = "preview.error"

	// Search events
	EventSearchStarting  EventType = "search.starting"
	EventSearchProgress  EventType = "search.progress"
	EventSearchComplete  EventType = "search.complete"
	EventSearchError     EventType = "search.error"

	// Extraction events
	EventExtractStarting  EventType = "extract.starting"
	EventExtractProgress  EventType = "extract.progress"
	EventExtractComplete  EventType = "extract.complete"
	EventExtractError     EventType = "extract.error"

	// Cache events
	EventCacheHit    EventType = "cache.hit"
	EventCacheMiss   EventType = "cache.miss"
	EventCacheEvict  EventType = "cache.evict"

	// Index events
	EventIndexProgress EventType = "index.progress"
	EventIndexComplete EventType = "index.complete"

	// Smart Open events
	EventSmartOpenStarting EventType = "smartopen.starting"
	EventSmartOpenProgress EventType = "smartopen.progress"
	EventSmartOpenComplete EventType = "smartopen.complete"
	EventSmartOpenError    EventType = "smartopen.error"
)

// Handler is a function that handles an event.
type Handler func(Event)

// Bus is the event bus that routes events between components.
type Bus struct {
	mu       sync.RWMutex
	handlers map[EventType][]Handler
}

// NewBus creates a new event bus.
func NewBus() *Bus {
	return &Bus{
		handlers: make(map[EventType][]Handler),
	}
}

// Subscribe registers a handler for an event type.
func (b *Bus) Subscribe(eventType EventType, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Publish sends an event to all subscribed handlers.
func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	handlers := b.handlers[event.Type]
	b.mu.RUnlock()

	for _, h := range handlers {
		go h(event)
	}
}

// PublishSync sends an event synchronously.
func (b *Bus) PublishSync(event Event) {
	b.mu.RLock()
	handlers := b.handlers[event.Type]
	b.mu.RUnlock()

	for _, h := range handlers {
		h(event)
	}
}

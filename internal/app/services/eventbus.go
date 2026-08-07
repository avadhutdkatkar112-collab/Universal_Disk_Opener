package services

import (
	"sync"
	"time"
)

type Event struct {
	Type      string    `json:"type"`
	Source    string    `json:"source"`
	Payload   any       `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string][]chan Event
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]chan Event),
	}
}

func (eb *EventBus) Subscribe(eventType string) <-chan Event {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	ch := make(chan Event, 100)
	eb.subscribers[eventType] = append(eb.subscribers[eventType], ch)
	return ch
}

func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if subs, ok := eb.subscribers[event.Type]; ok {
		for _, ch := range subs {
			select {
			case ch <- event:
			default:
			}
		}
	}
}

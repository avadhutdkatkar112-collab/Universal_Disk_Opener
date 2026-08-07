package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/user/vhd-opener/internal/jobs"
)

type Gateway struct {
	mu           sync.RWMutex
	capabilities map[string]jobs.Capability
	eventBus     *EventBus
	jobManager   *JobManager
}

func NewGateway(eb *EventBus, jm *JobManager) *Gateway {
	return &Gateway{
		capabilities: make(map[string]jobs.Capability),
		eventBus:     eb,
		jobManager:   jm,
	}
}

func (g *Gateway) RegisterCapability(cap jobs.Capability) {
	g.mu.Lock()
	defer g.mu.Unlock()

	meta := cap.Metadata()
	g.capabilities[meta.ID] = cap

	g.eventBus.Publish(Event{
		Type:      "CAPABILITY_REGISTERED",
		Source:    "Gateway",
		Payload:   meta,
		Timestamp: time.Now(),
	})
}

func (g *Gateway) GetCapability(id string) (jobs.Capability, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	cap, ok := g.capabilities[id]
	return cap, ok
}

func (g *Gateway) ListCapabilities() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	ids := make([]string, 0, len(g.capabilities))
	for id := range g.capabilities {
		ids = append(ids, id)
	}
	return ids
}

func (g *Gateway) Dispatch(ctx context.Context, capabilityID string, execCtx jobs.ExecutionContext) (*Job, error) {
	g.mu.RLock()
	cap, exists := g.capabilities[capabilityID]
	g.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("capability not registered: %s", capabilityID)
	}

	if err := cap.Validate(execCtx); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	job := g.jobManager.Submit(ctx, cap, execCtx)
	return job, nil
}

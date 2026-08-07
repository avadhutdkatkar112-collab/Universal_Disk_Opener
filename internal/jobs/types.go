package jobs

import "context"

type CapabilityType string

const (
	TypeAnalysis CapabilityType = "analysis"
	TypeSearch   CapabilityType = "search"
	TypeHash     CapabilityType = "hash"
)

type Metadata struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Type        CapabilityType `json:"type"`
	Description string         `json:"description"`
	Permissions []string       `json:"permissions,omitempty"`
	Version     string         `json:"version,omitempty"`
}

type ExecutionContext struct {
	SessionID       string         `json:"session_id,omitempty"`
	ActivePartition string         `json:"active_partition,omitempty"`
	CurrentPath     string         `json:"current_path,omitempty"`
	Params          map[string]any `json:"params"`
}

type Capability interface {
	Metadata() Metadata
	Validate(ctx ExecutionContext) error
	Execute(ctx context.Context, execCtx ExecutionContext, progress chan<- float64) (any, error)
}

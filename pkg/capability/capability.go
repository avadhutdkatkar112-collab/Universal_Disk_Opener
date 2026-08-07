package capability

import (
	"context"
)

type CapabilityType string

const (
	TypeSearch   CapabilityType = "SEARCH"
	TypeAnalysis CapabilityType = "ANALYSIS"
	TypeRecovery CapabilityType = "RECOVERY"
	TypePreview  CapabilityType = "PREVIEW"
	TypeVFS      CapabilityType = "VFS"
)

type Metadata struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Type        CapabilityType `json:"type"`
	Description string         `json:"description"`
	Permissions []string       `json:"permissions"`
}

type ExecutionContext struct {
	SessionID       string         `json:"session_id"`
	ActivePartition string         `json:"active_partition"`
	CurrentPath     string         `json:"current_path"`
	Params          map[string]any `json:"params"`
}

type Capability interface {
	Metadata() Metadata
	Validate(execCtx ExecutionContext) error
	Execute(ctx context.Context, execCtx ExecutionContext, progressChan chan<- float64) (any, error)
}

type BaseCapability struct {
	Meta Metadata
}

func (b *BaseCapability) Metadata() Metadata {
	return b.Meta
}

func (b *BaseCapability) Validate(execCtx ExecutionContext) error {
	return nil
}

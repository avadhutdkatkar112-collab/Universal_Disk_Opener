// Package die implements the Disk Intelligence Engine — a deterministic,
// zero-latency, local command palette for the Universal Disk Explorer.
// No internet, no LLM, no APIs. 100% local parsing in <10ms.
package die

import "context"

// ActionType enumerates all supported command categories.
type ActionType string

const (
	ActionSearch    ActionType = "SEARCH"
	ActionNavigate  ActionType = "NAVIGATE"
	ActionAnalyze   ActionType = "ANALYZE"
	ActionExtract   ActionType = "EXTRACT"
	ActionPreview   ActionType = "PREVIEW"
	ActionCompare   ActionType = "COMPARE"
	ActionHash      ActionType = "HASH"
	ActionExport    ActionType = "EXPORT"
	ActionRecovery  ActionType = "RECOVERY"
	ActionReport    ActionType = "REPORT"
	ActionSettings  ActionType = "SETTINGS"
	ActionUnknown   ActionType = "UNKNOWN"
)

// Intent contains the parsed execution plan from natural language input.
type Intent struct {
	Action     ActionType         `json:"action"`
	Query      string             `json:"query,omitempty"`
	Filters    map[string]string  `json:"filters,omitempty"`
	Target     string             `json:"target,omitempty"`
	Params     map[string]string  `json:"params,omitempty"`
	RawCommand string             `json:"raw_command"`
	Confidence float64            `json:"confidence"` // 0.0 - 1.0
}

// CommandContext holds active application state for context-aware dispatch.
type CommandContext struct {
	ActivePartition  string `json:"active_partition"`
	CurrentPath      string `json:"current_path"`
	SelectedFile     string `json:"selected_file,omitempty"`
	SelectedFiles    []string `json:"selected_files,omitempty"`
	ActiveTab        string `json:"active_tab"`
	TotalFiles       uint64 `json:"total_files"`
	TotalPartitions  int    `json:"total_partitions"`
	DiskFormat       string `json:"disk_format"`
	FilesystemType   string `json:"filesystem_type"`
}

// CommandHandler is a function that executes an intent within a context.
type CommandHandler func(ctx context.Context, intent Intent, cmdCtx CommandContext) (interface{}, error)

// CommandRegistration defines a single command in the DIE registry.
type CommandRegistration struct {
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	Category     string         `json:"category"`
	Keywords     []string       `json:"keywords"`
	Aliases      []string       `json:"aliases,omitempty"`
	Action       ActionType     `json:"action"`
	Handler      CommandHandler `json:"-"`
	NeedsConfirm bool           `json:"needs_confirm"`
}

// CommandResult wraps a command execution result with metadata.
type CommandResult struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Suggestion is a single autocomplete entry for the command palette.
type Suggestion struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Icon        string `json:"icon,omitempty"`
	Score       float64 `json:"score"` // Relevance score for fuzzy matching
}

// HistoryEntry records a previously executed command.
type HistoryEntry struct {
	Command   string `json:"command"`
	Timestamp int64  `json:"timestamp"`
	Success   bool   `json:"success"`
}

// FavoriteEntry is a user-pinned command.
type FavoriteEntry struct {
	Command     string `json:"command"`
	Label       string `json:"label"`
	Icon        string `json:"icon,omitempty"`
}

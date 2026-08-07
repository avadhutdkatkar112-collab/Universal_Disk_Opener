package services

import "context"

type CommandContext struct {
	SessionID string
	Workspace *Workspace
}

type HandlerDeps struct {
	Workspace         *Workspace
	Gateway           *Gateway
	EventBus          *EventBus
	SearchFunc        func(query string, filters map[string]string, path string) ([]map[string]any, error)
	ListDirFunc       func(path string) ([]map[string]any, error)
	GetDiskInfoFunc   func() (any, error)
	GetPartitionsFunc func() ([]map[string]any, error)
	HashFunc          func(path string, algo string) (string, error)
	ExportFunc        func(format string, data any) (string, error)
	PreviewFunc       func(path string) (any, error)
	ExtractFunc       func(paths []string, dest string) (string, error)
}

type Intent struct {
	Command    string         `json:"command"`
	Action     string         `json:"action"`
	Parameters map[string]any `json:"parameters"`
	Confidence float64        `json:"confidence"`
}

type CommandHistoryEntry struct {
	Command   string `json:"command"`
	Timestamp string `json:"timestamp"`
}

type FavoriteEntry struct {
	Command string `json:"command"`
	Label   string `json:"label"`
}

type Suggestion struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

type DieRegistry struct {
	handlers map[string]func(context.Context, string, CommandContext) (any, error)
}

func NewDieRegistry() *DieRegistry {
	return &DieRegistry{handlers: make(map[string]func(context.Context, string, CommandContext) (any, error))}
}

func RegisterDefaultHandlers(r *DieRegistry, deps *HandlerDeps) {}

func (r *DieRegistry) Execute(ctx context.Context, command string, cmdCtx CommandContext) (any, error) {
	return nil, nil
}

func (r *DieRegistry) GetSuggestions(query string, cmdCtx CommandContext) []Suggestion {
	return nil
}

func (r *DieRegistry) ParseOnly(command string, cmdCtx CommandContext) (*Intent, error) {
	return &Intent{Command: command}, nil
}

func (r *DieRegistry) GetHistory() []CommandHistoryEntry {
	return nil
}

func (r *DieRegistry) GetFavorites() []FavoriteEntry {
	return nil
}

func (r *DieRegistry) AddFavorite(command, label string) {}

func (r *DieRegistry) RemoveFavorite(command string) {}

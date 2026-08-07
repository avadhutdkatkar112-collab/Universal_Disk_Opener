package die

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// TaskStatus represents the state of an async command execution.
type TaskStatus string

const (
	TaskPending    TaskStatus = "pending"
	TaskRunning    TaskStatus = "running"
	TaskCompleted  TaskStatus = "completed"
	TaskFailed     TaskStatus = "failed"
	TaskCancelled  TaskStatus = "cancelled"
)

// Task represents an async command execution.
type Task struct {
	ID        string                 `json:"id"`
	Status    TaskStatus             `json:"status"`
	Intent    *Intent                `json:"intent"`
	Result    interface{}            `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Progress  float64                `json:"progress"` // 0.0 - 1.0
	StartedAt time.Time              `json:"started_at"`
	EndedAt   *time.Time             `json:"ended_at,omitempty"`
	cancel    context.CancelFunc
}

// Registry is the central command registry for the Disk Intelligence Engine.
type Registry struct {
	mu       sync.RWMutex
	commands map[ActionType]CommandRegistration
	parser   *CommandTrie
	history  []HistoryEntry
	favs     []FavoriteEntry
	tasks    map[string]*Task
	taskMu   sync.RWMutex
}

// NewRegistry creates a new DIE registry with Trie-based autocomplete.
func NewRegistry() *Registry {
	return &Registry{
		commands: make(map[ActionType]CommandRegistration),
		parser:   BuildDefaultTrie(),
		history:  make([]HistoryEntry, 0, 64),
		favs:     make([]FavoriteEntry, 0, 16),
		tasks:    make(map[string]*Task),
	}
}

// Register adds a command to the registry.
func (r *Registry) Register(cmd CommandRegistration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands[cmd.Action] = cmd
}

// Execute parses input and runs the handler synchronously.
func (r *Registry) Execute(ctx context.Context, input string, cmdCtx CommandContext) (interface{}, error) {
	intent, err := ParseInput(input, cmdCtx)
	if err != nil {
		return nil, fmt.Errorf("die: parse error: %w", err)
	}

	if intent.Action == ActionUnknown || intent.Action == "" {
		return r.fallbackSearch(ctx, input, cmdCtx)
	}

	r.mu.RLock()
	cmd, exists := r.commands[intent.Action]
	r.mu.RUnlock()

	if !exists {
		return r.fallbackSearch(ctx, input, cmdCtx)
	}

	r.recordHistory(input, true)

	if cmd.NeedsConfirm {
		return map[string]interface{}{
			"action":  "confirm_required",
			"command": cmd.Title,
			"message": fmt.Sprintf("This action requires confirmation: %s", cmd.Description),
			"intent":  intent,
		}, nil
	}

	return cmd.Handler(ctx, *intent, cmdCtx)
}

// ExecuteAsync runs a command in the background and returns a task ID.
func (r *Registry) ExecuteAsync(input string, cmdCtx CommandContext) (string, error) {
	intent, err := ParseInput(input, cmdCtx)
	if err != nil {
		return "", fmt.Errorf("die: parse error: %w", err)
	}

	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())

	task := &Task{
		ID:        taskID,
		Status:    TaskPending,
		Intent:    intent,
		StartedAt: time.Now(),
		cancel:    cancel,
	}

	r.taskMu.Lock()
	r.tasks[taskID] = task
	r.taskMu.Unlock()

	r.mu.RLock()
	cmd, exists := r.commands[intent.Action]
	r.mu.RUnlock()

	go func() {
		task.Status = TaskRunning
		defer func() {
			now := time.Now()
			task.EndedAt = &now
		}()

		if !exists {
			task.Status = TaskFailed
			task.Error = fmt.Sprintf("no handler for action: %s", intent.Action)
			return
		}

		result, err := cmd.Handler(ctx, *intent, cmdCtx)
		if ctx.Err() != nil {
			task.Status = TaskCancelled
			task.Error = "cancelled"
			return
		}
		if err != nil {
			task.Status = TaskFailed
			task.Error = err.Error()
			return
		}

		task.Status = TaskCompleted
		task.Result = result
		task.Progress = 1.0
	}()

	return taskID, nil
}

// GetTask returns the status of an async task.
func (r *Registry) GetTask(taskID string) (*Task, bool) {
	r.taskMu.RLock()
	defer r.taskMu.RUnlock()
	task, exists := r.tasks[taskID]
	return task, exists
}

// CancelTask cancels a running async task.
func (r *Registry) CancelTask(taskID string) bool {
	r.taskMu.RLock()
	defer r.taskMu.RUnlock()
	task, exists := r.tasks[taskID]
	if !exists || task.cancel == nil {
		return false
	}
	task.cancel()
	return true
}

// ParseOnly parses input without executing. Used for preview/suggestions.
func (r *Registry) ParseOnly(input string, cmdCtx CommandContext) (*Intent, error) {
	return ParseInput(input, cmdCtx)
}

// GetSuggestions returns autocomplete suggestions using the Trie.
func (r *Registry) GetSuggestions(partial string, cmdCtx CommandContext) []Suggestion {
	if partial == "" {
		return r.getDefaultSuggestions(cmdCtx)
	}
	return r.parser.SearchPrefix(partial)
}

// FuzzySearch performs Levenshtein-based fuzzy matching for typos.
func (r *Registry) FuzzySearch(query string) []Suggestion {
	return r.parser.FuzzySearch(query, 2)
}

// GetCommand returns the registration for an action type.
func (r *Registry) GetCommand(action ActionType) (CommandRegistration, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cmd, exists := r.commands[action]
	return cmd, exists
}

// ListCommands returns all registered commands sorted by category.
func (r *Registry) ListCommands() []CommandRegistration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cmds := make([]CommandRegistration, 0, len(r.commands))
	for _, cmd := range r.commands {
		cmds = append(cmds, cmd)
	}

	sort.Slice(cmds, func(i, j int) bool {
		if cmds[i].Category != cmds[j].Category {
			return cmds[i].Category < cmds[j].Category
		}
		return cmds[i].Title < cmds[j].Title
	})

	return cmds
}

// GetHistory returns the command execution history.
func (r *Registry) GetHistory() []HistoryEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]HistoryEntry, len(r.history))
	copy(result, r.history)
	return result
}

// AddFavorite pins a command.
func (r *Registry) AddFavorite(cmd string, label string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.favs = append(r.favs, FavoriteEntry{Command: cmd, Label: label})
}

// GetFavorites returns pinned commands.
func (r *Registry) GetFavorites() []FavoriteEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]FavoriteEntry, len(r.favs))
	copy(result, r.favs)
	return result
}

// RemoveFavorite removes a pinned command by label.
func (r *Registry) RemoveFavorite(label string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, fav := range r.favs {
		if fav.Label == label {
			r.favs = append(r.favs[:i], r.favs[i+1:]...)
			return
		}
	}
}

// fallbackSearch is the deterministic fallback when no command matches.
func (r *Registry) fallbackSearch(ctx context.Context, input string, cmdCtx CommandContext) (interface{}, error) {
	intent := &Intent{
		Action:     ActionSearch,
		Query:      input,
		Target:     cmdCtx.CurrentPath,
		Filters:    make(map[string]string),
		RawCommand: input,
		Confidence: 0.3,
	}

	r.mu.RLock()
	cmd, exists := r.commands[ActionSearch]
	r.mu.RUnlock()

	if exists {
		return cmd.Handler(ctx, *intent, cmdCtx)
	}

	return map[string]interface{}{
		"action":  "search",
		"query":   input,
		"message": "Fallback search initiated",
	}, nil
}

func (r *Registry) recordHistory(input string, success bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.history = append(r.history, HistoryEntry{
		Command: input,
		Success: success,
	})

	if len(r.history) > 50 {
		r.history = r.history[len(r.history)-50:]
	}
}

func (r *Registry) getDefaultSuggestions(cmdCtx CommandContext) []Suggestion {
	var suggestions []Suggestion

	suggestions = append(suggestions,
		Suggestion{Title: "Find Files", Description: "Search for files", Category: "Search", Score: 1.0},
		Suggestion{Title: "Show Largest Files", Description: "Display largest files", Category: "Analyze", Score: 0.9},
		Suggestion{Title: "Show Disk Info", Description: "View disk details", Category: "Analyze", Score: 0.85},
		Suggestion{Title: "Show Partitions", Description: "View partition layout", Category: "Analyze", Score: 0.8},
	)

	if cmdCtx.SelectedFile != "" {
		suggestions = append(suggestions,
			Suggestion{Title: "Preview " + cmdCtx.SelectedFile, Description: "Preview selected file", Category: "Preview", Score: 0.95},
			Suggestion{Title: "Hash " + cmdCtx.SelectedFile, Description: "Calculate file hash", Category: "Hash", Score: 0.9},
		)
	}

	if cmdCtx.CurrentPath != "" {
		suggestions = append(suggestions,
			Suggestion{Title: "Search in " + cmdCtx.CurrentPath, Description: "Search current folder", Category: "Search", Score: 0.88},
		)
	}

	return suggestions
}

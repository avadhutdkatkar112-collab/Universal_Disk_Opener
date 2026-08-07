package platform

import (
	"fmt"
	"sync"
	"time"
)

type WorkspaceTarget struct {
	ID          string   `json:"id"`
	ImagePath   string   `json:"image_path"`
	Format      string   `json:"format"`
	Partitions  []string `json:"partitions"`
	IsEncrypted bool     `json:"is_encrypted"`
}

type Bookmark struct {
	ID        string    `json:"id"`
	TargetID  string    `json:"target_id"`
	Path      string    `json:"path"`
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"created_at"`
}

type HistoryEntry struct {
	TargetID    string    `json:"target_id"`
	Partition   string    `json:"partition"`
	Path        string    `json:"path"`
	Timestamp   time.Time `json:"timestamp"`
}

type WorkspaceState struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Targets         []WorkspaceTarget `json:"targets"`
	ActiveTargetID  string            `json:"active_target_id"`
	ActivePartition string            `json:"active_partition"`
	Bookmarks       []Bookmark        `json:"bookmarks"`
	OpenedTabs      []string          `json:"opened_tabs"`
	ActiveTabIndex  int               `json:"active_tab_index"`
	History         []HistoryEntry    `json:"history"`
}

type WorkspaceManager struct {
	mu       sync.RWMutex
	state    WorkspaceState
	eventBus *EventBus
}

func NewWorkspaceManager(eb *EventBus) *WorkspaceManager {
	return &WorkspaceManager{
		state: WorkspaceState{
			ID:         fmt.Sprintf("ws_%d", time.Now().Unix()),
			Name:       "Default Workspace",
			Targets:    make([]WorkspaceTarget, 0),
			Bookmarks:  make([]Bookmark, 0),
			OpenedTabs: []string{"/"},
		},
		eventBus: eb,
	}
}

func (wm *WorkspaceManager) MountTarget(target WorkspaceTarget) {
	wm.mu.Lock()
	wm.state.Targets = append(wm.state.Targets, target)
	if wm.state.ActiveTargetID == "" {
		wm.state.ActiveTargetID = target.ID
		wm.state.ActivePartition = ""
	}
	wm.mu.Unlock()

	wm.eventBus.Publish(Event{
		Type:      "WORKSPACE_TARGET_MOUNTED",
		Source:    "WorkspaceManager",
		Payload:   target,
		Timestamp: time.Now(),
	})
}

func (wm *WorkspaceManager) UnmountTarget(targetID string) {
	wm.mu.Lock()
	for i, t := range wm.state.Targets {
		if t.ID == targetID {
			wm.state.Targets = append(wm.state.Targets[:i], wm.state.Targets[i+1:]...)
			break
		}
	}
	if wm.state.ActiveTargetID == targetID {
		wm.state.ActiveTargetID = ""
		if len(wm.state.Targets) > 0 {
			wm.state.ActiveTargetID = wm.state.Targets[0].ID
		}
	}
	wm.mu.Unlock()

	wm.eventBus.Publish(Event{
		Type:      "WORKSPACE_TARGET_UNMOUNTED",
		Source:    "WorkspaceManager",
		Payload:   map[string]any{"target_id": targetID},
		Timestamp: time.Now(),
	})
}

func (wm *WorkspaceManager) GetState() WorkspaceState {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return wm.state
}

func (wm *WorkspaceManager) SetActiveTarget(targetID string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.state.ActiveTargetID = targetID
	wm.state.ActivePartition = ""
	wm.state.ActiveTabIndex = 0
}

func (wm *WorkspaceManager) SetActivePartition(partition string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.state.ActivePartition = partition
}

func (wm *WorkspaceManager) Navigate(path string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	wm.state.History = append(wm.state.History, HistoryEntry{
		TargetID:  wm.state.ActiveTargetID,
		Partition: wm.state.ActivePartition,
		Path:      path,
		Timestamp: time.Now(),
	})

	if len(wm.state.History) > 1000 {
		wm.state.History = wm.state.History[len(wm.state.History)-1000:]
	}

	if wm.state.ActiveTabIndex < len(wm.state.OpenedTabs) {
		wm.state.OpenedTabs[wm.state.ActiveTabIndex] = path
	} else {
		wm.state.OpenedTabs = append(wm.state.OpenedTabs, path)
		wm.state.ActiveTabIndex = len(wm.state.OpenedTabs) - 1
	}
}

func (wm *WorkspaceManager) AddBookmark(label, path string) *Bookmark {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	bm := Bookmark{
		ID:        fmt.Sprintf("bm_%d", time.Now().UnixNano()),
		TargetID:  wm.state.ActiveTargetID,
		Path:      path,
		Label:     label,
		CreatedAt: time.Now(),
	}
	wm.state.Bookmarks = append(wm.state.Bookmarks, bm)
	return &bm
}

func (wm *WorkspaceManager) RemoveBookmark(bookmarkID string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	for i, bm := range wm.state.Bookmarks {
		if bm.ID == bookmarkID {
			wm.state.Bookmarks = append(wm.state.Bookmarks[:i], wm.state.Bookmarks[i+1:]...)
			return
		}
	}
}

func (wm *WorkspaceManager) GetBookmarks() []Bookmark {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return wm.state.Bookmarks
}

func (wm *WorkspaceManager) OpenTab(path string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.state.OpenedTabs = append(wm.state.OpenedTabs, path)
	wm.state.ActiveTabIndex = len(wm.state.OpenedTabs) - 1
}

func (wm *WorkspaceManager) CloseTab(index int) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	if index < 0 || index >= len(wm.state.OpenedTabs) {
		return
	}
	wm.state.OpenedTabs = append(wm.state.OpenedTabs[:index], wm.state.OpenedTabs[index+1:]...)
	if wm.state.ActiveTabIndex >= len(wm.state.OpenedTabs) {
		wm.state.ActiveTabIndex = len(wm.state.OpenedTabs) - 1
	}
	if wm.state.ActiveTabIndex < 0 {
		wm.state.ActiveTabIndex = 0
	}
}

func (wm *WorkspaceManager) SetActiveTab(index int) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	if index >= 0 && index < len(wm.state.OpenedTabs) {
		wm.state.ActiveTabIndex = index
	}
}

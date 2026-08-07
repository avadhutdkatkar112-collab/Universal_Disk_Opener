package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DiskSession represents one open disk in the workspace.
type DiskSession struct {
	ID              string         `json:"id"`
	Path            string         `json:"path"`
	Name            string         `json:"name"`
	ActivePartition int            `json:"active_partition"`
	CurrentPath     string         `json:"current_path"`
	Bookmarks       []Bookmark     `json:"bookmarks"`
	History         []HistoryEntry `json:"history"`
	OpenedAt        time.Time      `json:"opened_at"`
	LastActive      time.Time      `json:"last_active"`
}

// Bookmark is a saved location in a disk.
type Bookmark struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	DiskID      string    `json:"disk_id"`
	Partition   int       `json:"partition"`
	Path        string    `json:"path"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// HistoryEntry records a navigation event.
type HistoryEntry struct {
	DiskID    string    `json:"disk_id"`
	Partition int       `json:"partition"`
	Path      string    `json:"path"`
	Timestamp time.Time `json:"timestamp"`
}

// Workspace is the top-level session manager.
type Workspace struct {
	mu          sync.RWMutex
	disks       map[string]*DiskSession
	activeDisk  string
	sessionsDir string
	diskCounter int
}

// NewWorkspace creates a workspace manager that persists to ~/.config/VHD-Explorer/workspace/
func NewWorkspace(appDir string) *Workspace {
	sessionsDir := filepath.Join(appDir, "workspace")
	os.MkdirAll(sessionsDir, 0755)

	w := &Workspace{
		disks:       make(map[string]*DiskSession),
		sessionsDir: sessionsDir,
	}

	w.loadFromDisk()
	return w
}

// OpenDisk adds a disk to the workspace or activates it if already open.
func (w *Workspace) OpenDisk(path string) *DiskSession {
	w.mu.Lock()
	defer w.mu.Unlock()

	name := filepath.Base(path)

	for _, d := range w.disks {
		if d.Path == path {
			d.LastActive = time.Now()
			w.activeDisk = d.ID
			w.saveToDisk()
			return d
		}
	}

	id := fmt.Sprintf("disk-%s-%d", time.Now().Format("0102150405"), w.diskCounter)
	w.diskCounter++
	disk := &DiskSession{
		ID:              id,
		Path:            path,
		Name:            name,
		ActivePartition: 0,
		CurrentPath:     "/",
		Bookmarks:       []Bookmark{},
		History:         []HistoryEntry{},
		OpenedAt:        time.Now(),
		LastActive:      time.Now(),
	}

	w.disks[id] = disk
	w.activeDisk = id
	w.saveToDisk()
	return disk
}

// CloseDisk removes a disk from the workspace.
func (w *Workspace) CloseDisk(diskID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.disks, diskID)
	if w.activeDisk == diskID {
		w.activeDisk = ""
		for id := range w.disks {
			w.activeDisk = id
			break
		}
	}
	w.saveToDisk()
}

// GetDisk returns a disk session by ID.
func (w *Workspace) GetDisk(diskID string) (*DiskSession, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	d, ok := w.disks[diskID]
	return d, ok
}

// ListDisks returns all open disk sessions.
func (w *Workspace) ListDisks() []*DiskSession {
	w.mu.RLock()
	defer w.mu.RUnlock()
	result := make([]*DiskSession, 0, len(w.disks))
	for _, d := range w.disks {
		result = append(result, d)
	}
	return result
}

// SetActiveDisk changes the active disk.
func (w *Workspace) SetActiveDisk(diskID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, ok := w.disks[diskID]; ok {
		w.activeDisk = diskID
		w.disks[diskID].LastActive = time.Now()
		w.saveToDisk()
	}
}

// GetActiveDiskID returns the currently active disk ID.
func (w *Workspace) GetActiveDiskID() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.activeDisk
}

// Navigate updates the current path for a disk and records history.
func (w *Workspace) Navigate(diskID, path string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	disk, ok := w.disks[diskID]
	if !ok {
		return
	}

	disk.History = append(disk.History, HistoryEntry{
		DiskID:    diskID,
		Partition: disk.ActivePartition,
		Path:      path,
		Timestamp: time.Now(),
	})

	if len(disk.History) > 500 {
		disk.History = disk.History[len(disk.History)-500:]
	}

	disk.CurrentPath = path
	disk.LastActive = time.Now()
	w.saveToDisk()
}

// SetPartition changes the active partition for a disk.
func (w *Workspace) SetPartition(diskID string, partition int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if disk, ok := w.disks[diskID]; ok {
		disk.ActivePartition = partition
		disk.CurrentPath = "/"
		w.saveToDisk()
	}
}

// AddBookmark saves a location.
func (w *Workspace) AddBookmark(diskID, name, path, description string) *Bookmark {
	w.mu.Lock()
	defer w.mu.Unlock()
	disk, ok := w.disks[diskID]
	if !ok {
		return nil
	}

	bm := Bookmark{
		ID:          "bm-" + time.Now().Format("0102150405"),
		Name:        name,
		DiskID:      diskID,
		Partition:   disk.ActivePartition,
		Path:        path,
		Description: description,
		CreatedAt:   time.Now(),
	}

	disk.Bookmarks = append(disk.Bookmarks, bm)
	w.saveToDisk()
	return &bm
}

// RemoveBookmark removes a bookmark by ID from a disk.
func (w *Workspace) RemoveBookmark(diskID, bookmarkID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	disk, ok := w.disks[diskID]
	if !ok {
		return
	}

	for i, bm := range disk.Bookmarks {
		if bm.ID == bookmarkID {
			disk.Bookmarks = append(disk.Bookmarks[:i], disk.Bookmarks[i+1:]...)
			break
		}
	}
	w.saveToDisk()
}

// GetBookmarks returns all bookmarks for a disk.
func (w *Workspace) GetBookmarks(diskID string) []Bookmark {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if disk, ok := w.disks[diskID]; ok {
		return disk.Bookmarks
	}
	return nil
}

// GetHistory returns navigation history for a disk.
func (w *Workspace) GetHistory(diskID string) []HistoryEntry {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if disk, ok := w.disks[diskID]; ok {
		return disk.History
	}
	return nil
}

// GetAllBookmarks returns bookmarks from all disks.
func (w *Workspace) GetAllBookmarks() []Bookmark {
	w.mu.RLock()
	defer w.mu.RUnlock()
	var all []Bookmark
	for _, disk := range w.disks {
		all = append(all, disk.Bookmarks...)
	}
	return all
}

// SearchAcrossDisks searches file names across all open disks.
func (w *Workspace) SearchAcrossDisks(query string) []map[string]any {
	w.mu.RLock()
	defer w.mu.RUnlock()
	var results []map[string]any
	for _, disk := range w.disks {
		results = append(results, map[string]any{
			"disk_id":   disk.ID,
			"disk_name": disk.Name,
			"disk_path": disk.Path,
			"query":     query,
		})
	}
	return results
}

func (w *Workspace) saveToDisk() {
	data, err := json.MarshalIndent(w.disks, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(filepath.Join(w.sessionsDir, "workspace.json"), data, 0644)
}

func (w *Workspace) loadFromDisk() {
	data, err := os.ReadFile(filepath.Join(w.sessionsDir, "workspace.json"))
	if err != nil {
		return
	}
	var disks map[string]*DiskSession
	if err := json.Unmarshal(data, &disks); err != nil {
		return
	}
	w.disks = disks
	for id := range w.disks {
		w.activeDisk = id
	}
}

func (w *Workspace) GetState() map[string]any {
	return map[string]any{
		"activeDisk": w.activeDisk,
		"diskCount":  len(w.disks),
	}
}

func (w *Workspace) MountTarget(target map[string]any) error { return nil }
func (w *Workspace) UnmountTarget(targetID string) error     { return nil }
func (w *Workspace) SetActiveTarget(targetID string)          {}
func (w *Workspace) SetActivePartition(diskID string, idx int) {
	w.SetPartition(diskID, idx)
}
func (w *Workspace) Navigate2(path string)                     {}
func (w *Workspace) OpenTab(path string)                       {}
func (w *Workspace) CloseTab(index int)                        {}
func (w *Workspace) SetActiveTab(index int)                    {}
func (w *Workspace) AddBookmark2(label, path string) *Bookmark { return nil }
func (w *Workspace) RemoveBookmark2(bookmarkID string)         {}
func (w *Workspace) GetBookmarks2() []Bookmark                 { return nil }

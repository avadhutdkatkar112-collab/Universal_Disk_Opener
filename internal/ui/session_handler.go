package ui

import (
	"context"
	"fmt"
	"sync"

	"github.com/user/vhd-opener/pkg/storage"
)

type SessionHandler struct {
	mu       sync.RWMutex
	ctx      context.Context
	sessions map[string]*storage.EvidenceSession
	activeID string
}

func NewSessionHandler() *SessionHandler {
	return &SessionHandler{
		sessions: make(map[string]*storage.EvidenceSession),
	}
}

func (h *SessionHandler) Startup(ctx context.Context) {
	h.ctx = ctx
}

type SessionInfo struct {
	ID          string                `json:"id"`
	State       string                `json:"state"`
	ImagePath   string                `json:"image_path"`
	FileName    string                `json:"file_name"`
	Format      string                `json:"format"`
	TotalSize   uint64                `json:"total_size"`
	IsReadOnly  bool                  `json:"is_read_only"`
	PartitionCnt int                  `json:"partition_count"`
	Filesystem  string                `json:"filesystem"`
	Provenance  []storage.ProvenanceEntry `json:"provenance"`
}

func (h *SessionHandler) OpenSession(imagePath string) (*SessionInfo, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	sessionID := fmt.Sprintf("session-%d", len(h.sessions)+1)
	session := storage.NewEvidenceSession(sessionID)

	if err := session.Open(h.ctx, imagePath); err != nil {
		return nil, fmt.Errorf("failed to open session: %w", err)
	}

	h.sessions[sessionID] = session
	h.activeID = sessionID

	return h.sessionInfo(session), nil
}

func (h *SessionHandler) MountSession(sessionID, imagePath string) (*SessionInfo, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	session, ok := h.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	disk, err := storage.OpenRawDisk(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open disk: %w", err)
	}

	roDisk := storage.NewReadOnlyDisk(disk)

	sessionCtx := session.Context()
	partitions, err := roDisk.Partitions(sessionCtx)
	if err != nil {
		disk.Close()
		return nil, fmt.Errorf("failed to parse partitions: %w", err)
	}

	var fs storage.FileSystem
	if len(partitions) > 0 {
		bestIdx := storage.SelectBestPartition(partitions)
		targetPart := partitions[bestIdx-1]
		fs, err = storage.OpenNTFS(sessionCtx, targetPart)
		if err != nil {
			disk.Close()
			return nil, fmt.Errorf("failed to mount NTFS: %w", err)
		}
	}

	if err := session.Mount(sessionCtx, roDisk, partitions, fs); err != nil {
		disk.Close()
		return nil, fmt.Errorf("failed to mount session: %w", err)
	}

	return h.sessionInfo(session), nil
}

func (h *SessionHandler) ListSessionDirectory(sessionID, path string) ([]storage.UniversalFileNode, error) {
	h.mu.RLock()
	session, ok := h.sessions[sessionID]
	h.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	sessionCtx := session.Context()
	return session.ListDirectory(sessionCtx, path)
}

func (h *SessionHandler) GetSessionInfo(sessionID string) (*SessionInfo, error) {
	h.mu.RLock()
	session, ok := h.sessions[sessionID]
	h.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	return h.sessionInfo(session), nil
}

func (h *SessionHandler) GetActiveSession() (*SessionInfo, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.activeID == "" {
		return nil, fmt.Errorf("no active session")
	}

	session, ok := h.sessions[h.activeID]
	if !ok {
		return nil, fmt.Errorf("active session %s not found", h.activeID)
	}

	return h.sessionInfo(session), nil
}

func (h *SessionHandler) CloseSession(sessionID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	session, ok := h.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.Cancel()

	if err := session.Close(); err != nil {
		return err
	}

	delete(h.sessions, sessionID)
	if h.activeID == sessionID {
		h.activeID = ""
	}

	return nil
}

func (h *SessionHandler) LogSessionProvenance(sessionID, action, target, details string) error {
	h.mu.RLock()
	session, ok := h.sessions[sessionID]
	h.mu.RUnlock()

	if !ok {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.LogProvenance("analyst", action, target, details)
	return nil
}

func (h *SessionHandler) sessionInfo(session *storage.EvidenceSession) *SessionInfo {
	meta := session.Metadata()
	state := session.State()
	provenance := session.Provenance()

	stateStr := "closed"
	switch state {
	case storage.SessionOpen:
		stateStr = "open"
	case storage.SessionMounted:
		stateStr = "mounted"
	case storage.SessionAnalyzing:
		stateStr = "analyzing"
	}

	return &SessionInfo{
		ID:           session.ID(),
		State:        stateStr,
		ImagePath:    meta.ImagePath,
		FileName:     meta.FileName,
		Format:       string(meta.Format),
		TotalSize:    meta.TotalSize,
		IsReadOnly:   meta.IsReadOnly,
		PartitionCnt: meta.PartitionCnt,
		Filesystem:   meta.Filesystem,
		Provenance:   provenance,
	}
}

func (h *SessionHandler) SelectBestPartition(partitions []storage.Partition) int {
	return storage.SelectBestPartition(partitions)
}

func (h *SessionHandler) DetectFormat(imagePath string) string {
	return string(storage.DetectFormatFromPath(imagePath))
}

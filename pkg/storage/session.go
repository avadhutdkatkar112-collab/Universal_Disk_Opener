package storage

import (
	"context"
	"os"
	"sync"
	"time"
)

type SessionState int

const (
	SessionClosed SessionState = iota
	SessionOpen
	SessionMounted
	SessionAnalyzing
)

type Timestamps struct {
	Created     time.Time `json:"created"`
	Modified    time.Time `json:"modified"`
	Accessed    time.Time `json:"accessed"`
	MFTModified time.Time `json:"mft_modified"`
}

type UniversalFileNode struct {
	Name        string     `json:"name"`
	Path        string     `json:"path"`
	IsDir       bool       `json:"is_dir"`
	Size        uint64     `json:"size"`
	Timestamps  Timestamps `json:"timestamps"`
	Permissions string     `json:"permissions"`
	OwnerID     int        `json:"owner_id"`
	GroupID     int        `json:"group_id"`
	FileID      uint64     `json:"file_id"`
	Attributes  []string   `json:"attributes"`
	IsDeleted   bool       `json:"is_deleted"`
	StreamName  string     `json:"stream_name,omitempty"`
}

type ProvenanceEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Details   string    `json:"details"`
	SessionID string    `json:"session_id"`
}

type EvidenceMetadata struct {
	ImagePath    string `json:"image_path"`
	FileName     string `json:"file_name"`
	Format       Format `json:"format"`
	TotalSize    uint64 `json:"total_size"`
	SHA256       string `json:"sha256"`
	IsReadOnly   bool   `json:"is_read_only"`
	PartitionCnt int    `json:"partition_count"`
	Filesystem   string `json:"filesystem"`
}

type EvidenceSession struct {
	mu          sync.RWMutex
	id          string
	state       SessionState
	metadata    EvidenceMetadata
	disk        DiskImage
	partitions  []Partition
	filesystem  FileSystem
	provenance  []ProvenanceEntry
	openedAt    time.Time
	lastAccess  time.Time
	cancelFunc  context.CancelFunc
	ctx         context.Context
}

func NewEvidenceSession(id string) *EvidenceSession {
	ctx, cancel := context.WithCancel(context.Background())
	return &EvidenceSession{
		id:        id,
		state:     SessionClosed,
		openedAt:  time.Now().UTC(),
		lastAccess: time.Now().UTC(),
		cancelFunc: cancel,
		ctx:       ctx,
	}
}

func (s *EvidenceSession) Context() context.Context {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ctx
}

func (s *EvidenceSession) Cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
}

func (s *EvidenceSession) ID() string { return s.id }

func (s *EvidenceSession) State() SessionState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *EvidenceSession) Metadata() EvidenceMetadata {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.metadata
}

func (s *EvidenceSession) Provenance() []ProvenanceEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ProvenanceEntry, len(s.provenance))
	copy(out, s.provenance)
	return out
}

func (s *EvidenceSession) Open(ctx context.Context, imagePath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.metadata.ImagePath = imagePath
	if len(imagePath) > 0 {
		lastSlash := -1
		for i, c := range imagePath {
			if c == '/' || c == '\\' {
				lastSlash = i
			}
		}
		if lastSlash >= 0 && lastSlash < len(imagePath)-1 {
			s.metadata.FileName = imagePath[lastSlash+1:]
		} else {
			s.metadata.FileName = imagePath
		}
	}
	s.metadata.IsReadOnly = true
	s.state = SessionOpen
	s.lastAccess = time.Now().UTC()

	s.addProvenance("system", "session.open", imagePath, "Evidence session opened")

	return nil
}

func (s *EvidenceSession) Mount(ctx context.Context, disk DiskImage, partitions []Partition, fs FileSystem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.disk = disk
	s.partitions = partitions
	s.filesystem = fs
	s.state = SessionMounted

	s.metadata.TotalSize = 0
	for _, p := range partitions {
		size, _ := p.Size(ctx)
		s.metadata.TotalSize += size
	}

	s.addProvenance("system", "session.mount", "filesystem", "Filesystem mounted")

	return nil
}

func (s *EvidenceSession) ListDirectory(ctx context.Context, path string) ([]UniversalFileNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.lastAccess = time.Now().UTC()

	if s.filesystem == nil {
		return nil, ErrNoFilesystemMounted
	}

	var node Node
	var err error

	if path == "" || path == "/" {
		node, err = s.filesystem.Root(ctx)
	} else {
		node, err = s.filesystem.Open(ctx, path)
	}

	if err != nil {
		return nil, err
	}

	children, err := node.ReadDir(ctx)
	if err != nil {
		return nil, err
	}

	nodes := make([]UniversalFileNode, 0, len(children))
	for _, child := range children {
		nodes = append(nodes, UniversalFileNode{
			Name:  child.Name(),
			Path:  child.Path(),
			IsDir: child.IsDir(),
			Size:  child.Size(),
			Timestamps: Timestamps{
				Created:  child.ModTime(),
				Modified: child.ModTime(),
			},
		})
	}

	return nodes, nil
}

func (s *EvidenceSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.addProvenance("system", "session.close", s.metadata.ImagePath, "Evidence session closed")
	s.state = SessionClosed

	if s.cancelFunc != nil {
		s.cancelFunc()
	}

	return nil
}

func (s *EvidenceSession) LogProvenance(actor, action, target, details string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.addProvenance(actor, action, target, details)
}

func (s *EvidenceSession) addProvenance(actor, action, target, details string) {
	s.provenance = append(s.provenance, ProvenanceEntry{
		Timestamp: time.Now().UTC(),
		Actor:     actor,
		Action:    action,
		Target:    target,
		Details:   details,
		SessionID: s.id,
	})
}

func DetectFormatFromPath(imagePath string) Format {
	fileName := imagePath
	for i := len(imagePath) - 1; i >= 0; i-- {
		if imagePath[i] == '/' || imagePath[i] == '\\' {
			fileName = imagePath[i+1:]
			break
		}
	}

	ext := ""
	for i := len(fileName) - 1; i >= 0; i-- {
		if fileName[i] == '.' {
			ext = fileName[i+1:]
			break
		}
	}

	if fmt := detectFormatByMagic(imagePath); fmt != "UNKNOWN" {
		return fmt
	}

	switch ext {
	case "vhd":
		return "VHD"
	case "vhdx":
		return "VHDX"
	case "vdi":
		return "VDI"
	case "vmdk", "vmdk-full", "vmdk-sparse":
		return "VMDK"
	case "qcow2", "qcow":
		return "QCOW2"
	case "iso", "img":
		return "ISO"
	case "raw", "dd", "bin", "e01":
		return "RAW"
	default:
		return "UNKNOWN"
	}
}

func detectFormatByMagic(imagePath string) Format {
	f, err := os.Open(imagePath)
	if err != nil {
		return "UNKNOWN"
	}
	defer f.Close()

	buf := make([]byte, 64)
	n, err := f.Read(buf)
	if err != nil || n < 8 {
		return "UNKNOWN"
	}

	if n >= 12 && buf[0] == 'c' && buf[1] == 'o' && buf[2] == 'w' && buf[3] == 'q' &&
		buf[4] == 'd' && buf[5] == 'i' && buf[6] == 's' && buf[7] == 'k' {
		return "QCOW2"
	}

	if n >= 8 && buf[0] == 'V' && buf[1] == 'H' && buf[2] == 'D' && buf[3] == ' ' {
		return "VHD"
	}

	if n >= 8 && buf[0] == 'V' && buf[1] == 'H' && buf[2] == 'D' && buf[3] == 'X' {
		return "VHDX"
	}

	if n >= 16 && buf[0] == 0x7F && buf[1] == 'E' && buf[2] == 'N' && buf[3] == 'C' &&
		buf[4] == 'D' && buf[5] == 'K' && buf[6] == ' ' && buf[7] == 'Q' {
		return "VDI"
	}

	if n >= 16 && buf[0] == 'K' && buf[1] == 'D' && buf[2] == 'M' && buf[3] == 'V' {
		return "VMDK"
	}

	if n >= 8 && buf[0] == '#' && buf[1] == 'Q' && buf[2] == 'C' && buf[3] == 'O' &&
		buf[4] == 'W' && buf[5] == '-' && buf[6] == 'F' && buf[7] == 'I' {
		return "QCOW2"
	}

	return "UNKNOWN"
}

func SelectBestPartition(partitions []Partition) int {
	for _, p := range partitions {
		if p.Type() == "NTFS / exFAT" {
			return p.Index()
		}
	}
	for _, p := range partitions {
		if p.Type() == "Linux EXT4" {
			return p.Index()
		}
	}
	if len(partitions) > 0 {
		return partitions[0].Index()
	}
	return 1
}

type ReadOnlyDisk struct {
	inner DiskImage
}

func NewReadOnlyDisk(inner DiskImage) *ReadOnlyDisk {
	return &ReadOnlyDisk{inner: inner}
}

func (r *ReadOnlyDisk) Size(ctx context.Context) (uint64, error) {
	return r.inner.Size(ctx)
}

func (r *ReadOnlyDisk) ReadAt(ctx context.Context, p []byte, off uint64) (int, error) {
	return r.inner.ReadAt(ctx, p, off)
}

func (r *ReadOnlyDisk) Format() Format {
	return r.inner.Format()
}

func (r *ReadOnlyDisk) VirtualSize(ctx context.Context) (uint64, error) {
	return r.inner.VirtualSize(ctx)
}

func (r *ReadOnlyDisk) Partitions(ctx context.Context) ([]Partition, error) {
	return r.inner.Partitions(ctx)
}

func (r *ReadOnlyDisk) WriteAt(p []byte, off int64) (int, error) {
	return 0, ErrWriteDenied
}

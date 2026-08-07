package engine

import (
	"io"
	"os"

	"github.com/user/vhd-opener/internal/engine/core"
	"github.com/user/vhd-opener/internal/domain/disk"
	"github.com/user/vhd-opener/internal/domain/filesystem"
)

// FilesystemAdapter bridges an old filesystem.Reader to the new
// core.FilesystemDriver interface.
type FilesystemAdapter struct {
	reader      filesystem.Reader
	fsType      filesystem.FSType
	disk        core.DiskDriver
	partStart   uint64
	files       uint64
	directories uint64
}

// NewFilesystemAdapter creates a FilesystemAdapter from an existing reader.
func NewFilesystemAdapter(reader filesystem.Reader, fsType filesystem.FSType, disk core.DiskDriver, partStart uint64) *FilesystemAdapter {
	return &FilesystemAdapter{
		reader:    reader,
		fsType:    fsType,
		disk:      disk,
		partStart: partStart,
	}
}

func (a *FilesystemAdapter) Name() string {
	return string(a.fsType)
}

func (a *FilesystemAdapter) Detect(disk core.DiskDriver, startLBA uint64) bool {
	// Detection already happened when the adapter was created
	return a.fsType != filesystem.Unknown
}

func (a *FilesystemAdapter) Mount(disk core.DiskDriver, startLBA uint64) error {
	// Already mounted
	return nil
}

func (a *FilesystemAdapter) ReadDir(path string) ([]core.VFSNode, error) {
	entries, err := a.reader.ListDirectory(a.makeDiskReader(), a.partStart, path)
	if err != nil {
		return nil, err
	}

	nodes := make([]core.VFSNode, len(entries))
	for i, e := range entries {
		nodes[i] = a.convertEntry(e)
	}
	return nodes, nil
}

func (a *FilesystemAdapter) OpenFile(path string) (io.ReaderAt, uint64, error) {
	// Find the file entry
	entry, err := a.findEntry(path)
	if err != nil {
		return nil, 0, err
	}

	data, err := a.reader.GetFileContent(a.makeDiskReader(), a.partStart, entry)
	if err != nil {
		return nil, 0, err
	}

	return &bytesReaderAt{data: data}, uint64(len(data)), nil
}

func (a *FilesystemAdapter) GetNode(path string) (*core.VFSNode, error) {
	entry, err := a.findEntry(path)
	if err != nil {
		return nil, err
	}
	node := a.convertEntry(*entry)
	return &node, nil
}

func (a *FilesystemAdapter) Info() core.FSInfo {
	return core.FSInfo{
		Type:        string(a.fsType),
		Files:       a.files,
		Directories: a.directories,
	}
}

func (a *FilesystemAdapter) makeDiskReader() filesystem.DiskReader {
	// Use the LegacyAdapter to bridge to the old filesystem.DiskReader interface
	if la, ok := a.disk.(*LegacyAdapter); ok {
		return filesystem.NewDiskAdapter(la.Inner())
	}
	// Fallback: use a minimal adapter
	return filesystem.NewDiskAdapter(&minimalVDisk{a.disk})
}

// minimalVDisk adapts a core.DiskDriver to disk.VirtualDisk for the filesystem adapter.
type minimalVDisk struct {
	d core.DiskDriver
}

func (m *minimalVDisk) Open(path string) error            { return nil }
func (m *minimalVDisk) Close() error                      { return m.d.Close() }
func (m *minimalVDisk) ReadSectors(start uint64, count uint32) ([]byte, error) {
	offset := int64(start) * int64(m.d.SectorSize())
	size := int64(count) * int64(m.d.SectorSize())
	buf := make([]byte, size)
	n, err := m.d.ReadAt(buf, offset)
	return buf[:n], err
}
func (m *minimalVDisk) ReadAt(buf []byte, off int64) (int, error) { return m.d.ReadAt(buf, off) }
func (m *minimalVDisk) Size() uint64                              { return m.d.SizeBytes() }
func (m *minimalVDisk) SectorSize() uint32                        { return m.d.SectorSize() }
func (m *minimalVDisk) TotalSectors() uint64                      { return m.d.TotalSectors() }
func (m *minimalVDisk) DiskType() string                          { return string(m.d.Type()) }
func (m *minimalVDisk) Format() string                            { return string(m.d.Type()) }
func (m *minimalVDisk) Info() disk.DiskInfo                       { return disk.DiskInfo{} }
func (m *minimalVDisk) FilePath() string                          { return m.d.FilePath() }
func (m *minimalVDisk) FileName() string                          { return m.d.FileName() }
func (m *minimalVDisk) Warnings() []string                        { return m.d.Warnings() }

func (a *FilesystemAdapter) findEntry(path string) (*filesystem.FileEntry, error) {
	entries, err := a.reader.ListDirectory(a.makeDiskReader(), a.partStart, "")
	if err != nil {
		return nil, err
	}

	// Simple path matching (root level)
	for i := range entries {
		if entries[i].Path == path || entries[i].FullFSPath == path {
			return &entries[i], nil
		}
	}
	return nil, os.ErrNotExist
}

func (a *FilesystemAdapter) convertEntry(e filesystem.FileEntry) core.VFSNode {
	nodeType := core.EntryTypeFile
	if e.IsDirectory {
		nodeType = core.EntryTypeDirectory
	}
	if e.FileType == 2 { // Symlink
		nodeType = core.EntryTypeSymlink
	}

	return core.VFSNode{
		Name:    e.Name,
		Path:    e.Path,
		Type:    nodeType,
		Size:    uint64(e.Size),
		Mode:    0,
		Inode:   0,
		ModTime: e.ModifiedTime,
	}
}

// bytesReaderAt wraps a byte slice as io.ReaderAt.
type bytesReaderAt struct {
	data []byte
}

func (r *bytesReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(r.data)) {
		return 0, io.EOF
	}
	n := copy(p, r.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

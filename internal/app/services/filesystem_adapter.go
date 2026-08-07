package services

import (
	"io"
	"os"

	"github.com/user/vhd-opener/internal/storage/filesystems"
)

type FilesystemAdapter struct {
	reader      filesystems.Reader
	fsType      string
	partStart   uint64
	files       uint64
	directories uint64
}

func NewFilesystemAdapter(reader filesystems.Reader, fsType string, partStart uint64) *FilesystemAdapter {
	return &FilesystemAdapter{
		reader:    reader,
		fsType:    fsType,
		partStart: partStart,
	}
}

func (a *FilesystemAdapter) Name() string {
	return a.fsType
}

func (a *FilesystemAdapter) ReadDir(path string) ([]VFSNode, error) {
	return nil, nil
}

func (a *FilesystemAdapter) OpenFile(path string) (io.ReaderAt, uint64, error) {
	return nil, 0, os.ErrNotExist
}

func (a *FilesystemAdapter) GetNode(path string) (*VFSNode, error) {
	return nil, os.ErrNotExist
}

func (a *FilesystemAdapter) Info() FSInfo {
	return FSInfo{
		Type:        a.fsType,
		Files:       a.files,
		Directories: a.directories,
	}
}

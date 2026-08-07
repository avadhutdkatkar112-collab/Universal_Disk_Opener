package storage

import (
	"context"
	"io"
	"time"
)

type Format string

const (
	FormatRaw   Format = "RAW"
	FormatVHD   Format = "VHD"
	FormatVHDX  Format = "VHDX"
	FormatVDI   Format = "VDI"
	FormatVMDK  Format = "VMDK"
	FormatQCOW2 Format = "QCOW2"
)

type BlockReader interface {
	Size(ctx context.Context) (uint64, error)
	ReadAt(ctx context.Context, p []byte, off uint64) (int, error)
}

// Partition represents a discovered partition with filesystem info.
type Partition struct {
	Index      int    `json:"index"`
	Start      uint64 `json:"start"`
	End        uint64 `json:"end"`
	Size       uint64 `json:"size"`
	Type       string `json:"type"`
	Filesystem string `json:"filesystem"`
	Bootable   bool   `json:"bootable"`
	Label      string `json:"label"`
	Active     bool   `json:"active"`
	HasContent bool   `json:"hasContent"`
}

type DiskImage interface {
	BlockReader
	Format() Format
	VirtualSize(ctx context.Context) (uint64, error)
	Partitions(ctx context.Context) ([]Partition, error)
}

type FileSystem interface {
	Name() string
	Root(ctx context.Context) (Node, error)
	Open(ctx context.Context, path string) (Node, error)
}

type Node interface {
	io.ReaderAt
	Name() string
	Path() string
	IsDir() bool
	Size() uint64
	ModTime() time.Time
	ReadDir(ctx context.Context) ([]Node, error)
}

type PartitionBlockReader struct {
	disk DiskImage
	part Partition
}

func NewPartitionBlockReader(disk DiskImage, part Partition) *PartitionBlockReader {
	return &PartitionBlockReader{disk: disk, part: part}
}

func (r *PartitionBlockReader) Size(ctx context.Context) (uint64, error) {
	return r.part.Size, nil
}

func (r *PartitionBlockReader) ReadAt(ctx context.Context, p []byte, off uint64) (int, error) {
	return r.disk.ReadAt(ctx, p, r.part.Start+off)
}

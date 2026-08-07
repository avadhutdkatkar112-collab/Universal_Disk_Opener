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

type DiskImage interface {
	BlockReader
	Format() Format
	VirtualSize(ctx context.Context) (uint64, error)
	Partitions(ctx context.Context) ([]Partition, error)
}

type Partition interface {
	BlockReader
	Start() uint64
	Type() string
	Index() int
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

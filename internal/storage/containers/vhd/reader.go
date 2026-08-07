package vhd

import "io"

// SectorReader provides an interface for reading sectors from a VHD.
// Implementations handle both Fixed and Dynamic disk types transparently.
type SectorReader interface {
	ReadSectors(offset uint64, count uint32) ([]byte, error)
	SectorSize() uint64
	TotalSectors() uint64
	Close() error
}

// VHDReader provides high-level access to VHD file contents.
type VHDReader interface {
	Open(filePath string) error
	Close() error
	GetFooter() *Footer
	GetDynamicHeader() *DynamicHeader
	GetBAT() []uint32
	GetDiskType() DiskType
	GetDiskInfo() *DiskInfo
	GetSectorReader() SectorReader
}

// FileAtReader extends io.ReaderAt for VHD file access.
type FileAtReader interface {
	io.ReaderAt
	io.Closer
	Size() (int64, error)
}

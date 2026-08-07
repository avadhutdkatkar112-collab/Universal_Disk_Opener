// Package core defines the universal disk engine interfaces.
// This layer knows nothing about Wails, React, VHD, or ext4 — it only knows generic interfaces.
package core

import (
	"io"
	"time"
)

// DiskType enumerates supported virtual disk container formats.
type DiskType string

const (
	DiskTypeVHD   DiskType = "VHD"
	DiskTypeVHDX  DiskType = "VHDX"
	DiskTypeRAW   DiskType = "RAW"
	DiskTypeVMDK  DiskType = "VMDK"
	DiskTypeQCOW2 DiskType = "QCOW2"
	DiskTypeVDI   DiskType = "VDI"
)

// FSType enumerates supported filesystem types.
type FSType string

const (
	FSTypeExt4  FSType = "ext4"
	FSTypeExt2  FSType = "ext2"
	FSTypeFAT16 FSType = "FAT16"
	FSTypeFAT32 FSType = "FAT32"
	FSTypeNTFS  FSType = "NTFS"
	FSTypeUnknown FSType = "Unknown"
)

// DiskDriver abstracts physical file offsets and block tables.
// Each format driver (VHD, RAW, VHDX, etc.) implements this interface.
type DiskDriver interface {
	io.ReaderAt
	io.Closer

	// Type returns the disk container format.
	Type() DiskType

	// SectorSize returns the logical sector size in bytes.
	SectorSize() uint32

	// TotalSectors returns the total number of sectors.
	TotalSectors() uint64

	// SizeBytes returns the virtual disk size in bytes.
	SizeBytes() uint64

	// Metadata returns format-specific key-value metadata.
	Metadata() map[string]string

	// FilePath returns the path to the underlying file.
	FilePath() string

	// FileName returns the base name of the file.
	FileName() string

	// Warnings returns any non-fatal warnings from opening the disk.
	Warnings() []string
}

// DiskDriverFactory is a constructor function for a disk driver.
// The registry calls this when opening a file by extension.
type DiskDriverFactory func(filePath string) (DiskDriver, error)

// EntryType enumerates virtual filesystem entry types.
type EntryType uint8

const (
	EntryTypeFile EntryType = iota
	EntryTypeDirectory
	EntryTypeSymlink
)

// VFSNode represents a single entry in a virtual filesystem.
type VFSNode struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	Type     EntryType `json:"type"`
	Size     uint64    `json:"size"`
	Mode     uint32    `json:"mode"`
	Inode    uint64    `json:"inode"`
	ModTime  time.Time `json:"mod_time"`
	Target   string    `json:"target,omitempty"` // For symlinks
	Metadata map[string]string `json:"metadata,omitempty"`
}

// PartitionInfo describes a disk partition detected by the partition engine.
type PartitionInfo struct {
	Index      int    `json:"index"`
	Start      uint64 `json:"start"`
	End        uint64 `json:"end"`
	Size       uint64 `json:"size"`
	Type       string `json:"type"`
	Filesystem string `json:"filesystem"`
	Bootable   bool   `json:"bootable"`
	Label      string `json:"label"`
}

// FilesystemDriver abstracts ext4, NTFS, FAT, etc.
type FilesystemDriver interface {
	// Name returns the filesystem name ("ext4", "FAT32", "NTFS").
	Name() string

	// Detect checks if the filesystem at the given partition start is this type.
	Detect(disk DiskDriver, startLBA uint64) bool

	// Mount initializes the filesystem reader.
	Mount(disk DiskDriver, startLBA uint64) error

	// ReadDir lists directory contents.
	ReadDir(path string) ([]VFSNode, error)

	// OpenFile returns a reader and size for the given file path.
	OpenFile(path string) (io.ReaderAt, uint64, error)

	// GetNode returns metadata for a single node.
	GetNode(path string) (*VFSNode, error)

	// Info returns filesystem-level information.
	Info() FSInfo
}

// FSInfo holds filesystem-level statistics.
type FSInfo struct {
	Type        string `json:"type"`
	Label       string `json:"label"`
	TotalSpace  uint64 `json:"totalSpace"`
	FreeSpace   uint64 `json:"freeSpace"`
	UsedSpace   uint64 `json:"usedSpace"`
	Files       uint64 `json:"files"`
	Directories uint64 `json:"directories"`
}

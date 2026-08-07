// Package vhd defines the domain types for Virtual Hard Disk files.
// It implements Microsoft's VHD specification for Fixed and Dynamic disk types.
package vhd

import "time"

const (
	FooterSize         = 512
	DynamicHeaderSize  = 1024
	BATEntrySize       = 4
	SectorSizeBytes    = 512
	MaxSectorReadSize  = 1024 * 1024 // 1MB max per read

	SignatureFooter    = "conectix"
	SignatureDynamic   = "cxsparse"

	// Disk type constants per Microsoft VHD spec
	DiskTypeNone         DiskType = 0
	DiskTypeReserved1    DiskType = 1
	DiskTypeFixed        DiskType = 2
	DiskTypeDynamic      DiskType = 3
	DiskTypeDifferencing DiskType = 4
	DiskTypeReserved2    DiskType = 5
	DiskTypeReserved3    DiskType = 6

	// Platform codes for parent locators
	PlatformCodeWinNTParentPath = "W2ku"
	PlatformCodeWinNTFTPParentPath = "W2ru"
	PlatformCodeMacintoshParentPath = "MacX"
	PlatformCodeMacintoshHostPath = "Mac "
)

// Footer represents the 512-byte VHD footer structure.
type Footer struct {
	Signature       [8]byte
	Features        uint32
	FileFormatVer   uint32
	DataOffset      uint64
	Timestamp       uint32
	CreatorApp      [4]byte
	CreatorVer      uint32
	CreatorHostOS   uint32
	OriginalSize    uint64
	CurrentSize     uint64
	DiskGeometry    DiskGeometry
	DiskType        DiskType
	Checksum        uint32
	UniqueID        [16]byte
	SavedState      uint8
	Reserved        [427]byte
}

// DiskGeometry represents the CHS geometry of a virtual disk.
type DiskGeometry struct {
	Cylinders       uint16
	Heads           uint8
	SectorsPerTrack uint8
}

// DiskType identifies the type of VHD disk.
type DiskType uint32

// String returns a human-readable name for the disk type.
func (dt DiskType) String() string {
	switch dt {
	case DiskTypeFixed:
		return "Fixed"
	case DiskTypeDynamic:
		return "Dynamic"
	case DiskTypeDifferencing:
		return "Differencing"
	default:
		return "Unknown"
	}
}

// DynamicHeader represents the 1024-byte dynamic disk header.
type DynamicHeader struct {
	Signature         [8]byte
	DataOffset        uint64
	TableOffset       uint64
	FileFormatVer     uint32
	MaxTableEntries   uint32
	BlockSize         uint32
	Checksum          uint32
	ParentUniqueID    [16]byte
	ParentTimestamp   uint32
	Reserved1         [4]byte
	ParentName        [512]byte
	ParentLocators    [8]ParentLocatorEntry
	Reserved2         [256]byte
}

// ParentLocatorEntry describes where a parent disk path is stored.
type ParentLocatorEntry struct {
	PlatformCode       [4]byte
	PlatformDataSpace  uint32
	PlatformDataSize   uint32
	Reserved           uint32
	PlatformDataOffset uint64
}

// DiskInfo holds computed information about an opened VHD.
type DiskInfo struct {
	FilePath       string
	FileName       string
	FileSize       int64
	VirtualSize    uint64
	DiskType       string
	CreatorApp     string
	CreatorVersion string
	CreatorHostOS  string
	CreatedAt      time.Time
	Geometry       DiskGeometry
	UniqueID       string
	ChecksumValid  bool
	BlockSize      uint32
	MaxBATEntries  uint32
}

// ParseTimestamp converts a VHD timestamp (seconds since Jan 1, 2000 UTC) to time.Time.
func ParseTimestamp(ts uint32) time.Time {
	epoch := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	return epoch.Add(time.Duration(ts) * time.Second)
}

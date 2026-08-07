// Package filesystem defines the domain types for filesystem access.
// It supports reading FAT16, FAT32, and NTFS filesystems from VHD images.
package filesystem

import "time"

// FSType identifies a filesystem type.
type FSType string

const (
	FAT12   FSType = "FAT12"
	FAT16   FSType = "FAT16"
	FAT32   FSType = "FAT32"
	NTFS    FSType = "NTFS"
	EXT2    FSType = "EXT2"
	Unknown FSType = "Unknown"
)

// FileEntry represents a file or directory in the filesystem.
type FileEntry struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Path         string         `json:"path"`
	FullFSPath   string         `json:"fullFsPath"`
	Size         int64          `json:"size"`
	IsDirectory  bool           `json:"isDirectory"`
	IsHidden     bool           `json:"isHidden"`
	ModifiedTime time.Time      `json:"modifiedTime"`
	CreatedTime  time.Time      `json:"createdTime"`
	AccessedTime time.Time      `json:"accessedTime"`
	Attributes   FileAttributes `json:"attributes"`
	DataOffset   uint64         `json:"-"`
	DataLength   uint64         `json:"-"`
	ClusterStart uint32         `json:"-"`
	Extension    string         `json:"extension"`
}

// FileAttributes stores file attribute flags.
type FileAttributes struct {
	ReadOnly    bool `json:"readOnly"`
	Hidden      bool `json:"hidden"`
	System      bool `json:"system"`
	Archive     bool `json:"archive"`
	Directory   bool `json:"directory"`
	VolumeLabel bool `json:"volumeLabel"`
}

// FileProperties holds detailed file metadata for display.
type FileProperties struct {
	Name         string         `json:"name"`
	Extension    string         `json:"extension"`
	FullPath     string         `json:"fullPath"`
	Size         int64          `json:"size"`
	SizeFormatted string        `json:"sizeFormatted"`
	IsDirectory  bool           `json:"isDirectory"`
	ModifiedTime time.Time      `json:"modifiedTime"`
	CreatedTime  time.Time      `json:"createdTime"`
	AccessedTime time.Time      `json:"accessedTime"`
	Attributes   FileAttributes `json:"attributes"`
	ClusterStart uint32         `json:"clusterStart"`
	FSType       string         `json:"fsType"`
}

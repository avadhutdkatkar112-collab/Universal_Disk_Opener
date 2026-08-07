// Package filesystem provides a bridge between storage.VirtualDisk and filesystem readers.
// This adapter allows the filesystem layer to read from any disk format.
package filesystems

import (
	"github.com/user/vhd-opener/internal/storage"
)

// DiskAdapter wraps a storage.VirtualDisk to implement the filesystem.DiskReader interface.
type DiskAdapter struct {
	vdisk storage.VirtualDisk
}

// NewDiskAdapter creates a new adapter.
func NewDiskAdapter(vdisk storage.VirtualDisk) *DiskAdapter {
	return &DiskAdapter{vdisk: vdisk}
}

// ReadSectors reads sectors from the underlying disk.
func (a *DiskAdapter) ReadSectors(offset uint64, count uint32) ([]byte, error) {
	return a.vdisk.ReadSectors(offset, count)
}

// SectorSize returns the sector size.
func (a *DiskAdapter) SectorSize() uint64 {
	return uint64(a.vdisk.SectorSize())
}

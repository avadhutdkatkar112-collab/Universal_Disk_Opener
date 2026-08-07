// Package filesystem provides a bridge between disk.VirtualDisk and filesystem readers.
// This adapter allows the filesystem layer to read from any disk format.
package filesystem

import (
	"github.com/user/vhd-opener/internal/domain/disk"
)

// DiskAdapter wraps a disk.VirtualDisk to implement the filesystem.DiskReader interface.
type DiskAdapter struct {
	vdisk disk.VirtualDisk
}

// NewDiskAdapter creates a new adapter.
func NewDiskAdapter(vdisk disk.VirtualDisk) *DiskAdapter {
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

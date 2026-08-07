package services

import (
	"fmt"

	"github.com/user/vhd-opener/internal/storage"
)

// LegacyAdapter wraps an existing disk.storage.VirtualDisk and exposes it
// as a DiskDriver. This allows the old drivers (VHD, VHDX, VMDK,
// QCOW2, VDI, RAW) to be used through the new engine without rewriting.
type LegacyAdapter struct {
	inner storage.VirtualDisk
}

// NewLegacyAdapter wraps a disk.storage.VirtualDisk as a DiskDriver.
func NewLegacyAdapter(vd storage.VirtualDisk) *LegacyAdapter {
	return &LegacyAdapter{inner: vd}
}

func (a *LegacyAdapter) ReadAt(p []byte, off int64) (int, error) {
	return a.inner.ReadAt(p, off)
}

func (a *LegacyAdapter) Close() error {
	return a.inner.Close()
}

func (a *LegacyAdapter) Type() DiskType {
	return map[string]DiskType{
		"VHD":   DiskTypeVHD,
		"VHDX":  DiskTypeVHDX,
		"RAW":   DiskTypeRAW,
		"VMDK":  DiskTypeVMDK,
		"QCOW2": DiskTypeQCOW2,
		"VDI":   DiskTypeVDI,
	}[a.inner.Format()]
}

func (a *LegacyAdapter) SectorSize() uint32 {
	return a.inner.SectorSize()
}

func (a *LegacyAdapter) TotalSectors() uint64 {
	return a.inner.TotalSectors()
}

func (a *LegacyAdapter) SizeBytes() uint64 {
	return a.inner.Size()
}

func (a *LegacyAdapter) Metadata() map[string]string {
	info := a.inner.Info()
	return map[string]string{
		"Format":      info.Format,
		"DiskType":    info.DiskType,
		"UniqueID":    info.UniqueID,
		"CreatorApp":  info.CreatorApp,
		"BlockSize":   fmt.Sprintf("%d", info.BlockSize),
	}
}

func (a *LegacyAdapter) FilePath() string {
	return a.inner.FilePath()
}

func (a *LegacyAdapter) FileName() string {
	return a.inner.FileName()
}

func (a *LegacyAdapter) Warnings() []string {
	return a.inner.Warnings()
}

// Inner returns the underlying storage.VirtualDisk for backward compatibility.
func (a *LegacyAdapter) Inner() storage.VirtualDisk {
	return a.inner
}

package engine

import (
	"fmt"

	"github.com/user/vhd-opener/internal/engine/core"
	"github.com/user/vhd-opener/internal/domain/disk"
)

// LegacyAdapter wraps an existing disk.VirtualDisk and exposes it
// as a core.DiskDriver. This allows the old drivers (VHD, VHDX, VMDK,
// QCOW2, VDI, RAW) to be used through the new engine without rewriting.
type LegacyAdapter struct {
	inner disk.VirtualDisk
}

// NewLegacyAdapter wraps a disk.VirtualDisk as a core.DiskDriver.
func NewLegacyAdapter(vd disk.VirtualDisk) *LegacyAdapter {
	return &LegacyAdapter{inner: vd}
}

func (a *LegacyAdapter) ReadAt(p []byte, off int64) (int, error) {
	return a.inner.ReadAt(p, off)
}

func (a *LegacyAdapter) Close() error {
	return a.inner.Close()
}

func (a *LegacyAdapter) Type() core.DiskType {
	return map[string]core.DiskType{
		"VHD":   core.DiskTypeVHD,
		"VHDX":  core.DiskTypeVHDX,
		"RAW":   core.DiskTypeRAW,
		"VMDK":  core.DiskTypeVMDK,
		"QCOW2": core.DiskTypeQCOW2,
		"VDI":   core.DiskTypeVDI,
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

// Inner returns the underlying disk.VirtualDisk for backward compatibility.
func (a *LegacyAdapter) Inner() disk.VirtualDisk {
	return a.inner
}

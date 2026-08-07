package vmdk

import (
	"path/filepath"
	"runtime"
	"testing"
)

var testDir string

func init() {
	_, thisFile, _, _ := runtime.Caller(0)
	testDir = filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
}

func TestVMDKDriver_OpenSparse(t *testing.T) {
	d := &VMDKDriver{}
	err := d.Open(filepath.Join(testDir, "test_sparse.vmdk"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer d.Close()

	if d.Format() != "VMDK" {
		t.Errorf("Format = %q, want VMDK", d.Format())
	}

	if d.DiskType() != "Dynamic" {
		t.Errorf("DiskType = %q, want Dynamic", d.DiskType())
	}

	if d.SectorSize() != 512 {
		t.Errorf("SectorSize = %d, want 512", d.SectorSize())
	}

	if d.TotalSectors() != 102400 {
		t.Errorf("TotalSectors = %d, want 102400", d.TotalSectors())
	}
}

func TestVMDKDriver_ReadSectors(t *testing.T) {
	d := &VMDKDriver{}
	err := d.Open(filepath.Join(testDir, "test_sparse.vmdk"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer d.Close()

	data, err := d.ReadSectors(0, 1)
	if err != nil {
		t.Fatalf("ReadSectors failed: %v", err)
	}

	if len(data) != 512 {
		t.Fatalf("ReadSectors returned %d bytes, want 512", len(data))
	}
}

func TestVMDKDriver_Info(t *testing.T) {
	d := &VMDKDriver{}
	err := d.Open(filepath.Join(testDir, "test_sparse.vmdk"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer d.Close()

	info := d.Info()

	if info.Format != "VMDK" {
		t.Errorf("Info.Format = %q, want VMDK", info.Format)
	}

	if info.DiskType != "Dynamic" {
		t.Errorf("Info.DiskType = %q, want Dynamic", info.DiskType)
	}

	if info.VirtualSize != 102400*512 {
		t.Errorf("Info.VirtualSize = %d, want %d", info.VirtualSize, 102400*512)
	}
}

package raw

import (
	"bytes"
	"path/filepath"
	"runtime"
	"testing"
)

var testImagePath string

func init() {
	_, thisFile, _, _ := runtime.Caller(0)
	testImagePath = filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "test_disk.dd")
}

func TestRawDriver_Open(t *testing.T) {
	d := &RAWDriver{}
	err := d.Open(testImagePath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer d.Close()

	if d.Format() != "RAW" {
		t.Errorf("Format = %q, want RAW", d.Format())
	}

	if d.DiskType() != "Fixed" {
		t.Errorf("DiskType = %q, want Fixed", d.DiskType())
	}

	if d.SectorSize() != 512 {
		t.Errorf("SectorSize = %d, want 512", d.SectorSize())
	}

	expectedSize := uint64(50 * 1024 * 1024)
	if d.Size() != expectedSize {
		t.Errorf("Size = %d, want %d", d.Size(), expectedSize)
	}

	expectedSectors := expectedSize / 512
	if d.TotalSectors() != expectedSectors {
		t.Errorf("TotalSectors = %d, want %d", d.TotalSectors(), expectedSectors)
	}
}

func TestRawDriver_ReadSectors(t *testing.T) {
	d := &RAWDriver{}
	err := d.Open(testImagePath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer d.Close()

	// Read first sector
	data, err := d.ReadSectors(0, 1)
	if err != nil {
		t.Fatalf("ReadSectors failed: %v", err)
	}

	if len(data) != 512 {
		t.Fatalf("ReadSectors returned %d bytes, want 512", len(data))
	}

	// Read multiple sectors
	data, err = d.ReadSectors(0, 10)
	if err != nil {
		t.Fatalf("ReadSectors(10) failed: %v", err)
	}

	if len(data) != 5120 {
		t.Fatalf("ReadSectors(10) returned %d bytes, want 5120", len(data))
	}
}

func TestRawDriver_ReadAt(t *testing.T) {
	d := &RAWDriver{}
	err := d.Open(testImagePath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer d.Close()

	// Read at offset 0
	buf := make([]byte, 1024)
	n, err := d.ReadAt(buf, 0)
	if err != nil {
		t.Fatalf("ReadAt failed: %v", err)
	}

	if n != 1024 {
		t.Errorf("ReadAt returned %d bytes, want 1024", n)
	}

	// Read at offset 1024
	buf2 := make([]byte, 512)
	n, err = d.ReadAt(buf2, 1024)
	if err != nil {
		t.Fatalf("ReadAt(1024) failed: %v", err)
	}

	if n != 512 {
		t.Errorf("ReadAt(1024) returned %d bytes, want 512", n)
	}

	// Verify both reads at same offset return same data
	if !bytes.Equal(buf[:512], buf2) {
		t.Error("ReadAt(0) and ReadAt(1024) returned different data for same region")
	}
}

func TestRawDriver_Info(t *testing.T) {
	d := &RAWDriver{}
	err := d.Open(testImagePath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer d.Close()

	info := d.Info()

	if info.Format != "RAW" {
		t.Errorf("Info.Format = %q, want RAW", info.Format)
	}

	if info.DiskType != "Fixed" {
		t.Errorf("Info.DiskType = %q, want Fixed", info.DiskType)
	}

	if info.FileSize != 50*1024*1024 {
		t.Errorf("Info.FileSize = %d, want %d", info.FileSize, 50*1024*1024)
	}

	if info.VirtualSize != 50*1024*1024 {
		t.Errorf("Info.VirtualSize = %d, want %d", info.VirtualSize, 50*1024*1024)
	}
}

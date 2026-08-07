package qcow2

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

var testDir string

func init() {
	_, thisFile, _, _ := runtime.Caller(0)
	testDir = filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
}

func TestQCOW2Driver_OpenMinimal(t *testing.T) {
	path := filepath.Join(testDir, "test_minimal.qcow2")

	// Create a minimal QCOW2 file
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(path)

	// Write QCOW2 header (at least 72 bytes)
	buf := make([]byte, 1024*1024) // 1MB
	f.Write(buf)
	f.Close()

	// Write header fields
	f2, _ := os.OpenFile(path, os.O_WRONLY, 0)
	defer f2.Close()

	// Magic number at offset 0: "QFI\xfb"
	f2.WriteAt([]byte{0x51, 0x46, 0x49, 0xfb}, 0)

	// Version (offset 4, big-endian uint32) = 2
	verBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(verBytes, 2)
	f2.WriteAt(verBytes, 4)

	// Cluster bits (offset 24, big-endian uint32) = 16
	clusterBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(clusterBytes, 16)
	f2.WriteAt(clusterBytes, 24)

	// Virtual disk size (offset 32, big-endian uint64) = 1GB
	sizeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(sizeBytes, 1024*1024*1024)
	f2.WriteAt(sizeBytes, 32)

	// Now test opening
	d := &QCOW2Driver{}
	err = d.Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer d.Close()

	if d.Format() != "QCOW2" {
		t.Errorf("Format = %q, want QCOW2", d.Format())
	}

	if d.Size() != 1024*1024*1024 {
		t.Errorf("Size = %d, want %d", d.Size(), 1024*1024*1024)
	}

	if d.SectorSize() != 512 {
		t.Errorf("SectorSize = %d, want 512", d.SectorSize())
	}
}

func TestQCOW2Driver_ReadSectors(t *testing.T) {
	path := filepath.Join(testDir, "test_minimal.qcow2")

	// Create test file
	f, _ := os.Create(path)
	buf := make([]byte, 1024*1024)
	f.Write(buf)
	f.Close()
	defer os.Remove(path)

	// Write header
	f2, _ := os.OpenFile(path, os.O_WRONLY, 0)
	defer f2.Close()

	f2.WriteAt([]byte{0x51, 0x46, 0x49, 0xfb}, 0)
	verBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(verBytes, 2)
	f2.WriteAt(verBytes, 4)
	clusterBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(clusterBytes, 16)
	f2.WriteAt(clusterBytes, 24)
	sizeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(sizeBytes, 1024*1024*1024)
	f2.WriteAt(sizeBytes, 32)

	d := &QCOW2Driver{}
	err := d.Open(path)
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
}

func TestQCOW2Driver_Info(t *testing.T) {
	path := filepath.Join(testDir, "test_minimal.qcow2")

	// Create test file
	f, _ := os.Create(path)
	buf := make([]byte, 1024*1024)
	f.Write(buf)
	f.Close()
	defer os.Remove(path)

	// Write header
	f2, _ := os.OpenFile(path, os.O_WRONLY, 0)
	defer f2.Close()

	f2.WriteAt([]byte{0x51, 0x46, 0x49, 0xfb}, 0)
	verBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(verBytes, 2)
	f2.WriteAt(verBytes, 4)
	clusterBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(clusterBytes, 16)
	f2.WriteAt(clusterBytes, 24)
	sizeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(sizeBytes, 1024*1024*1024)
	f2.WriteAt(sizeBytes, 32)

	d := &QCOW2Driver{}
	err := d.Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer d.Close()

	info := d.Info()

	if info.Format != "QCOW2" {
		t.Errorf("Info.Format = %q, want QCOW2", info.Format)
	}

	if info.VirtualSize != 1024*1024*1024 {
		t.Errorf("Info.VirtualSize = %d, want %d", info.VirtualSize, 1024*1024*1024)
	}
}

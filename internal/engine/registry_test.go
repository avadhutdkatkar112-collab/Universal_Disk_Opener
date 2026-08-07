package engine

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

var testDir string

func init() {
	_, thisFile, _, _ := runtime.Caller(0)
	testDir = filepath.Join(filepath.Dir(thisFile), "..", "..")
}

func TestRegistry_OpenRAW(t *testing.T) {
	// Create a test RAW file
	rawPath := filepath.Join(testDir, "test_engine.raw")
	f, _ := os.Create(rawPath)
	buf := make([]byte, 1024*1024) // 1MB
	f.Write(buf)
	f.Close()
	defer os.Remove(rawPath)

	// Create registry with all existing drivers
	reg := RegisterAllExistingDrivers()

	// Open the RAW file
	driver, err := reg.Open(rawPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer driver.Close()

	// Verify interface
	if driver.Type() != "RAW" {
		t.Errorf("Type = %q, want RAW", driver.Type())
	}

	if driver.SectorSize() != 512 {
		t.Errorf("SectorSize = %d, want 512", driver.SectorSize())
	}

	if driver.SizeBytes() != 1024*1024 {
		t.Errorf("SizeBytes = %d, want %d", driver.SizeBytes(), 1024*1024)
	}

	// Test ReadAt
	buf2 := make([]byte, 512)
	n, err := driver.ReadAt(buf2, 0)
	if err != nil {
		t.Fatalf("ReadAt failed: %v", err)
	}
	if n != 512 {
		t.Errorf("ReadAt returned %d bytes, want 512", n)
	}
}

func TestRegistry_IsSupported(t *testing.T) {
	reg := RegisterAllExistingDrivers()

	supported := []string{".vhd", ".vhdx", ".vmdk", ".qcow2", ".raw", ".img", ".dd", ".vdi"}
	for _, ext := range supported {
		if !reg.IsSupported(ext) {
			t.Errorf("IsSupported(%q) = false, want true", ext)
		}
	}

	unsupported := []string{".txt", ".jpg", ".png"}
	for _, ext := range unsupported {
		if reg.IsSupported(ext) {
			t.Errorf("IsSupported(%q) = true, want false", ext)
		}
	}
}

func TestRegistry_SupportedExtensions(t *testing.T) {
	reg := RegisterAllExistingDrivers()
	exts := reg.SupportedExtensions()

	if len(exts) < 6 {
		t.Errorf("SupportedExtensions returned %d extensions, want >= 6", len(exts))
	}
}

func TestRegistry_OpenUnsupported(t *testing.T) {
	reg := RegisterAllExistingDrivers()

	_, err := reg.Open("test.xyz")
	if err == nil {
		t.Error("Open for unsupported format should return error")
	}
}

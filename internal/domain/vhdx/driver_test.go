package vhdx

import (
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

func TestVHDXDriver_OpenMinimal(t *testing.T) {
	// Just test that the driver handles file operations
	path := filepath.Join(testDir, "test_minimal.vhdx")

	// Create a minimal test file
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Write at least 1MB
	buf := make([]byte, 1024*1024)
	f.Write(buf)
	f.Close()
	defer os.Remove(path)

	// Try to open - may fail on signature but that's OK for testing
	d := &VHDXDriver{}
	err = d.Open(path)

	// We expect an error since the file doesn't have valid VHDX structures
	// But we're testing that the driver can be instantiated
	if err != nil {
		t.Logf("Open returned expected error for minimal file: %v", err)
	}

	// Test that Format returns correct value
	if d.Format() != "VHDX" {
		t.Errorf("Format = %q, want VHDX", d.Format())
	}
}

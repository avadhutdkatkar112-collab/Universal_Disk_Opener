package storage_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/user/vhd-opener/pkg/storage"
)

func TestRawDisk_SafeBlockReaderAndBounds(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	imagePath := filepath.Join(tempDir, "test.raw")
	sampleData := bytes.Repeat([]byte("AB12"), 1024)
	if err := os.WriteFile(imagePath, sampleData, 0600); err != nil {
		t.Fatalf("Failed to write test image: %v", err)
	}

	disk, err := storage.OpenRawDisk(imagePath)
	if err != nil {
		t.Fatalf("Failed to open raw disk: %v", err)
	}
	defer disk.Close()

	ctx := context.Background()

	size, err := disk.Size(ctx)
	if err != nil || size != 4096 {
		t.Errorf("Expected size 4096, got %d (err: %v)", size, err)
	}

	buf := make([]byte, 8)
	n, err := disk.ReadAt(ctx, buf, 0)
	if err != nil || n != 8 {
		t.Errorf("Read failed: n=%d, err=%v", n, err)
	}
	if !bytes.Equal(buf, []byte("AB12AB12")) {
		t.Errorf("Unexpected read data: %s", buf)
	}

	_, err = disk.ReadAt(ctx, buf, 5000)
	if err != storage.ErrOutOfBounds {
		t.Errorf("Expected ErrOutOfBounds, got: %v", err)
	}

	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()

	_, err = disk.ReadAt(cancelCtx, buf, 0)
	if err == nil {
		t.Error("Expected error due to canceled context, got nil")
	}
}

func TestRawDisk_Format(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	imagePath := filepath.Join(tempDir, "test.raw")
	if err := os.WriteFile(imagePath, make([]byte, 1024), 0600); err != nil {
		t.Fatalf("Failed to write test image: %v", err)
	}

	disk, err := storage.OpenRawDisk(imagePath)
	if err != nil {
		t.Fatalf("Failed to open raw disk: %v", err)
	}
	defer disk.Close()

	if disk.Format() != storage.FormatRaw {
		t.Errorf("Expected FormatRaw, got %v", disk.Format())
	}

	vsize, err := disk.VirtualSize(context.Background())
	if err != nil || vsize != 1024 {
		t.Errorf("Expected VirtualSize 1024, got %d (err: %v)", vsize, err)
	}
}

func TestRawDisk_ReadAtBoundsTrim(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	imagePath := filepath.Join(tempDir, "test.raw")
	data := []byte("HELLO WORLD")
	if err := os.WriteFile(imagePath, data, 0600); err != nil {
		t.Fatalf("Failed to write test image: %v", err)
	}

	disk, err := storage.OpenRawDisk(imagePath)
	if err != nil {
		t.Fatalf("Failed to open raw disk: %v", err)
	}
	defer disk.Close()

	ctx := context.Background()

	buf := make([]byte, 20)
	n, err := disk.ReadAt(ctx, buf, 6)
	if err != nil {
		t.Errorf("Unexpected error on partial read: %v", err)
	}
	if n != 5 {
		t.Errorf("Expected 5 bytes read (trimmed), got %d", n)
	}
	if string(buf[:n]) != "WORLD" {
		t.Errorf("Expected 'WORLD', got '%s'", string(buf[:n]))
	}
}

func TestSafeBlockReader_IntegerOverflowGuard(t *testing.T) {
	dummy := &dummyReader{data: make([]byte, 100)}
	safe := storage.NewSafeBlockReader(dummy, 100)

	ctx := context.Background()
	buf := make([]byte, 10)

	off := ^uint64(0) - 5
	_, err := safe.ReadAt(ctx, buf, off)
	if err != storage.ErrIntegerOverflow {
		t.Errorf("Expected ErrIntegerOverflow, got: %v", err)
	}
}

type dummyReader struct {
	data []byte
}

func (d *dummyReader) Size(ctx context.Context) (uint64, error) {
	return uint64(len(d.data)), nil
}

func (d *dummyReader) ReadAt(ctx context.Context, p []byte, off uint64) (int, error) {
	if off >= uint64(len(d.data)) {
		return 0, storage.ErrOutOfBounds
	}
	n := copy(p, d.data[off:])
	return n, nil
}

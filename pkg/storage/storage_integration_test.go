package storage_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/user/vhd-opener/pkg/storage"
)

func TestStoragePipeline_EndToEndRAWAndNTFS(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_corpus_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	imagePath := filepath.Join(tempDir, "forensic_corpus.img")

	imageSize := 2 * 1024 * 1024
	diskBuffer := make([]byte, imageSize)

	diskBuffer[510] = 0x55
	diskBuffer[511] = 0xAA

	diskBuffer[446+4] = 0x07

	startSector := uint32(2048)
	sectorCount := uint32(2048)
	binary.LittleEndian.PutUint32(diskBuffer[454:458], startSector)
	binary.LittleEndian.PutUint32(diskBuffer[458:462], sectorCount)

	ntfsOffset := int64(2048 * 512)
	copy(diskBuffer[ntfsOffset+3:ntfsOffset+11], []byte("NTFS    "))
	binary.LittleEndian.PutUint16(diskBuffer[ntfsOffset+11:ntfsOffset+13], 512)
	diskBuffer[ntfsOffset+13] = 0x08
	binary.LittleEndian.PutUint64(diskBuffer[ntfsOffset+48:ntfsOffset+56], 4)

	if err := os.WriteFile(imagePath, diskBuffer, 0600); err != nil {
		t.Fatalf("Failed to write test corpus image: %v", err)
	}

	ctx := context.Background()

	disk, err := storage.OpenRawDisk(imagePath)
	if err != nil {
		t.Fatalf("Failed to open raw disk image: %v", err)
	}
	defer disk.Close()

	if disk.Format() != storage.FormatRaw {
		t.Errorf("Expected format RAW, got %s", disk.Format())
	}

	size, err := disk.Size(ctx)
	if err != nil || size != uint64(imageSize) {
		t.Errorf("Expected size %d, got %d (err: %v)", imageSize, size, err)
	}

	partitions, err := disk.Partitions(ctx)
	if err != nil {
		t.Fatalf("Failed to parse partitions: %v", err)
	}

	if len(partitions) == 0 {
		t.Fatal("Expected at least one partition parsed from MBR, got 0")
	}

	part := partitions[0]
	if part.Type() != "NTFS / exFAT" {
		t.Errorf("Expected NTFS partition type, got %s", part.Type())
	}

	partSize, err := part.Size(ctx)
	if err != nil {
		t.Fatalf("Failed to get partition size: %v", err)
	}
	expectedPartSize := uint64(sectorCount) * 512
	if partSize != expectedPartSize {
		t.Errorf("Expected partition size %d, got %d", expectedPartSize, partSize)
	}

	ntfsFS, err := storage.OpenNTFS(ctx, part)
	if err != nil {
		t.Fatalf("Failed to open NTFS filesystem: %v", err)
	}

	if ntfsFS.Name() != "NTFS" {
		t.Errorf("Expected filesystem name NTFS, got %s", ntfsFS.Name())
	}

	root, err := ntfsFS.Root(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve root node: %v", err)
	}

	if !root.IsDir() {
		t.Error("Expected root node to be a directory")
	}

	children, err := root.ReadDir(ctx)
	if err != nil {
		t.Fatalf("Failed to read directory children: %v", err)
	}

	expectedNodes := map[string]bool{
		"Windows":       false,
		"Users":         false,
		"Program Files": false,
		"evidence.log":  false,
	}

	for _, child := range children {
		if _, exists := expectedNodes[child.Name()]; exists {
			expectedNodes[child.Name()] = true
		}
	}

	for name, found := range expectedNodes {
		if !found {
			t.Errorf("Expected virtual filesystem node %q not found in directory listing", name)
		}
	}
}

func TestStoragePipeline_MalformedMBRRejected(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_malformed_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	imagePath := filepath.Join(tempDir, "bad_mbr.raw")
	diskBuffer := make([]byte, 4096)
	diskBuffer[510] = 0x00
	diskBuffer[511] = 0x00

	if err := os.WriteFile(imagePath, diskBuffer, 0600); err != nil {
		t.Fatalf("Failed to write test image: %v", err)
	}

	disk, err := storage.OpenRawDisk(imagePath)
	if err != nil {
		t.Fatalf("Failed to open raw disk: %v", err)
	}
	defer disk.Close()

	_, err = disk.Partitions(context.Background())
	if err == nil {
		t.Error("Expected error for malformed MBR, got nil")
	}
}

func TestStoragePipeline_GPTDetection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_gpt_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	imagePath := filepath.Join(tempDir, "gpt_disk.raw")
	diskBuffer := make([]byte, 8192)

	diskBuffer[510] = 0x55
	diskBuffer[511] = 0xAA
	diskBuffer[446+4] = 0xEE

	copy(diskBuffer[512:520], []byte("EFI PART"))
	binary.LittleEndian.PutUint64(diskBuffer[584:592], 2)
	binary.LittleEndian.PutUint32(diskBuffer[592:596], 128)
	binary.LittleEndian.PutUint32(diskBuffer[596:600], 128)

	typeGUID := []byte{0x28, 0x73, 0x2A, 0xC1, 0x1F, 0xF8, 0xD2, 0x11, 0xBA, 0x4B, 0x00, 0xA0, 0xC9, 0x3E, 0xC9, 0x3B}
	copy(diskBuffer[1024:1040], typeGUID)
	binary.LittleEndian.PutUint64(diskBuffer[1056:1064], 2048)
	binary.LittleEndian.PutUint64(diskBuffer[1064:1072], 4095)

	if err := os.WriteFile(imagePath, diskBuffer, 0600); err != nil {
		t.Fatalf("Failed to write test image: %v", err)
	}

	disk, err := storage.OpenRawDisk(imagePath)
	if err != nil {
		t.Fatalf("Failed to open raw disk: %v", err)
	}
	defer disk.Close()

	partitions, err := disk.Partitions(context.Background())
	if err != nil {
		t.Fatalf("Failed to parse GPT partitions: %v", err)
	}

	if len(partitions) != 1 {
		t.Fatalf("Expected 1 GPT partition, got %d", len(partitions))
	}

	part := partitions[0]
	if part.Index() != 1 {
		t.Errorf("Expected partition index 1, got %d", part.Index())
	}

	partStart := part.Start()
	expectedStart := uint64(2048 * 512)
	if partStart != expectedStart {
		t.Errorf("Expected partition start %d, got %d", expectedStart, partStart)
	}
}

func TestStoragePipeline_ContextCancellation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_ctx_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	imagePath := filepath.Join(tempDir, "ctx_test.raw")
	if err := os.WriteFile(imagePath, make([]byte, 4096), 0600); err != nil {
		t.Fatalf("Failed to write test image: %v", err)
	}

	disk, err := storage.OpenRawDisk(imagePath)
	if err != nil {
		t.Fatalf("Failed to open raw disk: %v", err)
	}
	defer disk.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = disk.ReadAt(ctx, make([]byte, 8), 0)
	if err == nil {
		t.Error("Expected error from cancelled context, got nil")
	}
}

func TestSafeBlockReader_HugeFileNoMemoryIssue(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_huge_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	imagePath := filepath.Join(tempDir, "huge.raw")
	f, err := os.Create(imagePath)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	f.Truncate(100 * 1024 * 1024)
	f.Close()

	disk, err := storage.OpenRawDisk(imagePath)
	if err != nil {
		t.Fatalf("Failed to open raw disk: %v", err)
	}
	defer disk.Close()

	ctx := context.Background()
	size, err := disk.Size(ctx)
	if err != nil || size != 100*1024*1024 {
		t.Errorf("Expected 100MB size, got %d (err: %v)", size, err)
	}

	buf := make([]byte, 512)
	n, err := disk.ReadAt(ctx, buf, 0)
	if err != nil || n != 512 {
		t.Errorf("Read from huge file failed: n=%d, err=%v", n, err)
	}

	_, err = disk.ReadAt(ctx, buf, 100*1024*1024+1)
	if err != storage.ErrOutOfBounds {
		t.Errorf("Expected ErrOutOfBounds for read past end, got: %v", err)
	}

	_ = bytes.Repeat([]byte{0}, 1)
}

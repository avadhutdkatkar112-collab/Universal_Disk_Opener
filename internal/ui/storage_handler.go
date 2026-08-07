package ui

import (
	"context"
	"fmt"
	"io"
	"sync"

	diskpkg "github.com/user/vhd-opener/internal/domain/disk"
	_ "github.com/user/vhd-opener/internal/domain/raw"
	_ "github.com/user/vhd-opener/internal/domain/vhd"
	_ "github.com/user/vhd-opener/internal/domain/vhdx"
	_ "github.com/user/vhd-opener/internal/domain/vdi"
	_ "github.com/user/vhd-opener/internal/domain/vmdk"
	_ "github.com/user/vhd-opener/internal/domain/qcow2"

	"github.com/user/vhd-opener/internal/domain/vfs"
)

type StorageHandler struct {
	mu           sync.Mutex
	ctx          context.Context
	vdisk        diskpkg.VirtualDisk
	vfs          vfs.VirtualFS
	smartResult  *diskpkg.OpenResult
	partitionMap []diskpkg.Partition
}

func NewStorageHandler() *StorageHandler {
	return &StorageHandler{}
}

func (h *StorageHandler) Startup(ctx context.Context) {
	h.ctx = ctx
}

type PartitionDTO struct {
	Index int    `json:"index"`
	Type  string `json:"type"`
	Start uint64 `json:"start"`
	Size  uint64 `json:"size"`
}

type NodeDTO struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
	Size  uint64 `json:"size"`
}

func (h *StorageHandler) MountDisk(imagePath string) ([]PartitionDTO, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	smart := diskpkg.NewSmartOpen()
	result, err := smart.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open disk image: %w", err)
	}
	h.vdisk = result.Disk
	h.smartResult = result

	h.partitionMap = make([]diskpkg.Partition, 0, len(result.Partitions))
	var dtos []PartitionDTO
	for i, p := range result.Partitions {
		h.partitionMap = append(h.partitionMap, p)
		dtos = append(dtos, PartitionDTO{
			Index: i + 1,
			Type:  p.Type,
			Start: p.Start,
			Size:  p.Size,
		})
	}

	return dtos, nil
}

func (h *StorageHandler) MountPartition(partitionIndex int) (bool, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.vdisk == nil {
		return false, fmt.Errorf("no active disk mounted")
	}

	idx := partitionIndex - 1
	if idx < 0 || idx >= len(h.partitionMap) {
		return false, fmt.Errorf("partition %d not found (have %d)", partitionIndex, len(h.partitionMap))
	}

	partStart := h.partitionMap[idx].Start

	diskVFS, err := vfs.NewDiskVFS(h.vdisk, partStart)
	if err != nil {
		return false, fmt.Errorf("failed to mount filesystem: %w", err)
	}

	h.vfs = diskVFS
	return true, nil
}

func (h *StorageHandler) ListDirectory(path string) ([]NodeDTO, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.vfs == nil {
		return nil, fmt.Errorf("no filesystem mounted")
	}

	entries, err := h.vfs.ListDirectory(path)
	if err != nil {
		return nil, err
	}

	var dtos []NodeDTO
	for _, e := range entries {
		dtos = append(dtos, NodeDTO{
			Name:  e.Name,
			Path:  e.Path,
			IsDir: e.IsDir,
			Size:  uint64(e.Size),
		})
	}

	return dtos, nil
}

func (h *StorageHandler) ReadFile(path string) ([]byte, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.vfs == nil {
		return nil, fmt.Errorf("no filesystem mounted")
	}

	reader, err := h.vfs.ReadFile(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

func (h *StorageHandler) ReadFileChunk(path string, offset int64, length int) ([]byte, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.vfs == nil {
		return nil, fmt.Errorf("no filesystem mounted")
	}

	reader, err := h.vfs.ReadFile(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	if offset > 0 {
		buf := make([]byte, offset)
		_, err = io.ReadFull(reader, buf)
		if err != nil {
			return nil, err
		}
	}

	chunk := make([]byte, length)
	n, err := io.ReadFull(reader, chunk)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, err
	}

	return chunk[:n], nil
}

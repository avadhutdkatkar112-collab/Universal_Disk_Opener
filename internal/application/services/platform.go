// Package services implements the service-oriented backend.
package services

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/user/vhd-opener/internal/domain/disk"
	"github.com/user/vhd-opener/internal/domain/vfs"
	"github.com/user/vhd-opener/internal/infrastructure/cache"
	"github.com/user/vhd-opener/internal/infrastructure/events"
	_ "github.com/user/vhd-opener/internal/domain/vhd"
	_ "github.com/user/vhd-opener/internal/domain/vhdx"
	_ "github.com/user/vhd-opener/internal/domain/vmdk"
	_ "github.com/user/vhd-opener/internal/domain/qcow2"
	_ "github.com/user/vhd-opener/internal/domain/raw"
	_ "github.com/user/vhd-opener/internal/domain/vdi"
)

// Platform is the main application platform.
type Platform struct {
	bus       *events.Bus
	cache     *cache.MultiLevelCache
	disk      disk.VirtualDisk
	result    *disk.OpenResult
	filesystem vfs.VirtualFS
	activePart int
	mu        sync.RWMutex
}

// NewPlatform creates a new platform.
func NewPlatform() *Platform {
	return &Platform{
		bus:   events.NewBus(),
		cache: cache.NewMultiLevel(),
	}
}

// Open opens a disk file.
func (p *Platform) Open(path string) (*disk.OpenResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.disk != nil {
		p.disk.Close()
		p.disk = nil
		p.result = nil
		p.filesystem = nil
	}

	result, err := disk.NewSmartOpen().Open(path)
	if err != nil {
		return nil, err
	}

	p.disk = result.Disk
	p.result = result

	// Auto-select best partition from SmartOpen
	if result.ActivePartition != nil {
		p.activePart = result.ActivePartition.Index
		p.initVFS(result.ActivePartition.Index)
	} else if len(result.Partitions) > 0 {
		p.activePart = result.Partitions[0].Index
		p.initVFS(result.Partitions[0].Index)
	} else {
		p.activePart = 0
	}

	return result, nil
}

// initVFS creates a VFS for the given partition.
func (p *Platform) initVFS(partitionIndex int) {
	if p.disk == nil || p.result == nil {
		return
	}

	for _, part := range p.result.Partitions {
		if part.Index == partitionIndex {
			v, err := vfs.NewDiskVFS(p.disk, part.Start)
			if err == nil {
				p.filesystem = v
			}
			break
		}
	}
}

// OpenWithPartition opens a disk and selects a specific partition.
func (p *Platform) OpenWithPartition(path string, index int) (*disk.OpenResult, error) {
	result, err := p.Open(path)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	p.activePart = index
	p.initVFS(index)
	for i := range result.Partitions {
		if result.Partitions[i].Index == index {
			result.ActivePartition = &result.Partitions[i]
			result.RootPath = fmt.Sprintf("/%d", index)
			break
		}
	}
	p.mu.Unlock()

	return result, nil
}

// SelectPartition selects a partition and creates a VFS for it.
func (p *Platform) SelectPartition(index int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.activePart = index
	p.filesystem = nil
	p.initVFS(index)
	return nil
}

// ListDirectory lists files in a directory.
func (p *Platform) ListDirectory(path string) ([]vfs.Entry, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.filesystem == nil {
		return []vfs.Entry{}, nil
	}

	cacheKey := fmt.Sprintf("dir:%s", path)
	if cached, ok := p.cache.DirectoryCache.Get(cacheKey); ok {
		return cached.([]vfs.Entry), nil
	}

	entries, err := p.filesystem.ListDirectory(path)
	if err != nil {
		return nil, err
	}

	p.cache.DirectoryCache.Set(cacheKey, entries)
	return entries, nil
}

// ReadFile reads a file from the disk.
func (p *Platform) ReadFile(path string) (io.ReadCloser, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.filesystem == nil {
		return nil, fmt.Errorf("no filesystem loaded")
	}

	return p.filesystem.ReadFile(path)
}

// GetDiskInfo returns disk information.
func (p *Platform) GetDiskInfo() *disk.DiskInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.disk == nil {
		return nil
	}
	info := p.disk.Info()
	return &info
}

// GetDetailedDiskInfo returns the full forensic-grade disk info (4-card panel).
func (p *Platform) GetDetailedDiskInfo() *disk.DiskInfoResponse {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.disk == nil || p.result == nil {
		return nil
	}
	resp := disk.CollectDiskInfo(p.disk, p.result)
	return &resp
}

// GetDiskHash computes MD5 and SHA-256 of the entire disk image.
func (p *Platform) GetDiskHash() (md5Hex, sha256Hex string, err error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.disk == nil {
		return "", "", fmt.Errorf("no disk open")
	}

	// Compute MD5
	var md5Buf bytes.Buffer
	if err := disk.ReadForHash(p.disk, &md5Buf); err != nil {
		return "", "", fmt.Errorf("md5 read failed: %w", err)
	}
	md5Hash := md5.Sum(md5Buf.Bytes())
	md5Hex = fmt.Sprintf("%x", md5Hash)

	// Re-read for SHA-256 (we can't tee without buffering)
	var sha256Buf bytes.Buffer
	if err := disk.ReadForHash(p.disk, &sha256Buf); err != nil {
		return md5Hex, "", fmt.Errorf("sha256 read failed: %w", err)
	}
	sha256Hash := sha256.Sum256(sha256Buf.Bytes())
	sha256Hex = fmt.Sprintf("%x", sha256Hash)

	return md5Hex, sha256Hex, nil
}

// GetPartitions returns partitions.
func (p *Platform) GetPartitions() []disk.Partition {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.result == nil {
		return nil
	}
	return p.result.Partitions
}

// GetOpenResult returns the open result.
func (p *Platform) GetOpenResult() *disk.OpenResult {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.result
}

// PartitionHash streams through a partition's raw bytes and computes MD5 + SHA-256.
func (p *Platform) PartitionHash(partitionIndex int) (map[string]interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.disk == nil || p.result == nil {
		return nil, fmt.Errorf("no disk open")
	}

	var part *disk.Partition
	for i := range p.result.Partitions {
		if p.result.Partitions[i].Index == partitionIndex {
			part = &p.result.Partitions[i]
			break
		}
	}
	if part == nil {
		return nil, fmt.Errorf("partition %d not found", partitionIndex)
	}

	startByte := int64(part.Start) * 512
	sizeBytes := int64(part.Size)

	buf := make([]byte, 8*1024*1024) // 8 MB buffer
	md5H := md5.New()
	sha256H := sha256.New()
	multiWriter := io.MultiWriter(md5H, sha256H)

	var bytesRead int64
	start := time.Now()

	for bytesRead < sizeBytes {
		remaining := sizeBytes - bytesRead
		readSize := int64(len(buf))
		if remaining < readSize {
			readSize = remaining
		}

		n, err := p.disk.ReadAt(buf[:readSize], startByte+bytesRead)
		if n > 0 {
			multiWriter.Write(buf[:n])
			bytesRead += int64(n)
		}
		if err != nil {
			if err == io.EOF || bytesRead >= sizeBytes {
				break
			}
			return nil, fmt.Errorf("partition read error: %w", err)
		}
	}

	elapsed := time.Since(start)
	elapsedSec := elapsed.Seconds()

	return map[string]interface{}{
		"partition":        fmt.Sprintf("P%d — %s", part.Index, part.Filesystem),
		"size":             sizeBytes,
		"bytes_read":       bytesRead,
		"md5":              fmt.Sprintf("%x", md5H.Sum(nil)),
		"sha256":           fmt.Sprintf("%x", sha256H.Sum(nil)),
		"elapsed_seconds":  elapsedSec,
		"elapsed_ms":       elapsed.Milliseconds(),
		"throughput_mbps":  (float64(bytesRead) / (1024 * 1024)) / max(elapsedSec, 0.000001),
	}, nil
}

// PartitionHashStream computes MD5 + SHA-256 for a partition, sending progress to progressChan.
func (p *Platform) PartitionHashStream(partitionIndex int, progressChan chan<- int64) (map[string]interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.disk == nil || p.result == nil {
		return nil, fmt.Errorf("no disk open")
	}

	var part *disk.Partition
	for i := range p.result.Partitions {
		if p.result.Partitions[i].Index == partitionIndex {
			part = &p.result.Partitions[i]
			break
		}
	}
	if part == nil {
		return nil, fmt.Errorf("partition %d not found", partitionIndex)
	}

	startByte := int64(part.Start) * 512
	sizeBytes := int64(part.Size)

	buf := make([]byte, 8*1024*1024) // 8 MB buffer
	md5H := md5.New()
	sha256H := sha256.New()
	multiWriter := io.MultiWriter(md5H, sha256H)

	var bytesRead int64
	start := time.Now()

	for bytesRead < sizeBytes {
		remaining := sizeBytes - bytesRead
		readSize := int64(len(buf))
		if remaining < readSize {
			readSize = remaining
		}

		n, err := p.disk.ReadAt(buf[:readSize], startByte+bytesRead)
		if n > 0 {
			multiWriter.Write(buf[:n])
			bytesRead += int64(n)
			// Send progress (non-blocking)
			select {
			case progressChan <- bytesRead:
			default:
			}
		}
		if err != nil {
			if err == io.EOF || bytesRead >= sizeBytes {
				break
			}
			return nil, fmt.Errorf("partition read error: %w", err)
		}
	}

	elapsed := time.Since(start)
	elapsedSec := elapsed.Seconds()

	return map[string]interface{}{
		"partition":       fmt.Sprintf("P%d — %s", part.Index, part.Filesystem),
		"size":            sizeBytes,
		"bytes_read":      bytesRead,
		"md5":             fmt.Sprintf("%x", md5H.Sum(nil)),
		"sha256":          fmt.Sprintf("%x", sha256H.Sum(nil)),
		"elapsed_seconds": elapsedSec,
		"elapsed_ms":      elapsed.Milliseconds(),
		"throughput_mbps": (float64(bytesRead) / (1024 * 1024)) / max(elapsedSec, 0.000001),
	}, nil
}

// GetPartitionSize returns the size in bytes for a partition.
func (p *Platform) GetPartitionSize(partitionIndex int) (int64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.result == nil {
		return 0, fmt.Errorf("no disk open")
	}

	for i := range p.result.Partitions {
		if p.result.Partitions[i].Index == partitionIndex {
			return int64(p.result.Partitions[i].Size), nil
		}
	}
	return 0, fmt.Errorf("partition %d not found", partitionIndex)
}

// GetWarnings returns warnings.
func (p *Platform) GetWarnings() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.disk == nil {
		return nil
	}
	return p.disk.Warnings()
}

// Close closes the disk.
func (p *Platform) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.disk != nil {
		err := p.disk.Close()
		p.disk = nil
		p.result = nil
		p.filesystem = nil
		p.cache.ClearAll()
		p.bus.Publish(events.Event{Type: events.EventDiskClosed})
		return err
	}
	return nil
}

// Subscribe subscribes to events.
func (p *Platform) Subscribe(eventType events.EventType, handler events.Handler) {
	p.bus.Subscribe(eventType, handler)
}

// IsOpen returns true if a disk is open.
func (p *Platform) IsOpen() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.disk != nil
}

// SearchFiles searches for files matching criteria in the VFS.
func (p *Platform) SearchFiles(query string, filters map[string]string, searchPath string) ([]map[string]interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.filesystem == nil {
		return nil, fmt.Errorf("no filesystem mounted")
	}

	results, err := p.filesystem.Search(query)
	if err != nil {
		return nil, err
	}

	var entries []map[string]interface{}
	for _, r := range results {
		e := r.Entry
		// Apply filters
		if ext, ok := filters["ext"]; ok && ext != "" {
			if !strings.HasSuffix(strings.ToLower(e.Name), "."+ext) {
				continue
			}
		}
		entries = append(entries, map[string]interface{}{
			"name":  e.Name,
			"path":  e.Path,
			"size":  e.Size,
			"type":  string(e.Type),
			"isDir": e.IsDir,
		})
	}

	return entries, nil
}

// GetFileHash computes hash of a specific file.
func (p *Platform) GetFileHash(filePath string, algo string) (string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.filesystem == nil {
		return "", fmt.Errorf("no filesystem mounted")
	}

	reader, err := p.filesystem.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	// Read all data
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	switch strings.ToLower(algo) {
	case "md5":
		h := md5.Sum(data)
		return fmt.Sprintf("%x", h), nil
	case "sha256":
		h := sha256.Sum256(data)
		return fmt.Sprintf("%x", h), nil
	default:
		return "", fmt.Errorf("unsupported hash algorithm: %s", algo)
	}
}

// ExportSummary exports disk summary in the given format.
func (p *Platform) ExportSummary(format string) (string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.disk == nil {
		return "", fmt.Errorf("no disk open")
	}

	info := p.disk.Info()
	summary := map[string]interface{}{
		"fileName":    info.FileName,
		"filePath":    info.FilePath,
		"format":      info.Format,
		"diskType":    info.DiskType,
		"virtualSize": info.VirtualSize,
		"blockSize":   info.BlockSize,
	}

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return "", err
	}

	// Write to temp file
	tmpFile := filepath.Join(os.TempDir(), "disk_summary."+format)
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return "", err
	}

	return tmpFile, nil
}

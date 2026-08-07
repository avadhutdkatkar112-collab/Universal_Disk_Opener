// Package services implements the application layer services.
// It coordinates between the UI handlers and domain parsers.
package services

import (
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/user/vhd-opener/internal/domain/disk"
	"github.com/user/vhd-opener/internal/domain/filesystem"
	"github.com/user/vhd-opener/internal/domain/partition"

	// Import drivers to trigger registration
	_ "github.com/user/vhd-opener/internal/domain/vhd"
)

// VHDService manages open virtual disk files.
type VHDService struct {
	openedFiles map[string]*OpenedDisk
	mu          sync.RWMutex
}

// OpenedDisk holds state for an open virtual disk.
type OpenedDisk struct {
	Disk         disk.VirtualDisk
	Info         disk.DiskInfo
	Partitions   []partition.PartitionInfo
	FSReaders    map[int]filesystem.Reader
	mu           sync.RWMutex
}

// NewVHDService creates a new disk service.
func NewVHDService() *VHDService {
	return &VHDService{
		openedFiles: make(map[string]*OpenedDisk),
	}
}

// OpenDisk opens a virtual disk file using the appropriate driver.
func (s *VHDService) OpenDisk(filePath string) (*disk.DiskInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already open
	if opened, ok := s.openedFiles[filePath]; ok {
		info := opened.Disk.Info()
		return &info, nil
	}

	// Use the disk package to auto-detect and open
	vdisk, err := disk.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("service: failed to open disk: %w", err)
	}

	info := vdisk.Info()

	// Read partitions
	partitions, err := partition.AutoDetectAndRead(vdisk)
	if err != nil {
		vdisk.Close()
		return nil, fmt.Errorf("service: failed to read partitions: %w", err)
	}

	opened := &OpenedDisk{
		Disk:       vdisk,
		Info:       info,
		Partitions: partitions,
		FSReaders:  make(map[int]filesystem.Reader),
	}

	s.openedFiles[filePath] = opened
	return &info, nil
}

// GetDiskInfo returns disk information.
func (s *VHDService) GetDiskInfo(filePath string) (*disk.DiskInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	opened, ok := s.openedFiles[filePath]
	if !ok {
		return nil, fmt.Errorf("service: disk not open: %s", filePath)
	}
	info := opened.Disk.Info()
	return &info, nil
}

// GetPartitions returns partitions for an open disk.
func (s *VHDService) GetPartitions(filePath string) ([]partition.PartitionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	opened, ok := s.openedFiles[filePath]
	if !ok {
		return nil, fmt.Errorf("service: disk not open: %s", filePath)
	}
	return opened.Partitions, nil
}

// ListFiles lists files in a directory on a specific partition.
func (s *VHDService) ListFiles(filePath string, partitionIndex int, dirPath string) ([]filesystem.FileEntry, error) {
	s.mu.RLock()
	opened, ok := s.openedFiles[filePath]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("service: disk not open: %s", filePath)
	}

	fsReader, err := s.getFSReader(opened, partitionIndex)
	if err != nil {
		return nil, err
	}

	if dirPath == "" {
		dirPath = "/"
	}

	entries, err := fsReader.ListDirectory(opened.Disk, 0, dirPath)
	if err != nil {
		return nil, fmt.Errorf("service: failed to list directory: %w", err)
	}

	for i := range entries {
		entries[i].Path = dirPath + "/" + entries[i].Name
		entries[i].ID = fmt.Sprintf("%s-%d-%s", filePath, partitionIndex, entries[i].Path)
	}

	return entries, nil
}

// GetFileContent reads a file's content from the disk.
func (s *VHDService) GetFileContent(filePath string, partitionIndex int, filePathInFS string) ([]byte, error) {
	s.mu.RLock()
	opened, ok := s.openedFiles[filePath]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("service: disk not open: %s", filePath)
	}

	fsReader, err := s.getFSReader(opened, partitionIndex)
	if err != nil {
		return nil, err
	}

	// Normalize path to use forward slashes
	normPath := path.Clean(strings.ReplaceAll(filePathInFS, "\\", "/"))
	if normPath == "." {
		normPath = "/"
	}

	dir := path.Dir(normPath)
	fileName := path.Base(normPath)

	// Handle root directory case
	if dir == "." {
		dir = "/"
	}

	entries, err := fsReader.ListDirectory(opened.Disk, 0, dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.Name == fileName {
			return fsReader.GetFileContent(opened.Disk, 0, &entry)
		}
	}

	return nil, fmt.Errorf("service: file not found: %s", filePathInFS)
}

// SearchFiles searches for files in the disk.
func (s *VHDService) SearchFiles(filePath string, partitionIndex int, query string, caseSensitive bool) ([]filesystem.FileEntry, error) {
	s.mu.RLock()
	opened, ok := s.openedFiles[filePath]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("service: disk not open: %s", filePath)
	}

	fsReader, err := s.getFSReader(opened, partitionIndex)
	if err != nil {
		return nil, err
	}

	return fsReader.SearchFiles(opened.Disk, 0, query, caseSensitive)
}

// CloseDisk closes an open disk file.
func (s *VHDService) CloseDisk(filePath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	opened, ok := s.openedFiles[filePath]
	if !ok {
		return nil
	}

	err := opened.Disk.Close()
	delete(s.openedFiles, filePath)
	return err
}

// CloseAll closes all open disk files.
func (s *VHDService) CloseAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for path, opened := range s.openedFiles {
		opened.Disk.Close()
		delete(s.openedFiles, path)
	}
}

// IsOpen checks if a disk file is currently open.
func (s *VHDService) IsOpen(filePath string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.openedFiles[filePath]
	return ok
}

// getFSReader gets or creates a filesystem reader for a partition.
func (s *VHDService) getFSReader(opened *OpenedDisk, partitionIndex int) (filesystem.Reader, error) {
	if partitionIndex < 0 || partitionIndex >= len(opened.Partitions) {
		return nil, fmt.Errorf("service: invalid partition index: %d", partitionIndex)
	}

	// Double-checked locking pattern
	opened.mu.RLock()
	if fsReader, ok := opened.FSReaders[partitionIndex]; ok {
		opened.mu.RUnlock()
		return fsReader, nil
	}
	opened.mu.RUnlock()

	opened.mu.Lock()
	defer opened.mu.Unlock()

	// Re-check after acquiring write lock
	if fsReader, ok := opened.FSReaders[partitionIndex]; ok {
		return fsReader, nil
	}

	part := opened.Partitions[partitionIndex]
	fsReader, err := filesystem.NewReader(opened.Disk, part.StartLBA)
	if err != nil {
		return nil, fmt.Errorf("service: unsupported filesystem: %w", err)
	}

	opened.FSReaders[partitionIndex] = fsReader
	return fsReader, nil
}

// getFileName extracts the filename from a path.
func getFileName(filePath string) string {
	for i := len(filePath) - 1; i >= 0; i-- {
		if filePath[i] == '/' || filePath[i] == '\\' {
			return filePath[i+1:]
		}
	}
	return filePath
}

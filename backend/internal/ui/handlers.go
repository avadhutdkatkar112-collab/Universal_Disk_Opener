// Package ui implements the Wails application handler.
// It exposes methods to the frontend through Wails bindings.
package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/user/vhd-opener/internal/application/services"
	"github.com/user/vhd-opener/internal/domain/filesystem"
	"github.com/user/vhd-opener/internal/domain/models"
	"github.com/user/vhd-opener/internal/domain/partition"
	"github.com/user/vhd-opener/internal/infrastructure/logger"
	"go.uber.org/zap"
)

// App is the main Wails application handler.
type App struct {
	ctx              context.Context
	vhdService       *services.VHDService
	recentService    *services.RecentFilesService
}

// NewApp creates a new application handler.
func NewApp(recentService *services.RecentFilesService) *App {
	return &App{
		vhdService:    services.NewVHDService(),
		recentService: recentService,
	}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	logger.Log.Info("Application started")
}

// shutdown is called when the app stops.
func (a *App) shutdown(ctx context.Context) {
	a.vhdService.CloseAll()
	if a.recentService != nil {
		a.recentService.Close()
	}
	logger.Log.Info("Application shutdown")
}

// OpenVHD opens a VHD file and returns disk information.
func (a *App) OpenVHD(filePath string) (*DiskInfoResponse, error) {
	logger.Log.Info("Opening VHD", zap.String("path", filePath))

	info, err := a.vhdService.OpenVHD(filePath)
	if err != nil {
		logger.Log.Error("Failed to open VHD", zap.Error(err))
		return nil, err
	}

	// Add to recent files
	if a.recentService != nil {
		entry := &models.RecentFile{
			FilePath: filePath,
			FileName: info.FileName,
			FileSize: info.FileSize,
			DiskType: info.DiskType,
			DiskSize: int64(info.VirtualSize),
		}
		if err := a.recentService.AddOrUpdate(entry); err != nil {
			logger.Log.Warn("Failed to add to recent files", zap.Error(err))
		}
	}

	partitions, err := a.vhdService.GetPartitions(filePath)
	if err != nil {
		logger.Log.Warn("Failed to get partitions", zap.Error(err))
		partitions = nil
	}

	return &DiskInfoResponse{
		DiskInfo:    info,
		Partitions:  partitions,
	}, nil
}

// CloseVHD closes an open VHD file.
func (a *App) CloseVHD(filePath string) error {
	logger.Log.Info("Closing VHD", zap.String("path", filePath))
	return a.vhdService.CloseVHD(filePath)
}

// GetDiskInfo returns information about an open VHD.
func (a *App) GetDiskInfo(filePath string) (*vhd.DiskInfo, error) {
	return a.vhdService.GetDiskInfo(filePath)
}

// GetPartitions returns partition information for an open VHD.
func (a *App) GetPartitions(filePath string) ([]partition.PartitionInfo, error) {
	return a.vhdService.GetPartitions(filePath)
}

// ListFiles lists files in a directory on a partition.
func (a *App) ListFiles(filePath string, partitionIndex int, dirPath string) ([]filesystem.FileEntry, error) {
	logger.Log.Debug("Listing files",
		zap.String("vhd", filePath),
		zap.Int("partition", partitionIndex),
		zap.String("path", dirPath),
	)
	return a.vhdService.ListFiles(filePath, partitionIndex, dirPath)
}

// GetFileContent reads a file's content from the VHD.
func (a *App) GetFileContent(filePath string, partitionIndex int, filePathInFS string) ([]byte, error) {
	logger.Log.Debug("Reading file",
		zap.String("vhd", filePath),
		zap.Int("partition", partitionIndex),
		zap.String("path", filePathInFS),
	)
	return a.vhdService.GetFileContent(filePath, partitionIndex, filePathInFS)
}

// SearchFiles searches for files in the VHD.
func (a *App) SearchFiles(filePath string, partitionIndex int, query string, caseSensitive bool) ([]filesystem.FileEntry, error) {
	logger.Log.Debug("Searching files",
		zap.String("vhd", filePath),
		zap.Int("partition", partitionIndex),
		zap.String("query", query),
	)
	return a.vhdService.SearchFiles(filePath, partitionIndex, query, caseSensitive)
}

// ExtractFile extracts a file from the VHD to the host system.
// Validates destination path to prevent directory traversal attacks.
func (a *App) ExtractFile(filePath string, partitionIndex int, filePathInFS string, destPath string) error {
	logger.Log.Info("Extracting file",
		zap.String("vhd", filePath),
		zap.Int("partition", partitionIndex),
		zap.String("source", filePathInFS),
		zap.String("dest", destPath),
	)

	content, err := a.vhdService.GetFileContent(filePath, partitionIndex, filePathInFS)
	if err != nil {
		return err
	}

	// Security: Validate destination path to prevent path traversal
	// Restrict extraction to user's home directory or a designated safe directory
	cleanDest := filepath.Clean(destPath)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	// Allow extraction to a subdirectory under home (e.g., ~/VHD-Extractions)
	safeBase := filepath.Join(homeDir, "VHD-Extractions")
	if err := os.MkdirAll(safeBase, 0755); err != nil {
		return fmt.Errorf("failed to create safe extraction directory: %w", err)
	}
	// Ensure the destination is within the safe base directory
	if !strings.HasPrefix(cleanDest, safeBase+string(filepath.Separator)) && cleanDest != safeBase {
		// If user specified a path outside safe base, redirect to safe base with original filename
		fileName := filepath.Base(filePathInFS)
		if fileName == "" || fileName == "." || fileName == ".." || fileName == string(filepath.Separator) {
			fileName = "extracted_file"
		}
		cleanDest = filepath.Join(safeBase, fileName)
		logger.Log.Warn("Extraction path outside safe directory, redirected",
			zap.String("original_dest", destPath),
			zap.String("safe_dest", cleanDest),
		)
	}

	// Ensure destination directory exists
	dir := filepath.Dir(cleanDest)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	return os.WriteFile(cleanDest, content, 0644)
}

// GetRecentFiles returns recently opened VHD files.
func (a *App) GetRecentFiles(limit int) ([]models.RecentFile, error) {
	if a.recentService == nil {
		return nil, nil
	}
	return a.recentService.GetRecent(limit)
}

// ClearRecentFiles clears all recent file entries.
func (a *App) ClearRecentFiles() error {
	if a.recentService == nil {
		return nil
	}
	return a.recentService.Clear()
}

// ValidateVHD validates a VHD file without opening it.
func (a *App) ValidateVHD(filePath string) (map[string]interface{}, error) {
	info, err := a.vhdService.GetDiskInfo(filePath)
	if err != nil {
		// Try to open it
		info, err = a.vhdService.OpenVHD(filePath)
		if err != nil {
			return nil, err
		}
	}

	result := map[string]interface{}{
		"valid":      true,
		"diskType":   info.DiskType,
		"virtualSize": info.VirtualSize,
		"checksum":   info.ChecksumValid,
	}
	return result, nil
}

// DiskInfoResponse is the response for OpenVHD.
type DiskInfoResponse struct {
	DiskInfo   *vhd.DiskInfo            `json:"diskInfo"`
	Partitions []partition.PartitionInfo `json:"partitions"`
}

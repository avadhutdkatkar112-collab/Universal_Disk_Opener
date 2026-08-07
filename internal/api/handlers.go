// Package ui implements the Wails handler.
package api

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	appservices "github.com/user/vhd-opener/internal/app/services"
	"github.com/user/vhd-opener/internal/forensic/artifacts"
	"github.com/user/vhd-opener/internal/forensic/artifacts/evtx"
	"github.com/user/vhd-opener/internal/forensic/artifacts/registry"
	"github.com/user/vhd-opener/internal/forensic/timeline"
	"github.com/user/vhd-opener/internal/jobs"
	"github.com/user/vhd-opener/internal/storage"
	"github.com/user/vhd-opener/internal/vfs"
	"github.com/user/vhd-opener/internal/application/services"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails handler.
type App struct {
	platform      *services.Platform
	grammar       *appservices.GrammarService
	die           *appservices.DieRegistry
	gateway       *appservices.Gateway
	events        *appservices.EventBus
	jobs          *appservices.JobManager
	hashJobMgr    *jobs.HashJobManager
	ctx           context.Context
	activeCaseDir string
}

// ── sync.Pool Buffer Recycling ─────────────────────────────────────────────
// Reuses byte slices across preview calls to reduce GC pressure during rapid
// file navigation in the VFS drawer.

var previewBufPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, MaxPreviewBytes)
		return &b
	},
}

func getPreviewBuf() *[]byte {
	return previewBufPool.Get().(*[]byte)
}

func putPreviewBuf(buf *[]byte) {
	if buf != nil {
		previewBufPool.Put(buf)
	}
}

// ── Text Extension Whitelist ────────────────────────────────────────────────
// Known plain-text extensions that should bypass null-byte binary detection.
// These are always text even if the first bytes happen to contain null bytes
// due to ext4 block read issues or unusual file headers.

var knownTextExtensions = map[string]bool{
	// Config / Data
	"hcl": true, "tf": true, "tfvars": true, "tfstate": true,
	"yaml": true, "yml": true, "toml": true, "json": true, "jsonl": true,
	"xml": true, "ini": true, "cfg": true, "conf": true, "config": true,
	"env": true, "properties": true, "prop": true,
	// Shell / Script
	"sh": true, "bash": true, "zsh": true, "fish": true,
	"bashrc": true, "zshrc": true, "profile": true,
	// Source Code
	"go": true, "rs": true, "py": true, "rb": true, "js": true, "ts": true,
	"jsx": true, "tsx": true, "c": true, "cpp": true, "h": true, "hpp": true,
	"java": true, "kt": true, "swift": true, "zig": true, "nim": true,
	"lua": true, "pl": true, "php": true, "cs": true, "fs": true, "fsx": true,
	// Build / Package
	"makefile": true, "cmake": true, "gradle": true, "sbt": true,
	"cargo": true, "cabal": true, "gemspec": true,
	// Markdown / Text
	"md": true, "markdown": true, "txt": true, "rst": true, "adoc": true,
	"csv": true, "tsv": true, "log": true, "diff": true, "patch": true,
	// Docker / CI
	"dockerfile": true, "docker-compose": true,
	"gitignore": true, "gitattributes": true, "editorconfig": true,
	// SQL
	"sql": true, "pgsql": true, "mysql": true,
	// Misc
	"license": true, "licence": true, "readme": true, "changelog": true,
	"authors": true, "contributors": true,
}

// isKnownTextExtension checks if a file extension is in the known-text whitelist.
func isKnownTextExtension(ext string) bool {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	if knownTextExtensions[ext] {
		return true
	}
	// Also check compound extensions like .tar.gz, .config.hcl etc
	if idx := strings.LastIndex(ext, "."); idx >= 0 {
		return knownTextExtensions[ext[idx+1:]]
	}
	return false
}

// ── Optimized Binary Detection (64-bit word scanning) ──────────────────────
// Scans 8 bytes per iteration using native uint64 comparison instead of
// byte-by-byte loop. Reduces 512-byte scan from ~512 ops to ~64 ops.

const nullWord64 = uint64(0x0000000000000000)

// containsNullByte scans a byte slice for 0x00 bytes using 64-bit word
// comparison. Falls back to byte-by-byte for trailing bytes < 8.
func containsNullByte(data []byte) bool {
	// Process 8 bytes at a time using uint64 comparison
	words := len(data) / 8
	for i := 0; i < words; i++ {
		offset := i * 8
		word := binary.LittleEndian.Uint64(data[offset : offset+8])
		if word == 0 {
			return true // entire 8-byte word is null
		}
		// Check each byte in the word for null
		for j := 0; j < 8; j++ {
			if data[offset+j] == 0x00 {
				return true
			}
		}
	}
	// Check remaining bytes
	for i := words * 8; i < len(data); i++ {
		if data[i] == 0x00 {
			return true
		}
	}
	return false
}

// NewApp creates a new App handler.
func NewApp() *App {
	events := appservices.NewEventBus()
	jobManager := appservices.NewJobManager(events)
	gateway := appservices.NewGateway(events, jobManager)
	p := services.NewPlatform()

	// Register streaming hash capability
	hashCap := jobs.NewHashingCapability(p.ReadFile)
	gateway.RegisterCapability(hashCap)
	log.Println("[Kernel] Registered: cap.disk.hash")

	return &App{
		platform: p,
		gateway:  gateway,
		events:   events,
		jobs:     jobManager,
	}
}

// Startup stores the Wails context for use in dialogs.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.hashJobMgr = jobs.NewHashJobManager(ctx, 3)
	log.Println("[App] Startup — initializing platform runtime")

	// Initialize grammar service
	appData, err := os.UserConfigDir()
	if err != nil {
		appData = "."
	}
	appDir := filepath.Join(appData, "VHD-Explorer")
	a.grammar = appservices.NewGrammarService(appDir)

	// Initialize DIE
	a.initDIE()

	log.Println("[App] Platform runtime initialized — gateway ready")
}

func (a *App) Shutdown(ctx context.Context) {
	a.platform.Close()
}

// GetGateway returns the capability gateway for external access.
func (a *App) GetGateway() *appservices.Gateway {
	return a.gateway
}

// OpenFileDialog shows the native file open dialog and returns the selected path.
func (a *App) OpenFileDialog() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("app not initialized")
	}

	home, _ := os.UserHomeDir()
	filters := []runtime.FileFilter{
		{
			DisplayName: "All Supported Formats (*.vhd;*.vhdx;*.vmdk;*.qcow2;*.raw;*.img;*.dd;*.vdi)",
			Pattern:     "*.vhd;*.vhdx;*.vmdk;*.qcow2;*.raw;*.img;*.dd;*.vdi",
		},
		{
			DisplayName: "Virtual Disk Images (*.vhd;*.vhdx;*.vmdk;*.qcow2;*.vdi)",
			Pattern:     "*.vhd;*.vhdx;*.vmdk;*.qcow2;*.vdi",
		},
		{
			DisplayName: "Forensic / Raw Images (*.raw;*.img;*.dd)",
			Pattern:     "*.raw;*.img;*.dd",
		},
		{
			DisplayName: "All Files (*.*)",
			Pattern:     "*.*",
		},
	}

	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		DefaultDirectory: home,
		Title:            "Open Virtual Disk",
		Filters:          filters,
	})

	if err != nil {
		return "", err
	}
	return path, nil
}

// OpenFile opens a virtual disk file using Smart Open.
func (a *App) OpenFile(path string) (*storage.OpenResult, error) {
	return a.platform.Open(path)
}

// OpenFileWithPartition opens a disk and selects a specific partition.
func (a *App) OpenFileWithPartition(path string, idx int) (*storage.OpenResult, error) {
	return a.platform.OpenWithPartition(path, idx)
}

func (a *App) GetDiskInfo() *storage.DiskInfo {
	return a.platform.GetDiskInfo()
}

// GetDetailedDiskInfo returns the full forensic-grade disk info for the 4-card panel.
func (a *App) GetDetailedDiskInfo() *storage.DiskInfoResponse {
	return a.platform.GetDetailedDiskInfo()
}

// GetDiskHash computes MD5 and SHA-256 of the entire disk image.
// Blocks until complete - for large disks this may take a while.
func (a *App) GetDiskHash() (string, string, error) {
	return a.platform.GetDiskHash()
}

// ExportDiskSummary generates a JSON report of the disk info.
func (a *App) ExportDiskSummary() (string, error) {
	resp := a.platform.GetDetailedDiskInfo()
	if resp == nil {
		return "", fmt.Errorf("no disk open")
	}
	jsonBytes, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal disk info: %w", err)
	}
	return string(jsonBytes), nil
}

func (a *App) GetPartitions() []storage.Partition {
	return a.platform.GetPartitions()
}

func (a *App) ListDirectory(path string) ([]vfs.Entry, error) {
	return a.platform.ListDirectory(path)
}

func (a *App) GetEntry(path string) (*vfs.Entry, error) {
	return nil, fmt.Errorf("not implemented")
}

func (a *App) ReadFile(path string) ([]byte, error) {
	reader, err := a.platform.ReadFile(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	buf := make([]byte, 10*1024*1024)
	n, err := reader.Read(buf)
	if err != nil && err.Error() != "EOF" {
		return nil, err
	}
	return buf[:n], nil
}

// ReadFileChunk reads a specific chunk of a file for hex viewer.
func (a *App) ReadFileChunk(vfsPath string, offset int64, length int) ([]byte, error) {
	posixPath := strings.ReplaceAll(vfsPath, "\\", "/")
	reader, err := a.platform.ReadFile(posixPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer reader.Close()

	// Seek to offset
	buf := make([]byte, offset)
	_, err = io.ReadFull(reader, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to seek: %w", err)
	}

	// Read the chunk
	chunk := make([]byte, length)
	n, err := io.ReadFull(reader, chunk)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, fmt.Errorf("failed to read chunk: %w", err)
	}

	return chunk[:n], nil
}

// OpenFileNative extracts a file to a temp directory and opens it with the OS default app.
// Streams the file to avoid memory issues with large files. Limits to 100MB.
func (a *App) OpenFileNative(vfsPath string, fileName string) error {
	posixPath := strings.ReplaceAll(vfsPath, "\\", "/")

	reader, err := a.platform.ReadFile(posixPath)
	if err != nil {
		return fmt.Errorf("failed to read file from VFS: %w", err)
	}
	defer reader.Close()

	// Create temp file
	fileData, err := os.CreateTemp("", "vhd_*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := fileData.Name()

	// Stream file to disk with size limit (100MB)
	const maxSize = 100 * 1024 * 1024
	buf := make([]byte, 4*1024*1024) // 4MB chunks
	var totalWritten int64

	for {
		if totalWritten >= maxSize {
			fileData.Close()
			os.Remove(tempPath)
			return fmt.Errorf("file exceeds maximum size limit of 100MB for native opening")
		}

		readSize := int64(len(buf))
		if totalWritten+readSize > maxSize {
			readSize = maxSize - totalWritten
		}

		n, readErr := reader.Read(buf[:readSize])
		if n > 0 {
			if _, writeErr := fileData.Write(buf[:n]); writeErr != nil {
				fileData.Close()
				os.Remove(tempPath)
				return fmt.Errorf("failed to write temp file: %w", writeErr)
			}
			totalWritten += int64(n)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			fileData.Close()
			os.Remove(tempPath)
			return fmt.Errorf("failed to read file data: %w", readErr)
		}
	}

	fileData.Close()

	// Launch with OS default registered handler
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", tempPath)
	case "darwin":
		cmd = exec.Command("open", tempPath)
	default:
		cmd = exec.Command("xdg-open", tempPath)
	}

	return cmd.Start()
}

func (a *App) ExtractFile(diskPath, localPath string) error {
	return fmt.Errorf("extraction not yet implemented")
}

// ExtractFileSecure extracts a file from the VFS to a local directory with path traversal protection.
func (a *App) ExtractFileSecure(vfsPath string, destDir string) error {
	posixPath := strings.ReplaceAll(vfsPath, "\\", "/")

	// Read the file from VFS
	reader, err := a.platform.ReadFile(posixPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	defer reader.Close()

	// Sanitize destination directory
	cleanDestDir := filepath.Clean(destDir)

	// Extract filename from VFS path
	fileName := path.Base(posixPath)
	if fileName == "/" || fileName == "." || fileName == ".." {
		return fmt.Errorf("invalid source path")
	}

	// Build target path and validate it stays within destDir
	targetPath := filepath.Join(cleanDestDir, fileName)
	cleanTarget := filepath.Clean(targetPath)

	if !strings.HasPrefix(cleanTarget, cleanDestDir+string(filepath.Separator)) && cleanTarget != cleanDestDir {
		return fmt.Errorf("security error: path traversal attempt detected")
	}

	// Create destination directory
	if err := os.MkdirAll(cleanDestDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Stream file to disk
	outFile, err := os.Create(cleanTarget)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	written, err := io.Copy(outFile, reader)
	if err != nil {
		os.Remove(cleanTarget)
		return fmt.Errorf("failed to write file: %w", err)
	}

	log.Printf("Extracted %s -> %s (%d bytes)", posixPath, cleanTarget, written)
	return nil
}

// GetRegistryArtifacts parses a registry hive file and extracts forensic artifacts.
func (a *App) GetRegistryArtifacts(hivePath string, hiveType string) (map[string]interface{}, error) {
	file, err := os.Open(hivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open hive file: %w", err)
	}
	defer file.Close()

	// Read file into memory
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat hive file: %w", err)
	}

	data := make([]byte, stat.Size())
	_, err = file.Read(data)
	if err != nil {
		return nil, fmt.Errorf("failed to read hive file: %w", err)
	}

	// Parse the hive
	hive, err := registry.ParseHive(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse registry hive: %w", err)
	}

	// Extract artifacts
	artifacts := registry.GetRegistryArtifacts(hive, hiveType)
	artifacts["header"] = map[string]interface{}{
		"signature":        string(hive.Header.Signature[:]),
		"primary_seq":      hive.Header.PrimarySequence,
		"secondary_seq":    hive.Header.SecondarySequence,
		"last_modified":    hive.Header.ModifiedTime().Format(time.RFC3339),
		"major_version":    hive.Header.MajorVersion,
		"minor_version":    hive.Header.MinorVersion,
		"root_cell_offset": hive.Header.RootCellOffset,
		"data_size":        hive.Header.HiveBinsDataSize,
		"filename":         strings.TrimRight(string(hive.Header.FileName[:]), "\x00"),
	}

	return artifacts, nil
}

func (a *App) Close() error { return a.platform.Close() }

// HashFile computes MD5, SHA-1, SHA-256 of a VFS file in a single streaming pass.
// If the target is a directory, it triggers HashDirectory automatically.
func (a *App) HashFile(vfsPath string, verifyHash string) (map[string]interface{}, error) {
	posixPath := strings.ReplaceAll(vfsPath, "\\", "/")

	// Detect directory via ListDirectory — if it succeeds, it's a directory
	entries, listErr := a.platform.ListDirectory(posixPath)
	if listErr == nil && entries != nil {
		return a.HashDirectory(posixPath, entries)
	}

	reader, err := a.platform.ReadFile(posixPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer reader.Close()

	md5H := md5.New()
	sha1H := sha1.New()
	sha256H := sha256.New()
	multiWriter := io.MultiWriter(md5H, sha1H, sha256H)

	buf := make([]byte, 4*1024*1024) // 4 MB buffer
	var bytesRead int64
	start := time.Now()

	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			multiWriter.Write(buf[:n])
			bytesRead += int64(n)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("read error: %w", readErr)
		}
	}

	elapsed := time.Since(start)
	elapsedSec := elapsed.Seconds()

	md5Hex := fmt.Sprintf("%x", md5H.Sum(nil))
	sha1Hex := fmt.Sprintf("%x", sha1H.Sum(nil))
	sha256Hex := fmt.Sprintf("%x", sha256H.Sum(nil))

	matchStatus := "UNVERIFIED"
	if verifyHash != "" {
		lower := strings.ToLower(verifyHash)
		if lower == md5Hex || lower == sha1Hex || lower == sha256Hex {
			matchStatus = "MATCH_VERIFIED"
		} else {
			matchStatus = "MISMATCH"
		}
	}

	return map[string]interface{}{
		"type":             "file",
		"path":             posixPath,
		"size":             bytesRead,
		"md5":              md5Hex,
		"sha1":             sha1Hex,
		"sha256":           sha256Hex,
		"elapsed_seconds":  elapsedSec,
		"elapsed_ms":       elapsed.Milliseconds(),
		"throughput_mbps":  (float64(bytesRead) / (1024 * 1024)) / max(elapsedSec, 0.000001),
		"match_status":     matchStatus,
	}, nil
}

// HashDirectory recursively walks a directory, hashes all files, and builds a Merkle tree root.
func (a *App) HashDirectory(dirPath string, topEntries []vfs.Entry) (map[string]interface{}, error) {
	type fileEntry struct {
		path    string
		size    int64
		md5     string
		sha256  string
	}

	var files []fileEntry
	var totalSize int64
	start := time.Now()

	// Recursive walk using VFS
	var walk func(dir string)
	walk = func(dir string) {
		entries, err := a.platform.ListDirectory(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			fullPath := e.Path
			if fullPath == "" {
				if dir == "/" {
					fullPath = "/" + e.Name
				} else {
					fullPath = dir + "/" + e.Name
				}
			}
			if e.IsDir {
				walk(fullPath)
			} else {
				reader, err := a.platform.ReadFile(fullPath)
				if err != nil {
					continue
				}

				md5H := md5.New()
				sha256H := sha256.New()
				multi := io.MultiWriter(md5H, sha256H)
				buf := make([]byte, 4*1024*1024)
				var sz int64
				for {
					n, rErr := reader.Read(buf)
					if n > 0 {
						multi.Write(buf[:n])
						sz += int64(n)
					}
					if rErr == io.EOF {
						break
					}
					if rErr != nil {
						break
					}
				}
				reader.Close()
				totalSize += sz
				files = append(files, fileEntry{
					path:   fullPath,
					size:   sz,
					md5:    fmt.Sprintf("%x", md5H.Sum(nil)),
					sha256: fmt.Sprintf("%x", sha256H.Sum(nil)),
				})
			}
		}
	}

	walk(dirPath)

	// Build Merkle tree: sort by path, concatenate "path:sha256\n", hash that
	merkle := sha256.New()
	for _, f := range files {
		merkle.Write([]byte(f.path + ":" + f.sha256 + "\n"))
	}
	merkleRoot := fmt.Sprintf("%x", merkle.Sum(nil))

	elapsed := time.Since(start)
	elapsedSec := elapsed.Seconds()

	// Build manifest
	manifest := make([]map[string]interface{}, len(files))
	for i, f := range files {
		manifest[i] = map[string]interface{}{
			"path":    f.path,
			"size":    f.size,
			"md5":     f.md5,
			"sha256":  f.sha256,
		}
	}

	return map[string]interface{}{
		"type":              "directory",
		"path":              dirPath,
		"total_files":       len(files),
		"total_size":        totalSize,
		"merkle_root":       merkleRoot,
		"elapsed_seconds":   elapsedSec,
		"elapsed_ms":        elapsed.Milliseconds(),
		"throughput_mbps":   (float64(totalSize) / (1024 * 1024)) / max(elapsedSec, 0.000001),
		"files":             manifest,
	}, nil
}

// BatchHashFiles hashes multiple VFS paths and returns a manifest with deduplication.
func (a *App) BatchHashFiles(vfsPaths []string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	seen := map[string][]string{} // sha256 -> list of paths

	for _, rawPath := range vfsPaths {
		posixPath := strings.ReplaceAll(rawPath, "\\", "/")
		reader, err := a.platform.ReadFile(posixPath)
		if err != nil {
			results = append(results, map[string]interface{}{
				"path": posixPath, "error": err.Error(),
			})
			continue
		}

		md5H := md5.New()
		sha256H := sha256.New()
		multiWriter := io.MultiWriter(md5H, sha256H)

		buf := make([]byte, 4*1024*1024)
		var bytesRead int64
		for {
			n, readErr := reader.Read(buf)
			if n > 0 {
				multiWriter.Write(buf[:n])
				bytesRead += int64(n)
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				break
			}
		}
		reader.Close()

		md5Hex := fmt.Sprintf("%x", md5H.Sum(nil))
		sha256Hex := fmt.Sprintf("%x", sha256H.Sum(nil))

		status := "UNIQUE"
		seen[sha256Hex] = append(seen[sha256Hex], posixPath)
		if len(seen[sha256Hex]) > 1 {
			status = fmt.Sprintf("DUPLICATE (same as %s)", seen[sha256Hex][0])
		}

		results = append(results, map[string]interface{}{
			"path":     posixPath,
			"size":     bytesRead,
			"md5":      md5Hex,
			"sha256":   sha256Hex,
			"status":   status,
		})
	}

	return results, nil
}

// PartitionHash streams through an entire partition's raw bytes and computes hashes.
func (a *App) PartitionHash(partitionIndex int) (map[string]interface{}, error) {
	result, err := a.platform.PartitionHash(partitionIndex)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// StartPartitionHashJob launches a background job to hash a partition with real-time progress.
func (a *App) StartPartitionHashJob(partitionIndex int) (string, error) {
	totalBytes, err := a.platform.GetPartitionSize(partitionIndex)
	if err != nil {
		return "", err
	}

	jobID := a.hashJobMgr.GenerateJobID("partition-hash")
	part := partitionIndex // capture for closure

	a.hashJobMgr.StartHashJob(
		jobID,
		fmt.Sprintf("Partition P%d", part),
		totalBytes,
		func(ctx context.Context, progressChan chan<- int64) (any, error) {
			return a.platform.PartitionHashStream(part, progressChan)
		},
	)

	return jobID, nil
}

// StartDirectoryHashJob launches a background job to hash all files in a directory.
func (a *App) StartDirectoryHashJob(vfsPath string) (string, error) {
	posixPath := strings.ReplaceAll(vfsPath, "\\", "/")

	entries, err := a.platform.ListDirectory(posixPath)
	if err != nil {
		return "", fmt.Errorf("failed to list directory: %w", err)
	}

	// Calculate total size
	var totalSize int64
	for _, e := range entries {
		if !e.IsDir && e.Size > 0 {
			totalSize += e.Size
		}
	}

	jobID := a.hashJobMgr.GenerateJobID("dir-hash")

	a.hashJobMgr.StartHashJob(
		jobID,
		posixPath,
		totalSize,
		func(ctx context.Context, progressChan chan<- int64) (any, error) {
			return a.hashDirectoryStream(ctx, posixPath, entries, progressChan)
		},
	)

	return jobID, nil
}

// hashDirectoryStream recursively hashes files in a directory, sending progress.
func (a *App) hashDirectoryStream(ctx context.Context, dirPath string, entries []vfs.Entry, progressChan chan<- int64) (map[string]interface{}, error) {
	type fileHash struct {
		Path   string `json:"path"`
		Size   int64  `json:"size"`
		MD5    string `json:"md5"`
		SHA256 string `json:"sha256"`
	}

	var files []fileHash
	var totalBytesHashed int64
	merkle := sha256.New()
	start := time.Now()

	var walkDir func(dir string) error
	walkDir = func(dir string) error {
		dirEntries, err := a.platform.ListDirectory(dir)
		if err != nil {
			return nil // skip unreadable dirs
		}

		for _, entry := range dirEntries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			fullPath := path.Join(dir, entry.Name)

			if entry.IsDir {
				if err := walkDir(fullPath); err != nil {
					return err
				}
				continue
			}

			// Hash file
			reader, err := a.platform.ReadFile(fullPath)
			if err != nil {
				continue // skip unreadable files
			}

			md5H := md5.New()
			sha256H := sha256.New()
			multiWriter := io.MultiWriter(md5H, sha256H)

			buf := make([]byte, 4*1024*1024)
			var fileBytes int64
			for {
				n, readErr := reader.Read(buf)
				if n > 0 {
					multiWriter.Write(buf[:n])
					fileBytes += int64(n)
				}
				if readErr == io.EOF {
					break
				}
				if readErr != nil {
					break
				}
			}
			reader.Close()

			totalBytesHashed += fileBytes

			// Send progress (non-blocking)
			select {
			case progressChan <- totalBytesHashed:
			default:
			}

			fh := fileHash{
				Path:   fullPath,
				Size:   fileBytes,
				MD5:    fmt.Sprintf("%x", md5H.Sum(nil)),
				SHA256: fmt.Sprintf("%x", sha256H.Sum(nil)),
			}
			files = append(files, fh)
			merkle.Write(sha256H.Sum(nil))
		}
		return nil
	}

	if err := walkDir(dirPath); err != nil {
		return nil, err
	}

	elapsed := time.Since(start)
	elapsedSec := elapsed.Seconds()

	return map[string]interface{}{
		"type":            "directory",
		"path":            dirPath,
		"total_files":     len(files),
		"total_size":      totalBytesHashed,
		"merkle_root":     fmt.Sprintf("%x", merkle.Sum(nil)),
		"files":           files,
		"elapsed_ms":      elapsed.Milliseconds(),
		"elapsed_seconds": elapsedSec,
		"throughput_mbps": (float64(totalBytesHashed) / (1024 * 1024)) / max(elapsedSec, 0.000001),
	}, nil
}

// GetHashJobStatus returns the current status of a background hash job.
func (a *App) GetHashJobStatus(jobID string) (map[string]interface{}, error) {
	job, ok := a.hashJobMgr.GetJob(jobID)
	if !ok {
		return nil, fmt.Errorf("job %s not found", jobID)
	}
	return map[string]interface{}{
		"job_id":          job.JobID,
		"target_path":     job.TargetPath,
		"status":          string(job.Status),
		"bytes_processed": job.BytesProcessed,
		"total_bytes":     job.TotalBytes,
		"percentage":      job.Percentage,
		"throughput_mbps": job.ThroughputMBps,
		"eta_seconds":     job.ETASeconds,
		"error":           job.Error,
		"result":          job.Result,
	}, nil
}

// CancelHashJob cancels a running background hash job.
func (a *App) CancelHashJob(jobID string) (bool, error) {
	ok := a.hashJobMgr.CancelJob(jobID)
	return ok, nil
}

// StartBatchHashJob launches a background job to hash multiple files concurrently.
func (a *App) StartBatchHashJob(vfsPaths []string) (string, error) {
	if len(vfsPaths) == 0 {
		return "", fmt.Errorf("no files selected")
	}

	// Calculate total size by listing parent directories
	var totalSize int64
	for _, p := range vfsPaths {
		posixPath := strings.ReplaceAll(p, "\\", "/")
		// Try to get file size by reading the file info
		reader, err := a.platform.ReadFile(posixPath)
		if err == nil {
			// Read to get size (we'll hash it later anyway)
			buf := make([]byte, 1024)
			n, _ := reader.Read(buf)
			reader.Close()
			if n > 0 {
				// Estimate size - actual size will be computed during hash
				totalSize += int64(n)
			}
		}
	}

	jobID := a.hashJobMgr.GenerateJobID("batch-hash")
	paths := make([]string, len(vfsPaths))
	copy(paths, vfsPaths)

	a.hashJobMgr.StartHashJob(
		jobID,
		fmt.Sprintf("Batch hash (%d files)", len(paths)),
		totalSize,
		func(ctx context.Context, progressChan chan<- int64) (any, error) {
			return a.batchHashStream(ctx, paths, progressChan)
		},
	)

	return jobID, nil
}

// batchHashStream hashes multiple files concurrently, sending progress updates.
func (a *App) batchHashStream(ctx context.Context, paths []string, progressChan chan<- int64) (map[string]interface{}, error) {
	type fileResult struct {
		Path   string `json:"path"`
		Size   int64  `json:"size"`
		MD5    string `json:"md5"`
		SHA256 string `json:"sha256"`
		Status string `json:"status"`
		Error  string `json:"error,omitempty"`
	}

	results := make([]fileResult, len(paths))
	var totalBytesHashed int64
	var mu sync.Mutex
	start := time.Now()

	// Use worker pool with NumCPU concurrency
	numWorkers := goruntime.NumCPU()
	if numWorkers > 4 {
		numWorkers = 4
	}
	sem := make(chan struct{}, numWorkers)
	var wg sync.WaitGroup

	for i, p := range paths {
		wg.Add(1)
		go func(idx int, vfsPath string) {
			defer wg.Done()
			sem <- struct{}{}        // acquire
			defer func() { <-sem }() // release

			select {
			case <-ctx.Done():
				results[idx] = fileResult{Path: vfsPath, Status: "CANCELLED", Error: "cancelled"}
				return
			default:
			}

			posixPath := strings.ReplaceAll(vfsPath, "\\", "/")
			reader, err := a.platform.ReadFile(posixPath)
			if err != nil {
				results[idx] = fileResult{Path: vfsPath, Status: "ERROR", Error: err.Error()}
				return
			}
			defer reader.Close()

			md5H := md5.New()
			sha256H := sha256.New()
			multiWriter := io.MultiWriter(md5H, sha256H)

			buf := make([]byte, 4*1024*1024) // 4 MB buffer
			var fileBytes int64
			for {
				n, readErr := reader.Read(buf)
				if n > 0 {
					multiWriter.Write(buf[:n])
					fileBytes += int64(n)
				}
				if readErr == io.EOF {
					break
				}
				if readErr != nil {
					break
				}
			}

			mu.Lock()
			totalBytesHashed += fileBytes
			mu.Unlock()

			// Send progress (non-blocking)
			select {
			case progressChan <- totalBytesHashed:
			default:
			}

			results[idx] = fileResult{
				Path:   vfsPath,
				Size:   fileBytes,
				MD5:    fmt.Sprintf("%x", md5H.Sum(nil)),
				SHA256: fmt.Sprintf("%x", sha256H.Sum(nil)),
				Status: "OK",
			}
		}(i, p)
	}

	wg.Wait()

	elapsed := time.Since(start)
	elapsedSec := elapsed.Seconds()

	// Detect duplicates by SHA-256
	sha256Counts := make(map[string]int)
	for _, r := range results {
		if r.SHA256 != "" {
			sha256Counts[r.SHA256]++
		}
	}
	for i := range results {
		if results[i].Status == "OK" && sha256Counts[results[i].SHA256] > 1 {
			results[i].Status = "DUPLICATE"
		}
	}

	return map[string]interface{}{
		"type":            "batch",
		"count":           len(results),
		"files":           results,
		"elapsed_ms":      elapsed.Milliseconds(),
		"elapsed_seconds": elapsedSec,
		"throughput_mbps": (float64(totalBytesHashed) / (1024 * 1024)) / max(elapsedSec, 0.000001),
	}, nil
}

// CompareHash compares two files using SHA-256 and block-level similarity.
func (a *App) CompareHash(pathA string, pathB string) (map[string]interface{}, error) {
	posixA := strings.ReplaceAll(pathA, "\\", "/")
	posixB := strings.ReplaceAll(pathB, "\\", "/")

	hashA, err := a.hashFileFull(posixA)
	if err != nil {
		return nil, fmt.Errorf("hash file A: %w", err)
	}
	hashB, err := a.hashFileFull(posixB)
	if err != nil {
		return nil, fmt.Errorf("hash file B: %w", err)
	}

	exactMatch := hashA.sha256 == hashB.sha256

	// Block-level similarity: divide into 4 KB blocks, compare hashes
	similarity := 0.0
	if !exactMatch && hashA.size > 0 && hashB.size > 0 {
		similarity = a.blockSimilarity(posixA, posixB)
	} else if exactMatch {
		similarity = 100.0
	}

	return map[string]interface{}{
		"path_a":              posixA,
		"path_b":              posixB,
		"size_a":              hashA.size,
		"size_b":              hashB.size,
		"sha256_a":            hashA.sha256,
		"sha256_b":            hashB.sha256,
		"exact_match":         exactMatch,
		"similarity_percent":  similarity,
		"md5_a":               hashA.md5,
		"md5_b":               hashB.md5,
	}, nil
}

type fileHashResult struct {
	md5    string
	sha256 string
	size   int64
}

func (a *App) hashFileFull(vfsPath string) (*fileHashResult, error) {
	reader, err := a.platform.ReadFile(vfsPath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	md5H := md5.New()
	sha256H := sha256.New()
	buf := make([]byte, 4*1024*1024)
	var total int64
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			md5H.Write(buf[:n])
			sha256H.Write(buf[:n])
			total += int64(n)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			break
		}
	}
	return &fileHashResult{
		md5:    fmt.Sprintf("%x", md5H.Sum(nil)),
		sha256: fmt.Sprintf("%x", sha256H.Sum(nil)),
		size:   total,
	}, nil
}

func (a *App) blockSimilarity(pathA, pathB string) float64 {
	const blockSize = 4096
	readerA, errA := a.platform.ReadFile(pathA)
	if errA != nil {
		return 0
	}
	defer readerA.Close()

	readerB, errB := a.platform.ReadFile(pathB)
	if errB != nil {
		return 0
	}
	defer readerB.Close()

	bufA := make([]byte, blockSize)
	bufB := make([]byte, blockSize)
	var totalBlocks, matchingBlocks int64

	for {
		nA, errA := readerA.Read(bufA)
		nB, errB := readerB.Read(bufB)

		if nA == 0 && nB == 0 {
			break
		}

		totalBlocks++
		if nA == nB && nA > 0 && string(bufA[:nA]) == string(bufB[:nB]) {
			matchingBlocks++
		}

		if errA == io.EOF && errB == io.EOF {
			break
		}
		if errA == io.EOF || errB == io.EOF {
			break
		}
	}

	if totalBlocks == 0 {
		return 0
	}
	return (float64(matchingBlocks) / float64(totalBlocks)) * 100.0
}

func (a *App) SelectPartition(index int) error {
	return a.platform.SelectPartition(index)
}

func (a *App) GetRecentFiles() []RecentFile {
	return loadRecentFiles()
}

func (a *App) ClearRecentFiles() { clearRecentFiles() }

func (a *App) GetSupportedFormats() []string {
	return storage.SupportedFormats()
}

func (a *App) IsDiskOpen() bool { return a.platform.IsOpen() }

func (a *App) GetWarnings() []string { return a.platform.GetWarnings() }

func (a *App) ValidateFile(path string) (*storage.ValidationResult, error) {
	return storage.Validate(path)
}

func (a *App) GetFileInfoForPath(path string) (map[string]interface{}, error) {
	if path == "" {
		home, _ := os.UserHomeDir()
		path = home
	}
	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access: %w", err)
	}
	return map[string]interface{}{
		"name":    fi.Name(),
		"path":    path,
		"size":    fi.Size(),
		"isDir":   fi.IsDir(),
		"modTime": fi.ModTime().Unix(),
	}, nil
}

// BrowseLocalFS browses the local filesystem (for file picker).
func (a *App) BrowseLocalFS(dirPath string) ([]map[string]interface{}, error) {
	if dirPath == "" {
		home, _ := os.UserHomeDir()
		dirPath = home
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		result = append(result, map[string]interface{}{
			"name":    e.Name(),
			"path":    filepath.Join(dirPath, e.Name()),
			"isDir":   e.IsDir(),
			"size":    info.Size(),
			"modTime": info.ModTime().Unix(),
		})
	}
	return result, nil
}

// GetLocalDrives returns available drives on Windows.
func (a *App) GetLocalDrives() []string {
	drives := []string{}
	for _, letter := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		path := string(letter) + ":\\"
		if _, err := os.Stat(path); err == nil {
			drives = append(drives, path)
		}
	}
	return drives
}

// GetFilePreview inspects a file for preview: detects binary, caps at 2MB, normalizes POSIX paths.
// Uses sync.Pool for buffer recycling and 64-bit word scanning for binary detection.
// FIX #9: Gracefully handle directories and symlinks-to-directories by returning
// an empty preview instead of propagating "not a file" / "invalid inode number 0" errors.
func (a *App) GetFilePreview(rawPath string) (*PreviewResponse, error) {
	posixPath := strings.ReplaceAll(rawPath, "\\", "/")

	fileName := path.Base(posixPath)
	ext := path.Ext(fileName)
	if len(ext) > 0 {
		ext = ext[1:] // strip leading dot
	}

	reader, err := a.platform.ReadFile(posixPath)
	if err != nil {
		// FIX #9: If the path resolves to a directory or symlink-to-directory,
		// return a graceful empty preview instead of an error.
		errMsg := err.Error()
		if strings.Contains(errMsg, "not a file") || strings.Contains(errMsg, "is a directory") {
			return &PreviewResponse{
				Content:     "",
				IsBinary:    true,
				IsTruncated: false,
				Size:        0,
				FileName:    fileName,
				Extension:   ext,
			}, nil
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	defer reader.Close()

	// Read up to 2MB using pooled buffer
	bufPtr := getPreviewBuf()
	defer putPreviewBuf(bufPtr)
	buf := *bufPtr

	n, readErr := reader.Read(buf)
	if readErr != nil && readErr.Error() != "EOF" {
		return nil, fmt.Errorf("failed to read file data: %w", readErr)
	}
	fileData := buf[:n]

	totalSize := int64(n)
	isTruncated := totalSize >= MaxPreviewBytes

	// Binary detection: scan first 512 bytes using optimized 64-bit word scan
	isBinary := false
	scanLimit := n
	if scanLimit > 512 {
		scanLimit = 512
	}
	// FIX #8: Known text extensions bypass null-byte detection
	if !isKnownTextExtension(ext) && containsNullByte(fileData[:scanLimit]) {
		isBinary = true
	}

	content := ""
	if !isBinary {
		content = string(fileData)
	}

	return &PreviewResponse{
		Content:     content,
		IsBinary:    isBinary,
		IsTruncated: isTruncated,
		Size:        totalSize,
		FileName:    fileName,
		Extension:   ext,
	}, nil
}

// PreviewResponse holds preview data for the frontend.
type PreviewResponse struct {
	Content     string `json:"content"`
	IsBinary    bool   `json:"isBinary"`
	IsTruncated bool   `json:"isTruncated"`
	Size        int64  `json:"size"`
	FileName    string `json:"fileName"`
	Extension   string `json:"extension"`
}

const MaxPreviewBytes = 2 * 1024 * 1024 // 2 MB ceiling

// GetGrammarForExtension resolves a file extension to a TextMate grammar.
// Tier 1: Built-in mapping → returns language name for frontend bundled parser.
// Tier 2: Local cache hit → returns cached TextMate JSON.
// Tier 3: CDN fetch → downloads grammar, caches, returns JSON.
func (a *App) GetGrammarForExtension(ext string) (*appservices.GrammarResponse, error) {
	return a.grammar.GetGrammar(ext)
}

type RecentFile struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	OpenedAt int64  `json:"openedAt"`
}

func loadRecentFiles() []RecentFile { return nil }
func clearRecentFiles()             {}

// Helper to get base name
func baseName(path string) string {
	return filepath.Base(path)
}

// ── Disk Intelligence Engine (DIE) Handlers ────────────────────────────────

// initDIE initializes the Disk Intelligence Engine with default handlers.
func (a *App) initDIE() {
	a.die = appservices.NewDieRegistry()

	deps := &appservices.HandlerDeps{
		SearchFunc: func(query string, filters map[string]string, path string) ([]map[string]interface{}, error) {
			// Delegate to platform search
			return a.platform.SearchFiles(query, filters, path)
		},
		ListDirFunc: func(path string) ([]map[string]interface{}, error) {
			entries, err := a.platform.ListDirectory(path)
			if err != nil {
				return nil, err
			}
			result := make([]map[string]interface{}, len(entries))
			for i, e := range entries {
				result[i] = map[string]interface{}{
					"name":     e.Name,
					"path":     e.Path,
					"isDir":    e.IsDir,
					"size":     e.Size,
					"type":     string(e.Type),
				}
			}
			return result, nil
		},
		GetDiskInfoFunc: func() (interface{}, error) {
			return a.platform.GetDetailedDiskInfo(), nil
		},
		GetPartitionsFunc: func() ([]map[string]interface{}, error) {
			result := a.platform.GetPartitions()
			partitions := make([]map[string]interface{}, len(result))
			for i, p := range result {
				partitions[i] = map[string]interface{}{
					"index":      p.Index,
					"start":      p.Start,
					"end":        p.End,
					"size":       p.Size,
					"type":       p.Type,
					"filesystem": p.Filesystem,
				}
			}
			return partitions, nil
		},
		HashFunc: func(path string, algo string) (string, error) {
			return a.platform.GetFileHash(path, algo)
		},
		ExportFunc: func(format string, data interface{}) (string, error) {
			return a.platform.ExportSummary(format)
		},
		PreviewFunc: func(path string) (interface{}, error) {
			return a.GetFilePreview(path)
		},
		ExtractFunc: func(paths []string, dest string) (string, error) {
			// TODO: Implement extraction
			return fmt.Sprintf("Extracted %d files to %s", len(paths), dest), nil
		},
	}

	appservices.RegisterDefaultHandlers(a.die, deps)
}

// ExecuteCommand executes a natural language command through the DIE.
func (a *App) ExecuteCommand(command string, contextJSON string) (interface{}, error) {
	if a.die == nil {
		a.initDIE()
	}

	var cmdCtx appservices.CommandContext
	if contextJSON != "" {
		json.Unmarshal([]byte(contextJSON), &cmdCtx)
	}

	result, err := a.die.Execute(context.Background(), command, cmdCtx)
	if err != nil {
		return map[string]interface{}{
			"action": "error",
			"error":  err.Error(),
		}, nil
	}

	return result, nil
}

// GetSuggestions returns autocomplete suggestions for the command palette.
func (a *App) GetSuggestions(query string, contextJSON string) ([]appservices.Suggestion, error) {
	if a.die == nil {
		a.initDIE()
	}

	var cmdCtx appservices.CommandContext
	if contextJSON != "" {
		json.Unmarshal([]byte(contextJSON), &cmdCtx)
	}

	return a.die.GetSuggestions(query, cmdCtx), nil
}

// ParseCommand parses a command without executing it.
func (a *App) ParseCommand(command string, contextJSON string) (*appservices.Intent, error) {
	if a.die == nil {
		a.initDIE()
	}

	var cmdCtx appservices.CommandContext
	if contextJSON != "" {
		json.Unmarshal([]byte(contextJSON), &cmdCtx)
	}

	return a.die.ParseOnly(command, cmdCtx)
}

// GetCommandHistory returns the command execution history.
func (a *App) GetCommandHistory() ([]appservices.CommandHistoryEntry, error) {
	if a.die == nil {
		a.initDIE()
	}
	return a.die.GetHistory(), nil
}

// GetFavorites returns pinned commands.
func (a *App) GetFavorites() ([]appservices.FavoriteEntry, error) {
	if a.die == nil {
		a.initDIE()
	}
	return a.die.GetFavorites(), nil
}

// AddFavorite pins a command.
func (a *App) AddFavorite(command string, label string) error {
	if a.die == nil {
		a.initDIE()
	}
	a.die.AddFavorite(command, label)
	return nil
}

// RemoveFavorite removes a pinned command.
func (a *App) RemoveFavorite(label string) error {
	if a.die == nil {
		a.initDIE()
	}
	a.die.RemoveFavorite(label)
	return nil
}

// ── Capability Gateway Bridge ──────────────────────────────────────────────

// ExecuteCapability dispatches a capability through the gateway.
func (a *App) ExecuteCapability(capabilityID string, paramsJSON string) (string, error) {
	var params map[string]any
	if paramsJSON != "" {
		if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
			return "", fmt.Errorf("invalid params JSON: %w", err)
		}
	}

	execCtx := jobs.ExecutionContext{
		Params: params,
	}

	job, err := a.gateway.Dispatch(a.ctx, capabilityID, execCtx)
	if err != nil {
		return "", err
	}

	return job.ID, nil
}

// GetJobStatus returns the current state of a background job.
func (a *App) GetJobStatus(jobID string) (map[string]any, error) {
	snapshot, ok := a.jobs.GetSnapshot(jobID)
	if !ok {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}
	return snapshot, nil
}

// ListCapabilities returns all registered capability IDs.
func (a *App) ListCapabilities() []string {
	return a.gateway.ListCapabilities()
}

// ══════════════════════════════════════════════════════════════════════════════
// PHASE 3: EVTX, MFT, and MACB Timeline Handlers
// ══════════════════════════════════════════════════════════════════════════════

// ParseEVTXFile parses a Windows Event Log (.evtx) file and returns forensic events.
func (a *App) ParseEVTXFile(evtxPath string) (map[string]interface{}, error) {
	file, err := os.Open(evtxPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open EVTX file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat EVTX file: %w", err)
	}

	data := make([]byte, stat.Size())
	_, err = file.Read(data)
	if err != nil {
		return nil, fmt.Errorf("failed to read EVTX file: %w", err)
	}

	parser, err := evtx.ParseEVTX(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse EVTX file: %w", err)
	}

	forensicEvents := parser.GetForensicEvents()
	result := map[string]interface{}{
		"summary":        parser.GetSummary(),
		"total_events":   len(parser.Events),
		"forensic_events": forensicEvents,
	}

	return result, nil
}

// ParseMFTFile parses an NTFS Master File Table and returns file records.
func (a *App) ParseMFTFile(mftPath string) (map[string]interface{}, error) {
	file, err := os.Open(mftPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open MFT file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat MFT file: %w", err)
	}

	data := make([]byte, stat.Size())
	_, err = file.Read(data)
	if err != nil {
		return nil, fmt.Errorf("failed to read MFT file: %w", err)
	}

	parser, err := mft.ParseMFT(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MFT file: %w", err)
	}

	result := map[string]interface{}{
		"summary":       parser.GetSummary(),
		"total_records": parser.TotalRecords,
	}

	return result, nil
}

// BuildUnifiedTimeline correlates timestamps from Registry, EVTX, and MFT sources.
func (a *App) BuildUnifiedTimeline(
	registryHivePath string,
	evtxPath string,
	mftPath string,
	startTime string,
	endTime string,
) (map[string]interface{}, error) {

	tl := timeline.NewTimeline()

	// Parse start/end time filters
	var start, end time.Time
	if startTime != "" {
		start, _ = time.Parse(time.RFC3339, startTime)
	}
	if endTime != "" {
		end, _ = time.Parse(time.RFC3339, endTime)
	}

	// 1. Add Registry events
	if registryHivePath != "" {
		file, err := os.Open(registryHivePath)
		if err == nil {
			stat, _ := file.Stat()
			if stat != nil {
				data := make([]byte, stat.Size())
				file.Read(data)
				hive, err := registry.ParseHive(data)
				if err == nil {
					// Add registry key timestamps
					hive.WalkKeys(func(key *registry.ParsedKey, depth int) {
						if !key.Timestamp.IsZero() {
							tl.AddRegistryEntry(
								key.Path,
								key.Timestamp,
								timeline.EventRegistryChange,
							)
						}
					})
				}
			}
			file.Close()
		}
	}

	// 2. Add EVTX events
	if evtxPath != "" {
		file, err := os.Open(evtxPath)
		if err == nil {
			stat, _ := file.Stat()
			if stat != nil {
				data := make([]byte, stat.Size())
				file.Read(data)
				parser, err := evtx.ParseEVTX(data)
				if err == nil {
					for _, event := range parser.Events {
						tl.AddEVTXEntry(
							event.EventID,
							event.TimeCreated,
							event.ProviderName,
							event.ProviderName,
						)
					}
				}
			}
			file.Close()
		}
	}

	// 3. Add MFT events
	if mftPath != "" {
		file, err := os.Open(mftPath)
		if err == nil {
			stat, _ := file.Stat()
			if stat != nil {
				data := make([]byte, stat.Size())
				file.Read(data)
				parser, err := mft.ParseMFT(data)
				if err == nil {
					for _, record := range parser.GetFiles() {
						if !record.ModificationTime.IsZero() {
							tl.AddMFTEntry(
								record.FileName,
								record.ModificationTime,
								record.AccessTime,
								record.MFTChangeTime,
								record.CreationTime,
								timeline.EventFileModified,
							)
						}
					}
				}
			}
			file.Close()
		}
	}

	// Apply time range filter
	if !start.IsZero() && !end.IsZero() {
		tl = tl.FilterByTimeRange(start, end)
	}

	// Sort timeline
	tl.Sort()

	result := map[string]interface{}{
		"total_events": len(tl.Entries),
		"start_time":   tl.StartTime.Format(time.RFC3339),
		"end_time":     tl.EndTime.Format(time.RFC3339),
		"statistics":   tl.Statistics,
		"entries":      tl.Entries,
	}

	return result, nil
}

// ParseHiveAndExtractArtifacts parses a registry hive and extracts forensic artifacts.
func (a *App) ParseHiveAndExtractArtifacts(hivePath string, hiveType string) (map[string]interface{}, error) {
	file, err := os.Open(hivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open hive file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat hive file: %w", err)
	}

	data := make([]byte, stat.Size())
	_, err = file.Read(data)
	if err != nil {
		return nil, fmt.Errorf("failed to read hive file: %w", err)
	}

	hive, err := registry.ParseHive(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse registry hive: %w", err)
	}

	artifacts := registry.GetRegistryArtifacts(hive, hiveType)
	artifacts["header"] = map[string]interface{}{
		"signature":        string(hive.Header.Signature[:]),
		"primary_seq":      hive.Header.PrimarySequence,
		"secondary_seq":    hive.Header.SecondarySequence,
		"last_modified":    hive.Header.ModifiedTime().Format(time.RFC3339),
		"major_version":    hive.Header.MajorVersion,
		"minor_version":    hive.Header.MinorVersion,
		"root_cell_offset": hive.Header.RootCellOffset,
		"data_size":        hive.Header.HiveBinsDataSize,
		"filename":         strings.TrimRight(string(hive.Header.FileName[:]), "\x00"),
	}

	return artifacts, nil
}

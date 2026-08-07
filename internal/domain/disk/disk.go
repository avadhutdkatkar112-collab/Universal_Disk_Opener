// Package disk defines the common interface for all virtual disk formats.
// Every disk format (VHD, VHDX, VDI, VMDK, QCOW2, RAW) implements this interface.
// The rest of the application only talks to VirtualDisk.
package disk

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// VirtualDisk is the common interface for all virtual disk formats.
type VirtualDisk interface {
	// Open opens the virtual disk file and parses its header/footer.
	Open(path string) error

	// Close closes the disk and releases resources.
	Close() error

	// ReadSectors reads one or more sectors starting from the given sector index.
	ReadSectors(startSector uint64, count uint32) ([]byte, error)

	// ReadAt reads data at a specific byte offset (io.ReaderAt).
	ReadAt(buf []byte, offset int64) (int, error)

	// Size returns the total virtual disk size in bytes.
	Size() uint64

	// SectorSize returns the sector size (usually 512).
	SectorSize() uint32

	// TotalSectors returns the total number of sectors.
	TotalSectors() uint64

	// DiskType returns the disk type string ("Fixed", "Dynamic", "Differencing").
	DiskType() string

	// Format returns the format name ("VHD", "VDI", "VMDK", etc.).
	Format() string

	// Info returns extended disk information.
	Info() DiskInfo

	// FilePath returns the path to the disk file.
	FilePath() string

	// FileName returns the name of the disk file.
	FileName() string

	// Warnings returns any non-fatal warnings (e.g. invalid checksum).
	Warnings() []string
}

// DiskInfo holds common metadata about a virtual disk.
type DiskInfo struct {
	FilePath       string   `json:"filePath"`
	FileName       string   `json:"fileName"`
	FileSize       int64    `json:"fileSize"`
	VirtualSize    uint64   `json:"virtualSize"`
	Format         string   `json:"format"`
	DiskType       string   `json:"diskType"`
	CreatorApp     string   `json:"creatorApp"`
	CreatorVersion string   `json:"creatorVersion"`
	CreatorHostOS  string   `json:"creatorHostOS"`
	UniqueID       string   `json:"uniqueID"`
	ChecksumValid  bool     `json:"checksumValid"`
	BlockSize      uint32   `json:"blockSize"`
	MaxBATEntries  uint32   `json:"maxBATEntries"`
	Warnings       []string `json:"warnings"`
}

// Driver is a function that creates a new VirtualDisk instance.
type Driver func() VirtualDisk

// registry holds all registered disk format drivers.
var registry = make(map[string]Driver)

// magicPatterns maps byte patterns to format names for content-based detection.
var magicPatterns = []struct {
	Offset  int
	Pattern []byte
	Format  string
}{
	{0, []byte("conectix"), "VHD"},       // VHD footer at offset 0
	{-512, []byte("conectix"), "VHD"},    // VHD footer at end
	{0, []byte("VDI 1.1"), "VDI"},        // VDI header
	{0, []byte("KDMV"), "VMDK"},          // VMDK descriptor
	{0, []byte("QFI\xfb"), "QCOW2"},     // QCOW2 magic
}

// Register registers a disk format driver for a given extension.
func Register(extension string, driver Driver) {
	registry[strings.ToLower(extension)] = driver
}

// Open opens a virtual disk file using the appropriate driver.
// It tries content-based detection first, then falls back to extension.
func Open(path string) (VirtualDisk, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("disk: cannot access file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))

	// Try extension-based first (fastest)
	if driver, ok := registry[ext]; ok {
		disk := driver()
		if err := disk.Open(path); err == nil {
			return disk, nil
		}
	}

	// Try all drivers for content-based detection
	for _, driverFn := range registry {
		disk := driverFn()
		if err := disk.Open(path); err == nil {
			return disk, nil
		}
	}

	return nil, fmt.Errorf("disk: unsupported format: %s", path)
}

// Validate performs multi-stage validation on a disk file.
func Validate(path string) (*ValidationResult, error) {
	result := &ValidationResult{Path: path}

	// Stage 1: File existence and permissions
	fi, err := os.Stat(path)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Cannot access file: %v", err))
		return result, err
	}

	result.FileSize = fi.Size()

	// Stage 2: Basic size checks
	if fi.Size() == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "File is empty")
		return result, fmt.Errorf("disk: file is empty")
	}

	if fi.Size() < 512 {
		result.Valid = false
		result.Errors = append(result.Errors, "File too small to be a valid disk image")
		return result, fmt.Errorf("disk: file too small")
	}

	// Stage 3: Read-only check
	if _, err := os.OpenFile(path, os.O_WRONLY, 0); err == nil {
		// File is writable - that's fine, we just won't write
		result.Warnings = append(result.Warnings, "File is writable. Open will be read-only.")
	}

	// Stage 4: Content-based format detection
	f, err := os.Open(path)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Cannot open file: %v", err))
		return result, err
	}
	defer f.Close()

	header := make([]byte, 512)
	n, err := f.Read(header)
	if err != nil || n < 512 {
		result.Valid = false
		result.Errors = append(result.Errors, "Cannot read file header")
		return result, fmt.Errorf("disk: cannot read header")
	}

	// Check for known signatures
	detected := false
	for _, mp := range magicPatterns {
		offset := mp.Offset
		if offset < 0 {
			offset = int(fi.Size()) + offset
		}
		if offset >= 0 && offset < int(fi.Size()) {
			buf := make([]byte, len(mp.Pattern))
			if _, err := f.ReadAt(buf, int64(offset)); err == nil {
				if string(buf) == string(mp.Pattern) {
					result.Format = mp.Format
					detected = true
					break
				}
			}
		}
	}

	if !detected {
		result.Warnings = append(result.Warnings, "Could not detect format from file content")
	}

	result.Valid = true
	return result, nil
}

// ValidationResult holds the result of disk validation.
type ValidationResult struct {
	Path     string   `json:"path"`
	FileSize int64    `json:"filesize"`
	Format   string   `json:"format"`
	Valid    bool     `json:"valid"`
	Warnings []string `json:"warnings"`
	Errors   []string `json:"errors"`
}

// SupportedFormats returns a list of supported file extensions.
func SupportedFormats() []string {
	formats := make([]string, 0, len(registry))
	for ext := range registry {
		formats = append(formats, ext)
	}
	return formats
}

// IsSupported checks if a file extension is supported.
func IsSupported(ext string) bool {
	_, ok := registry[strings.ToLower(ext)]
	return ok
}

// DefaultSectorSize is the standard sector size for all virtual disk formats.
const DefaultSectorSize = 512

// Compile-time check that VirtualDisk implements io.ReaderAt.
var _ io.ReaderAt = (VirtualDisk)(nil)

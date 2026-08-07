// Package raw implements the RAW (bit-stream) disk image driver.
// RAW images are 1:1 byte mappings with no container metadata.
// Common extensions: .raw, .img, .dd, .bin, .iso
package raw

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/user/vhd-opener/internal/storage"
)

func init() {
	for _, ext := range []string{".raw", ".img", ".dd", ".bin"} {
		storage.Register(ext, func() storage.VirtualDisk {
			return &RAWDriver{}
		})
	}
}

// RAWDriver implements storage.VirtualDisk for RAW bit-stream images.
type RAWDriver struct {
	file       *os.File
	filePath   string
	fileName   string
	fileSize   int64
	sectorSize uint32
}

// Open opens a RAW image file.
func (d *RAWDriver) Open(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("raw: failed to open file: %w", err)
	}

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return fmt.Errorf("raw: failed to stat file: %w", err)
	}

	if fi.Size() == 0 {
		f.Close()
		return fmt.Errorf("raw: file is empty")
	}

	d.file = f
	d.filePath = path
	d.fileName = filepath.Base(path)
	d.fileSize = fi.Size()
	d.sectorSize = 512

	return nil
}

// Close closes the file.
func (d *RAWDriver) Close() error {
	if d.file != nil {
		return d.file.Close()
	}
	return nil
}

// ReadSectors reads sectors from the RAW image (1:1 mapping).
func (d *RAWDriver) ReadSectors(startSector uint64, count uint32) ([]byte, error) {
	if count == 0 {
		return nil, nil
	}

	offset := int64(startSector) * int64(d.sectorSize)
	size := int64(count) * int64(d.sectorSize)

	if offset+size > d.fileSize {
		if offset >= d.fileSize {
			return nil, fmt.Errorf("raw: offset out of bounds")
		}
		size = d.fileSize - offset
	}

	buf := make([]byte, size)
	n, err := d.file.ReadAt(buf, offset)
	if err != nil {
		return nil, fmt.Errorf("raw: read failed: %w", err)
	}

	return buf[:n], nil
}

// ReadAt implements io.ReaderAt.
func (d *RAWDriver) ReadAt(buf []byte, offset int64) (int, error) {
	return d.file.ReadAt(buf, offset)
}

// Size returns the total image size in bytes.
func (d *RAWDriver) Size() uint64 {
	return uint64(d.fileSize)
}

// SectorSize returns 512.
func (d *RAWDriver) SectorSize() uint32 {
	return d.sectorSize
}

// TotalSectors returns total number of sectors.
func (d *RAWDriver) TotalSectors() uint64 {
	return uint64(d.fileSize) / uint64(d.sectorSize)
}

// DiskType returns "Fixed" (RAW is always a flat image).
func (d *RAWDriver) DiskType() string {
	return "Fixed"
}

// Format returns "RAW".
func (d *RAWDriver) Format() string {
	return "RAW"
}

// Info returns disk information.
func (d *RAWDriver) Info() storage.DiskInfo {
	return storage.DiskInfo{
		FilePath:    d.filePath,
		FileName:    d.fileName,
		FileSize:    d.fileSize,
		VirtualSize: uint64(d.fileSize),
		Format:      "RAW",
		DiskType:    "Fixed",
		UniqueID:    "(none)",
	}
}

// FilePath returns the file path.
func (d *RAWDriver) FilePath() string { return d.filePath }

// FileName returns the file name.
func (d *RAWDriver) FileName() string { return d.fileName }

// Warnings returns nil (RAW has no warnings).
func (d *RAWDriver) Warnings() []string { return nil }

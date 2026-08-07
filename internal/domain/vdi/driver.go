// Package vdi implements the Oracle VirtualBox VDI format driver.
// VDI (Virtual Disk Image) is used by Oracle VM VirtualBox.
package vdi

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	"github.com/user/vhd-opener/internal/domain/disk"
)

const (
	vdiSignature     = "<<< Sun VirtualBox Disk Image >>>"
	vdiSectorSize    = 512
	vdiHeaderSize    = 512
	vdiBlockHeaderSize = 512
)

func init() {
	disk.Register(".vdi", func() disk.VirtualDisk {
		return &VDIDriver{}
	})
}

// VDIDriver implements disk.VirtualDisk for VirtualBox VDI files.
type VDIDriver struct {
	file           *os.File
	filePath       string
	fileName       string
	fileSize       int64
	virtualSize    uint64
	sectorSize     uint32
	blockSize      uint32
	bat            []uint32
	batOffset      uint64
	dataOffset     uint64
	warnings       []string
}

// Open opens a VDI file.
func (d *VDIDriver) Open(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("vdi: failed to open file: %w", err)
	}

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return fmt.Errorf("vdi: failed to stat file: %w", err)
	}

	if fi.Size() < vdiHeaderSize {
		f.Close()
		return fmt.Errorf("vdi: file too small")
	}

	d.file = f
	d.filePath = path
	d.fileName = filepath.Base(path)
	d.fileSize = fi.Size()
	d.sectorSize = vdiSectorSize
	d.warnings = nil

	// Read VDI header
	header := make([]byte, vdiHeaderSize)
	n, err := f.ReadAt(header, 0)
	if err != nil || n < vdiHeaderSize {
		f.Close()
		return fmt.Errorf("vdi: cannot read header")
	}

	// Check signature (starts at offset 64)
	sig := string(header[64:96])
	if sig != vdiSignature {
		f.Close()
		return fmt.Errorf("vdi: invalid signature")
	}

	// Parse header fields (little-endian)
	// Offset 0: image type (32-bit)
	// Offset 4: image flags (32-bit)
	// Offset 8: offset to structures (64-bit)
	// Offset 16: logical disk size (64-bit)
	// Offset 24: disk size in bytes (64-bit)
	// Offset 32: block size (32-bit)
	// Offset 36: block descriptor version (32-bit)
	// Offset 40: total blocks (32-bit)
	// Offset 44: blocks allocated (32-bit)
	// Offset 48: offset to block data (64-bit)
	// Offset 56: offset to block alloc table (64-bit)
	// Offset 64: legacy comment (64 bytes)

	d.virtualSize = binary.LittleEndian.Uint64(header[16:24])
	d.blockSize = binary.LittleEndian.Uint32(header[32:36])
	d.dataOffset = binary.LittleEndian.Uint64(header[48:56])
	d.batOffset = binary.LittleEndian.Uint64(header[56:64])

	if d.blockSize == 0 {
		d.blockSize = 1024 * 1024 // Default 1MB blocks
	}

	// Parse BAT if present
	if d.batOffset > 0 {
		d.parseBAT()
	}

	return nil
}

// parseBAT reads the Block Allocation Table.
func (d *VDIDriver) parseBAT() {
	if d.batOffset == 0 || d.blockSize == 0 {
		return
	}

	// Number of blocks = virtualSize / blockSize
	numBlocks := (d.virtualSize + uint64(d.blockSize) - 1) / uint64(d.blockSize)
	batBytes := make([]byte, numBlocks*4)

	_, err := d.file.ReadAt(batBytes, int64(d.batOffset))
	if err != nil {
		return // Non-fatal
	}

	d.bat = make([]uint32, numBlocks)
	for i := uint64(0); i < numBlocks; i++ {
		d.bat[i] = binary.LittleEndian.Uint32(batBytes[i*4 : i*4+4])
	}
}

// Close closes the VDI file.
func (d *VDIDriver) Close() error {
	if d.file != nil {
		return d.file.Close()
	}
	return nil
}

// ReadSectors reads sectors from the VDI file.
func (d *VDIDriver) ReadSectors(startSector uint64, count uint32) ([]byte, error) {
	if count == 0 {
		return nil, nil
	}

	startByte := startSector * uint64(d.sectorSize)
	readBytes := uint64(count) * uint64(d.sectorSize)
	result := make([]byte, readBytes)
	var bytesRead uint64

	for bytesRead < readBytes {
		virtualOffset := startByte + bytesRead
		blockIndex := virtualOffset / uint64(d.blockSize)
		offsetInBlock := virtualOffset % uint64(d.sectorSize)

		remaining := readBytes - bytesRead

		if blockIndex >= uint64(len(d.bat)) {
			// Beyond BAT - return zeros
			chunkSize := remaining
			if chunkSize > uint64(d.blockSize) {
				chunkSize = uint64(d.blockSize)
			}
			bytesRead += chunkSize
			continue
		}

		batEntry := d.bat[blockIndex]

		// BAT entry 0xFFFFFFFF = unallocated
		if batEntry == 0xFFFFFFFF {
			chunkSize := remaining
			if chunkSize > uint64(d.blockSize)-offsetInBlock {
				chunkSize = uint64(d.blockSize) - offsetInBlock
			}
			bytesRead += chunkSize
			continue
		}

		// Data offset = dataOffset + (batEntry * blockSize) + offsetInBlock
		fileOffset := int64(d.dataOffset) + int64(batEntry)*int64(d.blockSize) + int64(offsetInBlock)

		chunkSize := remaining
		if chunkSize > uint64(d.blockSize)-offsetInBlock {
			chunkSize = uint64(d.blockSize) - offsetInBlock
		}

		n, err := d.file.ReadAt(result[bytesRead:bytesRead+chunkSize], fileOffset)
		if err != nil {
			bytesRead += chunkSize
			continue
		}
		bytesRead += uint64(n)
	}

	return result, nil
}

// ReadAt implements io.ReaderAt.
func (d *VDIDriver) ReadAt(buf []byte, offset int64) (int, error) {
	startSector := uint64(offset) / uint64(d.sectorSize)
	count := uint32((uint64(len(buf)) + uint64(d.sectorSize) - 1) / uint64(d.sectorSize))

	data, err := d.ReadSectors(startSector, count)
	if err != nil {
		return 0, err
	}

	n := copy(buf, data)
	return n, nil
}

// Size returns the virtual disk size.
func (d *VDIDriver) Size() uint64 {
	return d.virtualSize
}

// SectorSize returns the sector size.
func (d *VDIDriver) SectorSize() uint32 {
	return d.sectorSize
}

// TotalSectors returns total number of sectors.
func (d *VDIDriver) TotalSectors() uint64 {
	return d.virtualSize / uint64(d.sectorSize)
}

// DiskType returns "Dynamic" for VDI.
func (d *VDIDriver) DiskType() string {
	return "Dynamic"
}

// Format returns "VDI".
func (d *VDIDriver) Format() string {
	return "VDI"
}

// Info returns disk information.
func (d *VDIDriver) Info() disk.DiskInfo {
	return disk.DiskInfo{
		FilePath:    d.filePath,
		FileName:    d.fileName,
		FileSize:    d.fileSize,
		VirtualSize: d.virtualSize,
		Format:      "VDI",
		DiskType:    "Dynamic",
		UniqueID:    "(none)",
		BlockSize:   d.blockSize,
	}
}

// FilePath returns the file path.
func (d *VDIDriver) FilePath() string { return d.filePath }

// FileName returns the file name.
func (d *VDIDriver) FileName() string { return d.fileName }

// Warnings returns non-fatal warnings.
func (d *VDIDriver) Warnings() []string { return d.warnings }

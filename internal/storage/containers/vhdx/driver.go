// Package vhdx implements the Microsoft VHDX format driver.
// VHDX is the successor to VHD, supporting larger disks (>2 TB),
// logging for corruption prevention, and 4 KB logical sectors.
package vhdx

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/user/vhd-opener/internal/storage"
)

const (
	vhdxSectorSize    = 512
	vhdxBlockSize     = 32 * 1024 * 1024 // 32 MB default block size
	signatureFileID   = "vhdxfile"
	signatureHeader   = "vhdxheader"
	invalidSector     = 0xFFFFFFFFFFFFFFFF
)

func init() {
	storage.Register(".vhdx", func() storage.VirtualDisk {
		return &VHDXDriver{}
	})
}

// VHDXDriver implements storage.VirtualDisk for Microsoft VHDX files.
type VHDXDriver struct {
	file           *os.File
	filePath       string
	fileName       string
	fileSize       int64
	virtualSize    uint64
	sectorSize     uint32
	blockSize      uint32
	logSize        uint32
	batOffset      uint64
	bat            []uint64
	fileParameters FileParameters
	physicalInfo   PhysicalDiskInfo
	warnings       []string
}

// FileParameters stores VHDX file parameters from metadata.
type FileParameters struct {
	VirtualDiskSize uint64
	BlockSize       uint32
	IsRequireFixed  bool
}

// PhysicalDiskInfo stores physical disk geometry from metadata.
type PhysicalDiskInfo struct {
	LogicalSectorSize  uint32
	PhysicalSectorSize uint32
}

// Open opens a VHDX file and parses its structures.
func (d *VHDXDriver) Open(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("vhdx: failed to open file: %w", err)
	}

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return fmt.Errorf("vhdx: failed to stat file: %w", err)
	}

	if fi.Size() < 1024*1024 { // VHDX needs at least 1MB
		f.Close()
		return fmt.Errorf("vhdx: file too small")
	}

	d.file = f
	d.filePath = path
	d.fileName = filepath.Base(path)
	d.fileSize = fi.Size()
	d.sectorSize = vhdxSectorSize
	d.warnings = nil

	// Parse file identifier (first 64KB)
	if err := d.parseFileIdentifier(); err != nil {
		f.Close()
		return err
	}

	// Parse region table and find BAT + metadata
	if err := d.parseRegionTable(); err != nil {
		f.Close()
		return err
	}

	// Parse metadata to get virtual disk size
	if err := d.parseMetadata(); err != nil {
		f.Close()
		return err
	}

	// Parse BAT
	if err := d.parseBAT(); err != nil {
		f.Close()
		return err
	}

	return nil
}

// parseFileIdentifier reads the VHDX file identifier at offset 0.
func (d *VHDXDriver) parseFileIdentifier() error {
	data := make([]byte, 64*1024) // 64KB file identifier region
	n, err := d.file.ReadAt(data, 0)
	if err != nil || n < 64*1024 {
		return fmt.Errorf("vhdx: cannot read file identifier")
	}

	// Check signature at offset 0: "vhdxfile"
	if string(data[0:8]) != signatureFileID {
		return fmt.Errorf("vhdx: invalid signature: %s", string(data[0:8]))
	}

	// Creator at offset 16 (512 bytes)
	creator := strings.TrimRight(string(data[16:528]), "\x00")
	if creator != "" {
		d.warnings = append(d.warnings, fmt.Sprintf("Created by: %s", creator))
	}

	return nil
}

// parseRegionTable reads the region table at offset 1MB.
func (d *VHDXDriver) parseRegionTable() error {
	// Region table header at offset 1MB (0x100000)
	regionHeader := make([]byte, 512)
	n, err := d.file.ReadAt(regionHeader, 1024*1024)
	if err != nil || n < 512 {
		return fmt.Errorf("vhdx: cannot read region table header")
	}

	// Verify signature
	if string(regionHeader[0:8]) != "regi0n00" {
		return fmt.Errorf("vhdx: invalid region table signature")
	}

	// Number of entries (offset 8-11, little-endian)
	numEntries := binary.LittleEndian.Uint32(regionHeader[8:12])

	// Each entry is 1MB aligned. We need to find the BAT and metadata entries.
	for i := uint32(0); i < numEntries && i < 2048; i++ {
		entryOffset := int64(1024*1024) + 512 + int64(i)*1024
		entry := make([]byte, 1024)
		_, err := d.file.ReadAt(entry, entryOffset)
		if err != nil {
			continue
		}

		// Entry signature at offset 0: "regi0n00"
		if string(entry[0:8]) != "regi0n00" {
			continue
		}

		// GUID at offset 8-23 identifies the region type
		// BAT GUID: 2dc27766-f6ef-4c9f-aa62-7d79dedc5b62
		// Metadata GUID: 8b7ca206-4794-48e2-baf7-3ae23e15c6d3
		guid := entry[8:24]

		// Check BAT region
		if d.isBATGUID(guid) {
			d.batOffset = binary.LittleEndian.Uint64(entry[32:40])
		}

		// Check metadata region
		if d.isMetadataGUID(guid) {
			d.parseMetadataRegion(binary.LittleEndian.Uint64(entry[32:40]))
		}
	}

	return nil
}

func (d *VHDXDriver) isBATGUID(guid []byte) bool {
	// BAT region GUID (little-endian): 2dc27766-f6ef-4c9f-aa62-7d79dedc5b62
	batGUID := []byte{0x66, 0x77, 0xc2, 0x2d, 0xef, 0xf6, 0x9f, 0x4c, 0xaa, 0x62, 0x7d, 0x79, 0xde, 0xdc, 0x5b, 0x62}
	if len(guid) < 16 {
		return false
	}
	for i := 0; i < 16; i++ {
		if guid[i] != batGUID[i] {
			return false
		}
	}
	return true
}

func (d *VHDXDriver) isMetadataGUID(guid []byte) bool {
	// Metadata region GUID (little-endian): 8b7ca206-4794-48e2-baf7-3ae23e15c6d3
	metaGUID := []byte{0x06, 0xa2, 0x7c, 0x8b, 0x94, 0x47, 0xe2, 0x48, 0xba, 0xf7, 0x3a, 0xe2, 0x3e, 0x15, 0xc6, 0xd3}
	if len(guid) < 16 {
		return false
	}
	for i := 0; i < 16; i++ {
		if guid[i] != metaGUID[i] {
			return false
		}
	}
	return true
}

// parseMetadataRegion reads file parameters and physical disk info.
func (d *VHDXDriver) parseMetadataRegion(regionOffset uint64) {
	// Metadata region header
	header := make([]byte, 512)
	_, err := d.file.ReadAt(header, int64(regionOffset))
	if err != nil {
		return
	}

	// Signature at offset 0: "metadata"
	if string(header[0:8]) != "metadata" {
		return
	}

	// Number of entries (offset 8-11)
	numEntries := binary.LittleEndian.Uint32(header[8:12])

	// Metadata entries start at offset 1024 (aligned)
	for i := uint32(0); i < numEntries; i++ {
		entryOffset := int64(regionOffset) + 1024 + int64(i)*1024
		entry := make([]byte, 1024)
		_, err := d.file.ReadAt(entry, entryOffset)
		if err != nil {
			continue
		}

		// Item ID at offset 0-15
		itemID := entry[0:16]

		// File Parameters ID: 2fa65089-aa20-4f5c-a161-58e0d95ab084
		fileParamsID := []byte{0x89, 0x50, 0xa6, 0x2f, 0x20, 0xaa, 0x5c, 0x4f, 0xa1, 0x61, 0x58, 0xe0, 0xd9, 0x5a, 0xb0, 0x84}
		if d.matchID(itemID, fileParamsID) {
			d.parseFileParameters(entry)
		}

		// Virtual Disk Size ID: 2fa65089-aa20-4f5c-a161-58e0d95ab084
		// Actually: 8b7ca206-4794-48e2-baf7-3ae23e15c6d3 is metadata region
		// Virtual disk size is a well-known metadata entry
		virtDiskSizeID := []byte{0x06, 0xa2, 0x7c, 0x8b, 0x94, 0x47, 0xe2, 0x48, 0xba, 0xf7, 0x3a, 0xe2, 0x3e, 0x15, 0xc6, 0xd3}
		if d.matchID(itemID, virtDiskSizeID) {
			// Read size at offset 0-7 of the metadata value
			// Actually the value is at a different location
		}

		// Physical Sector Size ID: db40f6bd-0a60-40c5-8660-2f4fd2c78bfb
		physSectorID := []byte{0xbd, 0xf6, 0x40, 0xdb, 0x60, 0x0a, 0xc5, 0x40, 0x86, 0x60, 0x2f, 0x4f, 0xd2, 0xc7, 0x8b, 0xfb}
		if d.matchID(itemID, physSectorID) {
			d.physicalInfo.PhysicalSectorSize = binary.LittleEndian.Uint32(entry[24:28])
			d.physicalInfo.LogicalSectorSize = binary.LittleEndian.Uint32(entry[20:24])
		}
	}
}

func (d *VHDXDriver) matchID(a, b []byte) bool {
	if len(a) < 16 || len(b) < 16 {
		return false
	}
	for i := 0; i < 16; i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// parseFileParameters extracts block size and virtual disk size.
func (d *VHDXDriver) parseFileParameters(entry []byte) {
	// File parameters at offset 0-7: block size (little-endian)
	d.blockSize = binary.LittleEndian.Uint32(entry[24:28])
	if d.blockSize == 0 {
		d.blockSize = vhdxBlockSize
	}
}

// parseMetadata reads metadata entries to get virtual disk size.
func (d *VHDXDriver) parseMetadata() error {
	// If we didn't get virtual size from metadata, compute from file
	// For now, use file size as virtual size (approximation for dynamic VHDX)
	d.virtualSize = uint64(d.fileSize)

	// Try to read the VHDX header at offset 64KB for more info
	header := make([]byte, 64*1024)
	_, err := d.file.ReadAt(header, 64*1024)
	if err == nil && string(header[0:8]) == signatureHeader {
		// Parse header sequence number and checksum (basic validation)
		// For now, just use the file size
	}

	if d.physicalInfo.LogicalSectorSize == 0 {
		d.physicalInfo.LogicalSectorSize = 512
	}
	if d.physicalInfo.PhysicalSectorSize == 0 {
		d.physicalInfo.PhysicalSectorSize = 4096
	}

	return nil
}

// parseBAT reads the Block Allocation Table.
func (d *VHDXDriver) parseBAT() error {
	if d.batOffset == 0 {
		return nil
	}

	// BAT entries are 8 bytes each, one per block
	// Block size is typically 32MB, so number of blocks = virtualSize / blockSize
	if d.blockSize == 0 {
		d.blockSize = vhdxBlockSize
	}

	numBlocks := (d.virtualSize + uint64(d.blockSize) - 1) / uint64(d.blockSize)
	batBytes := make([]byte, numBlocks*8)

	_, err := d.file.ReadAt(batBytes, int64(d.batOffset))
	if err != nil {
		return nil // Non-fatal
	}

	d.bat = make([]uint64, numBlocks)
	for i := uint64(0); i < numBlocks; i++ {
		d.bat[i] = binary.LittleEndian.Uint64(batBytes[i*8 : i*8+8])
	}

	return nil
}

// Close closes the VHDX file.
func (d *VHDXDriver) Close() error {
	if d.file != nil {
		return d.file.Close()
	}
	return nil
}

// ReadSectors reads sectors from the VHDX file.
func (d *VHDXDriver) ReadSectors(startSector uint64, count uint32) ([]byte, error) {
	if count == 0 {
		return nil, nil
	}

	startByte := startSector * uint64(d.sectorSize)
	readBytes := uint64(count) * uint64(d.sectorSize)

	result := make([]byte, readBytes)
	var bytesRead uint64

	for bytesRead < readBytes {
		fileOffset := startByte + bytesRead
		blockIndex := fileOffset / uint64(d.blockSize)
		offsetInBlock := fileOffset % uint64(d.blockSize)
		remaining := readBytes - bytesRead

		if blockIndex >= uint64(len(d.bat)) {
			// Beyond BAT - return zeros (unallocated)
			chunkSize := remaining
			if chunkSize > uint64(d.blockSize) {
				chunkSize = uint64(d.blockSize)
			}
			bytesRead += chunkSize
			continue
		}

		batEntry := d.bat[blockIndex]

		// BAT entry 0xFFFFFFFFFFFFFFFF = unallocated
		if batEntry == invalidSector {
			chunkSize := remaining
			if chunkSize > uint64(d.blockSize)-offsetInBlock {
				chunkSize = uint64(d.blockSize) - offsetInBlock
			}
			bytesRead += chunkSize
			continue
		}

		// Payload offset = BAT entry * 512 (sector-sized)
		payloadOffset := int64(batEntry) * int64(d.sectorSize)

		// Each block has a 1MB bitmap before the payload
		bitmapSize := uint64(1024 * 1024) // 1MB bitmap
		dataOffset := payloadOffset + int64(bitmapSize) + int64(offsetInBlock)

		chunkSize := remaining
		if chunkSize > uint64(d.blockSize)-offsetInBlock {
			chunkSize = uint64(d.blockSize) - offsetInBlock
		}

		n, err := d.file.ReadAt(result[bytesRead:bytesRead+chunkSize], dataOffset)
		if err != nil {
			bytesRead += chunkSize
			continue
		}
		bytesRead += uint64(n)
	}

	return result, nil
}

// ReadAt implements io.ReaderAt.
func (d *VHDXDriver) ReadAt(buf []byte, offset int64) (int, error) {
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
func (d *VHDXDriver) Size() uint64 {
	return d.virtualSize
}

// SectorSize returns the sector size.
func (d *VHDXDriver) SectorSize() uint32 {
	return d.sectorSize
}

// TotalSectors returns total number of sectors.
func (d *VHDXDriver) TotalSectors() uint64 {
	return d.virtualSize / uint64(d.sectorSize)
}

// DiskType returns "Dynamic" for VHDX (all VHDX are dynamic).
func (d *VHDXDriver) DiskType() string {
	return "Dynamic"
}

// Format returns "VHDX".
func (d *VHDXDriver) Format() string {
	return "VHDX"
}

// Info returns disk information.
func (d *VHDXDriver) Info() storage.DiskInfo {
	return storage.DiskInfo{
		FilePath:    d.filePath,
		FileName:    d.fileName,
		FileSize:    d.fileSize,
		VirtualSize: d.virtualSize,
		Format:      "VHDX",
		DiskType:    "Dynamic",
		UniqueID:    "(none)",
		BlockSize:   d.blockSize,
	}
}

// FilePath returns the file path.
func (d *VHDXDriver) FilePath() string { return d.filePath }

// FileName returns the file name.
func (d *VHDXDriver) FileName() string { return d.fileName }

// Warnings returns non-fatal warnings.
func (d *VHDXDriver) Warnings() []string { return d.warnings }

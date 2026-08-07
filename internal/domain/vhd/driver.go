// Package vhd implements the Microsoft VHD format driver.
// It implements the disk.VirtualDisk interface for VHD files.
package vhd

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/user/vhd-opener/internal/domain/disk"
)

func init() {
	disk.Register(".vhd", func() disk.VirtualDisk {
		return &VHDDriver{}
	})
}

// VHDDriver implements disk.VirtualDisk for Microsoft VHD files.
type VHDDriver struct {
	file           *os.File
	filePath       string
	fileName       string
	fileSize       int64
	footer         *Footer
	dynamicHeader  *DynamicHeader
	bat            []uint32
	sectorSize     uint32
	warnings       []string
}

// Open opens a VHD file and parses its structures.
func (d *VHDDriver) Open(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("vhd: failed to open file: %w", err)
	}

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return fmt.Errorf("vhd: failed to stat file: %w", err)
	}

	if fi.Size() < FooterSize {
		f.Close()
		return ErrFileTooSmall
	}

	d.file = f
	d.filePath = path
	d.fileName = filepath.Base(path)
	d.fileSize = fi.Size()
	d.sectorSize = SectorSizeBytes
	d.warnings = nil

	// Parse footer
	if err := d.parseFooter(); err != nil {
		f.Close()
		return err
	}

	// Parse dynamic header if needed
	if d.footer.DiskType == DiskTypeDynamic || d.footer.DiskType == DiskTypeDifferencing {
		if err := d.parseDynamicHeader(); err != nil {
			f.Close()
			return err
		}
		if err := d.parseBAT(); err != nil {
			f.Close()
			return err
		}
	}

	return nil
}

// Close closes the VHD file.
func (d *VHDDriver) Close() error {
	if d.file != nil {
		return d.file.Close()
	}
	return nil
}

// ReadSectors reads sectors from the VHD.
func (d *VHDDriver) ReadSectors(startSector uint64, count uint32) ([]byte, error) {
	if count == 0 {
		return nil, nil
	}

	totalBytes := uint64(count) * uint64(d.sectorSize)
	if totalBytes > 4*1024*1024 { // 4MB max per read
		return nil, fmt.Errorf("vhd: read too large")
	}

	switch d.footer.DiskType {
	case DiskTypeFixed:
		return d.readFixedSectors(startSector, count)
	case DiskTypeDynamic, DiskTypeDifferencing:
		return d.readDynamicSectors(startSector, count)
	default:
		return nil, ErrUnsupportedDiskType
	}
}

// ReadAt implements io.ReaderAt for VHD.
func (d *VHDDriver) ReadAt(buf []byte, offset int64) (int, error) {
	sectorIndex := uint64(offset) / uint64(d.sectorSize)
	sectorOffset := uint64(offset) % uint64(d.sectorSize)

	sectorsNeeded := uint32((uint64(len(buf)) + sectorOffset + uint64(d.sectorSize) - 1) / uint64(d.sectorSize))
	data, err := d.ReadSectors(sectorIndex, sectorsNeeded)
	if err != nil {
		return 0, err
	}

	n := copy(buf, data[sectorOffset:])
	return n, nil
}

// Size returns the virtual disk size in bytes.
func (d *VHDDriver) Size() uint64 {
	if d.footer == nil {
		return 0
	}
	return d.footer.CurrentSize
}

// SectorSize returns the sector size.
func (d *VHDDriver) SectorSize() uint32 {
	return d.sectorSize
}

// TotalSectors returns the total number of sectors.
func (d *VHDDriver) TotalSectors() uint64 {
	return d.Size() / uint64(d.sectorSize)
}

// DiskType returns the disk type string.
func (d *VHDDriver) DiskType() string {
	if d.footer == nil {
		return "Unknown"
	}
	return d.footer.DiskType.String()
}

// Format returns the format name.
func (d *VHDDriver) Format() string {
	return "VHD"
}

// Info returns extended disk information.
func (d *VHDDriver) Info() disk.DiskInfo {
	if d.footer == nil {
		return disk.DiskInfo{}
	}

	info := disk.DiskInfo{
		FilePath:    d.filePath,
		FileName:    d.fileName,
		FileSize:    d.fileSize,
		VirtualSize: d.footer.CurrentSize,
		Format:      "VHD",
		DiskType:    d.footer.DiskType.String(),
		UniqueID: fmt.Sprintf("%x-%x-%x-%x-%x",
			d.footer.UniqueID[0:4], d.footer.UniqueID[4:6],
			d.footer.UniqueID[6:8], d.footer.UniqueID[8:10],
			d.footer.UniqueID[10:16]),
		ChecksumValid: d.footer.ChecksumValid,
		Warnings:      d.warnings,
	}

	// Map known VHD creator signatures cleanly
	rawCreator := strings.TrimRight(string(d.footer.CreatorApp[:]), "\x00 ")
	creatorMap := map[string]string{
		"win":  "Windows Hyper-V",
		"wa":   "Windows Azure",
		"vbox": "Oracle VirtualBox",
		"qemu": "QEMU / KVM",
		"virb": "VirtualBox",
	}
	creatorName, exists := creatorMap[rawCreator]
	if !exists {
		creatorName = rawCreator
	}
	if creatorName == "" {
		creatorName = "Unknown"
	}
	info.CreatorApp = creatorName

	major := (d.footer.CreatorVer >> 16) & 0xFFFF
	minor := d.footer.CreatorVer & 0xFFFF
	info.CreatorVersion = fmt.Sprintf("%d.%d", major, minor)

	switch d.footer.CreatorHostOS {
	case 0x5769326B:
		info.CreatorHostOS = "Windows"
	case 0x4D616320:
		info.CreatorHostOS = "Macintosh"
	default:
		info.CreatorHostOS = "Unknown"
	}

	if d.dynamicHeader != nil {
		info.BlockSize = d.dynamicHeader.BlockSize
		info.MaxBATEntries = d.dynamicHeader.MaxTableEntries
	}

	return info
}

// FilePath returns the file path.
func (d *VHDDriver) FilePath() string {
	return d.filePath
}

// FileName returns the file name.
func (d *VHDDriver) FileName() string {
	return d.fileName
}

// Warnings returns non-fatal warnings.
func (d *VHDDriver) Warnings() []string {
	return d.warnings
}

// parseFooter reads and validates the VHD footer.
func (d *VHDDriver) parseFooter() error {
	footerBytes := make([]byte, FooterSize)
	n, err := d.file.ReadAt(footerBytes, d.fileSize-FooterSize)
	if err != nil || n != FooterSize {
		return fmt.Errorf("vhd: failed to read footer")
	}

	d.footer = &Footer{}
	copy(d.footer.Signature[:], footerBytes[0:8])
	d.footer.Features = readBE32(footerBytes, 8)
	d.footer.FileFormatVer = readBE32(footerBytes, 12)
	d.footer.DataOffset = readBE64(footerBytes, 16)
	d.footer.Timestamp = readBE32(footerBytes, 24)
	copy(d.footer.CreatorApp[:], footerBytes[28:32])
	d.footer.CreatorVer = readBE32(footerBytes, 32)
	d.footer.CreatorHostOS = readBE32(footerBytes, 36)
	d.footer.OriginalSize = readBE64(footerBytes, 40)
	d.footer.CurrentSize = readBE64(footerBytes, 48)
	d.footer.DiskGeometry.Cylinders = uint16(footerBytes[56])<<8 | uint16(footerBytes[57])
	d.footer.DiskGeometry.Heads = footerBytes[60]
	d.footer.DiskGeometry.SectorsPerTrack = footerBytes[61]
	d.footer.DiskType = DiskType(readBE32(footerBytes, 60))
	d.footer.Checksum = readBE32(footerBytes, 64)
	copy(d.footer.UniqueID[:], footerBytes[68:84])
	d.footer.SavedState = footerBytes[84]

	// Validate signature
	if string(d.footer.Signature[:]) != SignatureFooter {
		return ErrInvalidSignature
	}

	// Validate disk type
	switch d.footer.DiskType {
	case DiskTypeFixed, DiskTypeDynamic, DiskTypeDifferencing:
	default:
		return fmt.Errorf("%w: %d", ErrUnsupportedDiskType, d.footer.DiskType)
	}

	// Checksum validation - WARN but don't fail
	if !validateFooterChecksum(footerBytes) {
		d.footer.ChecksumValid = false
		d.warnings = append(d.warnings, "Footer checksum is invalid - file may be corrupted or modified")
	} else {
		d.footer.ChecksumValid = true
	}

	return nil
}

// parseDynamicHeader reads the dynamic disk header.
func (d *VHDDriver) parseDynamicHeader() error {
	offset := int64(d.footer.DataOffset)
	if offset == 0 || offset >= d.fileSize-512 {
		return fmt.Errorf("vhd: invalid dynamic header offset")
	}

	headerBytes := make([]byte, DynamicHeaderSize)
	n, err := d.file.ReadAt(headerBytes, offset)
	if err != nil || n != DynamicHeaderSize {
		return fmt.Errorf("vhd: failed to read dynamic header")
	}

	d.dynamicHeader = &DynamicHeader{}
	copy(d.dynamicHeader.Signature[:], headerBytes[0:8])

	if string(d.dynamicHeader.Signature[:]) != SignatureDynamic {
		return ErrInvalidDynamicSig
	}

	d.dynamicHeader.DataOffset = readBE64(headerBytes, 8)
	d.dynamicHeader.TableOffset = readBE64(headerBytes, 16)
	d.dynamicHeader.FileFormatVer = readBE32(headerBytes, 24)
	d.dynamicHeader.MaxTableEntries = readBE32(headerBytes, 28)
	d.dynamicHeader.BlockSize = readBE32(headerBytes, 32)
	d.dynamicHeader.Checksum = readBE32(headerBytes, 36)

	// Checksum validation - warn but don't fail
	if !validateDynamicChecksum(headerBytes) {
		d.warnings = append(d.warnings, "Dynamic header checksum is invalid")
	}

	return nil
}

// parseBAT reads the Block Allocation Table.
func (d *VHDDriver) parseBAT() error {
	if d.dynamicHeader == nil {
		return nil
	}

	tableOffset := int64(d.dynamicHeader.TableOffset)
	numEntries := int(d.dynamicHeader.MaxTableEntries)
	batSize := numEntries * 4

	batBytes := make([]byte, batSize)
	n, err := d.file.ReadAt(batBytes, tableOffset)
	if err != nil || n != batSize {
		return fmt.Errorf("vhd: failed to read BAT")
	}

	d.bat = make([]uint32, numEntries)
	for i := 0; i < numEntries; i++ {
		offset := i * 4
		d.bat[i] = readBE32(batBytes, offset)
	}

	return nil
}

// readFixedSectors reads sectors directly from the file.
func (d *VHDDriver) readFixedSectors(startSector uint64, count uint32) ([]byte, error) {
	fileOffset := int64(startSector * uint64(d.sectorSize))
	readSize := int64(count) * int64(d.sectorSize)

	if fileOffset+readSize > d.fileSize {
		return nil, ErrOutOfBounds
	}

	buf := make([]byte, readSize)
	n, err := d.file.ReadAt(buf, fileOffset)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

// readDynamicSectors reads sectors from a Dynamic VHD using the BAT.
// Each sector has a bitmap bit; only allocated sectors are read from disk.
// Sparse/unallocated sectors return zero-filled blocks.
func (d *VHDDriver) readDynamicSectors(startSector uint64, count uint32) ([]byte, error) {
	blockSizeSectors := uint64(d.dynamicHeader.BlockSize) / uint64(d.sectorSize)
	result := make([]byte, uint64(count)*uint64(d.sectorSize))

	for i := uint32(0); i < count; i++ {
		sectorOffset := startSector + uint64(i)
		blockIndex := uint32(sectorOffset / blockSizeSectors)
		sectorInBlock := uint32(sectorOffset % blockSizeSectors)

		if int(blockIndex) >= len(d.bat) {
			continue
		}

		batEntry := d.bat[blockIndex]
		if batEntry == 0xFFFFFFFF {
			continue
		}

		blockStart := uint64(batEntry) * uint64(d.sectorSize)
		bitmapSectors := (blockSizeSectors + 7) / 8 / uint64(d.sectorSize)
		if bitmapSectors == 0 {
			bitmapSectors = 1
		}
		dataStart := blockStart + bitmapSectors*uint64(d.sectorSize)

		bitmapByte, err := d.readByte(blockStart + uint64(sectorInBlock/8))
		if err != nil {
			continue
		}
		if bitmapByte&(1<<(sectorInBlock%8)) == 0 {
			continue
		}

		sectorFileOffset := dataStart + uint64(sectorInBlock)*uint64(d.sectorSize)

		// Guard: ensure the sector read stays within the physical VHD file bounds
		if sectorFileOffset+uint64(d.sectorSize) > uint64(d.fileSize) {
			continue
		}

		sectorData := make([]byte, d.sectorSize)
		_, err = d.file.ReadAt(sectorData, int64(sectorFileOffset))
		if err != nil {
			continue
		}
		copy(result[i*uint32(d.sectorSize):], sectorData)
	}

	return result, nil
}

// readByte reads a single byte at the given file offset.
func (d *VHDDriver) readByte(offset uint64) (byte, error) {
	if offset >= uint64(d.fileSize) {
		return 0, ErrOutOfBounds
	}
	buf := make([]byte, 1)
	_, err := d.file.ReadAt(buf, int64(offset))
	if err != nil {
		return 0, err
	}
	return buf[0], nil
}

// Helper functions for reading big-endian integers
func readBE32(data []byte, offset int) uint32 {
	return uint32(data[offset])<<24 | uint32(data[offset+1])<<16 | uint32(data[offset+2])<<8 | uint32(data[offset+3])
}

func readBE64(data []byte, offset int) uint64 {
	return uint64(data[offset])<<56 | uint64(data[offset+1])<<48 | uint64(data[offset+2])<<40 | uint64(data[offset+3])<<32 |
		uint64(data[offset+4])<<24 | uint64(data[offset+5])<<16 | uint64(data[offset+6])<<8 | uint64(data[offset+7])
}

// validateFooterChecksum validates the VHD footer checksum.
// Per VHD spec: zero out bytes 64-67, sum all 512 bytes as unsigned 8-bit integers,
// take bitwise NOT of the sum, compare with stored checksum.
func validateFooterChecksum(data []byte) bool {
	if len(data) < 512 {
		return false
	}
	var sum uint32
	for i := 0; i < 512; i++ {
		if i >= 64 && i < 68 {
			continue // Zero out the checksum field during calculation
		}
		sum += uint32(data[i])
	}
	storedChecksum := binary.BigEndian.Uint32(data[64:68])
	calculatedChecksum := ^sum // One's complement (bitwise NOT)
	return storedChecksum == calculatedChecksum
}

// validateDynamicChecksum validates the dynamic header checksum.
func validateDynamicChecksum(data []byte) bool {
	var sum uint32
	for i := 0; i < len(data); i += 4 {
		if i == 36 {
			continue
		}
		sum += readBE32(data, i)
	}
	checksum := readBE32(data, 36)
	return checksum == ^sum
}

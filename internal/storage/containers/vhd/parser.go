// Package vhd implements the Microsoft Virtual Hard Disk format parser.
// It reads VHD files according to the Microsoft VHD specification,
// supporting both Fixed and Dynamic disk types.
//
// Reference: https://download.microsoft.com/download/f/4/8/f48a7e8f-28d1-4d24-b3e0-a06993c4f2a0/virtual_hard_disk_format_spec_v1_04.doc
package vhd

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

var (
	ErrInvalidSignature    = errors.New("vhd: invalid footer signature")
	ErrInvalidChecksum     = errors.New("vhd: invalid footer checksum")
	ErrInvalidDynamicSig   = errors.New("vhd: invalid dynamic header signature")
	ErrInvalidDynamicChecksum = errors.New("vhd: invalid dynamic header checksum")
	ErrUnsupportedDiskType = errors.New("vhd: unsupported disk type")
	ErrFileTooSmall        = errors.New("vhd: file too small to contain a VHD footer")
	ErrOutOfBounds         = errors.New("vhd: read offset out of bounds")
	ErrSparseBlock         = errors.New("vhd: block is sparse (not allocated)")
)

// VHDFile implements VHDReader for reading VHD files.
type VHDFile struct {
	file           *os.File
	fileSize       int64
	footer         *Footer
	dynamicHeader  *DynamicHeader
	bat            []uint32
	sectorReader   SectorReader
	mu             sync.RWMutex
}

// Open opens a VHD file and parses its footer and optional dynamic header.
func Open(filePath string) (*VHDFile, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("vhd: failed to open file: %w", err)
	}

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("vhd: failed to stat file: %w", err)
	}

	if fi.Size() < FooterSize {
		f.Close()
		return nil, ErrFileTooSmall
	}

	vhd := &VHDFile{
		file:     f,
		fileSize: fi.Size(),
	}

	if err := vhd.parseFooter(); err != nil {
		f.Close()
		return nil, err
	}

	if vhd.footer.DiskType == DiskTypeDynamic || vhd.footer.DiskType == DiskTypeDifferencing {
		if err := vhd.parseDynamicHeader(); err != nil {
			f.Close()
			return nil, err
		}
		if err := vhd.parseBAT(); err != nil {
			f.Close()
			return nil, err
		}
	}

	vhd.sectorReader = newSectorReader(vhd)
	return vhd, nil
}

// parseFooter reads and validates the 512-byte VHD footer.
func (v *VHDFile) parseFooter() error {
	footerBytes := make([]byte, FooterSize)
	n, err := v.file.ReadAt(footerBytes, v.fileSize-FooterSize)
	if err != nil || n != FooterSize {
		return fmt.Errorf("vhd: failed to read footer: %w", err)
	}

	footer := &Footer{}
	reader := bytes.NewReader(footerBytes)

	if err := binary.Read(reader, binary.BigEndian, footer); err != nil {
		return fmt.Errorf("vhd: failed to parse footer: %w", err)
	}

	// Validate signature
	if string(footer.Signature[:]) != SignatureFooter {
		return ErrInvalidSignature
	}

	// Validate checksum
	if !validateChecksum(footerBytes) {
		return ErrInvalidChecksum
	}

	// Validate disk type
	switch footer.DiskType {
	case DiskTypeFixed, DiskTypeDynamic, DiskTypeDifferencing:
		// Supported
	default:
		return fmt.Errorf("%w: %d", ErrUnsupportedDiskType, footer.DiskType)
	}

	v.footer = footer
	return nil
}

// parseDynamicHeader reads and validates the dynamic disk header.
func (v *VHDFile) parseDynamicHeader() error {
	// Dynamic header offset is stored at byte 8 of the footer (DataOffset)
	// For dynamic disks, DataOffset points to the dynamic header
	offset := int64(v.footer.DataOffset)

	headerBytes := make([]byte, DynamicHeaderSize)
	n, err := v.file.ReadAt(headerBytes, offset)
	if err != nil || n != DynamicHeaderSize {
		return fmt.Errorf("vhd: failed to read dynamic header: %w", err)
	}

	header := &DynamicHeader{}
	reader := bytes.NewReader(headerBytes)

	if err := binary.Read(reader, binary.BigEndian, header); err != nil {
		return fmt.Errorf("vhd: failed to parse dynamic header: %w", err)
	}

	// Validate signature
	if string(header.Signature[:]) != SignatureDynamic {
		return ErrInvalidDynamicSig
	}

	// Validate checksum
	if !validateChecksum(headerBytes) {
		return ErrInvalidDynamicChecksum
	}

	v.dynamicHeader = header
	return nil
}

// parseBAT reads the Block Allocation Table.
func (v *VHDFile) parseBAT() error {
	if v.dynamicHeader == nil {
		return nil
	}

	tableOffset := int64(v.dynamicHeader.TableOffset)
	numEntries := int(v.dynamicHeader.MaxTableEntries)
	batSize := numEntries * BATEntrySize

	batBytes := make([]byte, batSize)
	n, err := v.file.ReadAt(batBytes, tableOffset)
	if err != nil || n != batSize {
		return fmt.Errorf("vhd: failed to read BAT: %w", err)
	}

	v.bat = make([]uint32, numEntries)
	for i := 0; i < numEntries; i++ {
		offset := i * BATEntrySize
		v.bat[i] = binary.BigEndian.Uint32(batBytes[offset : offset+BATEntrySize])
	}

	return nil
}

// GetFooter returns the parsed VHD footer.
func (v *VHDFile) GetFooter() *Footer {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.footer
}

// GetDynamicHeader returns the parsed dynamic disk header.
func (v *VHDFile) GetDynamicHeader() *DynamicHeader {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.dynamicHeader
}

// GetBAT returns the Block Allocation Table.
func (v *VHDFile) GetBAT() []uint32 {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.bat
}

// GetDiskType returns the VHD disk type.
func (v *VHDFile) GetDiskType() DiskType {
	if v.footer == nil {
		return DiskTypeNone
	}
	return v.footer.DiskType
}

// GetSectorReader returns the sector reader for this VHD.
func (v *VHDFile) GetSectorReader() SectorReader {
	return v.sectorReader
}

// GetDiskInfo computes and returns disk information.
func (v *VHDFile) GetDiskInfo() *DiskInfo {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.footer == nil {
		return nil
	}

	info := &DiskInfo{
		VirtualSize:  v.footer.CurrentSize,
		Geometry:     v.footer.DiskGeometry,
		DiskType:     v.footer.DiskType.String(),
		BlockSize:    0,
		MaxBATEntries: 0,
		CreatedAt:    ParseTimestamp(v.footer.Timestamp),
		UniqueID:     fmt.Sprintf("%x-%x-%x-%x-%x",
			v.footer.UniqueID[0:4], v.footer.UniqueID[4:6],
			v.footer.UniqueID[6:8], v.footer.UniqueID[8:10],
			v.footer.UniqueID[10:16]),
	}

	// Creator app
	app := strings.TrimRight(string(v.footer.CreatorApp[:]), "\x00")
	if app == "" {
		app = "Unknown"
	}
	info.CreatorApp = app

	// Creator version (major.minor)
	major := (v.footer.CreatorVer >> 16) & 0xFFFF
	minor := v.footer.CreatorVer & 0xFFFF
	info.CreatorVersion = fmt.Sprintf("%d.%d", major, minor)

	// Host OS
	switch v.footer.CreatorHostOS {
	case 0x5769326B: // "Wi2k"
		info.CreatorHostOS = "Windows"
	case 0x4D616320: // "Mac "
		info.CreatorHostOS = "Macintosh"
	default:
		info.CreatorHostOS = "Unknown"
	}

	if v.dynamicHeader != nil {
		info.BlockSize = v.dynamicHeader.BlockSize
		info.MaxBATEntries = v.dynamicHeader.MaxTableEntries
	}

	return info
}

// Close closes the underlying file.
func (v *VHDFile) Close() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.file != nil {
		return v.file.Close()
	}
	return nil
}

// ValidateChecksum checks if the one's complement checksum is correct.
func validateChecksum(data []byte) bool {
	// The checksum is the one's complement of the sum of all uint32 values
	// in the structure, excluding the checksum field itself (bytes 64-67).
	var sum uint32
	for i := 0; i < len(data); i += 4 {
		// Skip the checksum field (bytes 64-67)
		if i == 64 {
			continue
		}
		sum += binary.BigEndian.Uint32(data[i : i+4])
	}
	checksum := binary.BigEndian.Uint32(data[64:68])
	return checksum == ^sum
}

// sectorReader implements SectorReader for both Fixed and Dynamic VHDs.
type sectorReader struct {
	vhd *VHDFile
}

func newSectorReader(vhd *VHDFile) SectorReader {
	return &sectorReader{vhd: vhd}
}

func (sr *sectorReader) Close() error {
	return nil
}

func (sr *sectorReader) SectorSize() uint64 {
	return SectorSizeBytes
}

func (sr *sectorReader) TotalSectors() uint64 {
	if sr.vhd.footer == nil {
		return 0
	}
	return sr.vhd.footer.CurrentSize / SectorSizeBytes
}

func (sr *sectorReader) ReadSectors(offset uint64, count uint32) ([]byte, error) {
	if count == 0 {
		return nil, nil
	}

	totalBytes := uint64(count) * SectorSizeBytes
	if totalBytes > MaxSectorReadSize {
		return nil, fmt.Errorf("vhd: read size %d exceeds maximum %d", totalBytes, MaxSectorReadSize)
	}

	sr.vhd.mu.RLock()
	defer sr.vhd.mu.RUnlock()

	switch sr.vhd.footer.DiskType {
	case DiskTypeFixed:
		return sr.readFixedSectors(offset, count)
	case DiskTypeDynamic, DiskTypeDifferencing:
		return sr.readDynamicSectors(offset, count)
	default:
		return nil, ErrUnsupportedDiskType
	}
}

// readFixedSectors reads sectors directly from the file (Fixed VHD).
func (sr *sectorReader) readFixedSectors(offset uint64, count uint32) ([]byte, error) {
	fileOffset := int64(offset * SectorSizeBytes)
	readSize := int64(count) * int64(SectorSizeBytes)

	// Bounds check
	if fileOffset+readSize > sr.vhd.fileSize {
		return nil, ErrOutOfBounds
	}

	buf := make([]byte, readSize)
	n, err := sr.vhd.file.ReadAt(buf, fileOffset)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("vhd: failed to read fixed sectors: %w", err)
	}
	return buf[:n], nil
}

// readDynamicSectors reads sectors from a Dynamic VHD using the BAT.
func (sr *sectorReader) readDynamicSectors(offset uint64, count uint32) ([]byte, error) {
	blockSizeSectors := uint64(sr.vhd.dynamicHeader.BlockSize / SectorSizeBytes)
	result := make([]byte, uint64(count)*SectorSizeBytes)

	for i := uint32(0); i < count; i++ {
		sectorOffset := offset + uint64(i)
		blockIndex := uint32(sectorOffset / blockSizeSectors)
		sectorInBlock := uint32(sectorOffset % blockSizeSectors)

		if int(blockIndex) >= len(sr.vhd.bat) {
			// Out of bounds, fill with zeros
			continue
		}

		batEntry := sr.vhd.bat[blockIndex]
		// 0xFFFFFFFF means sparse block
		if batEntry == 0xFFFFFFFF {
			continue
		}

		// Calculate data offset
		// BAT entry * 512 gives sector offset of the block data
		blockStart := uint64(batEntry) * SectorSizeBytes

		// Sector bitmap immediately follows the block start
		// Bitmap size = ceil(blockSizeSectors / 8) bytes, padded to sector boundary
		bitmapSectors := (blockSizeSectors + 7) / 8 / SectorSizeBytes
		if bitmapSectors == 0 {
			bitmapSectors = 1
		}
		dataStart := blockStart + uint64(bitmapSectors)*SectorSizeBytes

		// Check the bitmap to see if this sector has been written
		bitmapOffset := blockStart + uint64(sectorInBlock/8)
		bitmapByte, err := sr.readByte(bitmapOffset)
		if err != nil {
			// If we can't read bitmap, treat as sparse
			continue
		}
		bitMask := byte(1 << (sectorInBlock % 8))
		if bitmapByte&bitMask == 0 {
			// Sector not allocated, return zeros
			continue
		}

		// Read the actual sector data
		sectorFileOffset := dataStart + uint64(sectorInBlock)*SectorSizeBytes
		sectorData, err := sr.readRawSector(sectorFileOffset)
		if err != nil {
			continue
		}

		copy(result[i*SectorSizeBytes:], sectorData)
	}

	return result, nil
}

// readByte reads a single byte at the given file offset.
func (sr *sectorReader) readByte(offset uint64) (byte, error) {
	buf := make([]byte, 1)
	_, err := sr.vhd.file.ReadAt(buf, int64(offset))
	if err != nil {
		return 0, err
	}
	return buf[0], nil
}

// readRawSector reads 512 bytes at the given file offset.
func (sr *sectorReader) readRawSector(offset uint64) ([]byte, error) {
	buf := make([]byte, SectorSizeBytes)
	_, err := sr.vhd.file.ReadAt(buf, int64(offset))
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// IsSparseBlock checks if a block is sparse in the BAT.
func (v *VHDFile) IsSparseBlock(blockIndex uint32) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if int(blockIndex) >= len(v.bat) {
		return true
	}
	return v.bat[blockIndex] == 0xFFFFFFFF
}

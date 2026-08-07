// Package partition implements MBR and GPT partition table parsing.
package partition

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
)

var (
	ErrInvalidMBRSignature = errors.New("partition: invalid MBR signature")
	ErrInvalidGPTSignature = errors.New("partition: invalid GPT signature")
	ErrInvalidGPTCRC       = errors.New("partition: invalid GPT header CRC32")
	ErrNoPartitionsFound   = errors.New("partition: no partitions found")
)

// MBRParser implements Reader for Master Boot Record partition tables.
type MBRParser struct{}

// NewMBRParser creates a new MBR parser.
func NewMBRParser() *MBRParser {
	return &MBRParser{}
}

// ReadPartitions reads and parses MBR partition entries.
func (p *MBRParser) ReadPartitions(diskReader DiskReader) ([]PartitionInfo, error) {
	mbr, err := p.ReadMBR(diskReader)
	if err != nil {
		return nil, err
	}

	var partitions []PartitionInfo
	sectorSize := diskReader.SectorSize()

	for i, part := range mbr.Partitions {
		if part.IsEmpty() {
			continue
		}

		// Skip GPT protective partition
		if part.PartitionType == MBRTypeGPTProtective {
			return nil, nil // Caller should try GPT parser
		}

		info := PartitionInfo{
			Index:      i,
			Number:     i + 1,
			StartLBA:   uint64(part.LBAFirst),
			EndLBA:     uint64(part.LBAFirst) + uint64(part.Sectors) - 1,
			SizeBytes:  uint64(part.Sectors) * sectorSize,
			TypeID:     part.PartitionType,
			Type:       PartitionTypeName(part.PartitionType),
			IsActive:  part.Status == 0x80,
		}
		info.SizeFormatted = FormatSize(info.SizeBytes)

		partitions = append(partitions, info)
	}

	if len(partitions) == 0 {
		return nil, ErrNoPartitionsFound
	}

	return partitions, nil
}

// ReadMBR reads and parses the Master Boot Record.
func (p *MBRParser) ReadMBR(diskReader DiskReader) (*MBR, error) {
	// Read sector 0 (LBA 0)
	data, err := diskReader.ReadSectors(0, 1)
	if err != nil {
		return nil, fmt.Errorf("partition: failed to read MBR: %w", err)
	}

	if len(data) < MBRSize {
		return nil, fmt.Errorf("partition: MBR data too short: %d bytes", len(data))
	}

	mbr := &MBR{}
	copy(mbr.BootCode[:], data[:446])

	// Parse 4 partition entries (each 16 bytes)
	for i := 0; i < 4; i++ {
		offset := MBRPartitionTableOff + (i * 16)
		entry := data[offset : offset+16]
		mbr.Partitions[i].Status = entry[0]
		copy(mbr.Partitions[i].CHSFirst[:], entry[1:4])
		mbr.Partitions[i].PartitionType = entry[4]
		copy(mbr.Partitions[i].CHSLast[:], entry[5:8])
		mbr.Partitions[i].LBAFirst = binary.LittleEndian.Uint32(entry[8:12])
		mbr.Partitions[i].Sectors = binary.LittleEndian.Uint32(entry[12:16])
	}

	mbr.Signature = binary.LittleEndian.Uint16(data[510:512])
	if mbr.Signature != MBRSignature {
		return nil, ErrInvalidMBRSignature
	}

	return mbr, nil
}

// ReadGPT reads and parses the GPT header and partition entries.
func (p *MBRParser) ReadGPT(diskReader DiskReader) (*GPTHeader, []GPTPartitionEntry, error) {
	// Read LBA 1 (GPT Header)
	data, err := diskReader.ReadSectors(GPTHeaderLBA, 1)
	if err != nil {
		return nil, nil, fmt.Errorf("partition: failed to read GPT header: %w", err)
	}

	if len(data) < 92 {
		return nil, nil, fmt.Errorf("partition: GPT header too short")
	}

	header := &GPTHeader{}
	copy(header.Signature[:], data[0:8])
	header.Revision = binary.LittleEndian.Uint32(data[8:12])
	header.HeaderSize = binary.LittleEndian.Uint32(data[12:16])
	header.HeaderCRC32 = binary.LittleEndian.Uint32(data[16:20])
	header.Reserved = binary.LittleEndian.Uint32(data[20:24])
	header.MyLBA = binary.LittleEndian.Uint64(data[24:32])
	header.AlternateLBA = binary.LittleEndian.Uint64(data[32:40])
	header.FirstUsableLBA = binary.LittleEndian.Uint64(data[40:48])
	header.LastUsableLBA = binary.LittleEndian.Uint64(data[48:56])
	copy(header.DiskGUID[:], data[56:72])
	header.PartitionEntryLBA = binary.LittleEndian.Uint64(data[72:80])
	header.NumPartEntries = binary.LittleEndian.Uint32(data[80:84])
	header.PartEntrySize = binary.LittleEndian.Uint32(data[84:88])
	header.PartitionEntryCRC32 = binary.LittleEndian.Uint32(data[88:92])

	// Validate signature
	if string(header.Signature[:]) != GPTSignature {
		return nil, nil, ErrInvalidGPTSignature
	}

	// Validate header CRC32
	savedCRC := header.HeaderCRC32
	header.HeaderCRC32 = 0
	headerBytes := make([]byte, header.HeaderSize)
	copy(headerBytes, data[:min(int(header.HeaderSize), len(data))])
	calculatedCRC := crc32.ChecksumIEEE(headerBytes)
	if calculatedCRC != savedCRC {
		return nil, nil, ErrInvalidGPTCRC
	}
	header.HeaderCRC32 = savedCRC

	// Read partition entries
	entrySize := header.PartEntrySize
	numEntries := header.NumPartEntries
	totalEntriesSize := uint64(numEntries) * uint64(entrySize)
	entriesSectors := (totalEntriesSize + uint64(diskReader.SectorSize()) - 1) / uint64(diskReader.SectorSize())

	entriesData, err := diskReader.ReadSectors(header.PartitionEntryLBA, uint32(entriesSectors))
	if err != nil {
		return nil, nil, fmt.Errorf("partition: failed to read GPT entries: %w", err)
	}

	var entries []GPTPartitionEntry
	for i := uint32(0); i < numEntries; i++ {
		entryOffset := uint64(i) * uint64(entrySize)
		if entryOffset+uint64(entrySize) > uint64(len(entriesData)) {
			break
		}

		entryBytes := entriesData[entryOffset : entryOffset+uint64(entrySize)]
		entry := GPTPartitionEntry{}
		copy(entry.TypeGUID[:], entryBytes[0:16])
		copy(entry.UniqueGUID[:], entryBytes[16:32])
		entry.StartingLBA = binary.LittleEndian.Uint64(entryBytes[32:40])
		entry.EndingLBA = binary.LittleEndian.Uint64(entryBytes[40:48])
		entry.Attributes = binary.LittleEndian.Uint64(entryBytes[48:56])
		copy(entry.PartitionName[:], entryBytes[56:128])

		if !entry.IsEmpty() {
			entries = append(entries, entry)
		}
	}

	return header, entries, nil
}

// GPTParser implements Reader for GUID Partition Table.
type GPTParser struct {
	mbrParser *MBRParser
}

// NewGPTParser creates a new GPT parser.
func NewGPTParser() *GPTParser {
	return &GPTParser{mbrParser: NewMBRParser()}
}

// ReadPartitions reads and parses GPT partition entries.
func (p *GPTParser) ReadPartitions(diskReader DiskReader) ([]PartitionInfo, error) {
	_, gptEntries, err := p.ReadGPT(diskReader)
	if err != nil {
		return nil, err
	}

	var partitions []PartitionInfo
	sectorSize := diskReader.SectorSize()

	for i, entry := range gptEntries {
		if entry.IsEmpty() {
			continue
		}

		info := PartitionInfo{
			Index:      i,
			Number:     i + 1,
			StartLBA:   entry.StartingLBA,
			EndLBA:     entry.EndingLBA,
			SizeBytes:  (entry.EndingLBA - entry.StartingLBA + 1) * sectorSize,
			TypeGUID:   fmt.Sprintf("%x", entry.TypeGUID),
			Name:       cleanGPTName(entry.PartitionName[:]),
			IsActive:  true,
		}
		info.SizeFormatted = FormatSize(info.SizeBytes)
		info.Type = gptPartitionTypeName(entry.TypeGUID[:])

		partitions = append(partitions, info)
	}

	if len(partitions) == 0 {
		return nil, ErrNoPartitionsFound
	}

	return partitions, nil
}

// ReadMBR reads the MBR (for GPT protective MBR detection).
func (p *GPTParser) ReadMBR(diskReader DiskReader) (*MBR, error) {
	return p.mbrParser.ReadMBR(diskReader)
}

// ReadGPT reads the GPT header and entries.
func (p *GPTParser) ReadGPT(diskReader DiskReader) (*GPTHeader, []GPTPartitionEntry, error) {
	return p.mbrParser.ReadGPT(diskReader)
}

// AutoDetectAndRead tries to detect the partition table type and read partitions.
func AutoDetectAndRead(diskReader DiskReader) ([]PartitionInfo, error) {
	mbrParser := NewMBRParser()
	mbr, err := mbrParser.ReadMBR(diskReader)
	if err != nil {
		return nil, err
	}

	// Check for GPT protective MBR
	for _, part := range mbr.Partitions {
		if part.PartitionType == MBRTypeGPTProtective {
			gptParser := NewGPTParser()
			return gptParser.ReadPartitions(diskReader)
		}
	}

	// Default to MBR
	return mbrParser.ReadPartitions(diskReader)
}

// cleanGPTName removes null bytes and converts GPT partition name to string.
func cleanGPTName(nameBytes []byte) string {
	// GPT names are UTF-16LE
	var runes []rune
	for i := 0; i < len(nameBytes)-1; i += 2 {
		r := rune(nameBytes[i]) | rune(nameBytes[i+1])<<8
		if r == 0 {
			break
		}
		runes = append(runes, r)
	}
	return string(runes)
}

// gptPartitionTypeName maps GPT type GUIDs to names.
func gptPartitionTypeName(guid []byte) string {
	// Microsoft basic data partition
	microsoftBasicData := []byte{0xA2, 0xA0, 0xD0, 0xEB, 0xE5, 0xB9, 0x33, 0x44, 0x87, 0xC0, 0x68, 0xB6, 0xB7, 0x26, 0x99, 0xC7}
	if bytes.Equal(guid, microsoftBasicData) {
		return "Microsoft Basic Data"
	}

	// EFI System Partition
	efiSystem := []byte{0x28, 0x73, 0x2A, 0xC1, 0x1F, 0xF8, 0xD2, 0x11, 0xBA, 0x4B, 0x00, 0xA0, 0xC9, 0x3E, 0xC9, 0x3B}
	if bytes.Equal(guid, efiSystem) {
		return "EFI System"
	}

	// Linux filesystem
	linuxFS := []byte{0xAF, 0x3D, 0xC6, 0x0F, 0x83, 0x84, 0x72, 0x47, 0x8E, 0x79, 0x3D, 0x69, 0xD8, 0x47, 0x7D, 0xE4}
	if bytes.Equal(guid, linuxFS) {
		return "Linux Filesystem"
	}

	// Linux swap
	linuxSwap := []byte{0x06, 0x57, 0xFD, 0x6D, 0xA4, 0xAB, 0x43, 0xC4, 0x84, 0xE5, 0x09, 0x33, 0xC8, 0x4B, 0x4F, 0x4F}
	if bytes.Equal(guid, linuxSwap) {
		return "Linux Swap"
	}

	return "Unknown"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

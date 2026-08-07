// Package partition defines the domain types for disk partition tables.
// It supports both MBR and GPT partition table formats.
package partition

import "fmt"

const (
	MBRSize              = 512
	MBRPartitionTableOff = 446
	MBRSignature         = 0xAA55
	GPTSignature         = "EFI PART"
	GPTHeaderLBA         = 1

	// Common MBR partition types
	MBRTypeEmpty         = 0x00
	MBRTypeFAT12         = 0x01
	MBRTypeFAT16Small    = 0x04
	MBRTypeExtended      = 0x05
	MBRTypeFAT16B        = 0x06
	MBRTypeNTFS          = 0x07
	MBRTypeFAT32CHS      = 0x0B
	MBRTypeFAT32LBA      = 0x0C
	MBRTypeFAT16BLBA     = 0x0E
	MBRTypeLinuxSwap     = 0x82
	MBRTypeLinux         = 0x83
	MBRTypeGPTProtective = 0xEE
)

// MBR represents the Master Boot Record structure.
type MBR struct {
	BootCode   [446]byte
	Partitions [4]MBRPartition
	Signature  uint16
}

// MBRPartition represents a single MBR partition entry.
type MBRPartition struct {
	Status         uint8
	CHSFirst       [3]byte
	PartitionType  uint8
	CHSLast        [3]byte
	LBAFirst       uint32
	Sectors        uint32
}

// IsEmpty returns true if this partition entry is unused.
func (p *MBRPartition) IsEmpty() bool {
	return p.PartitionType == MBRTypeEmpty
}

// IsExtended returns true if this is an extended partition.
func (p *MBRPartition) IsExtended() bool {
	return p.PartitionType == MBRTypeExtended || p.PartitionType == MBRTypeGPTProtective
}

// GPTHeader represents the GUID Partition Table header.
type GPTHeader struct {
	Signature          [8]byte
	Revision           uint32
	HeaderSize         uint32
	HeaderCRC32        uint32
	Reserved           uint32
	MyLBA              uint64
	AlternateLBA       uint64
	FirstUsableLBA     uint64
	LastUsableLBA      uint64
	DiskGUID           [16]byte
	PartitionEntryLBA  uint64
	NumPartEntries     uint32
	PartEntrySize      uint32
	PartitionEntryCRC32 uint32
}

// GPTPartitionEntry represents a single GPT partition entry.
type GPTPartitionEntry struct {
	TypeGUID       [16]byte
	UniqueGUID     [16]byte
	StartingLBA    uint64
	EndingLBA      uint64
	Attributes     uint64
	PartitionName  [72]byte
}

// IsEmpty returns true if this partition entry is unused.
func (e *GPTPartitionEntry) IsEmpty() bool {
	allZero := true
	for _, b := range e.TypeGUID {
		if b != 0 {
			allZero = false
			break
		}
	}
	return allZero
}

// PartitionInfo is a unified representation of a partition for display.
type PartitionInfo struct {
	Index           int    `json:"index"`
	Number          int    `json:"number"`
	StartLBA        uint64 `json:"startLBA"`
	EndLBA          uint64 `json:"endLBA"`
	SizeBytes       uint64 `json:"sizeBytes"`
	SizeFormatted   string `json:"sizeFormatted"`
	Type            string `json:"type"`
	TypeID          byte   `json:"typeId"`
	TypeGUID        string `json:"typeGuid"`
	Name            string `json:"name"`
	IsActive        bool   `json:"isActive"`
	FilesystemType  string `json:"filesystemType"`
}

// FormatSize formats a byte count into a human-readable string.
func FormatSize(bytes uint64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)
	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// PartitionTypeName returns a human-readable name for a partition type.
func PartitionTypeName(typeID byte) string {
	switch typeID {
	case MBRTypeEmpty:
		return "Empty"
	case MBRTypeFAT12:
		return "FAT12"
	case MBRTypeFAT16Small:
		return "FAT16 (<32MB)"
	case MBRTypeExtended:
		return "Extended"
	case MBRTypeFAT16B:
		return "FAT16B"
	case MBRTypeNTFS:
		return "NTFS"
	case MBRTypeFAT32CHS:
		return "FAT32 (CHS)"
	case MBRTypeFAT32LBA:
		return "FAT32 (LBA)"
	case MBRTypeFAT16BLBA:
		return "FAT16B (LBA)"
	case MBRTypeLinuxSwap:
		return "Linux Swap"
	case MBRTypeLinux:
		return "Linux"
	case MBRTypeGPTProtective:
		return "GPT Protective"
	default:
		return fmt.Sprintf("Unknown (0x%02X)", typeID)
	}
}

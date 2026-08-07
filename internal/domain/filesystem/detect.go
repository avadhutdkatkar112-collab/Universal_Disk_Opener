package filesystem

import (
	"encoding/binary"
	"fmt"
)

// DetectFSType detects the filesystem type based on boot sector signature.
func DetectFSType(diskReader DiskReader, partitionStart uint64) FSType {
	data, err := diskReader.ReadSectors(partitionStart, 1)
	if err != nil {
		return Unknown
	}

	// Check for NTFS
	if len(data) >= 11 && string(data[3:11]) == "NTFS    " {
		return NTFS
	}

	// Check for exFAT signature at offset 3
	if len(data) >= 11 && string(data[3:11]) == "EXFAT   " {
		return exFAT
	}

	// Check for FAT
	// FAT12/FAT16/FAT32 share similar BPB structure
	if len(data) >= 90 {
		// Check FAT32 signature
		if string(data[82:90]) == "FAT32   " {
			return FAT32
		}

		// Check FAT16/FAT12
		sectorsPerFAT := uint16(data[22])
		totalSectors16 := binary.LittleEndian.Uint16(data[19:21])
		totalSectors32 := binary.LittleEndian.Uint32(data[32:36])
		rootEntryCount := binary.LittleEndian.Uint16(data[17:19])

		if sectorsPerFAT == 0 && totalSectors32 > 0 {
			return FAT32
		}

		if rootEntryCount > 0 {
			totalSectors := uint64(totalSectors16)
			if totalSectors == 0 {
				totalSectors = uint64(totalSectors32)
			}

			sectorsPerCluster := uint64(data[13])
			if sectorsPerCluster == 0 {
				// Fall through to ext check
			} else {
				reservedSectors := uint64(binary.LittleEndian.Uint16(data[14:16]))
				numFATs := uint64(data[16])
				dataSectors := totalSectors - reservedSectors - numFATs*uint64(sectorsPerFAT) - (uint64(rootEntryCount)*32+511)/512
				totalClusters := dataSectors / sectorsPerCluster

				if totalClusters < 4085 {
					return FAT12
				}
				return FAT16
			}
		}
	}

	// Check for ext2/3/4: superblock at byte 1024 (sector 2), magic at offset 0x38
	extData, err := diskReader.ReadSectors(partitionStart+2, 1)
	if err == nil && len(extData) >= 0x40 {
		magic := binary.LittleEndian.Uint16(extData[0x38:0x3A])
		if magic == 0xEF53 {
			return EXT4
		}
	}

	return Unknown
}

// NewReader creates the appropriate filesystem reader for a partition.
func NewReader(diskReader DiskReader, partitionStart uint64) (Reader, error) {
	fsType := DetectFSType(diskReader, partitionStart)

	switch fsType {
	case FAT32:
		return NewFAT32Reader(diskReader, partitionStart)
	case FAT16:
		return NewFAT16Reader(diskReader, partitionStart)
	case exFAT:
		return newExFATReader(diskReader, partitionStart)
	case NTFS:
		return NewNTFSReader(diskReader, partitionStart)
	case EXT4:
		return NewEXT4Reader(diskReader, partitionStart)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFS, fsType)
	}
}

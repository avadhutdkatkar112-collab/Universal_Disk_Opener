package filesystem

import (
	"fmt"
)

// DetectFSType detects the filesystem type based on boot sector signature.
func DetectFSType(diskReader DiskReader, partitionStart uint64) FSType {
	data, err := diskReader.ReadSectors(partitionStart, 1)
	if err != nil {
		return Unknown
	}

	// Check for NTFS
	if len(data) >= 3 && string(data[3:11]) == "NTFS    " {
		return NTFS
	}

	// Check for FAT
	// FAT12/FAT16/FAT32 share similar BPB structure
	if len(data) >= 54 {
		// Check FAT32 signature
		if string(data[82:90]) == "FAT32   " {
			return FAT32
		}

		// Check FAT16/FAT12
		// FAT32: sectors per FAT at offset 36 is 0
		sectorsPerFAT := uint16(data[22])
		totalSectors16 := binary.LittleEndian.Uint16(data[19:21])
		totalSectors32 := binary.LittleEndian.Uint32(data[32:36])
		rootEntryCount := binary.LittleEndian.Uint16(data[17:19])

		if sectorsPerFAT == 0 && totalSectors32 > 0 {
			// Likely FAT32
			return FAT32
		}

		if rootEntryCount > 0 {
			// FAT16 or FAT12
			totalSectors := uint64(totalSectors16)
			if totalSectors == 0 {
				totalSectors = uint64(totalSectors32)
			}

			sectorsPerCluster := data[13]
			if sectorsPerCluster == 0 {
				return Unknown
			}

			dataSectors := totalSectors - uint64(data[14:16]) - uint64(data[16])*uint64(sectorsPerFAT) - (uint64(rootEntryCount)*32+511)/512
			totalClusters := dataSectors / uint64(sectorsPerCluster)

			if totalClusters < 4085 {
				return FAT12
			}
			return FAT16
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
	case NTFS:
		return NewNTFSReader(diskReader, partitionStart)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFS, fsType)
	}
}

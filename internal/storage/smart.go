// package storage provides the Smart Open service that coordinates the entire
// open pipeline: validation, format detection, driver loading, partition
// discovery, filesystem detection, and virtual file tree construction.
package storage

import (
	"fmt"
	"io"
	"sort"
)

// SmartOpen is the high-level orchestrator for opening a virtual disk.
// It runs the complete pipeline from file path to virtual file tree.
type SmartOpen struct{}

// NewSmartOpen creates a new SmartOpen instance.
func NewSmartOpen() *SmartOpen {
	return &SmartOpen{}
}

// OpenResult holds the result of a SmartOpen operation.
type OpenResult struct {
	Disk            VirtualDisk   `json:"-"`
	Info            DiskInfo      `json:"info"`
	Partitions      []Partition   `json:"partitions"`
	ActivePartition *Partition    `json:"activePartition"`
	RootPath        string        `json:"rootPath"`
	Warnings        []string      `json:"warnings"`
}

// Open performs the complete Smart Open pipeline.
func (s *SmartOpen) Open(path string) (*OpenResult, error) {
	// Stage 1: Validate
	validation, err := Validate(path)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Stage 2: Format detection and driver loading
	vdisk, err := Open(path)
	if err != nil {
		return nil, fmt.Errorf("format detection failed: %w", err)
	}

	// Stage 3: Build result
	result := &OpenResult{
		Disk:     vdisk,
		Info:     vdisk.Info(),
		Warnings: append(validation.Warnings, vdisk.Warnings()...),
	}

	// Stage 4: Detect partitions (read sector 0 for MBR, or scan for GPT)
	partitions, err := s.detectPartitions(vdisk)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Partition detection: %v", err))
	} else {
	// Stage 4b: Detect filesystem on each partition and check for content
	for i := range partitions {
		partitions[i].Filesystem = s.detectFilesystem(vdisk, partitions[i].Start)
		partitions[i].HasContent = s.checkPartitionContent(vdisk, partitions[i])
	}
		result.Partitions = partitions
	}

	// Stage 5: Smart partition selection
	result.ActivePartition = s.selectBestPartition(partitions)
	if result.ActivePartition != nil {
		result.RootPath = fmt.Sprintf("/%d", result.ActivePartition.Index)
	} else if len(partitions) > 0 {
		result.RootPath = fmt.Sprintf("/%d", partitions[0].Index)
	} else {
		result.RootPath = "/"
	}

	return result, nil
}

// detectPartitions reads sector 0 and determines MBR vs GPT.
func (s *SmartOpen) detectPartitions(disk VirtualDisk) ([]Partition, error) {
	sector0, err := disk.ReadSectors(0, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to read sector 0: %w", err)
	}

	// Check for GPT FIRST (more specific than MBR)
	if disk.Size() >= 1024 {
		gptHeader, err := disk.ReadSectors(1, 1)
		if err == nil && len(gptHeader) >= 8 {
			if string(gptHeader[0:8]) == "EFI PART" {
				return s.parseGPT(disk)
			}
		}
	}

	// Check for MBR signature (0x55AA at bytes 510-511)
	if len(sector0) >= 512 && sector0[510] == 0x55 && sector0[511] == 0xAA {
		return s.parseMBR(sector0)
	}

	return nil, nil
}

// detectFilesystem detects the filesystem type at a given sector offset.
// Reads the boot sector and checks for known signatures.
func (s *SmartOpen) detectFilesystem(disk VirtualDisk, startSector uint64) string {
	data, err := disk.ReadSectors(startSector, 1)
	if err != nil || len(data) < 11 {
		return "Unknown"
	}

	// Check for NTFS (signature at offset 3-10: "NTFS    ")
	if string(data[3:11]) == "NTFS    " {
		return "NTFS"
	}

	// Check for FAT32 (signature at offset 82-89: "FAT32   ")
	if len(data) >= 90 && string(data[82:90]) == "FAT32   " {
		return "FAT32"
	}

	// Check for FAT12/FAT16 (has root entry count > 0 and sectors per FAT)
	if len(data) >= 90 {
		rootEntryCount := uint16(data[17]) | uint16(data[18])<<8
		if rootEntryCount > 0 {
			totalSectors16 := uint16(data[19]) | uint16(data[20])<<8
			totalSectors32 := uint32(data[32]) | uint32(data[33])<<8 | uint32(data[34])<<16 | uint32(data[35])<<24
			sectorsPerFAT := uint16(data[22]) | uint16(data[23])<<8
			sectorsPerCluster := uint64(data[13])

			if sectorsPerCluster > 0 {
				reservedSectors := uint64(uint16(data[14]) | uint16(data[15])<<8)
				numFATs := uint64(data[16])
				totalSectors := uint64(totalSectors16)
				if totalSectors == 0 {
					totalSectors = uint64(totalSectors32)
				}
				dataSectors := totalSectors - reservedSectors - numFATs*uint64(sectorsPerFAT) - (uint64(rootEntryCount)*32+511)/512
				totalClusters := dataSectors / sectorsPerCluster

				if totalClusters < 4085 {
					return "FAT12"
				}
				return "FAT16"
			}
		}

		// FAT32 without signature (sectorsPerFAT == 0 and totalSectors32 > 0)
		sectorsPerFAT := uint16(data[22]) | uint16(data[23])<<8
		totalSectors32 := uint32(data[32]) | uint32(data[33])<<8 | uint32(data[34])<<16 | uint32(data[35])<<24
		if sectorsPerFAT == 0 && totalSectors32 > 0 {
			return "FAT32"
		}
	}

	// Check for ext2/3/4: superblock starts at byte 1024 (0x400) from partition start
	// Magic 0xEF53 is at offset 0x438 within the superblock, which is byte 0x838 from start
	// That is at sector 4 (byte 2048+), offset 0x38 within that sector
	extData, err := disk.ReadSectors(startSector+2, 1)
	if err == nil && len(extData) >= 0x40 {
		magic := uint16(extData[0x38]) | uint16(extData[0x39])<<8
		if magic == 0xEF53 {
			return "ext4"
		}
	}

	return "Unknown"
}

// parseMBR parses the MBR partition table from sector 0.
func (s *SmartOpen) parseMBR(data []byte) ([]Partition, error) {
	var partitions []Partition

	for i := 0; i < 4; i++ {
		offset := 446 + i*16
		status := data[offset]
		pType := data[offset+4]
		lbaStart := readLE32(data, offset+8)
		lbaSize := readLE32(data, offset+12)

		if lbaSize == 0 {
			continue
		}

		part := Partition{
			Index:    i,
			Start:    uint64(lbaStart),
			End:      uint64(lbaStart + lbaSize),
			Size:     uint64(lbaSize) * 512,
			Bootable: status == 0x80,
			Type:     formatMBRType(pType),
		}

		partitions = append(partitions, part)
	}

	return partitions, nil
}

// parseGPT parses the GPT partition table.
func (s *SmartOpen) parseGPT(disk VirtualDisk) ([]Partition, error) {
	var partitions []Partition

	gptHeader, err := disk.ReadSectors(1, 1)
	if err != nil {
		return nil, err
	}

	// Validate GPT signature
	if string(gptHeader[0:8]) != "EFI PART" {
		return nil, fmt.Errorf("invalid GPT signature")
	}

	partStart := readLE64(gptHeader, 72)
	partSize := readLE32(gptHeader, 80)
	partCount := readLE32(gptHeader, 84)

	if partCount > 128 {
		partCount = 128
	}

	tableSectors := uint32((uint64(partCount) * uint64(partSize) + 511) / 512)

	partData, err := disk.ReadSectors(partStart, tableSectors)
	if err != nil {
		return nil, fmt.Errorf("failed to read GPT table: %w", err)
	}

	for i := uint32(0); i < partCount; i++ {
		entryOffset := uint64(i) * uint64(partSize)
		if entryOffset+16 > uint64(len(partData)) {
			break
		}

		typeGUID := partData[entryOffset : entryOffset+16]
		startLBA := readLE64(partData, int(entryOffset+32))
		endLBA := readLE64(partData, int(entryOffset+40))
		nameRaw := partData[entryOffset+56 : entryOffset+128]

		isEmpty := true
		for _, b := range typeGUID {
			if b != 0 {
				isEmpty = false
				break
			}
		}
		if isEmpty {
			continue
		}

		part := Partition{
			Index:  int(i),
			Start:  startLBA,
			End:    endLBA,
			Size:   (endLBA - startLBA + 1) * 512,
			Type:   formatGPTType(typeGUID),
			Active: true,
		}

		part.Label = trimUTF16LE(nameRaw)

		partitions = append(partitions, part)
	}

	return partitions, nil
}

// selectBestPartition chooses the most useful partition.
func (s *SmartOpen) selectBestPartition(partitions []Partition) *Partition {
	if len(partitions) == 0 {
		return nil
	}

	// Priority: ext4 > NTFS > FAT32 > FAT16 > others
	// Partitions with content are strongly preferred over empty ones
	scored := make([]scoredPartition, 0, len(partitions))
	for i := range partitions {
		p := &partitions[i]
		score := 0

		switch p.Filesystem {
		case "ext4", "ext3", "ext2":
			score = 100
		case "NTFS":
			score = 90
		case "FAT32":
			score = 80
		case "FAT16":
			score = 70
		default:
			score = 0
		}

		if p.HasContent {
			score += 200
		}

		if p.Bootable {
			score += 50
		}

		scored = append(scored, scoredPartition{
			part:  p,
			score: score,
			size:  p.Size,
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].size > scored[j].size
	})

	return scored[0].part
}

type scoredPartition struct {
	part  *Partition
	score int
	size  uint64
}

func formatMBRType(t byte) string {
	types := map[byte]string{
		0x01: "FAT12",
		0x04: "FAT16",
		0x06: "FAT16",
		0x07: "NTFS",
		0x0B: "FAT32",
		0x0C: "FAT32",
		0x0E: "FAT16",
		0x0F: "Extended",
		0x83: "Linux",
		0x82: "Swap",
	}
	if t, ok := types[t]; ok {
		return t
	}
	return fmt.Sprintf("0x%02X", t)
}

func formatGPTType(guid []byte) string {
	if len(guid) < 16 {
		return "Unknown"
	}

	// GPT GUIDs are mixed-endian: first 3 fields little-endian, last 2 big-endian
	// Convert to standard GUID string format for comparison
	g := fmt.Sprintf("%02X%02X%02X%02X-%02X%02X-%02X%02X-%02X%02X-%02X%02X%02X%02X%02X%02X",
		guid[3], guid[2], guid[1], guid[0], // little-endian uint32
		guid[5], guid[4],                     // little-endian uint16
		guid[7], guid[6],                     // little-endian uint16
		guid[8], guid[9],                     // big-endian
		guid[10], guid[11], guid[12], guid[13], guid[14], guid[15]) // big-endian

	knownTypes := map[string]string{
		"A2A0D0EB-E5B9-3344-87C0-68B6B72699C7": "NTFS",
		"EBD0A0A2-B9E5-4433-87C0-68B6B72699C7": "Microsoft Basic Data",
		"E3C9E316-0B5C-B810-4DA7-080039EC1443": "Microsoft Reserved",
		"C12A7328-F81F-11D2-BA4B-00A0C93EC93B": "EFI System",
		"0FC63DAF-8483-4772-8E79-3D69D8477DE4": "Linux Filesystem",
		"16E3C9AF-B71D-4363-8948-48F6073D57C8": "Linux Swap",
		"BDF07024-C1FE-48F4-9846-56BF032E5273": "Linux LVM",
		"DE94BBA4-06D1-4D40-A16A-BFD50179D6AC": "Windows Recovery",
		"21686148-6449-6E6F-744E-656564454649": "BIOS Boot",
		"BC13C2FF-59E6-4262-A352-B275FD6F7172": "Windows Storage",
	}

	if t, ok := knownTypes[g]; ok {
		return t
	}
	return "Unknown"
}

func trimUTF16LE(data []byte) string {
	var runes []rune
	for i := 0; i+1 < len(data); i += 2 {
		r := rune(data[i]) | rune(data[i+1])<<8
		if r == 0 {
			break
		}
		runes = append(runes, r)
	}
	return string(runes)
}

func readLE32(data []byte, offset int) uint32 {
	if offset+4 > len(data) {
		return 0
	}
	return uint32(data[offset]) | uint32(data[offset+1])<<8 | uint32(data[offset+2])<<16 | uint32(data[offset+3])<<24
}

func readLE64(data []byte, offset int) uint64 {
	if offset+8 > len(data) {
		return 0
	}
	return uint64(data[offset]) | uint64(data[offset+1])<<8 | uint64(data[offset+2])<<16 | uint64(data[offset+3])<<24 |
		uint64(data[offset+4])<<32 | uint64(data[offset+5])<<40 | uint64(data[offset+6])<<48 | uint64(data[offset+7])<<56
}

// checkPartitionContent checks if a partition has files by reading its root directory.
// For ext4: reads root inode (inode 2) and checks if its first data block has directory entries.
// For FAT: the filesystem always has a root directory.
// For NTFS: the MFT always exists.
func (s *SmartOpen) checkPartitionContent(disk VirtualDisk, part Partition) bool {
	if part.Filesystem == "Unknown" {
		return false
	}

	switch part.Filesystem {
	case "ext4", "ext3", "ext2":
		return s.checkExt4Content(disk, part)
	case "FAT32", "FAT16", "FAT12":
		return true // FAT always has root directory
	case "NTFS":
		return true // NTFS always has MFT
	default:
		return false
	}
}

// checkExt4Content reads the root inode (inode 2) and checks if it has directory entries.
func (s *SmartOpen) checkExt4Content(disk VirtualDisk, part Partition) bool {
	// Read superblock at byte 1024 (sector part.Start + 2)
	sbData, err := disk.ReadSectors(part.Start+2, 3)
	if err != nil || len(sbData) < 0x5C {
		return false
	}

	// Verify ext4 magic
	magic := uint16(sbData[0x38]) | uint16(sbData[0x39])<<8
	if magic != 0xEF53 {
		return false
	}

	// Parse key superblock fields
	blockSize := uint32(1024) << uint32(sbData[0x18]) // s_log_block_size
	inodeSize := uint16(sbData[0x58]) | uint16(sbData[0x59])<<8
	if inodeSize == 0 {
		inodeSize = 128
	}

	// BGDT starts at the block after the superblock
	bgdtBlock := uint32(1)
	if blockSize == 1024 {
		bgdtBlock = 2
	}

	// Read first block group descriptor (32 bytes)
	bgdtSector := part.Start + uint64(bgdtBlock)*uint64(blockSize)/512
	bgdtData, err := disk.ReadSectors(bgdtSector, 1)
	if err != nil || len(bgdtData) < 32 {
		return false
	}

	// inode table block number from first BGDT entry (offset 8-11)
	inodeTableBlock := uint32(bgdtData[8]) | uint32(bgdtData[9])<<8 |
		uint32(bgdtData[10])<<16 | uint32(bgdtData[11])<<24

	// Read inode 2 (root directory)
	inodeOffset := uint64(inodeTableBlock)*uint64(blockSize) + uint64(inodeSize)
	inodeSector := part.Start + inodeOffset/512
	inodeWithinSector := inodeOffset % 512

	inodeRaw, err := disk.ReadSectors(inodeSector, 2)
	if err != nil {
		return false
	}

	// Extract inode 2 from the sector data
	inode := make([]byte, inodeSize)
	copy(inode, inodeRaw[inodeWithinSector:])

	// Check inode mode (offset 0-1): should be directory (0x4000)
	mode := uint16(inode[0]) | uint16(inode[1])<<8
	if mode&0xF000 != 0x4000 {
		return false
	}

	// Check inode size
	sizeLo := uint32(inode[4]) | uint32(inode[5])<<8 | uint32(inode[6])<<16 | uint32(inode[7])<<24
	if sizeLo == 0 {
		return false
	}

	// Determine the physical block containing directory data
	var dirSector uint64
	sectorsToRead := blockSize / 512
	if sectorsToRead < 1 {
		sectorsToRead = 1
	}

	// Check if inode uses extents (i_flags bit 19 = 0x00080000)
	iFlags := uint32(inode[0x20]) | uint32(inode[0x21])<<8 | uint32(inode[0x22])<<16 | uint32(inode[0x23])<<24
	if iFlags&0x00080000 != 0 {
		// Extent-based inode: parse extent tree
		ehMagic := uint16(inode[0x28]) | uint16(inode[0x29])<<8
		if ehMagic != 0xF30A {
			return false
		}
		ehEntries := uint16(inode[0x2A]) | uint16(inode[0x2B])<<8
		ehDepth := uint16(inode[0x2E]) | uint16(inode[0x2F])<<8
		if ehEntries == 0 {
			return false
		}

		if ehDepth == 0 {
			// Leaf node: first extent entry at offset 0x28 + 12 = 0x34
			off := uint32(0x34)
			if off+12 > uint32(len(inode)) {
				return false
			}
			_ = uint32(inode[off]) | uint32(inode[off+1])<<8 | uint32(inode[off+2])<<16 | uint32(inode[off+3])<<24 // ee_block
			_ = uint16(inode[off+4]) | uint16(inode[off+5])<<8                                                          // ee_len
			eeStartHi := uint32(inode[off+6]) | uint32(inode[off+7])<<8
			eeStartLo := uint32(inode[off+8]) | uint32(inode[off+9])<<8 | uint32(inode[off+10])<<16 | uint32(inode[off+11])<<24
			physBlock := uint64(eeStartHi)<<32 | uint64(eeStartLo)
			if physBlock == 0 {
				return false
			}
			dirSector = part.Start + physBlock*uint64(blockSize)/512
		} else {
			// Index node: read child leaf block from disk
			off := uint32(0x34)
			if off+12 > uint32(len(inode)) {
				return false
			}
			leafLo := uint32(inode[off+4]) | uint32(inode[off+5])<<8 | uint32(inode[off+6])<<16 | uint32(inode[off+7])<<24
			leafHi := uint32(inode[off+8]) | uint32(inode[off+9])<<8
			leafBlock := uint64(leafHi)<<32 | uint64(leafLo)
			if leafBlock == 0 {
				return false
			}
			leafSector := part.Start + leafBlock*uint64(blockSize)/512
			leafData, err := disk.ReadSectors(leafSector, uint32(blockSize/512))
			if err != nil || len(leafData) < 24 {
				return false
			}
			// Parse leaf node extent entries
			leafEntries := uint16(leafData[0x0A]) | uint16(leafData[0x0B])<<8
			leafDepth := uint16(leafData[0x0E]) | uint16(leafData[0x0F])<<8
			if leafEntries == 0 || leafDepth != 0 {
				return false
			}
			extOff := uint32(12)
			if extOff+12 > uint32(len(leafData)) {
				return false
			}
			eStartHi := uint32(leafData[extOff+6]) | uint32(leafData[extOff+7])<<8
			eStartLo := uint32(leafData[extOff+8]) | uint32(leafData[extOff+9])<<8 | uint32(leafData[extOff+10])<<16 | uint32(leafData[extOff+11])<<24
			physBlock := uint64(eStartHi)<<32 | uint64(eStartLo)
			if physBlock == 0 {
				return false
			}
			dirSector = part.Start + physBlock*uint64(blockSize)/512
		}
	} else {
		// Direct block pointer
		firstBlock := uint32(inode[40]) | uint32(inode[41])<<8 | uint32(inode[42])<<16 | uint32(inode[43])<<24
		if firstBlock == 0 {
			return false
		}
		dirSector = part.Start + uint64(firstBlock)*uint64(blockSize)/512
	}

	// Read the data block and check for directory entries
	dirData, err := disk.ReadSectors(dirSector, uint32(sectorsToRead))
	if err != nil {
		return false
	}

	// Check for non-zero directory entries
	offset := 0
	for offset+8 <= len(dirData) {
		entryInode := uint32(dirData[offset]) | uint32(dirData[offset+1])<<8 |
			uint32(dirData[offset+2])<<16 | uint32(dirData[offset+3])<<24
		entryRecLen := uint16(dirData[offset+4]) | uint16(dirData[offset+5])<<8
		if entryRecLen == 0 || int(entryRecLen) > len(dirData)-offset {
			break
		}
		if entryInode != 0 {
			return true
		}
		offset += int(entryRecLen)
	}

	return false
}

// Compile-time check
var _ io.ReaderAt = (VirtualDisk)(nil)

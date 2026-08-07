package filesystems

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/user/vhd-opener/internal/infrastructure/logger"
	"go.uber.org/zap"
)

// POSIX Inode Mode Constants
const (
	sIFMT  uint16 = 0xF000 // Bitmask for file type
	sIFDIR uint16 = 0x4000 // Directory
	sIFREG uint16 = 0x8000 // Regular File
	sIFLNK uint16 = 0xA000 // Symbolic Link

	maxSymlinkHops = 8 // Maximum symlink resolution depth to prevent loops

	ext4ExtentsFl uint32 = 0x00080000 // EXT4_EXTENTS_FL
	ext4IndexFl   uint32 = 0x00001000 // EXT4_INDEX_FL (HTree)
	ext4ExtMagic  uint16 = 0xF30A     // Extent header magic

	// Feature flags
	featureLargeFile uint32 = 0x0002 // RO_COMPAT_LARGE_FILE
	feature64Bit     uint32 = 0x0080 // INCOMPAT_64BIT
)

// EXT4Reader implements Reader for ext2/3/4 filesystems.
type EXT4Reader struct {
	blockSize         uint32
	inodeSize         uint16
	inodesPerGroup    uint32
	numBlockGroups    uint32
	bgdtStart         uint64 // sector offset of block group descriptor table
	partStart         uint64
	blocksPerGroup    uint32
	descSize          uint16 // Block group descriptor size (32 or 64)
	largeFile         bool   // RO_COMPAT_LARGE_FILE feature
	featuresIncompat  uint32 // INCOMPAT feature flags (for 64BIT detection)
}

// NewEXT4Reader creates a new ext4 reader by parsing the superblock.
func NewEXT4Reader(diskReader DiskReader, partitionStart uint64) (*EXT4Reader, error) {
	// Superblock starts at byte 1024 (0x400) from partition start = sector 2
	sbData, err := diskReader.ReadSectors(partitionStart+2, 3) // sectors 2-4 (bytes 1024-2559)
	if err != nil {
		return nil, fmt.Errorf("ext4: failed to read superblock: %w", err)
	}

	// Verify magic at offset 0x38 within the superblock (byte 1080 from partition)
	magic := binary.LittleEndian.Uint16(sbData[0x38:0x3A])
	if magic != 0xEF53 {
		return nil, fmt.Errorf("ext4: invalid superblock magic: 0x%04X", magic)
	}

	r := &EXT4Reader{partStart: partitionStart}

	// Parse superblock fields
	r.blockSize = 1024 << binary.LittleEndian.Uint32(sbData[0x18:0x1C]) // s_log_block_size
	r.inodeSize = binary.LittleEndian.Uint16(sbData[0x58:0x5A])          // s_inode_size
	r.inodesPerGroup = binary.LittleEndian.Uint32(sbData[0x28:0x2C])     // s_inodes_per_group
	blocksPerGroup32 := binary.LittleEndian.Uint32(sbData[0x20:0x24])    // s_blocks_per_group
	r.blocksPerGroup = blocksPerGroup32
	numBlocks64 := binary.LittleEndian.Uint32(sbData[0x04:0x08])         // s_blocks_count_lo
	r.numBlockGroups = (numBlocks64 + blocksPerGroup32 - 1) / blocksPerGroup32
	revLevel := binary.LittleEndian.Uint32(sbData[0x50:0x54])            // s_rev_level

	// FIX #2: s_feature_ro_compat at offset 0x64 in superblock
	roCompat := binary.LittleEndian.Uint32(sbData[0x64:0x68])
	r.largeFile = (roCompat & featureLargeFile) != 0

	// FIX #1: s_feature_incompat at offset 0x60 in superblock (for 64BIT detection)
	// NOTE: This field is at 0x60, NOT 0x100. Offset 0x100 is s_first_ino.
	if len(sbData) > 0x64 {
		r.featuresIncompat = binary.LittleEndian.Uint32(sbData[0x60:0x64])
	}

	// FIX #3: s_desc_size at offset 0xFC in superblock
	// CRITICAL: s_desc_size is only valid in rev 1+ when INCOMPAT_64BIT is set.
	// For rev 0 filesystems with 64BIT (possible when upgraded from ext2→ext4),
	// s_desc_size at 0xFC is actually s_reserved_gdt_blocks, not the descriptor size.
	// In that case, use 64 as the default.
	if r.featuresIncompat&feature64Bit != 0 {
		if revLevel >= 1 {
			r.descSize = binary.LittleEndian.Uint16(sbData[0xFC:0xFE])
			if r.descSize == 0 {
				r.descSize = 64
			}
		} else {
			r.descSize = 64
		}
	} else {
		r.descSize = 32
	}

	// BGDT starts right after the superblock
	if r.blockSize == 1024 {
		r.bgdtStart = partitionStart + 4 // byte 2048 = sector 4
	} else {
		r.bgdtStart = partitionStart + uint64(r.blockSize/512) // byte block_size
	}

	logger.Log.Info("ext4: reader initialized",
		zap.Uint32("blockSize", r.blockSize),
		zap.Uint16("inodeSize", r.inodeSize),
		zap.Uint16("descSize", r.descSize),
		zap.Bool("largeFile", r.largeFile),
		zap.Uint32("featuresIncompat", r.featuresIncompat),
		zap.Uint32("revLevel", revLevel),
		zap.Uint32("numBlockGroups", r.numBlockGroups),
	)

	return r, nil
}

// getBlockGroupDescriptor reads a block group descriptor from the BGDT.
// FIX #3: Uses r.descSize (32 or 64 bytes) instead of hardcoded 32.
// FIX #1: For 64BIT feature, reads bg_inode_table_hi from offset 40.
func (r *EXT4Reader) getBlockGroupDescriptor(diskReader DiskReader, bgIndex uint32) (blockBitmap, inodeBitmap uint32, inodeTable uint64, err error) {
	if bgIndex >= r.numBlockGroups {
		return 0, 0, 0, fmt.Errorf("ext4: block group index %d out of range (max %d)", bgIndex, r.numBlockGroups)
	}

	// Each descriptor is r.descSize bytes
	descOffset := uint64(bgIndex) * uint64(r.descSize)
	descSector := r.bgdtStart + descOffset/512
	descWithinSector := descOffset % 512

	// Read enough sectors to cover the descriptor
	sectorsNeeded := (descWithinSector + uint64(r.descSize) + 511) / 512
	if sectorsNeeded < 1 {
		sectorsNeeded = 1
	}
	data, err := diskReader.ReadSectors(descSector, uint32(sectorsNeeded))
	if err != nil {
		return 0, 0, 0, err
	}

	off := descWithinSector
	if off+12 > uint64(len(data)) {
		return 0, 0, 0, fmt.Errorf("ext4: BGD read overflow for group %d", bgIndex)
	}
	blockBitmap = binary.LittleEndian.Uint32(data[off : off+4])
	inodeBitmap = binary.LittleEndian.Uint32(data[off+4 : off+8])
	inodeTable = uint64(binary.LittleEndian.Uint32(data[off+8 : off+12]))

	// FIX #1: For 64BIT filesystems, bg_inode_table_hi is at offset 40 in the 64-byte descriptor
	if r.featuresIncompat&feature64Bit != 0 && r.descSize >= 64 && off+44 <= uint64(len(data)) {
		inodeTableHi := uint64(binary.LittleEndian.Uint32(data[off+40 : off+44]))
		inodeTable = (inodeTableHi << 32) | inodeTable
	}

	return
}

// getBlockLocation converts a logical block number to a sector offset on disk.
func (r *EXT4Reader) getBlockLocation(diskReader DiskReader, logicalBlock uint32) (uint64, error) {
	if logicalBlock == 0 {
		return 0, nil
	}

	bgIndex := (logicalBlock - 1) / r.blocksPerGroup

	firstDataBlock := uint32(0)
	if r.blockSize == 1024 {
		firstDataBlock = 1
	}

	groupStartBlock := bgIndex*r.blocksPerGroup + firstDataBlock
	offsetInGroup := (logicalBlock - 1) % r.blocksPerGroup
	physicalBlock := groupStartBlock + offsetInGroup

	return r.partStart + uint64(physicalBlock)*uint64(r.blockSize)/512, nil
}

// getInode reads an inode structure from disk.
func (r *EXT4Reader) getInode(diskReader DiskReader, inodeNum uint32) (inodeData []byte, err error) {
	if inodeNum == 0 {
		return nil, fmt.Errorf("ext4: invalid inode number 0")
	}

	bgIndex := (inodeNum - 1) / r.inodesPerGroup
	offsetInGroup := (inodeNum - 1) % r.inodesPerGroup

	_, _, inodeTableStart, err := r.getBlockGroupDescriptor(diskReader, bgIndex)
	if err != nil {
		return nil, err
	}

	// inodeTableStart is a block number (uint64 for 64BIT support), convert to sector
	inodeByteOffset := inodeTableStart*uint64(r.blockSize) + uint64(offsetInGroup)*uint64(r.inodeSize)
	inodeSector := r.partStart + inodeByteOffset/512
	inodeWithinSector := inodeByteOffset % 512

	sectorsNeeded := (inodeWithinSector + uint64(r.inodeSize) + 511) / 512
	data, err := diskReader.ReadSectors(inodeSector, uint32(sectorsNeeded))
	if err != nil {
		return nil, err
	}

	inodeData = make([]byte, r.inodeSize)
	copy(inodeData, data[inodeWithinSector:])
	return inodeData, nil
}

// getInodeSize returns the size of an inode's file data.
// FIX #1: i_size_high is at offset 0x6C (not 0x64 which is i_generation).
// FIX #2: i_size_high is only valid for regular files when LARGE_FILE feature is active.
func getInodeSize(inodeData []byte, inodeSize uint16, largeFile bool) int64 {
	// i_size_lo at offset 0x04 (4 bytes)
	sizeLo := int64(binary.LittleEndian.Uint32(inodeData[0x04:0x08]))

	// FIX #1 + #2: i_size_high at offset 0x6C, only for regular files with LARGE_FILE
	// i_size_high only exists in inodes >= 256 bytes (ext4, not ext2/3)
	if largeFile && inodeSize >= 0x100 && len(inodeData) > 0x70 {
		mode := binary.LittleEndian.Uint16(inodeData[0x00:0x02])
		if mode&0xF000 == sIFREG { // Regular file only
			sizeHi := int64(binary.LittleEndian.Uint32(inodeData[0x6C:0x70]))
			return sizeLo | (sizeHi << 32)
		}
	}
	return sizeLo
}

// getInodeMode returns the file mode (type + permissions).
func getInodeMode(inodeData []byte) uint16 {
	return binary.LittleEndian.Uint16(inodeData[0x00:0x02])
}

// hasExtents checks if the inode uses extent tree instead of block pointers.
func hasExtents(inodeData []byte) bool {
	if len(inodeData) < 0x24 {
		return false
	}
	iFlags := binary.LittleEndian.Uint32(inodeData[0x20:0x24])
	return iFlags&ext4ExtentsFl != 0
}

// hasIndex checks if the inode uses HTree indexing for directory entries.
func hasIndex(inodeData []byte) bool {
	if len(inodeData) < 0x24 {
		return false
	}
	iFlags := binary.LittleEndian.Uint32(inodeData[0x20:0x24])
	return iFlags&ext4IndexFl != 0
}

// extentEntry represents a single extent in an extent tree.
type extentEntry struct {
	fileBlock uint32 // first file block covered by this extent
	length    uint16 // number of blocks in this extent
	physBlock uint64 // physical block number (48-bit)
}

// parseExtents parses the extent tree from an inode's i_block area.
func (r *EXT4Reader) parseExtents(diskReader DiskReader, inodeData []byte) ([]extentEntry, error) {
	if len(inodeData) < 0x34 {
		return nil, fmt.Errorf("ext4: inode too small for extent header")
	}

	ehMagic := binary.LittleEndian.Uint16(inodeData[0x28:0x2A])
	if ehMagic != ext4ExtMagic {
		return nil, fmt.Errorf("ext4: invalid extent magic 0x%04X", ehMagic)
	}

	ehEntries := binary.LittleEndian.Uint16(inodeData[0x2A:0x2C])
	ehDepth := binary.LittleEndian.Uint16(inodeData[0x2E:0x30])

	if ehDepth == 0 {
		return r.parseExtentLeaf(inodeData, 0x34, ehEntries), nil
	}

	return r.parseExtentIndexNode(diskReader, inodeData, 0x34, ehEntries, ehDepth)
}

func (r *EXT4Reader) parseExtentLeaf(data []byte, startOffset uint16, count uint16) []extentEntry {
	var entries []extentEntry
	for i := uint16(0); i < count; i++ {
		off := uint32(startOffset) + uint32(i)*12
		if off+12 > uint32(len(data)) {
			break
		}
		eeBlock := binary.LittleEndian.Uint32(data[off : off+4])
		eeLen := binary.LittleEndian.Uint16(data[off+4 : off+6])
		eeStartHi := binary.LittleEndian.Uint16(data[off+6 : off+8])
		eeStartLo := binary.LittleEndian.Uint32(data[off+8 : off+12])
		physBlock := uint64(eeStartHi)<<32 | uint64(eeStartLo)
		entries = append(entries, extentEntry{
			fileBlock: eeBlock,
			length:    eeLen,
			physBlock: physBlock,
		})
	}
	return entries
}

func (r *EXT4Reader) parseExtentIndexNode(diskReader DiskReader, data []byte, startOffset uint16, count uint16, depth uint16) ([]extentEntry, error) {
	var allEntries []extentEntry

	for i := uint16(0); i < count; i++ {
		off := uint32(startOffset) + uint32(i)*12
		if off+12 > uint32(len(data)) {
			break
		}
		_ = binary.LittleEndian.Uint32(data[off : off+4])
		leafBlockLo := binary.LittleEndian.Uint32(data[off+4 : off+8])
		leafBlockHi := binary.LittleEndian.Uint16(data[off+8 : off+10])
		leafBlock := uint64(leafBlockHi)<<32 | uint64(leafBlockLo)

		childSector := r.partStart + leafBlock*uint64(r.blockSize)/512
		childData, err := diskReader.ReadSectors(childSector, uint32(r.blockSize/512))
		if err != nil {
			return nil, fmt.Errorf("ext4: failed to read extent child block %d: %w", leafBlock, err)
		}

		if len(childData) < 12 {
			continue
		}
		childMagic := binary.LittleEndian.Uint16(childData[0:2])
		if childMagic != ext4ExtMagic {
			continue
		}
		childEntries := binary.LittleEndian.Uint16(childData[0x0A:0x0C])
		childDepth := binary.LittleEndian.Uint16(childData[0x0E:0x10])

		if childDepth == 0 {
			entries := r.parseExtentLeaf(childData, 12, childEntries)
			allEntries = append(allEntries, entries...)
		} else {
			subEntries, err := r.parseExtentIndexNode(diskReader, childData, 12, childEntries, childDepth)
			if err == nil {
				allEntries = append(allEntries, subEntries...)
			}
		}
	}

	return allEntries, nil
}

func (r *EXT4Reader) readDataFromExtent(diskReader DiskReader, inodeData []byte, size int64, maxBytes int64) ([]byte, error) {
	extents, err := r.parseExtents(diskReader, inodeData)
	if err != nil {
		return nil, err
	}

	var result []byte
	var bytesRead int64

	for _, ext := range extents {
		if bytesRead >= size {
			break
		}

		blocksNeeded := (size - bytesRead + int64(r.blockSize) - 1) / int64(r.blockSize)
		blocksToRead := int64(ext.length)
		if blocksToRead > blocksNeeded {
			blocksToRead = blocksNeeded
		}

		for b := int64(0); b < blocksToRead; b++ {
			if bytesRead >= size {
				break
			}

			physBlock := ext.physBlock + uint64(b)
			sector := r.partStart + physBlock*uint64(r.blockSize)/512

			data, err := diskReader.ReadSectors(sector, uint32(r.blockSize/512))
			if err != nil {
				return nil, err
			}

			remaining := size - bytesRead
			if remaining < int64(len(data)) {
				data = data[:remaining]
			}
			if maxBytes > 0 && int64(len(result))+int64(len(data)) > maxBytes {
				allowed := maxBytes - int64(len(result))
				data = data[:allowed]
			}
			result = append(result, data...)
			bytesRead += int64(len(data))
		}
	}

	return result, nil
}

func (r *EXT4Reader) readDataBlocks(diskReader DiskReader, inodeData []byte, size int64, maxBytes int64) ([]byte, error) {
	if size == 0 {
		return nil, nil
	}

	if hasExtents(inodeData) {
		return r.readDataFromExtent(diskReader, inodeData, size, maxBytes)
	}

	ptrs := getInodeBlockPtrs(inodeData)
	return r.readDataFromBlockPtrs(diskReader, ptrs, size, maxBytes)
}

// getInodeBlockPtrs returns the block pointers from an inode (15 total: 12 direct + 1 indirect + 1 double + 1 triple).
func getInodeBlockPtrs(inodeData []byte) []uint32 {
	ptrs := make([]uint32, 15)
	for i := 0; i < 15; i++ {
		off := 0x28 + i*4
		if off+4 <= len(inodeData) {
			ptrs[i] = binary.LittleEndian.Uint32(inodeData[off : off+4])
		}
	}
	return ptrs
}

// FIX #6: readDataFromBlockPtrs now supports single indirect blocks.
func (r *EXT4Reader) readDataFromBlockPtrs(diskReader DiskReader, ptrs []uint32, size int64, maxBytes int64) ([]byte, error) {
	var result []byte
	var bytesRead int64
	blockSize := int64(r.blockSize)
	ptrsPerBlock := int64(r.blockSize / 4) // number of uint32 pointers per block

	// Helper to read a single logical block and append to result
	readBlock := func(logicalBlock uint32) error {
		if bytesRead >= size || logicalBlock == 0 {
			return nil
		}
		sector, err := r.getBlockLocation(diskReader, logicalBlock)
		if err != nil {
			return err
		}
		data, err := diskReader.ReadSectors(sector, uint32(r.blockSize/512))
		if err != nil {
			return err
		}
		remaining := size - bytesRead
		if remaining < int64(len(data)) {
			data = data[:remaining]
		}
		if maxBytes > 0 && int64(len(result))+int64(len(data)) > maxBytes {
			allowed := maxBytes - int64(len(result))
			data = data[:allowed]
		}
		result = append(result, data...)
		bytesRead += int64(len(data))
		return nil
	}

	// Helper to read a block of pointers from disk
	readPtrBlock := func(blockNum uint32) ([]uint32, error) {
		if blockNum == 0 {
			return nil, nil
		}
		sector, err := r.getBlockLocation(diskReader, blockNum)
		if err != nil {
			return nil, err
		}
		data, err := diskReader.ReadSectors(sector, uint32(r.blockSize/512))
		if err != nil {
			return nil, err
		}
		ptrs2 := make([]uint32, ptrsPerBlock)
		for i := int64(0); i < ptrsPerBlock && int64(i)*4+4 <= int64(len(data)); i++ {
			ptrs2[i] = binary.LittleEndian.Uint32(data[i*4 : i*4+4])
		}
		return ptrs2, nil
	}

	// 12 direct blocks
	for i := 0; i < 12 && bytesRead < size; i++ {
		if err := readBlock(ptrs[i]); err != nil {
			return nil, err
		}
	}

	// Single indirect block (ptrs[12])
	if bytesRead < size && ptrs[12] != 0 {
		singlePtrs, err := readPtrBlock(ptrs[12])
		if err != nil {
			return nil, err
		}
		for i := int64(0); i < ptrsPerBlock && bytesRead < size; i++ {
			if err := readBlock(singlePtrs[i]); err != nil {
				return nil, err
			}
		}
	}

	// Double indirect block (ptrs[13])
	if bytesRead < size && ptrs[13] != 0 {
		doublePtrs, err := readPtrBlock(ptrs[13])
		if err != nil {
			return nil, err
		}
		for i := int64(0); i < ptrsPerBlock && bytesRead < size; i++ {
			if doublePtrs[i] == 0 {
				// Sparse block - advance by one block worth of space
				remaining := size - bytesRead
				skip := blockSize
				if remaining < skip {
					skip = remaining
				}
				result = append(result, make([]byte, skip)...)
				bytesRead += skip
				continue
			}
			innerPtrs, err := readPtrBlock(doublePtrs[i])
			if err != nil {
				return nil, err
			}
			for j := int64(0); j < ptrsPerBlock && bytesRead < size; j++ {
				if err := readBlock(innerPtrs[j]); err != nil {
					return nil, err
				}
			}
		}
	}

	return result, nil
}

// FIX #4 + #5: HTree indexed directory support.
// readDirEntries reads directory entries, handling both linear and HTree-indexed directories.
func (r *EXT4Reader) readDirEntries(diskReader DiskReader, inodeData []byte) ([]FileEntry, error) {
	size := getInodeSize(inodeData, r.inodeSize, r.largeFile)

	// Check if directory uses HTree indexing
	if hasIndex(inodeData) {
		return r.readDirEntriesIndexed(diskReader, inodeData, size)
	}

	// Linear directory - read all data and parse
	data, err := r.readDataBlocks(diskReader, inodeData, size, 10*1024*1024)
	if err != nil {
		return nil, err
	}
	return listDirEntries(data)
}

// readDirEntriesIndexed parses an HTree-indexed directory.
// The first block contains dx_root with index entries pointing to leaf blocks.
func (r *EXT4Reader) readDirEntriesIndexed(diskReader DiskReader, inodeData []byte, size int64) ([]FileEntry, error) {
	// Read all directory data blocks
	allData, err := r.readDataBlocks(diskReader, inodeData, size, 10*1024*1024)
	if err != nil {
		return nil, err
	}

	if len(allData) < int(r.blockSize) {
		return listDirEntries(allData)
	}

	// Parse dx_root from first block
	block0 := allData[:r.blockSize]

	// First entry is "." (dot entry) - read its rec_len to find dx_root_info
	if len(block0) < 8 {
		return listDirEntries(allData)
	}
	dotInode := binary.LittleEndian.Uint32(block0[0:4])
	dotRecLen := binary.LittleEndian.Uint16(block0[4:6])
	if dotInode == 0 || dotRecLen < 8 || int(dotRecLen) > len(block0) {
		return listDirEntries(allData)
	}

	// dx_root_info starts after the "." entry, aligned to 8 bytes
	rootInfoOffset := (uint32(dotRecLen) + 7) & ^uint32(7)
	if rootInfoOffset+8 > uint32(len(block0)) {
		return listDirEntries(allData)
	}

	// Validate dx_root_info
	reservedZero := binary.LittleEndian.Uint32(block0[rootInfoOffset : rootInfoOffset+4])
	infoLength := uint32(block0[rootInfoOffset+5])
	indirectLevels := block0[rootInfoOffset+6]

	if reservedZero != 0 || infoLength < 8 {
		// Not a valid dx_root, fall back to linear parse
		return listDirEntries(allData)
	}

	// dx_entries start after dx_root_info
	entriesOffset := rootInfoOffset + infoLength
	if entriesOffset >= uint32(len(block0)) {
		return listDirEntries(allData)
	}

	// Parse dx_entries: each is 8 bytes (hash:4 + block:4)
	numDxEntries := (uint32(len(block0)) - entriesOffset) / 8

	logger.Log.Debug("ext4: HTree indexed directory",
		zap.Uint16("dotRecLen", dotRecLen),
		zap.Uint32("rootInfoOffset", rootInfoOffset),
		zap.Uint32("infoLength", infoLength),
		zap.Uint8("indirectLevels", indirectLevels),
		zap.Uint32("numDxEntries", numDxEntries),
	)

	// Collect leaf block numbers
	var leafBlockNums []uint32
	for i := uint32(0); i < numDxEntries; i++ {
		off := entriesOffset + i*8
		if off+8 > uint32(len(block0)) {
			break
		}
		block := binary.LittleEndian.Uint32(block0[off+4 : off+8])
		if block != 0 {
			leafBlockNums = append(leafBlockNums, block)
		}
	}

	if len(leafBlockNums) == 0 {
		return listDirEntries(allData)
	}

	// For depth > 0, we need to recursively resolve index nodes.
	// For depth == 0, leafBlockNums directly point to leaf blocks.
	if indirectLevels == 0 {
		// Depth 0: leaf blocks contain directory entries
		return r.parseLeafBlocks(allData, leafBlockNums)
	}

	// Depth > 0: leaf blocks point to more index nodes
	// Read each intermediate block and extract leaf block numbers
	var leafBlocks []uint32
	for _, blockNum := range leafBlockNums {
		blockOffset := int(blockNum) * int(r.blockSize)
		if blockOffset+int(r.blockSize) > len(allData) {
			continue
		}
		intermediateData := allData[blockOffset : blockOffset+int(r.blockSize)]

		// Parse the intermediate node header
		if len(intermediateData) < 12 {
			continue
		}
		childEntries := binary.LittleEndian.Uint16(intermediateData[0x0A:0x0C])

		// Entries start at offset 12
		for j := uint16(0); j < childEntries; j++ {
			entryOff := 12 + j*8
			if int(entryOff)+8 > len(intermediateData) {
				break
			}
			childBlock := binary.LittleEndian.Uint32(intermediateData[entryOff+4 : entryOff+8])
			if childBlock != 0 {
				leafBlocks = append(leafBlocks, childBlock)
			}
		}
	}

	if len(leafBlocks) == 0 {
		leafBlocks = leafBlockNums
	}

	return r.parseLeafBlocks(allData, leafBlocks)
}

// parseLeafBlocks extracts directory entries from leaf blocks.
func (r *EXT4Reader) parseLeafBlocks(allData []byte, leafBlockNums []uint32) ([]FileEntry, error) {
	var allEntries []FileEntry
	for _, blockNum := range leafBlockNums {
		blockOffset := int(blockNum) * int(r.blockSize)
		if blockOffset+int(r.blockSize) > len(allData) {
			continue
		}
		leafData := allData[blockOffset : blockOffset+int(r.blockSize)]
		entries, err := listDirEntries(leafData)
		if err != nil {
			continue
		}
		allEntries = append(allEntries, entries...)
	}

	if len(allEntries) == 0 {
		// Fallback: try parsing all data as linear directory
		return listDirEntries(allData)
	}

	return allEntries, nil
}

// listDirEntries parses directory entries from raw block data.
// FIX #5: Properly skips inode==0 (deleted/unused) entries and validates rec_len.
func listDirEntries(data []byte) ([]FileEntry, error) {
	var entries []FileEntry
	offset := 0

	for offset+8 <= len(data) {
		inode := binary.LittleEndian.Uint32(data[offset : offset+4])
		recLen := binary.LittleEndian.Uint16(data[offset+4 : offset+6])
		nameLen := uint8(data[offset+6])
		fileType := uint8(data[offset+7])

		// Guard against corrupt/zero-length records (infinite loop prevention)
		if recLen < 8 {
			break
		}

		// FIX #5: Skip unused/deleted directory slots (inode == 0)
		if inode != 0 && nameLen > 0 && int(nameLen) <= int(recLen)-8 {
			nameBytes := data[offset+8 : offset+8+int(nameLen)]
			name := string(nameBytes)

			// Skip . and .. entries
			if name != "." && name != ".." {
				isDir := fileType == 2 // EXT4_FT_DIR = 2
				entries = append(entries, FileEntry{
					ID:          fmt.Sprintf("%d", inode),
					Name:        name,
					IsDirectory: isDir,
					FileType:    fileType,
				})
			}
		}

		offset += int(recLen)
	}

	return entries, nil
}

// DetectFSType implements Reader.
func (r *EXT4Reader) DetectFSType(diskReader DiskReader, partitionStart uint64) FSType {
	return EXT2 // ext2/3/4 all use the same reader
}

// ListRootDirectory lists the root directory contents.
func (r *EXT4Reader) ListRootDirectory(diskReader DiskReader, partitionStart uint64) ([]FileEntry, error) {
	return r.ListDirectory(diskReader, partitionStart, "/")
}

// ListDirectory lists the contents of a directory at the given path.
func (r *EXT4Reader) ListDirectory(diskReader DiskReader, partitionStart uint64, dirPath string) ([]FileEntry, error) {
	dirPath = strings.ReplaceAll(dirPath, "\\", "/")
	dirPath = path.Clean(dirPath)
	if !strings.HasPrefix(dirPath, "/") {
		dirPath = "/" + dirPath
	}

	inodeNum, err := r.resolvePath(diskReader, dirPath)
	if err != nil {
		return nil, fmt.Errorf("ext4: %w", err)
	}

	inodeData, err := r.getInode(diskReader, inodeNum)
	if err != nil {
		return nil, fmt.Errorf("ext4: failed to read inode %d: %w", inodeNum, err)
	}

	mode := getInodeMode(inodeData)
	if mode&sIFMT != sIFDIR {
		return nil, fmt.Errorf("ext4: %s is not a directory (inode=%d, mode=0x%04X)", dirPath, inodeNum, mode)
	}

	// FIX #4 + #5: Use readDirEntries which handles HTree-indexed directories
	rawEntries, err := r.readDirEntries(diskReader, inodeData)
	if err != nil {
		return nil, err
	}

	// Enrich entries with mode/size info by reading their inodes
	var result []FileEntry
	for _, e := range rawEntries {
		inodeID, err := parseInodeID(e.ID)
		if err != nil || inodeID == 0 {
			result = append(result, e)
			continue
		}

		childData, err := r.getInode(diskReader, inodeID)
		if err != nil {
			result = append(result, e)
			continue
		}

		childMode := getInodeMode(childData)
		childSize := getInodeSize(childData, r.inodeSize, r.largeFile)
		childModeType := childMode & sIFMT

		// FIX #10: Trust the fileType byte from the directory entry as ground truth.
		// The inode i_mode can be garbage if the inode was read from a wrong location
		// (e.g. due to BGDT misalignment). The fileType byte is stored in the directory
		// data block itself and is reliable.
		if e.FileType != 2 && e.FileType != 7 {
			// Only override IsDirectory when the dir entry fileType is NOT already
			// telling us it's a directory (2) or symlink (7).
			switch childModeType {
			case sIFDIR:
				e.IsDirectory = true
				e.Size = 0
			case sIFREG:
				e.IsDirectory = false
				e.Size = childSize
			case sIFLNK:
				e.IsDirectory = false
				e.Size = 0
			default:
				e.Size = childSize
			}
		} else if e.FileType == 7 {
			// Symlink: trust fileType and read symlink target
			e.IsDirectory = false
			e.Size = 0
			linkTarget, err := r.readSymlinkTarget(diskReader, childData)
			if err == nil && linkTarget != "" {
				if !strings.HasPrefix(linkTarget, "/") {
					linkTarget = path.Clean(dirPath + "/" + linkTarget)
				}
				e.ID = fmt.Sprintf("link:%s", linkTarget)
			}
		} else {
			// fileType == 2 (directory): keep IsDirectory=true from listDirEntries,
			// just fix up the size.
			e.IsDirectory = true
			e.Size = 0
		}

		// Always update size for regular files even when fileType says directory
		// (shouldn't happen, but handle gracefully)
		if e.FileType == 1 {
			e.Size = childSize
		}

		// FIX #4: Get modification time from correct inode offset (0x10 = i_mtime)
		if len(childData) > 0x14 {
			mtime := binary.LittleEndian.Uint32(childData[0x10:0x14])
			if mtime > 0 {
				e.ModifiedTime = time.Unix(int64(mtime), 0)
			}
		}

		result = append(result, e)
	}

	return result, nil
}

// resolvePath walks a path and returns the inode number of the final component.
func (r *EXT4Reader) resolvePath(diskReader DiskReader, resolvedPath string) (uint32, error) {
	return r.resolvePathHops(diskReader, resolvedPath, 0)
}

func (r *EXT4Reader) resolvePathHops(diskReader DiskReader, resolvedPath string, depth int) (uint32, error) {
	if depth > maxSymlinkHops {
		return 0, fmt.Errorf("ext4: symlink resolution depth limit exceeded (%d hops)", maxSymlinkHops)
	}

	resolvedPath = strings.ReplaceAll(resolvedPath, "\\", "/")
	resolvedPath = path.Clean(resolvedPath)
	if !strings.HasPrefix(resolvedPath, "/") {
		resolvedPath = "/" + resolvedPath
	}

	if resolvedPath == "/" {
		return 2, nil // root inode
	}

	parts := strings.Split(strings.Trim(resolvedPath, "/"), "/")
	currentInode := uint32(2)

	for idx, part := range parts {
		if part == "" {
			continue
		}

		inodeData, err := r.getInode(diskReader, currentInode)
		if err != nil {
			return 0, err
		}

		mode := getInodeMode(inodeData)
		modeType := mode & sIFMT

		// If current inode is a symlink, dereference it
		if modeType == sIFLNK {
			target, err := r.readSymlinkTarget(diskReader, inodeData)
			if err != nil {
				return 0, fmt.Errorf("ext4: failed to read symlink: %w", err)
			}

			var newPath string
			if strings.HasPrefix(target, "/") {
				newPath = target
			} else {
				parentDir := "/"
				if idx > 0 {
					parentDir = "/" + strings.Join(parts[:idx], "/")
				}
				newPath = path.Clean(parentDir + "/" + target)
			}

			remaining := strings.Join(parts[idx:], "/")
			if remaining != "" {
				newPath = path.Clean(newPath + "/" + remaining)
			}

			return r.resolvePathHops(diskReader, newPath, depth+1)
		}

		// Strict directory check before listing entries
		if modeType != sIFDIR {
			dirPath := "/"
			if idx > 0 {
				dirPath = "/" + strings.Join(parts[:idx+1], "/")
			}
			return 0, fmt.Errorf("ext4: %s is not a directory", dirPath)
		}

		// FIX #4 + #5: Use readDirEntries for HTree support
		entries, err := r.readDirEntries(diskReader, inodeData)
		if err != nil {
			return 0, err
		}

		found := false
		for _, e := range entries {
			if e.Name == part {
				if strings.HasPrefix(e.ID, "link:") {
					symlinkTarget := e.ID[5:]
					remaining := strings.Join(parts[idx+1:], "/")
					var newPath string
					if remaining != "" {
						newPath = path.Clean(symlinkTarget + "/" + remaining)
					} else {
						newPath = symlinkTarget
					}
					return r.resolvePathHops(diskReader, newPath, depth+1)
				}
				childInode, err := parseInodeID(e.ID)
				if err != nil {
					return 0, fmt.Errorf("ext4: failed to parse inode for %q: %w", part, err)
				}
				currentInode = childInode
				found = true
				break
			}
		}

		if !found {
			return 0, fmt.Errorf("ext4: path component %q not found in %s", part, resolvedPath)
		}
	}

	// Dereference final inode if it's a symlink
	finalData, err := r.getInode(diskReader, currentInode)
	if err == nil {
		finalMode := getInodeMode(finalData)
		if finalMode&sIFMT == sIFLNK {
			target, err := r.readSymlinkTarget(diskReader, finalData)
			if err != nil {
				return 0, err
			}
			var newPath string
			if strings.HasPrefix(target, "/") {
				newPath = target
			} else {
				newPath = path.Clean(path.Dir(resolvedPath) + "/" + target)
			}
			return r.resolvePathHops(diskReader, newPath, depth+1)
		}
	}

	return currentInode, nil
}

// readSymlinkTarget reads the target path from a symlink inode.
func (r *EXT4Reader) readSymlinkTarget(diskReader DiskReader, inodeData []byte) (string, error) {
	size := getInodeSize(inodeData, r.inodeSize, r.largeFile)
	if size == 0 {
		return "", fmt.Errorf("ext4: symlink has zero length")
	}

	// Fast symlink: target stored directly in i_block area (offset 0x28, 60 bytes)
	if size < 60 {
		targetBytes := inodeData[0x28 : 0x28+int(size)]
		nullIdx := bytes.IndexByte(targetBytes, 0x00)
		if nullIdx != -1 {
			return string(targetBytes[:nullIdx]), nil
		}
		return string(targetBytes), nil
	}

	// Slow symlink: target stored in data blocks
	blockData, err := r.readDataBlocks(diskReader, inodeData, size, 1024)
	if err != nil {
		return "", fmt.Errorf("ext4: failed to read slow symlink data: %w", err)
	}
	nullIdx := bytes.IndexByte(blockData, 0x00)
	if nullIdx != -1 {
		return string(blockData[:nullIdx]), nil
	}
	return string(blockData), nil
}

// FIX #7: GetFileContent now handles symlink IDs ("link:..." prefix).
func (r *EXT4Reader) GetFileContent(diskReader DiskReader, partitionStart uint64, file *FileEntry) ([]byte, error) {
	var inodeID uint32

	if strings.HasPrefix(file.ID, "link:") {
		// Symlink ID - resolve the target path
		targetPath := file.ID[5:]
		var err error
		inodeID, err = r.resolvePath(diskReader, targetPath)
		if err != nil {
			return nil, fmt.Errorf("ext4: failed to resolve symlink target %s: %w", targetPath, err)
		}
	} else {
		var err error
		inodeID, err = parseInodeID(file.ID)
		if err != nil {
			return nil, err
		}
	}

	inodeData, err := r.getInode(diskReader, inodeID)
	if err != nil {
		return nil, err
	}

	size := getInodeSize(inodeData, r.inodeSize, r.largeFile)
	return r.readDataBlocks(diskReader, inodeData, size, size)
}

// FIX #7: GetFileProperties now handles symlink IDs and correct timestamp offsets.
func (r *EXT4Reader) GetFileProperties(diskReader DiskReader, partitionStart uint64, file *FileEntry) (*FileProperties, error) {
	var inodeID uint32

	if strings.HasPrefix(file.ID, "link:") {
		targetPath := file.ID[5:]
		var err error
		inodeID, err = r.resolvePath(diskReader, targetPath)
		if err != nil {
			return nil, fmt.Errorf("ext4: failed to resolve symlink target %s: %w", targetPath, err)
		}
	} else {
		var err error
		inodeID, err = parseInodeID(file.ID)
		if err != nil {
			return nil, err
		}
	}

	inodeData, err := r.getInode(diskReader, inodeID)
	if err != nil {
		return nil, err
	}

	mode := getInodeMode(inodeData)
	size := getInodeSize(inodeData, r.inodeSize, r.largeFile)

	// FIX #4: Correct timestamp offsets from ext4 inode spec
	// i_atime = 0x08, i_ctime = 0x0C, i_mtime = 0x10
	var mtime, ctime, atime time.Time
	if len(inodeData) > 0x14 {
		at := binary.LittleEndian.Uint32(inodeData[0x08:0x0C])
		ct := binary.LittleEndian.Uint32(inodeData[0x0C:0x10])
		mt := binary.LittleEndian.Uint32(inodeData[0x10:0x14])
		if at > 0 {
			atime = time.Unix(int64(at), 0)
		}
		if ct > 0 {
			ctime = time.Unix(int64(ct), 0)
		}
		if mt > 0 {
			mtime = time.Unix(int64(mt), 0)
		}
	}

	return &FileProperties{
		Name:          file.Name,
		Extension:     file.Extension,
		FullPath:      file.Path,
		Size:          size,
		SizeFormatted: fmt.Sprintf("%d bytes", size),
		IsDirectory:   (mode & sIFMT) == sIFDIR,
		ModifiedTime:  mtime,
		CreatedTime:   ctime,
		AccessedTime:  atime,
	}, nil
}

// SearchFiles searches for files matching a query.
func (r *EXT4Reader) SearchFiles(diskReader DiskReader, partitionStart uint64, query string, caseSensitive bool) ([]FileEntry, error) {
	var results []FileEntry
	queryLower := strings.ToLower(query)

	var walkDir func(dirPath string, depth int)
	walkDir = func(dirPath string, depth int) {
		if depth > 10 {
			return
		}

		entries, err := r.ListDirectory(diskReader, partitionStart, dirPath)
		if err != nil {
			return
		}

		for _, e := range entries {
			name := e.Name
			if !caseSensitive {
				name = strings.ToLower(name)
			}
			if strings.Contains(name, queryLower) || strings.Contains(queryLower, name) {
				e.Path = path.Join(dirPath, e.Name)
				results = append(results, e)
			}
			if e.IsDirectory {
				walkDir(path.Join(dirPath, e.Name), depth+1)
			}
		}
	}

	walkDir("/", 0)
	return results, nil
}

func parseInodeID(id string) (uint32, error) {
	// FIX #7: Handle "link:" prefix gracefully
	if strings.HasPrefix(id, "link:") {
		return 0, fmt.Errorf("inode ID is a symlink reference: %s", id)
	}
	var n uint32
	for _, c := range id {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid inode id: %s", id)
		}
		n = n*10 + uint32(c-'0')
	}
	return n, nil
}

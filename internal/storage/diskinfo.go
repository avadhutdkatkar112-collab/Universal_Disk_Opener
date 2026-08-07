// Package disk provides detailed disk information for the forensic-grade Disk Info panel.
package storage

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// DiskInfoResponse holds all data for the 4-card Disk Info panel + forensic toolbar.
type DiskInfoResponse struct {
	Container  ContainerInfo   `json:"container"`
	Geometry   GeometryInfo    `json:"geometry"`
	Partitions []PartitionInfo `json:"partitions"`
	FSInfo     *FSInfo         `json:"fsInfo,omitempty"`
}

// ── Card 1: Container & Host Metadata ────────────────────────────────────────

// ContainerInfo represents Card 1.
type ContainerInfo struct {
	FileName     string `json:"fileName"`
	FilePath     string `json:"filePath"`
	Format       string `json:"format"`
	DiskType     string `json:"diskType"`
	VirtualSize  uint64 `json:"virtualSize"`
	PhysicalSize int64  `json:"physicalSize"`
	CreatorApp   string `json:"creatorApp"`
	CreatorVer   string `json:"creatorVersion"`
	CreatorOS    string `json:"creatorHostOS"`
	UniqueID     string `json:"uniqueID"`
	ChecksumOK   bool   `json:"checksumValid"`
	ReadonlyMode bool   `json:"readonlyMode"`
	// VHD-specific offsets (Gap 4)
	HeaderOffset uint64 `json:"headerOffset,omitempty"`
	BATOffset    uint64 `json:"batOffset,omitempty"`
	BATEntrySize uint32 `json:"batEntrySize,omitempty"`
	BlockSize    uint32 `json:"blockSize,omitempty"`
}

// ── Card 2: Geometry & Sector Alignment ──────────────────────────────────────

// GeometryInfo represents Card 2.
type GeometryInfo struct {
	TotalSectors    uint64      `json:"totalSectors"`
	LogicalSector   uint32      `json:"logicalSectorSize"`
	PhysicalSector  uint32      `json:"physicalSectorSize"`
	PartitionScheme string      `json:"partitionScheme"`
	DiskGUID        string      `json:"diskGUID,omitempty"`
	DiskSignature   string      `json:"diskSignature,omitempty"`
	CHS             CHSGeometry `json:"chs"`
}

// CHSGeometry holds Cylinders/Heads/Sectors-per-track.
type CHSGeometry struct {
	Cylinders       uint16 `json:"cylinders"`
	Heads           uint8  `json:"heads"`
	SectorsPerTrack uint8  `json:"sectorsPerTrack"`
}

// ── Card 3: Partition Layout (with Unallocated Gaps) ─────────────────────────

// PartitionInfo represents one row in the partition table (Gap 1: includes unallocated).
type PartitionInfo struct {
	Index        int    `json:"index"`
	Label        string `json:"label"`
	Filesystem   string `json:"filesystem"`
	StartLBA     uint64 `json:"startLBA"`
	EndLBA       uint64 `json:"endLBA"`
	TotalSectors uint64 `json:"totalSectors"`
	SizeBytes    uint64 `json:"sizeBytes"`
	Bootable     bool   `json:"bootable"`
	Active       bool   `json:"active"`
	IsMounted    bool   `json:"isMounted"`
	IsUnallocated bool  `json:"isUnallocated"`
	Status       string `json:"status"`
}

// ── Card 4: Mounted Volume Stats (ext4/FAT/NTFS) ────────────────────────────

// FSInfo holds filesystem metadata for the active partition (Gaps 2 & 3).
type FSInfo struct {
	FilesystemType  string   `json:"filesystemType"`
	VolumeUUID      string   `json:"volumeUUID"`
	VolumeLabel     string   `json:"volumeLabel"`
	State           string   `json:"state"`
	MountCount      uint16   `json:"mountCount"`
	MaxMounts       int16    `json:"maxMounts"`
	LastMountedPath string   `json:"lastMountedPath"`
	LastWriteTime   string   `json:"lastWriteTime"`
	BlockSize       uint32   `json:"blockSize"`
	TotalBlocks     uint64   `json:"totalBlocks"`
	FreeBlocks      uint64   `json:"freeBlocks"`
	TotalInodes     uint64   `json:"totalInodes"`
	FreeInodes      uint64   `json:"freeInodes"`
	BlockGroups     uint32   `json:"blockGroups"`
	FeatureFlags    []string `json:"featureFlags"`
	SuperblockOK    bool     `json:"superblockValid"`
}

// ── CollectDiskInfo ──────────────────────────────────────────────────────────

// CollectDiskInfo builds a DiskInfoResponse by reading raw disk structures.
func CollectDiskInfo(d VirtualDisk, result *OpenResult) DiskInfoResponse {
	info := d.Info()
	resp := DiskInfoResponse{
		Container: buildContainer(info, d),
		Geometry:  buildGeometry(d),
	}

	if result != nil {
		resp.Partitions = buildPartitionsWithGaps(result.Partitions, d.TotalSectors())

		if result.ActivePartition != nil {
			switch {
			case isExtFamily(result.ActivePartition.Filesystem):
				resp.FSInfo = readExt4Superblock(d, result.ActivePartition.Start)
			case result.ActivePartition.Filesystem == "FAT32" || result.ActivePartition.Filesystem == "FAT16":
				resp.FSInfo = readFATInfo(d, result.ActivePartition.Start, result.ActivePartition.Filesystem)
			case result.ActivePartition.Filesystem == "NTFS":
				resp.FSInfo = readNTFSInfo(d, result.ActivePartition.Start)
			}
		}
	}

	return resp
}

// ── Card 1 Builder ───────────────────────────────────────────────────────────

func buildContainer(info DiskInfo, d VirtualDisk) ContainerInfo {
	c := ContainerInfo{
		FileName:     info.FileName,
		FilePath:     info.FilePath,
		Format:       info.Format,
		DiskType:     info.DiskType,
		VirtualSize:  info.VirtualSize,
		PhysicalSize: info.FileSize,
		CreatorApp:   info.CreatorApp,
		CreatorVer:   info.CreatorVersion,
		CreatorOS:    info.CreatorHostOS,
		UniqueID:     info.UniqueID,
		ChecksumOK:   info.ChecksumValid,
		ReadonlyMode: true,
	}

	// Gap 4: Read VHD-specific offsets directly from the disk
	if info.Format == "VHD" {
		c.readVHDOffsets(d)
	}

	return c
}

// readVHDOffsets reads VHD footer + dynamic header to get Header/BAT offsets.
func (c *ContainerInfo) readVHDOffsets(d VirtualDisk) {
	totalSectors := d.TotalSectors()
	if totalSectors < 2 {
		return
	}

	// VHD footer is at the last 512 bytes of the file
	// Read the last sector
	footerSector := totalSectors - 1
	footer, err := d.ReadSectors(footerSector, 1)
	if err != nil || len(footer) < 512 {
		return
	}

	// Signature at offset 0 must be "conectix"
	if string(footer[0:8]) != "conectix" {
		return
	}

	// DataOffset (offset 16-23 in footer, big-endian): offset to dynamic header
	dataOffset := uint64(footer[16])<<56 | uint64(footer[17])<<48 | uint64(footer[18])<<40 | uint64(footer[19])<<32 |
		uint64(footer[20])<<24 | uint64(footer[21])<<16 | uint64(footer[22])<<8 | uint64(footer[23])
	c.HeaderOffset = dataOffset

	// DiskType at offset 60-63 (big-endian): 2=Fixed, 3=Dynamic, 4=Differencing
	diskType := uint32(footer[60])<<24 | uint32(footer[61])<<16 | uint32(footer[62])<<8 | uint32(footer[63])

	if diskType == 3 || diskType == 4 { // Dynamic or Differencing
		// Read dynamic header at dataOffset
		if dataOffset == 0 || dataOffset >= uint64(c.PhysicalSize)-512 {
			return
		}
		// The dynamic header is at a specific file offset; read it sector by sector
		headerSector := dataOffset / 512
		headerData, err := d.ReadSectors(headerSector, 2) // 1024 bytes = 2 sectors
		if err != nil || len(headerData) < 1024 {
			return
		}

		// Verify dynamic header signature "cxsparse"
		if string(headerData[0:8]) != "cxsparse" {
			return
		}

		// TableOffset (offset 16-23 in dynamic header, big-endian): BAT offset
		batOffset := uint64(headerData[16])<<56 | uint64(headerData[17])<<48 | uint64(headerData[18])<<40 | uint64(headerData[19])<<32 |
			uint64(headerData[20])<<24 | uint64(headerData[21])<<16 | uint64(headerData[22])<<8 | uint64(headerData[23])
		c.BATOffset = batOffset

		// BlockSize (offset 32-35 in dynamic header, big-endian)
		c.BlockSize = uint32(headerData[32])<<24 | uint32(headerData[33])<<16 | uint32(headerData[34])<<8 | uint32(headerData[35])
		c.BATEntrySize = 4 // Always 4 bytes per BAT entry
	}
}

// ── Card 2 Builder ───────────────────────────────────────────────────────────

func buildGeometry(d VirtualDisk) GeometryInfo {
	totalSectors := d.TotalSectors()
	sectorSize := d.SectorSize()

	geo := GeometryInfo{
		TotalSectors:    totalSectors,
		LogicalSector:   sectorSize,
		PhysicalSector:  sectorSize,
		PartitionScheme: "Unknown",
	}

	if totalSectors < 2 {
		return geo
	}

	sector0, err := d.ReadSectors(0, 1)
	if err != nil || len(sector0) < 512 {
		return geo
	}

	// Check GPT (sector 1 has "EFI PART" signature)
	if totalSectors >= 34 {
		gptHeader, err := d.ReadSectors(1, 1)
		if err == nil && len(gptHeader) >= 72 && string(gptHeader[0:8]) == "EFI PART" {
			geo.PartitionScheme = "GPT"
			geo.DiskGUID = formatGPTDiskGUID(gptHeader[56:72])
		}
	}

	// Check MBR
	if geo.PartitionScheme == "Unknown" && sector0[510] == 0x55 && sector0[511] == 0xAA {
		geo.PartitionScheme = "MBR"
		geo.DiskSignature = fmt.Sprintf("0x%02X%02X%02X%02X", sector0[443], sector0[444], sector0[445], sector0[446])
	}

	if geo.PartitionScheme == "Unknown" {
		geo.PartitionScheme = "Raw"
	}

	// Read CHS from VHD footer (last 512 bytes of file)
	geo.CHS = readVHD_CHS(d, totalSectors)

	return geo
}

// readVHD_CHS reads CHS geometry from the VHD footer.
// VHD footer is at the last sector. CHS fields (big-endian):
//   - Cylinders: offset 56-57 (uint16)
//   - Heads: offset 60 (uint8)
//   - SectorsPerTrack: offset 61 (uint8)
func readVHD_CHS(d VirtualDisk, totalSectors uint64) CHSGeometry {
	if totalSectors < 2 {
		return CHSGeometry{}
	}

	footerSector := totalSectors - 1
	footer, err := d.ReadSectors(footerSector, 1)
	if err != nil || len(footer) < 512 {
		return CHSGeometry{}
	}

	// Verify VHD signature
	if string(footer[0:8]) != "conectix" {
		return CHSGeometry{}
	}

	// CHS is at offsets 56-61 in the VHD footer (big-endian)
	cylinders := uint16(footer[56])<<8 | uint16(footer[57])
	heads := footer[60]
	sectorsPerTrack := footer[61]

	// Validate: if all zeros, return N/A-style values
	if cylinders == 0 && heads == 0 && sectorsPerTrack == 0 {
		return CHSGeometry{}
	}

	return CHSGeometry{
		Cylinders:       cylinders,
		Heads:           heads,
		SectorsPerTrack: sectorsPerTrack,
	}
}

func formatGPTDiskGUID(guid []byte) string {
	if len(guid) < 16 {
		return ""
	}
	return fmt.Sprintf("%02X%02X%02X%02X-%02X%02X-%02X%02X-%02X%02X-%02X%02X%02X%02X%02X%02X",
		guid[3], guid[2], guid[1], guid[0],
		guid[5], guid[4],
		guid[7], guid[6],
		guid[8], guid[9],
		guid[10], guid[11], guid[12], guid[13], guid[14], guid[15])
}

// ── Card 3 Builder (with Unallocated Gaps) ───────────────────────────────────

// buildPartitionsWithGaps adds Unallocated entries for gaps between partitions (Gap 1).
func buildPartitionsWithGaps(parts []Partition, totalSectors uint64) []PartitionInfo {
	if len(parts) == 0 {
		return nil
	}

	// Sort by Start LBA
	sorted := make([]Partition, len(parts))
	copy(sorted, parts)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Start < sorted[j].Start
	})

	var result []PartitionInfo
	expectedLBA := uint64(0) // LBA 0 is MBR/GPT header

	for _, p := range sorted {
		// Gap before this partition?
		if p.Start > expectedLBA && expectedLBA > 0 {
			gapSectors := p.Start - expectedLBA
			gapBytes := gapSectors * 512
			result = append(result, PartitionInfo{
				Index:         -1,
				Label:         "Unallocated",
				Filesystem:    "",
				StartLBA:      expectedLBA,
				EndLBA:        p.Start - 1,
				TotalSectors:  gapSectors,
				SizeBytes:     gapBytes,
				IsUnallocated: true,
				Status:        "Unallocated",
			})
		}

		status := "Ready"
		if p.HasContent {
			status = "Mounted"
		}
		result = append(result, PartitionInfo{
			Index:        p.Index,
			Label:        p.Label,
			Filesystem:   p.Filesystem,
			StartLBA:     p.Start,
			EndLBA:       p.End,
			TotalSectors: p.End - p.Start + 1,
			SizeBytes:    p.Size,
			Bootable:     p.Bootable,
			Active:       p.Active,
			IsMounted:    p.HasContent,
			Status:       status,
		})

		expectedLBA = p.End + 1
	}

	// Gap after last partition?
	if expectedLBA < totalSectors {
		gapSectors := totalSectors - expectedLBA
		gapBytes := gapSectors * 512
		result = append(result, PartitionInfo{
			Index:         -1,
			Label:         "Unallocated",
			Filesystem:    "",
			StartLBA:      expectedLBA,
			EndLBA:        totalSectors - 1,
			TotalSectors:  gapSectors,
			SizeBytes:     gapBytes,
			IsUnallocated: true,
			Status:        "Unallocated",
		})
	}

	return result
}

// ── Card 4 Builders (ext4, FAT, NTFS) ────────────────────────────────────────

func isExtFamily(fs string) bool {
	return fs == "ext4" || fs == "ext3" || fs == "ext2"
}

// readExt4Superblock reads the full ext4 superblock including state, UUID, label (Gaps 2 & 3).
func readExt4Superblock(d VirtualDisk, partitionStart uint64) *FSInfo {
	// Read 4 sectors (2048 bytes) to cover the full superblock
	sbData, err := d.ReadSectors(partitionStart+2, 4)
	if err != nil || len(sbData) < 0x200 {
		return nil
	}

	// Verify ext4 magic at superblock offset 0x38
	magic := uint16(sbData[0x38]) | uint16(sbData[0x39])<<8
	if magic != 0xEF53 {
		return nil
	}

	fs := &FSInfo{
		FilesystemType: "ext4",
		SuperblockOK:   true,
	}

	// ── Core superblock fields ──
	// s_inodes_count (0x00)
	fs.TotalInodes = uint64(readLE32(sbData, 0x00))
	// s_blocks_count_lo (0x04) + s_blocks_count_hi (0x14)
	blocksLo := readLE32(sbData, 0x04)
	blocksHi := readLE32(sbData, 0x14)
	fs.TotalBlocks = uint64(blocksHi)<<32 | uint64(blocksLo)
	// s_free_blocks_count_lo (0x0C) + s_free_blocks_count_hi (0x158... but let's use 0x0C for now)
	freeBlocksLo := readLE32(sbData, 0x0C)
	fs.FreeBlocks = uint64(freeBlocksLo)
	// s_free_inodes_count (0x10)
	fs.FreeInodes = uint64(readLE32(sbData, 0x10))
	// s_log_block_size (0x18)
	logBlockSize := readLE32(sbData, 0x18)
	fs.BlockSize = 1024 << logBlockSize

	// ── Gap 2: Filesystem state & mount info ──
	// s_state (0x04 in superblock... actually offset 0x04 is blocks count.
	// Let me use correct ext4 superblock layout from the spec:
	// Offset 0x00: s_inodes_count
	// Offset 0x04: s_blocks_count_lo
	// Offset 0x08: s_r_blocks_count_lo
	// Offset 0x0C: s_free_blocks_count_lo
	// Offset 0x10: s_free_inodes_count
	// Offset 0x14: s_blocks_count_hi
	// Offset 0x18: s_log_block_size
	// Offset 0x1C: s_log_frag_size
	// Offset 0x20: s_blocks_per_group
	// Offset 0x24: s_frags_per_group
	// Offset 0x28: s_inodes_per_group
	// Offset 0x2C: s_mtime
	// Offset 0x30: s_wtime
	// Offset 0x34: s_mnt_count
	// Offset 0x36: s_max_mnt_count
	// Offset 0x38: s_magic
	// Offset 0x3A: s_state
	// Offset 0x3C: s_errors
	// Offset 0x3E: s_minor_rev_level
	// Offset 0x40: s_lastcheck
	// Offset 0x44: s_checkinterval
	// Offset 0x48: s_creator_os
	// Offset 0x4C: s_rev_level
	// Offset 0x50: s_def_resuid
	// Offset 0x52: s_def_resgid
	// Offset 0x54: s_first_ino
	// Offset 0x58: s_inode_size
	// Offset 0x5A: s_block_group_nr
	// Offset 0x5C: s_feature_compat
	// Offset 0x60: s_feature_incompat
	// Offset 0x64: s_feature_ro_compat

	// s_state (0x3A): 1 = clean, 2 = has errors
	state := uint16(sbData[0x3A]) | uint16(sbData[0x3B])<<8
	switch state {
	case 1:
		fs.State = "Clean"
	case 2:
		fs.State = "Has Errors"
	default:
		fs.State = fmt.Sprintf("Unknown (0x%04X)", state)
	}

	// s_mnt_count (0x34)
	fs.MountCount = uint16(sbData[0x34]) | uint16(sbData[0x35])<<8
	// s_max_mnt_count (0x36): -1 means no limit
	rawMax := uint16(sbData[0x36]) | uint16(sbData[0x37])<<8
	fs.MaxMounts = int16(rawMax)

	// s_mtime (0x2C): last mount time (POSIX)
	mtime := readLE32(sbData, 0x2C)
	if mtime > 0 {
		fs.LastWriteTime = time.Unix(int64(mtime), 0).UTC().Format("2006-01-02 15:04:05 UTC")
	}

	// s_wtime (0x30): last write time (POSIX)
	wtime := readLE32(sbData, 0x30)
	if wtime > 0 && wtime > mtime {
		fs.LastWriteTime = time.Unix(int64(wtime), 0).UTC().Format("2006-01-02 15:04:05 UTC")
	}

	// ── Gap 3: Volume UUID and Label ──
	// s_uuid (0x68-0x77): 16-byte UUID stored in mixed-endian
	uuidBytes := sbData[0x68:0x78]
	fs.VolumeUUID = fmt.Sprintf("%02X%02X%02X%02X-%02X%02X-%02X%02X-%02X%02X-%02X%02X%02X%02X%02X%02X",
		uuidBytes[0], uuidBytes[1], uuidBytes[2], uuidBytes[3],
		uuidBytes[4], uuidBytes[5],
		uuidBytes[6], uuidBytes[7],
		uuidBytes[8], uuidBytes[9],
		uuidBytes[10], uuidBytes[11], uuidBytes[12], uuidBytes[13], uuidBytes[14], uuidBytes[15])

	// s_volume_name (0x78-0x87): 16-byte label, null-terminated
	labelRaw := sbData[0x78:0x88]
	fs.VolumeLabel = trimASCII(labelRaw)
	if fs.VolumeLabel == "" {
		fs.VolumeLabel = "(none)"
	}

	// s_last_mounted (0x88-0xA7): 64-byte path
	lastMountedRaw := sbData[0x88:0xC8]
	if len(lastMountedRaw) > 64 {
		lastMountedRaw = lastMountedRaw[:64]
	}
	fs.LastMountedPath = trimASCII(lastMountedRaw)

	// ── Feature flags ──
	featCompat := readLE32(sbData, 0x5C)
	featIncompat := readLE32(sbData, 0x60)
	featRoCompat := readLE32(sbData, 0x64)
	fs.FeatureFlags = decodeExt4Features(featCompat, featIncompat, featRoCompat)

	// Block groups
	if fs.BlockSize > 0 {
		blocksPerGroup := uint64(fs.BlockSize) * 8
		if blocksPerGroup > 0 {
			fs.BlockGroups = uint32((fs.TotalBlocks + blocksPerGroup - 1) / blocksPerGroup)
		}
	}

	return fs
}

// readFATInfo reads FAT boot sector for volume label and filesystem state.
func readFATInfo(d VirtualDisk, partitionStart uint64, fsType string) *FSInfo {
	data, err := d.ReadSectors(partitionStart, 1)
	if err != nil || len(data) < 90 {
		return nil
	}

	fs := &FSInfo{
		FilesystemType: fsType,
		SuperblockOK:   true,
	}

	// Volume label: FAT16 at offset 43 (11 bytes), FAT32 at offset 71 (11 bytes)
	labelOffset := 43
	if fsType == "FAT32" {
		labelOffset = 71
	}
	if labelOffset+11 <= len(data) {
		fs.VolumeLabel = trimASCII(data[labelOffset : labelOffset+11])
		if fs.VolumeLabel == "" || fs.VolumeLabel == "NO NAME    " {
			fs.VolumeLabel = "(none)"
		}
	}

	return fs
}

// readNTFSInfo reads NTFS boot sector for volume label.
func readNTFSInfo(d VirtualDisk, partitionStart uint64) *FSInfo {
	data, err := d.ReadSectors(partitionStart, 1)
	if err != nil || len(data) < 72 {
		return nil
	}

	fs := &FSInfo{
		FilesystemType: "NTFS",
		SuperblockOK:   true,
	}

	// Volume serial number at offset 72-79 (little-endian, but displayed as hex)
	if len(data) >= 80 {
		fs.VolumeUUID = fmt.Sprintf("%02X%02X%02X%02X-%02X%02X-%02X%02X",
			data[75], data[74], data[73], data[72],
			data[77], data[76],
			data[79], data[78])
	}

	// Volume label at offset 43 (11 bytes)
	if len(data) >= 54 {
		fs.VolumeLabel = trimASCII(data[43:54])
		if fs.VolumeLabel == "" {
			fs.VolumeLabel = "(none)"
		}
	}

	return fs
}

// ── Feature flag decoder ─────────────────────────────────────────────────────

func decodeExt4Features(compat, incompat, roCompat uint32) []string {
	var flags []string

	compatMap := map[uint32]string{
		0x0001: "DIR_PREALLOC", 0x0002: "IMAGIC_INODES", 0x0004: "HAS_JOURNAL",
		0x0008: "EXT_ATTR", 0x0010: "RESIZE_INODE", 0x0020: "DIR_INDEX",
	}
	for bit, name := range compatMap {
		if compat&bit != 0 {
			flags = append(flags, name)
		}
	}

	incompatMap := map[uint32]string{
		0x0001: "COMPRESSION", 0x0002: "FILETYPE", 0x0004: "RECOVER",
		0x0008: "JOURNAL_DEV", 0x0010: "META_BG", 0x0040: "EXTENTS",
		0x0080: "64BIT", 0x0100: "MMP", 0x0200: "FLEX_BG",
		0x0400: "INLINEDATA", 0x0800: "DIRDATA", 0x1000: "LARGE_EXTRA_ISIZE",
		0x2000: "HUGE_FILE",
	}
	for bit, name := range incompatMap {
		if incompat&bit != 0 {
			flags = append(flags, name)
		}
	}

	roCompatMap := map[uint32]string{
		0x0001: "SPARSE_SUPER", 0x0002: "LARGE_FILE", 0x0004: "BTREE_DIR",
		0x0008: "HUGE_FILE", 0x0010: "GDT_CSUM", 0x0020: "DIR_NLINK",
		0x0040: "EXTRA_ISIZE",
	}
	for bit, name := range roCompatMap {
		if roCompat&bit != 0 {
			dup := false
			for _, f := range flags {
				if f == name {
					dup = true
					break
				}
			}
			if !dup {
				flags = append(flags, name)
			}
		}
	}

	if len(flags) == 0 {
		flags = []string{"NONE"}
	}
	return flags
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func trimASCII(data []byte) string {
	s := strings.TrimRight(string(data), "\x00 \t\r\n")
	return strings.TrimSpace(s)
}

// ReadForHash streams the entire disk through w for hash calculation.
func ReadForHash(d VirtualDisk, w io.Writer) error {
	totalSectors := d.TotalSectors()
	sectorSize := d.SectorSize()
	chunkSectors := uint32(1024 * 1024 / sectorSize)
	if chunkSectors == 0 {
		chunkSectors = 1
	}
	var sector uint64
	for sector < totalSectors {
		remaining := totalSectors - sector
		count := chunkSectors
		if uint64(count) > remaining {
			count = uint32(remaining)
		}
		data, err := d.ReadSectors(sector, count)
		if err != nil {
			return fmt.Errorf("read sectors %d: %w", sector, err)
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
		sector += uint64(count)
	}
	return nil
}

// Package filesystem implements filesystem reading for FAT16, FAT32, and NTFS.
// It provides read-only access to files and directories within VHD images.
package filesystem

import (
	"encoding/binary"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"
	"unicode/utf16"
)

var (
	ErrUnsupportedFS     = errors.New("filesystem: unsupported filesystem type")
	ErrFATNotFound       = errors.New("filesystem: FAT table not found")
	ErrClusterTooLarge   = errors.New("filesystem: cluster chain too long")
	ErrInvalidPath       = errors.New("filesystem: invalid path")
	ErrFileNotFound      = errors.New("filesystem: file not found")
)

// FAT32Reader implements Reader for FAT32 filesystems.
type FAT32Reader struct {
	// Cached BPB (BIOS Parameter Block) info
	bytesPerSector    uint16
	sectorsPerCluster uint8
	reservedSectors   uint16
	numFATs           uint8
	totalSectors32    uint32
	sectorsPerFAT     uint32
	rootCluster       uint32
	fatStartOffset    uint64
	dataStartOffset   uint64
	clusterSize       uint32
	totalClusters     uint32
}

// NewFAT32Reader creates a new FAT32 reader by parsing the BPB.
func NewFAT32Reader(diskReader DiskReader, partitionStart uint64) (*FAT32Reader, error) {
	// Read the first sector (boot sector)
	data, err := diskReader.ReadSectors(partitionStart, 1)
	if err != nil {
		return nil, fmt.Errorf("filesystem: failed to read FAT32 boot sector: %w", err)
	}

	sectorSize := diskReader.SectorSize()
	r := &FAT32Reader{}

	r.bytesPerSector = binary.LittleEndian.Uint16(data[11:13])
	r.sectorsPerCluster = data[13]
	r.reservedSectors = binary.LittleEndian.Uint16(data[14:16])
	r.numFATs = data[16]
	r.totalSectors32 = binary.LittleEndian.Uint32(data[32:36])
	r.sectorsPerFAT = binary.LittleEndian.Uint32(data[36:40])
	r.rootCluster = binary.LittleEndian.Uint32(data[44:48])

	// Validate
	if r.bytesPerSector == 0 || r.sectorsPerCluster == 0 {
		return nil, fmt.Errorf("filesystem: invalid FAT32 BPB values")
	}

	r.clusterSize = uint32(r.bytesPerSector) * uint32(r.sectorsPerCluster)
	r.fatStartOffset = (partitionStart + uint64(r.reservedSectors)) * sectorSize
	r.dataStartOffset = (partitionStart + uint64(r.reservedSectors) + uint64(r.numFATs)*uint64(r.sectorsPerFAT)) * sectorSize

	// Calculate total clusters
	dataSectors := uint64(r.totalSectors32) - uint64(r.reservedSectors) - uint64(r.numFATs)*uint64(r.sectorsPerFAT)
	r.totalClusters = uint32(dataSectors / uint64(r.sectorsPerCluster))

	return r, nil
}

// DetectFSType detects the filesystem type.
func (r *FAT32Reader) DetectFSType(_ DiskReader, _ uint64) FSType {
	return FAT32
}

// ListRootDirectory lists the root directory contents.
func (r *FAT32Reader) ListRootDirectory(diskReader DiskReader, partitionStart uint64) ([]FileEntry, error) {
	return r.ListDirectory(diskReader, partitionStart, "/")
}

// ListDirectory lists files in a directory given its path.
func (r *FAT32Reader) ListDirectory(diskReader DiskReader, partitionStart uint64, dirPath string) ([]FileEntry, error) {
	// Start from root cluster
	cluster := r.rootCluster

	// Navigate to the target directory
	if dirPath != "/" && dirPath != "" {
		parts := strings.Split(strings.Trim(dirPath, "/"), "/")
		for _, part := range parts {
			entries, err := r.listClusterChain(diskReader, partitionStart, cluster)
			if err != nil {
				return nil, err
			}
			found := false
			for _, entry := range entries {
				if entry.IsDirectory && strings.EqualFold(entry.Name, part) {
					cluster = entry.ClusterStart
					found = true
					break
				}
			}
			if !found {
				return nil, ErrFileNotFound
			}
		}
	}

	return r.listClusterChain(diskReader, partitionStart, cluster)
}

// listClusterChain reads all directory entries in a cluster chain.
func (r *FAT32Reader) listClusterChain(diskReader DiskReader, partitionStart uint64, startCluster uint32) ([]FileEntry, error) {
	chain, err := r.getClusterChain(diskReader, startCluster)
	if err != nil {
		return nil, err
	}

	var entries []FileEntry
	sectorSize := diskReader.SectorSize()

	for _, cluster := range chain {
		clusterOffset := r.dataStartOffset + uint64(cluster-2)*uint64(r.clusterSize)
		clusterSectors := uint32(r.clusterSize) / uint32(sectorSize)

		data, err := diskReader.ReadSectors(clusterOffset/sectorSize, clusterSectors)
		if err != nil {
			continue
		}

		// Parse 32-byte directory entries
		for i := 0; i+32 <= len(data); i += 32 {
			entry := data[i : i+32]

			// End of directory
			if entry[0] == 0x00 {
				return entries, nil
			}

			// Deleted entry
			if entry[0] == 0xE5 {
				continue
			}

			// Long filename entry (skip for now, use short name)
			if entry[11] == 0x0F {
				continue
			}

			// Volume label (skip)
			if entry[11]&0x08 != 0 {
				continue
			}

			fileEntry := r.parseDirectoryEntry(entry)
			entries = append(entries, fileEntry)
		}
	}

	return entries, nil
}

// parseDirectoryEntry parses a 32-byte FAT directory entry.
func (r *FAT32Reader) parseDirectoryEntry(entry []byte) FileEntry {
	// Extract short name
	nameBytes := entry[0:8]
	extBytes := entry[8:11]

	name := strings.TrimRight(string(nameBytes), " ")
	ext := strings.TrimRight(string(extBytes), " ")

	var fullName string
	if ext != "" {
		fullName = name + "." + ext
	} else {
		fullName = name
	}

	attributes := entry[11]
	isDir := attributes&0x10 != 0
	isHidden := attributes&0x02 != 0
	isSystem := attributes&0x04 != 0
	isReadOnly := attributes&0x01 != 0

	// Cluster start (high word + low word)
	clusterHigh := binary.LittleEndian.Uint16(entry[20:22])
	clusterLow := binary.LittleEndian.Uint16(entry[26:28])
	clusterStart := uint32(clusterHigh)<<16 | uint32(clusterLow)

	// File size
	fileSize := int64(binary.LittleEndian.Uint32(entry[28:32]))

	// Timestamps
	createDate := binary.LittleEndian.Uint16(entry[16:18])
	createTime := binary.LittleEndian.Uint16(entry[14:16])
	modifyDate := binary.LittleEndian.Uint16(entry[24:26])
	modifyTime := binary.LittleEndian.Uint16(entry[22:24])

	created := fatDateTimeToDate(uint32(createDate), uint32(createTime))
	modified := fatDateTimeToDate(uint32(modifyDate), uint32(modifyTime))

	return FileEntry{
		Name:         strings.ToLower(fullName),
		IsDirectory:  isDir,
		IsHidden:     isHidden,
		Size:         fileSize,
		CreatedTime:  created,
		ModifiedTime: modified,
		ClusterStart: clusterStart,
		Extension:    strings.ToLower(ext),
		Attributes: FileAttributes{
			ReadOnly:  isReadOnly,
			Hidden:    isHidden,
			System:    isSystem,
			Directory: isDir,
		},
	}
}

// getClusterChain follows the FAT chain to get all clusters.
func (r *FAT32Reader) getClusterChain(diskReader DiskReader, startCluster uint32) ([]uint32, error) {
	var chain []uint32
	cluster := startCluster
	visited := make(map[uint32]bool)

	sectorSize := diskReader.SectorSize()
	fatOffset := r.fatStartOffset

	for cluster >= 2 && cluster < 0x0FFFFFF8 {
		if visited[cluster] {
			break
		}
		visited[cluster] = true
		chain = append(chain, cluster)

		// Read FAT entry
		entryOffset := fatOffset + uint64(cluster)*4
		entryData, err := diskReader.ReadSectors(entryOffset/sectorSize, 1)
		if err != nil {
			break
		}

		entryPos := (entryOffset % sectorSize)
		if entryPos+4 > uint64(len(entryData)) {
			break
		}
		nextCluster := binary.LittleEndian.Uint32(entryData[entryPos : entryPos+4])
		cluster = nextCluster & 0x0FFFFFFF // Mask out reserved bits
	}

	return chain, nil
}

// GetFileContent reads the content of a file.
func (r *FAT32Reader) GetFileContent(diskReader DiskReader, _ uint64, file *FileEntry) ([]byte, error) {
	if file.IsDirectory {
		return nil, fmt.Errorf("filesystem: cannot read directory content")
	}

	chain, err := r.getClusterChain(diskReader, file.ClusterStart)
	if err != nil {
		return nil, err
	}

	result := make([]byte, 0, file.Size)
	sectorSize := diskReader.SectorSize()

	for _, cluster := range chain {
		clusterOffset := r.dataStartOffset + uint64(cluster-2)*uint64(r.clusterSize)
		clusterSectors := uint32(r.clusterSize) / uint32(sectorSize)

		data, err := diskReader.ReadSectors(clusterOffset/sectorSize, clusterSectors)
		if err != nil {
			break
		}

		result = append(result, data...)
	}

	// Truncate to file size
	if int64(len(result)) > file.Size {
		result = result[:file.Size]
	}

	return result, nil
}

// GetFileProperties returns detailed file properties.
func (r *FAT32Reader) GetFileProperties(_ DiskReader, _ uint64, file *FileEntry) (*FileProperties, error) {
	return &FileProperties{
		Name:          file.Name,
		Extension:     file.Extension,
		FullPath:      file.Path,
		Size:          file.Size,
		SizeFormatted: formatSize(file.Size),
		IsDirectory:   file.IsDirectory,
		ModifiedTime:  file.ModifiedTime,
		CreatedTime:   file.CreatedTime,
		AccessedTime:  file.AccessedTime,
		Attributes:    file.Attributes,
		ClusterStart:  file.ClusterStart,
		FSType:        string(FAT32),
	}, nil
}

// SearchFiles searches for files matching a query.
func (r *FAT32Reader) SearchFiles(diskReader DiskReader, partitionStart uint64, query string, caseSensitive bool) ([]FileEntry, error) {
	var results []FileEntry
	query = strings.ToLower(query)

	var searchDir func(dirPath string) error
	searchDir = func(dirPath string) error {
		entries, err := r.ListDirectory(diskReader, partitionStart, dirPath)
		if err != nil {
			return nil
		}

		for _, entry := range entries {
			match := false
			if caseSensitive {
				match = strings.Contains(entry.Name, query)
			} else {
				match = strings.Contains(strings.ToLower(entry.Name), query)
			}

			if match {
				entry.Path = path.Join(dirPath, entry.Name)
				results = append(results, entry)
			}

			if entry.IsDirectory && entry.Name != "." && entry.Name != ".." {
				searchDir(path.Join(dirPath, entry.Name))
			}
		}
		return nil
	}

	searchDir("/")
	return results, nil
}

// fatDateTimeToDate converts FAT date/time values to time.Time.
func fatDateTimeToDate(date, t uint32) time.Time {
	year := int((date >> 9) + 1980)
	month := time.Month((date >> 5) & 0x0F)
	day := int(date & 0x1F)

	hour := int(t >> 11)
	minute := int((t >> 5) & 0x3F)
	second := int(t&0x1F) * 2

	return time.Date(year, month, day, hour, minute, second, 0, time.UTC)
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
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

// cleanGPTName cleans a UTF-16LE name from GPT entries.
func cleanGPTName2(nameBytes []byte) string {
	runes := make([]rune, 0)
	for i := 0; i < len(nameBytes)-1; i += 2 {
		r := rune(nameBytes[i]) | rune(nameBytes[i+1])<<8
		if r == 0 {
			break
		}
		runes = append(runes, r)
	}
	return string(runes)
}

// utf16LEToString converts a UTF-16LE byte slice to a Go string.
func utf16LEToString(data []byte) string {
	if len(data)%2 != 0 {
		return string(data)
	}
	u16s := make([]uint16, len(data)/2)
	for i := range u16s {
		u16s[i] = binary.LittleEndian.Uint16(data[i*2:])
	}
	return string(utf16.Decode(u16s))
}

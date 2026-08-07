package filesystems

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

// exFATReader implements Reader for exFAT filesystems.
type exFATReader struct {
	bytesPerSector    uint16
	sectorsPerCluster uint8
	totalSectors      uint64
	mediaDescriptor   uint8
	fatStartSector    uint64
	fatSize           uint32
	clusterCount      uint32
	rootDirCluster    uint32
	volumeLength      uint64
	bytesPerCluster   uint32
}

func newExFATReader(diskReader DiskReader, partitionStart uint64) (*exFATReader, error) {
	// Read boot sector
	bootSector, err := diskReader.ReadSectors(partitionStart, 1)
	if err != nil {
		return nil, fmt.Errorf("exFAT: failed to read boot sector: %w", err)
	}

	// Verify exFAT signature at offset 3
	if len(bootSector) < 512 {
		return nil, fmt.Errorf("exFAT: boot sector too small")
	}
	// exFAT signature: "EXFAT   " at offset 3
	sig := string(bootSector[3:11])
	if sig != "EXFAT   " {
		return nil, fmt.Errorf("exFAT: invalid signature: %s", sig)
	}

	r := &exFATReader{
		bytesPerSector:    binary.LittleEndian.Uint16(bootSector[11:13]),
		sectorsPerCluster: bootSector[108],
		mediaDescriptor:   bootSector[10],
		fatStartSector:    partitionStart + uint64(binary.LittleEndian.Uint32(bootSector[84:88])),
		fatSize:           binary.LittleEndian.Uint32(bootSector[80:84]),
		clusterCount:      binary.LittleEndian.Uint32(bootSector[92:96]),
		rootDirCluster:    binary.LittleEndian.Uint32(bootSector[96:100]),
		volumeLength:      binary.LittleEndian.Uint64(bootSector[72:80]),
	}

	r.bytesPerCluster = uint32(r.bytesPerSector) * uint32(r.sectorsPerCluster)

	return r, nil
}

func (r *exFATReader) DetectFSType(diskReader DiskReader, partitionStart uint64) FSType {
	return exFAT
}

func (r *exFATReader) ListRootDirectory(diskReader DiskReader, partitionStart uint64) ([]FileEntry, error) {
	return r.ListDirectory(diskReader, partitionStart, "/")
}

func (r *exFATReader) ListDirectory(diskReader DiskReader, partitionStart uint64, dirPath string) ([]FileEntry, error) {
	// Get cluster chain for directory
	clusters, err := r.getClusterChain(diskReader, partitionStart, r.rootDirCluster)
	if err != nil {
		return nil, fmt.Errorf("exFAT: failed to read root directory clusters: %w", err)
	}

	var entries []FileEntry
	for _, cluster := range clusters {
		// Read cluster data
		offset := partitionStart + uint64(cluster-2)*uint64(r.sectorsPerCluster)
		data, err := diskReader.ReadSectors(offset, uint32(r.sectorsPerCluster))
		if err != nil {
			continue
		}

		// Parse directory entries (32 bytes each)
		for i := 0; i+31 < len(data); i += 32 {
			entryType := data[i]
			if entryType == 0x00 {
				// End of directory
				break
			}
			if entryType&0x80 != 0 {
				// Inactive entry, skip
				continue
			}

			// File entry (0x85)
			if entryType == 0x85 {
				entry, err := r.parseFileEntry(data[i:])
				if err == nil && entry != nil {
					entries = append(entries, *entry)
				}
			}
		}
	}

	return entries, nil
}

func (r *exFATReader) parseFileEntry(data []byte) (*FileEntry, error) {
	if len(data) < 32 {
		return nil, fmt.Errorf("exFAT: entry too small")
	}

	// Secondary count (number of additional entries)
	secondaryCount := data[1]

	// File attributes
	attributes := data[4]
	isDir := attributes&0x10 != 0

	// First cluster
	firstCluster := binary.LittleEndian.Uint32(data[20:24])

	// Data length
	dataLength := binary.LittleEndian.Uint64(data[24:32])

	// Parse stream entries for filename
	var name string
	for i := 1; i <= int(secondaryCount) && (i*32) < len(data); i++ {
		streamEntry := data[i*32:]
		if len(streamEntry) < 32 {
			break
		}

		entryType := streamEntry[0]
		if entryType == 0xC1 {
			// File name entry
			nameLen := int(streamEntry[3])
			nameBytes := streamEntry[2 : 2+nameLen*2]
			name = decodeUTF16LE(nameBytes)
			break
		}
	}

	if name == "" {
		return nil, fmt.Errorf("exFAT: no filename found")
	}

	return &FileEntry{
		Name:        name,
		IsDirectory: isDir,
		Size:        int64(dataLength),
		ClusterStart: firstCluster,
	}, nil
}

func (r *exFATReader) getClusterChain(diskReader DiskReader, partitionStart uint64, startCluster uint32) ([]uint32, error) {
	var chain []uint32
	cluster := startCluster

	// Read FAT
	fatData, err := diskReader.ReadSectors(r.fatStartSector, uint32(r.fatSize))
	if err != nil {
		return nil, err
	}

	visited := make(map[uint32]bool)
	for cluster >= 2 && cluster < r.clusterCount {
		if visited[cluster] {
			break
		}
		visited[cluster] = true
		chain = append(chain, cluster)

		// Read next cluster from FAT
		offset := int(cluster) * 4
		if offset+4 > len(fatData) {
			break
		}
		cluster = binary.LittleEndian.Uint32(fatData[offset : offset+4])
	}

	return chain, nil
}

func (r *exFATReader) GetFileContent(diskReader DiskReader, partitionStart uint64, file *FileEntry) ([]byte, error) {
	if file.ClusterStart == 0 {
		return nil, fmt.Errorf("exFAT: invalid cluster")
	}

	clusters, err := r.getClusterChain(diskReader, partitionStart, file.ClusterStart)
	if err != nil {
		return nil, err
	}

	var content []byte
	for _, cluster := range clusters {
		offset := partitionStart + uint64(cluster-2)*uint64(r.sectorsPerCluster)
		data, err := diskReader.ReadSectors(offset, uint32(r.sectorsPerCluster))
		if err != nil {
			continue
		}
		content = append(content, data...)
	}

	// Trim to file size
	if int64(len(content)) > file.Size {
		content = content[:file.Size]
	}

	return content, nil
}

func (r *exFATReader) GetFileProperties(diskReader DiskReader, partitionStart uint64, file *FileEntry) (*FileProperties, error) {
	return &FileProperties{
		Name:         file.Name,
		Extension:    getExtension(file.Name),
		FullPath:     file.Path,
		Size:         file.Size,
		SizeFormatted: formatSize(file.Size),
		IsDirectory:  file.IsDirectory,
		ModifiedTime: file.ModifiedTime,
		CreatedTime:  file.CreatedTime,
		AccessedTime: file.AccessedTime,
		Attributes:   file.Attributes,
		ClusterStart: file.ClusterStart,
		FSType:       string(exFAT),
	}, nil
}

func (r *exFATReader) SearchFiles(diskReader DiskReader, partitionStart uint64, query string, caseSensitive bool) ([]FileEntry, error) {
	var results []FileEntry
	q := strings.ToLower(query)

	var searchDir func(path string) error
	searchDir = func(path string) error {
		entries, err := r.ListDirectory(diskReader, partitionStart, path)
		if err != nil {
			return nil
		}

		for _, entry := range entries {
			name := entry.Name
			if !caseSensitive {
				name = strings.ToLower(name)
			}
			if strings.Contains(name, q) {
				results = append(results, entry)
			}
			if entry.IsDirectory {
				searchDir(path + "/" + entry.Name)
			}
		}
		return nil
	}

	searchDir("/")
	return results, nil
}

// decodeUTF16LE decodes a UTF-16LE byte slice to a Go string.
func decodeUTF16LE(data []byte) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	u16s := make([]uint16, len(data)/2)
	for i := range u16s {
		u16s[i] = binary.LittleEndian.Uint16(data[i*2:])
	}
	return string(runeSlice(u16s))
}

func runeSlice(u16s []uint16) []rune {
	runes := make([]rune, len(u16s))
	for i, u := range u16s {
		runes[i] = rune(u)
	}
	return runes
}

func getExtension(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return name[i+1:]
		}
	}
	return ""
}

// Timestamp conversion helpers
func fileTimeToTime(ft uint64) time.Time {
	if ft == 0 {
		return time.Time{}
	}
	// Windows FILETIME: 100-nanosecond intervals since January 1, 1601
	nsec := int64(ft) * 100
	return time.Date(1601, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(nsec))
}

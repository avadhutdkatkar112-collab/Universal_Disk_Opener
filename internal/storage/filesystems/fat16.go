package filesystems

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// FAT16Reader implements Reader for FAT16 filesystems.
type FAT16Reader struct {
	bytesPerSector    uint16
	sectorsPerCluster uint8
	reservedSectors   uint16
	numFATs           uint8
	totalSectors16    uint16
	totalSectors32    uint32
	sectorsPerFAT     uint16
	rootEntryCount    uint16
	rootDirSectors    uint16
	fatStartOffset    uint64
	dataStartOffset   uint64
	clusterSize       uint32
	totalClusters     uint32
}

// NewFAT16Reader creates a new FAT16 reader by parsing the BPB.
func NewFAT16Reader(diskReader DiskReader, partitionStart uint64) (*FAT16Reader, error) {
	data, err := diskReader.ReadSectors(partitionStart, 1)
	if err != nil {
		return nil, fmt.Errorf("filesystem: failed to read FAT16 boot sector: %w", err)
	}

	sectorSize := diskReader.SectorSize()
	r := &FAT16Reader{}

	r.bytesPerSector = binary.LittleEndian.Uint16(data[11:13])
	r.sectorsPerCluster = data[13]
	r.reservedSectors = binary.LittleEndian.Uint16(data[14:16])
	r.numFATs = data[16]
	r.rootEntryCount = binary.LittleEndian.Uint16(data[17:19])
	r.totalSectors16 = binary.LittleEndian.Uint16(data[19:21])
	r.sectorsPerFAT = binary.LittleEndian.Uint16(data[22:24])
	r.totalSectors32 = binary.LittleEndian.Uint32(data[32:36])

	if r.bytesPerSector == 0 || r.sectorsPerCluster == 0 {
		return nil, fmt.Errorf("filesystem: invalid FAT16 BPB values")
	}

	r.clusterSize = uint32(r.bytesPerSector) * uint32(r.sectorsPerCluster)
	r.rootDirSectors = (uint16(r.rootEntryCount)*32 + uint16(r.bytesPerSector) - 1) / uint16(r.bytesPerSector)

	fatStart := partitionStart + uint64(r.reservedSectors)
	r.fatStartOffset = fatStart * sectorSize
	r.dataStartOffset = (fatStart + uint64(r.numFATs)*uint64(r.sectorsPerFAT) + uint64(r.rootDirSectors)) * sectorSize

	totalSectors := uint64(r.totalSectors32)
	if totalSectors == 0 {
		totalSectors = uint64(r.totalSectors16)
	}

	dataSectors := totalSectors - uint64(r.reservedSectors) - uint64(r.numFATs)*uint64(r.sectorsPerFAT) - uint64(r.rootDirSectors)
	r.totalClusters = uint32(dataSectors / uint64(r.sectorsPerCluster))

	return r, nil
}

func (r *FAT16Reader) DetectFSType(_ DiskReader, _ uint64) FSType {
	return FAT16
}

func (r *FAT16Reader) ListRootDirectory(diskReader DiskReader, partitionStart uint64) ([]FileEntry, error) {
	return r.listRootDir(diskReader)
}

func (r *FAT16Reader) listRootDir(diskReader DiskReader) ([]FileEntry, error) {
	sectorSize := diskReader.SectorSize()
	rootDirOffset := r.fatStartOffset - uint64(r.rootDirSectors)*sectorSize

	totalBytes := uint32(r.rootEntryCount) * 32
	totalSectors := (totalBytes + uint32(sectorSize) - 1) / uint32(sectorSize)

	data, err := diskReader.ReadSectors(rootDirOffset/sectorSize, uint32(totalSectors))
	if err != nil {
		return nil, err
	}

	var entries []FileEntry
	for i := 0; i+32 <= len(data) && i/32 < int(r.rootEntryCount); i += 32 {
		entry := data[i : i+32]
		if entry[0] == 0x00 {
			break
		}
		if entry[0] == 0xE5 || entry[11] == 0x0F || entry[11]&0x08 != 0 {
			continue
		}

		fileEntry := r.parseDirectoryEntry(entry)
		entries = append(entries, fileEntry)
	}

	return entries, nil
}

func (r *FAT16Reader) ListDirectory(diskReader DiskReader, partitionStart uint64, dirPath string) ([]FileEntry, error) {
	if dirPath == "/" || dirPath == "" {
		return r.listRootDir(diskReader)
	}

	// For subdirectories, we need to navigate via FAT
	// First find the directory in root
	rootEntries, err := r.listRootDir(diskReader)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(strings.Trim(dirPath, "/"), "/")
	var cluster uint32

	for _, part := range parts {
		found := false
		for _, entry := range rootEntries {
			if entry.IsDirectory && strings.EqualFold(entry.Name, part) {
				cluster = entry.ClusterStart
				found = true
				break
			}
		}
		if !found {
			return nil, ErrFileNotFound
		}

		// List subdirectory entries
		rootEntries, err = r.listSubdirClusterChain(diskReader, cluster)
		if err != nil {
			return nil, err
		}
	}

	return rootEntries, nil
}

func (r *FAT16Reader) listSubdirClusterChain(diskReader DiskReader, startCluster uint32) ([]FileEntry, error) {
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

		for i := 0; i+32 <= len(data); i += 32 {
			entry := data[i : i+32]
			if entry[0] == 0x00 {
				return entries, nil
			}
			if entry[0] == 0xE5 || entry[11] == 0x0F || entry[11]&0x08 != 0 {
				continue
			}
			fileEntry := r.parseDirectoryEntry(entry)
			entries = append(entries, fileEntry)
		}
	}

	return entries, nil
}

func (r *FAT16Reader) parseDirectoryEntry(entry []byte) FileEntry {
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

	clusterHigh := binary.LittleEndian.Uint16(entry[20:22])
	clusterLow := binary.LittleEndian.Uint16(entry[26:28])
	clusterStart := uint32(clusterHigh)<<16 | uint32(clusterLow)

	fileSize := int64(binary.LittleEndian.Uint32(entry[28:32]))
	createDate := binary.LittleEndian.Uint16(entry[16:18])
	createTime := binary.LittleEndian.Uint16(entry[14:16])
	modifyDate := binary.LittleEndian.Uint16(entry[24:26])
	modifyTime := binary.LittleEndian.Uint16(entry[22:24])

	return FileEntry{
		Name:         strings.ToLower(fullName),
		IsDirectory:  isDir,
		IsHidden:     attributes&0x02 != 0,
		Size:         fileSize,
		CreatedTime:  fatDateTimeToDate(uint32(createDate), uint32(createTime)),
		ModifiedTime: fatDateTimeToDate(uint32(modifyDate), uint32(modifyTime)),
		ClusterStart: clusterStart,
		Extension:    strings.ToLower(ext),
		Attributes: FileAttributes{
			ReadOnly:  attributes&0x01 != 0,
			Hidden:    attributes&0x02 != 0,
			System:    attributes&0x04 != 0,
			Directory: isDir,
		},
	}
}

func (r *FAT16Reader) getClusterChain(diskReader DiskReader, startCluster uint32) ([]uint32, error) {
	var chain []uint32
	cluster := startCluster
	visited := make(map[uint32]bool)

	sectorSize := diskReader.SectorSize()

	for cluster >= 2 && cluster < 0xFFF8 {
		if visited[cluster] {
			break
		}
		visited[cluster] = true
		chain = append(chain, cluster)

		entryOffset := r.fatStartOffset + uint64(cluster)*2
		entryData, err := diskReader.ReadSectors(entryOffset/sectorSize, 1)
		if err != nil {
			break
		}

		entryPos := entryOffset % sectorSize
		if entryPos+2 > uint64(len(entryData)) {
			break
		}
		nextCluster := uint32(binary.LittleEndian.Uint16(entryData[entryPos : entryPos+2]))
		cluster = nextCluster
	}

	return chain, nil
}

func (r *FAT16Reader) GetFileContent(diskReader DiskReader, _ uint64, file *FileEntry) ([]byte, error) {
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

	if int64(len(result)) > file.Size {
		result = result[:file.Size]
	}

	return result, nil
}

func (r *FAT16Reader) GetFileProperties(_ DiskReader, _ uint64, file *FileEntry) (*FileProperties, error) {
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
		FSType:        string(FAT16),
	}, nil
}

func (r *FAT16Reader) SearchFiles(diskReader DiskReader, partitionStart uint64, query string, caseSensitive bool) ([]FileEntry, error) {
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
				entry.Path = dirPath + "/" + entry.Name
				results = append(results, entry)
			}
			if entry.IsDirectory {
				searchDir(entry.Path)
			}
		}
		return nil
	}

	searchDir("/")
	return results, nil
}

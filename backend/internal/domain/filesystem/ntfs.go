package filesystem

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// NTFSReader implements Reader for NTFS filesystems.
type NTFSReader struct {
	bytesPerSector    uint16
	sectorsPerCluster uint8
	mftStartCluster   uint64
	mftMirrorCluster  uint64
	clusterSize       uint32
	sectorSize        uint64
	partitionStart    uint64
}

// MFTEntry represents an NTFS Master File Table entry.
type MFTEntry struct {
	Signature       [4]byte // "FILE"
	OffsetSeq       uint16
	Sequence        uint16
	FirstAttrOffset uint16
	Flags           uint16 // 0x01 = file, 0x02 = directory
	RealSize         uint32
	AllocSize       uint32
	BaseRecord      uint64
	NextAttrID      uint16
}

// MFTAttribute represents an attribute within an MFT entry.
type MFTAttribute struct {
	Type     uint32
	Length   uint32
	NonResident uint8
	NameLen  uint8
	NameOffset uint16
	Flags    uint16
	AttrID   uint16
}

// NTFS attribute type constants
const (
	AttrStandardInfo = 0x10
	AttrFileName     = 0x30
	AttrData         = 0x80
	AttrBitmap       = 0xB0
	AttrIndexRoot    = 0x90
	AttrIndexAlloc   = 0xA0
)

// NewNTFSReader creates a new NTFS reader.
func NewNTFSReader(diskReader DiskReader, partitionStart uint64) (*NTFSReader, error) {
	data, err := diskReader.ReadSectors(partitionStart, 1)
	if err != nil {
		return nil, fmt.Errorf("filesystem: failed to read NTFS boot sector: %w", err)
	}

	sectorSize := binary.LittleEndian.Uint16(data[11:13])
	sectorsPerCluster := data[13]

	// MFT start cluster at offset 48
	mftStartCluster := binary.LittleEndian.Uint64(data[48:56])
	mftMirrorCluster := binary.LittleEndian.Uint64(data[56:64])

	// Bytes per cluster
	clusterSize := uint32(sectorSize) * uint32(sectorsPerCluster)

	r := &NTFSReader{
		bytesPerSector:    sectorSize,
		sectorsPerCluster: sectorsPerCluster,
		mftStartCluster:   mftStartCluster,
		mftMirrorCluster:  mftMirrorCluster,
		clusterSize:       clusterSize,
		sectorSize:        uint64(sectorSize),
		partitionStart:    partitionStart,
	}

	return r, nil
}

func (r *NTFSReader) DetectFSType(_ DiskReader, _ uint64) FSType {
	return NTFS
}

func (r *NTFSReader) ListRootDirectory(diskReader DiskReader, partitionStart uint64) ([]FileEntry, error) {
	// MFT entry 5 is the root directory
	return r.listMFTEntry(diskReader, 5)
}

func (r *NTFSReader) ListDirectory(diskReader DiskReader, partitionStart uint64, dirPath string) ([]FileEntry, error) {
	if dirPath == "/" || dirPath == "" {
		return r.ListRootDirectory(diskReader, partitionStart)
	}

	// Navigate directory tree using MFT
	parts := strings.Split(strings.Trim(dirPath, "/"), "/")
	mftRef := uint64(5) // Start with root

	for _, part := range parts {
		entries, err := r.listMFTEntry(diskReader, mftRef)
		if err != nil {
			return nil, err
		}
		found := false
		for _, entry := range entries {
			if strings.EqualFold(entry.Name, part) {
				// Use cluster start as MFT reference for simplicity
				mftRef = uint64(entry.ClusterStart)
				found = true
				break
			}
		}
		if !found {
			return nil, ErrFileNotFound
		}
	}

	return r.listMFTEntry(diskReader, mftRef)
}

func (r *NTFSReader) listMFTEntry(diskReader DiskReader, mftEntryNum uint64) ([]FileEntry, error) {
	// Calculate MFT entry offset
	entrySize := uint64(r.clusterSize)
	entryOffset := r.partitionStart*r.sectorSize + r.mftStartCluster*entrySize + mftEntryNum*entrySize

	data, err := diskReader.ReadSectors(entryOffset/r.sectorSize, uint32(entrySize/r.sectorSize))
	if err != nil {
		return nil, err
	}

	if len(data) < 42 {
		return nil, fmt.Errorf("filesystem: MFT entry too small")
	}

	// Check signature
	if string(data[0:4]) != "FILE" {
		return nil, fmt.Errorf("filesystem: invalid MFT entry signature")
	}

	// Parse MFT entry header
	firstAttrOffset := binary.LittleEndian.Uint16(data[20:22])
	flags := binary.LittleEndian.Uint16(data[22:24])

	// Check if it's a directory
	if flags&0x02 == 0 {
		// Not a directory
		return nil, nil
	}

	return r.parseMFTAttributes(diskReader, data, firstAttrOffset, "")
}

func (r *NTFSReader) parseMFTAttributes(diskReader DiskReader, entryData []byte, startOffset uint16, parentPath string) ([]FileEntry, error) {
	var entries []FileEntry
	offset := int(startOffset)

	for offset < len(entryData)-4 {
		attrType := binary.LittleEndian.Uint32(entryData[offset : offset+4])
		attrLen := binary.LittleEndian.Uint32(entryData[offset+4 : offset+8])

		if attrLen == 0 || attrLen > uint32(len(entryData)-offset) {
			break
		}

		if attrType == 0xFFFFFFFF || attrType == 0 {
			break
		}

		if attrType == AttrFileName {
			fileEntry := r.parseFileNameAttribute(entryData[offset:offset+int(attrLen)], parentPath)
			if fileEntry != nil {
				// Skip . and .. entries
				if fileEntry.Name != "." && fileEntry.Name != ".." {
					entries = append(entries, *fileEntry)
				}
			}
		}

		offset += int(attrLen)
	}

	return entries, nil
}

func (r *NTFSReader) parseFileNameAttribute(attrData []byte, parentPath string) *FileEntry {
	if len(attrData) < 66 {
		return nil
	}

	nonResident := attrData[8]
	if nonResident != 0 {
		return nil
	}

	nameLen := attrData[64]
	nameOffset := binary.LittleEndian.Uint16(attrData[62:64])

	if int(nameOffset)+int(nameLen)*2 > len(attrData) {
		return nil
	}

	// File name is UTF-16LE
	nameBytes := attrData[nameOffset : nameOffset+uint16(nameLen)*2]
	fileName := utf16LEToString(nameBytes)

	// File reference (MFT entry of parent) at offset 0-7
	parentRef := binary.LittleEndian.Uint64(attrData[0:8]) & 0xFFFFFFFFFFFF

	// File size at offset 48-55
	fileSize := int64(binary.LittleEndian.Uint64(attrData[48:56]))

	// Flags at offset 56-57
	flags := binary.LittleEndian.Uint16(attrData[56:58])
	isDir := flags&0x02 != 0

	// Creation time at offset 8-15
	created := filetimeToTime(binary.LittleEndian.Uint64(attrData[8:16]))
	// Modification time at offset 16-23
	modified := filetimeToTime(binary.LittleEndian.Uint64(attrData[16:24]))
	// MFT change time at offset 24-31
	// Accessed time at offset 32-39

	ext := ""
	if !isDir {
		if dotIdx := strings.LastIndex(fileName, "."); dotIdx >= 0 {
			ext = strings.ToLower(fileName[dotIdx+1:])
		}
	}

	return &FileEntry{
		Name:         fileName,
		Path:         parentPath + "/" + fileName,
		IsDirectory:  isDir,
		Size:         fileSize,
		CreatedTime:  created,
		ModifiedTime: modified,
		ClusterStart: uint32(parentRef),
		Extension:    ext,
		Attributes: FileAttributes{
			Directory: isDir,
			Hidden:    fileName[0] == '.',
		},
	}
}

func (r *NTFSReader) GetFileContent(diskReader DiskReader, _ uint64, file *FileEntry) ([]byte, error) {
	if file.IsDirectory {
		return nil, fmt.Errorf("filesystem: cannot read directory content")
	}

	// For NTFS, we'd need to follow the $DATA attribute
	// Simplified implementation: read clusters directly
	if file.ClusterStart == 0 {
		return nil, fmt.Errorf("filesystem: file has no data clusters")
	}

	totalClusters := (file.Size + int64(r.clusterSize) - 1) / int64(r.clusterSize)
	result := make([]byte, 0, file.Size)

	for i := int64(0); i < totalClusters; i++ {
		clusterOffset := r.partitionStart*r.sectorSize + (int64(file.ClusterStart)+i)*int64(r.clusterSize)
		clusterSectors := uint32(r.clusterSize) / uint32(r.sectorSize)

		data, err := diskReader.ReadSectors(uint64(clusterOffset)/r.sectorSize, clusterSectors)
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

func (r *NTFSReader) GetFileProperties(_ DiskReader, _ uint64, file *FileEntry) (*FileProperties, error) {
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
		FSType:        string(NTFS),
	}, nil
}

func (r *NTFSReader) SearchFiles(diskReader DiskReader, partitionStart uint64, query string, caseSensitive bool) ([]FileEntry, error) {
	var results []FileEntry
	query = strings.ToLower(query)

	var searchDir func(mftRef uint64, dirPath string) error
	searchDir = func(mftRef uint64, dirPath string) error {
		entries, err := r.listMFTEntry(diskReader, mftRef)
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
				searchDir(uint64(entry.ClusterStart), entry.Path)
			}
		}
		return nil
	}

	searchDir(5, "")
	return results, nil
}

// filetimeToTime converts Windows FILETIME (100-ns intervals since 1601) to time.Time.
func filetimeToTime(ft uint64) time.Time {
	const windowsEpoch = 116444736000000000 // 1601 to 1970 in 100-ns
	if ft == 0 {
		return time.Time{}
	}
	nano := (ft - windowsEpoch) * 100
	return time.Unix(0, int64(nano))
}

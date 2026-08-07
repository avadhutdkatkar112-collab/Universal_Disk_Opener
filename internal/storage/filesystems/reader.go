package filesystems

// DiskReader abstracts reading sectors from a disk.
type DiskReader interface {
	ReadSectors(offset uint64, count uint32) ([]byte, error)
	SectorSize() uint64
}

// Reader defines the interface for filesystem access.
type Reader interface {
	DetectFSType(diskReader DiskReader, partitionStart uint64) FSType
	ListRootDirectory(diskReader DiskReader, partitionStart uint64) ([]FileEntry, error)
	ListDirectory(diskReader DiskReader, partitionStart uint64, path string) ([]FileEntry, error)
	GetFileContent(diskReader DiskReader, partitionStart uint64, file *FileEntry) ([]byte, error)
	GetFileProperties(diskReader DiskReader, partitionStart uint64, file *FileEntry) (*FileProperties, error)
	SearchFiles(diskReader DiskReader, partitionStart uint64, query string, caseSensitive bool) ([]FileEntry, error)
}

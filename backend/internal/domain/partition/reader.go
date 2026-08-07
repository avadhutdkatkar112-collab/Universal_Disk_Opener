package partition

// DiskReader abstracts reading sectors from a disk.
type DiskReader interface {
	ReadSectors(offset uint64, count uint32) ([]byte, error)
	SectorSize() uint64
	TotalSectors() uint64
}

// Reader defines the interface for partition table parsers.
type Reader interface {
	ReadPartitions(diskReader DiskReader) ([]PartitionInfo, error)
	ReadMBR(diskReader DiskReader) (*MBR, error)
	ReadGPT(diskReader DiskReader) (*GPTHeader, []GPTPartitionEntry, error)
}

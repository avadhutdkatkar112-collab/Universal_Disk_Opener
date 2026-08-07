// Package qcow2 implements the QCOW2 (QEMU Copy-On-Write v2) format driver.
// QCOW2 is used by QEMU/KVM and supports compression, encryption, and snapshots.
package qcow2

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	"github.com/user/vhd-opener/internal/domain/disk"
)

const (
	qcow2Magic      = 0x514649FB // "QFI\xfb"
	qcow2Version2   = 2
	qcow2Version3   = 3
	qcow2SectorSize = 512
)

func init() {
	disk.Register(".qcow2", func() disk.VirtualDisk {
		return &QCOW2Driver{}
	})
}

// QCOW2Driver implements disk.VirtualDisk for QCOW2 files.
type QCOW2Driver struct {
	file           *os.File
	filePath       string
	fileName       string
	fileSize       int64
	virtualSize    uint64
	sectorSize     uint32
	clusterBits    uint32
	clusterSize    uint32
	l1Size         uint32
	l1TableOffset  uint64
	l2Cache        map[uint64][]uint64
	warnings       []string
}

// Open opens a QCOW2 file.
func (d *QCOW2Driver) Open(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("qcow2: failed to open file: %w", err)
	}

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return fmt.Errorf("qcow2: failed to stat file: %w", err)
	}

	if fi.Size() < 72 {
		f.Close()
		return fmt.Errorf("qcow2: file too small")
	}

	d.file = f
	d.filePath = path
	d.fileName = filepath.Base(path)
	d.fileSize = fi.Size()
	d.sectorSize = qcow2SectorSize
	d.l2Cache = make(map[uint64][]uint64)

	// Read header
	header := make([]byte, 72)
	n, err := f.ReadAt(header, 0)
	if err != nil || n < 72 {
		f.Close()
		return fmt.Errorf("qcow2: cannot read header")
	}

	// Verify magic
	magic := binary.BigEndian.Uint32(header[0:4])
	if magic != qcow2Magic {
		f.Close()
		return fmt.Errorf("qcow2: invalid magic: 0x%08X", magic)
	}

	// Version (offset 4, big-endian uint32)
	version := binary.BigEndian.Uint32(header[4:8])
	if version != qcow2Version2 && version != qcow2Version3 {
		f.Close()
		return fmt.Errorf("qcow2: unsupported version: %d", version)
	}

	// Backing file offset (offset 8, big-endian uint64)
	// Backing file size (offset 16, big-endian uint32)
	// Cluster bits (offset 24, big-endian uint32)
	d.clusterBits = binary.BigEndian.Uint32(header[24:28])
	if d.clusterBits < 8 || d.clusterBits > 30 {
		d.clusterBits = 16 // Default 64KB
	}
	d.clusterSize = 1 << d.clusterBits

	// Virtual disk size (offset 32, big-endian uint64)
	d.virtualSize = binary.BigEndian.Uint64(header[32:40])

	// Encryption method (offset 40, big-endian uint32)
	// L1 size (offset 44, big-endian uint32)
	d.l1Size = binary.BigEndian.Uint32(header[44:48])

	// L1 table offset (offset 48, big-endian uint64)
	d.l1TableOffset = binary.BigEndian.Uint64(header[48:56])

	return nil
}

// Close closes the QCOW2 file.
func (d *QCOW2Driver) Close() error {
	if d.file != nil {
		return d.file.Close()
	}
	return nil
}

// ReadSectors reads sectors from the QCOW2 file.
func (d *QCOW2Driver) ReadSectors(startSector uint64, count uint32) ([]byte, error) {
	if count == 0 {
		return nil, nil
	}

	startByte := startSector * uint64(d.sectorSize)
	readBytes := uint64(count) * uint64(d.sectorSize)
	result := make([]byte, readBytes)
	var bytesRead uint64

	for bytesRead < readBytes {
		virtualOffset := startByte + bytesRead
		clusterOffset := virtualOffset / uint64(d.clusterSize)
		offsetInCluster := virtualOffset % uint64(d.clusterSize)

		// Resolve L1 -> L2 -> host offset
		hostOffset, err := d.resolveOffset(clusterOffset)
		if err != nil || hostOffset == 0 {
			// Unallocated cluster - return zeros
			chunkSize := uint64(d.clusterSize) - offsetInCluster
			if chunkSize > readBytes-bytesRead {
				chunkSize = readBytes - bytesRead
			}
			bytesRead += chunkSize
			continue
		}

		// Read from host offset
		fileOffset := int64(hostOffset) + int64(offsetInCluster)
		chunkSize := uint64(d.clusterSize) - offsetInCluster
		if chunkSize > readBytes-bytesRead {
			chunkSize = readBytes - bytesRead
		}

		n, err := d.file.ReadAt(result[bytesRead:bytesRead+chunkSize], fileOffset)
		if err != nil {
			bytesRead += chunkSize
			continue
		}
		bytesRead += uint64(n)
	}

	return result, nil
}

// resolveOffset translates a virtual cluster offset to a host file offset.
func (d *QCOW2Driver) resolveOffset(virtualCluster uint64) (uint64, error) {
	if d.l1TableOffset == 0 || d.l1Size == 0 {
		return 0, fmt.Errorf("no L1 table")
	}

	// L1 index = virtualCluster / (clusterSize / 8)
	l2EntriesPerCluster := uint64(d.clusterSize) / 8
	if l2EntriesPerCluster == 0 {
		return 0, fmt.Errorf("invalid cluster size")
	}
	l1Index := virtualCluster / l2EntriesPerCluster
	l2Index := virtualCluster % l2EntriesPerCluster

	if l1Index >= uint64(d.l1Size) {
		return 0, fmt.Errorf("L1 index out of bounds")
	}

	// Read L1 entry
	l1EntryOffset := int64(d.l1TableOffset) + int64(l1Index)*8
	l1EntryBuf := make([]byte, 8)
	_, err := d.file.ReadAt(l1EntryBuf, l1EntryOffset)
	if err != nil {
		return 0, err
	}

	l1Entry := binary.BigEndian.Uint64(l1EntryBuf)
	l2Offset := l1Entry & 0x00FFFFFFFFFFFE00 // Mask out flags (bits 0-8, 63)
	if l2Offset == 0 {
		return 0, fmt.Errorf("L2 offset is zero")
	}

	// Check L2 cache
	var l2Table []uint64
	if cached, ok := d.l2Cache[l2Offset]; ok {
		l2Table = cached
	} else {
		// Read L2 table
		l2Entries := d.clusterSize / 8
		l2Buf := make([]byte, l2Entries*8)
		_, err := d.file.ReadAt(l2Buf, int64(l2Offset))
		if err != nil {
			return 0, err
		}

		l2Table = make([]uint64, l2Entries)
		for i := uint32(0); i < l2Entries; i++ {
			l2Table[i] = binary.BigEndian.Uint64(l2Buf[i*8 : i*8+8])
		}
		d.l2Cache[l2Offset] = l2Table
	}

	if l2Index >= uint64(len(l2Table)) {
		return 0, fmt.Errorf("L2 index out of bounds")
	}

	l2Entry := l2Table[l2Index]
	hostOffset := l2Entry & 0x00FFFFFFFFFFFE00 // Mask out flags

	return hostOffset, nil
}

// ReadAt implements io.ReaderAt.
func (d *QCOW2Driver) ReadAt(buf []byte, offset int64) (int, error) {
	startSector := uint64(offset) / uint64(d.sectorSize)
	count := uint32((uint64(len(buf)) + uint64(d.sectorSize) - 1) / uint64(d.sectorSize))

	data, err := d.ReadSectors(startSector, count)
	if err != nil {
		return 0, err
	}

	n := copy(buf, data)
	return n, nil
}

// Size returns the virtual disk size.
func (d *QCOW2Driver) Size() uint64 {
	return d.virtualSize
}

// SectorSize returns the sector size.
func (d *QCOW2Driver) SectorSize() uint32 {
	return d.sectorSize
}

// TotalSectors returns total number of sectors.
func (d *QCOW2Driver) TotalSectors() uint64 {
	return d.virtualSize / uint64(d.sectorSize)
}

// DiskType returns "Dynamic" for QCOW2 (always sparse).
func (d *QCOW2Driver) DiskType() string {
	return "Dynamic"
}

// Format returns "QCOW2".
func (d *QCOW2Driver) Format() string {
	return "QCOW2"
}

// Info returns disk information.
func (d *QCOW2Driver) Info() disk.DiskInfo {
	return disk.DiskInfo{
		FilePath:    d.filePath,
		FileName:    d.fileName,
		FileSize:    d.fileSize,
		VirtualSize: d.virtualSize,
		Format:      "QCOW2",
		DiskType:    "Dynamic",
		UniqueID:    "(none)",
		BlockSize:   d.clusterSize,
	}
}

// FilePath returns the file path.
func (d *QCOW2Driver) FilePath() string { return d.filePath }

// FileName returns the file name.
func (d *QCOW2Driver) FileName() string { return d.fileName }

// Warnings returns non-fatal warnings.
func (d *QCOW2Driver) Warnings() []string { return d.warnings }

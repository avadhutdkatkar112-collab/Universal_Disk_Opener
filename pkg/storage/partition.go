package storage

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
)

type MBRPartition struct {
	index         int
	startSector   uint32
	sectorCount   uint32
	partitionType byte
	parent        BlockReader
	safeReader    *SafeBlockReader
}

func (p *MBRPartition) Start() uint64 {
	return uint64(p.startSector) * 512
}

func (p *MBRPartition) Size(ctx context.Context) (uint64, error) {
	return uint64(p.sectorCount) * 512, nil
}

func (p *MBRPartition) Type() string {
	switch p.partitionType {
	case 0x07:
		return "NTFS / exFAT"
	case 0x83:
		return "Linux EXT4"
	case 0x0B, 0x0C:
		return "FAT32"
	case 0xEE:
		return "GPT Protective"
	default:
		return fmt.Sprintf("Unknown (0x%02X)", p.partitionType)
	}
}

func (p *MBRPartition) Index() int { return p.index }

func (p *MBRPartition) ReadAt(ctx context.Context, buf []byte, off uint64) (int, error) {
	absoluteOffset := p.Start() + off
	return p.parent.ReadAt(ctx, buf, absoluteOffset)
}

func ParsePartitions(ctx context.Context, disk DiskImage) ([]Partition, error) {
	sector0 := make([]byte, 512)
	if _, err := disk.ReadAt(ctx, sector0, 0); err != nil {
		return nil, fmt.Errorf("failed to read MBR sector 0: %w", err)
	}

	if sector0[510] != 0x55 || sector0[511] != 0xAA {
		return nil, fmt.Errorf("invalid MBR signature: %02X %02X", sector0[510], sector0[511])
	}

	firstEntryType := sector0[446+4]
	if firstEntryType == 0xEE {
		return parseGPT(ctx, disk)
	}

	var partitions []Partition

	for i := 0; i < 4; i++ {
		offset := 446 + (i * 16)
		entryBytes := sector0[offset : offset+16]

		partitionType := entryBytes[4]
		if partitionType == 0x00 {
			continue
		}

		startSector := binary.LittleEndian.Uint32(entryBytes[8:12])
		sectorCount := binary.LittleEndian.Uint32(entryBytes[12:16])

		if sectorCount == 0 {
			continue
		}

		part := &MBRPartition{
			index:         i + 1,
			startSector:   startSector,
			sectorCount:   sectorCount,
			partitionType: partitionType,
			parent:        disk,
		}
		partSize := uint64(sectorCount) * 512
		part.safeReader = NewSafeBlockReader(part, partSize)

		partitions = append(partitions, part)
	}

	return partitions, nil
}

func parseGPT(ctx context.Context, disk DiskImage) ([]Partition, error) {
	gptHeaderBytes := make([]byte, 512)
	if _, err := disk.ReadAt(ctx, gptHeaderBytes, 512); err != nil {
		return nil, fmt.Errorf("failed to read GPT header at LBA 1: %w", err)
	}

	if !bytes.Equal(gptHeaderBytes[0:8], []byte("EFI PART")) {
		return nil, fmt.Errorf("invalid GPT signature")
	}

	partitionEntryLBA := binary.LittleEndian.Uint64(gptHeaderBytes[72:80])
	numTableEntries := binary.LittleEndian.Uint32(gptHeaderBytes[80:84])
	entrySize := binary.LittleEndian.Uint32(gptHeaderBytes[84:88])

	tableOffset := partitionEntryLBA * 512
	totalTableBytes := uint64(numTableEntries) * uint64(entrySize)

	tableBytes := make([]byte, totalTableBytes)
	if _, err := disk.ReadAt(ctx, tableBytes, tableOffset); err != nil {
		return nil, fmt.Errorf("failed to read GPT partition entries: %w", err)
	}

	var partitions []Partition
	emptyGUID := make([]byte, 16)

	for i := uint32(0); i < numTableEntries; i++ {
		entryOffset := uint64(i) * uint64(entrySize)
		entry := tableBytes[entryOffset : entryOffset+uint64(entrySize)]

		typeGUID := entry[0:16]
		if bytes.Equal(typeGUID, emptyGUID) {
			continue
		}

		firstLBA := binary.LittleEndian.Uint64(entry[32:40])
		lastLBA := binary.LittleEndian.Uint64(entry[40:48])
		sectorCount := (lastLBA - firstLBA) + 1

		part := &MBRPartition{
			index:         int(i + 1),
			startSector:   uint32(firstLBA),
			sectorCount:   uint32(sectorCount),
			partitionType: 0x07,
			parent:        disk,
		}
		partSize := sectorCount * 512
		part.safeReader = NewSafeBlockReader(part, partSize)

		partitions = append(partitions, part)
	}

	return partitions, nil
}

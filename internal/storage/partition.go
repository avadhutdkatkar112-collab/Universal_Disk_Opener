package storage

import (
	"context"
	"encoding/binary"
	"fmt"
)

type MBRPartition struct {
	index         int
	startSector   uint32
	sectorCount   uint32
	partitionType byte
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

		ptype := entryBytes[4]
		if ptype == 0x00 {
			continue
		}

		startSector := binary.LittleEndian.Uint32(entryBytes[8:12])
		sectorCount := binary.LittleEndian.Uint32(entryBytes[12:16])

		if sectorCount == 0 {
			continue
		}

		start := uint64(startSector) * 512
		size := uint64(sectorCount) * 512
		parts := append(partitions, Partition{
			Index: i + 1,
			Start: start,
			End:   start + size - 1,
			Size:  size,
			Type:  partitionTypeName(ptype),
		})
		partitions = parts
	}

	return partitions, nil
}

func parseGPT(ctx context.Context, disk DiskImage) ([]Partition, error) {
	gptHeaderBytes := make([]byte, 512)
	if _, err := disk.ReadAt(ctx, gptHeaderBytes, 512); err != nil {
		return nil, fmt.Errorf("failed to read GPT header: %w", err)
	}

	if string(gptHeaderBytes[0:8]) != "EFI PART" {
		return nil, fmt.Errorf("invalid GPT signature")
	}

	partEntryStart := binary.LittleEndian.Uint64(gptHeaderBytes[72:80])
	numPartEntries := binary.LittleEndian.Uint32(gptHeaderBytes[80:84])
	partEntrySize := binary.LittleEndian.Uint32(gptHeaderBytes[84:88])

	var partitions []Partition
	entryBuf := make([]byte, partEntrySize)

	for i := uint32(0); i < numPartEntries; i++ {
		off := uint64(partEntryStart)*512 + uint64(i)*uint64(partEntrySize)
		if _, err := disk.ReadAt(ctx, entryBuf, off); err != nil {
			continue
		}

		typeGUID := entryBuf[0:16]
		allZeros := true
		for _, b := range typeGUID {
			if b != 0 {
				allZeros = false
				break
			}
		}
		if allZeros {
			continue
		}

		startLBA := binary.LittleEndian.Uint64(entryBuf[32:40])
		endLBA := binary.LittleEndian.Uint64(entryBuf[40:48])
		size := (endLBA - startLBA + 1) * 512

		partitions = append(partitions, Partition{
			Index: int(i + 1),
			Start: startLBA * 512,
			End:   endLBA * 512,
			Size:  size,
			Type:  "GPT",
		})
	}

	return partitions, nil
}

func partitionTypeName(t byte) string {
	switch t {
	case 0x07:
		return "NTFS / exFAT"
	case 0x83:
		return "Linux EXT4"
	case 0x0B, 0x0C:
		return "FAT32"
	case 0xEE:
		return "GPT Protective"
	default:
		return fmt.Sprintf("Unknown (0x%02X)", t)
	}
}

package registry

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Index cell signatures
var (
	LFSignature = [2]byte{'l', 'f'} // Fast leaf (ASCII prefix match)
	LHSignature = [2]byte{'l', 'h'} // Hash leaf (Murmur hash)
	RISignature = [2]byte{'r', 'i'} // Root indirect (nested indexes)
	LISignature = [2]byte{'l', 'i'} // Direct leaf list
)

// SubkeyElement represents an element in a subkey index.
type SubkeyElement struct {
	NKOffset uint32
	Hash     [4]byte
}

// ReadSubkeyOffsets parses any index cell type (lf, lh, ri, li) and returns NK cell offsets.
func (h *RegistryHive) ReadSubkeyOffsets(indexOffset uint32) ([]uint32, error) {
	cell, ok := h.Cells[indexOffset]
	if !ok {
		return nil, fmt.Errorf("cell not found at offset 0x%X", indexOffset)
	}

	if len(cell.Raw) < 4 {
		return nil, errors.New("corrupted subkey index cell: buffer too short")
	}

	sig := [2]byte{cell.Raw[4], cell.Raw[5]}
	count := binary.LittleEndian.Uint16(cell.Raw[6:8])

	switch sig {
	case LFSignature, LHSignature:
		return parseLeafIndex(cell.Raw[8:], count)
	case LISignature:
		return parseDirectList(cell.Raw[8:], count)
	case RISignature:
		return parseIndirectList(h, cell.Raw[8:], count)
	default:
		return nil, fmt.Errorf("unsupported index signature '%s' at offset 0x%X", string(sig[:]), indexOffset)
	}
}

// parseLeafIndex extracts NK offsets from 'lf' or 'lh' elements (8 bytes each).
func parseLeafIndex(buf []byte, count uint16) ([]uint32, error) {
	requiredLen := int(count) * 8
	if len(buf) < requiredLen {
		return nil, fmt.Errorf("truncated leaf index: need %d bytes, got %d", requiredLen, len(buf))
	}

	offsets := make([]uint32, count)
	for i := 0; i < int(count); i++ {
		offsets[i] = binary.LittleEndian.Uint32(buf[i*8 : i*8+4])
	}
	return offsets, nil
}

// parseDirectList extracts NK offsets from 'li' elements (4 bytes each).
func parseDirectList(buf []byte, count uint16) ([]uint32, error) {
	requiredLen := int(count) * 4
	if len(buf) < requiredLen {
		return nil, fmt.Errorf("truncated direct list: need %d bytes, got %d", requiredLen, len(buf))
	}

	offsets := make([]uint32, count)
	for i := 0; i < int(count); i++ {
		offsets[i] = binary.LittleEndian.Uint32(buf[i*4 : i*4+4])
	}
	return offsets, nil
}

// parseIndirectList resolves 'ri' indirect cells recursively.
func parseIndirectList(h *RegistryHive, buf []byte, count uint16) ([]uint32, error) {
	requiredLen := int(count) * 4
	if len(buf) < requiredLen {
		return nil, fmt.Errorf("truncated indirect list: need %d bytes, got %d", requiredLen, len(buf))
	}

	var allOffsets []uint32
	for i := 0; i < int(count); i++ {
		childOffset := binary.LittleEndian.Uint32(buf[i*4 : i*4+4])
		childOffsets, err := h.ReadSubkeyOffsets(childOffset)
		if err != nil {
			continue // Skip corrupted child indexes
		}
		allOffsets = append(allOffsets, childOffsets...)
	}
	return allOffsets, nil
}

package registry

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

// ParseVK parses a Value Key cell.
func (h *RegistryHive) ParseVK(cellOffset uint32) (*VKCell, error) {
	cell, ok := h.Cells[cellOffset]
	if !ok {
		return nil, fmt.Errorf("cell not found at offset 0x%X", cellOffset)
	}

	if len(cell.Raw) < 24 {
		return nil, fmt.Errorf("VK cell too small: %d bytes", len(cell.Raw))
	}

	vk := &VKCell{}

	vk.Signature = binary.LittleEndian.Uint16(cell.Raw[4:6])
	if vk.Signature != cellVK {
		return nil, fmt.Errorf("not a VK cell: 0x%04X", vk.Signature)
	}

	vk.NameLength = binary.LittleEndian.Uint16(cell.Raw[6:8])
	vk.DataLength = binary.LittleEndian.Uint32(cell.Raw[8:12])
	vk.DataOffset = binary.LittleEndian.Uint32(cell.Raw[12:16])
	vk.DataType = binary.LittleEndian.Uint32(cell.Raw[16:20])
	vk.Flags = binary.LittleEndian.Uint16(cell.Raw[20:22])

	// Parse value name
	nameStart := 24
	nameEnd := nameStart + int(vk.NameLength)
	if nameEnd > len(cell.Raw) {
		nameEnd = len(cell.Raw)
	}
	nameBytes := cell.Raw[nameStart:nameEnd]

	if vk.Flags&0x0001 != 0 {
		// ASCII name
		vk.ValueName = string(nameBytes)
	} else if vk.NameLength > 0 {
		// UTF-16LE name
		vk.ValueName = decodeUTF16LE(nameBytes)
	}

	if vk.ValueName == "" {
		vk.ValueName = "(Default)"
	}

	return vk, nil
}

// ReadVKData retrieves the raw data for a VK cell.
func (h *RegistryHive) ReadVKData(vk *VKCell) ([]byte, error) {
	// Bit 31 set = inline data
	isInline := (vk.DataLength & 0x80000000) != 0
	realLength := vk.DataLength &^ 0x80000000

	if isInline {
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, vk.DataOffset)
		if realLength > 4 {
			return nil, fmt.Errorf("invalid inline data length: %d", realLength)
		}
		return buf[:realLength], nil
	}

	if realLength == 0 {
		return []byte{}, nil
	}

	cell, ok := h.Cells[vk.DataOffset]
	if !ok {
		return nil, fmt.Errorf("data cell not found at offset 0x%X", vk.DataOffset)
	}

	dataLen := int(realLength)
	if dataLen > len(cell.Raw)-4 {
		dataLen = len(cell.Raw) - 4
	}
	if dataLen <= 0 {
		return []byte{}, nil
	}

	return cell.Raw[4 : 4+dataLen], nil
}

// GetValuesForNK returns all values for a key node.
func (h *RegistryHive) GetValuesForNK(nk *NKCell) ([]*ParsedValue, error) {
	if nk.NumValues == 0 || nk.ValueListOffset == 0xFFFFFFFF {
		return []*ParsedValue{}, nil
	}

	// Read value list cell
	listCell, ok := h.Cells[nk.ValueListOffset]
	if !ok {
		return nil, fmt.Errorf("value list cell not found at offset 0x%X", nk.ValueListOffset)
	}

	values := make([]*ParsedValue, 0, nk.NumValues)
	for i := uint32(0); i < nk.NumValues; i++ {
		off := int(i) * 4
		if off+4 > len(listCell.Raw) {
			break
		}
		vkOffset := binary.LittleEndian.Uint32(listCell.Raw[off : off+4])

		vk, err := h.ParseVK(vkOffset)
		if err != nil {
			continue
		}

		data, err := h.ReadVKData(vk)
		if err != nil {
			data = []byte{}
		}

		values = append(values, &ParsedValue{
			Name:     vk.ValueName,
			DataType: vk.DataType,
			Data:     data,
		})
	}

	return values, nil
}

// FormatValueData formats a value for display.
func FormatValueData(val *ParsedValue) string {
	switch val.DataType {
	case TypeString, TypeExpandString:
		s := decodeUTF16LE(val.Data)
		s = strings.TrimRight(s, "\x00")
		return s
	case TypeDWORD:
		if len(val.Data) >= 4 {
			n := binary.LittleEndian.Uint32(val.Data)
			return fmt.Sprintf("0x%08X (%d)", n, n)
		}
	case TypeQWORD:
		if len(val.Data) >= 8 {
			n := binary.LittleEndian.Uint64(val.Data)
			return fmt.Sprintf("0x%016X (%d)", n, n)
		}
	case TypeBinary:
		if len(val.Data) <= 64 {
			return hex.EncodeToString(val.Data)
		}
		return hex.EncodeToString(val.Data[:64]) + "..."
	case TypeMultiString:
		return decodeMultiSZ(val.Data)
	}
	return hex.EncodeToString(val.Data)
}

// decodeMultiSZ decodes a REG_MULTI_SZ value.
func decodeMultiSZ(data []byte) string {
	if len(data) < 2 {
		return ""
	}
	u16s := make([]uint16, len(data)/2)
	for i := range u16s {
		u16s[i] = binary.LittleEndian.Uint16(data[i*2:])
	}
	runes := make([]rune, len(u16s))
	for i, u := range u16s {
		runes[i] = rune(u)
	}
	raw := strings.TrimRight(string(runes), "\x00")
	parts := strings.Split(raw, "\x00")
	var cleaned []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			cleaned = append(cleaned, p)
		}
	}
	return strings.Join(cleaned, ", ")
}

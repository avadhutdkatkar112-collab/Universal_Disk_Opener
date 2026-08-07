// Package registry implements binary parsing for Windows Registry hive files.
// Parses REGF headers, HBIN cells, and walks key/value trees for forensic extraction.
package registry

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode/utf16"
)

// REGF signature constants
const (
	regfSignature = "regf"
	hbinSignature = "hbin"
	cellFree      = 0xfffffff7 // -9 = free cell
	cellAllocMin  = 32        // minimum allocated cell size

	// Cell signatures (2 bytes, big-endian for nk, vk, etc.)
	cellNK = 0x6e6b // "nk" - key node
	cellVK = 0x766b // "vk" - value key
	cellLF = 0x6c66 // "lf" - fast leaf (name prefix)
	cellLH = 0x6c68 // "lh" - hash leaf (name hash)
	cellRI = 0x7269 // "ri" - root index
	cellLI = 0x6c69 // "li" - leaf index
	cellSK = 0x736b // "sk" - security key
	cellCL = 0x636c // "cl" - class name
	cellDB = 0x6462 // "db" - big data
)

// ─── REGF File Header (4096 bytes) ──────────────────────────────────────────
type REGFHeader struct {
	Signature           [4]byte    // "regf"
	PrimarySequence    uint32     // Primary sequence number
	SecondarySequence  uint32     // Secondary sequence number
	LastModified       uint64     // Windows FILETIME
	MajorVersion       uint16
	MinorVersion       uint16
	Type               uint32     // 0 = primary, 1 = log
	Format             uint32     // 1 = direct memory load
	RootCellOffset     uint32     // Offset to root key cell (from start of hive bin data)
	HiveBinsDataSize   uint32     // Size of hive bins data in bytes
	ClusteringFactor   uint32     // 1 = no clustering
	FileName           [64]byte   // UTF-16LE encoded filename
	Checksum           uint32     // XOR checksum of first 500 bytes (bytes 0-499)
	// Padding to 4096 bytes
	Reserved           [3572]byte
}

// ─── HBIN Header (32 bytes) ─────────────────────────────────────────────────
type HBINHeader struct {
	Signature      [4]byte // "hbin"
	Offset         uint32  // Offset from start of hive bins data (always multiple of 4096)
	Size           uint32  // Size of this hive bin in bytes (multiple of 4096)
	Reserved       [8]byte
	Timestamp      uint64  // Windows FILETIME
	Spare          uint32
}

// ─── Cell Types ──────────────────────────────────────────────────────────────

// CellHeader is the common header for all cells.
// The first 4 bytes of every cell is the signed size (negative = free, positive = allocated).
type CellHeader struct {
	Size int32 // Negative = free cell, positive = allocated
}

// NK Cell - Key Node (registry key)
type NKCell struct {
	// Cell header (already parsed)
	Signature         uint16 // 0x6e6b = "nk"
	Flags             uint16
	Timestamp         uint64 // FILETIME
	AccessMask        uint32
	ParentOffset      uint32 // Offset to parent key cell
	NumSubkeys        uint32
	NumVolatileSubkeys uint32
	SubkeyListOffset  uint32 // Offset to subkey list (lf/lh/ri/li cell)
	VolatileSubkeyListOffset uint32
	NumValues         uint32
	ValueListOffset   uint32 // Offset to value list cell
	SecurityKeyOffset uint32 // Offset to sk cell
	ClassNamesOffset  uint32
	MaxSubkeyNameLength   uint32
	MaxSubkeyClassLength  uint32
	MaxValueNameLength    uint32
	MaxValueDataLength    uint32
	WorkVar           uint32
	KeyNameLength     uint16
	ClassNamesLength  uint16
	KeyName           string // ASCII or UTF-16LE (depending on Flags)
}

// Flags for NK cell
const (
	nkFlagKeyDeleted  = 0x0010
	nkFlagVolatileKey = 0x0020
	nkFlagHiveEntry   = 0x0040
	nkFlagHiveEntry2  = 0x0080
	nkFlagCompressed  = 0x0200
	nkFlagSymbolicLink = 0x1000
	nkFlagVolatileConn = 0x2000
	nkFlagPredefinedName = 0x4000
	nkFlagVirtMirror = 0x8000
)

// VK Cell - Value Key (registry value)
type VKCell struct {
	Signature       uint16 // 0x766b = "vk"
	NameLength      uint16
	DataLength      uint32
	DataOffset      uint32 // Offset to data cell (or inline if <= 4 bytes)
	DataType        uint32
	Flags           uint16
	Spare           uint16
	ValueName       string // ASCII or UTF-16LE
	Data            []byte
}

// Data types for VK cell
const (
	TypeNone           uint32 = 0
	TypeString         uint32 = 1
	TypeExpandString   uint32 = 2
	TypeBinary         uint32 = 3
	TypeDWORD          uint32 = 4
	TypeDWORDBigEndian uint32 = 5
	TypeLink           uint32 = 6
	TypeMultiString    uint32 = 7
	TypeResourceList   uint32 = 8
	TypeFullResourceDescriptor uint32 = 9
	TypeResourceRequirementsList uint32 = 10
	TypeQWORD          uint32 = 11
)

// LF/LH Cell - Leaf (fast leaf / hash leaf) for subkey indexing
type LeafCell struct {
	Signature   uint16 // 0x6c66 (lf) or 0x6c68 (lh)
	NumKeys     uint16
	Elements    []LeafElement
}

type LeafElement struct {
	KeyOffset  uint32 // Offset to NK cell
	NameHint   [4]byte // For LF: first 4 chars; For LH: hash of name
}

// RI Cell - Root Index (for large key counts)
type RootIndexCell struct {
	Signature uint16 // 0x7269 = "ri"
	NumLists  uint16
	ListOffsets []uint32 // Offsets to leaf/index cells
}

// LI Cell - Leaf Index
type LeafIndexCell struct {
	Signature uint16 // 0x6c69 = "li"
	NumKeys   uint16
	Offsets   []uint32
}

// SK Cell - Security Key
type SKCell struct {
	Signature       uint16 // 0x736b = "sk"
	Reserved        uint16
	FlinkOffset     uint32 // Forward link
	BlinkOffset     uint32 // Backlink
	ReferenceCount  uint32
	SecuritySize    uint32
	Flags           uint32
}

// ─── Parsed Registry Key ─────────────────────────────────────────────────────

type ParsedKey struct {
	Name         string
	Path         string
	NumSubkeys   uint32
	NumValues    uint32
	Timestamp    time.Time
	Subkeys      []*ParsedKey
	Values       []*ParsedValue
	ClassName    string
	SecurityDesc []byte
}

type ParsedValue struct {
	Name     string
	DataType uint32
	Data     []byte
	RawData  []byte
}

// ─── Main Parser Structure ───────────────────────────────────────────────────

type RegistryHive struct {
	RawData     []byte
	Header      *REGFHeader
	HiveBins    []*HBINHeader
	Cells       map[uint32]*CellData // offset -> parsed cell
	RootKey     *ParsedKey
	ParsedAt    time.Time
}

type CellData struct {
	Offset uint32
	Size   int32
	Type   string // "nk", "vk", "lf", "lh", "ri", "li", "sk", "db", "free"
	Raw    []byte
}

// ─── REGF Header Parser ──────────────────────────────────────────────────────

func ParseREGFHeader(data []byte) (*REGFHeader, error) {
	if len(data) < 4096 {
		return nil, fmt.Errorf("data too small for REGF header: %d bytes (need 4096)", len(data))
	}

	h := &REGFHeader{}

	// Signature
	copy(h.Signature[:], data[0:4])
	if string(h.Signature[:]) != regfSignature {
		return nil, fmt.Errorf("invalid REGF signature: got %q, expected %q", string(h.Signature[:]), regfSignature)
	}

	h.PrimarySequence = binary.LittleEndian.Uint32(data[4:8])
	h.SecondarySequence = binary.LittleEndian.Uint32(data[8:12])
	h.LastModified = binary.LittleEndian.Uint64(data[12:20])
	h.MajorVersion = binary.LittleEndian.Uint16(data[20:22])
	h.MinorVersion = binary.LittleEndian.Uint16(data[22:24])
	h.Type = binary.LittleEndian.Uint32(data[24:28])
	h.Format = binary.LittleEndian.Uint32(data[28:32])
	h.RootCellOffset = binary.LittleEndian.Uint32(data[32:36])
	h.HiveBinsDataSize = binary.LittleEndian.Uint32(data[36:40])
	h.ClusteringFactor = binary.LittleEndian.Uint32(data[40:44])
	copy(h.FileName[:], data[44:108])
	h.Checksum = binary.LittleEndian.Uint32(data[500:504])

	// Validate checksum
	if err := h.ValidateChecksum(data[:500]); err != nil {
		// Warning only - some hives have bad checksums
		_ = err
	}

	return h, nil
}

// ValidateChecksum verifies the XOR checksum of the first 500 bytes.
func (h *REGFHeader) ValidateChecksum(data []byte) error {
	if len(data) < 500 {
		return fmt.Errorf("data too small for checksum")
	}
	var sum uint32
	for i := 0; i < 500; i += 4 {
		sum ^= binary.LittleEndian.Uint32(data[i : i+4])
	}
	if sum != h.Checksum {
		return fmt.Errorf("checksum mismatch: got 0x%08X, expected 0x%08X", sum, h.Checksum)
	}
	return nil
}

// ModifiedTime returns the last modified time as Go time.Time.
func (h *REGFHeader) ModifiedTime() time.Time {
	return fileTimeToTime(h.LastModified)
}

// ─── HBIN Parser ─────────────────────────────────────────────────────────────

func ParseHBIN(data []byte, binOffset uint32) (*HBINHeader, error) {
	if len(data) < 32 {
		return nil, fmt.Errorf("data too small for HBIN header: %d bytes", len(data))
	}

	h := &HBINHeader{}

	copy(h.Signature[:], data[0:4])
	if string(h.Signature[:]) != hbinSignature {
		return nil, fmt.Errorf("invalid HBIN signature: got %q, expected %q", string(h.Signature[:]), hbinSignature)
	}

	h.Offset = binary.LittleEndian.Uint32(data[4:8])
	h.Size = binary.LittleEndian.Uint32(data[8:12])
	h.Timestamp = binary.LittleEndian.Uint64(data[24:32])

	if h.Size < 4096 {
		return nil, fmt.Errorf("HBIN too small: %d bytes", h.Size)
	}

	return h, nil
}

// ─── Cell Parsers ────────────────────────────────────────────────────────────

func ParseCell(data []byte, cellOffset uint32) (*CellData, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("cell too small: %d bytes", len(data))
	}

	size := int32(binary.LittleEndian.Uint32(data[0:4]))
	cellSize := size
	if cellSize < 0 {
		cellSize = -cellSize
	}

	if cellSize < cellAllocMin || int(cellSize) > len(data) {
		return nil, fmt.Errorf("invalid cell size: %d (offset: 0x%X)", size, cellOffset)
	}

	cd := &CellData{
		Offset: cellOffset,
		Size:   size,
		Raw:    data[:cellSize],
	}

	if size > 0 && len(data) >= 8 {
		sig := binary.LittleEndian.Uint16(data[4:6])
		switch sig {
		case cellNK:
			cd.Type = "nk"
		case cellVK:
			cd.Type = "vk"
		case cellLF:
			cd.Type = "lf"
		case cellLH:
			cd.Type = "lh"
		case cellRI:
			cd.Type = "ri"
		case cellLI:
			cd.Type = "li"
		case cellSK:
			cd.Type = "sk"
		default:
			cd.Type = "unknown"
		}
	} else if size < 0 {
		cd.Type = "free"
	}

	return cd, nil
}

// ─── NK Cell Parser (Key Node) ───────────────────────────────────────────────

func ParseNKCell(data []byte) (*NKCell, error) {
	if len(data) < 76 {
		return nil, fmt.Errorf("NK cell too small: %d bytes", len(data))
	}

	nk := &NKCell{}

	nk.Signature = binary.LittleEndian.Uint16(data[4:6])
	if nk.Signature != cellNK {
		return nil, fmt.Errorf("not an NK cell: 0x%04X", nk.Signature)
	}

	nk.Flags = binary.LittleEndian.Uint16(data[6:8])
	nk.Timestamp = binary.LittleEndian.Uint64(data[8:16])
	nk.AccessMask = binary.LittleEndian.Uint32(data[16:20])
	nk.ParentOffset = binary.LittleEndian.Uint32(data[20:24])
	nk.NumSubkeys = binary.LittleEndian.Uint32(data[24:28])
	nk.NumVolatileSubkeys = binary.LittleEndian.Uint32(data[28:32])
	nk.SubkeyListOffset = binary.LittleEndian.Uint32(data[32:36])
	nk.VolatileSubkeyListOffset = binary.LittleEndian.Uint32(data[36:40])
	nk.NumValues = binary.LittleEndian.Uint32(data[40:44])
	nk.ValueListOffset = binary.LittleEndian.Uint32(data[44:48])
	nk.SecurityKeyOffset = binary.LittleEndian.Uint32(data[48:52])
	nk.ClassNamesOffset = binary.LittleEndian.Uint32(data[52:56])
	nk.MaxSubkeyNameLength = binary.LittleEndian.Uint32(data[56:60])
	nk.MaxSubkeyClassLength = binary.LittleEndian.Uint32(data[60:64])
	nk.MaxValueNameLength = binary.LittleEndian.Uint32(data[64:68])
	nk.MaxValueDataLength = binary.LittleEndian.Uint32(data[68:72])
	nk.WorkVar = binary.LittleEndian.Uint32(data[72:76])
	nk.KeyNameLength = binary.LittleEndian.Uint16(data[76:78])
	nk.ClassNamesLength = binary.LittleEndian.Uint16(data[78:80])

	// Parse key name
	nameStart := 80
	nameEnd := nameStart + int(nk.KeyNameLength)
	if nameEnd > len(data) {
		nameEnd = len(data)
	}
	nameBytes := data[nameStart:nameEnd]

	if nk.Flags&nkFlagVolatileKey != 0 {
		// Volatile key - ASCII name
		nk.KeyName = string(nameBytes)
	} else if nk.KeyNameLength > 0 {
		// Try to detect if ASCII or UTF-16LE
		if isASCII(nameBytes) {
			nk.KeyName = string(nameBytes)
		} else {
			nk.KeyName = decodeUTF16LE(nameBytes)
		}
	}

	return nk, nil
}

// ─── VK Cell Parser (Value Key) ──────────────────────────────────────────────

func ParseVKCell(data []byte) (*VKCell, error) {
	if len(data) < 20 {
		return nil, fmt.Errorf("VK cell too small: %d bytes", len(data))
	}

	vk := &VKCell{}

	vk.Signature = binary.LittleEndian.Uint16(data[4:6])
	if vk.Signature != cellVK {
		return nil, fmt.Errorf("not a VK cell: 0x%04X", vk.Signature)
	}

	vk.NameLength = binary.LittleEndian.Uint16(data[6:8])
	vk.DataLength = binary.LittleEndian.Uint32(data[8:12])
	vk.DataOffset = binary.LittleEndian.Uint32(data[12:16])
	vk.DataType = binary.LittleEndian.Uint32(data[16:20])
	vk.Flags = binary.LittleEndian.Uint16(data[20:22])
	vk.Spare = binary.LittleEndian.Uint16(data[22:24])

	// Parse value name
	nameStart := 24
	nameEnd := nameStart + int(vk.NameLength)
	if nameEnd > len(data) {
		nameEnd = len(data)
	}
	nameBytes := data[nameStart:nameEnd]

	if vk.Flags&0x0001 != 0 {
		// ASCII name
		vk.ValueName = string(nameBytes)
	} else if vk.NameLength > 0 {
		// UTF-16LE name
		vk.ValueName = decodeUTF16LE(nameBytes)
	}

	return vk, nil
}

// ─── Leaf Cell Parser (LF/LH) ───────────────────────────────────────────────

func ParseLeafCell(data []byte) (*LeafCell, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("leaf cell too small")
	}

	leaf := &LeafCell{}

	leaf.Signature = binary.LittleEndian.Uint16(data[4:6])
	leaf.NumKeys = binary.LittleEndian.Uint16(data[6:8])

	if leaf.Signature != cellLF && leaf.Signature != cellLH {
		return nil, fmt.Errorf("not a leaf cell: 0x%04X", leaf.Signature)
	}

	// Each element is 8 bytes (4 offset + 4 hint)
	elemStart := 8
	for i := uint16(0); i < leaf.NumKeys; i++ {
		off := elemStart + int(i)*8
		if off+8 > len(data) {
			break
		}
		elem := LeafElement{
			KeyOffset: binary.LittleEndian.Uint32(data[off : off+4]),
		}
		copy(elem.NameHint[:], data[off+4:off+8])
		leaf.Elements = append(leaf.Elements, elem)
	}

	return leaf, nil
}

// ─── Root Index Cell Parser (RI) ─────────────────────────────────────────────

func ParseRootIndexCell(data []byte) (*RootIndexCell, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("RI cell too small")
	}

	ri := &RootIndexCell{}

	ri.Signature = binary.LittleEndian.Uint16(data[4:6])
	ri.NumLists = binary.LittleEndian.Uint16(data[6:8])

	if ri.Signature != cellRI {
		return nil, fmt.Errorf("not an RI cell: 0x%04X", ri.Signature)
	}

	// Each element is 4 bytes (offset)
	elemStart := 8
	for i := uint16(0); i < ri.NumLists; i++ {
		off := elemStart + int(i)*4
		if off+4 > len(data) {
			break
		}
		ri.ListOffsets = append(ri.ListOffsets, binary.LittleEndian.Uint32(data[off:off+4]))
	}

	return ri, nil
}

// ─── Full Hive Parser ────────────────────────────────────────────────────────

func ParseHive(data []byte) (*RegistryHive, error) {
	if len(data) < 4096 {
		return nil, fmt.Errorf("hive too small: %d bytes", len(data))
	}

	hive := &RegistryHive{
		RawData:  data,
		Cells:    make(map[uint32]*CellData),
		ParsedAt: time.Now(),
	}

	// Parse REGF header
	header, err := ParseREGFHeader(data)
	if err != nil {
		return nil, fmt.Errorf("parse REGF header: %w", err)
	}
	hive.Header = header

	// Parse HBINs
	hiveBinsDataStart := 4096
	hiveBinsDataEnd := hiveBinsDataStart + int(header.HiveBinsDataSize)
	if hiveBinsDataEnd > len(data) {
		hiveBinsDataEnd = len(data)
	}

	offset := hiveBinsDataStart
	for offset < hiveBinsDataEnd {
		if offset+32 > len(data) {
			break
		}

		hbin, err := ParseHBIN(data[offset:], uint32(offset))
		if err != nil {
			// Skip bad HBIN
			offset += 4096
			continue
		}
		hive.HiveBins = append(hive.HiveBins, hbin)

		// Parse cells within this HBIN
		cellOffset := offset + 32 // Skip HBIN header
		hbinEnd := offset + int(hbin.Size)

		for cellOffset < hbinEnd && cellOffset+4 <= len(data) {
			cellSize := int32(binary.LittleEndian.Uint32(data[cellOffset:]))
			absSize := cellSize
			if absSize < 0 {
				absSize = -absSize
			}
			if absSize < cellAllocMin {
				break
			}

			cellEnd := cellOffset + int(absSize)
			if cellEnd > len(data) {
				break
			}

			cell, err := ParseCell(data[cellOffset:cellEnd], uint32(cellOffset))
			if err == nil && cell != nil {
				hive.Cells[uint32(cellOffset)] = cell
			}

			cellOffset += int(absSize)
			// Align to 8-byte boundary
			if cellOffset%8 != 0 {
				cellOffset += 8 - (cellOffset % 8)
			}
		}

		offset += int(hbin.Size)
	}

	// Parse root key
	if header.RootCellOffset > 0 {
		rootCell, ok := hive.Cells[header.RootCellOffset]
		if ok && rootCell.Type == "nk" {
			nk, err := ParseNKCell(rootCell.Raw)
			if err == nil {
				hive.RootKey = &ParsedKey{
					Name:       nk.KeyName,
					Path:       "\\",
					NumSubkeys: nk.NumSubkeys,
					NumValues:  nk.NumValues,
					Timestamp:  fileTimeToTime(nk.Timestamp),
				}
			}
		}
	}

	return hive, nil
}

// ─── Key/Value Tree Walker ───────────────────────────────────────────────────

// WalkKeys recursively walks all keys in the hive.
func (h *RegistryHive) WalkKeys(callback func(key *ParsedKey, depth int)) {
	if h.RootKey == nil {
		return
	}
	h.walkKeyRecursive(h.RootKey, 0, callback)
}

func (h *RegistryHive) walkKeyRecursive(key *ParsedKey, depth int, callback func(*ParsedKey, int)) {
	callback(key, depth)

	for _, subkey := range key.Subkeys {
		h.walkKeyRecursive(subkey, depth+1, callback)
	}
}

// FindKey searches for a key by path.
func (h *RegistryHive) FindKey(path string) *ParsedKey {
	if h.RootKey == nil {
		return nil
	}
	return h.findKeyRecursive(h.RootKey, path)
}

func (h *RegistryHive) findKeyRecursive(key *ParsedKey, targetPath string) *ParsedKey {
	if key.Path == targetPath {
		return key
	}
	for _, subkey := range key.Subkeys {
		if result := h.findKeyRecursive(subkey, targetPath); result != nil {
			return result
		}
	}
	return nil
}

// ─── Specialized Extractors ──────────────────────────────────────────────────

// ExtractOSInfo extracts OS information from SYSTEM hive.
func (h *RegistryHive) ExtractOSInfo() map[string]interface{} {
	info := make(map[string]interface{})

	// Try to find ComputerName
	computerNameKey := h.FindKey(`Select`)
	if computerNameKey != nil {
		info["select_key"] = computerNameKey.Path
	}

	// Try CurrentControlSet
	ccsKey := h.FindKey(`ControlSet001`)
	if ccsKey != nil {
		info["control_set"] = "ControlSet001"
	}

	return info
}

// ExtractRunKeys extracts Run/RunOnce keys (autostart locations).
func (h *RegistryHive) ExtractRunKeys() []map[string]interface{} {
	var results []map[string]interface{}

	runKeyPaths := []string{
		`Microsoft\Windows\CurrentVersion\Run`,
		`Microsoft\Windows\CurrentVersion\RunOnce`,
		`Microsoft\Windows\CurrentVersion\RunServices`,
		`Microsoft\Windows\CurrentVersion\RunServicesOnce`,
		`Microsoft\Windows\CurrentVersion\Explorer\Shell Folders`,
		`Microsoft\Windows\CurrentVersion\Explorer\User Shell Folders`,
		`Microsoft\Windows NT\CurrentVersion\Winlogon`,
		`Microsoft\Windows NT\CurrentVersion\Windows`,
		`Microsoft\Windows NT\CurrentVersion\Terminal Server\Install\Software\Microsoft\Windows\CurrentVersion\Run`,
		`Microsoft\Windows NT\CurrentVersion\Terminal Server\Install\Software\Microsoft\Windows\CurrentVersion\RunOnce`,
		`Microsoft\Windows\CurrentVersion\Policies\Explorer\Run`,
		`Microsoft\Windows NT\CurrentVersion\Winlogon\Shell`,
		`Microsoft\Windows NT\CurrentVersion\Winlogon\Userinit`,
		`Microsoft\Windows NT\CurrentVersion\Winlogon\Notify`,
	}

	for _, path := range runKeyPaths {
		key := h.FindKey(path)
		if key != nil {
			for _, val := range key.Values {
				results = append(results, map[string]interface{}{
					"key_path":  path,
					"value_name": val.Name,
					"data":       formatRegistryData(val),
					"data_type":  dataTypeName(val.DataType),
					"timestamp":  key.Timestamp.Format(time.RFC3339),
				})
			}
		}
	}

	return results
}

// ExtractUserAssist extracts UserAssist entries (recently executed programs).
func (h *RegistryHive) ExtractUserAssist() []map[string]interface{} {
	var results []map[string]interface{}

	// UserAssist is under NTUSER.DAT\Software\Microsoft\Windows\CurrentVersion\Explorer\UserAssist
	userAssistKey := h.FindKey(`Software\Microsoft\Windows\CurrentVersion\Explorer\UserAssist`)
	if userAssistKey == nil {
		return results
	}

	for _, subkey := range userAssistKey.Subkeys {
		for _, val := range subkey.Values {
			// UserAssist values contain ROT13-encoded paths with execution count
			results = append(results, map[string]interface{}{
				"key_path":  subkey.Path,
				"value_name": val.Name,
				"data":       formatRegistryData(val),
				"data_type":  dataTypeName(val.DataType),
			})
		}
	}

	return results
}

// ExtractInstalledSoftware extracts installed software list.
func (h *RegistryHive) ExtractInstalledSoftware() []map[string]interface{} {
	var results []map[string]interface{}

	uninstallKey := h.FindKey(`Microsoft\Windows\CurrentVersion\Uninstall`)
	if uninstallKey == nil {
		return results
	}

	for _, subkey := range uninstallKey.Subkeys {
		entry := map[string]interface{}{
			"name": subkey.Name,
			"path": subkey.Path,
		}
		for _, val := range subkey.Values {
			switch strings.ToLower(val.Name) {
			case "displayname":
				entry["display_name"] = string(val.Data)
			case "displayversion":
				entry["version"] = string(val.Data)
			case "publisher":
				entry["publisher"] = string(val.Data)
			case "installdate":
				entry["install_date"] = string(val.Data)
			}
		}
		results = append(results, entry)
	}

	return results
}

// ExtractServices extracts services information.
func (h *RegistryHive) ExtractServices() []map[string]interface{} {
	var results []map[string]interface{}

	servicesKey := h.FindKey(`ControlSet001\Services`)
	if servicesKey == nil {
		return results
	}

	for _, subkey := range servicesKey.Subkeys {
		entry := map[string]interface{}{
			"name": subkey.Name,
			"path": subkey.Path,
		}
		for _, val := range subkey.Values {
			switch strings.ToLower(val.Name) {
			case "imagepath":
				entry["image_path"] = string(val.Data)
			case "start":
				entry["start_type"] = uint32(val.Data[0])
			case "type":
				entry["type"] = uint32(val.Data[0])
			case "errorcontrol":
				entry["error_control"] = uint32(val.Data[0])
			}
		}
		results = append(results, entry)
	}

	return results
}

// ─── Helper Functions ────────────────────────────────────────────────────────

func isASCII(data []byte) bool {
	for _, b := range data {
		if b > 127 {
			return false
		}
	}
	return true
}

func decodeUTF16LE(data []byte) string {
	if len(data) < 2 {
		return ""
	}
	// Pad if odd length
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	u16s := make([]uint16, len(data)/2)
	for i := range u16s {
		u16s[i] = binary.LittleEndian.Uint16(data[i*2:])
	}
	return string(utf16.Decode(u16s))
}

func formatRegistryData(val *ParsedValue) string {
	switch val.DataType {
	case TypeString, TypeExpandString:
		s := string(val.Data)
		s = strings.TrimRight(s, "\x00")
		return s
	case TypeDWORD:
		if len(val.Data) >= 4 {
			return fmt.Sprintf("0x%08X (%d)", binary.LittleEndian.Uint32(val.Data), binary.LittleEndian.Uint32(val.Data))
		}
	case TypeBinary:
		if len(val.Data) <= 64 {
			return hex.EncodeToString(val.Data)
		}
		return hex.EncodeToString(val.Data[:64]) + "..."
	case TypeMultiString:
		return strings.ReplaceAll(string(val.Data), "\x00", " | ")
	}
	return hex.EncodeToString(val.Data)
}

func fileTimeToTime(ft uint64) time.Time {
	if ft == 0 {
		return time.Time{}
	}
	nsec := int64(ft) * 100
	return time.Date(1601, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(nsec))
}

// ParseHiveFromBytes is an alias for ParseHive.
func ParseHiveFromBytes(data []byte, hiveType string) (*RegistryHive, error) {
	return ParseHive(data)
}

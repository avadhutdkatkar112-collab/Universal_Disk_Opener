// Package mft parses the NTFS Master File Table ($MFT) for forensic file timeline analysis.
// Extracts MACB timestamps, file names, and metadata from raw MFT records.
package mft

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

// MFT Record signatures
const (
	mftSignature    = "FILE"
	attributeEnd    = 0xFFFFFFFF
	attrStandard    = 0x10
	attrFileName    = 0x30
	attrData        = 0x80
	attrBitmap      = 0xB0
	attrINDX        = 0xC0
)

// MFTRecord represents a single MFT file record (1024 bytes).
type MFTRecord struct {
	Signature       [4]byte  // "FILE"
	UpdateSequenceOffset uint16
	UpdateSequenceSize   uint16
	LogfileSequenceNumber uint64
	SequenceNumber       uint16
	HardLinkCount        uint16
	AttributeOffset      uint16
	Flags                uint16   // 0x01 = file, 0x02 = directory
	RecordSize           uint32
	AttributeListSize    uint32
	BaseRecordRecordID   uint32
	NextAttributeID      uint16
	Reserved             uint16
	MFTRecordNumber      uint32
	FileName             string
	ParentDirectory      uint64

	// MACB Timestamps
	CreationTime    time.Time
	ModificationTime time.Time
	MFTChangeTime   time.Time
	AccessTime      time.Time

	// File attributes
	Size            uint64
	FileAttributes  uint32
	IsDirectory     bool
	IsHidden        bool
	IsSystem        bool
	IsReadOnly      bool
}

// StandardInfoAttribute (0x10) contains MACB timestamps.
type StandardInfoAttribute struct {
	CreationTime       uint64
	ModificationTime   uint64
	MFTChangeTime      uint64
	AccessTime         uint64
	FileAttributes     uint32
	Reserved           uint32
	MaximumVersion     uint32
	VersionNumber      uint32
	ClassID            uint32
	OwnerID            uint32
	SecurityID         uint32
	QuotaCharged       uint64
	LastAccessUSN      uint64
}

// FileNameAttribute (0x30) contains file name and parent directory.
type FileNameAttribute struct {
	ParentDirectoryRecordID uint64  // Low 48 bits = MFT record, High 16 bits = sequence
	CreationTime            uint64
	ModificationTime        uint64
	MFTChangeTime           uint64
	AccessTime              uint64
	AllocatedSize           uint64
	RealSize                uint64
	Flags                   uint32
	FileNameLength          uint8
	FileNameNamespace       uint8
	FileName                string
}

// MFTParser is the main parser structure.
type MFTParser struct {
	data     []byte
	records  []*MFTRecord
	TotalRecords int
	StartTime    time.Time
	EndTime      time.Time
}

// ParseMFT parses an MFT file from bytes.
func ParseMFT(data []byte) (*MFTParser, error) {
	if len(data) < 1024 {
		return nil, fmt.Errorf("MFT too small: %d bytes (need at least 1024)", len(data))
	}

	// Verify signature
	if string(data[0:4]) != mftSignature {
		return nil, fmt.Errorf("invalid MFT signature: %s", string(data[0:4]))
	}

	parser := &MFTParser{
		data: data,
	}

	// Parse records (each is 1024 bytes)
	recordSize := 1024
	numRecords := len(data) / recordSize
	parser.TotalRecords = numRecords

	for i := 0; i < numRecords; i++ {
		offset := i * recordSize
		record, err := parseRecord(data[offset:])
		if err != nil {
			continue
		}
		parser.records = append(parser.records, record)
	}

	// Determine time range
	if len(parser.records) > 0 {
		parser.StartTime = parser.records[0].CreationTime
		parser.EndTime = parser.records[0].ModificationTime
		for _, r := range parser.records[1:] {
			if !r.CreationTime.IsZero() && r.CreationTime.Before(parser.StartTime) {
				parser.StartTime = r.CreationTime
			}
			if !r.ModificationTime.IsZero() && r.ModificationTime.After(parser.EndTime) {
				parser.EndTime = r.ModificationTime
			}
		}
	}

	return parser, nil
}

func parseRecord(data []byte) (*MFTRecord, error) {
	if len(data) < 1024 {
		return nil, fmt.Errorf("record too small")
	}

	// Check signature
	if string(data[0:4]) != mftSignature {
		return nil, fmt.Errorf("invalid record signature")
	}

	record := &MFTRecord{
		UpdateSequenceOffset: binary.LittleEndian.Uint16(data[4:6]),
		UpdateSequenceSize:   binary.LittleEndian.Uint16(data[6:8]),
		LogfileSequenceNumber: binary.LittleEndian.Uint64(data[8:16]),
		SequenceNumber:       binary.LittleEndian.Uint16(data[16:18]),
		HardLinkCount:        binary.LittleEndian.Uint16(data[18:20]),
		AttributeOffset:      binary.LittleEndian.Uint16(data[20:22]),
		Flags:                binary.LittleEndian.Uint16(data[22:24]),
		RecordSize:           binary.LittleEndian.Uint32(data[24:28]),
		AttributeListSize:    binary.LittleEndian.Uint32(data[28:32]),
		BaseRecordRecordID:   binary.LittleEndian.Uint32(data[32:36]),
		NextAttributeID:      binary.LittleEndian.Uint16(data[36:38]),
		MFTRecordNumber:      binary.LittleEndian.Uint32(data[44:48]),
	}

	record.IsDirectory = record.Flags&0x02 != 0

	// Parse attributes
	attrOffset := uint32(record.AttributeOffset)
	for attrOffset < 1024-8 {
		attrType := binary.LittleEndian.Uint32(data[attrOffset:])
		if attrType == attributeEnd || attrType == 0 {
			break
		}
		attrSize := binary.LittleEndian.Uint32(data[attrOffset+4 : attrOffset+8])
		if attrSize < 24 || attrSize > 1024 {
			break
		}

		switch attrType {
		case attrStandard:
			if attrSize >= 48 {
				si := parseStandardInfo(data[attrOffset+8:])
				record.CreationTime = fileTimeToTime(si.CreationTime)
				record.ModificationTime = fileTimeToTime(si.ModificationTime)
				record.MFTChangeTime = fileTimeToTime(si.MFTChangeTime)
				record.AccessTime = fileTimeToTime(si.AccessTime)
				record.FileAttributes = si.FileAttributes
				record.IsHidden = si.FileAttributes&0x04 != 0
				record.IsSystem = si.FileAttributes&0x06 != 0
				record.IsReadOnly = si.FileAttributes&0x01 != 0
			}
		case attrFileName:
			if attrSize >= 66 {
				fn := parseFileName(data[attrOffset+8:])
				record.FileName = fn.FileName
				record.Size = fn.RealSize
				// Extract parent directory MFT record number
				record.ParentDirectory = fn.ParentDirectoryRecordID & 0xFFFFFFFFFFFF
			}
		}

		attrOffset += attrSize
		// Align to 8-byte boundary
		if attrOffset%8 != 0 {
			attrOffset += 8 - (attrOffset % 8)
		}
	}

	return record, nil
}

func parseStandardInfo(data []byte) StandardInfoAttribute {
	return StandardInfoAttribute{
		CreationTime:       binary.LittleEndian.Uint64(data[0:8]),
		ModificationTime:   binary.LittleEndian.Uint64(data[8:16]),
		MFTChangeTime:      binary.LittleEndian.Uint64(data[16:24]),
		AccessTime:         binary.LittleEndian.Uint64(data[24:32]),
		FileAttributes:     binary.LittleEndian.Uint32(data[32:36]),
	}
}

func parseFileName(data []byte) FileNameAttribute {
	fn := FileNameAttribute{
		ParentDirectoryRecordID: binary.LittleEndian.Uint64(data[0:8]),
		CreationTime:            binary.LittleEndian.Uint64(data[8:16]),
		ModificationTime:        binary.LittleEndian.Uint64(data[16:24]),
		MFTChangeTime:           binary.LittleEndian.Uint64(data[24:32]),
		AccessTime:              binary.LittleEndian.Uint64(data[32:40]),
		AllocatedSize:           binary.LittleEndian.Uint64(data[40:48]),
		RealSize:                binary.LittleEndian.Uint64(data[48:56]),
		Flags:                   binary.LittleEndian.Uint32(data[56:60]),
		FileNameLength:          data[60],
		FileNameNamespace:       data[61],
	}

	// Parse file name (UTF-16LE)
	nameBytes := data[62:]
	if int(fn.FileNameLength)*2 <= len(nameBytes) {
		fn.FileName = decodeUTF16LE(nameBytes[:fn.FileNameLength*2])
	}

	return fn
}

// GetFiles returns all file records.
func (p *MFTParser) GetFiles() []*MFTRecord {
	return p.records
}

// GetDirectories returns all directory records.
func (p *MFTParser) GetDirectories() []*MFTRecord {
	var dirs []*MFTRecord
	for _, r := range p.records {
		if r.IsDirectory {
			dirs = append(dirs, r)
		}
	}
	return dirs
}

// GetModifiedAfter returns files modified after a given time.
func (p *MFTParser) GetModifiedAfter(t time.Time) []*MFTRecord {
	var results []*MFTRecord
	for _, r := range p.records {
		if !r.ModificationTime.IsZero() && r.ModificationTime.After(t) {
			results = append(results, r)
		}
	}
	return results
}

// GetCreatedAfter returns files created after a given time.
func (p *MFTParser) GetCreatedAfter(t time.Time) []*MFTRecord {
	var results []*MFTRecord
	for _, r := range p.records {
		if !r.CreationTime.IsZero() && r.CreationTime.After(t) {
			results = append(results, r)
		}
	}
	return results
}

// GetRecentFiles returns files with MACB timestamps within a time range.
func (p *MFTParser) GetRecentFiles(start, end time.Time) []*MFTRecord {
	var results []*MFTRecord
	for _, r := range p.records {
		if r.ModificationTime.After(start) && r.ModificationTime.Before(end) {
			results = append(results, r)
		}
	}
	return results
}

// GetTimeline returns all timestamps for a file in chronological order.
func (r *MFTRecord) GetTimeline() []time.Time {
	var times []time.Time
	if !r.AccessTime.IsZero() {
		times = append(times, r.AccessTime)
	}
	if !r.CreationTime.IsZero() {
		times = append(times, r.CreationTime)
	}
	if !r.ModificationTime.IsZero() {
		times = append(times, r.ModificationTime)
	}
	if !r.MFTChangeTime.IsZero() {
		times = append(times, r.MFTChangeTime)
	}
	return times
}

// GetSummary returns a summary of parsed MFT records.
func (p *MFTParser) GetSummary() map[string]interface{} {
	summary := map[string]interface{}{
		"total_records": len(p.records),
		"start_time":    p.StartTime.Format(time.RFC3339),
		"end_time":      p.EndTime.Format(time.RFC3339),
	}

	// Count files vs directories
	files := 0
	dirs := 0
	for _, r := range p.records {
		if r.IsDirectory {
			dirs++
		} else {
			files++
		}
	}
	summary["files"] = files
	summary["directories"] = dirs

	return summary
}

func fileTimeToTime(ft uint64) time.Time {
	if ft == 0 {
		return time.Time{}
	}
	nsec := int64(ft) * 100
	return time.Date(1601, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(nsec))
}

func decodeUTF16LE(data []byte) string {
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
	return strings.TrimRight(string(runes), "\x00")
}

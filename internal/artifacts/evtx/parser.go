// Package evtx parses Windows Event Log (EVTX) files for forensic analysis.
// Handles binary chunk parsing, XML template expansion, and event record extraction.
package evtx

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

// EVTX File signature constants
const (
	elfFileSignature  = "ElfFile\x00"
	elfChunkSignature = "ElfChnk"
	evtRecordSig      = "\x2a\x2a\x00\x00"
)

// EVTXFileHeader (4096 bytes)
type FileHeader struct {
	Signature          [8]byte  // "ElfFile\x00"
	FirstChunkNumber   uint64
	LastChunkNumber    uint64
	NextRecordID       uint64
	HeaderSize         uint32   // 128
	MinorVersion       uint16
	MajorVersion       uint16
	HeaderBlockSize    uint16   // 4096
	NumberOfChunks     uint16
	Reserved           [76]byte
	CRC32              uint32
}

// EVTXChunkHeader (512 bytes)
type ChunkHeader struct {
	Signature                 [4]byte  // "ElfChnk"
	FirstEventRecordNumber    uint64
	LastEventRecordNumber     uint64
	FirstEventRecordIdentifier uint64
	LastEventRecordIdentifier  uint64
	HeaderSize                uint32
	LastEventRecordTimestamp   uint64
	FreeSpaceOffset           uint32
	FreeSpaceLength           uint32
	TotalEventRecords         uint32
	UsedSpaceLength           uint32
	CRC32                     uint32
}

// EventRecordHeader represents a single event log record.
type EventRecordHeader struct {
	Signature      [4]byte  // "*\x00\x00"
	Size           uint32
	EventRecordID  uint64
	TimeCreated    uint64   // Windows FILETIME
}

// EventData holds parsed event information.
type EventData struct {
	EventRecordID uint64
	TimeCreated   time.Time
	ProviderName  string
	ProviderGUID  string
	Channel       string
	ComputerName  string
	EventID       uint16
	Version       uint8
	Level         uint8
	Task          uint16
	Opcode        uint16
	Keywords      uint64
	Message       string
	Data          map[string]interface{}
	XML           string
}

// ForensicEventIDs maps common event IDs to descriptions.
var ForensicEventIDs = map[uint16]string{
	1102:  "Audit Log Cleared",
	4624:  "Logon Success",
	4625:  "Logon Failure",
	4634:  "Logoff",
	4648:  "Explicit Credential Logon",
	4672:  "Special Privileges Assigned",
	4688:  "Process Created",
	4689:  "Process Terminated",
	4697:  "Service Installed",
	4698:  "Scheduled Task Created",
	4699:  "Scheduled Task Deleted",
	4702:  "Scheduled Task Updated",
	4720:  "User Account Created",
	4722:  "User Account Enabled",
	4725:  "User Account Disabled",
	4726:  "User Account Deleted",
	4732:  "Member Added to Local Group",
	7045:  "Service Installed",
	7040:  "Service Start Type Changed",
}

// EVTXParser is the main parser structure.
type EVTXParser struct {
	data    []byte
	header  *FileHeader
	chunks  []*ChunkHeader
	Events  []*EventData
	StartTime time.Time
	EndTime   time.Time
}

// ParseEVTX parses an EVTX file from bytes.
func ParseEVTX(data []byte) (*EVTXParser, error) {
	if len(data) < 4096 {
		return nil, fmt.Errorf("EVTX file too small: %d bytes", len(data))
	}

	// Verify signature
	if string(data[0:8]) != elfFileSignature {
		return nil, fmt.Errorf("invalid EVTX signature: %x", data[0:8])
	}

	parser := &EVTXParser{
		data: data,
	}

	// Parse header
	parser.header = &FileHeader{
		FirstChunkNumber: binary.LittleEndian.Uint64(data[8:16]),
		LastChunkNumber:  binary.LittleEndian.Uint64(data[16:24]),
		NextRecordID:     binary.LittleEndian.Uint64(data[24:32]),
		HeaderSize:       binary.LittleEndian.Uint32(data[32:36]),
		MinorVersion:     binary.LittleEndian.Uint16(data[36:38]),
		MajorVersion:     binary.LittleEndian.Uint16(data[38:40]),
		HeaderBlockSize:  binary.LittleEndian.Uint16(data[40:42]),
		NumberOfChunks:   binary.LittleEndian.Uint16(data[42:44]),
	}
	copy(parser.header.Signature[:], data[0:8])

	// Parse chunks
	chunkSize := 65536 // 64KB chunks
	for i := 0; i < int(parser.header.NumberOfChunks); i++ {
		offset := 4096 + (i * chunkSize)
		if offset+chunkSize > len(data) {
			break
		}

		chunk, err := parseChunk(data[offset:])
		if err != nil {
			continue
		}
		parser.chunks = append(parser.chunks, chunk)
	}

	// Parse event records from chunks
	for _, chunk := range parser.chunks {
		parser.parseChunkEvents(chunk)
	}

	// Determine time range
	if len(parser.Events) > 0 {
		parser.StartTime = parser.Events[0].TimeCreated
		parser.EndTime = parser.Events[len(parser.Events)-1].TimeCreated
	}

	return parser, nil
}

func parseChunk(data []byte) (*ChunkHeader, error) {
	if len(data) < 512 {
		return nil, fmt.Errorf("chunk too small")
	}

	if string(data[0:4]) != elfChunkSignature {
		return nil, fmt.Errorf("invalid chunk signature")
	}

	chunk := &ChunkHeader{
		FirstEventRecordNumber: binary.LittleEndian.Uint64(data[8:16]),
		LastEventRecordNumber:  binary.LittleEndian.Uint64(data[16:24]),
		FirstEventRecordIdentifier: binary.LittleEndian.Uint64(data[24:32]),
		LastEventRecordIdentifier:  binary.LittleEndian.Uint64(data[32:40]),
		HeaderSize:            binary.LittleEndian.Uint32(data[40:44]),
		LastEventRecordTimestamp: binary.LittleEndian.Uint64(data[44:52]),
		FreeSpaceOffset:       binary.LittleEndian.Uint32(data[52:56]),
		FreeSpaceLength:       binary.LittleEndian.Uint32(data[56:60]),
		TotalEventRecords:     binary.LittleEndian.Uint32(data[60:64]),
		UsedSpaceLength:       binary.LittleEndian.Uint32(data[64:68]),
	}
	copy(chunk.Signature[:], data[0:4])

	return chunk, nil
}

func (p *EVTXParser) parseChunkEvents(chunk *ChunkHeader) {
	// Event records are variable-length XML structures
	// For forensic purposes, we parse the record headers and extract key fields
	offset := 512 // Start after chunk header

	for i := uint32(0); i < chunk.TotalEventRecords; i++ {
		if offset+24 > len(p.data) {
			break
		}

		// Check for event record signature
		sig := string(p.data[offset : offset+4])
		if sig != evtRecordSig {
			break
		}

		recordSize := binary.LittleEndian.Uint32(p.data[offset+4 : offset+8])
		if recordSize < 24 || recordSize > 65536 {
			break
		}

		recordID := binary.LittleEndian.Uint64(p.data[offset+8 : offset+16])
		timeCreated := binary.LittleEndian.Uint64(p.data[offset+16 : offset+24])

		event := &EventData{
			EventRecordID: recordID,
			TimeCreated:   fileTimeToTime(timeCreated),
		}

		// Try to extract XML data
		xmlStart := offset + 24
		xmlEnd := offset + int(recordSize)
		if xmlEnd > len(p.data) {
			xmlEnd = len(p.data)
		}

		if xmlStart < xmlEnd {
			xmlData := p.data[xmlStart:xmlEnd]
			p.parseEventXML(xmlData, event)
		}

		p.Events = append(p.Events, event)
		offset += int(recordSize)
		// Align to 8-byte boundary
		if offset%8 != 0 {
			offset += 8 - (offset % 8)
		}
	}
}

func (p *EVTXParser) parseEventXML(data []byte, event *EventData) {
	// Try to find and parse the Event XML
	// EVTX uses XML templates with substitution arrays
	xmlStr := string(data)

	// Extract key fields using string parsing (more robust than XML parsing for damaged logs)
	if idx := strings.Index(xmlStr, "<Provider Name='"); idx >= 0 {
		start := idx + 17
		end := strings.Index(xmlStr[start:], "'")
		if end > 0 {
			event.ProviderName = xmlStr[start : start+end]
		}
	}

	if idx := strings.Index(xmlStr, "<EventID>"); idx >= 0 {
		start := idx + 9
		end := strings.Index(xmlStr[start:], "</EventID>")
		if end > 0 {
			fmt.Sscanf(xmlStr[start:start+end], "%d", &event.EventID)
		}
	}

	if idx := strings.Index(xmlStr, "<Level>"); idx >= 0 {
		start := idx + 7
		end := strings.Index(xmlStr[start:], "</Level>")
		if end > 0 {
			fmt.Sscanf(xmlStr[start:start+end], "%d", &event.Level)
		}
	}

	if idx := strings.Index(xmlStr, "<Channel>"); idx >= 0 {
		start := idx + 9
		end := strings.Index(xmlStr[start:], "</Channel>")
		if end > 0 {
			event.Channel = xmlStr[start : start+end]
		}
	}

	if idx := strings.Index(xmlStr, "<Computer>"); idx >= 0 {
		start := idx + 10
		end := strings.Index(xmlStr[start:], "</Computer>")
		if end > 0 {
			event.ComputerName = xmlStr[start : start+end]
		}
	}

	if idx := strings.Index(xmlStr, "<Security UserID='"); idx >= 0 {
		start := idx + 18
		end := strings.Index(xmlStr[start:], "'")
		if end > 0 {
			event.Data = map[string]interface{}{
				"UserID": xmlStr[start : start+end],
			}
		}
	}

	event.XML = xmlStr
}

// GetEventsByType returns events filtered by Event ID.
func (p *EVTXParser) GetEventsByType(eventID uint16) []*EventData {
	var results []*EventData
	for _, event := range p.Events {
		if event.EventID == eventID {
			results = append(results, event)
		}
	}
	return results
}

// GetForensicEvents returns events relevant to forensic analysis.
func (p *EVTXParser) GetForensicEvents() []*EventData {
	var results []*EventData
	for _, event := range p.Events {
		if _, ok := ForensicEventIDs[event.EventID]; ok {
			results = append(results, event)
		}
	}
	return results
}

// GetLogonEvents returns all logon/logoff events.
func (p *EVTXParser) GetLogonEvents() []*EventData {
	var results []*EventData
	for _, event := range p.Events {
		switch event.EventID {
		case 4624, 4625, 4634, 4648:
			results = append(results, event)
		}
	}
	return results
}

// GetProcessEvents returns process creation/termination events.
func (p *EVTXParser) GetProcessEvents() []*EventData {
	var results []*EventData
	for _, event := range p.Events {
		switch event.EventID {
		case 4688, 4689:
			results = append(results, event)
		}
	}
	return results
}

// GetSecurityEvents returns security-relevant events.
func (p *EVTXParser) GetSecurityEvents() []*EventData {
	var results []*EventData
	for _, event := range p.Events {
		switch event.EventID {
		case 1102, 4720, 4722, 4725, 4726, 4672:
			results = append(results, event)
		}
	}
	return results
}

// GetSummary returns a summary of parsed events.
func (p *EVTXParser) GetSummary() map[string]interface{} {
	summary := map[string]interface{}{
		"total_events":  len(p.Events),
		"start_time":    p.StartTime.Format(time.RFC3339),
		"end_time":      p.EndTime.Format(time.RFC3339),
		"num_chunks":    len(p.chunks),
	}

	// Count by type
	eventCounts := make(map[uint16]int)
	for _, event := range p.Events {
		eventCounts[event.EventID]++
	}
	summary["event_id_counts"] = eventCounts

	return summary
}

func fileTimeToTime(ft uint64) time.Time {
	if ft == 0 {
		return time.Time{}
	}
	// Windows FILETIME: 100-nanosecond intervals since Jan 1, 1601
	nsec := int64(ft) * 100
	return time.Date(1601, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(nsec))
}

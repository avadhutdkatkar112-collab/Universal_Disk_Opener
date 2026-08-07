package vault

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type AuditRecord struct {
	Index        uint64            `json:"index"`
	PreviousHash string            `json:"previous_hash"`
	Timestamp    string            `json:"timestamp"`
	Actor        string            `json:"actor"`
	Action       string            `json:"action"`
	EvidenceID   string            `json:"evidence_id"`
	Metadata     map[string]string `json:"metadata"`
	RecordHash   string            `json:"record_hash"`
}

type AuditLogger struct {
	filePath string
	mu       sync.Mutex
	lastHash string
	lastIdx  uint64
}

func NewAuditLogger(filePath string) (*AuditLogger, error) {
	logger := &AuditLogger{
		filePath: filePath,
		lastHash: "0000000000000000000000000000000000000000000000000000000000000000",
		lastIdx:  0,
	}

	if _, err := os.Stat(filePath); err == nil {
		valid, count, lastH, _, err := VerifyAuditChain(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load audit chain: %w", err)
		}
		if !valid {
			return nil, fmt.Errorf("CRITICAL: Audit chain at %s has been tampered with!", filePath)
		}
		logger.lastIdx = count
		logger.lastHash = lastH
	}

	return logger, nil
}

func (a *AuditLogger) LogRecord(actor, action, evidenceID string, metadata map[string]string) (*AuditRecord, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	idx := a.lastIdx + 1
	ts := time.Now().UTC().Format(time.RFC3339)

	metaJSON, _ := json.Marshal(metadata)
	payload := fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s", idx, a.lastHash, ts, actor, action, evidenceID, string(metaJSON))

	h := sha256.Sum256([]byte(payload))
	recordHash := hex.EncodeToString(h[:])

	record := AuditRecord{
		Index:        idx,
		PreviousHash: a.lastHash,
		Timestamp:    ts,
		Actor:        actor,
		Action:       action,
		EvidenceID:   evidenceID,
		Metadata:     metadata,
		RecordHash:   recordHash,
	}

	data, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(a.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return nil, err
	}

	a.lastIdx = idx
	a.lastHash = recordHash
	return &record, nil
}

func VerifyAuditChain(filePath string) (bool, uint64, string, []AuditRecord, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, 0, "", nil, err
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	lastHash := "0000000000000000000000000000000000000000000000000000000000000000"
	var count uint64 = 0
	var records []AuditRecord

	for dec.More() {
		var rec AuditRecord
		if err := dec.Decode(&rec); err != nil {
			return false, count, "", records, err
		}

		count++
		if rec.Index != count {
			return false, count, "", records, fmt.Errorf("sequence broken at record %d", count)
		}
		if rec.PreviousHash != lastHash {
			return false, count, "", records, fmt.Errorf("hash chain mismatch at record %d", count)
		}

		metaJSON, _ := json.Marshal(rec.Metadata)
		payload := fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s", rec.Index, rec.PreviousHash, rec.Timestamp, rec.Actor, rec.Action, rec.EvidenceID, string(metaJSON))
		h := sha256.Sum256([]byte(payload))
		computedHash := hex.EncodeToString(h[:])

		if computedHash != rec.RecordHash {
			return false, count, "", records, fmt.Errorf("tampering detected in record %d", count)
		}

		lastHash = rec.RecordHash
		records = append(records, rec)
	}

	return true, count, lastHash, records, nil
}

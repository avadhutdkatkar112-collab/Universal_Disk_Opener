package vault_test

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/user/vhd-opener/internal/vault"
)

func TestDVE1_E2E_LifecycleAndTamperDetection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dve1_e2e_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourcePath := filepath.Join(tempDir, "synthetic_disk.raw")
	evidenceData := make([]byte, 10*1024*1024)
	if _, err := rand.Read(evidenceData); err != nil {
		t.Fatalf("Failed to generate random evidence data: %v", err)
	}
	if err := os.WriteFile(sourcePath, evidenceData, 0600); err != nil {
		t.Fatalf("Failed to write synthetic evidence: %v", err)
	}

	caseDir := filepath.Join(tempDir, "TEST-CASE-001.case")
	passphrase := "CorrectHorseBatteryStaple2026!"
	caseID := "CASE-2026-E2E"

	manifest, err := vault.IngestEvidenceFile(sourcePath, caseDir, passphrase, caseID)
	if err != nil {
		t.Fatalf("Ingestion failed: %v", err)
	}

	if manifest.TotalSize != int64(len(evidenceData)) {
		t.Errorf("Expected size %d, got %d", len(evidenceData), manifest.TotalSize)
	}

	if manifest.SourceSHA256 == "" {
		t.Error("Manifest has empty SourceSHA256")
	}

	if manifest.TotalChunks != 3 {
		t.Errorf("Expected 3 chunks for 10 MiB with 4 MiB chunk size, got %d", manifest.TotalChunks)
	}

	evidenceDir := filepath.Join(caseDir, "evidence")
	chunkFiles, _ := filepath.Glob(filepath.Join(evidenceDir, "*.chunk"))
	if len(chunkFiles) != 3 {
		t.Errorf("Expected 3 chunk files on disk, got %d", len(chunkFiles))
	}

	auditPath := filepath.Join(caseDir, "audit", "audit.log")
	if _, err := os.Stat(auditPath); os.IsNotExist(err) {
		t.Error("Audit log not created during ingestion")
	}

	valid, count, _, records, err := vault.VerifyAuditChain(auditPath)
	if err != nil {
		t.Fatalf("Audit chain verification failed: %v", err)
	}
	if !valid {
		t.Error("Audit chain invalid on pristine container")
	}
	if count != 1 {
		t.Errorf("Expected 1 audit record from ingestion, got %d", count)
	}
	if len(records) != 1 {
		t.Errorf("Expected 1 record in slice, got %d", len(records))
	}

	logger, err := vault.NewAuditLogger(auditPath)
	if err != nil {
		t.Fatalf("Failed to initialize audit logger: %v", err)
	}

	_, err = logger.LogRecord("Examiner_01", "evidence.mount", caseID, map[string]string{
		"mode": "read_only",
	})
	if err != nil {
		t.Fatalf("Failed to log second audit record: %v", err)
	}

	valid, count, _, _, err = vault.VerifyAuditChain(auditPath)
	if err != nil || !valid {
		t.Fatalf("Audit chain verification failed after second record: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 audit records, got %d", count)
	}

	extractPath := filepath.Join(tempDir, "extracted.raw")
	err = vault.ExtractEvidenceFile(caseDir, extractPath, passphrase)
	if err != nil {
		t.Fatalf("Extraction failed on pristine container: %v", err)
	}

	extractedData, err := os.ReadFile(extractPath)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}

	if !bytesEqual(extractedData, evidenceData) {
		t.Error("Extracted data does not match original evidence")
	}

	payloadFiles, _ := filepath.Glob(filepath.Join(evidenceDir, "*.chunk"))
	if len(payloadFiles) > 0 {
		firstChunk, err := os.ReadFile(payloadFiles[0])
		if err == nil && len(firstChunk) > 100 {
			firstChunk[100] ^= 0xFF
			os.WriteFile(payloadFiles[0], firstChunk, 0600)

			err = vault.ExtractEvidenceFile(caseDir, filepath.Join(tempDir, "tampered.raw"), passphrase)
			if err == nil {
				t.Error("Expected extraction failure due to ciphertext tampering, but succeeded")
			}

			firstChunk[100] ^= 0xFF
			os.WriteFile(payloadFiles[0], firstChunk, 0600)
		}
	}

	auditData, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("Failed to read audit log: %v", err)
	}

	tamperedAudit := append(auditData, []byte("TAMPERED_RECORD\n")...)
	if err := os.WriteFile(auditPath, tamperedAudit, 0600); err != nil {
		t.Fatalf("Failed to write tampered audit log: %v", err)
	}

	valid, _, _, _, _ = vault.VerifyAuditChain(auditPath)
	if valid {
		t.Error("Audit log verification passed despite injected tampered record")
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

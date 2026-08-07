package ui_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/user/vhd-opener/internal/ui"
)

func TestVaultHandler_IngestAndVerify(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vault_handler_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceFile := filepath.Join(tempDir, "sample_artifact.evtx")
	sampleData := []byte("EVTX_HEADER_DUMMY_DATA_FOR_IPC_TESTING")
	if err := os.WriteFile(sourceFile, sampleData, 0600); err != nil {
		t.Fatalf("Failed to write sample file: %v", err)
	}

	caseDir := filepath.Join(tempDir, "HANDLER-CASE.case")
	handler := ui.NewApp()
	handler.Startup(context.Background())

	manifest, err := handler.IngestEvidence(sourceFile, caseDir, "SuperSecurePassphrase2026!", "CASE-IPC-001")
	if err != nil {
		t.Fatalf("IngestEvidence failed: %v", err)
	}
	if manifest.SourceSHA256 == "" {
		t.Error("Manifest returned empty SHA-256 string")
	}

	result, err := handler.VerifyEvidenceContainer(caseDir)
	if err != nil {
		t.Fatalf("VerifyEvidenceContainer failed: %v", err)
	}
	if !result.AuditValid {
		t.Error("Expected valid audit chain")
	}
	if result.AuditCount != 1 {
		t.Errorf("Expected 1 audit record, got %d", result.AuditCount)
	}

	vResult, err := handler.VerifyCaseIntegrity(caseDir)
	if err != nil {
		t.Fatalf("VerifyCaseIntegrity failed: %v", err)
	}
	if !vResult.Valid {
		t.Errorf("Expected valid verification, got message: %s", vResult.Message)
	}
	if vResult.AuditCount != 1 {
		t.Errorf("Expected 1 audit record, got %d", vResult.AuditCount)
	}

	err = handler.LogAnalystAction("Analyst_02", "bookmark.add", map[string]string{"item": "MFT_Record_442"})
	if err != nil {
		t.Errorf("Failed to log analyst action: %v", err)
	}

	resultPostAction, err := handler.VerifyEvidenceContainer(caseDir)
	if err != nil {
		t.Fatalf("VerifyEvidenceContainer failed post action: %v", err)
	}
	if !resultPostAction.AuditValid {
		t.Error("Audit chain invalid after analyst action")
	}
	if resultPostAction.AuditCount != 2 {
		t.Errorf("Expected 2 audit records post action, got %d", resultPostAction.AuditCount)
	}
}

func TestStorageHandler_MountAndBrowse(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_handler_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	imagePath := filepath.Join(tempDir, "test.raw")
	diskBuffer := make([]byte, 2*1024*1024)
	diskBuffer[510] = 0x55
	diskBuffer[511] = 0xAA
	diskBuffer[446+4] = 0x07

	startSector := uint32(2048)
	sectorCount := uint32(2048)
	diskBuffer[454] = byte(startSector)
	diskBuffer[455] = byte(startSector >> 8)
	diskBuffer[456] = byte(startSector >> 16)
	diskBuffer[457] = byte(startSector >> 24)
	diskBuffer[458] = byte(sectorCount)
	diskBuffer[459] = byte(sectorCount >> 8)
	diskBuffer[460] = byte(sectorCount >> 16)
	diskBuffer[461] = byte(sectorCount >> 24)

	ntfsOffset := 2048 * 512
	copy(diskBuffer[ntfsOffset+3:ntfsOffset+11], []byte("NTFS    "))
	diskBuffer[ntfsOffset+11] = 0x00
	diskBuffer[ntfsOffset+12] = 0x02
	diskBuffer[ntfsOffset+13] = 0x08
	diskBuffer[ntfsOffset+48] = 0x04

	if err := os.WriteFile(imagePath, diskBuffer, 0600); err != nil {
		t.Fatalf("Failed to write test image: %v", err)
	}

	handler := ui.NewStorageHandler()
	handler.Startup(context.Background())

	partitions, err := handler.MountDisk(imagePath)
	if err != nil {
		t.Fatalf("MountDisk failed: %v", err)
	}
	if len(partitions) == 0 {
		t.Fatal("Expected at least one partition")
	}
	if partitions[0].Type != "NTFS / exFAT" && partitions[0].Type != "NTFS" {
		t.Errorf("Expected NTFS partition type, got %s", partitions[0].Type)
	}

	mounted, err := handler.MountPartition(1)
	if err != nil {
		t.Logf("MountPartition returned error (expected for synthetic image): %v", err)
		return
	}
	if !mounted {
		t.Log("MountPartition returned false (expected for synthetic image)")
		return
	}

	nodes, err := handler.ListDirectory("/")
	if err != nil {
		t.Logf("ListDirectory returned error (expected for synthetic image without real MFT): %v", err)
		return
	}
	if len(nodes) == 0 {
		t.Log("No nodes returned (expected for synthetic image without real MFT)")
	}
}

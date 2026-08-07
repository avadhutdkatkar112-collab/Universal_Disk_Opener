package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/user/vhd-opener/internal/memory"
	"github.com/user/vhd-opener/internal/vault"
)

type VaultResult struct {
	Manifest   *vault.Manifest `json:"manifest"`
	AuditValid bool            `json:"audit_valid"`
	AuditCount uint64          `json:"audit_count"`
	ChunksOK   bool            `json:"chunks_ok"`
}

type IngestRequest struct {
	SourcePath string `json:"source_path"`
	CaseDir    string `json:"case_dir"`
	Passphrase string `json:"passphrase"`
	CaseID     string `json:"case_id"`
	Actor      string `json:"actor"`
}

type VerificationResult struct {
	Valid           bool   `json:"valid"`
	AuditCount      uint64 `json:"audit_count"`
	ManifestPresent bool   `json:"manifest_present"`
	SourceHash      string `json:"source_hash"`
	Message         string `json:"message"`
}

type ExportReportRequest struct {
	CaseDir      string `json:"caseDir"`
	ExaminerName string `json:"examinerName"`
	OutputPath   string `json:"outputPath"`
}

type ExportReportResponse struct {
	ReportPath   string `json:"reportPath"`
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

var vaultMu sync.Mutex

func (a *App) IngestEvidence(srcPath, outputDir, passphrase, vaultID string) (*vault.Manifest, error) {
	a.activeCaseDir = outputDir
	return vault.IngestEvidenceFile(srcPath, outputDir, passphrase, vaultID)
}

func (a *App) IngestEvidenceStruct(req IngestRequest) (*vault.Manifest, error) {
	a.activeCaseDir = req.CaseDir
	manifest, err := vault.IngestEvidenceFile(req.SourcePath, req.CaseDir, req.Passphrase, req.CaseID)
	if err != nil {
		return nil, err
	}

	auditPath := filepath.Join(req.CaseDir, "audit", "audit.log")
	logger, err := vault.NewAuditLogger(auditPath)
	if err == nil {
		logger.LogRecord(req.Actor, "evidence.ingest", req.CaseID, map[string]string{
			"source_path":   req.SourcePath,
			"source_sha256": manifest.SourceSHA256,
			"total_bytes":   fmt.Sprintf("%d", manifest.TotalSize),
		})
	}

	return manifest, nil
}

func (a *App) VerifyEvidenceContainer(caseDir string) (*VaultResult, error) {
	manifestPath := filepath.Join(caseDir, "manifest.json")
	manifest, err := vault.LoadManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("manifest load failed: %w", err)
	}

	auditPath := filepath.Join(caseDir, "audit", "audit.log")
	valid, count, _, _, err := vault.VerifyAuditChain(auditPath)
	if err != nil {
		valid = false
	}

	evidenceDir := filepath.Join(caseDir, "evidence")
	chunkCount := 0
	entries, _ := filepath.Glob(filepath.Join(evidenceDir, "*.chunk"))
	chunkCount = len(entries)

	chunksOK := int64(chunkCount) == manifest.TotalChunks

	return &VaultResult{
		Manifest:   manifest,
		AuditValid: valid,
		AuditCount: count,
		ChunksOK:   chunksOK,
	}, nil
}

func (a *App) VerifyCaseIntegrity(caseDir string) (VerificationResult, error) {
	manifestPath := filepath.Join(caseDir, "manifest.json")
	auditPath := filepath.Join(caseDir, "audit", "audit.log")

	manifest, err := vault.LoadManifest(manifestPath)
	if err != nil {
		return VerificationResult{Valid: false, Message: "Failed to load DVE1 manifest"}, err
	}

	valid, count, _, _, err := vault.VerifyAuditChain(auditPath)
	if err != nil || !valid {
		return VerificationResult{Valid: false, Message: "Audit log tampering or link breakage detected!"}, err
	}

	return VerificationResult{
		Valid:           true,
		AuditCount:      count,
		ManifestPresent: true,
		SourceHash:      manifest.SourceSHA256,
		Message:         "Container verified. Chain of custody hash linkages intact.",
	}, nil
}

func (a *App) LogAnalystAction(actor, action string, metadata map[string]string) error {
	vaultMu.Lock()
	defer vaultMu.Unlock()

	caseDir := "."
	if a.activeCaseDir != "" {
		caseDir = a.activeCaseDir
	}
	auditPath := filepath.Join(caseDir, "audit", "audit.log")
	logger, err := vault.NewAuditLogger(auditPath)
	if err != nil {
		return err
	}
	_, err = logger.LogRecord(actor, action, "", metadata)
	return err
}

func (a *App) ExportChainOfCustodyReport(req ExportReportRequest) (*ExportReportResponse, error) {
	vaultMu.Lock()
	defer vaultMu.Unlock()

	outPath := req.OutputPath
	if outPath == "" {
		outPath = filepath.Join(req.CaseDir, "reports", "chain_of_custody_report.html")
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return &ExportReportResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to create report directory: %v", err),
		}, nil
	}

	reportFile, err := vault.GenerateReportHTML(req.CaseDir, req.ExaminerName, outPath)
	if err != nil {
		return &ExportReportResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &ExportReportResponse{
		ReportPath: reportFile,
		Success:    true,
	}, nil
}

func (a *App) GetMemorySnapshot() (map[string]interface{}, error) {
	procs, err := memory.EnumerateLiveProcesses()
	if err != nil {
		return nil, err
	}

	yaraMatches := memory.ScanProcessMemory(procs)

	suspiciousCount := 0
	for _, p := range procs {
		if p.IsSuspicious {
			suspiciousCount++
		}
	}

	return map[string]interface{}{
		"total_processes": len(procs),
		"suspicious":      suspiciousCount,
		"yara_matches":    len(yaraMatches),
		"processes":       procs,
		"matches":         yaraMatches,
	}, nil
}

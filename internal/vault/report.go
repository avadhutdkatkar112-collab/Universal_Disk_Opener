package vault

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"
)

type ReportData struct {
	CaseID       string
	ReportDate   string
	Manifest     *Manifest
	AuditRecords []AuditRecord
	ChainValid   bool
	AuditCount   int
	ReportSHA256 string
	GeneratedBy  string
}

const htmlReportTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Forensic Chain-of-Custody Report - {{.CaseID}}</title>
    <style>
        @media print {
            body { background: #fff !important; color: #000 !important; }
            .no-print { display: none !important; }
        }
        body { font-family: 'Segoe UI', Arial, sans-serif; margin: 40px; color: #222; background: #fff; line-height: 1.5; }
        .header { border-bottom: 2px solid #003366; padding-bottom: 10px; margin-bottom: 20px; }
        .header h1 { margin: 0; color: #003366; font-size: 24px; text-transform: uppercase; }
        .header p { margin: 5px 0 0 0; color: #666; font-size: 13px; }
        .badge { display: inline-block; padding: 4px 8px; border-radius: 4px; font-weight: bold; font-size: 12px; }
        .badge-success { background-color: #e6f4ea; color: #137333; border: 1px solid #ceedd6; }
        .badge-danger { background-color: #fce8e6; color: #c5221f; border: 1px solid #fad2cf; }
        .section { margin-bottom: 25px; }
        .section-title { font-size: 16px; color: #003366; border-bottom: 1px solid #ddd; padding-bottom: 4px; margin-bottom: 12px; font-weight: bold; }
        table { width: 100%; border-collapse: collapse; margin-top: 8px; font-size: 13px; }
        th, td { border: 1px solid #dcdcdc; padding: 8px 12px; text-align: left; }
        th { background-color: #f4f6f8; color: #333; font-weight: 600; }
        tr:nth-child(even) { background-color: #fafafa; }
        .hash-code { font-family: 'Consolas', 'Courier New', monospace; font-size: 11px; word-break: break-all; background: #f0f0f0; padding: 2px 4px; border-radius: 3px; }
        .footer { margin-top: 40px; border-top: 1px solid #ddd; padding-top: 10px; font-size: 11px; color: #777; text-align: center; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Forensic Chain-of-Custody Report</h1>
        <p>Generated on {{.ReportDate}} | Issued by: {{.GeneratedBy}}</p>
    </div>

    <div class="section">
        <div class="section-title">Case Overview & Integrity Verdict</div>
        <table>
            <tr><th style="width: 200px;">Case Identifier</th><td><strong>{{.CaseID}}</strong></td></tr>
            <tr>
                <th>Audit Chain Verification</th>
                <td>
                    {{if .ChainValid}}
                        <span class="badge badge-success">&#10004; VERIFIED INTEGRITY (0 Tamper Events Detected)</span>
                    {{else}}
                        <span class="badge badge-danger">&#10008; INTEGRITY VIOLATION DETECTED</span>
                    {{end}}
                </td>
            </tr>
            <tr><th>Total Recorded Audit Entries</th><td>{{.AuditCount}}</td></tr>
        </table>
    </div>

    <div class="section">
        <div class="section-title">Evidence Container Manifest (DVE1)</div>
        <table>
            <tr><th style="width: 200px;">Vault ID</th><td>{{.Manifest.VaultID}}</td></tr>
            <tr><th>Format Version</th><td>{{.Manifest.Magic}} v{{.Manifest.FormatVersion}}</td></tr>
            <tr><th>Ingestion Timestamp</th><td>{{.Manifest.CreatedAt}}</td></tr>
            <tr><th>Logical Size</th><td>{{.Manifest.TotalSize}} bytes ({{.Manifest.TotalChunks}} chunks)</td></tr>
            <tr><th>KDF / Cipher Parameters</th><td>Argon2id (Salt: <span class="hash-code">{{printf "%x" .Manifest.Salt}}</span>) | {{.Manifest.Cipher}} ({{.Manifest.ChunkSize}} byte chunks)</td></tr>
            <tr><th>Master Source SHA-256</th><td><span class="hash-code">{{.Manifest.SourceSHA256}}</span></td></tr>
        </table>
    </div>

    <div class="section">
        <div class="section-title">Cryptographic Audit Log (Chain of Custody)</div>
        <table>
            <thead>
                <tr><th style="width: 40px;">#</th><th style="width: 180px;">Timestamp (UTC)</th><th style="width: 120px;">Actor</th><th style="width: 130px;">Action</th><th>Record Hash</th></tr>
            </thead>
            <tbody>
                {{range .AuditRecords}}
                <tr>
                    <td>{{.Index}}</td>
                    <td>{{.Timestamp}}</td>
                    <td>{{.Actor}}</td>
                    <td><strong>{{.Action}}</strong></td>
                    <td><span class="hash-code">{{.RecordHash}}</span></td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>

    <div class="footer">
        <p>Report SHA-256: <span class="hash-code">{{.ReportSHA256}}</span></p>
        <p>Confidential Forensic Evidence Document - Meteor Forensic Workbench</p>
    </div>
</body>
</html>`

func GenerateReportHTML(caseDir, examinerName, outputPath string) (string, error) {
	manifest, err := LoadManifest(filepath.Join(caseDir, "manifest.json"))
	if err != nil {
		return "", fmt.Errorf("failed to read manifest: %w", err)
	}

	auditPath := filepath.Join(caseDir, "audit", "audit.log")
	valid, count, _, records, err := VerifyAuditChain(auditPath)
	if err != nil {
		valid = false
	}

	data := ReportData{
		CaseID:       manifest.VaultID,
		ReportDate:   time.Now().UTC().Format(time.RFC3339),
		Manifest:     manifest,
		AuditRecords: records,
		ChainValid:   valid,
		AuditCount:   int(count),
		GeneratedBy:  examinerName,
	}

	tmpl, err := template.New("report").Parse(htmlReportTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	hasher := sha256.New()
	hasher.Write(buf.Bytes())
	data.ReportSHA256 = hex.EncodeToString(hasher.Sum(nil))

	var finalBuf bytes.Buffer
	if err := tmpl.Execute(&finalBuf, data); err != nil {
		return "", fmt.Errorf("failed to execute final template: %w", err)
	}

	if err := os.WriteFile(outputPath, finalBuf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("failed to write report: %w", err)
	}

	return outputPath, nil
}

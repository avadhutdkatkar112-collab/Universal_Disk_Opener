package intelligence

import (
	"fmt"
	"strings"
	"time"

	"github.com/user/vhd-opener/internal/timeline"
)

type ThreatLevel string

const (
	LevelCritical ThreatLevel = "CRITICAL"
	LevelHigh     ThreatLevel = "HIGH"
	LevelMedium   ThreatLevel = "MEDIUM"
	LevelInfo     ThreatLevel = "INFO"
)

type Finding struct {
	Level       ThreatLevel `json:"level"`
	Category    string      `json:"category"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Source      string      `json:"source"`
	Path        string      `json:"path"`
	Timestamp   time.Time   `json:"timestamp"`
	FormattedTS string      `json:"formatted_ts"`
}

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (a *Analyzer) AnalyzeTimeline(entries []timeline.TimelineEntry) []Finding {
	findings := make([]Finding, 0)

	for _, entry := range entries {
		switch entry.Source {
		case timeline.SourceRegistry:
			if f, ok := a.checkRegistryPersistence(entry); ok {
				findings = append(findings, f)
			}
		case timeline.SourceEVTX:
			if f, ok := a.checkEVTXEvents(entry); ok {
				findings = append(findings, f)
			}
		case timeline.SourceMFT:
			if f, ok := a.checkMFTAnomalies(entry); ok {
				findings = append(findings, f)
			}
		}
	}

	return findings
}

func (a *Analyzer) checkRegistryPersistence(entry timeline.TimelineEntry) (Finding, bool) {
	pathLower := strings.ToLower(entry.Path)
	descLower := strings.ToLower(entry.Description)

	isRunKey := strings.Contains(pathLower, "\\run") || strings.Contains(pathLower, "\\runonce")
	isService := strings.Contains(pathLower, "\\services\\")

	if isRunKey || isService {
		suspiciousTokens := []string{"powershell", "cmd.exe", "wscript", "cscript", "appdata\\local\\temp", "bitsadmin", "mshta", "regsvr32", "rundll32"}
		for _, token := range suspiciousTokens {
			if strings.Contains(descLower, token) {
				return Finding{
					Level:       LevelCritical,
					Category:    "Persistence",
					Title:       "Suspicious Registry Auto-Start / Service Entry",
					Description: fmt.Sprintf("Registry entry references suspicious tool/path '%s': %s", token, entry.Description),
					Source:      string(entry.Source),
					Path:        entry.Path,
					Timestamp:   entry.Timestamp,
					FormattedTS: entry.Timestamp.UTC().Format("2006-01-02 15:04:05"),
				}, true
			}
		}

		return Finding{
			Level:       LevelMedium,
			Category:    "Persistence",
			Title:       "Auto-Start / Service Registry Modification",
			Description: fmt.Sprintf("Modification detected in startup hive path: %s", entry.Description),
			Source:      string(entry.Source),
			Path:        entry.Path,
			Timestamp:   entry.Timestamp,
			FormattedTS: entry.Timestamp.UTC().Format("2006-01-02 15:04:05"),
		}, true
	}

	return Finding{}, false
}

func (a *Analyzer) checkEVTXEvents(entry timeline.TimelineEntry) (Finding, bool) {
	desc := entry.Description

	if strings.Contains(desc, "1102") || strings.Contains(strings.ToLower(desc), "cleared") {
		return Finding{
			Level:       LevelCritical,
			Category:    "Anti-Forensics",
			Title:       "Audit Log Cleared (Event ID 1102)",
			Description: "The Windows Audit Log was cleared, potentially masking adversary activity.",
			Source:      string(entry.Source),
			Path:        entry.Path,
			Timestamp:   entry.Timestamp,
			FormattedTS: entry.Timestamp.UTC().Format("2006-01-02 15:04:05"),
		}, true
	}

	if strings.Contains(desc, "7045") {
		return Finding{
			Level:       LevelHigh,
			Category:    "Execution / Persistence",
			Title:       "New Service Created (Event ID 7045)",
			Description: fmt.Sprintf("A new service was installed on the system: %s", desc),
			Source:      string(entry.Source),
			Path:        entry.Path,
			Timestamp:   entry.Timestamp,
			FormattedTS: entry.Timestamp.UTC().Format("2006-01-02 15:04:05"),
		}, true
	}

	if strings.Contains(desc, "4688") {
		return Finding{
			Level:       LevelInfo,
			Category:    "Process Execution",
			Title:       "Process Created (Event ID 4688)",
			Description: desc,
			Source:      string(entry.Source),
			Path:        entry.Path,
			Timestamp:   entry.Timestamp,
			FormattedTS: entry.Timestamp.UTC().Format("2006-01-02 15:04:05"),
		}, true
	}

	if strings.Contains(desc, "4624") || strings.Contains(desc, "4625") {
		return Finding{
			Level:       LevelInfo,
			Category:    "Authentication",
			Title:       "Logon Activity Detected",
			Description: desc,
			Source:      string(entry.Source),
			Path:        entry.Path,
			Timestamp:   entry.Timestamp,
			FormattedTS: entry.Timestamp.UTC().Format("2006-01-02 15:04:05"),
		}, true
	}

	return Finding{}, false
}

func (a *Analyzer) checkMFTAnomalies(entry timeline.TimelineEntry) (Finding, bool) {
	pathLower := strings.ToLower(entry.Path)
	descLower := strings.ToLower(entry.Description)

	if strings.Contains(descLower, ".exe") || strings.Contains(descLower, ".dll") || strings.Contains(descLower, ".bat") {
		if strings.Contains(pathLower, "temp") || strings.Contains(pathLower, "public") || strings.Contains(pathLower, "appdata\\local\\temp") {
			return Finding{
				Level:       LevelHigh,
				Category:    "Defense Evasion",
				Title:       "Executable Dropped in Temp Directory",
				Description: fmt.Sprintf("Binary file detected in temporary location: %s", entry.Description),
				Source:      string(entry.Source),
				Path:        entry.Path,
				Timestamp:   entry.Timestamp,
				FormattedTS: entry.Timestamp.UTC().Format("2006-01-02 15:04:05"),
			}, true
		}
	}

	return Finding{}, false
}

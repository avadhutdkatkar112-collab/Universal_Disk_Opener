package reporting

import (
	"bytes"
	"html/template"
	"os"
	"time"

	"github.com/user/vhd-opener/internal/forensic"
	"github.com/user/vhd-opener/internal/forensic/sigma"
	"github.com/user/vhd-opener/internal/forensic/timeline"
)

type ReportData struct {
	CaseName            string
	Investigator        string
	GeneratedAt         string
	TotalTimelineEvents int
	SigmaAlerts         []sigma.Alert
	SuspiciousProcesses []forensic.ProcessInfo
	YaraMatches         []forensic.YaraMatch
	TimelineSample      []timeline.TimelineEntry
}

func CompileReport(caseName, investigator string, entries []timeline.TimelineEntry, alerts []sigma.Alert, procs []forensic.ProcessInfo, yaraMatches []forensic.YaraMatch) (string, error) {
	suspiciousProcs := make([]forensic.ProcessInfo, 0)
	for _, p := range procs {
		if p.IsSuspicious {
			suspiciousProcs = append(suspiciousProcs, p)
		}
	}

	sampleSize := 20
	if len(entries) < sampleSize {
		sampleSize = len(entries)
	}
	sampleTimeline := entries[:sampleSize]

	data := ReportData{
		CaseName:            caseName,
		Investigator:        investigator,
		GeneratedAt:         time.Now().Format("2006-01-02 15:04:05 MST"),
		TotalTimelineEvents: len(entries),
		SigmaAlerts:         alerts,
		SuspiciousProcesses: suspiciousProcs,
		YaraMatches:         yaraMatches,
		TimelineSample:      sampleTimeline,
	}

	tmpl, err := template.New("executive_report").Parse(ExecutiveReportTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func ExportReportToFile(filePath, htmlContent string) error {
	return os.WriteFile(filePath, []byte(htmlContent), 0644)
}

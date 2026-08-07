package api

import (
	"github.com/user/vhd-opener/internal/forensic"
	"github.com/user/vhd-opener/internal/reporting"
	"github.com/user/vhd-opener/internal/forensic/sigma"
	"github.com/user/vhd-opener/internal/forensic/timeline"
)

func (a *App) GenerateReport(caseName, investigator string, entries []timeline.TimelineEntry, alerts []sigma.Alert, procs []forensic.ProcessInfo, yaraMatches []forensic.YaraMatch) (string, error) {
	return reporting.CompileReport(caseName, investigator, entries, alerts, procs, yaraMatches)
}

func (a *App) SaveReportToFile(filePath, htmlContent string) error {
	return reporting.ExportReportToFile(filePath, htmlContent)
}

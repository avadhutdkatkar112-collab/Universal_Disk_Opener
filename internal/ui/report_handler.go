package ui

import (
	"github.com/user/vhd-opener/internal/memory"
	"github.com/user/vhd-opener/internal/report"
	"github.com/user/vhd-opener/internal/sigma"
	"github.com/user/vhd-opener/internal/timeline"
)

func (a *App) GenerateReport(caseName, investigator string, entries []timeline.TimelineEntry, alerts []sigma.Alert, procs []memory.ProcessInfo, yaraMatches []memory.YaraMatch) (string, error) {
	return report.CompileReport(caseName, investigator, entries, alerts, procs, yaraMatches)
}

func (a *App) SaveReportToFile(filePath, htmlContent string) error {
	return report.ExportReportToFile(filePath, htmlContent)
}

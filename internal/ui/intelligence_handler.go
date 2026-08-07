package ui

import (
	"github.com/user/vhd-opener/internal/intelligence"
	"github.com/user/vhd-opener/internal/timeline"
)

// AnalyzeFindings runs the forensic intelligence analyzer on timeline entries.
func (a *App) AnalyzeFindings(entries []timeline.TimelineEntry) ([]intelligence.Finding, error) {
	analyzer := intelligence.NewAnalyzer()
	findings := analyzer.AnalyzeTimeline(entries)
	return findings, nil
}

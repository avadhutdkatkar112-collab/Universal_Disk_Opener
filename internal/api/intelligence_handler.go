package api

import (
	"github.com/user/vhd-opener/internal/forensic/search"
	"github.com/user/vhd-opener/internal/forensic/timeline"
)

// AnalyzeFindings runs the forensic intelligence analyzer on timeline entries.
func (a *App) AnalyzeFindings(entries []timeline.TimelineEntry) ([]search.Finding, error) {
	analyzer := search.NewAnalyzer()
	findings := analyzer.AnalyzeTimeline(entries)
	return findings, nil
}

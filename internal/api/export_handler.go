package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"

	"github.com/user/vhd-opener/internal/forensic/timeline"
)

// ExportTimelineCSV saves normalized timeline entries to a local CSV file.
func (a *App) ExportTimelineCSV(entries []timeline.TimelineEntry, outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed creating CSV file: %w", err)
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// Write CSV Header
	header := []string{"Timestamp (UTC)", "Source", "Event Type", "Title", "Description", "Path"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed writing CSV header: %w", err)
	}

	// Write Data Rows
	for _, entry := range entries {
		row := []string{
			entry.Timestamp.UTC().Format("2006-01-02T15:04:05.000Z"),
			string(entry.Source),
			string(entry.EventType),
			entry.Title,
			entry.Description,
			entry.Path,
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed writing CSV row: %w", err)
		}
	}

	return nil
}

// ExportTimelineJSON exports normalized timeline entries to formatted JSON.
func (a *App) ExportTimelineJSON(entries []timeline.TimelineEntry, outputPath string) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed serializing timeline to JSON: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed writing JSON file: %w", err)
	}

	return nil
}

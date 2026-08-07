package ui

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/user/vhd-opener/internal/timeline"
	_ "github.com/mattn/go-sqlite3"
)

type SQLQueryResult struct {
	Columns []string                 `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
	Count   int                      `json:"count"`
	TimeMs  float64                  `json:"time_ms"`
}

var timelineDB *sql.DB

func (a *App) getTimelineDB() (*sql.DB, error) {
	if timelineDB != nil {
		return timelineDB, nil
	}

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS timeline_events (
			id TEXT PRIMARY KEY,
			timestamp DATETIME,
			source TEXT,
			event_type TEXT,
			title TEXT,
			description TEXT,
			path TEXT
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	timelineDB = db
	return timelineDB, nil
}

func (a *App) LoadTimelineToSQL(entries []timeline.TimelineEntry) (int, error) {
	db, err := a.getTimelineDB()
	if err != nil {
		return 0, err
	}

	_, err = db.Exec("DELETE FROM timeline_events")
	if err != nil {
		return 0, fmt.Errorf("failed to clear table: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	stmt, err := tx.Prepare("INSERT INTO timeline_events (id, timestamp, source, event_type, title, description, path) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	defer stmt.Close()

	count := 0
	for _, entry := range entries {
		_, err := stmt.Exec(
			entry.ID,
			entry.Timestamp.UTC().Format(time.RFC3339),
			string(entry.Source),
			string(entry.EventType),
			entry.Title,
			entry.Description,
			entry.Path,
		)
		if err != nil {
			continue
		}
		count++
	}

	return count, tx.Commit()
}

func (a *App) ExecuteSQLQuery(query string) (*SQLQueryResult, error) {
	db, err := a.getTimelineDB()
	if err != nil {
		return nil, err
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("empty query")
	}

	upperQuery := strings.ToUpper(query)
	if strings.HasPrefix(upperQuery, "INSERT") || strings.HasPrefix(upperQuery, "UPDATE") || strings.HasPrefix(upperQuery, "DELETE") || strings.HasPrefix(upperQuery, "DROP") || strings.HasPrefix(upperQuery, "CREATE") {
		return nil, fmt.Errorf("only SELECT queries are allowed")
	}

	start := time.Now()

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var resultRows []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			switch v := val.(type) {
			case []byte:
				row[col] = string(v)
			case time.Time:
				row[col] = v.Format(time.RFC3339)
			default:
				row[col] = v
			}
		}
		resultRows = append(resultRows, row)
	}

	elapsed := float64(time.Since(start).Microseconds()) / 1000.0

	return &SQLQueryResult{
		Columns: columns,
		Rows:    resultRows,
		Count:   len(resultRows),
		TimeMs:  elapsed,
	}, nil
}

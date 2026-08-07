package memory

import (
	"strings"
)

type YaraMatch struct {
	RuleName    string   `json:"rule_name"`
	PID         int32    `json:"pid"`
	ProcessName string   `json:"process_name"`
	MatchedData string   `json:"matched_data"`
	Severity    string   `json:"severity"`
	Tags        []string `json:"tags"`
}

type YaraSignature struct {
	Name     string
	Keywords []string
	Severity string
	Tags     []string
}

func GetBuiltinSignatures() []YaraSignature {
	return []YaraSignature{
		{
			Name:     "CobaltStrike_Beacon_Buffer",
			Keywords: []string{"reflectiveloader", "beacon.dll", "vssadmin delete", "amsiinitfailed"},
			Severity: "CRITICAL",
			Tags:     []string{"c2", "fileless", "execution"},
		},
		{
			Name:     "Process_Hollowing_SuspiciousCmd",
			Keywords: []string{"-nop -w hidden -encodedcommand", "downloadstring('http", "invoke-expression", "iex (new-object"},
			Severity: "HIGH",
			Tags:     []string{"defense_evasion", "t1055"},
		},
		{
			Name:     "Credential_Dumping_Tool",
			Keywords: []string{"sekurlsa", "lsass", "sam", "ntds.dit", "mimikatz", "procdump -ma"},
			Severity: "CRITICAL",
			Tags:     []string{"credential_access", "t1003"},
		},
		{
			Name:     "Lateral_Movement_PsExec",
			Keywords: []string{"psexec", "\\pipe\\svcctl", "remotecommand", "schtasks /create"},
			Severity: "HIGH",
			Tags:     []string{"lateral_movement", "t1021"},
		},
		{
			Name:     "Ransomware_ShadowCopy",
			Keywords: []string{"vssadmin delete shadows", "wmic shadowcopy delete", "bcdedit /set {default} recoveryenabled no"},
			Severity: "CRITICAL",
			Tags:     []string{"impact", "t1490"},
		},
		{
			Name:     "Persistence_RunKey",
			Keywords: []string{"\\software\\microsoft\\windows\\currentversion\\run", "\\software\\microsoft\\windows\\currentversion\\runonce"},
			Severity: "MEDIUM",
			Tags:     []string{"persistence", "t1547.001"},
		},
	}
}

func ScanProcessMemory(processes []ProcessInfo) []YaraMatch {
	matches := make([]YaraMatch, 0)
	signatures := GetBuiltinSignatures()

	for _, proc := range processes {
		targetText := strings.ToLower(proc.Name + " " + proc.CommandLine + " " + proc.Path)

		for _, sig := range signatures {
			for _, kw := range sig.Keywords {
				if strings.Contains(targetText, strings.ToLower(kw)) {
					matches = append(matches, YaraMatch{
						RuleName:    sig.Name,
						PID:         proc.PID,
						ProcessName: proc.Name,
						MatchedData: kw,
						Severity:    sig.Severity,
						Tags:        sig.Tags,
					})
					break
				}
			}
		}
	}

	return matches
}

package forensic

import (
	"fmt"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

type ProcessInfo struct {
	PID          int32    `json:"pid"`
	PPID         int32    `json:"ppid"`
	Name         string   `json:"name"`
	Path         string   `json:"path"`
	CommandLine  string   `json:"command_line"`
	Username     string   `json:"username"`
	CreateTime   string   `json:"create_time"`
	OpenSockets  []Socket `json:"open_sockets"`
	IsSuspicious bool     `json:"is_suspicious"`
	FlagReason   string   `json:"flag_reason"`
}

type Socket struct {
	Type       string `json:"type"`
	LocalAddr  string `json:"local_addr"`
	RemoteAddr string `json:"remote_addr"`
	State      string `json:"state"`
}

func EnumerateLiveProcesses() ([]ProcessInfo, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}

	connections, _ := net.Connections("all")
	connMap := make(map[int32][]Socket)

	for _, c := range connections {
		loc := fmt.Sprintf("%s:%d", c.Laddr.IP, c.Laddr.Port)
		rem := fmt.Sprintf("%s:%d", c.Raddr.IP, c.Raddr.Port)
		proto := "TCP"
		if c.Type == 2 {
			proto = "UDP"
		}

		connMap[c.Pid] = append(connMap[c.Pid], Socket{
			Type:       proto,
			LocalAddr:  loc,
			RemoteAddr: rem,
			State:      c.Status,
		})
	}

	results := make([]ProcessInfo, 0)

	for _, p := range procs {
		name, _ := p.Name()
		ppid, _ := p.Ppid()
		cmd, _ := p.Cmdline()
		exe, _ := p.Exe()
		user, _ := p.Username()
		cTimeMs, _ := p.CreateTime()

		cTime := time.Unix(cTimeMs/1000, 0).Format("2006-01-02 15:04:05")

		pInfo := ProcessInfo{
			PID:         p.Pid,
			PPID:        ppid,
			Name:        name,
			Path:        exe,
			CommandLine: cmd,
			Username:    user,
			CreateTime:  cTime,
			OpenSockets: connMap[p.Pid],
		}

		if isSuspiciousProcess(pInfo) {
			pInfo.IsSuspicious = true
			pInfo.FlagReason = "Execution from temporary path or suspicious parent"
		}

		results = append(results, pInfo)
	}

	return results, nil
}

func isSuspiciousProcess(p ProcessInfo) bool {
	target := strings.ToLower(p.Path)
	if target == "" {
		target = strings.ToLower(p.CommandLine)
	}

	suspiciousPaths := []string{
		"\\appdata\\local\\temp",
		"\\users\\public\\",
		"\\temp\\",
		"\\appdata\\local\\microsoft\\windows\\inetcache",
	}

	for _, sp := range suspiciousPaths {
		if strings.Contains(target, sp) {
			return true
		}
	}

	suspiciousNames := []string{
		"mimikatz", "lazagne", "procdump", "psexec", "nc.exe",
		"ncat.exe", "meterpreter", "cobalt", "beacon",
	}

	for _, sn := range suspiciousNames {
		if strings.Contains(strings.ToLower(p.Name), sn) {
			return true
		}
	}

	return false
}

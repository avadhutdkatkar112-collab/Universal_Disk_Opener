package api

import (
	"github.com/user/vhd-opener/internal/forensic"
)

func (a *App) GetLiveProcesses() ([]forensic.ProcessInfo, error) {
	return forensic.EnumerateLiveProcesses()
}

func (a *App) RunMemoryYaraScan(procs []forensic.ProcessInfo) ([]forensic.YaraMatch, error) {
	return forensic.ScanProcessMemory(procs), nil
}

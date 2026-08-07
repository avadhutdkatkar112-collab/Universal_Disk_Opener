package ui

import (
	"github.com/user/vhd-opener/internal/memory"
)

func (a *App) GetLiveProcesses() ([]memory.ProcessInfo, error) {
	return memory.EnumerateLiveProcesses()
}

func (a *App) RunMemoryYaraScan(procs []memory.ProcessInfo) ([]memory.YaraMatch, error) {
	return memory.ScanProcessMemory(procs), nil
}

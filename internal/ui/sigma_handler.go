package ui

import (
	"github.com/user/vhd-opener/internal/sigma"
	"github.com/user/vhd-opener/internal/timeline"
)

var sigmaEngine *sigma.Engine

func (a *App) getSigmaEngine() *sigma.Engine {
	if sigmaEngine == nil {
		sigmaEngine = sigma.NewEngine()
		sigmaEngine.AddEmbeddedDefaults()
	}
	return sigmaEngine
}

func (a *App) LoadSigmaRuleDirectory(dir string) (int, error) {
	engine := a.getSigmaEngine()
	_, err := engine.LoadRulesFromDir(dir)
	return len(engine.Rules), err
}

func (a *App) RunSigmaScan(entries []timeline.TimelineEntry) ([]sigma.Alert, error) {
	engine := a.getSigmaEngine()
	alerts := engine.ScanTimeline(entries)
	return alerts, nil
}

func (a *App) GetSigmaRuleCount() int {
	engine := a.getSigmaEngine()
	return len(engine.Rules)
}

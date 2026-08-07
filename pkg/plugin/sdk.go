package plugin

import (
	"github.com/user/vhd-opener/pkg/capability"
)

type ExtensionType string

const (
	ExtCapability   ExtensionType = "CAPABILITY"
	ExtPreviewer    ExtensionType = "PREVIEWER"
	ExtContextMenu  ExtensionType = "CONTEXT_MENU"
	ExtSearch       ExtensionType = "SEARCH_PROVIDER"
)

type ContextMenuItem struct {
	ID           string   `json:"id"`
	Label        string   `json:"label"`
	CapabilityID string   `json:"capability_id"`
	FileTypes    []string `json:"file_types"`
}

type Manifest struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Author      string            `json:"author"`
	Commands    []string          `json:"commands"`
	MenuItems   []ContextMenuItem `json:"menu_items"`
}

type Plugin interface {
	Manifest() Manifest
	RegisterCapabilities() []capability.Capability
	OnLoad(gatewayCtx any) error
	OnUnload() error
}

type Registry struct {
	plugins map[string]Plugin
}

func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]Plugin),
	}
}

func (r *Registry) Register(p Plugin) error {
	m := p.Manifest()
	r.plugins[m.ID] = p
	return p.OnLoad(nil)
}

func (r *Registry) Unregister(pluginID string) error {
	if p, ok := r.plugins[pluginID]; ok {
		err := p.OnUnload()
		delete(r.plugins, pluginID)
		return err
	}
	return nil
}

func (r *Registry) List() []Manifest {
	manifests := make([]Manifest, 0, len(r.plugins))
	for _, p := range r.plugins {
		manifests = append(manifests, p.Manifest())
	}
	return manifests
}

func (r *Registry) GetCapabilities() []capability.Capability {
	var caps []capability.Capability
	for _, p := range r.plugins {
		caps = append(caps, p.RegisterCapabilities()...)
	}
	return caps
}

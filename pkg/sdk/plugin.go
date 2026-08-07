// Package sdk provides the Plugin SDK for the Universal Disk Explorer.
// Third-party plugins register commands, suggestions, and handlers
// without modifying the core system codebase.
package sdk

import "context"

// Manifest describes a plugin's identity and metadata.
type Manifest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Author      string `json:"author"`
	Description string `json:"description"`
}

// CommandDeclaration defines a single command that a plugin exposes.
type CommandDeclaration struct {
	Verb        string   `json:"verb"`
	Target      string   `json:"target"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Examples    []string `json:"examples"`
}

// Plugin is the interface that all plugins must implement.
type Plugin interface {
	// Manifest returns the plugin's identity metadata.
	Manifest() Manifest

	// Commands returns the list of commands this plugin registers.
	Commands() []CommandDeclaration

	// Execute runs a specific command declared by this plugin.
	Execute(ctx context.Context, cmd CommandDeclaration, params map[string]string) (interface{}, error)
}

// BasePlugin provides a default implementation for plugins that only
// need to override specific methods.
type BasePlugin struct {
	ManifestData  Manifest
	CommandDecls  []CommandDeclaration
	ExecuteFunc   func(ctx context.Context, cmd CommandDeclaration, params map[string]string) (interface{}, error)
}

func (p *BasePlugin) Manifest() Manifest {
	return p.ManifestData
}

func (p *BasePlugin) Commands() []CommandDeclaration {
	return p.CommandDecls
}

func (p *BasePlugin) Execute(ctx context.Context, cmd CommandDeclaration, params map[string]string) (interface{}, error) {
	if p.ExecuteFunc != nil {
		return p.ExecuteFunc(ctx, cmd, params)
	}
	return nil, nil
}

// PluginInfo holds runtime info about a loaded plugin.
type PluginInfo struct {
	Manifest Manifest            `json:"manifest"`
	Commands []CommandDeclaration `json:"commands"`
	Enabled  bool                `json:"enabled"`
}

// PluginRegistry manages loaded plugins and their command registrations.
type PluginRegistry struct {
	plugins map[string]Plugin
}

// NewPluginRegistry creates an empty plugin registry.
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string]Plugin),
	}
}

// Register adds a plugin to the registry.
func (r *PluginRegistry) Register(p Plugin) {
	manifest := p.Manifest()
	r.plugins[manifest.ID] = p
}

// Unregister removes a plugin by ID.
func (r *PluginRegistry) Unregister(id string) {
	delete(r.plugins, id)
}

// Get returns a plugin by ID.
func (r *PluginRegistry) Get(id string) (Plugin, bool) {
	p, ok := r.plugins[id]
	return p, ok
}

// List returns all registered plugins.
func (r *PluginRegistry) List() []PluginInfo {
	var result []PluginInfo
	for _, p := range r.plugins {
		result = append(result, PluginInfo{
			Manifest: p.Manifest(),
			Commands: p.Commands(),
			Enabled:  true,
		})
	}
	return result
}

// GetAllCommands returns all commands from all registered plugins.
func (r *PluginRegistry) GetAllCommands() []PluginCommand {
	var result []PluginCommand
	for _, p := range r.plugins {
		manifest := p.Manifest()
		for _, cmd := range p.Commands() {
			result = append(result, PluginCommand{
				PluginID:  manifest.ID,
				PluginName: manifest.Name,
				Command:   cmd,
			})
		}
	}
	return result
}

// PluginCommand associates a command declaration with its parent plugin.
type PluginCommand struct {
	PluginID   string              `json:"plugin_id"`
	PluginName string              `json:"plugin_name"`
	Command    CommandDeclaration  `json:"command"`
}

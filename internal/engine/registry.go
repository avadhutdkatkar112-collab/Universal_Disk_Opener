// Package engine provides the universal disk engine — a decoupled registry
// of disk drivers and filesystem drivers. The engine itself knows nothing
// about Wails, React, or specific formats; it only dispatches to registered
// driver factories.
package engine

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/user/vhd-opener/internal/engine/core"
)

// Registry holds all registered disk driver factories and manages
// the lifecycle of opening/closing virtual disks.
type Registry struct {
	drivers map[string]core.DiskDriverFactory
	mu      sync.RWMutex
}

// NewRegistry creates an empty driver registry.
func NewRegistry() *Registry {
	return &Registry{
		drivers: make(map[string]core.DiskDriverFactory),
	}
}

// Register adds a disk driver factory for one or more file extensions.
// Extensions should include the dot (e.g. ".vhd").
func (r *Registry) Register(factory core.DiskDriverFactory, extensions ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ext := range extensions {
		r.drivers[strings.ToLower(ext)] = factory
	}
}

// Open opens a virtual disk file using the appropriate driver.
// It resolves the driver by file extension, then delegates to the factory.
func (r *Registry) Open(filePath string) (core.DiskDriver, error) {
	ext := filepath.Ext(filePath)
	r.mu.RLock()
	factory, exists := r.drivers[strings.ToLower(ext)]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("engine: unsupported format: %s", ext)
	}
	return factory(filePath)
}

// IsSupported checks if a file extension has a registered driver.
func (r *Registry) IsSupported(ext string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.drivers[strings.ToLower(ext)]
	return exists
}

// SupportedExtensions returns all registered file extensions.
func (r *Registry) SupportedExtensions() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	exts := make([]string, 0, len(r.drivers))
	for ext := range r.drivers {
		exts = append(exts, ext)
	}
	return exts
}

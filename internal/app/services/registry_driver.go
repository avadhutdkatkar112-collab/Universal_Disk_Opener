package services

import "sync"

type Registry struct {
	mu        sync.RWMutex
	factories map[string]DiskDriverFactory
}

func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]DiskDriverFactory),
	}
}

func (r *Registry) Register(factory DiskDriverFactory, extensions ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ext := range extensions {
		r.factories[ext] = factory
	}
}

func (r *Registry) GetFactory(extension string) (DiskDriverFactory, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.factories[extension]
	return f, ok
}

func (r *Registry) SupportedExtensions() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	exts := make([]string, 0, len(r.factories))
	for ext := range r.factories {
		exts = append(exts, ext)
	}
	return exts
}

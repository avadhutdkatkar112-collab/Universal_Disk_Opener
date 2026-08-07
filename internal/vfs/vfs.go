// Package vfs implements the Virtual Filesystem abstraction.
// All disk formats, partition parsers, and filesystem readers are hidden
// behind this layer. The UI never knows which format is in use.
package vfs

import (
	"io"
)

// VirtualFS is the interface that all virtual filesystems implement.
type VirtualFS interface {
	// ListDirectory lists the contents of a directory.
	ListDirectory(path string) ([]Entry, error)

	// GetEntry returns information about a single entry.
	GetEntry(path string) (*Entry, error)

	// ReadFile reads a file and returns a reader.
	ReadFile(path string) (io.ReadCloser, error)

	// Search searches for files matching a pattern.
	Search(query string, opts ...SearchOption) ([]SearchResult, error)

	// Root returns the root path of this filesystem.
	Root() string

	// Label returns a human-readable label (e.g., "NTFS", "FAT32").
	Label() string

	// Info returns filesystem metadata.
	Info() FSInfo
}

// Entry represents a file or directory.
type Entry struct {
	Name       string     `json:"name"`
	Path       string     `json:"path"`
	Size       int64      `json:"size"`
	IsDir      bool       `json:"isDir"`
	ModTime    int64      `json:"modTime"`
	Extension  string     `json:"extension"`
	Type       EntryType  `json:"type"`
	TargetPath string     `json:"targetPath,omitempty"` // symlink target
	Metadata   *EntryMeta `json:"metadata,omitempty"`
}

// EntryType represents the type of entry.
type EntryType string

const (
	EntryTypeFile   EntryType = "file"
	EntryTypeDir    EntryType = "directory"
	EntryTypeLink   EntryType = "link"
	EntryTypeStream EntryType = "stream"
)

// EntryMeta holds additional metadata.
type EntryMeta struct {
	Owner      string `json:"owner,omitempty"`
	Group      string `json:"group,omitempty"`
	Permissions string `json:"permissions,omitempty"`
	MimeType   string `json:"mimeType,omitempty"`
}

// FSInfo holds filesystem information.
type FSInfo struct {
	Type       string `json:"type"`
	Label      string `json:"label"`
	TotalSpace uint64 `json:"totalSpace"`
	FreeSpace  uint64 `json:"freeSpace"`
	UsedSpace  uint64 `json:"usedSpace"`
	Files      uint64 `json:"files"`
	Directories uint64 `json:"directories"`
}

// SearchResult represents a search result.
type SearchResult struct {
	Entry     Entry   `json:"entry"`
	Highlight string  `json:"highlight,omitempty"`
	Score     float64 `json:"score"`
}

// SearchOption configures a search.
type SearchOption func(*searchConfig)

type searchConfig struct {
	maxResults int
	fileTypes  []string
	minSize    int64
	maxSize    int64
	recursive  bool
}

// WithMaxResults limits the number of results.
func WithMaxResults(n int) SearchOption {
	return func(c *searchConfig) { c.maxResults = n }
}

// WithFileType filters by file type.
func WithFileType(types ...string) SearchOption {
	return func(c *searchConfig) { c.fileTypes = types }
}

// WithSizeRange filters by size.
func WithSizeRange(min, max int64) SearchOption {
	return func(c *searchConfig) { c.minSize = min; c.maxSize = max }
}

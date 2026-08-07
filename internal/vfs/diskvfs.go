// Package vfs provides a concrete VFS implementation backed by filesystem readers.
// It bridges the storage.VirtualDisk interface with the filesystems.Reader interface.
package vfs

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/user/vhd-opener/internal/storage"
	"github.com/user/vhd-opener/internal/storage/filesystems"
)

// DiskVFS implements VirtualFS using a storage.VirtualDisk and filesystems.Reader.
type DiskVFS struct {
	vdisk    storage.VirtualDisk
	reader   filesystems.Reader
	partStart uint64
	fsType   filesystems.FSType
}

// NewDiskVFS creates a new VFS for a partition on a disk.
func NewDiskVFS(vdisk storage.VirtualDisk, partitionStart uint64) (*DiskVFS, error) {
	adapter := filesystems.NewDiskAdapter(vdisk)
	fsType := filesystems.DetectFSType(adapter, partitionStart)

	reader, err := filesystems.NewReader(adapter, partitionStart)
	if err != nil {
		return nil, fmt.Errorf("vfs: cannot create filesystem reader for type %s: %w", fsType, err)
	}

	return &DiskVFS{
		vdisk:     vdisk,
		reader:    reader,
		partStart: partitionStart,
		fsType:    fsType,
	}, nil
}

// ListDirectory lists the contents of a directory.
func (v *DiskVFS) ListDirectory(rawPath string) ([]Entry, error) {
	adapter := filesystems.NewDiskAdapter(v.vdisk)
	posixPath := NormalizePath(rawPath)

	entries, err := v.reader.ListDirectory(adapter, v.partStart, posixPath)
	if err != nil {
		return nil, fmt.Errorf("vfs: list directory failed: %w", err)
	}

	result := make([]Entry, len(entries))
	for i, e := range entries {
		targetPath := ""
		if strings.HasPrefix(e.ID, "link:") {
			targetPath = e.ID[5:]
		}
		result[i] = Entry{
			Name:       e.Name,
			Path:       normalizePath(posixPath, e.Name, e.IsDirectory),
			Size:       e.Size,
			IsDir:      e.IsDirectory,
			ModTime:    e.ModifiedTime.Unix(),
			Type:       entryType(e.IsDirectory),
			Extension:  path.Ext(e.Name),
			TargetPath: targetPath,
		}
	}

	return result, nil
}

// GetEntry returns information about a single entry.
func (v *DiskVFS) GetEntry(path string) (*Entry, error) {
	return nil, fmt.Errorf("vfs: GetEntry not implemented")
}

// ReadFile reads a file and returns a reader.
func (v *DiskVFS) ReadFile(rawPath string) (io.ReadCloser, error) {
	adapter := filesystems.NewDiskAdapter(v.vdisk)
	posixPath := NormalizePath(rawPath)

	// Find the file entry by traversing the path
	parts := strings.Split(strings.Trim(posixPath, "/"), "/")
	currentPath := "/"
	var targetEntry *filesystems.FileEntry

	for idx, part := range parts {
		if part == "" {
			continue
		}

		entries, err := v.reader.ListDirectory(adapter, v.partStart, currentPath)
		if err != nil {
			return nil, fmt.Errorf("vfs: cannot list %s: %w", currentPath, err)
		}

		found := false
		for _, e := range entries {
			if e.Name == part {
				if strings.HasPrefix(e.ID, "link:") {
					symlinkTarget := e.ID[5:]
					remaining := strings.Join(parts[idx+1:], "/")
					var resolvedPath string
					if remaining != "" {
						resolvedPath = path.Clean(symlinkTarget + "/" + remaining)
					} else {
						resolvedPath = symlinkTarget
					}
					if !strings.HasPrefix(resolvedPath, "/") {
						resolvedPath = "/" + resolvedPath
					}
					return v.ReadFile(resolvedPath)
				}
				targetEntry = &e
				currentPath = path.Join(currentPath, part)
				if !strings.HasPrefix(currentPath, "/") {
					currentPath = "/" + currentPath
				}
				found = true
				break
			}
		}

		if !found {
			return nil, fmt.Errorf("vfs: file not found: %s", posixPath)
		}
	}

	if targetEntry == nil || targetEntry.IsDirectory {
		return nil, fmt.Errorf("vfs: not a file: %s", posixPath)
	}

	data, err := v.reader.GetFileContent(adapter, v.partStart, targetEntry)
	if err != nil {
		return nil, fmt.Errorf("vfs: cannot read file: %w", err)
	}

	return io.NopCloser(bytes.NewReader(data)), nil
}

// Search searches for files matching a query.
func (v *DiskVFS) Search(query string, opts ...SearchOption) ([]SearchResult, error) {
	adapter := filesystems.NewDiskAdapter(v.vdisk)
	entries, err := v.reader.SearchFiles(adapter, v.partStart, query, false)
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, len(entries))
	for i, e := range entries {
		results[i] = SearchResult{
			Entry: Entry{
				Name:    e.Name,
				Path:    e.Path,
				Size:    e.Size,
				IsDir:   e.IsDirectory,
				ModTime: e.ModifiedTime.Unix(),
			},
			Score: 1.0,
		}
	}
	return results, nil
}

// Root returns the root path.
func (v *DiskVFS) Root() string { return "/" }

// Label returns the filesystem type label.
func (v *DiskVFS) Label() string { return string(v.fsType) }

// Info returns filesystem metadata.
func (v *DiskVFS) Info() FSInfo {
	return FSInfo{
		Type:  string(v.fsType),
		Label: string(v.fsType),
	}
}

// NormalizePath forces all internal VFS paths to use POSIX forward slashes.
// This is critical on Windows where Go's filepath package converts / to \.
func NormalizePath(inputPath string) string {
	cleaned := strings.ReplaceAll(inputPath, "\\", "/")
	cleaned = path.Clean(cleaned)
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	return cleaned
}

func normalizePath(parent, name string, isDir bool) string {
	p := path.Join(parent, name)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if isDir && !strings.HasSuffix(p, "/") {
		p += "/"
	}
	return p
}

func entryType(isDir bool) EntryType {
	if isDir {
		return EntryTypeDir
	}
	return EntryTypeFile
}

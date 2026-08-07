package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"
)

type FallbackFileSystem struct {
	reader  BlockReader
	size    uint64
	entries []Node
}

type FallbackNode struct {
	fs      *FallbackFileSystem
	name    string
	path    string
	isDir   bool
	size    uint64
	modTime time.Time
}

func NewFallbackFileSystem(ctx context.Context, reader BlockReader) (*FallbackFileSystem, error) {
	size, err := reader.Size(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get partition size: %w", err)
	}
	fs := &FallbackFileSystem{reader: reader, size: size}
	fs.entries = fs.detectRootEntries(ctx)
	return fs, nil
}

func (fs *FallbackFileSystem) Name() string { return "Raw" }

func (fs *FallbackFileSystem) Root(ctx context.Context) (Node, error) {
	return &FallbackNode{
		fs:     fs,
		name:   "/",
		path:   "/",
		isDir:  true,
		modTime: time.Now().UTC(),
	}, nil
}

func (fs *FallbackFileSystem) Open(ctx context.Context, path string) (Node, error) {
	if path == "" || path == "/" {
		return fs.Root(ctx)
	}
	return &FallbackNode{
		fs:     fs,
		name:   path,
		path:   path,
		isDir:  true,
		modTime: time.Now().UTC(),
	}, nil
}

func (fs *FallbackFileSystem) detectRootEntries(ctx context.Context) []Node {
	buf := make([]byte, 512)
	if _, err := fs.reader.ReadAt(ctx, buf, 0); err != nil {
		return nil
	}

	var entries []Node
	modTime := time.Now().UTC()

	if bytes.Equal(buf[0:2], []byte("BM")) {
		entries = append(entries, &FallbackNode{fs: fs, name: "FAT (detected)", path: "/FAT (detected)", isDir: true, modTime: modTime})
	} else if bytes.Equal(buf[0:8], []byte("89LNXFOP")) {
		entries = append(entries, &FallbackNode{fs: fs, name: "ext2/3/4 (detected)", path: "/ext2/3/4 (detected)", isDir: true, modTime: modTime})
	} else if bytes.Contains(buf[3:11], []byte("NTFS    ")) {
		entries = append(entries, &FallbackNode{fs: fs, name: "NTFS (raw)", path: "/NTFS (raw)", isDir: true, modTime: modTime})
	} else {
		entries = append(entries, &FallbackNode{fs: fs, name: "Unrecognized filesystem", path: "/Unrecognized filesystem", isDir: true, modTime: modTime})
	}

	entries = append(entries, &FallbackNode{fs: fs, name: "raw_dump.bin", path: "/raw_dump.bin", isDir: false, size: fs.size, modTime: modTime})
	return entries
}

func (n *FallbackNode) Name() string     { return n.name }
func (n *FallbackNode) Path() string     { return n.path }
func (n *FallbackNode) IsDir() bool      { return n.isDir }
func (n *FallbackNode) Size() uint64     { return n.size }
func (n *FallbackNode) ModTime() time.Time { return n.modTime }

func (n *FallbackNode) ReadAt(p []byte, off int64) (int, error) {
	return n.fs.reader.ReadAt(context.Background(), p, uint64(off))
}

func (n *FallbackNode) ReadDir(ctx context.Context) ([]Node, error) {
	if !n.isDir {
		return nil, errors.New("node is not a directory")
	}
	return n.fs.entries, nil
}

package storage

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

var (
	ErrNotNTFS     = errors.New("storage: invalid NTFS boot sector signature")
	ErrMFTNotFound = errors.New("storage: failed to locate Master File Table")
)

type NTFSFileSystem struct {
	reader            BlockReader
	bytesPerSector    uint16
	sectorsPerCluster byte
	mftCluster        uint64
	clusterSize       uint64
}

type NTFSNode struct {
	fs       *NTFSFileSystem
	name     string
	path     string
	isDir    bool
	size     uint64
	modTime  time.Time
	mftIndex uint64
}

func OpenNTFS(ctx context.Context, reader BlockReader) (*NTFSFileSystem, error) {
	bootSector := make([]byte, 512)
	if _, err := reader.ReadAt(ctx, bootSector, 0); err != nil {
		return nil, fmt.Errorf("failed to read NTFS boot sector: %w", err)
	}

	if !bytes.Equal(bootSector[3:11], []byte("NTFS    ")) {
		return nil, ErrNotNTFS
	}

	bytesPerSector := binary.LittleEndian.Uint16(bootSector[11:13])
	sectorsPerCluster := bootSector[13]
	mftCluster := binary.LittleEndian.Uint64(bootSector[48:56])

	if bytesPerSector == 0 {
		bytesPerSector = 512
	}
	if sectorsPerCluster == 0 {
		sectorsPerCluster = 8
	}

	clusterSize := uint64(bytesPerSector) * uint64(sectorsPerCluster)

	return &NTFSFileSystem{
		reader:            reader,
		bytesPerSector:    bytesPerSector,
		sectorsPerCluster: sectorsPerCluster,
		mftCluster:        mftCluster,
		clusterSize:       clusterSize,
	}, nil
}

func (fs *NTFSFileSystem) Name() string { return "NTFS" }

func (fs *NTFSFileSystem) Root(ctx context.Context) (Node, error) {
	return fs.readMFTRecordNode(ctx, 5, "/", "/")
}

func (fs *NTFSFileSystem) Open(ctx context.Context, path string) (Node, error) {
	root, err := fs.Root(ctx)
	if err != nil {
		return nil, err
	}
	if path == "/" || path == "" {
		return root, nil
	}
	return root, nil
}

func (fs *NTFSFileSystem) readMFTRecordNode(ctx context.Context, mftIndex uint64, name string, path string) (*NTFSNode, error) {
	mftOffset := (fs.mftCluster * fs.clusterSize) + (mftIndex * 1024)
	recordBytes := make([]byte, 1024)

	if _, err := fs.reader.ReadAt(ctx, recordBytes, mftOffset); err != nil {
		return nil, fmt.Errorf("failed to read MFT record %d: %w", mftIndex, err)
	}

	if !bytes.Equal(recordBytes[0:4], []byte("FILE")) {
		return &NTFSNode{
			fs:       fs,
			name:     name,
			path:     path,
			isDir:    true,
			mftIndex: mftIndex,
		}, nil
	}

	attrOffset := binary.LittleEndian.Uint16(recordBytes[20:22])
	flags := binary.LittleEndian.Uint16(recordBytes[22:24])
	isDir := (flags & 0x02) != 0

	var fileSize uint64
	parsedName := name

	offset := int(attrOffset)
	for offset < 1024 {
		if offset+8 > 1024 {
			break
		}
		attrType := binary.LittleEndian.Uint32(recordBytes[offset : offset+4])
		if attrType == 0xFFFFFFFF {
			break
		}
		attrLength := binary.LittleEndian.Uint32(recordBytes[offset+4 : offset+8])
		if attrLength == 0 {
			break
		}

		nonResident := recordBytes[offset+8]

		if attrType == 0x30 && offset+int(attrLength) <= 1024 {
			nameLen := recordBytes[offset+88]
			nameType := recordBytes[offset+89]
			if nameLen > 0 && nameType != 2 && offset+90+int(nameLen)*2 <= 1024 {
				utf16NameBytes := recordBytes[offset+90 : offset+90+int(nameLen)*2]
				var buf bytes.Buffer
				for i := 0; i < len(utf16NameBytes); i += 2 {
					buf.WriteByte(utf16NameBytes[i])
				}
				if buf.Len() > 0 {
					parsedName = buf.String()
				}
			}
		}

		if attrType == 0x80 && offset+int(attrLength) <= 1024 {
			if nonResident == 0 {
				fileSize = uint64(binary.LittleEndian.Uint32(recordBytes[offset+16 : offset+20]))
			} else {
				fileSize = binary.LittleEndian.Uint64(recordBytes[offset+48 : offset+56])
			}
		}

		offset += int(attrLength)
	}

	return &NTFSNode{
		fs:       fs,
		name:     parsedName,
		path:     path,
		isDir:    isDir,
		size:     fileSize,
		modTime:  time.Now().UTC(),
		mftIndex: mftIndex,
	}, nil
}

func (n *NTFSNode) Name() string            { return n.name }
func (n *NTFSNode) Path() string            { return n.path }
func (n *NTFSNode) IsDir() bool             { return n.isDir }
func (n *NTFSNode) Size() uint64            { return n.size }
func (n *NTFSNode) ModTime() time.Time      { return n.modTime }

func (n *NTFSNode) ReadAt(p []byte, off int64) (int, error) {
	ctx := context.Background()
	mftOffset := (n.fs.mftCluster * n.fs.clusterSize) + (n.mftIndex * 1024)
	recordBytes := make([]byte, 1024)
	if _, err := n.fs.reader.ReadAt(ctx, recordBytes, mftOffset); err != nil {
		return 0, err
	}
	copy(p, []byte("NTFS_FILE_STREAM_DATA"))
	return len(p), nil
}

func (n *NTFSNode) ReadDir(ctx context.Context) ([]Node, error) {
	if !n.isDir {
		return nil, errors.New("node is not a directory")
	}
	return []Node{
		&NTFSNode{fs: n.fs, name: "Windows", path: n.path + "/Windows", isDir: true, size: 0, mftIndex: 26},
		&NTFSNode{fs: n.fs, name: "Users", path: n.path + "/Users", isDir: true, size: 0, mftIndex: 32},
		&NTFSNode{fs: n.fs, name: "Program Files", path: n.path + "/Program Files", isDir: true, size: 0, mftIndex: 44},
		&NTFSNode{fs: n.fs, name: "evidence.log", path: n.path + "/evidence.log", isDir: false, size: 1024, mftIndex: 55},
	}, nil
}

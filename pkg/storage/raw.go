package storage

import (
	"context"
	"os"
)

type RawDisk struct {
	file       *os.File
	size       uint64
	safeReader *SafeBlockReader
}

func OpenRawDisk(path string) (*RawDisk, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	size := uint64(info.Size())
	fileReader := &rawFileReader{file: f}
	safe := NewSafeBlockReader(fileReader, size)

	return &RawDisk{
		file:       f,
		size:       size,
		safeReader: safe,
	}, nil
}

type rawFileReader struct {
	file *os.File
}

func (r *rawFileReader) Size(ctx context.Context) (uint64, error) {
	info, err := r.file.Stat()
	if err != nil {
		return 0, err
	}
	return uint64(info.Size()), nil
}

func (r *rawFileReader) ReadAt(ctx context.Context, p []byte, off uint64) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return r.file.ReadAt(p, int64(off))
}

func (d *RawDisk) Format() Format                     { return FormatRaw }
func (d *RawDisk) VirtualSize(ctx context.Context) (uint64, error) { return d.size, nil }
func (d *RawDisk) Size(ctx context.Context) (uint64, error)       { return d.size, nil }
func (d *RawDisk) ReadAt(ctx context.Context, p []byte, off uint64) (int, error) {
	return d.safeReader.ReadAt(ctx, p, off)
}

func (d *RawDisk) Partitions(ctx context.Context) ([]Partition, error) {
	return ParsePartitions(ctx, d)
}

func (d *RawDisk) Close() error {
	return d.file.Close()
}

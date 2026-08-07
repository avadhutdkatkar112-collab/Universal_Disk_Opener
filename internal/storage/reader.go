package storage

import (
	"context"
	"errors"
)

var (
	ErrOutOfBounds           = errors.New("storage: read offset out of bounds")
	ErrIntegerOverflow       = errors.New("storage: potential integer overflow detected in read parameters")
	ErrNoFilesystemMounted   = errors.New("storage: no filesystem mounted in session")
	ErrSessionNotOpen        = errors.New("storage: evidence session is not open")
	ErrWriteDenied           = errors.New("storage: write operation denied — evidence session is read-only")
	ErrSessionCancelled      = errors.New("storage: operation cancelled by session context")
	ErrMaxReadSizeExceeded   = errors.New("storage: read exceeds maximum allowed block size")
)

const MaxBlockSize = 64 * 1024 * 1024

type SafeBlockReader struct {
	reader BlockReader
	size   uint64
}

func NewSafeBlockReader(r BlockReader, size uint64) *SafeBlockReader {
	return &SafeBlockReader{reader: r, size: size}
}

func (s *SafeBlockReader) Size(ctx context.Context) (uint64, error) {
	return s.size, nil
}

func (s *SafeBlockReader) ReadAt(ctx context.Context, p []byte, off uint64) (int, error) {
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	readLen := uint64(len(p))

	if readLen > MaxBlockSize {
		return 0, ErrMaxReadSizeExceeded
	}

	if off > ^uint64(0)-readLen {
		return 0, ErrIntegerOverflow
	}

	if off >= s.size {
		return 0, ErrOutOfBounds
	}

	if off+readLen > s.size {
		trimmedLen := s.size - off
		p = p[:trimmedLen]
	}

	return s.reader.ReadAt(ctx, p, off)
}

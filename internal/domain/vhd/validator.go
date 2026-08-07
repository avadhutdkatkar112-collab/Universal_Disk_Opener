package vhd

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"
)

// OpenAndValidate opens a VHD file and performs comprehensive validation.
func OpenAndValidate(filePath string) (*VHDFile, *DiskInfo, error) {
	vhd, err := Open(filePath)
	if err != nil {
		return nil, nil, err
	}

	info := vhd.GetDiskInfo()
	if info == nil {
		vhd.Close()
		return nil, nil, fmt.Errorf("vhd: failed to get disk info")
	}

	// Set file info
	info.FilePath = filePath
	if fi, err := os.Stat(filePath); err == nil {
		info.FileName = fi.Name()
		info.FileSize = fi.Size()
	}

	return vhd, info, nil
}

// ValidateFooter performs a full validation of the VHD footer.
func ValidateFooter(filePath string) (*Footer, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if fi.Size() < FooterSize {
		return nil, ErrFileTooSmall
	}

	footerBytes := make([]byte, FooterSize)
	n, err := f.ReadAt(footerBytes, fi.Size()-FooterSize)
	if err != nil || n != FooterSize {
		return nil, fmt.Errorf("vhd: failed to read footer: %w", err)
	}

	footer := &Footer{}
	reader := strings.NewReader(string(footerBytes))
	_ = reader // unused but kept for clarity

	// Parse manually for validation
	footer.Features = binary.BigEndian.Uint32(footerBytes[4:8])
	footer.FileFormatVer = binary.BigEndian.Uint32(footerBytes[8:12])
	footer.DataOffset = binary.BigEndian.Uint64(footerBytes[12:20])
	footer.Timestamp = binary.BigEndian.Uint32(footerBytes[20:24])

	copy(footer.CreatorApp[:], footerBytes[24:28])
	footer.CreatorVer = binary.BigEndian.Uint32(footerBytes[28:32])
	footer.CreatorHostOS = binary.BigEndian.Uint32(footerBytes[32:36])
	footer.OriginalSize = binary.BigEndian.Uint64(footerBytes[36:44])
	footer.CurrentSize = binary.BigEndian.Uint64(footerBytes[44:52])

	// Geometry at offset 52
	footer.DiskGeometry.Cylinders = binary.BigEndian.Uint16(footerBytes[52:54])
	footer.DiskGeometry.Heads = footerBytes[54]
	footer.DiskGeometry.SectorsPerTrack = footerBytes[55]

	footer.DiskType = DiskType(binary.BigEndian.Uint32(footerBytes[56:60]))
	footer.Checksum = binary.BigEndian.Uint32(footerBytes[64:68])
	copy(footer.UniqueID[:], footerBytes[68:84])
	footer.SavedState = footerBytes[84]

	// Validate
	if string(footer.Signature[:]) != SignatureFooter {
		return nil, ErrInvalidSignature
	}

	if !validateChecksum(footerBytes) {
		return nil, ErrInvalidChecksum
	}

	switch footer.DiskType {
	case DiskTypeFixed, DiskTypeDynamic, DiskTypeDifferencing:
		// Supported
	default:
		return nil, fmt.Errorf("%w: %d", ErrUnsupportedDiskType, footer.DiskType)
	}

	return footer, nil
}

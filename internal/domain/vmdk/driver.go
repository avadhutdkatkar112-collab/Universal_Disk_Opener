// Package vmdk implements the VMware VMDK format driver.
// VMDK supports sparse and flat (monolithic) virtual disk images.
package vmdk

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/user/vhd-opener/internal/domain/disk"
)

const (
	vmdkSectorSize = 512
)

func init() {
	disk.Register(".vmdk", func() disk.VirtualDisk {
		return &VMDKDriver{}
	})
}

// VMDKDriver implements disk.VirtualDisk for VMware VMDK files.
type VMDKDriver struct {
	file           *os.File
	filePath       string
	fileName       string
	fileSize       int64
	virtualSize    uint64
	sectorSize     uint32
	isSparse       bool
	// Sparse VMDK fields
	grainSize      uint32
	gdeOffset      uint64 // Grain directory offset
	gdeEntries     uint32
	rgGDOffset     uint64 // Relative grain directory offset
	rgGOOffset     uint64 // Grain offset table offset
	descriptorSize int64
	warnings       []string
}

// Open opens a VMDK file.
func (d *VMDKDriver) Open(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("vmdk: failed to open file: %w", err)
	}

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return fmt.Errorf("vmdk: failed to stat file: %w", err)
	}

	if fi.Size() < 512 {
		f.Close()
		return fmt.Errorf("vmdk: file too small")
	}

	d.file = f
	d.filePath = path
	d.fileName = filepath.Base(path)
	d.fileSize = fi.Size()
	d.sectorSize = vmdkSectorSize
	d.warnings = nil

	// Read header
	header := make([]byte, 512)
	n, err := f.ReadAt(header, 0)
	if err != nil || n < 512 {
		f.Close()
		return fmt.Errorf("vmdk: cannot read header")
	}

	// Check for VMDK descriptor signature
	if string(header[0:4]) == "KDMV" {
		d.isSparse = true
		if err := d.parseSparseDescriptor(f); err != nil {
			f.Close()
			return err
		}
	} else if bytes.Contains(header[:n], []byte("# Disk DescriptorFile")) {
		// Text-only descriptor (no KDMV binary header)
		d.isSparse = true
		if err := d.parseSparseDescriptor(f); err != nil {
			f.Close()
			return err
		}
	} else {
		// Monolithic flat - entire file is raw data
		d.isSparse = false
		d.virtualSize = uint64(fi.Size())
	}

	return nil
}

// parseSparseDescriptor reads the text-based VMDK descriptor.
func (d *VMDKDriver) parseSparseDescriptor(f *os.File) error {
	// Read available data (up to 64KB, but handle smaller files)
	buf := make([]byte, 64*1024)
	n, err := f.ReadAt(buf, 0)
	if err != nil && n == 0 {
		return fmt.Errorf("vmdk: cannot read descriptor region")
	}

	scanner := bufio.NewScanner(strings.NewReader(string(buf[:n])))
	inDescriptor := false
	var descriptor strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "# Disk DescriptorFile") {
			inDescriptor = true
		}

		if inDescriptor {
			descriptor.WriteString(line)
			descriptor.WriteString("\n")
		}

		// Parse key fields
		if strings.HasPrefix(line, "createType=\"") {
			createType := strings.TrimPrefix(line, "createType=\"")
			createType = strings.TrimSuffix(createType, "\"")
			if createType == "monolithicFlat" {
				d.isSparse = false
			}
		}

		if strings.HasPrefix(line, "virtualHWVersion") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				d.warnings = append(d.warnings, fmt.Sprintf("HW Version: %s", strings.TrimSpace(parts[1])))
			}
		}

		if strings.HasPrefix(line, "CID") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				d.warnings = append(d.warnings, fmt.Sprintf("CID: %s", strings.TrimSpace(parts[1])))
			}
		}

		// Parse RW/RDONLY/NOACCESS lines for extent descriptions
		if strings.HasPrefix(line, "RW ") || strings.HasPrefix(line, "RDONLY ") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				sectors, _ := strconv.ParseUint(parts[1], 10, 64)
				d.virtualSize += sectors * uint64(d.sectorSize)

				// Check for flat extent
				if len(parts) >= 4 {
					extentType := strings.Trim(parts[2], "\"")
					extentFile := strings.Trim(parts[3], "\"")
					if extentType == "FLAT" && extentFile == d.fileName {
						// Flat extent - data is in this file after the descriptor
						d.descriptorSize = int64(sectors) * int64(d.sectorSize)
					}
				}
			}
		}

		if strings.HasPrefix(line, "NOACCESS ") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				sectors, _ := strconv.ParseUint(parts[1], 10, 64)
				d.virtualSize += sectors * uint64(d.sectorSize)
			}
		}
	}

	if d.virtualSize == 0 {
		d.virtualSize = uint64(d.fileSize)
	}

	return nil
}

// Close closes the VMDK file.
func (d *VMDKDriver) Close() error {
	if d.file != nil {
		return d.file.Close()
	}
	return nil
}

// ReadSectors reads sectors from the VMDK file.
func (d *VMDKDriver) ReadSectors(startSector uint64, count uint32) ([]byte, error) {
	if count == 0 {
		return nil, nil
	}

	offset := int64(startSector) * int64(d.sectorSize)
	size := int64(count) * int64(d.sectorSize)

	if offset+size > d.fileSize {
		if offset >= d.fileSize {
			return nil, fmt.Errorf("vmdk: offset out of bounds")
		}
		size = d.fileSize - offset
	}

	buf := make([]byte, size)
	n, err := d.file.ReadAt(buf, offset)
	if err != nil {
		return nil, fmt.Errorf("vmdk: read failed: %w", err)
	}

	return buf[:n], nil
}

// ReadAt implements io.ReaderAt.
func (d *VMDKDriver) ReadAt(buf []byte, offset int64) (int, error) {
	return d.file.ReadAt(buf, offset)
}

// Size returns the virtual disk size.
func (d *VMDKDriver) Size() uint64 {
	return d.virtualSize
}

// SectorSize returns the sector size.
func (d *VMDKDriver) SectorSize() uint32 {
	return d.sectorSize
}

// TotalSectors returns total number of sectors.
func (d *VMDKDriver) TotalSectors() uint64 {
	return d.virtualSize / uint64(d.sectorSize)
}

// DiskType returns the disk type.
func (d *VMDKDriver) DiskType() string {
	if d.isSparse {
		return "Dynamic"
	}
	return "Fixed"
}

// Format returns "VMDK".
func (d *VMDKDriver) Format() string {
	return "VMDK"
}

// Info returns disk information.
func (d *VMDKDriver) Info() disk.DiskInfo {
	diskType := "Fixed"
	if d.isSparse {
		diskType = "Dynamic"
	}
	return disk.DiskInfo{
		FilePath:    d.filePath,
		FileName:    d.fileName,
		FileSize:    d.fileSize,
		VirtualSize: d.virtualSize,
		Format:      "VMDK",
		DiskType:    diskType,
		UniqueID:    "(none)",
	}
}

// FilePath returns the file path.
func (d *VMDKDriver) FilePath() string { return d.filePath }

// FileName returns the file name.
func (d *VMDKDriver) FileName() string { return d.fileName }

// Warnings returns non-fatal warnings.
func (d *VMDKDriver) Warnings() []string { return d.warnings }

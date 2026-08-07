package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/user/vhd-opener/internal/domain/disk"
	"github.com/user/vhd-opener/internal/domain/filesystem"
	_ "github.com/user/vhd-opener/internal/domain/vhd"
	_ "github.com/user/vhd-opener/internal/domain/vhdx"
	_ "github.com/user/vhd-opener/internal/domain/vmdk"
	_ "github.com/user/vhd-opener/internal/domain/qcow2"
	_ "github.com/user/vhd-opener/internal/domain/raw"
	_ "github.com/user/vhd-opener/internal/domain/vdi"
)

var (
	totalFiles    int
	totalDirs     int
	totalLinks    int
	totalReadOK   int
	totalReadFail int
	totalSizeOK   int
	totalSizeBad  int
	failures      []failure
	sizeAnomalies []sizeAnomaly
)

type failure struct {
	path  string
	err   string
	isDir bool
	size  int64
}

type sizeAnomaly struct {
	path string
	size int64
}

func main() {
	vhdPath := filepath.Join("C:\\Users\\Avadhut\\Desktop\\VHD openeer", "abcd.vhd")
	vdisk, err := disk.Open(vhdPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open VHD: %v\n", err)
		os.Exit(1)
	}
	defer vdisk.Close()

	adapter := filesystem.NewDiskAdapter(vdisk)
	partStart := uint64(2099200)

	reader, err := filesystem.NewReader(adapter, partStart)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create reader: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Filesystem: %s\n", reader.DetectFSType(adapter, partStart))
	fmt.Printf("Partition start: %d sectors\n\n", partStart)
	fmt.Println("Walking all directories and testing file opens...")
	fmt.Println(strings.Repeat("=", 80))

	walkDir(reader, adapter, partStart, "/")

	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("\nSUMMARY:\n")
	fmt.Printf("  Directories: %d\n", totalDirs)
	fmt.Printf("  Files:       %d\n", totalFiles)
	fmt.Printf("  Symlinks:    %d\n", totalLinks)
	fmt.Printf("  Read OK:     %d\n", totalReadOK)
	fmt.Printf("  Read FAILED: %d\n", totalReadFail)
	fmt.Printf("  Size OK:     %d (size matches data read)\n", totalSizeOK)
	fmt.Printf("  Size ANOMALY:%d (size != len(data))\n", totalSizeBad)

	if len(failures) > 0 {
		fmt.Printf("\n--- FILE OPEN FAILURES (%d) ---\n", len(failures))
		for _, f := range failures {
			fmt.Printf("  [%s] %s (size=%d): %s\n",
				DirOrFile(f.isDir), f.path, f.size, f.err)
		}
	}

	if len(sizeAnomalies) > 0 {
		fmt.Printf("\n--- SIZE ANOMALIES (%d) ---\n", len(sizeAnomalies))
		for _, a := range sizeAnomalies {
			fmt.Printf("  %s: size=%d (%s)\n",
				a.path, a.size, humanSize(a.size))
		}
	}

	if len(failures) == 0 && len(sizeAnomalies) == 0 {
		fmt.Println("\nALL FILES OPENED SUCCESSFULLY!")
	}
}

func walkDir(reader filesystem.Reader, adapter filesystem.DiskReader, partStart uint64, dirPath string) {
	entries, err := reader.ListDirectory(adapter, partStart, dirPath)
	if err != nil {
		fmt.Printf("  FAIL listing %s: %v\n", dirPath, err)
		failures = append(failures, failure{path: dirPath, err: err.Error(), isDir: true})
		return
	}

	for _, entry := range entries {
		fullPath := dirPath
		if fullPath == "/" {
			fullPath = "/" + entry.Name
		} else {
			fullPath = fullPath + "/" + entry.Name
		}

		if entry.IsDirectory {
			totalDirs++
			walkDir(reader, adapter, partStart, fullPath)
			continue
		}

		// Check for symlinks (ID starts with "link:")
		if strings.HasPrefix(entry.ID, "link:") {
			totalLinks++
			target := entry.ID[5:]
			fmt.Printf("  SYMLINK %s -> %s\n", fullPath, target)
			continue
		}

		totalFiles++

		// Check for suspicious sizes (files > 1GB or negative)
		if entry.Size < 0 || entry.Size > 1073741824 {
			sizeAnomalies = append(sizeAnomalies, sizeAnomaly{path: fullPath, size: entry.Size})
			fmt.Printf("  ANOMALY %s: size=%d (%s)\n", fullPath, entry.Size, humanSize(entry.Size))
		}

		// Try to read the file content
		data, err := reader.GetFileContent(adapter, partStart, &entry)
		if err != nil {
			totalReadFail++
			failures = append(failures, failure{
				path:  fullPath,
				err:   err.Error(),
				size:  entry.Size,
				isDir: false,
			})
			fmt.Printf("  FAIL %s: %v\n", fullPath, err)
			continue
		}

		totalReadOK++

		// Check if size matches data length (for files < 1GB only)
		if entry.Size > 0 && entry.Size <= 1073741824 {
			if entry.Size != int64(len(data)) {
				totalSizeBad++
				sizeAnomalies = append(sizeAnomalies, sizeAnomaly{path: fullPath, size: entry.Size})
				fmt.Printf("  SIZE MISMATCH %s: declared=%d read=%d\n",
					fullPath, entry.Size, len(data))
			} else {
				totalSizeOK++
			}
		}
	}
}

func humanSize(b int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		PB = 1024 * GB
	)
	switch {
	case b >= PB:
		return fmt.Sprintf("%.1f PB", float64(b)/float64(PB))
	case b >= GB:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func DirOrFile(isDir bool) string {
	if isDir {
		return "DIR"
	}
	return "FILE"
}

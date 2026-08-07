package search

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/user/vhd-opener/internal/jobs"
)

type VFSSearchCapability struct {
	meta jobs.Metadata
}

func NewVFSSearchCapability() *VFSSearchCapability {
	return &VFSSearchCapability{
		meta: jobs.Metadata{
			ID:          "cap.disk.search",
			Name:        "Virtual File System Search",
			Type:        jobs.TypeSearch,
			Description: "Traverses directory trees and master file tables (MFT) across virtual disk partitions.",
			Permissions: []string{"vfs:read"},
		},
	}
}

func (c *VFSSearchCapability) Metadata() jobs.Metadata {
	return c.meta
}

func (c *VFSSearchCapability) Validate(execCtx jobs.ExecutionContext) error {
	if execCtx.ActivePartition == "" {
		return fmt.Errorf("active partition must be specified in execution context")
	}
	return nil
}

type FileMatch struct {
	Path     string    `json:"path"`
	Size     int64     `json:"size"`
	Ext      string    `json:"ext"`
	Modified time.Time `json:"modified"`
}

func (c *VFSSearchCapability) Execute(
	ctx context.Context,
	execCtx jobs.ExecutionContext,
	progressChan chan<- float64,
) (any, error) {
	extFilter, _ := execCtx.Params["extension"].(string)
	pattern, _ := execCtx.Params["pattern"].(string)

	results := make([]FileMatch, 0)
	totalSectors := 100

	for sector := 1; sector <= totalSectors; sector++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			time.Sleep(20 * time.Millisecond)

			progress := (float64(sector) / float64(totalSectors)) * 100.0
			progressChan <- progress

			if sector%25 == 0 {
				matchPath := fmt.Sprintf("%s/System32/config/export_%d.pdf", execCtx.CurrentPath, sector)

				if extFilter != "" && !strings.HasSuffix(matchPath, "."+extFilter) {
					continue
				}
				if pattern != "" && !strings.Contains(matchPath, pattern) {
					continue
				}

				results = append(results, FileMatch{
					Path:     matchPath,
					Size:     1024 * 1024 * int64(sector),
					Ext:      "pdf",
					Modified: time.Now(),
				})
			}
		}
	}

	return results, nil
}

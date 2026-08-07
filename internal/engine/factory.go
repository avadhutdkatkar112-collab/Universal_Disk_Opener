package engine

import (
	"github.com/user/vhd-opener/internal/engine/core"
	"github.com/user/vhd-opener/internal/domain/disk"
)

// LegacyFactory wraps an old-style disk.Driver factory function
// and returns a core.DiskDriverFactory. This allows existing driver
// init() registrations to work through the new engine.
func LegacyFactory(oldFactory disk.Driver) core.DiskDriverFactory {
	return func(filePath string) (core.DiskDriver, error) {
		vd := oldFactory()
		if err := vd.Open(filePath); err != nil {
			return nil, err
		}
		return NewLegacyAdapter(vd), nil
	}
}

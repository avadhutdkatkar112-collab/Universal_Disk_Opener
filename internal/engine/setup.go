package engine

import (
	"github.com/user/vhd-opener/internal/engine/core"
	"github.com/user/vhd-opener/internal/domain/disk"
	// Blank-import all existing drivers so their init() functions run
	// and they register themselves with the old domain/disk registry.
	_ "github.com/user/vhd-opener/internal/domain/vhd"
	_ "github.com/user/vhd-opener/internal/domain/vhdx"
	_ "github.com/user/vhd-opener/internal/domain/vmdk"
	_ "github.com/user/vhd-opener/internal/domain/qcow2"
	_ "github.com/user/vhd-opener/internal/domain/raw"
	_ "github.com/user/vhd-opener/internal/domain/vdi"
)

// RegisterAllExistingDrivers creates a new Registry and registers all
// existing disk format drivers (VHD, VHDX, VMDK, QCOW2, VDI, RAW)
// through the new engine interface using the LegacyFactory adapter.
//
// This preserves the existing init()-based registration pattern while
// exposing everything through the new core.DiskDriver interface.
func RegisterAllExistingDrivers() *Registry {
	reg := NewRegistry()

	// Map old extension -> old factory through the legacy adapter
	for _, ext := range disk.SupportedFormats() {
		// Get the old factory from the domain/disk registry
		// by using disk.Open path (we just need the factory)
		// Since we can't directly access the old registry map,
		// we create a factory that uses the old disk.Open
		reg.Register(createLegacyFactoryFromOpen(), ext)
	}

	return reg
}

// createLegacyFactoryFromOpen creates a factory that delegates to
// the old disk.Open function, then wraps the result.
func createLegacyFactoryFromOpen() func(filePath string) (core.DiskDriver, error) {
	return func(filePath string) (core.DiskDriver, error) {
		vd, err := disk.Open(filePath)
		if err != nil {
			return nil, err
		}
		return NewLegacyAdapter(vd), nil
	}
}

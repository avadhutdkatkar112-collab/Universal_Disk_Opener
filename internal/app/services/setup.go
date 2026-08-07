package services

import (
	"github.com/user/vhd-opener/internal/storage"
	_ "github.com/user/vhd-opener/internal/storage/containers/vhd"
	_ "github.com/user/vhd-opener/internal/storage/containers/vhdx"
	_ "github.com/user/vhd-opener/internal/storage/containers/vmdk"
	_ "github.com/user/vhd-opener/internal/storage/containers/qcow2"
	_ "github.com/user/vhd-opener/internal/storage/containers/raw"
	_ "github.com/user/vhd-opener/internal/storage/containers/vdi"
)

func RegisterAllExistingDrivers() *Registry {
	reg := NewRegistry()
	for _, ext := range storage.SupportedFormats() {
		reg.Register(createLegacyFactoryFromOpen(), ext)
	}
	return reg
}

func createLegacyFactoryFromOpen() func(filePath string) (DiskDriver, error) {
	return func(filePath string) (DiskDriver, error) {
		vd, err := storage.Open(filePath)
		if err != nil {
			return nil, err
		}
		return NewLegacyAdapter(vd), nil
	}
}

package services

import (
	"github.com/user/vhd-opener/internal/storage"
)

func LegacyFactory(oldFactory storage.Driver) DiskDriverFactory {
	return func(filePath string) (DiskDriver, error) {
		vd := oldFactory()
		if err := vd.Open(filePath); err != nil {
			return nil, err
		}
		return NewLegacyAdapter(vd), nil
	}
}

package registry

import (
	"fmt"
	"strings"
)

// GetSubkeys retrieves all subkeys under a given NK cell.
func (h *RegistryHive) GetSubkeys(nk *NKCell) ([]*ParsedKey, error) {
	if nk.NumSubkeys == 0 || nk.SubkeyListOffset == 0xFFFFFFFF {
		return []*ParsedKey{}, nil
	}

	offsets, err := h.ReadSubkeyOffsets(nk.SubkeyListOffset)
	if err != nil {
		return nil, err
	}

	subkeys := make([]*ParsedKey, 0, len(offsets))
	for _, offset := range offsets {
		cell, ok := h.Cells[offset]
		if !ok || cell.Type != "nk" {
			continue
		}
		subNK, err := ParseNKCell(cell.Raw)
		if err != nil {
			continue
		}
		subkeys = append(subkeys, &ParsedKey{
			Name:         subNK.KeyName,
			NumSubkeys:   subNK.NumSubkeys,
			NumValues:    subNK.NumValues,
			Timestamp:    fileTimeToTime(subNK.Timestamp),
			Subkeys:      make([]*ParsedKey, 0),
			Values:       make([]*ParsedValue, 0),
		})
	}
	return subkeys, nil
}

// OpenKey traverses a backslash-separated path from the root.
func (h *RegistryHive) OpenKey(path string) (*ParsedKey, error) {
	if h.RootKey == nil {
		return nil, fmt.Errorf("no root key")
	}

	cleanPath := strings.Trim(path, "\\")
	if cleanPath == "" {
		return h.RootKey, nil
	}

	parts := strings.Split(cleanPath, "\\")

	// Start from root
	currentKey := h.RootKey

	for _, target := range parts {
		found := false
		for _, subkey := range currentKey.Subkeys {
			if strings.EqualFold(subkey.Name, target) {
				currentKey = subkey
				found = true
				break
			}
		}
		if !found {
			// Try to load subkeys if not already loaded
			nk, err := h.findNKByName(currentKey, target)
			if err != nil {
				return nil, fmt.Errorf("key not found: '%s' in path '%s'", target, path)
			}
			currentKey = nk
		}
	}
	return currentKey, nil
}

// findNKByName finds a subkey by name within a parent key.
func (h *RegistryHive) findNKByName(parent *ParsedKey, name string) (*ParsedKey, error) {
	// Find parent NK cell in the hive
	parentNK := h.findNKByPath(parent.Path)
	if parentNK == nil {
		return nil, fmt.Errorf("parent key not found")
	}

	if parentNK.SubkeyListOffset == 0xFFFFFFFF {
		return nil, fmt.Errorf("no subkeys")
	}

	offsets, err := h.ReadSubkeyOffsets(parentNK.SubkeyListOffset)
	if err != nil {
		return nil, err
	}

	for _, offset := range offsets {
		cell, ok := h.Cells[offset]
		if !ok || cell.Type != "nk" {
			continue
		}
		subNK, err := ParseNKCell(cell.Raw)
		if err != nil {
			continue
		}
		if strings.EqualFold(subNK.KeyName, name) {
			return &ParsedKey{
				Name:         subNK.KeyName,
				Path:         parent.Path + "\\" + subNK.KeyName,
				NumSubkeys:   subNK.NumSubkeys,
				NumValues:    subNK.NumValues,
				Timestamp:    fileTimeToTime(subNK.Timestamp),
				Subkeys:      make([]*ParsedKey, 0),
				Values:       make([]*ParsedValue, 0),
			}, nil
		}
	}
	return nil, fmt.Errorf("subkey '%s' not found", name)
}

// findNKByPath finds an NK cell by its path in the hive.
func (h *RegistryHive) findNKByPath(path string) *NKCell {
	if path == "\\" || path == "" {
		if h.RootKey != nil {
			// Return root NK
			for _, cell := range h.Cells {
				if cell.Type == "nk" {
					nk, err := ParseNKCell(cell.Raw)
					if err == nil && nk.KeyName == h.RootKey.Name {
						return nk
					}
				}
			}
		}
		return nil
	}

	parts := strings.Split(strings.Trim(path, "\\"), "\\")
	_ = parts
	// Simplified: just return nil for non-root paths
	// Full implementation would traverse the tree
	return nil
}

// GetAllKeys returns all keys in the hive.
func (h *RegistryHive) GetAllKeys() []*ParsedKey {
	var keys []*ParsedKey
	h.WalkKeys(func(key *ParsedKey, depth int) {
		keys = append(keys, key)
	})
	return keys
}

// GetKeyValues returns all values for a key path.
func (h *RegistryHive) GetKeyValues(path string) ([]*ParsedValue, error) {
	key, err := h.OpenKey(path)
	if err != nil {
		return nil, err
	}
	return key.Values, nil
}

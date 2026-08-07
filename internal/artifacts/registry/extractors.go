package registry

import (
	"fmt"
	"strings"
	"time"
)

// SystemInfo contains extracted OS information.
type SystemInfo struct {
	ComputerName    string `json:"computer_name"`
	OSVersion       string `json:"os_version"`
	CurrentBuild    string `json:"current_build"`
	InstallDate     string `json:"install_date"`
	RegisteredOwner string `json:"registered_owner"`
	ProductName     string `json:"product_name"`
	EditionID       string `json:"edition_id"`
}

// RunKeyEntry represents an autostart entry.
type RunKeyEntry struct {
	KeyPath     string `json:"key_path"`
	ValueName   string `json:"value_name"`
	ValueData   string `json:"value_data"`
	DataType    string `json:"data_type"`
	Timestamp   string `json:"timestamp"`
}

// ServiceEntry represents a Windows service.
type ServiceEntry struct {
	Name        string `json:"name"`
	ImagePath   string `json:"image_path"`
	StartType   uint32 `json:"start_type"`
	Type        uint32 `json:"type"`
	ErrorControl uint32 `json:"error_control"`
}

// InstalledSoftware represents installed software.
type InstalledSoftware struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Publisher   string `json:"publisher"`
	InstallDate string `json:"install_date"`
}

// UserAssistEntry represents a UserAssist entry.
type UserAssistEntry struct {
	Name      string `json:"name"`
	ValueName string `json:"value_name"`
	RawData   string `json:"raw_data"`
}

// ExtractSystemInfo extracts OS information from SYSTEM and SOFTWARE hives.
func ExtractSystemInfo(systemHive, softwareHive *RegistryHive) (*SystemInfo, error) {
	info := &SystemInfo{}

	// From SYSTEM hive
	if systemHive != nil {
		// ComputerName
		key, err := systemHive.OpenKey(`ControlSet001\Control\ComputerName\ComputerName`)
		if err == nil {
			for _, val := range key.Values {
				if strings.EqualFold(val.Name, "ComputerName") {
					info.ComputerName = strings.TrimRight(string(val.Data), "\x00")
				}
			}
		}
	}

	// From SOFTWARE hive
	if softwareHive != nil {
		key, err := softwareHive.OpenKey(`Microsoft\Windows NT\CurrentVersion`)
		if err == nil {
			for _, val := range key.Values {
				data := strings.TrimRight(string(val.Data), "\x00")
				switch strings.ToLower(val.Name) {
				case "productname":
					info.ProductName = data
				case "currentbuild":
					info.CurrentBuild = data
				case "registeredowner":
					info.RegisteredOwner = data
				case "editionid":
					info.EditionID = data
				case "installtime":
					info.InstallDate = data
				}
			}
		}
		info.OSVersion = info.ProductName
	}

	return info, nil
}

// ExtractRunKeys extracts Run/RunOnce autostart entries.
func ExtractRunKeys(hive *RegistryHive) []RunKeyEntry {
	var results []RunKeyEntry

	runKeyPaths := []string{
		`Microsoft\Windows\CurrentVersion\Run`,
		`Microsoft\Windows\CurrentVersion\RunOnce`,
		`Microsoft\Windows\CurrentVersion\RunServices`,
		`Microsoft\Windows\CurrentVersion\RunServicesOnce`,
		`Microsoft\Windows NT\CurrentVersion\Winlogon`,
		`Microsoft\Windows NT\CurrentVersion\Windows`,
		`Microsoft\Windows\CurrentVersion\Policies\Explorer\Run`,
		`Microsoft\Windows NT\CurrentVersion\Terminal Server\Install\Software\Microsoft\Windows\CurrentVersion\Run`,
	}

	for _, path := range runKeyPaths {
		key, err := hive.OpenKey(path)
		if err != nil {
			continue
		}
		for _, val := range key.Values {
			results = append(results, RunKeyEntry{
				KeyPath:   path,
				ValueName: val.Name,
				ValueData: FormatValueData(val),
				DataType:  dataTypeName(val.DataType),
				Timestamp: key.Timestamp.Format(time.RFC3339),
			})
		}
	}

	return results
}

// ExtractServices extracts services from the SYSTEM hive.
func ExtractServices(hive *RegistryHive) []ServiceEntry {
	var results []ServiceEntry

	servicesKey, err := hive.OpenKey(`ControlSet001\Services`)
	if err != nil {
		return results
	}

	for _, subkey := range servicesKey.Subkeys {
		entry := ServiceEntry{
			Name: subkey.Name,
		}

		// Load values for each service
		svcKey, err := hive.OpenKey(subkey.Path)
		if err == nil {
			for _, val := range svcKey.Values {
				switch strings.ToLower(val.Name) {
				case "imagepath":
					entry.ImagePath = FormatValueData(val)
				case "start":
					if len(val.Data) >= 4 {
						entry.StartType = uint32(val.Data[0])
					}
				case "type":
					if len(val.Data) >= 4 {
						entry.Type = uint32(val.Data[0])
					}
				case "errorcontrol":
					if len(val.Data) >= 4 {
						entry.ErrorControl = uint32(val.Data[0])
					}
				}
			}
		}

		results = append(results, entry)
	}

	return results
}

// ExtractInstalledSoftware extracts installed software list.
func ExtractInstalledSoftware(hive *RegistryHive) []InstalledSoftware {
	var results []InstalledSoftware

	uninstallKey, err := hive.OpenKey(`Microsoft\Windows\CurrentVersion\Uninstall`)
	if err != nil {
		return results
	}

	for _, subkey := range uninstallKey.Subkeys {
		entry := InstalledSoftware{
			Name: subkey.Name,
		}

		swKey, err := hive.OpenKey(subkey.Path)
		if err == nil {
			for _, val := range swKey.Values {
				switch strings.ToLower(val.Name) {
				case "displayname":
					entry.Name = FormatValueData(val)
				case "displayversion":
					entry.Version = FormatValueData(val)
				case "publisher":
					entry.Publisher = FormatValueData(val)
				case "installdate":
					entry.InstallDate = FormatValueData(val)
				}
			}
		}

		if entry.Name != "" {
			results = append(results, entry)
		}
	}

	return results
}

// ExtractUserAssist extracts UserAssist entries (recently executed programs).
func ExtractUserAssist(hive *RegistryHive) []UserAssistEntry {
	var results []UserAssistEntry

	userAssistKey, err := hive.OpenKey(`Software\Microsoft\Windows\CurrentVersion\Explorer\UserAssist`)
	if err != nil {
		return results
	}

	for _, subkey := range userAssistKey.Subkeys {
		for _, val := range subkey.Values {
			results = append(results, UserAssistEntry{
				Name:      subkey.Name,
				ValueName: val.Name,
				RawData:   FormatValueData(val),
			})
		}
	}

	return results
}

// GetRegistryArtifacts returns all extracted artifacts from a registry hive.
func GetRegistryArtifacts(hive *RegistryHive, hiveType string) map[string]interface{} {
	result := make(map[string]interface{})

	switch strings.ToUpper(hiveType) {
	case "SYSTEM":
		info, _ := ExtractSystemInfo(hive, nil)
		result["system_info"] = info
		result["services"] = ExtractServices(hive)

	case "SOFTWARE":
		info, _ := ExtractSystemInfo(nil, hive)
		result["system_info"] = info
		result["installed_software"] = ExtractInstalledSoftware(hive)
		result["run_keys"] = ExtractRunKeys(hive)

	case "SAM":
		// SAM parsing is more complex, return basic info
		result["note"] = "SAM hive parsing requires specialized user account extraction"

	case "NTUSER.DAT":
		result["run_keys"] = ExtractRunKeys(hive)
		result["user_assist"] = ExtractUserAssist(hive)

	default:
		result["run_keys"] = ExtractRunKeys(hive)
	}

	return result
}

func dataTypeName(dt uint32) string {
	switch dt {
	case TypeString:
		return "REG_SZ"
	case TypeExpandString:
		return "REG_EXPAND_SZ"
	case TypeBinary:
		return "REG_BINARY"
	case TypeDWORD:
		return "REG_DWORD"
	case TypeMultiString:
		return "REG_MULTI_SZ"
	case TypeQWORD:
		return "REG_QWORD"
	default:
		return fmt.Sprintf("REG_UNKNOWN(0x%X)", dt)
	}
}

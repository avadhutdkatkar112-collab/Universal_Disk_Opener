# Universal Disk Platform

A forensic disk image analyzer built with [Wails v2](https://wails.io/) (Go backend + React/TypeScript frontend).

## Features

- **Disk Image Support**: Open VHD, VHDX, VDI, VMDK, QCOW2, and RAW disk images
- **Filesystem Parsing**: NTFS, ext4, FAT16, FAT32, exFAT filesystem support
- **Partition Discovery**: Automatic GPT/MBR partition table parsing
- **Hex Viewer**: Binary file inspection with Data Inspector (UInt8/16/32, Float32, timestamps)
- **Text Editor**: CodeMirror 6 editor with syntax highlighting for 20+ languages
- **Read-Only Enforcement**: All evidence access is forensic-safe and read-only
- **Timeline Analysis**: Event timeline reconstruction from filesystem metadata
- **Sigma Rule Scanning**: Security log analysis with Sigma rule engine
- **Filesystem Browser**: Navigate directories, view file sizes, switch between hex and text views

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Desktop Framework | Wails v2 |
| Backend | Go 1.21+ |
| Frontend | React 18 + TypeScript + Vite |
| Styling | Tailwind CSS + CSS Variables |
| Code Editor | CodeMirror 6 |
| State | Zustand |

## Prerequisites

- [Go 1.21+](https://go.dev/dl/)
- [Node.js 18+](https://nodejs.org/)
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

## Development

```bash
# Install frontend dependencies
cd frontend && npm install && cd ..

# Run in dev mode
wails dev

# Build for production
wails build
```

## Project Structure

```
vhd-opener/
├── main.go                    # Application entry point
├── wails.json                 # Wails configuration
├── go.mod / go.sum            # Go dependencies
├── internal/
│   ├── domain/
│   │   ├── disk/              # Disk image format drivers (VHD/VHDX/VDI/VMDK/QCOW2/RAW)
│   │   ├── filesystem/        # Filesystem parsers (NTFS/ext4/FAT16/FAT32/exFAT)
│   │   ├── partition/         # GPT/MBR partition table parsing
│   │   ├── vfs/               # Virtual filesystem layer bridging disks + filesystems
│   │   ├── vhd/               # VHD format driver
│   │   ├── vhdx/              # VHDX format driver
│   │   ├── vmdk/              # VMDK format driver
│   │   ├── qcow2/             # QCOW2 format driver
│   │   ├── vdi/               # VDI format driver
│   │   └── raw/               # Raw disk driver
│   ├── ui/                    # Wails-bound handlers (App, StorageHandler, etc.)
│   ├── platform/              # Platform services (gateway, eventbus, workspace)
│   ├── engine/                # Command engine (DIE - Dynamic Intent Evaluation)
│   ├── artifacts/             # Forensic artifact parsers (EVTX, MFT, Registry)
│   ├── sigma/                 # Sigma rule engine
│   ├── timeline/              # Timeline reconstruction
│   ├── vault/                 # Evidence vault (crypto, manifest, audit)
│   └── capabilities/          # Hash, search, and analysis capabilities
├── frontend/
│   ├── src/
│   │   ├── components/        # React UI components
│   │   ├── store/             # State management (evidenceStore, diskStore)
│   │   ├── lib/               # Wails API bindings, utilities
│   │   └── styles/            # CSS tokens, theme
│   └── wailsjs/               # Auto-generated Wails bindings
└── build/
    ├── appicon.png
    └── windows/               # Windows-specific build assets
```

## License

MIT

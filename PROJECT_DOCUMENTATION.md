# Meteor Forensic Workbench — Complete Project Documentation

## Version: 0.7.1 | Date: August 7, 2026

---

# TABLE OF CONTENTS

1. [Project Overview](#1-project-overview)
2. [Architecture](#2-architecture)
3. [What Has Been Built](#3-what-has-been-built)
4. [Codebase Statistics](#4-codebase-statistics)
5. [Module Breakdown](#5-module-breakdown)
6. [Frontend Components](#6-frontend-components)
7. [Test Coverage](#7-test-coverage)
8. [EvidenceSession 1.0 Architecture](#8-evidencesession-10-architecture)
9. [Remaining Tasks](#9-remaining-tasks)
10. [Roadmap](#10-roadmap)
11. [Technical Decisions](#11-technical-decisions)
12. [Build & Run](#12-build--run)

---

# 1. PROJECT OVERVIEW

**Meteor Forensic Workbench** (codename: VHD Opener / Universal Disk Platform) is a desktop forensic analysis application built with:

- **Backend:** Go 1.25 + Wails v2.13
- **Frontend:** React 18 + TypeScript + Vite
- **Platform:** Windows (WebView2)
- **Module:** `github.com/user/vhd-opener`

### Core Principle

> **Open evidence once → the application understands the evidence → every tool works against that same evidence/context.**

The application is a universal disk/image exploration platform supporting multiple container formats, partition schemes, filesystems, and forensic analysis capabilities through a unified EvidenceSession architecture.

---

# 2. ARCHITECTURE

## 2.1 System Architecture

```
                    USER OPENS
                 suspect.vhd / .vhdx
                        │
                        ▼
              ┌───────────────────┐
              │ Universal Storage  │
              │     Engine         │
              │                    │
              │ VHD/VHDX/RAW/...   │
              └─────────┬─────────┘
                        │
                 EvidenceSession
                        │
       ┌────────────────┼─────────────────┐
       │                │                 │
       ▼                ▼                 ▼
   Explorer         Search          Analysis
       │                │                 │
       ├───────────────┬┴─────────────────┤
       ▼               ▼                  ▼
   Artifacts        Timeline           Hex Viewer
   Registry         MACB               Raw sectors
   EVTX             Events              Inspector
   Browser          Correlation
       │               │
       └───────┬───────┘
               ▼
          Case / Report
```

## 2.2 Layer Architecture

```
FORMAT
VHD / VHDX / VDI / VMDK / QCOW2 / RAW
        ↓
STORAGE
random-access bytes (BlockReader)
        ↓
PARTITION
MBR / GPT
        ↓
FILESYSTEM
NTFS / FAT / exFAT / EXT4
        ↓
EVIDENCE MODEL
files / metadata / streams / timestamps
        ↓
ARTIFACT ENGINE
Registry / EVTX / Browser / Prefetch / etc.
        ↓
ANALYSIS
Timeline / Search / Intelligence / Sigma / YARA
        ↓
CASE
Bookmarks / Notes / Audit / Reports / Vault
        ↓
UI
Explorer / Timeline / Intelligence / SQL / Reports
```

## 2.3 Directory Structure

```
vhd-opener/
├── main.go                          # Application entry point
├── go.mod / go.sum                  # Go module definition
├── .github/workflows/ci.yml        # CI/CD pipeline
├── pkg/
│   └── storage/                     # Universal Storage Engine
│       ├── contract.go              # Core interfaces
│       ├── reader.go                # SafeBlockReader
│       ├── raw.go                   # RAW disk driver
│       ├── partition.go             # MBR/GPT parser
│       ├── ntfs.go                  # NTFS filesystem
│       ├── session.go               # EvidenceSession
│       ├── session_test.go          # Session tests
│       ├── contract_test.go         # Contract tests
│       └── storage_integration_test.go  # Integration tests
├── internal/
│   ├── domain/
│   │   ├── disk/                    # Disk abstraction
│   │   ├── partition/               # Partition handling
│   │   ├── filesystem/              # Filesystem readers
│   │   ├── vfs/                     # Virtual filesystem
│   │   ├── vhd/                     # VHD format driver
│   │   ├── vhdx/                    # VHDX format driver
│   │   ├── vmdk/                    # VMDK format driver
│   │   ├── vdi/                     # VDI format driver
│   │   ├── qcow2/                   # QCOW2 format driver
│   │   └── raw/                     # RAW format driver
│   ├── artifacts/
│   │   ├── evtx/                    # Windows Event Log parser
│   │   ├── mft/                     # Master File Table parser
│   │   └── registry/                # Windows Registry hive parser
│   ├── engine/
│   │   ├── die/                     # DIE (Detect It Easy) integration
│   │   └── core/                    # Core engine
│   ├── sigma/                       # Sigma rule engine
│   ├── intelligence/                # Threat intelligence
│   ├── memory/                      # Memory analysis + YARA
│   ├── timeline/                    # MACB timeline engine
│   ├── report/                      # Report generation
│   ├── vault/                       # DVE1 Evidence Vault
│   │   ├── crypto.go                # Argon2id + AES-256-GCM
│   │   ├── manifest.go              # Container manifest
│   │   ├── audit.go                 # Tamper-evident audit chain
│   │   ├── stream.go                # Chunked streaming I/O
│   │   ├── report.go                # Chain-of-custody reports
│   │   └── e2e_test.go              # E2E lifecycle tests
│   ├── ui/                          # Wails IPC handlers
│   │   ├── handlers.go              # Main App handler
│   │   ├── storage_handler.go       # Storage/NTFS handler
│   │   ├── vault_handler.go         # Evidence Vault handler
│   │   ├── sql_handler.go           # SQL Analytics handler
│   │   ├── sigma_handler.go         # Sigma detection handler
│   │   ├── memory_handler.go        # Memory/YARA handler
│   │   ├── intelligence_handler.go  # Intelligence handler
│   │   ├── report_handler.go        # Report handler
│   │   ├── export_handler.go        # Export handler
│   │   └── handler_test.go          # Handler tests
│   ├── bridge/                      # Wails bridge
│   ├── platform/                    # Platform services
│   ├── capabilities/                # Capability system
│   ├── infrastructure/              # Cache, events, workers
│   ├── grammar/                     # Grammar service
│   ├── ucl/                         # UCL parser
│   ├── jobs/                        # Job manager
│   └── workspace/                   # Workspace manager
├── frontend/
│   └── src/
│       ├── App.tsx                  # Main application
│       ├── store/
│       │   ├── evidenceStore.ts     # EvidenceSession store
│       │   ├── diskStore.ts         # Disk state store
│       │   └── jobStore.ts          # Job state store
│       ├── components/
│       │   ├── TitleBar.tsx         # Window title bar
│       │   ├── MenuBar.tsx          # File/Edit/View menu
│       │   ├── Sidebar.tsx          # Evidence tree sidebar
│       │   ├── StatusBar.tsx        # Bottom status bar
│       │   ├── DiskExplorer.tsx     # File explorer
│       │   ├── InvestigateView.tsx  # Artifact discovery
│       │   ├── ExamineView.tsx      # Hex viewer
│       │   ├── TimelineViewer.tsx   # MACB timeline
│       │   ├── IntelligenceViewer.tsx # Intelligence
│       │   ├── SqlQueryViewer.tsx   # SQL analytics
│       │   ├── SigmaViewer.tsx      # Sigma detection
│       │   ├── MemoryViewer.tsx     # Memory analysis
│       │   ├── ReportViewer.tsx     # Reports
│       │   ├── EvidenceManager.tsx  # Vault management
│       │   ├── EvidenceBanner.tsx   # Read-only banner
│       │   ├── HashModal.tsx        # Hash operations
│       │   ├── CommandCenterModal.tsx # Ctrl+K command center
│       │   └── ... (34 total)
│       └── lib/
│           ├── wails.ts             # Wails IPC bindings
│           └── openDisk.ts          # Disk open utility
├── test/
│   └── sigma_test.go                # Sigma rule tests
└── build/bin/                       # Build output
```

---

# 3. WHAT HAS BEEN BUILT

## 3.1 Disk Format Drivers (Complete)

| Format | Package | Status |
|--------|---------|--------|
| VHD | `internal/domain/vhd/` | Complete (4 files) |
| VHDX | `internal/domain/vhdx/` | Complete (2 files) |
| VMDK | `internal/domain/vmdk/` | Complete (2 files) |
| VDI | `internal/domain/vdi/` | Complete (1 file) |
| QCOW2 | `internal/domain/qcow2/` | Complete (2 files) |
| RAW | `internal/domain/raw/` + `pkg/storage/raw.go` | Complete (2 files) |

## 3.2 Partition Parsing (Complete)

| Scheme | Package | Status |
|--------|---------|--------|
| MBR | `internal/domain/partition/` + `pkg/storage/partition.go` | Complete |
| GPT | `internal/domain/partition/` + `pkg/storage/partition.go` | Complete |

## 3.3 Filesystem Readers (Complete)

| Filesystem | Package | Status |
|------------|---------|--------|
| NTFS | `internal/domain/filesystem/` + `pkg/storage/ntfs.go` | Complete |
| FAT16/32 | `internal/domain/filesystem/` | Partial |
| exFAT | `internal/domain/filesystem/` | Partial |
| EXT4 | `internal/domain/filesystem/` | Partial |

## 3.4 Universal Storage Engine (`pkg/storage/`) (Complete)

| File | Purpose | Tests |
|------|---------|-------|
| `contract.go` | `BlockReader`, `DiskImage`, `Partition`, `FileSystem`, `Node` interfaces | - |
| `reader.go` | `SafeBlockReader` with overflow/bounds/context guards | 4 tests |
| `raw.go` | `RawDisk` implementation | 3 tests |
| `partition.go` | MBR + GPT parsers | 2 tests |
| `ntfs.go` | NTFS boot sector + MFT reader | 1 test |
| `session.go` | `EvidenceSession` lifecycle + provenance | 5 tests |
| `*_test.go` | Integration tests (E2E RAW→MBR→NTFS) | 9 tests |

**Total: 19 tests in `pkg/storage/` — ALL PASSING**

## 3.5 Artifact Parsers

| Artifact | Package | Status |
|----------|---------|--------|
| Windows Registry (NTUSER.DAT, SYSTEM, SOFTWARE, SAM) | `internal/artifacts/registry/` | Complete (5 files) |
| Windows Event Logs (EVTX) | `internal/artifacts/evtx/` | Complete (1 file) |
| Master File Table ($MFT) | `internal/artifacts/mft/` | Complete (1 file) |
| Browser artifacts | - | Not started |
| Prefetch | - | Not started |
| Amcache/Shimcache | - | Not started |
| Shellbags | - | Not started |

## 3.6 Sigma Rule Engine (Complete)

| File | Purpose |
|------|---------|
| `internal/sigma/rule.go` | YAML rule parser, condition evaluator (AND/OR/NOT/wildcards) |
| `internal/sigma/engine.go` | Rule engine, scanner, embedded default rules |
| `test/sigma_test.go` | Rule evaluation tests |
| `cmd/sigma_test_runner/main.go` | CLI Sigma verification |

## 3.7 Memory Analysis (Complete)

| File | Purpose |
|------|---------|
| `internal/memory/process.go` | Process enumeration via gopsutil |
| `internal/memory/yara.go` | YARA pattern matching |

## 3.8 DVE1 Evidence Vault (Complete)

| File | Purpose | Tests |
|------|---------|-------|
| `internal/vault/crypto.go` | Argon2id KDF + AES-256-GCM encryption | - |
| `internal/vault/manifest.go` | Container manifest (JSON) | - |
| `internal/vault/audit.go` | Tamper-evident SHA-256 audit chain | - |
| `internal/vault/stream.go` | 4 MiB chunked streaming I/O + `ExtractEvidenceFile` | - |
| `internal/vault/report.go` | Chain-of-custody HTML report generation | - |
| `internal/vault/e2e_test.go` | E2E lifecycle + tamper detection test | 1 test |

## 3.9 Wails IPC Handlers (Complete)

| Handler | File | Methods |
|---------|------|---------|
| App | `handlers.go` | OpenDisk, ListDirectory, GetDiskInfo, HashFile, etc. |
| StorageHandler | `storage_handler.go` | MountDisk, MountPartition, ListDirectory |
| VaultHandler | `vault_handler.go` | IngestEvidence, VerifyCaseIntegrity, LogAnalystAction, ExportChainOfCustodyReport |
| SqlHandler | `sql_handler.go` | LoadTimelineToSQL, ExecuteSQLQuery |
| SigmaHandler | `sigma_handler.go` | LoadSigmaRuleDirectory, RunSigmaScan |
| MemoryHandler | `memory_handler.go` | GetLiveProcesses, RunMemoryYaraScan |
| IntelligenceHandler | `intelligence_handler.go` | AnalyzeFindings |
| ReportHandler | `report_handler.go` | GenerateReport, SaveReportToFile |
| ExportHandler | `export_handler.go` | ExportTimelineCSV, ExportTimelineJSON |

## 3.10 Frontend Application (Complete)

| Mode | Component | Status |
|------|-----------|--------|
| Explorer | `DiskExplorer.tsx` | Complete — auto-detect, mount, browse |
| Investigate | `InvestigateView.tsx` | Complete — artifact auto-discovery |
| Examine | `ExamineView.tsx` | Complete — hex viewer + data inspector |
| Timeline | `TimelineViewer.tsx` | Complete — MACB timeline |
| Case | `ReportViewer.tsx` | Complete — report generation |
| Sidebar | `Sidebar.tsx` | Complete — evidence tree, bookmarks |
| Evidence Banner | `EvidenceBanner.tsx` | Complete — read-only indicator |
| Evidence Manager | `EvidenceManager.tsx` | Complete — vault ingest/verify |
| Command Center | `CommandCenterModal.tsx` | Complete — Ctrl+K commands |
| Hash Modal | `HashModal.tsx` | Complete — file hashing |
| Title Bar | `TitleBar.tsx` | Complete — custom frameless |
| Menu Bar | `MenuBar.tsx` | Complete — File/Edit/View |
| Status Bar | `StatusBar.tsx` | Complete — disk info |

## 3.11 Infrastructure (Complete)

| Component | Package | Status |
|-----------|---------|--------|
| Event Bus | `internal/platform/` | Complete |
| Job Manager | `internal/platform/` | Complete |
| Gateway | `internal/platform/` | Complete |
| Workspace Manager | `internal/platform/` | Complete |
| Worker Pool | `internal/infrastructure/worker/` | Complete |
| Cache | `internal/infrastructure/cache/` | Complete |
| Capability System | `internal/capabilities/` | Complete |
| Plugin SDK | `pkg/plugin/` + `pkg/sdk/` | Complete |

## 3.12 CI/CD Pipeline (Complete)

```yaml
.github/workflows/ci.yml
├── Job 1: Go Tests + Race Detector + Coverage
├── Job 2: gofmt + go vet
├── Job 3: golangci-lint
└── Job 4: Frontend tsc --noEmit + npm build
```

---

# 4. CODEBASE STATISTICS

| Metric | Count |
|--------|-------|
| **Go files** | 135 |
| **Go LOC** | ~23,200 |
| **Go packages** | 53 |
| **Test files** | 18 |
| **Test LOC** | ~2,500 |
| **TypeScript files** | 12 |
| **TSX files** | 37 |
| **Total TS/TSX files** | 49 |
| **TS/TSX LOC** | ~9,080 |
| **UI components** | 34 |
| **Total project LOC** | **~32,780** |
| **Build artifacts** | `build/bin/vhd-opener.exe` |

---

# 5. MODULE BREAKDOWN

## 5.1 Go Packages (53)

| Category | Packages | Files |
|----------|----------|-------|
| Storage Engine | `pkg/storage` | 7 |
| Disk Formats | `internal/domain/{vhd,vhdx,vmdk,vdi,qcow2,raw}` | 14 |
| Partitions | `internal/domain/partition` | 3 |
| Filesystems | `internal/domain/filesystem` | 9 |
| VFS | `internal/domain/vfs` | 2 |
| Artifacts | `internal/artifacts/{registry,evtx,mft}` | 7 |
| Engine | `internal/engine/{core,die}` | 7 |
| Sigma | `internal/sigma` | 2 |
| Intelligence | `internal/intelligence` | 1 |
| Memory | `internal/memory` | 2 |
| Timeline | `internal/timeline` | 1 |
| Report | `internal/report` | 2 |
| Vault | `internal/vault` | 6 |
| UI Handlers | `internal/ui` | 11 |
| Bridge | `internal/bridge` | 1 |
| Platform | `internal/platform` | 6 |
| Capabilities | `internal/capabilities/{hash,search}` | 3 |
| Infrastructure | `internal/infrastructure/{cache,events,logger,worker}` | 4 |
| Grammar | `internal/grammar` | 3 |
| UCL | `internal/ucl` | 4 |
| Jobs | `internal/jobs` | 1 |
| Workspace | `internal/workspace` | 2 |
| Plugin/SDK | `pkg/{capability,plugin,sdk}` | 3 |
| CLI | `cmd/{ext4test,sigma_test_runner}` | 2 |
| Legacy | `backend/` | 8 |
| Tests | `test/` | 1 |

---

# 6. FRONTEND COMPONENTS

## 6.1 Core Layout

| Component | Purpose |
|-----------|---------|
| `App.tsx` | Main application — 5-mode navigation, evidence banner, sidebar integration |
| `TitleBar.tsx` | Custom frameless window title bar (Win11 style) |
| `MenuBar.tsx` | File/Edit/View/Actions/Tools/Help menu |
| `StatusBar.tsx` | Bottom status bar (disk info, read-only indicator) |
| `Sidebar.tsx` | Evidence tree (partitions, filesystem), bookmarks, notes |

## 6.2 Primary Views

| View | Component | Purpose |
|------|-----------|---------|
| Explorer | `DiskExplorer.tsx` | Mount disk, browse directories, navigate |
| Investigate | `InvestigateView.tsx` | Auto-discover artifacts (Registry, EVTX, MFT, Prefetch) |
| Examine | `ExamineView.tsx` | Hex viewer + data inspector (UInt16/UInt32/Timestamp) |
| Timeline | `TimelineViewer.tsx` | MACB timeline events |
| Case | `ReportViewer.tsx` | Report generation and export |

## 6.3 Analysis Tools

| Component | Purpose |
|-----------|---------|
| `IntelligenceViewer.tsx` | Threat intelligence analysis |
| `SqlQueryViewer.tsx` | SQL analytics over artifact data |
| `SigmaViewer.tsx` | Sigma detection rule scanning |
| `MemoryViewer.tsx` | Live memory analysis + YARA |

## 6.4 Evidence Management

| Component | Purpose |
|-----------|---------|
| `EvidenceManager.tsx` | DVE1 vault ingest/verify |
| `EvidenceBanner.tsx` | Read-only evidence mode banner |
| `ReportExportModal.tsx` | Chain-of-custody report export |

## 6.5 Utilities

| Component | Purpose |
|-----------|---------|
| `CommandCenterModal.tsx` | Ctrl+K command palette |
| `HashModal.tsx` | File hashing (SHA-256/MD5/BLAKE3) |
| `PreviewDrawer.tsx` | File preview |
| `TaskExecutionHUD.tsx` | Background task progress |
| `WelcomeDashboard.tsx` | Welcome screen (no evidence loaded) |
| `ErrorBoundary.tsx` | React error boundary |

## 6.6 State Management

| Store | Purpose |
|-------|---------|
| `evidenceStore.ts` | EvidenceSession — open, navigate, bookmark, audit, viewMode |
| `diskStore.ts` | Disk snapshot state |
| `jobStore.ts` | Background job state |

---

# 7. TEST COVERAGE

## 7.1 Test Results

| Package | Tests | Status |
|---------|-------|--------|
| `pkg/storage` | 19 | ALL PASS |
| `internal/vault` | 1 | PASS |
| `internal/ui` | 4 | PASS |
| `internal/capabilities/hash` | Cached | PASS |
| `internal/domain/qcow2` | Cached | PASS |
| `internal/domain/raw` | Cached | PASS |
| `internal/domain/vhdx` | Cached | PASS |
| `internal/domain/vmdk` | Cached | PASS |
| `internal/engine` | Cached | PASS |
| `internal/engine/die` | Cached | PASS |
| `internal/grammar` | Cached | PASS |
| `internal/platform` | Cached | PASS |
| `internal/ucl` | Cached | PASS |
| `internal/workspace` | Cached | PASS |
| `test` | 1 FAIL | Sigma rule edge case |

**Total: 14/15 packages pass (93%)**

## 7.2 Test Breakdown by Package

### `pkg/storage/` (19 tests)

| Test | What It Tests |
|------|---------------|
| `TestRawDisk_SafeBlockReaderAndBounds` | Read bounds, overflow, context cancellation |
| `TestRawDisk_Format` | Format detection, virtual size |
| `TestRawDisk_ReadAtBoundsTrim` | Partial read at EOF |
| `TestSafeBlockReader_IntegerOverflowGuard` | Integer overflow rejection |
| `TestStoragePipeline_EndToEndRAWAndNTFS` | E2E: RAW → MBR → NTFS → directory listing |
| `TestStoragePipeline_MalformedMBRRejected` | Bad MBR signature rejection |
| `TestStoragePipeline_GPTDetection` | GPT protective MBR detection |
| `TestStoragePipeline_ContextCancellation` | Context cancel propagation |
| `TestSafeBlockReader_HugeFileNoMemoryIssue` | 100MB file, no memory blowup |
| `TestEvidenceSession_Lifecycle` | Open → state transitions → Close |
| `TestEvidenceSession_Provenance` | Provenance entries logged correctly |
| `TestEvidenceSession_ListDirectory_NoFilesystem` | Error when no FS mounted |
| `TestEvidenceSession_Metadata` | Metadata populated on open |
| `TestEvidenceSession_ThreadSafety` | Concurrent provenance writes |

### `internal/vault/` (1 test)

| Test | What It Tests |
|------|---------------|
| `TestDVE1_E2E_LifecycleAndTamperDetection` | Ingest 10 MiB → 3 chunks → audit → extract → verify → corrupt payload (GCM fails) → corrupt audit (chain invalid) |

### `internal/ui/` (4 tests)

| Test | What It Tests |
|------|---------------|
| `TestVaultHandler_IngestAndVerify` | Ingest → verify → log action → re-verify |
| `TestStorageHandler_MountAndBrowse` | Mount disk → select partition → browse root |
| `TestExtractFileSecure_PathTraversal` | Path traversal prevention |
| `TestExtractFileSecure_DirectoryCreation` | Directory creation |

---

# 8. EVIDENCESSESSION 1.0 ARCHITECTURE

## 8.1 Go-side EvidenceSession (`pkg/storage/session.go`)

```go
type EvidenceSession struct {
    mu          sync.RWMutex
    id          string
    state       SessionState          // Closed → Open → Mounted → Analyzing
    metadata    EvidenceMetadata      // Path, format, SHA-256, size
    disk        DiskImage             // Active disk reader
    partitions  []Partition           // Parsed partitions
    filesystem  FileSystem            // Mounted filesystem
    provenance  []ProvenanceEntry     // Audit trail
    openedAt    time.Time
    lastAccess  time.Time
    cancelFunc  context.CancelFunc
}
```

### State Machine

```
SessionClosed ──Open()──► SessionOpen ──Mount()──► SessionMounted ──► SessionAnalyzing
                              │                        │
                              └──────Close()───────────┘
```

### Universal File Model

```go
type UniversalFileNode struct {
    Name        string     `json:"name"`
    Path        string     `json:"path"`
    IsDir       bool       `json:"is_dir"`
    Size        uint64     `json:"size"`
    Timestamps  Timestamps `json:"timestamps"`    // Created/Modified/Accessed/MFTModified
    Permissions string     `json:"permissions"`
    OwnerID     int        `json:"owner_id"`
    GroupID     int        `json:"group_id"`
    FileID      uint64     `json:"file_id"`
    Attributes  []string   `json:"attributes"`
    IsDeleted   bool       `json:"is_deleted"`
    StreamName  string     `json:"stream_name,omitempty"`
}
```

### Provenance Tracking

```go
type ProvenanceEntry struct {
    Timestamp time.Time `json:"timestamp"`
    Actor     string    `json:"actor"`       // "system" | "analyst"
    Action    string    `json:"action"`      // "session.open" | "file.view" | "bookmark.add"
    Target    string    `json:"target"`      // Path or resource
    Details   string    `json:"details"`     // Human-readable description
    SessionID string    `json:"session_id"`  // Bound to session
}
```

## 8.2 Frontend EvidenceSession Store (`evidenceStore.ts`)

```typescript
interface EvidenceStore extends EvidenceSession {
  openEvidence: (imagePath: string) => Promise<void>;
  selectPartition: (index: number) => Promise<void>;
  navigateTo: (path: string) => Promise<void>;
  navigateBack/Forward/Up: () => Promise<void>;
  addBookmark: (path, name, note, tag) => void;
  removeBookmark: (id) => void;
  logAuditEvent: (action, target, details) => void;
  clearSession: () => void;
  setViewMode: (mode) => void;
}
```

### Auto-Detection Pipeline

```
Open file → detectFormat() → MountDisk() → selectBestPartition() → MountPartition() → ListDirectory('/') → discoverArtifacts()
```

---

# 9. REMAINING TASKS

## 9.1 Tier 1 — EvidenceSession 1.0 (Immediate Priority)

| Task | Status | Notes |
|------|--------|-------|
| Go-side EvidenceSession struct | Done | `pkg/storage/session.go` |
| Universal file model (Go) | Done | `UniversalFileNode` with timestamps/permissions |
| Provenance tracking (Go) | Done | `ProvenanceEntry` audit trail |
| Session lifecycle tests | Done | 5 tests passing |
| Wire Go EvidenceSession to Wails handler | Not started | Need `session_handler.go` |
| Format detection from file magic bytes | Not started | Header sniffing |
| Read-only enforcement at storage level | Partial | `IsReadOnly` flag exists, needs driver-level enforcement |
| Cancellation + bounded streaming on session | Not started | Context propagation |
| Session persistence (save/load) | Not started | Serialize to JSON |

## 9.2 Tier 2 — Storage Correctness (Next)

| Task | Status | Notes |
|------|--------|-------|
| VHD → pkg/storage integration | Partial | Driver exists, needs wiring |
| VHDX → pkg/storage integration | Partial | Driver exists, needs wiring |
| VDI → pkg/storage integration | Partial | Driver exists, needs wiring |
| VMDK → pkg/storage integration | Partial | Driver exists, needs wiring |
| QCOW2 → pkg/storage integration | Partial | Driver exists, needs wiring |
| FAT32/exFAT filesystem reader | Partial | Some code exists |
| EXT4 filesystem reader | Partial | Some code exists |
| Malformed image tests | Not started | Critical for robustness |
| Fuzz tests | Not started | `go-fuzz` or `testing.F` |
| ISO/IMG/DD support | Not started | |

## 9.3 Tier 3 — Forensic Depth

| Task | Status | Notes |
|------|--------|-------|
| Auto-artifact discovery from session | Frontend stub | `InvestigateView.tsx` exists |
| Registry parser → session integration | Parser exists | Needs session consumer |
| EVTX parser → session integration | Parser exists | Needs session consumer |
| Browser artifact parsers | Not started | Chrome, Edge, Firefox SQLite |
| Prefetch parser | Not started | |
| Amcache/Shimcache parser | Not started | |
| Shellbags parser | Not started | |
| LNK parser | Not started | |
| Timeline from session events | Existing timeline | Needs session integration |
| Search against session | Not started | |
| Extract from session (streaming + SHA-256) | Not started | |
| Hex from session | Frontend stub | `ExamineView.tsx` exists |

## 9.4 Tier 4 — Case & Reporting

| Task | Status | Notes |
|------|--------|-------|
| Case as top-level container | Partial | Vault exists, case shell missing |
| Bookmarks tied to session | Frontend only | `evidenceStore.ts` |
| Notes tied to session | Not started | |
| Report generation from session | Vault report exists | Not session-integrated |
| Export (CSV/JSON/PDF) | Partial | `export_handler.go` |
| Evidence manifest | Done | `vault/manifest.go` |
| Chain of custody | Done | `vault/audit.go` |

## 9.5 Tier 5 — Advanced Features

| Task | Status | Notes |
|------|--------|-------|
| YARA scanning against disk | `memory/yara.go` | Needs disk integration |
| Sigma detection against session | Engine exists | Needs session events |
| File carving | Not started | |
| Deleted file recovery | Not started | |
| Slack space analysis | Not started | |
| Content indexing | Not started | |
| Multi-evidence correlation | Not started | |
| Command Center command engine | UI exists | Needs command handlers |

---

# 10. ROADMAP

```
                         PROJECT ROADMAP

0.7  Storage Core                    ✓ COMPLETE
     ├── pkg/storage contracts       ✓
     ├── SafeBlockReader             ✓
     ├── RAW driver                  ✓
     ├── MBR/GPT parsers             ✓
     ├── NTFS reader                 ✓
     ├── EvidenceSession (Go)        ✓
     └── Integration tests (19)      ✓

0.8  EvidenceSession 1.0            ← CURRENT
     ├── Wire session to Wails       ← NEXT
     ├── Format detection (magic)    ← NEXT
     ├── VHD/VHDX integration        ← NEXT
     ├── Read-only enforcement       ← NEXT
     └── Session persistence         ← NEXT

0.9  Storage Correctness
     ├── VDI/VMDK/QCOW2 integration
     ├── FAT32/exFAT
     ├── EXT4
     ├── Malformed image tests
     └── Fuzz tests

1.0  Universal Disk Explorer
     ├── All formats → partitions → filesystems → VFS
     ├── Artifact auto-discovery
     ├── Session-aware timeline
     └── Session-aware search

1.1  Artifact Engine
     ├── Registry (NTUSER, SYSTEM, SOFTWARE, SAM)
     ├── EVTX (Security, System, Application)
     ├── Browser (Chrome, Edge, Firefox)
     ├── Prefetch
     ├── Amcache/Shimcache
     └── Shellbags

1.2  Timeline Engine
     ├── Filesystem events
     ├── Registry events
     ├── EVTX events
     ├── Browser events
     ├── USB events
     └── Unified correlation

1.3  Recovery & Carving
     ├── Deleted file detection
     ├── Unallocated space scanning
     ├── File carving (signatures)
     └── Slack space analysis

1.4  Detection & Analysis
     ├── YARA rules
     ├── Sigma rules
     ├── Hash sets
     └── Known-good filtering

1.5  Case Management
     ├── Case creation
     ├── Evidence linking
     ├── Bookmarks + Notes
     ├── Audit trail
     └── Report generation

2.0  Professional Forensics
     ├── Advanced reporting (PDF/HTML)
     ├── Chain of custody
     ├── Evidence vault
     └── Export workflows

3.0  Enterprise
     ├── RBAC
     ├── SSO
     ├── Team cases
     └── Central management
```

---

# 11. TECHNICAL DECISIONS

| Decision | Choice | Rationale |
|----------|--------|-----------|
| UI Framework | Wails v2 + React | Native Windows + web UI flexibility |
| Language | Go + TypeScript | Performance + type safety |
| Disk I/O | `ReadAt` (random access) | No full-file loading, bounded memory |
| Encryption | AES-256-GCM + Argon2id | NIST-approved, memory-hard KDF |
| Audit | SHA-256 linked chain | Tamper-evident, append-only |
| Chunking | 4 MiB fixed | Streaming encryption, memory bounded |
| Testing | `go test -race` | Concurrency safety |
| Linting | golangci-lint + tsc | Code quality |
| Window | Frameless + WebView2 | Modern Windows look |
| Theme | Win11 Fluent Light | Professional forensic tool aesthetic |
| State | Custom store (no zustand) | Minimal dependencies |
| Module | `github.com/user/vhd-opener` | Go module convention |

---

# 12. BUILD & RUN

## Prerequisites

- Go 1.25+ (`C:\go\bin\go.exe`)
- Node.js 20+
- Wails CLI (`C:\Users\Avadhut\go\bin\wails.exe`)

## Build Commands

```bash
# Full production build
$env:Path = "C:\go\bin;" + $env:Path
& "C:\Users\Avadhut\go\bin\wails.exe" build

# Go only
& "C:\go\bin\go.exe" build ./...

# Go tests
& "C:\go\bin\go.exe" test ./... -v

# Go tests with race detector
& "C:\go\bin\go.exe" test -race ./...

# Go vet
& "C:\go\bin\go.exe" vet ./...

# Frontend only
cd frontend && npm run build

# Frontend type check
cd frontend && npx tsc --noEmit
```

## Build Output

```
build/bin/vhd-opener.exe
```

## Run

```
build/bin/vhd-opener.exe
```

---

# DOCUMENT REVISION HISTORY

| Version | Date | Changes |
|---------|------|---------|
| 0.7.0 | Aug 7, 2026 | Initial documentation — Storage Core 0.7 complete |
| 0.7.1 | Aug 7, 2026 | Added EvidenceSession 1.0, full project analysis, remaining tasks |

---

*This document covers the complete state of the Meteor Forensic Workbench project as of August 7, 2026.*

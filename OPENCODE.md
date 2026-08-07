# OpenCode System Directives

## Execution Strategy
You are authorized to autonomously inspect, refactor, and write production-ready code across this codebase. Always enforce the following standards without requiring manual reminder prompts.

## Core Rules

### 1. Code Style & Architecture
- **Linear Guard Clauses:** Use early returns (`if err != nil { return err }`) to eliminate nested control flow.
- **Explicit Naming:** Use clear, self-describing names (`calculatePartitionOffset`, `inodeTableBlock`). No cryptic abbreviations.
- **Domain Decoupling:** Keep data parsing (ext4/FAT/NTFS readers), core domain logic (VFS, partition, disk), and presentation/UI (Wails handlers, React components) strictly separate.

### 2. Defensive Engineering
- **Bounds Guarding:** Check slice lengths, buffer bounds, and pointer nil-states before accessing memory offsets or indices. Treat all raw disk bytes, API responses, and array offsets as untrusted input.
- **Strict Typing:** Never use dynamic escape hatches (`interface{}`, `any`, untyped maps) unless writing low-level reflection infrastructure. Use explicit Go structs and TypeScript interfaces.
- **Contextual Error Wrapping:** Wrap all errors with descriptive context explaining the exact operation and parameters that failed (e.g. `fmt.Errorf("ext4: failed to read inode %d: %w", inodeNum, err)`). Never swallow errors.

### 3. Documentation
- Do NOT write trivial comments explaining standard syntax (e.g. `// increment i`).
- DO write comments explaining:
  - Bitwise offset shifts and filesystem spec quirks (e.g. ext4 inode layout, BGD 64-byte stride).
  - Non-obvious protocol or file-system specifications.
  - Defensive assumptions and edge-case handling rationale.

### 4. Implementation Quality
- Do NOT output partial code snippets, pseudo-code, or `// TODO: implement later` comments.
- Always provide full, compilable, end-to-end file contents.
- Verify that imports, type definitions, and variable signatures are consistent across all modified files.
- Run `go build ./...` and `go vet ./...` after changes to verify correctness.

### 5. Project Conventions
- **Backend:** Go with Wails v2 framework. Package layout: `internal/domain/` (core), `internal/application/` (services), `internal/infrastructure/` (cache, logger, events), `internal/ui/` (Wails handlers).
- **Frontend:** React 18 + TypeScript + Tailwind CSS + Zustand state + CodeMirror 6.
- **Interfaces:** `filesystem.DiskReader` wraps `disk.VirtualDisk` via `DiskAdapter`. Always use the adapter when passing disk handles to filesystem readers.
- **Error messages:** Prefix with the module name (e.g. `ext4:`, `vfs:`, `fat32:`) for traceability.

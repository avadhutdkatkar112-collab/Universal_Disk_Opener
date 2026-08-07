// Canonical type definitions for the Universal Disk Platform.
// All other files should import from this module.

// ── Disk Snapshot (runtime state) ─────────────────────────────────────────

export interface PartitionSnapshot {
  index: number;
  type: string;
  fsType: string;
  startLBA: number;
  endLBA: number;
  sizeBytes: number;
  label?: string;
}

export interface DiskSnapshot {
  fileName: string;
  filePath: string;
  format: string;
  totalSize: number;
  partitions: PartitionSnapshot[];
  activePartition: number;
}

// ── Wails IPC Types ───────────────────────────────────────────────────────

export interface PreviewResponse {
  name: string;
  path: string;
  data: string;
  content: string;
  offset: number;
  totalSize: number;
  hasMore: boolean;
  fileName: string;
  extension: string;
  size: number;
  isBinary: boolean;
  isTruncated: boolean;
}

export interface GrammarResponse {
  language: string;
  scopeName: string;
  grammar: any;
  patterns: any[];
}

// ── Disk Info ─────────────────────────────────────────────────────────────

export interface DiskInfo {
  filePath: string;
  fileName: string;
  fileSize: number;
  virtualSize: number;
  format: string;
  diskType: string;
  creatorApp: string;
  creatorVersion: string;
  creatorHostOS: string;
  uniqueID: string;
  checksumValid: boolean;
  blockSize: number;
  maxBATEntries: number;
  warnings: string[];
}

export interface DiskInfoResponse {
  container: ContainerInfo;
  geometry: GeometryInfo;
  partitions: PartitionDetail[];
  fsInfo?: FSInfo;
}

export interface ContainerInfo {
  fileName: string;
  filePath: string;
  format: string;
  diskType: string;
  virtualSize: number;
  physicalSize: number;
  creatorApp: string;
  creatorVersion: string;
  creatorHostOS: string;
  uniqueID: string;
  checksumValid: boolean;
  readonlyMode: boolean;
  headerOffset?: number;
  batOffset?: number;
  batEntrySize?: number;
  blockSize?: number;
}

export interface GeometryInfo {
  totalSectors: number;
  logicalSectorSize: number;
  physicalSectorSize: number;
  partitionScheme: string;
  diskGUID?: string;
  diskSignature?: string;
  chs: CHSGeometry;
}

export interface CHSGeometry {
  cylinders: number;
  heads: number;
  sectorsPerTrack: number;
}

// ── Partitions ────────────────────────────────────────────────────────────

export interface PartitionInfo {
  index: number;
  start: number;
  end: number;
  size: number;
  type: string;
  filesystem: string;
  bootable: boolean;
  label: string;
  active: boolean;
}

export interface Partition {
  index: number;
  start: number;
  end: number;
  size: number;
  type: string;
  filesystem: string;
  bootable: boolean;
  label: string;
}

export interface PartitionDetail {
  index: number;
  label: string;
  filesystem: string;
  startLBA: number;
  endLBA: number;
  totalSectors: number;
  sizeBytes: number;
  bootable: boolean;
  active: boolean;
  isMounted: boolean;
  isUnallocated: boolean;
  status: string;
}

// ── Filesystem ────────────────────────────────────────────────────────────

export interface FileEntry {
  name: string;
  path: string;
  size: number;
  isDir: boolean;
  modTime: number;
  extension: string;
  targetPath?: string;
}

export interface FSInfo {
  filesystemType: string;
  volumeUUID: string;
  volumeLabel: string;
  state: string;
  mountCount: number;
  maxMounts: number;
  lastMountedPath: string;
  lastWriteTime: string;
  blockSize: number;
  totalBlocks: number;
  freeBlocks: number;
  totalInodes: number;
  freeInodes: number;
  blockGroups: number;
  featureFlags: string[];
  superblockValid: boolean;
}

// ── Smart Open ────────────────────────────────────────────────────────────

export interface SmartOpenResult {
  info: DiskInfo;
  partitions: PartitionInfo[];
  activePartition: PartitionInfo | null;
  rootPath: string;
  warnings: string[];
}

// ── Recent Files ──────────────────────────────────────────────────────────

export interface RecentFile {
  path: string;
  name: string;
  size: number;
  openedAt: number;
}

// ── DIE (Disk Intelligence Engine) ───────────────────────────────────────

export interface CommandContext {
  active_partition: string;
  current_path: string;
  selected_file: string;
  selected_files: string[];
  active_tab: string;
  total_files: number;
  total_partitions: number;
  disk_format: string;
  filesystem_type: string;
}

export interface Suggestion {
  title: string;
  description: string;
  category: string;
  score: number;
}

export interface CommandResult {
  action: string;
  data?: any;
  results?: any[];
  count?: number;
  message?: string;
  path?: string;
  error?: string;
}

export interface HistoryEntry {
  command: string;
  timestamp: number;
  success: boolean;
}

export interface FavoriteEntry {
  command: string;
  label: string;
  icon?: string;
}

export interface Intent {
  action: string;
  query: string;
  filters: Record<string, string>;
  target: string;
  params: Record<string, string>;
  raw_command: string;
  confidence: number;
}

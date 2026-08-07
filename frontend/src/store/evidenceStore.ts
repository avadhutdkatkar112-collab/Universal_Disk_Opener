import { useState, useEffect } from 'react';

export type EvidenceFormat = 'RAW' | 'VHD' | 'VHDX' | 'VDI' | 'VMDK' | 'QCOW2' | 'ISO' | 'UNKNOWN';
export type ViewMode = 'explorer' | 'investigate' | 'examine' | 'timeline' | 'case';

export interface PartitionInfo {
  index: number;
  type: string;
  start: number;
  size: number;
}

export interface Timestamps {
  created: string;
  modified: string;
  accessed: string;
  mftModified: string;
}

export interface FileNode {
  name: string;
  path: string;
  isDir: boolean;
  size: number;
  timestamps?: Timestamps;
  permissions?: string;
  ownerId?: number;
  groupId?: number;
  fileId?: number;
  attributes?: string[];
}

export interface ArtifactInfo {
  name: string;
  type: 'registry' | 'evtx' | 'mft' | 'browser' | 'prefetch' | 'amcache' | 'shellbag' | 'lnk' | 'other';
  path: string;
  found: boolean;
  count: number;
}

export interface Bookmark {
  id: string;
  path: string;
  name: string;
  note: string;
  tag: string;
  timestamp: number;
}

export interface AuditEvent {
  id: string;
  timestamp: number;
  action: string;
  target: string;
  details: string;
  actor: string;
}

export interface EvidenceSession {
  isActive: boolean;
  imagePath: string;
  fileName: string;
  format: EvidenceFormat;
  totalSize: number;
  sha256: string;
  isReadOnly: boolean;
  partitions: PartitionInfo[];
  selectedPartition: number | null;
  filesystemName: string | null;
  currentPath: string;
  currentNodes: FileNode[];
  pathHistory: string[];
  historyIndex: number;
  artifacts: ArtifactInfo[];
  bookmarks: Bookmark[];
  auditTrail: AuditEvent[];
  isAnalyzing: boolean;
  analysisProgress: string;
  examineFilePath: string | null;
}

export interface EvidenceStore extends EvidenceSession {
  openEvidence: (imagePath: string) => Promise<void>;
  selectPartition: (index: number) => Promise<boolean>;
  openFileForExamine: (path: string) => void;
  navigateTo: (path: string) => Promise<void>;
  navigateBack: () => Promise<void>;
  navigateForward: () => Promise<void>;
  navigateUp: () => Promise<void>;
  addBookmark: (path: string, name: string, note: string, tag: string) => void;
  removeBookmark: (id: string) => void;
  logAuditEvent: (action: string, target: string, details: string) => void;
  clearSession: () => void;
  setViewMode: (mode: ViewMode) => void;
  viewMode: ViewMode;
}

const initialState: EvidenceSession = {
  isActive: false,
  imagePath: '',
  fileName: '',
  format: 'UNKNOWN',
  totalSize: 0,
  sha256: '',
  isReadOnly: true,
  partitions: [],
  selectedPartition: null,
  filesystemName: null,
  currentPath: '/',
  currentNodes: [],
  pathHistory: ['/'],
  historyIndex: 0,
  artifacts: [],
  bookmarks: [],
  auditTrail: [],
  isAnalyzing: false,
  analysisProgress: '',
  examineFilePath: null,
};

let _auditCounter = 0;
let _state: EvidenceStore = {
  ...initialState,
  viewMode: 'explorer',
  openEvidence: async () => {},
  selectPartition: async () => false,
  openFileForExamine: () => {},
  navigateTo: async () => {},
  navigateBack: async () => {},
  navigateForward: async () => {},
  navigateUp: async () => {},
  addBookmark: () => {},
  removeBookmark: () => {},
  logAuditEvent: () => {},
  clearSession: () => {},
  setViewMode: () => {},
};
let _listeners: Array<() => void> = [];

function _emit() {
  for (const fn of _listeners) fn();
}

function _patch(partial: Partial<EvidenceStore>) {
  _state = { ..._state, ...partial };
  _emit();
}

function _logAudit(action: string, target: string, details: string) {
  _auditCounter++;
  const event: AuditEvent = {
    id: `audit-${_auditCounter}-${Date.now()}`,
    timestamp: Date.now(),
    action,
    target,
    details,
    actor: 'analyst',
  };
  _patch({ auditTrail: [..._state.auditTrail, event] });
}

async function openEvidence(imagePath: string) {
  const w = window as any;
  const fileName = imagePath.split(/[\\/]/).pop() || imagePath;

  _patch({
    isActive: true,
    imagePath,
    fileName,
    isReadOnly: true,
    isAnalyzing: true,
    analysisProgress: `Opening ${fileName}...`,
    currentPath: '/',
    currentNodes: [],
    pathHistory: ['/'],
    historyIndex: 0,
    selectedPartition: null,
    filesystemName: null,
    partitions: [],
    artifacts: [],
    auditTrail: [],
  });

  _logAudit('evidence.open', imagePath, `Opening evidence file: ${fileName}`);

  try {
    const partitions = await w.go.api.StorageHandler.MountDisk(imagePath);
    const format = detectFormat(fileName);
    const totalSize = partitions?.reduce((sum: number, p: any) => sum + (p.size || 0), 0) || 0;

    _patch({
      partitions: partitions || [],
      format,
      totalSize,
      isAnalyzing: true,
      analysisProgress: `Found ${partitions?.length || 0} partition(s). Auto-selecting best partition...`,
    });

    _logAudit('evidence.detect', format, `Detected ${partitions?.length || 0} partition(s)`);

    if (partitions && partitions.length > 0) {
      const bestIdx = selectBestPartition(partitions);
      const mountOk = await selectPartition(bestIdx);
      if (!mountOk) {
        _patch({ isAnalyzing: false });
        return;
      }
    }

    _patch({ isAnalyzing: false, analysisProgress: '' });
    _logAudit('evidence.ready', 'session', 'Evidence session ready for analysis');
  } catch (err: any) {
    _patch({ isAnalyzing: false, analysisProgress: `Error: ${err?.toString() || err}` });
    _logAudit('evidence.error', 'session', `Failed to open: ${err?.toString() || err}`);
  }
}

async function selectPartition(index: number): Promise<boolean> {
  const w = window as any;
  _patch({
    selectedPartition: index,
    isAnalyzing: true,
    analysisProgress: `Mounting filesystem on partition ${index}...`,
    currentPath: '/',
    currentNodes: [],
    pathHistory: ['/'],
    historyIndex: 0,
  });

  _logAudit('partition.select', `partition/${index}`, `Mounting filesystem on partition ${index}`);

  try {
    await w.go.api.StorageHandler.MountPartition(index);
    const nodes = await w.go.api.StorageHandler.ListDirectory('/');
    const artifacts = discoverArtifacts(nodes || []);

    const partition = _state.partitions.find((p: any) => p.index === index);
    const fsType = partition?.type || 'Unknown';

    _patch({
      currentNodes: nodes || [],
      filesystemName: fsType,
      isAnalyzing: false,
      analysisProgress: '',
      artifacts,
    });

    _logAudit('filesystem.mount', fsType, `Mounted ${fsType} filesystem, ${nodes?.length || 0} root entries`);
    return true;
  } catch (err: any) {
    _patch({ isAnalyzing: false, analysisProgress: `Mount failed: ${err?.toString() || err}` });
    _logAudit('filesystem.error', 'mount', `Mount failed: ${err?.toString() || err}`);
    return false;
  }
}

async function navigateTo(path: string) {
  const w = window as any;
  try {
    const nodes = await w.go.api.StorageHandler.ListDirectory(path);
    const newHistory = _state.pathHistory.slice(0, _state.historyIndex + 1);
    newHistory.push(path);
    _patch({
      currentNodes: nodes || [],
      currentPath: path,
      pathHistory: newHistory,
      historyIndex: newHistory.length - 1,
    });
    _logAudit('navigate', path, `Navigated to ${path}`);
  } catch (err: any) {
    _patch({ analysisProgress: `Navigation failed: ${err?.toString() || err}` });
  }
}

async function navigateBack() {
  if (_state.historyIndex <= 0) return;
  const newIndex = _state.historyIndex - 1;
  const path = _state.pathHistory[newIndex];
  const w = window as any;
  try {
    const nodes = await w.go.api.StorageHandler.ListDirectory(path);
    _patch({ currentNodes: nodes || [], currentPath: path, historyIndex: newIndex });
  } catch {}
}

async function navigateForward() {
  if (_state.historyIndex >= _state.pathHistory.length - 1) return;
  const newIndex = _state.historyIndex + 1;
  const path = _state.pathHistory[newIndex];
  const w = window as any;
  try {
    const nodes = await w.go.api.StorageHandler.ListDirectory(path);
    _patch({ currentNodes: nodes || [], currentPath: path, historyIndex: newIndex });
  } catch {}
}

async function navigateUp() {
  if (_state.currentPath === '/') return;
  const parts = _state.currentPath.replace(/\/+$/, '').split('/').filter(Boolean);
  parts.pop();
  const parentPath = parts.length === 0 ? '/' : '/' + parts.join('/');
  await navigateTo(parentPath);
}

function addBookmark(path: string, name: string, note: string, tag: string) {
  const id = `bm-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  _patch({ bookmarks: [..._state.bookmarks, { id, path, name, note, tag, timestamp: Date.now() }] });
  _logAudit('bookmark.add', path, `Bookmarked: ${name} [${tag}]`);
}

function removeBookmark(id: string) {
  const bm = _state.bookmarks.find(b => b.id === id);
  _patch({ bookmarks: _state.bookmarks.filter(b => b.id !== id) });
  if (bm) _logAudit('bookmark.remove', bm.path, `Removed bookmark: ${bm.name}`);
}

function logAuditEvent(action: string, target: string, details: string) {
  _logAudit(action, target, details);
}

function clearSession() {
  _patch({ ...initialState });
}

function setViewMode(mode: ViewMode) {
  _patch({ viewMode: mode });
}

function openFileForExamine(path: string) {
  _patch({ examineFilePath: path, viewMode: 'examine' });
  _logAudit('file.examine', path, `Opening file in Examine view: ${path}`);
}

function detectFormat(fileName: string): EvidenceFormat {
  const ext = fileName.split('.').pop()?.toLowerCase() || '';
  if (['vhd'].includes(ext)) return 'VHD';
  if (['vhdx'].includes(ext)) return 'VHDX';
  if (['vdi'].includes(ext)) return 'VDI';
  if (['vmdk', 'vmdk-full', 'vmdk-sparse'].includes(ext)) return 'VMDK';
  if (['qcow2', 'qcow'].includes(ext)) return 'QCOW2';
  if (['iso', 'img'].includes(ext)) return 'ISO';
  if (['raw', 'dd', 'bin', 'e01'].includes(ext)) return 'RAW';
  return 'UNKNOWN';
}

function selectBestPartition(partitions: any[]): number {
  const ntfs = partitions.find((p: any) => p.type?.includes('NTFS'));
  if (ntfs) return ntfs.index;
  const linux = partitions.find((p: any) => p.type?.includes('EXT'));
  if (linux) return linux.index;
  return partitions[0]?.index || 1;
}

function discoverArtifacts(nodes: FileNode[]): ArtifactInfo[] {
  const artifacts: ArtifactInfo[] = [];
  const knownPaths: Record<string, ArtifactInfo['type']> = {
    'Windows': 'other',
    'Users': 'other',
    'Program Files': 'other',
  };

  for (const node of nodes) {
    if (knownPaths[node.name]) {
      artifacts.push({
        name: node.name,
        type: knownPaths[node.name],
        path: node.path,
        found: true,
        count: 1,
      });
    }
  }

  return artifacts;
}

_state = {
  ...initialState,
  viewMode: 'explorer',
  openEvidence,
  selectPartition,
  openFileForExamine,
  navigateTo,
  navigateBack,
  navigateForward,
  navigateUp,
  addBookmark,
  removeBookmark,
  logAuditEvent,
  clearSession,
  setViewMode,
};

export function useEvidenceStore(): EvidenceStore {
  const [, rerender] = useState(0);

  useEffect(() => {
    const listener = () => rerender((c) => c + 1);
    _listeners.push(listener);
    return () => {
      _listeners = _listeners.filter((fn) => fn !== listener);
    };
  }, []);

  return _state;
}

useEvidenceStore.getState = () => _state;

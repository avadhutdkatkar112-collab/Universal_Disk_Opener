import type {
  PreviewResponse,
  GrammarResponse,
  Intent,
  HistoryEntry,
  FavoriteEntry,
  Suggestion,
} from '../types';

// ── Disk Operations ─────────────────────────────────────────────────────

export async function OpenFile(path: string): Promise<any> {
  const result = await (window as any).go.ui.App.OpenFile(path);
  return typeof result === 'string' ? JSON.parse(result) : result;
}

export async function OpenFileDialog(): Promise<string> {
  return await (window as any).go.ui.App.OpenFileDialog();
}

export async function OpenFileNative(vfsPath: string, fileName: string): Promise<void> {
  await (window as any).go.ui.App.OpenFileNative(vfsPath, fileName);
}

export async function Close(): Promise<void> {
  await (window as any).go.ui.App.Close();
}

export async function SelectPartition(index: number): Promise<any> {
  await (window as any).go.ui.App.SelectPartition(index);
}

export async function GetRecentFiles(): Promise<any[]> {
  try {
    const result = await (window as any).go.ui.App.GetRecentFiles();
    return typeof result === 'string' ? JSON.parse(result) : result || [];
  } catch {
    return [];
  }
}

export async function GetDiskInfo(): Promise<any> {
  const result = await (window as any).go.ui.App.GetDiskInfo();
  return typeof result === 'string' ? JSON.parse(result) : result;
}

export async function GetDetailedDiskInfo(): Promise<any> {
  const result = await (window as any).go.ui.App.GetDetailedDiskInfo();
  return typeof result === 'string' ? JSON.parse(result) : result;
}

export async function GetDiskHash(): Promise<{ md5: string; sha256: string }> {
  const result = await (window as any).go.ui.App.GetDiskHash();
  return typeof result === 'string' ? JSON.parse(result) : result;
}

export async function GetPartitions(): Promise<any[]> {
  const result = await (window as any).go.ui.App.GetPartitions();
  return typeof result === 'string' ? JSON.parse(result) : result || [];
}

// ── Filesystem ─────────────────────────────────────────────────────────

export async function ListDirectory(path: string): Promise<any> {
  const result = await (window as any).go.ui.App.ListDirectory(path);
  return typeof result === 'string' ? JSON.parse(result) : result;
}

export async function GetEntry(path: string): Promise<any> {
  const result = await (window as any).go.ui.App.GetEntry(path);
  return typeof result === 'string' ? JSON.parse(result) : result;
}

// Wails v2 serializes Go []byte as base64 string over JSON IPC.
// This helper decodes base64 → Uint8Array in all cases.
function decodeWailsBytes(data: any): Uint8Array {
  if (data instanceof Uint8Array) return data;
  if (Array.isArray(data)) return new Uint8Array(data);
  if (typeof data === 'string' && data.length > 0) {
    try {
      const binStr = atob(data);
      const bytes = new Uint8Array(binStr.length);
      for (let i = 0; i < binStr.length; i++) bytes[i] = binStr.charCodeAt(i);
      return bytes;
    } catch {
      return new TextEncoder().encode(data);
    }
  }
  if (data && typeof data === 'object' && data.data) {
    return new Uint8Array(data.data);
  }
  return new Uint8Array(0);
}

export async function ReadFile(path: string): Promise<Uint8Array> {
  const raw: any = await (window as any).go.ui.App.ReadFile(path);
  return decodeWailsBytes(raw);
}

export async function ReadFileChunk(path: string, offset: number, length: number): Promise<Uint8Array> {
  const raw: any = await (window as any).go.ui.StorageHandler.ReadFileChunk(path, offset, length);
  return decodeWailsBytes(raw);
}

export async function ReadFileText(path: string): Promise<string> {
  const raw: any = await (window as any).go.ui.StorageHandler.ReadFile(path);
  const bytes = decodeWailsBytes(raw);
  return new TextDecoder('utf-8', { fatal: false }).decode(bytes);
}

export async function ExtractFile(diskPath: string, localPath: string): Promise<void> {
  await (window as any).go.ui.App.ExtractFile(diskPath, localPath);
}

// ── Preview & File Info ──────────────────────────────────────────────────

export async function GetFilePreview(path: string): Promise<PreviewResponse> {
  const result = await (window as any).go.ui.App.GetFilePreview(path);
  return typeof result === 'string' ? JSON.parse(result) : result;
}

export async function GetGrammarForExtension(ext: string): Promise<GrammarResponse> {
  const result = await (window as any).go.ui.App.GetGrammarForExtension(ext);
  return typeof result === 'string' ? JSON.parse(result) : result;
}

export async function GetFileInfoForPath(path: string): Promise<any> {
  const result = await (window as any).go.ui.App.GetFileInfoForPath(path);
  return typeof result === 'string' ? JSON.parse(result) : result;
}

// ── Local Filesystem ───────────────────────────────────────────────────

export async function BrowseLocalFS(dirPath: string): Promise<any[]> {
  const result = await (window as any).go.ui.App.BrowseLocalFS(dirPath);
  return typeof result === 'string' ? JSON.parse(result) : result || [];
}

export async function GetLocalDrives(): Promise<string[]> {
  const result = await (window as any).go.ui.App.GetLocalDrives();
  return typeof result === 'string' ? JSON.parse(result) : result || [];
}

// ── DIE (Disk Intelligence Engine) ───────────────────────────────────────

export async function ExecuteCommand(
  command: string,
  context: any
): Promise<any> {
  try {
    const ctxStr = typeof context === 'string' ? context : JSON.stringify(context);
    const result = await (window as any).go.ui.App.ExecuteCommand(command, ctxStr);
    return typeof result === 'string' ? JSON.parse(result) : result;
  } catch (err) {
    return {
      action: 'error',
      error: err instanceof Error ? err.message : 'Command execution failed',
    };
  }
}

export async function GetSuggestions(
  query: string,
  context: any
): Promise<Suggestion[]> {
  try {
    const ctxStr = typeof context === 'string' ? context : JSON.stringify(context);
    const result = await (window as any).go.ui.App.GetSuggestions(query, ctxStr);
    return typeof result === 'string' ? JSON.parse(result) : result || [];
  } catch {
    return [];
  }
}

export async function ParseCommand(
  command: string,
  context: any
): Promise<Intent | null> {
  try {
    const ctxStr = typeof context === 'string' ? context : JSON.stringify(context);
    const result = await (window as any).go.ui.App.ParseCommand(command, ctxStr);
    return typeof result === 'string' ? JSON.parse(result) : result;
  } catch {
    return null;
  }
}

export async function GetRegistryArtifacts(
  hivePath: string,
  hiveType: string
): Promise<any> {
  const result = await (window as any).go.ui.App.GetRegistryArtifacts(hivePath, hiveType);
  return typeof result === 'string' ? JSON.parse(result) : result;
}

export async function ParseEVTXFile(evtxPath: string): Promise<any> {
  const result = await (window as any).go.ui.App.ParseEVTXFile(evtxPath);
  return typeof result === 'string' ? JSON.parse(result) : result;
}

export async function ParseMFTFile(mftPath: string): Promise<any> {
  const result = await (window as any).go.ui.App.ParseMFTFile(mftPath);
  return typeof result === 'string' ? JSON.parse(result) : result;
}

export async function BuildUnifiedTimeline(
  registryHivePath: string,
  evtxPath: string,
  mftPath: string,
  startTime: string,
  endTime: string
): Promise<any> {
  const result = await (window as any).go.ui.App.BuildUnifiedTimeline(
    registryHivePath, evtxPath, mftPath, startTime, endTime
  );
  return typeof result === 'string' ? JSON.parse(result) : result;
}

export async function ExportTimelineCSV(entries: any[], outputPath: string): Promise<void> {
  await (window as any).go.ui.App.ExportTimelineCSV(entries, outputPath);
}

export async function ExportTimelineJSON(entries: any[], outputPath: string): Promise<void> {
  await (window as any).go.ui.App.ExportTimelineJSON(entries, outputPath);
}

export async function AnalyzeFindings(entries: any[]): Promise<any[]> {
  const result = await (window as any).go.ui.App.AnalyzeFindings(entries);
  return typeof result === 'string' ? JSON.parse(result) : result || [];
}

export async function LoadTimelineToSQL(entries: any[]): Promise<number> {
  return await (window as any).go.ui.App.LoadTimelineToSQL(entries);
}

export async function ExecuteSQLQuery(query: string): Promise<any> {
  const result = await (window as any).go.ui.App.ExecuteSQLQuery(query);
  return typeof result === 'string' ? JSON.parse(result) : result;
}

export async function LoadSigmaRuleDirectory(dir: string): Promise<number> {
  return await (window as any).go.ui.App.LoadSigmaRuleDirectory(dir);
}

export async function RunSigmaScan(entries: any[]): Promise<any[]> {
  const result = await (window as any).go.ui.App.RunSigmaScan(entries);
  return typeof result === 'string' ? JSON.parse(result) : result || [];
}

export async function GetSigmaRuleCount(): Promise<number> {
  return await (window as any).go.ui.App.GetSigmaRuleCount();
}

export async function GetLiveProcesses(): Promise<any[]> {
  const result = await (window as any).go.ui.App.GetLiveProcesses();
  return typeof result === 'string' ? JSON.parse(result) : result || [];
}

export async function RunMemoryYaraScan(procs: any[]): Promise<any[]> {
  const result = await (window as any).go.ui.App.RunMemoryYaraScan(procs);
  return typeof result === 'string' ? JSON.parse(result) : result || [];
}

export async function GetMemorySnapshot(): Promise<any> {
  const result = await (window as any).go.ui.App.GetMemorySnapshot();
  return typeof result === 'string' ? JSON.parse(result) : result;
}

export async function GenerateReport(caseName: string, investigator: string, entries: any[], alerts: any[], procs: any[], yaraMatches: any[]): Promise<string> {
  return await (window as any).go.ui.App.GenerateReport(caseName, investigator, entries, alerts, procs, yaraMatches);
}

export async function SaveReportToFile(filePath: string, htmlContent: string): Promise<void> {
  await (window as any).go.ui.App.SaveReportToFile(filePath, htmlContent);
}

export async function IngestEvidence(srcPath: string, outputDir: string, passphrase: string, vaultID: string): Promise<any> {
  const result = await (window as any).go.ui.App.IngestEvidence(srcPath, outputDir, passphrase, vaultID);
  return typeof result === 'string' ? JSON.parse(result) : result;
}

export async function IngestEvidenceStruct(req: { source_path: string; case_dir: string; passphrase: string; case_id: string; actor: string }): Promise<any> {
  const result = await (window as any).go.ui.App.IngestEvidenceStruct(req);
  return typeof result === 'string' ? JSON.parse(result) : result;
}

export async function VerifyEvidenceContainer(caseDir: string): Promise<any> {
  const result = await (window as any).go.ui.App.VerifyEvidenceContainer(caseDir);
  return typeof result === 'string' ? JSON.parse(result) : result;
}

export async function VerifyCaseIntegrity(caseDir: string): Promise<any> {
  const result = await (window as any).go.ui.App.VerifyCaseIntegrity(caseDir);
  return typeof result === 'string' ? JSON.parse(result) : result;
}

export async function LogAnalystAction(actor: string, action: string, metadata: Record<string, string>): Promise<void> {
  await (window as any).go.ui.App.LogAnalystAction(actor, action, metadata);
}

export async function ExportChainOfCustodyReport(req: { caseDir: string; examinerName: string; outputPath: string }): Promise<any> {
  const result = await (window as any).go.ui.App.ExportChainOfCustodyReport(req);
  return typeof result === 'string' ? JSON.parse(result) : result;
}

export async function GetCommandHistory(): Promise<HistoryEntry[]> {
  try {
    const result = await (window as any).go.ui.App.GetCommandHistory();
    return typeof result === 'string' ? JSON.parse(result) : result || [];
  } catch {
    return [];
  }
}

export async function GetFavorites(): Promise<FavoriteEntry[]> {
  try {
    const result = await (window as any).go.ui.App.GetFavorites();
    return typeof result === 'string' ? JSON.parse(result) : result || [];
  } catch {
    return [];
  }
}

export async function AddFavorite(command: string, label: string): Promise<void> {
  await (window as any).go.ui.App.AddFavorite(command, label);
}

export async function RemoveFavorite(label: string): Promise<void> {
  await (window as any).go.ui.App.RemoveFavorite(label);
}

// ── Platform / Capability ─────────────────────────────────────────────

export async function ExecuteCapability(capabilityID: string, params: any): Promise<string> {
  const paramsStr = typeof params === 'string' ? params : JSON.stringify(params);
  return await (window as any).go.ui.App.ExecuteCapability(capabilityID, paramsStr);
}

export async function GetJobStatus(jobID: string): Promise<any> {
  const result = await (window as any).go.ui.App.GetJobStatus(jobID);
  return typeof result === 'string' ? JSON.parse(result) : result;
}

export async function ListCapabilities(): Promise<string[]> {
  const result = await (window as any).go.ui.App.ListCapabilities();
  return typeof result === 'string' ? JSON.parse(result) : result || [];
}

export interface HashResult {
  type: 'file' | 'directory';
  path: string;
  size?: number;
  md5?: string;
  sha1?: string;
  sha256?: string;
  elapsed_seconds: number;
  elapsed_ms: number;
  throughput_mbps: number;
  match_status?: string;
  // Directory-specific
  total_files?: number;
  total_size?: number;
  merkle_root?: string;
  files?: Array<{ path: string; size: number; md5: string; sha256: string }>;
}

export async function HashFile(vfsPath: string, verifyHash?: string): Promise<HashResult> {
  const result = await (window as any).go.ui.App.HashFile(vfsPath, verifyHash || '');
  return typeof result === 'string' ? JSON.parse(result) : result;
}

export interface BatchHashResult {
  path: string;
  size: number;
  md5: string;
  sha256: string;
  status: string;
  error?: string;
}

export async function BatchHashFiles(vfsPaths: string[]): Promise<BatchHashResult[]> {
  const result = await (window as any).go.ui.App.BatchHashFiles(vfsPaths);
  return typeof result === 'string' ? JSON.parse(result) : result;
}

export interface PartitionHashResult {
  partition: string;
  size: number;
  bytes_read: number;
  md5: string;
  sha256: string;
  elapsed_seconds: number;
  elapsed_ms: number;
  throughput_mbps: number;
}

export async function PartitionHash(partitionIndex: number): Promise<PartitionHashResult> {
  const result = await (window as any).go.ui.App.PartitionHash(partitionIndex);
  return typeof result === 'string' ? JSON.parse(result) : result;
}

export interface CompareResult {
  path_a: string;
  path_b: string;
  size_a: number;
  size_b: number;
  sha256_a: string;
  sha256_b: string;
  exact_match: boolean;
  similarity_percent: number;
  md5_a: string;
  md5_b: string;
}

export async function CompareHash(pathA: string, pathB: string): Promise<CompareResult> {
  const result = await (window as any).go.ui.App.CompareHash(pathA, pathB);
  return typeof result === 'string' ? JSON.parse(result) : result;
}

// ── Background Hash Jobs ──────────────────────────────────────────────

export interface HashJobProgress {
  job_id: string;
  target_path: string;
  status: 'PENDING' | 'RUNNING' | 'COMPLETED' | 'FAILED';
  bytes_processed: number;
  total_bytes: number;
  percentage: number;
  throughput_mbps: number;
  eta_seconds: number;
  error?: string;
  result?: any;
}

export async function StartPartitionHashJob(partitionIndex: number): Promise<string> {
  return await (window as any).go.ui.App.StartPartitionHashJob(partitionIndex);
}

export async function StartDirectoryHashJob(vfsPath: string): Promise<string> {
  return await (window as any).go.ui.App.StartDirectoryHashJob(vfsPath);
}

export async function GetHashJobStatus(jobID: string): Promise<HashJobProgress> {
  const result = await (window as any).go.ui.App.GetHashJobStatus(jobID);
  return typeof result === 'string' ? JSON.parse(result) : result;
}

export async function CancelHashJob(jobID: string): Promise<boolean> {
  return await (window as any).go.ui.App.CancelHashJob(jobID);
}

export async function StartBatchHashJob(vfsPaths: string[]): Promise<string> {
  return await (window as any).go.ui.App.StartBatchHashJob(vfsPaths);
}

// ── Main object (backward compat) ─────────────────────────────────────

export const Main = {
  OpenFile,
  OpenFileDialog,
  OpenFileNative,
  Close,
  SelectPartition,
  GetRecentFiles,
  GetDiskInfo,
  GetDetailedDiskInfo,
  GetDiskHash,
  GetPartitions,
  ListDirectory,
  GetEntry,
  ReadFile,
  ExtractFile,
  GetFilePreview,
  GetGrammarForExtension,
  GetFileInfoForPath,
  BrowseLocalFS,
  GetLocalDrives,
  ExecuteCommand,
  GetSuggestions,
  ParseCommand,
  GetCommandHistory,
  GetFavorites,
  AddFavorite,
  RemoveFavorite,
  ExecuteCapability,
  GetJobStatus,
  ListCapabilities,
  HashFile,
  BatchHashFiles,
  PartitionHash,
  CompareHash,
  StartPartitionHashJob,
  StartDirectoryHashJob,
  StartBatchHashJob,
  GetHashJobStatus,
  CancelHashJob,
};

// ── Storage Handler (pkg/storage bridge) ──────────────────────────────────

export async function MountDisk(imagePath: string): Promise<any[]> {
  const result = await (window as any).go.ui.StorageHandler.MountDisk(imagePath);
  return Array.isArray(result) ? result : (typeof result === 'string' ? JSON.parse(result) : result || []);
}

export async function MountPartition(index: number): Promise<boolean> {
  const result = await (window as any).go.ui.StorageHandler.MountPartition(index);
  return !!result;
}

export async function StorageListDirectory(path: string): Promise<any[]> {
  const result = await (window as any).go.ui.StorageHandler.ListDirectory(path);
  return Array.isArray(result) ? result : (typeof result === 'string' ? JSON.parse(result) : result || []);
}

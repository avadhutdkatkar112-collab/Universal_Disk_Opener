import { useState, useCallback, useMemo, useEffect } from 'react';
import { Icon } from './Icon';
import {
  HashFile, CompareHash,
  StartPartitionHashJob, StartDirectoryHashJob, StartBatchHashJob, CancelHashJob,
  type HashResult, type BatchHashResult, type PartitionHashResult, type CompareResult,
} from '../lib/wails';
import { fmtSize } from '../lib/utils';
import { useDiskStore } from '../store/diskStore';
import { useJobManager } from '../hooks/useJobManager';

type HashMode = 'single' | 'directory' | 'batch' | 'partition' | 'compare';

interface Props {
  filePath?: string | null;
  filePaths?: string[];
  partitionIndex?: number;
  comparePaths?: [string, string];
  onClose: () => void;
  onMinimize?: () => void;
}

function formatDuration(ms: number): string {
  if (ms < 1) return '< 1 ms';
  if (ms < 1000) return `${Math.round(ms)} ms`;
  return `${(ms / 1000).toFixed(2)} s`;
}

function formatThroughput(mbps: number | undefined, elapsedMs: number): string {
  if (elapsedMs < 10) return 'Instant';
  return `${(mbps ?? 0).toFixed(1)} MB/s`;
}

function exportCSV(data: Record<string, any>[], filename: string) {
  if (data.length === 0) return;
  const headers = Object.keys(data[0]);
  const csv = [
    headers.join(','),
    ...data.map(row => headers.map(h => `"${String(row[h] ?? '').replace(/"/g, '""')}"`).join(','))
  ].join('\n');
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

function exportJSON(data: any, filename: string) {
  const json = JSON.stringify(data, null, 2);
  const blob = new Blob([json], { type: 'application/json;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

function copyToClipboard(text: string): Promise<void> {
  return navigator.clipboard.writeText(text);
}

function generateHashCertificate(result: any, type: string): any {
  return {
    certificate_type: 'Cryptographic Hash Certificate',
    generated_at: new Date().toISOString(),
    tool: 'VHD Explorer - Universal Container Explorer',
    version: '0.6.0',
    target: {
      type,
      path: result.path || result.target_path,
      size_bytes: result.size || result.total_size || 0,
    },
    hashes: {
      md5: result.md5 || null,
      sha1: result.sha1 || null,
      sha256: result.sha256 || result.merkle_root || null,
    },
    metadata: {
      elapsed_ms: result.elapsed_ms || 0,
      throughput_mbps: result.throughput_mbps || 0,
      total_files: result.total_files || (type === 'file' ? 1 : 0),
      file_list: result.files?.map((f: any) => ({
        path: f.path,
        size: f.size,
        sha256: f.sha256,
        md5: f.md5,
      })) || [],
    },
    verification: {
      status: result.match_status || 'UNVERIFIED',
      reference_hash: null,
    },
  };
}

export function HashModal({ filePath, filePaths, partitionIndex, comparePaths, onClose, onMinimize }: Props) {
  const { disk } = useDiskStore();
  const { getJob, getCompletedJob } = useJobManager();
  const [verifyInput, setVerifyInput] = useState('');
  const [computing, setComputing] = useState(false);
  const [hashResult, setHashResult] = useState<HashResult | null>(null);
  const [batchResult, setBatchResult] = useState<BatchHashResult[] | null>(null);
  const [partitionResult, setPartitionResult] = useState<PartitionHashResult | null>(null);
  const [compareResult, setCompareResult] = useState<CompareResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [activeJobId, setActiveJobId] = useState<string | null>(null);

  const mode: HashMode = useMemo(() => {
    if (comparePaths) return 'compare';
    if (partitionIndex != null) return 'partition';
    if (filePaths && filePaths.length > 0) return 'batch';
    return 'single';
  }, [comparePaths, partitionIndex, filePaths]);

  const title = useMemo(() => {
    switch (mode) {
      case 'compare': return 'Structural Similarity Analysis';
      case 'partition': {
        const part = disk?.partitions[partitionIndex!];
        return `Sector-Level Partition Hash: ${part ? `P${part.index} — ${part.fsType}` : 'Partition'}`;
      }
      case 'batch': return `Batch Directory Hashing (${filePaths?.length || 0} items)`;
      default: return 'Cryptographic Integrity Hash';
    }
  }, [mode, partitionIndex, filePaths, disk]);

  const effectiveMode: HashMode = useMemo(() => {
    if (mode === 'single' && hashResult?.type === 'directory') return 'directory';
    return mode;
  }, [mode, hashResult]);

  const effectiveTitle = useMemo(() => {
    if (effectiveMode === 'directory' && hashResult) {
      return `Batch Directory Hash: ${hashResult.path}`;
    }
    return title;
  }, [effectiveMode, hashResult, title]);

  // Listen for background job completion
  useEffect(() => {
    if (!activeJobId) return;
    const completedJob = getCompletedJob(activeJobId);
    if (completedJob?.result) {
      if (mode === 'partition') {
        setPartitionResult(completedJob.result as PartitionHashResult);
      } else if (mode === 'single' && completedJob.result.type === 'directory') {
        setHashResult(completedJob.result as HashResult);
      } else if (mode === 'batch' && completedJob.result.type === 'batch') {
        setBatchResult(completedJob.result.files as BatchHashResult[]);
      }
      setComputing(false);
      setActiveJobId(null);
    } else if (completedJob?.status === 'FAILED') {
      setError(completedJob.error || 'Job failed');
      setComputing(false);
      setActiveJobId(null);
    }
  }, [activeJobId, getCompletedJob, mode]);

  const handleCancel = useCallback(async () => {
    if (activeJobId) {
      await CancelHashJob(activeJobId);
      setComputing(false);
      setActiveJobId(null);
    }
  }, [activeJobId]);

  const handleCompute = useCallback(async () => {
    setComputing(true);
    setError(null);
    try {
      switch (mode) {
        case 'single': {
          const res = await HashFile(filePath!, verifyInput.trim());
          if (res.type === 'directory') {
            // Directory detected - start background job
            const jobId = await StartDirectoryHashJob(filePath!);
            setActiveJobId(jobId);
            return; // Don't set computing=false yet
          }
          setHashResult(res);
          break;
        }
        case 'batch': {
          // Start background job for batch hashing
          const jobId = await StartBatchHashJob(filePaths!);
          setActiveJobId(jobId);
          return; // Don't set computing=false yet
        }
        case 'partition': {
          // Start background job for partition hashing
          const jobId = await StartPartitionHashJob(partitionIndex!);
          setActiveJobId(jobId);
          return; // Don't set computing=false yet
        }
        case 'compare':
          setCompareResult(await CompareHash(comparePaths![0], comparePaths![1]));
          break;
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Hash computation failed');
    } finally {
      setComputing(false);
    }
  }, [mode, filePath, filePaths, partitionIndex, comparePaths, verifyInput]);

  const hasResult = hashResult || batchResult || partitionResult || compareResult;

  return (
    <div role="dialog" aria-label="Hash Calculator" aria-modal="true" onClick={onClose}
      style={{
        position: 'fixed', inset: 0, zIndex: 100,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        background: 'rgba(0,0,0,0.4)',
      }}>
      <div onClick={e => e.stopPropagation()} style={{
        width: effectiveMode === 'batch' || effectiveMode === 'directory' ? 720 : effectiveMode === 'partition' ? 600 : 560,
        maxHeight: '85vh', overflow: 'auto',
        background: 'var(--win-surface)', border: '1px solid var(--win-stroke)',
        borderRadius: 'var(--win-radius-lg)', boxShadow: '0 8px 32px rgba(0,0,0,0.18)',
      }}>
        {/* Header */}
        <div style={{
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          padding: '14px 18px', borderBottom: '1px solid var(--win-stroke)',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Icon name={effectiveMode === 'partition' ? 'disk' : effectiveMode === 'compare' ? 'analyze' : 'hash'} size={16}
              style={{ color: 'var(--win-accent)' }} />
            <span style={{ fontSize: 14, fontWeight: 600, color: 'var(--win-text)' }}>{effectiveTitle}</span>
          </div>
          <button aria-label="Close" onClick={onClose}
            style={{
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              width: 24, height: 24, borderRadius: 'var(--win-radius-sm)',
              color: 'var(--win-text-tertiary)',
            }}
            onMouseEnter={e => (e.currentTarget.style.background = 'var(--win-subtle-hover)')}
            onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}>
            <Icon name="close" size={12} />
          </button>
        </div>

        <div style={{ padding: '16px 18px' }}>
          {/* Single file path */}
          {mode === 'single' && filePath && !hashResult && (
            <div style={{
              fontSize: 11, fontFamily: 'var(--win-font-mono)', color: 'var(--win-text-tertiary)',
              padding: '6px 8px', background: 'var(--win-bg)', borderRadius: 'var(--win-radius-sm)',
              marginBottom: 14, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
            }}>{filePath}</div>
          )}

          {/* Verify input (single/compare only) */}
          {(mode === 'single' || mode === 'compare') && (
            <div style={{ marginBottom: 14 }}>
              <label style={{ fontSize: 11, color: 'var(--win-text-secondary)', display: 'block', marginBottom: 4 }}>
                Verify Against Reference Hash (Optional)
              </label>
              <input type="text" value={verifyInput} onChange={e => setVerifyInput(e.target.value)}
                placeholder="Paste SHA-256 / MD5 / SHA-1 hash..."
                style={{
                  width: '100%', padding: '6px 8px', fontSize: 12, fontFamily: 'var(--win-font-mono)',
                  background: 'var(--win-bg)', border: '1px solid var(--win-stroke)',
                  borderRadius: 'var(--win-radius-sm)', color: 'var(--win-text)', outline: 'none',
                }}
                onFocus={e => (e.currentTarget.style.borderColor = 'var(--win-accent-default)')}
                onBlur={e => (e.currentTarget.style.borderColor = 'var(--win-stroke)')} />
            </div>
          )}

          {/* Computing indicator */}
          {computing && (
            <div style={{
              padding: '12px 0',
              fontSize: 12, color: 'var(--win-text-secondary)',
            }}>
              {activeJobId ? (
                // Show background job progress
                <BackgroundJobProgress job={getJob(activeJobId)} onCancel={handleCancel} />
              ) : (
                // Show simple spinner for synchronous operations
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <div style={{
                    width: 14, height: 14, border: '2px solid var(--win-stroke-strong)',
                    borderTopColor: 'var(--win-accent)', borderRadius: '50%',
                    animation: 'spin 0.6s linear infinite',
                  }} />
                  {effectiveMode === 'compare' ? 'Computing similarity...'
                    : 'Computing digests...'}
                </div>
              )}
            </div>
          )}

          {/* Error */}
          {error && (
            <div style={{
              display: 'flex', alignItems: 'center', gap: 6, padding: '8px 10px',
              background: 'rgba(232,17,35,0.08)', borderRadius: 'var(--win-radius-sm)',
              fontSize: 12, color: 'var(--win-danger)', marginBottom: 12,
            }}>
              <Icon name="alert" size={13} />{error}
            </div>
          )}

          {/* Single file result */}
          {hashResult?.type === 'file' && <FileResult result={hashResult} verifyInput={verifyInput} />}

          {/* Directory result (from single HashFile auto-detect) */}
          {hashResult?.type === 'directory' && <DirectoryResult result={hashResult} />}

          {/* Batch result */}
          {batchResult && <BatchResultView results={batchResult} />}

          {/* Partition result */}
          {partitionResult && <PartitionResultView result={partitionResult} />}

          {/* Compare result */}
          {compareResult && <CompareResultView result={compareResult} />}

          {/* Actions */}
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, marginTop: 14, flexWrap: 'wrap' }}>
            <button onClick={onClose} style={{
              padding: '6px 14px', fontSize: 12, borderRadius: 'var(--win-radius-sm)',
              background: 'var(--win-bg)', border: '1px solid var(--win-stroke)',
              color: 'var(--win-text-secondary)',
            }}
              onMouseEnter={e => (e.currentTarget.style.background = 'var(--win-subtle-hover)')}
              onMouseLeave={e => (e.currentTarget.style.background = 'var(--win-bg)')}>Close</button>
            
            {/* Minimize to Dock for running background jobs */}
            {computing && activeJobId && onMinimize && (
              <button onClick={onMinimize} style={{
                padding: '6px 14px', fontSize: 12, borderRadius: 'var(--win-radius-sm)',
                background: 'var(--win-bg)', border: '1px solid var(--win-stroke)',
                color: 'var(--win-text-secondary)',
              }}
                onMouseEnter={e => (e.currentTarget.style.background = 'var(--win-subtle-hover)')}
                onMouseLeave={e => (e.currentTarget.style.background = 'var(--win-bg)')}>
                Minimize to Dock
              </button>
            )}
            
            {/* Copy All Hashes for any result with hashes */}
            {hashResult?.type === 'file' && hashResult.sha256 && (
              <ActionButton label="Copy Hashes" onClick={() => copyToClipboard(
                `MD5: ${hashResult.md5}\nSHA-1: ${hashResult.sha1}\nSHA-256: ${hashResult.sha256}`
              )} />
            )}
            {hashResult?.type === 'directory' && hashResult.merkle_root && (
              <ActionButton label="Copy All Hashes" onClick={() => copyToClipboard(
                [`Directory: ${hashResult.path}`, `Merkle Root: ${hashResult.merkle_root}`, '',
                 ...(hashResult.files || []).map((f: any) => `${f.sha256}  ${f.path}`)].join('\n')
              )} />
            )}
            {batchResult && batchResult.length > 0 && (
              <ActionButton label="Copy All Hashes" onClick={() => copyToClipboard(
                batchResult.map(r => `${r.sha256}  ${r.path}`).join('\n')
              )} />
            )}
            
            {/* Export CSV */}
            {batchResult && batchResult.length > 0 && (
              <ActionButton label="Export CSV" onClick={() => exportCSV(
                batchResult.map(r => ({ path: r.path, size: r.size, sha256: r.sha256, status: r.status })),
                'hash-manifest.csv'
              )} />
            )}
            {hashResult?.type === 'directory' && hashResult.files && hashResult.files.length > 0 && (
              <ActionButton label="Export CSV" onClick={() => exportCSV(
                hashResult.files!.map(f => ({ path: f.path, size: f.size, sha256: f.sha256, md5: f.md5 })),
                'directory-manifest.csv'
              )} />
            )}
            
            {/* Export Hash Certificate */}
            {hasResult && (hashResult || partitionResult) && (
              <ActionButton label="Export Certificate" onClick={() => exportJSON(
                generateHashCertificate(hashResult || partitionResult, hashResult?.type || 'partition'),
                'hash-certificate.json'
              )} />
            )}
            
            {!hasResult && !computing && (
              <button onClick={handleCompute} style={{
                padding: '6px 14px', fontSize: 12, fontWeight: 600,
                borderRadius: 'var(--win-radius-sm)',
                background: 'var(--win-accent-default)', color: '#fff', border: 'none',
              }}
                onMouseEnter={e => (e.currentTarget.style.background = 'var(--win-accent-hover)')}
                onMouseLeave={e => (e.currentTarget.style.background = 'var(--win-accent-default)')}>
                Compute Hash
              </button>
            )}
            {hasResult && (
              <button onClick={handleCompute} style={{
                padding: '6px 14px', fontSize: 12, fontWeight: 600,
                borderRadius: 'var(--win-radius-sm)',
                background: 'var(--win-bg)', border: '1px solid var(--win-stroke-strong)',
                color: 'var(--win-text)',
              }}
                onMouseEnter={e => (e.currentTarget.style.background = 'var(--win-subtle-hover)')}
                onMouseLeave={e => (e.currentTarget.style.background = 'var(--win-bg)')}>Recompute</button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

// ── File Result ─────────────────────────────────────────────────────────

function FileResult({ result, verifyInput }: { result: HashResult; verifyInput: string }) {
  const status = useMemo(() => {
    if (!verifyInput.trim()) return { label: 'UNVERIFIED', color: 'var(--win-text-secondary)' };
    if (result.match_status === 'MATCH_VERIFIED') return { label: 'MATCH VERIFIED', color: 'var(--win-success)' };
    return { label: 'MISMATCH', color: 'var(--win-danger)' };
  }, [result.match_status, verifyInput]);

  return (
    <div style={{
      background: 'var(--win-bg)', border: '1px solid var(--win-stroke)',
      borderRadius: 'var(--win-radius-sm)', padding: '12px 14px', marginBottom: 12,
    }}>
      <div style={{ fontSize: 11, color: 'var(--win-text-tertiary)', marginBottom: 8 }}>
        Size: {fmtSize(result.size || 0)}
      </div>

      <div style={{
        display: 'flex', alignItems: 'center', gap: 6, padding: '6px 8px', marginBottom: 10,
        background: status.color === 'var(--win-success)' ? 'rgba(16,124,16,0.08)'
          : status.color === 'var(--win-danger)' ? 'rgba(232,17,35,0.08)' : 'var(--win-surface)',
        borderRadius: 'var(--win-radius-sm)', border: `1px solid ${status.color}20`,
      }}>
        <span style={{ fontSize: 11, color: 'var(--win-text-secondary)' }}>Status:</span>
        <span style={{ fontWeight: 600, fontSize: 12, color: status.color }}>{status.label}</span>
      </div>

      <HashRow algo="MD5" value={result.md5 || ''} badge="Legacy / Forensic" />
      <HashRow algo="SHA-1" value={result.sha1 || ''} badge="Legacy / Forensic" />
      <HashRow algo="SHA-256" value={result.sha256 || ''} badge="Primary" />

      <div style={{
        display: 'flex', justifyContent: 'space-between', marginTop: 10,
        paddingTop: 8, borderTop: '1px solid var(--win-stroke)',
        fontSize: 11, color: 'var(--win-text-tertiary)',
      }}>
        <span>{fmtSize(result.size || 0)} processed</span>
        <span>{formatThroughput(result.throughput_mbps, result.elapsed_ms || 0)} — {formatDuration(result.elapsed_ms || 0)}</span>
      </div>
    </div>
  );
}

// ── Directory Result (Merkle Tree) ─────────────────────────────────────

function DirectoryResult({ result }: { result: HashResult }) {
  const files = result.files || [];
  const [expanded, setExpanded] = useState(false);

  return (
    <div style={{
      background: 'var(--win-bg)', border: '1px solid var(--win-stroke)',
      borderRadius: 'var(--win-radius-sm)', padding: '14px', marginBottom: 12,
    }}>
      {/* Summary */}
      <div style={{ display: 'flex', gap: 16, fontSize: 11, color: 'var(--win-text-secondary)', marginBottom: 10 }}>
        <span>{result.total_files || 0} files</span>
        <span>{fmtSize(result.total_size || 0)}</span>
      </div>

      {/* Merkle root */}
      <div style={{ marginBottom: 10 }}>
        <div style={{ fontSize: 11, color: 'var(--win-text-tertiary)', marginBottom: 4 }}>
          Merkle Tree Root:
        </div>
        <HashRow algo="SHA-256" value={result.merkle_root || ''} />
      </div>

      {/* File table */}
      <div style={{ fontSize: 11, color: 'var(--win-text-secondary)', marginBottom: 6 }}>
        <button onClick={() => setExpanded(!expanded)}
          style={{
            display: 'flex', alignItems: 'center', gap: 4, padding: '2px 0',
            fontSize: 11, color: 'var(--win-text-secondary)', background: 'none', border: 'none',
            cursor: 'pointer',
          }}>
          <Icon name={expanded ? 'chevron-down' : 'chevron-right'} size={10} />
          File Manifest ({files.length} items)
        </button>
      </div>

      {expanded && (
        <div style={{ maxHeight: 200, overflow: 'auto' }}>
          <table style={{ width: '100%', fontSize: 11, borderCollapse: 'collapse' }}>
            <thead>
              <tr style={{ color: 'var(--win-text-tertiary)', borderBottom: '1px solid var(--win-stroke)' }}>
                <th style={{ textAlign: 'left', padding: '3px 6px', fontWeight: 500 }}>Path</th>
                <th style={{ textAlign: 'right', padding: '3px 6px', fontWeight: 500 }}>Size</th>
                <th style={{ textAlign: 'left', padding: '3px 6px', fontWeight: 500 }}>SHA-256</th>
              </tr>
            </thead>
            <tbody>
              {files.map((f, i) => (
                <tr key={i} style={{ borderBottom: '1px solid var(--win-stroke)' }}>
                  <td style={{ padding: '3px 6px', maxWidth: 300, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontFamily: 'var(--win-font-mono)' }}>
                    {f.path}
                  </td>
                  <td style={{ padding: '3px 6px', textAlign: 'right', fontFamily: 'var(--win-font-mono)' }}>
                    {fmtSize(f.size)}
                  </td>
                  <td style={{ padding: '3px 6px', fontFamily: 'var(--win-font-mono)', maxWidth: 160, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {f.sha256.slice(0, 16)}...
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div style={{
        display: 'flex', justifyContent: 'space-between', marginTop: 10,
        paddingTop: 8, borderTop: '1px solid var(--win-stroke)',
        fontSize: 11, color: 'var(--win-text-tertiary)',
      }}>
        <span>{formatThroughput(result.throughput_mbps, result.elapsed_ms || 0)}</span>
        <span>{formatDuration(result.elapsed_ms || 0)}</span>
      </div>
    </div>
  );
}

// ── Batch Result ────────────────────────────────────────────────────────

function BatchResultView({ results }: { results: BatchHashResult[] }) {
  const duplicates = results.filter(r => r.status?.startsWith('DUPLICATE'));

  return (
    <div style={{ marginBottom: 12 }}>
      <div style={{
        display: 'flex', gap: 12, padding: '6px 0', fontSize: 11, color: 'var(--win-text-secondary)',
        borderBottom: '1px solid var(--win-stroke)', marginBottom: 8,
      }}>
        <span>{results.length} files hashed</span>
        {duplicates.length > 0 && (
          <span style={{ color: 'var(--win-warning)', fontWeight: 500 }}>
            {duplicates.length} duplicate{duplicates.length > 1 ? 's' : ''} found
          </span>
        )}
      </div>

      <div style={{ maxHeight: 240, overflow: 'auto' }}>
        <table style={{ width: '100%', fontSize: 11, borderCollapse: 'collapse' }}>
          <thead>
            <tr style={{ color: 'var(--win-text-tertiary)', borderBottom: '1px solid var(--win-stroke)' }}>
              <th style={{ textAlign: 'left', padding: '4px 6px', fontWeight: 500 }}>File</th>
              <th style={{ textAlign: 'right', padding: '4px 6px', fontWeight: 500 }}>Size</th>
              <th style={{ textAlign: 'left', padding: '4px 6px', fontWeight: 500 }}>SHA-256</th>
              <th style={{ textAlign: 'left', padding: '4px 6px', fontWeight: 500 }}>Status</th>
            </tr>
          </thead>
          <tbody>
            {results.map((r, i) => (
              <tr key={i} style={{ borderBottom: '1px solid var(--win-stroke)' }}>
                <td style={{ padding: '4px 6px', maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontFamily: 'var(--win-font-mono)' }}>
                  {r.path?.split('/').pop() || r.path}
                </td>
                <td style={{ padding: '4px 6px', textAlign: 'right', fontFamily: 'var(--win-font-mono)' }}>
                  {r.error ? '—' : fmtSize(r.size)}
                </td>
                <td style={{ padding: '4px 6px', fontFamily: 'var(--win-font-mono)', maxWidth: 160, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {r.error ? r.error : r.sha256?.slice(0, 16) + '...'}
                </td>
                <td style={{ padding: '4px 6px' }}>
                  <span style={{
                    padding: '1px 6px', borderRadius: 8, fontSize: 10, fontWeight: 500,
                    background: r.status?.startsWith('DUPLICATE') ? 'rgba(255,185,0,0.12)' : 'rgba(16,124,16,0.1)',
                    color: r.status?.startsWith('DUPLICATE') ? '#B8860B' : 'var(--win-success)',
                  }}>
                    {r.status?.startsWith('DUPLICATE') ? 'DUP' : 'OK'}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// ── Partition Result ────────────────────────────────────────────────────

function PartitionResultView({ result }: { result: PartitionHashResult }) {
  const progress = result.size > 0 ? (result.bytes_read / result.size) * 100 : 0;

  return (
    <div style={{
      background: 'var(--win-bg)', border: '1px solid var(--win-stroke)',
      borderRadius: 'var(--win-radius-sm)', padding: '14px', marginBottom: 12,
    }}>
      <div style={{ marginBottom: 12 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 11, marginBottom: 4 }}>
          <span style={{ color: 'var(--win-text-secondary)' }}>
            {fmtSize(result.bytes_read)} / {fmtSize(result.size)}
          </span>
          <span style={{ color: 'var(--win-text-secondary)' }}>{progress.toFixed(1)}%</span>
        </div>
        <div style={{ height: 6, borderRadius: 3, background: 'var(--win-stroke-strong)', overflow: 'hidden' }}>
          <div style={{
            height: '100%', width: `${progress}%`, borderRadius: 3,
            background: 'var(--win-accent-default)', transition: 'width 0.3s',
          }} />
        </div>
      </div>

      <HashRow algo="MD5" value={result.md5} />
      <HashRow algo="SHA-256" value={result.sha256} />

      <div style={{
        display: 'flex', justifyContent: 'space-between', marginTop: 10,
        paddingTop: 8, borderTop: '1px solid var(--win-stroke)',
        fontSize: 11, color: 'var(--win-text-tertiary)',
      }}>
        <span>{result.partition}</span>
        <span>{formatThroughput(result.throughput_mbps, result.elapsed_ms)} — {formatDuration(result.elapsed_ms)}</span>
      </div>
    </div>
  );
}

// ── Compare Result ──────────────────────────────────────────────────────

function CompareResultView({ result }: { result: CompareResult }) {
  const simColor = result.similarity_percent > 90 ? 'var(--win-success)'
    : result.similarity_percent > 70 ? 'var(--win-accent-default)'
    : result.similarity_percent > 40 ? 'var(--win-warning)' : 'var(--win-text-tertiary)';

  return (
    <div style={{
      background: 'var(--win-bg)', border: '1px solid var(--win-stroke)',
      borderRadius: 'var(--win-radius-sm)', padding: '14px', marginBottom: 12,
    }}>
      <div style={{ fontSize: 11, marginBottom: 6 }}>
        <span style={{ color: 'var(--win-text-tertiary)' }}>File A: </span>
        <span style={{ fontFamily: 'var(--win-font-mono)', color: 'var(--win-text)' }}>{result.path_a?.split('/').pop()}</span>
        <span style={{ color: 'var(--win-text-tertiary)', marginLeft: 6 }}>({fmtSize(result.size_a)})</span>
      </div>
      <div style={{ fontSize: 11, marginBottom: 12 }}>
        <span style={{ color: 'var(--win-text-tertiary)' }}>File B: </span>
        <span style={{ fontFamily: 'var(--win-font-mono)', color: 'var(--win-text)' }}>{result.path_b?.split('/').pop()}</span>
        <span style={{ color: 'var(--win-text-tertiary)', marginLeft: 6 }}>({fmtSize(result.size_b)})</span>
      </div>

      <div style={{
        display: 'flex', alignItems: 'center', gap: 6, padding: '6px 8px', marginBottom: 8,
        background: result.exact_match ? 'rgba(16,124,16,0.08)' : 'var(--win-surface)',
        borderRadius: 'var(--win-radius-sm)', border: `1px solid ${result.exact_match ? 'var(--win-success)' : 'var(--win-stroke)'}20`,
      }}>
        <span style={{ fontSize: 11, color: 'var(--win-text-secondary)' }}>SHA-256 Exact Match:</span>
        <span style={{ fontWeight: 600, fontSize: 12, color: result.exact_match ? 'var(--win-success)' : 'var(--win-danger)' }}>
          {result.exact_match ? 'IDENTICAL' : 'NO MATCH'}
        </span>
      </div>

      <div style={{ marginBottom: 10 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 11, marginBottom: 4 }}>
          <span style={{ color: 'var(--win-text-secondary)' }}>Block Similarity</span>
          <span style={{ color: simColor, fontWeight: 600 }}>{(result.similarity_percent ?? 0).toFixed(1)}%</span>
        </div>
        <div style={{ height: 6, borderRadius: 3, background: 'var(--win-stroke-strong)', overflow: 'hidden' }}>
          <div style={{
            height: '100%', width: `${result.similarity_percent}%`, borderRadius: 3,
            background: simColor, transition: 'width 0.3s',
          }} />
        </div>
      </div>

      <div style={{
        padding: '8px 10px', fontSize: 11, color: 'var(--win-text-secondary)',
        background: 'var(--win-surface)', borderRadius: 'var(--win-radius-sm)',
        border: '1px solid var(--win-stroke)',
      }}>
        {result.exact_match
          ? 'These files are byte-identical. They share the same cryptographic fingerprint.'
          : result.similarity_percent > 80
          ? 'These files share substantial structure. File B may be a modified backup or truncated draft of File A.'
          : result.similarity_percent > 40
          ? 'These files show moderate structural similarity with significant differences.'
          : 'These files have very different content with minimal structural overlap.'}
      </div>

      <div style={{ marginTop: 10, paddingTop: 8, borderTop: '1px solid var(--win-stroke)' }}>
        <HashRow algo="SHA-256 A" value={result.sha256_a} />
        <HashRow algo="SHA-256 B" value={result.sha256_b} />
      </div>
    </div>
  );
}

// ── Hash Row (full width, word-break) ──────────────────────────────────

function HashRow({ algo, value, badge }: { algo: string; value: string; badge?: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(() => {
    navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }, [value]);

  return (
    <div style={{
      display: 'flex', alignItems: 'flex-start', gap: 6, padding: '3px 0',
      fontFamily: 'var(--win-font-mono)',
    }}>
      <span style={{ width: 68, color: 'var(--win-text-tertiary)', fontSize: 11, flexShrink: 0, paddingTop: 1 }}>
        {algo}:
      </span>
      <span style={{
        flex: 1, wordBreak: 'break-all',
        color: 'var(--win-text)', fontSize: 11, userSelect: 'all', cursor: 'text',
        lineHeight: '16px',
      }}>{value}</span>
      {badge && (
        <span style={{
          fontSize: 9, color: 'var(--win-text-tertiary)',
          background: 'var(--win-bg)', border: '1px solid var(--win-stroke)',
          borderRadius: 3, padding: '1px 4px', flexShrink: 0,
        }}>{badge}</span>
      )}
      <button aria-label={`Copy ${algo}`} onClick={handleCopy}
        style={{
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          width: 20, height: 20, borderRadius: 'var(--win-radius-sm)', flexShrink: 0,
          color: copied ? 'var(--win-success)' : 'var(--win-text-tertiary)',
        }}
        onMouseEnter={e => (e.currentTarget.style.background = 'var(--win-subtle-hover)')}
        onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}>
        <Icon name={copied ? 'check' : 'copy'} size={11} />
      </button>
    </div>
  );
}

// ── Background Job Progress ──────────────────────────────────────────

function BackgroundJobProgress({ job, onCancel }: { job: any; onCancel: () => void }) {
  if (!job) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <div style={{
          width: 14, height: 14, border: '2px solid var(--win-stroke-strong)',
          borderTopColor: 'var(--win-accent)', borderRadius: '50%',
          animation: 'spin 0.6s linear infinite',
        }} />
        <span>Starting background job...</span>
      </div>
    );
  }

  const isRunning = job.status === 'RUNNING';

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        {isRunning && (
          <div style={{
            width: 14, height: 14, border: '2px solid var(--win-stroke-strong)',
            borderTopColor: 'var(--win-accent)', borderRadius: '50%',
            animation: 'spin 0.6s linear infinite',
          }} />
        )}
        <span style={{ flex: 1, fontFamily: 'var(--win-font-mono)', fontSize: 11, color: 'var(--win-text-secondary)' }}>
          {job.target_path}
        </span>
        <span style={{ fontSize: 11, fontWeight: 500, color: 'var(--win-accent)' }}>
          {isRunning ? `${(job.percentage ?? 0).toFixed(1)}%` : job.status}
        </span>
      </div>

      {isRunning && (
        <>
          <div style={{
            height: 4, borderRadius: 2, background: 'var(--win-stroke-strong)',
            overflow: 'hidden',
          }}>
            <div style={{
              height: '100%', width: `${job.percentage}%`,
              background: 'var(--win-accent-default)', borderRadius: 2,
              transition: 'width 0.15s ease-out',
            }} />
          </div>
          <div style={{
            display: 'flex', justifyContent: 'space-between',
            fontSize: 10, color: 'var(--win-text-tertiary)',
          }}>
            <span>{(job.throughput_mbps ?? 0) > 0 ? `${(job.throughput_mbps ?? 0).toFixed(1)} MB/s` : 'Instant'}</span>
            <span>ETA: {job.eta_seconds > 0 ? `${Math.ceil(job.eta_seconds)}s` : '—'}</span>
            <span>{fmtSize(job.bytes_processed)} / {fmtSize(job.total_bytes)}</span>
          </div>
        </>
      )}

      <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 6 }}>
        <button onClick={onCancel} style={{
          padding: '4px 10px', fontSize: 11, borderRadius: 'var(--win-radius-sm)',
          background: 'var(--win-bg)', border: '1px solid var(--win-stroke)',
          color: 'var(--win-text-secondary)',
        }}
          onMouseEnter={e => (e.currentTarget.style.background = 'var(--win-subtle-hover)')}
          onMouseLeave={e => (e.currentTarget.style.background = 'var(--win-bg)')}>
          Cancel
        </button>
      </div>
    </div>
  );
}

// ── Action Button (reusable) ────────────────────────────────────────

function ActionButton({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <button onClick={onClick} style={{
      padding: '6px 14px', fontSize: 12, borderRadius: 'var(--win-radius-sm)',
      background: 'var(--win-bg)', border: '1px solid var(--win-stroke)',
      color: 'var(--win-text-secondary)',
    }}
      onMouseEnter={e => (e.currentTarget.style.background = 'var(--win-subtle-hover)')}
      onMouseLeave={e => (e.currentTarget.style.background = 'var(--win-bg)')}>
      {label}
    </button>
  );
}

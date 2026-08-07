import React, { useState } from 'react';
import { IngestEvidence, VerifyEvidenceContainer } from '../lib/wails';

interface VaultResult {
  manifest: {
    magic: string;
    format_version: number;
    vault_id: string;
    created_at: string;
    cipher: string;
    chunk_size: number;
    total_size_bytes: number;
    total_chunks: number;
    source_sha256: string;
  };
  audit_valid: boolean;
  audit_count: number;
  chunks_ok: boolean;
}

export const EvidenceVaultViewer: React.FC = () => {
  const [srcPath, setSrcPath] = useState('');
  const [outputDir, setOutputDir] = useState('C:\\evidence_cases');
  const [passphrase, setPassphrase] = useState('');
  const [vaultId, setVaultId] = useState('EV-2026-001');
  const [verifyPath, setVerifyPath] = useState('');
  const [ingesting, setIngesting] = useState(false);
  const [verifying, setVerifying] = useState(false);
  const [result, setResult] = useState<VaultResult | null>(null);
  const [ingestResult, setIngestResult] = useState<any>(null);

  const handleIngest = async () => {
    if (!srcPath || !passphrase) return;
    setIngesting(true);
    try {
      const res = await IngestEvidence(srcPath, outputDir, passphrase, vaultId);
      setIngestResult(res);
    } catch (err) {
      alert('Ingest failed: ' + err);
    } finally {
      setIngesting(false);
    }
  };

  const handleVerify = async () => {
    if (!verifyPath) return;
    setVerifying(true);
    try {
      const res = await VerifyEvidenceContainer(verifyPath);
      setResult(res);
    } catch (err) {
      alert('Verification failed: ' + err);
    } finally {
      setVerifying(false);
    }
  };

  const styles = {
    container: {
      padding: '16px',
      backgroundColor: 'var(--win-bg)',
      color: 'var(--win-text)',
      fontFamily: 'var(--win-font)',
      height: '100%',
      boxSizing: 'border-box' as const,
      display: 'flex',
      flexDirection: 'column' as const,
      gap: '16px',
      overflow: 'auto',
    },
    title: {
      fontSize: '18px',
      fontWeight: '600',
      margin: 0,
    },
    section: {
      backgroundColor: 'var(--win-surface)',
      border: '1px solid var(--win-stroke)',
      borderRadius: 'var(--win-radius)',
      padding: '16px',
    },
    sectionTitle: {
      fontSize: '14px',
      fontWeight: '600',
      color: 'var(--win-accent)',
      margin: '0 0 12px 0',
    },
    field: {
      display: 'flex',
      flexDirection: 'column' as const,
      gap: '4px',
      marginBottom: '10px',
    },
    label: {
      fontSize: '12px',
      color: 'var(--win-text-secondary)',
      fontWeight: '500',
    },
    input: {
      padding: '8px 12px',
      backgroundColor: 'var(--win-bg)',
      border: '1px solid var(--win-stroke-strong)',
      borderRadius: 'var(--win-radius-sm)',
      color: 'var(--win-text)',
      fontSize: '13px',
      fontFamily: 'var(--win-font-mono)',
      outline: 'none',
    },
    btnRow: {
      display: 'flex',
      gap: '10px',
    },
    ingestBtn: {
      padding: '8px 20px',
      backgroundColor: 'var(--win-accent)',
      color: '#FFFFFF',
      border: 'none',
      borderRadius: 'var(--win-radius-sm)',
      cursor: 'pointer',
      fontSize: '13px',
      fontWeight: '600',
    },
    verifyBtn: {
      padding: '8px 20px',
      backgroundColor: 'var(--win-success)',
      color: '#FFFFFF',
      border: 'none',
      borderRadius: 'var(--win-radius-sm)',
      cursor: 'pointer',
      fontSize: '13px',
      fontWeight: '600',
    },
    resultCard: {
      backgroundColor: 'var(--win-bg)',
      border: '1px solid var(--win-stroke)',
      borderRadius: 'var(--win-radius-sm)',
      padding: '12px',
      marginTop: '12px',
    },
    resultRow: {
      display: 'flex',
      justifyContent: 'space-between',
      padding: '4px 0',
      fontSize: '13px',
      borderBottom: '1px solid var(--win-stroke)',
    },
    passBadge: {
      backgroundColor: '#E8F5E9',
      color: '#0F7B0F',
      padding: '2px 8px',
      borderRadius: '10px',
      fontSize: '11px',
      fontWeight: '600',
    },
    failBadge: {
      backgroundColor: '#FDE7E9',
      color: '#C42B1C',
      padding: '2px 8px',
      borderRadius: '10px',
      fontSize: '11px',
      fontWeight: '600',
    },
  };

  return (
    <div style={styles.container}>
      <h2 style={styles.title}>DVE1 Digital Evidence Container</h2>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
        <div style={styles.section}>
          <h3 style={styles.sectionTitle}>Ingest Evidence (Stream + Encrypt)</h3>
          <div style={styles.field}>
            <label style={styles.label}>Source Evidence Path</label>
            <input value={srcPath} onChange={e => setSrcPath(e.target.value)} placeholder="C:\evidence\disk.raw" style={styles.input} />
          </div>
          <div style={styles.field}>
            <label style={styles.label}>Output Case Directory</label>
            <input value={outputDir} onChange={e => setOutputDir(e.target.value)} style={styles.input} />
          </div>
          <div style={styles.field}>
            <label style={styles.label}>Vault ID</label>
            <input value={vaultId} onChange={e => setVaultId(e.target.value)} style={styles.input} />
          </div>
          <div style={styles.field}>
            <label style={styles.label}>Encryption Passphrase</label>
            <input type="password" value={passphrase} onChange={e => setPassphrase(e.target.value)} style={styles.input} />
          </div>
          <button onClick={handleIngest} disabled={ingesting} style={styles.ingestBtn}>
            {ingesting ? 'Ingesting...' : 'Ingest & Encrypt'}
          </button>

          {ingestResult && (
            <div style={styles.resultCard}>
              <div style={styles.resultRow}><span>Vault ID</span><span>{ingestResult.vault_id}</span></div>
              <div style={styles.resultRow}><span>Format</span><span>{ingestResult.magic} v{ingestResult.format_version}</span></div>
              <div style={styles.resultRow}><span>Cipher</span><span>{ingestResult.cipher}</span></div>
              <div style={styles.resultRow}><span>Total Size</span><span>{ingestResult.total_size_bytes?.toLocaleString()} bytes</span></div>
              <div style={styles.resultRow}><span>Chunks</span><span>{ingestResult.total_chunks}</span></div>
              <div style={styles.resultRow}><span>SHA-256</span><span style={{ fontFamily: 'var(--win-font-mono)', fontSize: '11px', wordBreak: 'break-all' }}>{ingestResult.source_sha256}</span></div>
            </div>
          )}
        </div>

        <div style={styles.section}>
          <h3 style={styles.sectionTitle}>Verify Evidence Container Integrity</h3>
          <div style={styles.field}>
            <label style={styles.label}>Case Directory Path</label>
            <input value={verifyPath} onChange={e => setVerifyPath(e.target.value)} placeholder="C:\evidence_cases\CASE-001.case" style={styles.input} />
          </div>
          <button onClick={handleVerify} disabled={verifying} style={styles.verifyBtn}>
            {verifying ? 'Verifying...' : 'Verify Container'}
          </button>

          {result && (
            <div style={styles.resultCard}>
              <div style={styles.resultRow}>
                <span>Manifest Format</span>
                <span>{result.manifest?.magic} v{result.manifest?.format_version}</span>
              </div>
              <div style={styles.resultRow}>
                <span>Vault ID</span>
                <span>{result.manifest?.vault_id}</span>
              </div>
              <div style={styles.resultRow}>
                <span>Total Size</span>
                <span>{result.manifest?.total_size_bytes?.toLocaleString()} bytes</span>
              </div>
              <div style={styles.resultRow}>
                <span>Chunks</span>
                <span>{result.manifest?.total_chunks}</span>
              </div>
              <div style={styles.resultRow}>
                <span>Audit Chain ({result.audit_count} records)</span>
                <span style={result.audit_valid ? styles.passBadge : styles.failBadge}>
                  {result.audit_valid ? 'VERIFIED' : 'TAMPERED'}
                </span>
              </div>
              <div style={styles.resultRow}>
                <span>Chunk Integrity</span>
                <span style={result.chunks_ok ? styles.passBadge : styles.failBadge}>
                  {result.chunks_ok ? 'PASS' : 'MISMATCH'}
                </span>
              </div>
              <div style={{ ...styles.resultRow, borderBottom: 'none', fontSize: '11px', fontFamily: 'var(--win-font-mono)', color: 'var(--win-text-secondary)', wordBreak: 'break-all' }}>
                <span>Source SHA-256: {result.manifest?.source_sha256}</span>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

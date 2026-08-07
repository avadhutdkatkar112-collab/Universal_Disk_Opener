import React, { useState } from 'react';
import { IngestEvidenceStruct, VerifyCaseIntegrity } from '../lib/wails';

interface VerificationResult {
  valid: boolean;
  audit_count: number;
  manifest_present: boolean;
  source_hash: string;
  message: string;
}

interface Props {
  onCaseActivated: (caseId: string, hash: string, verified: boolean) => void;
}

export const EvidenceManager: React.FC<Props> = ({ onCaseActivated }) => {
  const [sourcePath, setSourcePath] = useState('');
  const [caseDir, setCaseDir] = useState('');
  const [passphrase, setPassphrase] = useState('');
  const [caseId, setCaseId] = useState('CASE-2026-001');
  const [actor, setActor] = useState('Lead Examiner');
  const [loading, setLoading] = useState(false);
  const [statusMsg, setStatusMsg] = useState('');
  const [verification, setVerification] = useState<VerificationResult | null>(null);

  const handleIngest = async () => {
    if (!sourcePath || !caseDir || !passphrase) {
      setStatusMsg('Please provide source path, output directory, and passphrase.');
      return;
    }
    setLoading(true);
    setStatusMsg('Encrypting evidence and computing SHA-256 in 4 MiB streaming chunks...');
    try {
      const manifest = await IngestEvidenceStruct({
        source_path: sourcePath,
        case_dir: caseDir,
        passphrase: passphrase,
        case_id: caseId,
        actor: actor,
      });
      setStatusMsg(`Ingestion complete. Source SHA-256: ${manifest.source_sha256}`);
      onCaseActivated(caseId, manifest.source_sha256, true);
    } catch (err: any) {
      setStatusMsg(`Ingestion failed: ${err?.toString() || err}`);
    } finally {
      setLoading(false);
    }
  };

  const handleVerify = async () => {
    if (!caseDir) {
      setStatusMsg('Specify Case Directory to verify.');
      return;
    }
    try {
      const res = await VerifyCaseIntegrity(caseDir);
      setVerification(res);
      setStatusMsg(res.message);
      if (res.valid) {
        onCaseActivated(caseId, res.source_hash, true);
      }
    } catch (err: any) {
      setStatusMsg(`Verification failed: ${err?.toString() || err}`);
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
      overflow: 'auto',
    },
    title: {
      fontSize: '18px',
      fontWeight: '600',
      margin: '0 0 16px 0',
    },
    grid: {
      display: 'grid',
      gridTemplateColumns: '1fr 1fr',
      gap: '16px',
    },
    panel: {
      backgroundColor: 'var(--win-surface)',
      padding: '20px',
      borderRadius: 'var(--win-radius)',
      border: '1px solid var(--win-stroke)',
    },
    panelTitle: {
      fontSize: '14px',
      fontWeight: '600',
      margin: '0 0 16px 0',
    },
    field: {
      marginBottom: '12px',
    },
    label: {
      display: 'block',
      fontSize: '12px',
      color: 'var(--win-text-secondary)',
      marginBottom: '4px',
    },
    input: {
      width: '100%',
      padding: '8px 12px',
      backgroundColor: 'var(--win-bg)',
      border: '1px solid var(--win-stroke-strong)',
      borderRadius: 'var(--win-radius-sm)',
      color: 'var(--win-text)',
      fontSize: '13px',
      fontFamily: 'var(--win-font)',
      outline: 'none',
      boxSizing: 'border-box' as const,
    },
    btn: {
      width: '100%',
      padding: '10px',
      border: 'none',
      borderRadius: 'var(--win-radius-sm)',
      cursor: 'pointer',
      fontSize: '13px',
      fontWeight: '600',
    },
    resultCard: (valid: boolean) => ({
      marginTop: '16px',
      padding: '16px',
      borderRadius: 'var(--win-radius-sm)',
      backgroundColor: valid ? '#E8F5E9' : '#FDE7E9',
      border: `1px solid ${valid ? '#0F7B0F' : '#C42B1C'}`,
    }),
    resultTitle: (valid: boolean) => ({
      margin: '0 0 8px 0',
      fontSize: '14px',
      fontWeight: '600',
      color: valid ? '#0F7B0F' : '#C42B1C',
    }),
    resultRow: {
      fontSize: '12px',
      margin: '4px 0',
      color: 'var(--win-text)',
    },
    statusMsg: {
      marginTop: '16px',
      padding: '12px',
      backgroundColor: 'var(--win-surface)',
      borderLeft: '4px solid var(--win-accent)',
      fontSize: '13px',
      fontFamily: 'var(--win-font-mono)',
      borderRadius: 'var(--win-radius-sm)',
    },
  };

  return (
    <div style={styles.container}>
      <h2 style={styles.title}>Digital Evidence Vault (DVE1 Engine)</h2>

      <div style={styles.grid}>
        <div style={styles.panel}>
          <h3 style={{ ...styles.panelTitle, color: 'var(--win-accent)' }}>Ingest & Containerize Evidence</h3>

          <div style={styles.field}>
            <label style={styles.label}>Source Evidence Path (.raw, .vhdx, .evtx):</label>
            <input value={sourcePath} onChange={e => setSourcePath(e.target.value)} style={styles.input} />
          </div>
          <div style={styles.field}>
            <label style={styles.label}>Destination Case Directory:</label>
            <input value={caseDir} onChange={e => setCaseDir(e.target.value)} style={styles.input} />
          </div>
          <div style={styles.field}>
            <label style={styles.label}>Passphrase (Argon2id Key Derivation):</label>
            <input type="password" value={passphrase} onChange={e => setPassphrase(e.target.value)} style={styles.input} />
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px', marginBottom: '16px' }}>
            <div style={styles.field}>
              <label style={styles.label}>Case ID:</label>
              <input value={caseId} onChange={e => setCaseId(e.target.value)} style={styles.input} />
            </div>
            <div style={styles.field}>
              <label style={styles.label}>Examiner Handle:</label>
              <input value={actor} onChange={e => setActor(e.target.value)} style={styles.input} />
            </div>
          </div>

          <button onClick={handleIngest} disabled={loading} style={{ ...styles.btn, backgroundColor: 'var(--win-accent)', color: '#fff' }}>
            {loading ? 'Ingesting...' : 'Ingest to DVE1 Container'}
          </button>
        </div>

        <div style={styles.panel}>
          <h3 style={{ ...styles.panelTitle, color: '#9D5D00' }}>Chain of Custody Verification</h3>
          <p style={{ fontSize: '13px', color: 'var(--win-text-secondary)', marginBottom: '16px' }}>
            Verifies the DVE1 manifest and validates tamper-evident SHA-256 hash linkages across all audit records.
          </p>

          <button onClick={handleVerify} style={{ ...styles.btn, backgroundColor: '#9D5D00', color: '#fff', marginBottom: '16px' }}>
            Verify Case Integrity
          </button>

          {verification && (
            <div style={styles.resultCard(verification.valid)}>
              <h4 style={styles.resultTitle(verification.valid)}>
                {verification.valid ? 'CONTAINER VERIFIED' : 'VERIFICATION FAILED'}
              </h4>
              <p style={styles.resultRow}>{verification.message}</p>
              <p style={styles.resultRow}><strong>Audit Records:</strong> {verification.audit_count}</p>
              <p style={{ ...styles.resultRow, fontFamily: 'var(--win-font-mono)', wordBreak: 'break-all' }}>
                <strong>Source SHA-256:</strong> {verification.source_hash}
              </p>
            </div>
          )}
        </div>
      </div>

      {statusMsg && (
        <div style={styles.statusMsg}>{statusMsg}</div>
      )}
    </div>
  );
};

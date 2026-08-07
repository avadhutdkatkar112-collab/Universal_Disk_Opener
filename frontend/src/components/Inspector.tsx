import { useState, useCallback } from 'react';
import { Icon } from './Icon';
import { useEvidenceStore } from '../store/evidenceStore';
import { fmtSize } from '../lib/utils';
import { HashFile } from '../lib/wails';

interface Props {
  selectedEntry?: { name: string; isDir: boolean; size?: number; modified?: string; path?: string } | null;
}

export function Inspector({ selectedEntry }: Props) {
  const evidence = useEvidenceStore();
  const [hashResult, setHashResult] = useState<string | null>(null);
  const [hashing, setHashing] = useState(false);

  const handleHash = useCallback(async (path: string) => {
    setHashing(true);
    setHashResult(null);
    try {
      const result = await HashFile(path);
      setHashResult(result.sha256 || 'No hash available');
    } catch {
      setHashResult('Hash failed');
    } finally {
      setHashing(false);
    }
  }, []);

  if (!evidence.isActive) return null;

  const part = evidence.partitions.find(p => p.index === evidence.selectedPartition);

  return (
    <aside aria-label="Inspector" style={{
      width: 260, borderLeft: '1px solid var(--win-stroke)', background: 'var(--win-surface)',
      display: 'flex', flexDirection: 'column', flexShrink: 0, overflow: 'hidden',
    }}>
      <div style={{ height: 32, display: 'flex', alignItems: 'center', gap: 6, padding: '0 12px', borderBottom: '1px solid var(--win-stroke)', flexShrink: 0 }}>
        <Icon name="info" size={12} style={{ color: 'var(--win-accent)' }} />
        <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--win-text-secondary)' }}>Inspector</span>
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: '8px 0' }}>
        {selectedEntry && (
          <>
            <SectionTitle icon={selectedEntry.isDir ? 'folder' : 'file'} title={selectedEntry.isDir ? 'Folder' : 'File'} />
            <InfoRow label="Name" value={selectedEntry.name} />
            {!selectedEntry.isDir && selectedEntry.size != null && (
              <InfoRow label="Size" value={fmtSize(selectedEntry.size)} />
            )}
            {selectedEntry.path && <InfoRow label="Path" value={selectedEntry.path} />}
            <div style={{ height: 1, background: 'var(--win-stroke)', margin: '6px 12px' }} />
          </>
        )}

        <SectionTitle icon="disk" title="Container" />
        <InfoRow label="File" value={evidence.fileName} />
        <InfoRow label="Format" value={evidence.format} />
        <InfoRow label="Size" value={fmtSize(evidence.totalSize)} />

        {part && (
          <>
            <SectionTitle icon="folder" title="Filesystem" />
            <InfoRow label="Type" value={part.type} />
            <InfoRow label="Part Size" value={fmtSize(part.size)} />
          </>
        )}

        {evidence.partitions.length > 0 && (
          <>
            <SectionTitle icon="analyze" title="Storage Layout" />
            <div style={{ padding: '4px 12px' }}>
              <div style={{ height: 6, borderRadius: 3, background: 'var(--win-stroke-strong)', overflow: 'hidden', display: 'flex' }}>
                {evidence.partitions.map((p, i) => (
                  <div key={i} style={{
                    width: `${(p.size / evidence.totalSize) * 100}%`,
                    background: p.index === evidence.selectedPartition ? 'var(--win-accent-default)' : 'var(--win-text-tertiary)',
                    borderRight: i < evidence.partitions.length - 1 ? '2px solid var(--win-surface)' : 'none',
                  }} />
                ))}
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 4, fontSize: 10, color: 'var(--win-text-tertiary)' }}>
                <span>{evidence.partitions.length} partition{evidence.partitions.length !== 1 ? 's' : ''}</span>
                <span>{fmtSize(evidence.totalSize)}</span>
              </div>
            </div>
          </>
        )}

        {/* Hash result */}
        {hashResult && (
          <div style={{ margin: '8px 12px', padding: '6px 8px', background: 'var(--win-bg)', borderRadius: 'var(--win-radius-sm)', border: '1px solid var(--win-stroke)' }}>
            <div style={{ fontSize: 10, color: 'var(--win-text-tertiary)', marginBottom: 2 }}>SHA-256</div>
            <div style={{ fontSize: 10, fontFamily: 'var(--win-font-mono)', color: 'var(--win-text)', wordBreak: 'break-all' }}>{hashResult}</div>
          </div>
        )}

        <SectionTitle icon="settings" title={selectedEntry ? 'File Actions' : 'Quick Actions'} />
        <div style={{ padding: '4px 12px', display: 'flex', gap: 6 }}>
          {selectedEntry && !selectedEntry.isDir && selectedEntry.path ? (
            <>
              <ActionBtn
                icon="hash"
                label={hashing ? 'Hashing...' : 'Hash'}
                onClick={() => handleHash(selectedEntry.path!)}
                disabled={hashing}
              />
              <ActionBtn icon="extract" label="Extract" />
            </>
          ) : (
            <>
              <ActionBtn icon="analyze" label="Analyze" disabled />
              <ActionBtn icon="extract" label="Extract" disabled />
            </>
          )}
        </div>
      </div>
    </aside>
  );
}

function SectionTitle({ icon, title }: { icon: any; title: string }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '10px 12px 4px' }}>
      <Icon name={icon} size={12} style={{ color: 'var(--win-accent)' }} />
      <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--win-text-secondary)' }}>{title}</span>
    </div>
  );
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '3px 12px', fontSize: 12 }}>
      <span style={{ color: 'var(--win-text-tertiary)' }}>{label}</span>
      <span style={{
        color: 'var(--win-text-secondary)', fontFamily: 'var(--win-font-mono)', fontSize: 11,
        maxWidth: 130, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', textAlign: 'right',
      }}>{value}</span>
    </div>
  );
}

function ActionBtn({ icon, label, onClick, disabled }: {
  icon: any; label: string; onClick?: () => void; disabled?: boolean;
}) {
  return (
    <button
      aria-label={label}
      disabled={disabled}
      onClick={onClick}
      style={{
        flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 4,
        padding: '8px 4px', borderRadius: 'var(--win-radius-sm)',
        border: '1px solid var(--win-stroke)', background: 'var(--win-card)',
        fontSize: 11, color: disabled ? 'var(--win-text-disabled)' : 'var(--win-text-secondary)',
        opacity: disabled ? 0.5 : 1, cursor: disabled ? 'default' : 'pointer',
      }}
      onMouseEnter={e => { if (!disabled) e.currentTarget.style.background = 'var(--win-control-hover)'; }}
      onMouseLeave={e => { if (!disabled) e.currentTarget.style.background = 'var(--win-card)'; }}
    >
      <Icon name={icon} size={14} style={{ color: 'var(--win-accent)' }} />
      {label}
    </button>
  );
}

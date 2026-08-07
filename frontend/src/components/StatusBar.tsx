import type { DiskSnapshot } from '../types';
import { useJobStore } from '../store/jobStore';
import { fmtSize } from '../lib/utils';

interface Props {
  disk: DiskSnapshot | null;
}

export function StatusBar({ disk }: Props) {
  const part = disk?.partitions[disk.activePartition];
  const jobs = useJobStore(s => s.jobs);
  const activeJobs = jobs.filter(j => j.status === 'running');

  return (
    <footer
      role="contentinfo"
      aria-label="Status bar"
      style={{
        height: 24, display: 'flex', alignItems: 'center', gap: 12, padding: '0 12px',
        background: 'var(--win-surface)', borderTop: '1px solid var(--win-stroke)',
        fontSize: 12, color: 'var(--win-text-secondary)', flexShrink: 0, zIndex: 40,
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
        <div
          role="status"
          aria-label={disk ? 'Ready' : 'No disk loaded'}
          style={{
            width: 8, height: 8, borderRadius: '50%',
            background: disk ? 'var(--win-success)' : 'var(--win-text-disabled)',
          }}
        />
        <span>{disk ? 'Ready' : 'No disk loaded'}</span>
      </div>

      {disk && (
        <>
          <div style={{ width: 1, height: 12, background: 'var(--win-stroke-strong)' }} />
          <span>{disk.format}</span>
          <div style={{ width: 1, height: 12, background: 'var(--win-stroke-strong)' }} />
          <span>{part?.fsType || '—'}</span>
          <div style={{ width: 1, height: 12, background: 'var(--win-stroke-strong)' }} />
          <span style={{ color: 'var(--win-text-tertiary)' }}>{fmtSize(disk.totalSize)}</span>
        </>
      )}

      <div style={{ flex: 1 }} />

      {activeJobs.length > 0 && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <div style={{ width: 80, height: 3, borderRadius: 2, background: 'var(--win-stroke-strong)', overflow: 'hidden' }}>
            <div style={{ height: '100%', width: `${activeJobs[0].progress}%`, background: 'var(--win-accent-default)', borderRadius: 2, transition: 'width 0.2s' }} />
          </div>
          <span style={{ fontSize: 11, color: 'var(--win-text-tertiary)' }}>{activeJobs[0].progress}%</span>
        </div>
      )}

      <span style={{ color: 'var(--win-text-tertiary)', fontSize: 11 }}>v0.8</span>
    </footer>
  );
}

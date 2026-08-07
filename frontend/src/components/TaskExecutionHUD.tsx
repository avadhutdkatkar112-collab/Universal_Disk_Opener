import { Icon } from './Icon';
import { useJobStore } from '../store/jobStore';

export function TaskExecutionHUD() {
  const jobs = useJobStore(s => s.jobs);
  const cancelJob = useJobStore(s => s.cancelJob);
  const activeJobs = jobs.filter(j => j.status === 'running');

  if (activeJobs.length === 0) return null;

  return (
    <div
      role="status"
      aria-label="Active tasks"
      style={{
        position: 'fixed', bottom: 36, right: 12, zIndex: 200, width: 240,
        background: 'var(--win-surface)', border: '1px solid var(--win-stroke-strong)',
        borderRadius: 'var(--win-radius)', boxShadow: 'var(--win-shadow)', overflow: 'hidden',
        animation: 'scaleIn 0.15s ease',
      }}
    >
      {activeJobs.slice(0, 3).map(job => (
        <div key={job.id} style={{ padding: '8px 10px', borderBottom: '1px solid var(--win-stroke)' }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 6 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 12, color: 'var(--win-text-secondary)', overflow: 'hidden' }}>
              <div style={{
                width: 12, height: 12, border: '2px solid var(--win-stroke-strong)',
                borderTopColor: 'var(--win-accent)', borderRadius: '50%',
                animation: 'spin 0.6s linear infinite', flexShrink: 0,
              }} />
              <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                {job.label || job.type}
              </span>
            </div>
            <button
              aria-label={`Cancel ${job.label || job.type}`}
              onClick={() => cancelJob(job.id)}
              style={{
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                width: 18, height: 18, borderRadius: 'var(--win-radius-sm)', flexShrink: 0,
              }}
              onMouseEnter={e => (e.currentTarget.style.background = 'rgba(196, 43, 28, 0.1)')}
              onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
            >
              <Icon name="cancel" size={10} style={{ color: 'var(--win-danger)' }} />
            </button>
          </div>
          <div style={{ height: 4, borderRadius: 2, background: 'var(--win-stroke-strong)', overflow: 'hidden' }}>
            <div style={{ height: '100%', width: `${job.progress}%`, background: 'var(--win-accent-default)', borderRadius: 2, transition: 'width 0.2s' }} />
          </div>
          <div style={{ fontSize: 11, color: 'var(--win-text-tertiary)', marginTop: 3, textAlign: 'right' }}>{job.progress}%</div>
        </div>
      ))}
    </div>
  );
}

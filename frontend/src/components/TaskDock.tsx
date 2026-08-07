import { useJobManager, type HashJobProgress } from '../hooks/useJobManager';
import { Icon } from './Icon';
import { fmtSize } from '../lib/utils';

function formatEta(seconds: number): string {
  if (seconds <= 0) return '—';
  if (seconds < 60) return `${Math.ceil(seconds)}s`;
  const mins = Math.floor(seconds / 60);
  const secs = Math.ceil(seconds % 60);
  return `${mins}m ${secs}s`;
}

interface JobRowProps {
  job: HashJobProgress;
  onRecall?: (job: HashJobProgress) => void;
}

function JobRow({ job, onRecall }: JobRowProps) {
  const isRunning = job.status === 'RUNNING';
  const isComplete = job.status === 'COMPLETED';
  const isFailed = job.status === 'FAILED';
  const isClickable = isComplete && onRecall;

  return (
    <div
      onClick={isClickable ? () => onRecall(job) : undefined}
      style={{
        padding: '6px 8px', borderRadius: 4,
        border: '1px solid var(--win-stroke)',
        background: 'var(--win-card)',
        marginBottom: 4,
        cursor: isClickable ? 'pointer' : 'default',
      }}
      onMouseEnter={e => {
        if (isClickable) e.currentTarget.style.background = 'var(--win-subtle-hover)';
      }}
      onMouseLeave={e => {
        if (isClickable) e.currentTarget.style.background = 'var(--win-card)';
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: isRunning ? 4 : 0 }}>
        {isRunning && (
          <div style={{
            width: 10, height: 10, border: '2px solid var(--win-stroke-strong)',
            borderTopColor: 'var(--win-accent)', borderRadius: '50%',
            animation: 'spin 0.6s linear infinite', flexShrink: 0,
          }} />
        )}
        {isComplete && (
          <Icon name="check" size={10} style={{ color: 'var(--win-success)', flexShrink: 0 }} />
        )}
        {isFailed && (
          <Icon name="alert" size={10} style={{ color: 'var(--win-danger)', flexShrink: 0 }} />
        )}
        <span style={{
          flex: 1, fontSize: 11, color: 'var(--win-text-secondary)',
          overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
          fontFamily: 'var(--win-font-mono)',
        }}>
          {job.target_path}
        </span>
        <span style={{
          fontSize: 10, fontWeight: 500, flexShrink: 0,
          color: isRunning ? 'var(--win-accent)' : isComplete ? 'var(--win-success)' : 'var(--win-danger)',
        }}>
          {isRunning ? `${(job.percentage ?? 0).toFixed(1)}%` : isComplete ? 'DONE' : 'FAILED'}
        </span>
      </div>

      {isRunning && (
        <>
          <div style={{
            height: 3, borderRadius: 2, background: 'var(--win-stroke-strong)',
            overflow: 'hidden', marginBottom: 4,
          }}>
            <div style={{
              height: '100%', width: `${job.percentage ?? 0}%`,
              background: 'var(--win-accent-default)', borderRadius: 2,
              transition: 'width 0.15s ease-out',
            }} />
          </div>
          <div style={{
            display: 'flex', justifyContent: 'space-between',
            fontSize: 10, color: 'var(--win-text-tertiary)',
          }}>
            <span>{(job.throughput_mbps ?? 0) > 0 ? `${(job.throughput_mbps ?? 0).toFixed(1)} MB/s` : 'Instant'}</span>
            <span>ETA: {formatEta(job.eta_seconds ?? 0)}</span>
            <span>{fmtSize(job.bytes_processed ?? 0)} / {fmtSize(job.total_bytes ?? 0)}</span>
          </div>
        </>
      )}

      {isComplete && job.result && (
        <div style={{
          display: 'flex', gap: 8, marginTop: 4,
          fontSize: 10, color: 'var(--win-text-tertiary)',
        }}>
          {job.result.md5 && <span>MD5: {job.result.md5.slice(0, 8)}...</span>}
          {job.result.sha256 && <span>SHA-256: {job.result.sha256.slice(0, 8)}...</span>}
          {job.result.total_files && <span>{job.result.total_files} files</span>}
          {onRecall && (
            <span style={{ color: 'var(--win-accent)', marginLeft: 'auto' }}>
              Click to view
            </span>
          )}
        </div>
      )}

      {isFailed && job.error && (
        <div style={{ fontSize: 10, color: 'var(--win-danger)', marginTop: 2 }}>
          {job.error}
        </div>
      )}
    </div>
  );
}

interface TaskDockProps {
  onRecall?: (job: HashJobProgress) => void;
}

export function TaskDock({ onRecall }: TaskDockProps) {
  const { activeJobs } = useJobManager();
  const jobList = Object.values(activeJobs);

  // Sort: running first, then completed, then failed
  const sortedJobs = [...jobList].sort((a, b) => {
    const order = { RUNNING: 0, PENDING: 1, COMPLETED: 2, FAILED: 3 };
    return (order[a.status] ?? 4) - (order[b.status] ?? 4);
  });

  if (sortedJobs.length === 0) {
    return (
      <div style={{
        color: 'var(--win-text-tertiary)', padding: '14px 0',
        textAlign: 'center', fontSize: 11,
      }}>
        No active tasks
      </div>
    );
  }

  const runningCount = sortedJobs.filter(j => j.status === 'RUNNING').length;
  const completedCount = sortedJobs.filter(j => j.status === 'COMPLETED').length;

  return (
    <div style={{ maxHeight: 180, overflow: 'auto' }}>
      {(runningCount > 0 || completedCount > 0) && (
        <div style={{
          display: 'flex', gap: 8, padding: '0 4px 6px',
          fontSize: 10, color: 'var(--win-text-tertiary)',
        }}>
          {runningCount > 0 && <span>{runningCount} running</span>}
          {completedCount > 0 && <span>{completedCount} completed</span>}
        </div>
      )}
      {sortedJobs.map(job => (
        <JobRow key={job.job_id} job={job} onRecall={onRecall} />
      ))}
    </div>
  );
}

import { useState, useEffect, useCallback } from 'react';
import { Icon } from './Icon';
import { useJobStore } from '../store/jobStore';
import { TaskDock } from './TaskDock';
import type { HashJobProgress } from '../hooks/useJobManager';

interface Props {
  onRecallJob?: (job: HashJobProgress) => void;
}

export function BottomPanel({ onRecallJob }: Props) {
  const [expanded, setExpanded] = useState(false);
  const [tab, setTab] = useState<'tasks' | 'logs' | 'output'>('tasks');
  const jobs = useJobStore(s => s.jobs);
  const activeJobs = jobs.filter(j => j.status === 'running' || j.status === 'queued');

  useEffect(() => {
    if (activeJobs.length > 0 && !expanded) setExpanded(true);
  }, [activeJobs.length]);

  const toggleExpand = useCallback(() => setExpanded(v => !v), []);

  const tabs = [
    { id: 'tasks' as const, label: 'Tasks', count: activeJobs.length },
    { id: 'logs' as const, label: 'Logs' },
    { id: 'output' as const, label: 'Output' },
  ];

  return (
    <div style={{ borderTop: '1px solid var(--win-stroke)', background: 'var(--win-surface)', flexShrink: 0 }}>
      <div style={{ display: 'flex', alignItems: 'center', height: 28, padding: '0 8px', gap: 0, borderBottom: expanded ? '1px solid var(--win-stroke)' : 'none' }}>
        {/* Drag handle */}
        <div style={{ width: 16, display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'ns-resize', marginRight: 2 }}>
          <div style={{ width: 12, height: 2, borderRadius: 1, background: 'var(--win-stroke-strong)' }} />
        </div>
        <div role="tablist" aria-label="Bottom panel tabs" style={{ display: 'flex', gap: 0 }}>
          {tabs.map(t => (
            <button
              key={t.id}
              role="tab"
              aria-selected={tab === t.id && expanded}
              aria-controls={`panel-${t.id}`}
              onClick={() => { setTab(t.id); setExpanded(true); }}
              style={{
                display: 'flex', alignItems: 'center', gap: 4, padding: '3px 8px', fontSize: 11,
                fontWeight: tab === t.id && expanded ? 500 : 400,
                color: tab === t.id && expanded ? 'var(--win-text)' : 'var(--win-text-tertiary)',
                borderBottom: tab === t.id && expanded ? '2px solid var(--win-accent-default)' : '2px solid transparent',
              }}
              onMouseEnter={e => (e.currentTarget.style.color = 'var(--win-text-secondary)')}
              onMouseLeave={e => (e.currentTarget.style.color = tab === t.id && expanded ? 'var(--win-text)' : 'var(--win-text-tertiary)')}
            >
              {t.label}
              {t.count !== undefined && t.count > 0 && (
                <span style={{
                  padding: '0 4px', borderRadius: 8, background: 'var(--win-accent-default)',
                  color: '#fff', fontSize: 9, fontWeight: 600, lineHeight: '14px',
                }}>{t.count}</span>
              )}
            </button>
          ))}
        </div>
        <div style={{ flex: 1 }} />
        <button
          aria-label={expanded ? 'Collapse panel' : 'Expand panel'}
          aria-expanded={expanded}
          onClick={toggleExpand}
          style={{
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            width: 18, height: 18, borderRadius: 'var(--win-radius-sm)',
          }}
          onMouseEnter={e => (e.currentTarget.style.background = 'var(--win-subtle-hover)')}
          onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
        >
          <Icon name={expanded ? 'chevron-down' : 'chevron-right'} size={10} style={{ color: 'var(--win-text-tertiary)' }} />
        </button>
      </div>

      {expanded && (
        <div id={`panel-${tab}`} role="tabpanel" aria-label={tab} style={{ height: 120, overflow: 'auto', padding: '8px 14px', fontSize: 12 }}>
          {tab === 'tasks' && (
            <TaskDock onRecall={onRecallJob} />
          )}
          {tab === 'logs' && <div style={{ color: 'var(--win-text-tertiary)', padding: '14px 0', textAlign: 'center', fontSize: 11 }}>No logs yet</div>}
          {tab === 'output' && <div style={{ color: 'var(--win-text-tertiary)', padding: '14px 0', textAlign: 'center', fontSize: 11 }}>No output yet</div>}
        </div>
      )}
    </div>
  );
}

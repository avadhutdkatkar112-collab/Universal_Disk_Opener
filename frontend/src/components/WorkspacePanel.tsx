import { useState, useCallback } from 'react';
import { Icon } from './Icon';
import { useDiskStore } from '../store/diskStore';
import { fmtSize } from '../lib/utils';
import { openDisk } from '../lib/openDisk';

export function WorkspacePanel() {
  const { disk } = useDiskStore();
  const [collapsed, setCollapsed] = useState(false);
  const [partExpanded, setPartExpanded] = useState(true);

  const toggleCollapse = useCallback(() => setCollapsed(v => !v), []);
  const togglePartitions = useCallback(() => setPartExpanded(v => !v), []);

  return (
    <nav
      aria-label="Workspace"
      style={{
        width: collapsed ? 40 : 200, display: 'flex', flexDirection: 'column',
        background: 'var(--win-surface)', borderRight: '1px solid var(--win-stroke)',
        flexShrink: 0, transition: 'width 0.15s ease', overflow: 'hidden',
      }}
    >
      <div style={{ height: 32, display: 'flex', alignItems: 'center', gap: 6, padding: '0 8px', borderBottom: '1px solid var(--win-stroke)', flexShrink: 0 }}>
        <button
          aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          aria-expanded={!collapsed}
          onClick={toggleCollapse}
          style={{
            display: 'flex', alignItems: 'center', justifyContent: 'center', width: 24, height: 24,
            borderRadius: 'var(--win-radius-sm)',
          }}
          onMouseEnter={e => (e.currentTarget.style.background = 'var(--win-subtle-hover)')}
          onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
        >
          <Icon name={collapsed ? 'chevron-right' : 'chevron-down'} size={12} style={{ color: 'var(--win-text-secondary)' }} />
        </button>
        {!collapsed && (
          <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--win-text-secondary)' }}>Workspace</span>
        )}
      </div>

      {!collapsed && (
        <div style={{ flex: 1, overflow: 'auto', padding: '8px 0' }}>
          {disk ? (
            <>
              <div style={{ padding: '0 8px', marginBottom: 8 }}>
                <div style={{
                  display: 'flex', alignItems: 'center', gap: 8, padding: '6px 8px',
                  borderRadius: 'var(--win-radius-sm)', background: 'var(--win-accent-default)',
                  fontSize: 12, fontWeight: 500, color: '#fff',
                }}>
                  <Icon name="disk" size={14} style={{ color: '#fff', flexShrink: 0 }} />
                  <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{disk.fileName}</span>
                </div>
                <div style={{ marginTop: 6, padding: '0 8px', fontSize: 11, color: 'var(--win-text-tertiary)', lineHeight: '18px' }}>
                  <div>{disk.format} · {disk.partitions.length} partition{disk.partitions.length !== 1 ? 's' : ''}</div>
                </div>
              </div>

              <TreeSection title="Partitions" icon="analyze" expanded={partExpanded} onToggle={togglePartitions}>
                {disk.partitions.map((p, i) => (
                  <div key={i}>
                    <div
                      role="button"
                      tabIndex={0}
                      style={{
                        display: 'flex', alignItems: 'center', gap: 6, padding: '3px 8px 3px 28px',
                        fontSize: 11, margin: '0 4px', borderRadius: 'var(--win-radius-sm)',
                        color: i === disk.activePartition ? 'var(--win-text)' : 'var(--win-text-tertiary)',
                      }}
                      onMouseEnter={e => (e.currentTarget.style.background = 'var(--win-subtle-hover)')}
                      onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                    >
                      <div style={{
                        width: 6, height: 6, borderRadius: 2, flexShrink: 0,
                        background: i === disk.activePartition ? 'var(--win-accent-default)' : 'var(--win-text-tertiary)',
                      }} />
                      <span style={{ fontWeight: i === disk.activePartition ? 500 : 400, flex: 1 }}>P{i} — {p.fsType}</span>
                      <span style={{ fontSize: 10, color: 'var(--win-text-tertiary)' }}>{fmtSize(p.sizeBytes)}</span>
                    </div>
                    {/* Storage usage bar */}
                    <div style={{ padding: '2px 28px 4px 34px' }}>
                      <div style={{
                        height: 3, borderRadius: 2, background: 'var(--win-stroke-strong)',
                        overflow: 'hidden',
                      }}>
                        <div style={{
                          height: '100%',
                          width: `${disk.totalSize > 0 ? (p.sizeBytes / disk.totalSize) * 100 : 0}%`,
                          background: i === disk.activePartition ? 'var(--win-accent-default)' : 'var(--win-text-tertiary)',
                          borderRadius: 2, transition: 'width 0.2s',
                        }} />
                      </div>
                    </div>
                  </div>
                ))}
              </TreeSection>

              <div
                role="button"
                tabIndex={0}
                onClick={openDisk}
                style={{
                  margin: '8px 8px 0', padding: '6px', borderRadius: 'var(--win-radius-sm)',
                  border: '1px solid var(--win-stroke-strong)', display: 'flex',
                  alignItems: 'center', justifyContent: 'center', gap: 6, fontSize: 11,
                  color: 'var(--win-text-tertiary)', background: 'var(--win-subtle)',
                }}
                onMouseEnter={e => {
                  e.currentTarget.style.background = 'var(--win-subtle-hover)';
                  e.currentTarget.style.borderColor = 'var(--win-accent-default)';
                  e.currentTarget.style.color = 'var(--win-text-secondary)';
                }}
                onMouseLeave={e => {
                  e.currentTarget.style.background = 'var(--win-subtle)';
                  e.currentTarget.style.borderColor = 'var(--win-stroke-strong)';
                  e.currentTarget.style.color = 'var(--win-text-tertiary)';
                }}
              >
                <Icon name="plus" size={12} /> Open another
              </div>
            </>
          ) : (
            <div style={{ padding: '20px 12px', textAlign: 'center' }}>
              <div style={{
                width: 40, height: 40, borderRadius: 'var(--win-radius)',
                border: '1px solid var(--win-stroke-strong)', display: 'flex',
                alignItems: 'center', justifyContent: 'center', margin: '0 auto 10px',
                background: 'var(--win-subtle)',
              }}>
                <Icon name="disk" size={18} style={{ color: 'var(--win-text-tertiary)' }} />
              </div>
              <div style={{ fontSize: 12, color: 'var(--win-text-tertiary)' }}>No disk loaded</div>
            </div>
          )}
        </div>
      )}
    </nav>
  );
}

function TreeSection({ title, icon, expanded, onToggle, children }: {
  title: string; icon: any; expanded: boolean; onToggle: () => void; children: React.ReactNode;
}) {
  return (
    <div>
      <button
        aria-expanded={expanded}
        onClick={onToggle}
        style={{
          display: 'flex', alignItems: 'center', gap: 4, padding: '4px 8px', width: '100%',
          fontSize: 11, fontWeight: 600, color: 'var(--win-text-secondary)', userSelect: 'none',
        }}
        onMouseEnter={e => (e.currentTarget.style.background = 'var(--win-subtle-hover)')}
        onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
      >
        <Icon name={expanded ? 'chevron-down' : 'chevron-right'} size={10} style={{ color: 'var(--win-text-tertiary)' }} />
        <Icon name={icon} size={11} style={{ color: 'var(--win-accent)' }} />
        <span>{title}</span>
      </button>
      {expanded && children}
    </div>
  );
}

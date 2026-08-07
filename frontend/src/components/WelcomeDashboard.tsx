import { useState, useEffect } from 'react';
import { Icon } from './Icon';
import { GetRecentFiles } from '../lib/wails';

interface Props {
  onOpen: () => void;
}

interface RecentFile {
  path: string;
  name: string;
  size: number;
  openedAt: number;
}

const FORMATS = [
  { name: 'VHD', color: '#0078D4' },
  { name: 'VHDX', color: '#0078D4' },
  { name: 'VMDK', color: '#6CCB5F' },
  { name: 'QCOW2', color: '#C19C00' },
  { name: 'VDI', color: '#9B59B6' },
  { name: 'RAW', color: '#7F8C8D' },
  { name: 'ISO', color: '#C0392B' },
];

export function WelcomeDashboard({ onOpen }: Props) {
  const [recentFiles, setRecentFiles] = useState<RecentFile[]>([]);

  useEffect(() => {
    GetRecentFiles().then(files => setRecentFiles(files || [])).catch(() => {});
  }, []);

  const formatSize = (bytes: number) => {
    if (bytes === 0) return '-';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
  };

  const formatTimestamp = (ts: number) => {
    if (!ts) return '';
    const d = new Date(ts);
    const now = new Date();
    const diffDays = Math.floor((now.getTime() - d.getTime()) / (1000 * 60 * 60 * 24));
    if (diffDays === 0) return 'Today';
    if (diffDays === 1) return 'Yesterday';
    if (diffDays < 7) return `${diffDays} days ago`;
    return d.toLocaleDateString();
  };

  return (
    <div style={{
      flex: 1,
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      padding: 32,
      overflow: 'auto',
      animation: 'fadeIn 0.2s ease',
    }}>
      {/* Logo */}
      <div style={{
        width: 56,
        height: 56,
        borderRadius: 'var(--win-radius-lg)',
        background: 'linear-gradient(135deg, #0078D4, #60CDFF)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        marginBottom: 16,
        boxShadow: '0 4px 16px rgba(0, 120, 212, 0.25)',
      }}>
        <Icon name="disk" size={28} style={{ color: '#fff' }} />
      </div>

      {/* Title */}
      <h1 style={{
        fontSize: 22,
        fontWeight: 600,
        color: 'var(--win-text)',
        letterSpacing: '-0.01em',
        marginBottom: 4,
      }}>
        Universal Container Explorer
      </h1>
      <p style={{
        fontSize: 13,
        color: 'var(--win-text-secondary)',
        marginBottom: 24,
      }}>
        Open disk images, containers & forensic evidence
      </p>

      {/* Open button */}
      <button
        onClick={onOpen}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          padding: '8px 24px',
          borderRadius: 'var(--win-radius-sm)',
          background: 'var(--win-accent-default)',
          color: '#fff',
          fontSize: 13,
          fontWeight: 500,
          marginBottom: 28,
          transition: 'background 0.1s',
        }}
        onMouseEnter={e => (e.currentTarget.style.background = '#106EBE')}
        onMouseLeave={e => (e.currentTarget.style.background = 'var(--win-accent-default)')}
      >
        <Icon name="disk" size={15} />
        Open Evidence
      </button>

      {/* Recent Evidence */}
      {recentFiles.length > 0 && (
        <div style={{ marginBottom: 24, width: '100%', maxWidth: 420 }}>
          <div style={{
            fontSize: 11,
            fontWeight: 600,
            color: 'var(--win-text-secondary)',
            textTransform: 'uppercase',
            letterSpacing: '0.5px',
            marginBottom: 8,
            textAlign: 'center',
          }}>
            Recent Evidence
          </div>
          {recentFiles.slice(0, 5).map((f, i) => (
            <div
              key={i}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 10,
                padding: '6px 12px',
                borderRadius: 'var(--win-radius-sm)',
                cursor: 'pointer',
                transition: 'background 0.1s',
              }}
              onMouseEnter={e => (e.currentTarget.style.background = 'var(--win-subtle-hover)')}
              onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
            >
              <Icon name="disk" size={14} style={{ color: 'var(--win-text-tertiary)', flexShrink: 0 }} />
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: 12, color: 'var(--win-text)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {f.name}
                </div>
                <div style={{ fontSize: 10, color: 'var(--win-text-tertiary)' }}>
                  {formatSize(f.size)} {f.openedAt ? `· ${formatTimestamp(f.openedAt)}` : ''}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Supported formats */}
      <div style={{
        display: 'flex',
        gap: 6,
        flexWrap: 'wrap',
        justifyContent: 'center',
        marginBottom: 24,
        maxWidth: 400,
      }}>
        {FORMATS.map(f => (
          <div key={f.name} style={{
            padding: '4px 10px',
            borderRadius: 'var(--win-radius-sm)',
            background: 'var(--win-card)',
            border: '1px solid var(--win-stroke)',
            fontSize: 11,
            color: 'var(--win-text-secondary)',
            display: 'flex',
            alignItems: 'center',
            gap: 5,
            cursor: 'default',
            transition: 'all 0.1s',
          }}
          onMouseEnter={e => {
            e.currentTarget.style.borderColor = f.color;
            e.currentTarget.style.color = 'var(--win-text)';
          }}
          onMouseLeave={e => {
            e.currentTarget.style.borderColor = 'var(--win-stroke)';
            e.currentTarget.style.color = 'var(--win-text-secondary)';
          }}
          >
            <div style={{
              width: 6,
              height: 6,
              borderRadius: '50%',
              background: f.color,
              flexShrink: 0,
            }} />
            {f.name}
          </div>
        ))}
      </div>

      {/* Keyboard shortcuts */}
      <div style={{
        display: 'flex',
        gap: 20,
        fontSize: 11,
        color: 'var(--win-text-tertiary)',
      }}>
        <span><kbd style={kbd}>Ctrl</kbd>+<kbd style={kbd}>O</kbd> Open</span>
        <span><kbd style={kbd}>Ctrl</kbd>+<kbd style={kbd}>K</kbd> Command</span>
      </div>
    </div>
  );
}

const kbd: React.CSSProperties = {
  display: 'inline-block',
  padding: '1px 4px',
  borderRadius: 3,
  border: '1px solid var(--win-stroke-strong)',
  background: 'var(--win-card)',
  fontSize: 10,
  fontFamily: 'var(--win-font-mono)',
  lineHeight: '16px',
};

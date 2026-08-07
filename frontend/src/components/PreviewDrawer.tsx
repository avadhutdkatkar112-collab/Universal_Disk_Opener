import { useEffect, useState } from 'react';
import { Icon } from './Icon';
import { fmtSize } from '../lib/utils';

interface Props {
  filePath: string | null;
  onClose: () => void;
}

interface FileInfo {
  name: string;
  size: number;
  mime: string;
}

export function PreviewDrawer({ filePath, onClose }: Props) {
  const [content, setContent] = useState('');
  const [info, setInfo] = useState<FileInfo | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!filePath) { setContent(''); setInfo(null); return; }
    let cancelled = false;
    setLoading(true);
    setError(null);
    import('../lib/wails').then(({ Main }) => {
      Main.GetFilePreview(filePath)
        .then((result: any) => {
          if (cancelled) return;
          if (result.error) { setError(result.error); return; }
          setInfo({ name: filePath.split('/').pop() || '', size: result.totalSize || 0, mime: result.mimeType || 'application/octet-stream' });
          setContent(result.content || '');
        })
        .catch((err: unknown) => { if (!cancelled) setError(err instanceof Error ? err.message : 'Preview failed'); })
        .finally(() => { if (!cancelled) setLoading(false); });
    });
    return () => { cancelled = true; };
  }, [filePath]);

  if (!filePath) return null;
  const lines = content.split('\n');

  return (
    <div
      role="complementary"
      aria-label="File preview"
      style={{
        position: 'fixed', top: 32, right: 0, bottom: 24, width: 420,
        background: 'var(--win-surface)', borderLeft: '1px solid var(--win-stroke)',
        boxShadow: '-4px 0 16px rgba(0, 0, 0, 0.08)', display: 'flex',
        flexDirection: 'column', zIndex: 300, animation: 'slideInRight 0.15s ease',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '8px 12px', borderBottom: '1px solid var(--win-stroke)', flexShrink: 0 }}>
        <Icon name="preview" size={14} style={{ color: 'var(--win-accent)', flexShrink: 0 }} />
        <div style={{ flex: 1, overflow: 'hidden' }}>
          <div style={{ fontSize: 13, fontWeight: 500, color: 'var(--win-text)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {filePath}
          </div>
          {info && <div style={{ fontSize: 11, color: 'var(--win-text-tertiary)' }}>{fmtSize(info.size)} · {info.mime}</div>}
        </div>
        <button
          aria-label="Close preview"
          onClick={onClose}
          style={{
            display: 'flex', alignItems: 'center', justifyContent: 'center', width: 24, height: 24,
            borderRadius: 'var(--win-radius-sm)', flexShrink: 0,
          }}
          onMouseEnter={e => (e.currentTarget.style.background = 'var(--win-subtle-hover)')}
          onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
        >
          <Icon name="close" size={12} style={{ color: 'var(--win-text-tertiary)' }} />
        </button>
      </div>

      <div style={{ flex: 1, overflow: 'auto', background: 'var(--win-bg)' }}>
        {loading ? (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: 100, color: 'var(--win-text-tertiary)', fontSize: 13, gap: 8 }}>
            <div style={{ width: 14, height: 14, border: '2px solid var(--win-stroke-strong)', borderTopColor: 'var(--win-accent)', borderRadius: '50%', animation: 'spin 0.6s linear infinite' }} />
            Loading…
          </div>
        ) : error ? (
          <div style={{ padding: 20, textAlign: 'center', color: 'var(--win-danger)', fontSize: 13 }}>
            <Icon name="alert" size={16} style={{ display: 'block', margin: '0 auto 8px' }} />{error}
          </div>
        ) : (
          <div style={{ fontFamily: 'var(--win-font-mono)', fontSize: 12, lineHeight: '20px', padding: '4px 0' }}>
            {lines.map((line, i) => (
              <div key={i} style={{ display: 'flex', padding: '0 12px' }}>
                <span style={{ width: 40, textAlign: 'right', paddingRight: 12, color: 'var(--win-text-tertiary)', userSelect: 'none', flexShrink: 0, fontSize: 11 }}>
                  {i + 1}
                </span>
                <span style={{ color: 'var(--win-text-secondary)', whiteSpace: 'pre', overflow: 'hidden' }}>{line}</span>
              </div>
            ))}
          </div>
        )}
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '6px 12px', borderTop: '1px solid var(--win-stroke)', flexShrink: 0 }}>
        <button
          aria-label="Copy content"
          onClick={() => navigator.clipboard?.writeText(content)}
          style={{
            display: 'flex', alignItems: 'center', gap: 5, padding: '4px 10px',
            borderRadius: 'var(--win-radius-sm)', border: '1px solid var(--win-stroke)',
            background: 'var(--win-card)', fontSize: 12, color: 'var(--win-text-secondary)',
          }}
          onMouseEnter={e => { e.currentTarget.style.background = 'var(--win-control-hover)'; e.currentTarget.style.borderColor = 'var(--win-stroke-strong)'; }}
          onMouseLeave={e => { e.currentTarget.style.background = 'var(--win-card)'; e.currentTarget.style.borderColor = 'var(--win-stroke)'; }}
        >
          <Icon name="copy" size={12} /> Copy
        </button>
        <div style={{ flex: 1 }} />
        {info && <span style={{ fontSize: 11, color: 'var(--win-text-tertiary)' }}>{lines.length} lines</span>}
      </div>
    </div>
  );
}

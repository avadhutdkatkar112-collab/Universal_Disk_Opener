import { useState, useEffect, useMemo, useCallback, useRef } from 'react';
import { ReadFileChunk } from '../lib/wails';

interface HexViewerProps {
  filePath: string;
  fileSize: number;
  onClose: () => void;
}

interface HexLine {
  offset: string;
  bytes: string[];
  ascii: string;
}

export default function HexViewer({ filePath, fileSize, onClose }: HexViewerProps) {
  const [data, setData] = useState<Uint8Array | null>(null);
  const [offset, setOffset] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [searchOffset, setSearchOffset] = useState<number | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  const BYTES_PER_LINE = 16;
  const LINES_PER_PAGE = 32;
  const BYTES_PER_PAGE = BYTES_PER_LINE * LINES_PER_PAGE;

  const loadChunk = useCallback(async (pos: number) => {
    setLoading(true);
    setError(null);
    try {
      const chunk = await ReadFileChunk(filePath, pos, BYTES_PER_PAGE);
      setData(new Uint8Array(chunk));
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }, [filePath]);

  useEffect(() => {
    loadChunk(offset);
  }, [offset, loadChunk]);

  const hexLines = useMemo((): HexLine[] => {
    if (!data) return [];
    const lines: HexLine[] = [];
    
    for (let i = 0; i < data.length; i += BYTES_PER_LINE) {
      const chunk = data.slice(i, i + BYTES_PER_LINE);
      const bytes = Array.from(chunk).map(b => b.toString(16).padStart(2, '0'));
      const ascii = Array.from(chunk)
        .map(b => (b >= 32 && b < 127 ? String.fromCharCode(b) : '.'))
        .join('');
      
      lines.push({
        offset: (offset + i).toString(16).padStart(8, '0'),
        bytes,
        ascii,
      });
    }
    return lines;
  }, [data, offset]);

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'PageDown') {
      e.preventDefault();
      setOffset(prev => Math.min(prev + BYTES_PER_PAGE, fileSize - BYTES_PER_PAGE));
    } else if (e.key === 'PageUp') {
      e.preventDefault();
      setOffset(prev => Math.max(prev - BYTES_PER_PAGE, 0));
    } else if (e.key === 'Escape') {
      onClose();
    }
  }, [fileSize, onClose]);

  const handleSearch = useCallback(() => {
    if (!searchTerm || !data) return;
    const searchBytes = searchTerm.split(' ').map(s => parseInt(s, 16)).filter(b => !isNaN(b));
    if (searchBytes.length === 0) return;

    for (let i = 0; i < data.length - searchBytes.length; i++) {
      let match = true;
      for (let j = 0; j < searchBytes.length; j++) {
        if (data[i + j] !== searchBytes[j]) {
          match = false;
          break;
        }
      }
      if (match) {
        setSearchOffset(offset + i);
        return;
      }
    }
    setSearchOffset(null);
  }, [searchTerm, data, offset]);

  const goToOffset = useCallback((hexOffset: string) => {
    const pos = parseInt(hexOffset, 16);
    if (!isNaN(pos)) {
      setOffset(Math.max(0, Math.min(pos, fileSize - BYTES_PER_PAGE)));
    }
  }, [fileSize]);

  return (
    <div 
      style={{
        position: 'fixed', top: 32, right: 0, bottom: 24, width: 600,
        background: 'var(--win-surface)', borderLeft: '1px solid var(--win-stroke)',
        boxShadow: '-4px 0 16px rgba(0, 0, 0, 0.08)', display: 'flex', flexDirection: 'column',
        zIndex: 350, animation: 'slideInRight 0.15s ease',
      }}
      onKeyDown={handleKeyDown}
      tabIndex={0}
    >
      {/* Header */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 8, padding: '8px 12px',
        borderBottom: '1px solid var(--win-stroke)', flexShrink: 0,
      }}>
        <div style={{ flex: 1 }}>
          <div style={{ fontSize: 13, fontWeight: 500, color: 'var(--win-text)' }}>Hex Inspector</div>
          <div style={{ fontSize: 11, color: 'var(--win-text-tertiary)' }}>
            {filePath} ({(fileSize / 1024 / 1024).toFixed(1)} MB)
          </div>
        </div>
        <button onClick={onClose} style={{
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          width: 24, height: 24, borderRadius: 'var(--win-radius-sm)',
        }}
          onMouseEnter={e => (e.currentTarget.style.background = 'var(--win-subtle-hover)')}
          onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}>
          ✕
        </button>
      </div>

      {/* Navigation */}
      <div style={{
        display: 'flex', gap: 8, padding: '8px 12px',
        borderBottom: '1px solid var(--win-stroke)', flexShrink: 0,
      }}>
        <input
          type="text"
          placeholder="Go to offset (hex)..."
          style={{
            flex: 1, padding: '4px 8px', fontSize: 11,
            border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
            background: 'var(--win-bg)', color: 'var(--win-text)',
            fontFamily: 'var(--win-font-mono)',
          }}
          onKeyDown={e => {
            if (e.key === 'Enter') {
              goToOffset((e.target as HTMLInputElement).value);
            }
          }}
        />
        <input
          type="text"
          placeholder="Search hex (e.g., 4D 5A 90)"
          value={searchTerm}
          onChange={e => setSearchTerm(e.target.value)}
          style={{
            flex: 1, padding: '4px 8px', fontSize: 11,
            border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
            background: 'var(--win-bg)', color: 'var(--win-text)',
            fontFamily: 'var(--win-font-mono)',
          }}
          onKeyDown={e => {
            if (e.key === 'Enter') handleSearch();
          }}
        />
      </div>

      {/* Hex Content */}
      <div ref={containerRef} style={{
        flex: 1, overflow: 'auto', padding: '8px 12px',
        fontFamily: 'var(--win-font-mono)', fontSize: 11, lineHeight: '18px',
        background: 'var(--win-bg)',
      }}>
        {loading ? (
          <div style={{ textAlign: 'center', padding: 20, color: 'var(--win-text-tertiary)' }}>
            Loading...
          </div>
        ) : error ? (
          <div style={{ textAlign: 'center', padding: 20, color: 'var(--win-danger)' }}>
            Error: {error}
          </div>
        ) : (
          <>
            {/* Header */}
            <div style={{ 
              display: 'flex', gap: 0, color: 'var(--win-text-tertiary)',
              borderBottom: '1px solid var(--win-stroke)', paddingBottom: 4, marginBottom: 4,
            }}>
              <span style={{ width: 90 }}>Offset</span>
              <span style={{ flex: 1 }}>
                {Array.from({ length: 16 }, (_, i) => (
                  <span key={i} style={{ display: 'inline-block', width: 24, textAlign: 'center' }}>
                    {i.toString(16).toUpperCase().padStart(2, '0')}
                  </span>
                ))}
              </span>
              <span style={{ width: 140 }}>ASCII</span>
            </div>

            {/* Lines */}
            {hexLines.map((line, idx) => (
              <div key={idx} style={{
                display: 'flex', gap: 0,
                background: searchOffset !== null && 
                  parseInt(line.offset, 16) === searchOffset 
                  ? 'rgba(0, 120, 212, 0.2)' : 'transparent',
              }}>
                <span style={{ width: 90, color: 'var(--win-accent)' }}>{line.offset}</span>
                <span style={{ flex: 1 }}>
                  {line.bytes.map((byte, bIdx) => (
                    <span key={bIdx} style={{
                      display: 'inline-block', width: 24, textAlign: 'center',
                      color: byte === '00' ? 'var(--win-text-tertiary)' : 'var(--win-text)',
                    }}>
                      {byte}
                    </span>
                  ))}
                </span>
                <span style={{ width: 140, color: 'var(--win-text-secondary)' }}>
                  {line.ascii}
                </span>
              </div>
            ))}
          </>
        )}
      </div>

      {/* Footer Navigation */}
      <div style={{
        display: 'flex', justifyContent: 'space-between', alignItems: 'center',
        padding: '8px 12px', borderTop: '1px solid var(--win-stroke)',
        fontSize: 11, color: 'var(--win-text-tertiary)', flexShrink: 0,
      }}>
        <button 
          onClick={() => setOffset(0)}
          disabled={offset === 0}
          style={{
            padding: '4px 8px', fontSize: 11,
            border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
            background: 'var(--win-bg)', color: offset === 0 ? 'var(--win-text-disabled)' : 'var(--win-text)',
            cursor: offset === 0 ? 'not-allowed' : 'pointer',
          }}
        >
          First
        </button>
        <button 
          onClick={() => setOffset(prev => Math.max(0, prev - BYTES_PER_PAGE))}
          disabled={offset === 0}
          style={{
            padding: '4px 8px', fontSize: 11,
            border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
            background: 'var(--win-bg)', color: offset === 0 ? 'var(--win-text-disabled)' : 'var(--win-text)',
            cursor: offset === 0 ? 'not-allowed' : 'pointer',
          }}
        >
          ← Prev
        </button>
        <span>Offset: 0x{offset.toString(16).toUpperCase().padStart(8, '0')}</span>
        <button 
          onClick={() => setOffset(prev => Math.min(prev + BYTES_PER_PAGE, fileSize - BYTES_PER_PAGE))}
          disabled={offset >= fileSize - BYTES_PER_PAGE}
          style={{
            padding: '4px 8px', fontSize: 11,
            border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
            background: 'var(--win-bg)', 
            color: offset >= fileSize - BYTES_PER_PAGE ? 'var(--win-text-disabled)' : 'var(--win-text)',
            cursor: offset >= fileSize - BYTES_PER_PAGE ? 'not-allowed' : 'pointer',
          }}
        >
          Next →
        </button>
        <button 
          onClick={() => setOffset(Math.max(0, fileSize - BYTES_PER_PAGE))}
          disabled={offset >= fileSize - BYTES_PER_PAGE}
          style={{
            padding: '4px 8px', fontSize: 11,
            border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
            background: 'var(--win-bg)', 
            color: offset >= fileSize - BYTES_PER_PAGE ? 'var(--win-text-disabled)' : 'var(--win-text)',
            cursor: offset >= fileSize - BYTES_PER_PAGE ? 'not-allowed' : 'pointer',
          }}
        >
          Last
        </button>
      </div>
    </div>
  );
}

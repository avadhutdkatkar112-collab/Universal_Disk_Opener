import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { Icon } from './Icon';
import { useDiskStore } from '../store/diskStore';
import { fmtSize } from '../lib/utils';

interface Props {
  onPreview: (path: string) => void;
  onSelect?: (entry: DirEntry | null) => void;
  onSelectionChange?: (paths: string[]) => void;
  onHashSelected?: (paths: string[]) => void;
}

export interface DirEntry {
  name: string;
  isDir: boolean;
  size?: number;
  modified?: string;
  path?: string;
}

type SortKey = 'name' | 'size' | 'modified';
type SortDir = 'asc' | 'desc';

const FILE_COLORS: Record<string, string> = {
  pdf: '#E81123', doc: '#2B5797', docx: '#2B5797', xls: '#217346', xlsx: '#217346',
  ppt: '#D04423', pptx: '#D04423', txt: '#6E6E6E', log: '#6E6E6E', csv: '#217346',
  json: '#C19C00', xml: '#C19C00', yaml: '#C19C00', yml: '#C19C00',
  js: '#C19C00', ts: '#3178C6', py: '#3776AB', go: '#00ADD8', rs: '#DEA584',
  java: '#ED8B00', c: '#555', cpp: '#555', h: '#555',
  html: '#E34F26', css: '#1572B6', md: '#6E6E6E',
  sh: '#4EAA25', bat: '#4EAA25', ps1: '#012456',
  exe: '#6B4C9A', dll: '#6B4C9A', so: '#6B4C9A',
  img: '#7B7B7B', iso: '#C0392B', vhd: '#0078D4', vhdx: '#0078D4',
  vmdk: '#6CCB5F', qcow2: '#C19C00', vdi: '#9B59B6',
  zip: '#E67E22', '7z': '#E67E22', tar: '#7B7B7B', gz: '#7B7B7B',
  mp3: '#1DB954', mp4: '#1DB954', jpg: '#00A3E0', png: '#00A3E0', svg: '#FFB13B',
};

function getFileColor(name: string): string {
  const ext = name.split('.').pop()?.toLowerCase() || '';
  return FILE_COLORS[ext] || 'var(--win-text-tertiary)';
}

function getFileExt(name: string): string {
  return (name.split('.').pop()?.toUpperCase() || '').slice(0, 4);
}

export function ExplorerPanel({ onPreview, onSelect, onSelectionChange, onHashSelected }: Props) {
  const { disk } = useDiskStore();
  const [files, setFiles] = useState<DirEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [path, setPath] = useState('/');
  const [selected, setSelected] = useState<number | null>(null);
  const [sortKey, setSortKey] = useState<SortKey>('name');
  const [sortDir, setSortDir] = useState<SortDir>('asc');
  const [history, setHistory] = useState<string[]>(['/']);
  const [histIdx, setHistIdx] = useState(0);
  const [selectMode, setSelectMode] = useState(false);
  const [selectedPaths, setSelectedPaths] = useState<Set<string>>(new Set());
  const [lastSelectedIdx, setLastSelectedIdx] = useState<number | null>(null);
  const gridRef = useRef<HTMLDivElement>(null);

  const loadDir = useCallback(async (dirPath: string, pushHistory = true) => {
    setLoading(true);
    setError(null);
    setSelected(null);
    setSelectedPaths(new Set());
    onSelect?.(null);
    try {
      const { Main } = await import('../lib/wails');
      const result = await Main.ListDirectory(dirPath);
      if (result && result.error) {
        setError(result.error);
        setFiles([]);
      } else {
        setFiles(Array.isArray(result) ? result : []);
        setPath(dirPath);
        if (pushHistory) {
          setHistory(prev => {
            const newHist = prev.slice(0, histIdx + 1);
            newHist.push(dirPath);
            return newHist;
          });
          setHistIdx(prev => prev + 1);
        }
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to list directory');
      setFiles([]);
    } finally {
      setLoading(false);
    }
  }, [histIdx, onSelect]);

  useEffect(() => {
    if (disk) loadDir('/');
  }, [disk]);

  // Notify parent of selection changes
  useEffect(() => {
    onSelectionChange?.(Array.from(selectedPaths));
  }, [selectedPaths, onSelectionChange]);

  const navigate = useCallback((name: string, isDir: boolean) => {
    const next = path === '/' ? `/${name}` : `${path}/${name}`;
    if (isDir) loadDir(next);
    else onPreview(next);
  }, [path, loadDir, onPreview]);

  const goUp = useCallback(() => {
    if (path === '/') return;
    const parts = path.split('/').filter(Boolean);
    parts.pop();
    loadDir('/' + parts.join('/'));
  }, [path, loadDir]);

  const goBack = useCallback(() => {
    if (histIdx <= 0) return;
    const newPath = history[histIdx - 1];
    setHistIdx(histIdx - 1);
    loadDir(newPath, false);
  }, [histIdx, history, loadDir]);

  const goForward = useCallback(() => {
    if (histIdx >= history.length - 1) return;
    const newPath = history[histIdx + 1];
    setHistIdx(histIdx + 1);
    loadDir(newPath, false);
  }, [histIdx, history, loadDir]);

  const handleSort = useCallback((key: SortKey) => {
    if (sortKey === key) setSortDir(d => d === 'asc' ? 'desc' : 'asc');
    else { setSortKey(key); setSortDir('asc'); }
  }, [sortKey]);

  const sorted = useMemo(() => {
    return [...files].sort((a, b) => {
      if (a.isDir && !b.isDir) return -1;
      if (!a.isDir && b.isDir) return 1;
      let cmp = 0;
      if (sortKey === 'name') cmp = a.name.localeCompare(b.name);
      else if (sortKey === 'size') cmp = (a.size ?? 0) - (b.size ?? 0);
      else cmp = (a.modified || '').localeCompare(b.modified || '');
      return sortDir === 'asc' ? cmp : -cmp;
    });
  }, [files, sortKey, sortDir]);

  const segments = useMemo(() => path.split('/').filter(Boolean), [path]);

  // Selection helpers
  const getEntryPath = useCallback((entry: DirEntry): string => {
    if (entry.path) return entry.path;
    return path === '/' ? `/${entry.name}` : `${path}/${entry.name}`;
  }, [path]);

  const toggleSelectAll = useCallback(() => {
    if (selectedPaths.size === sorted.length) {
      setSelectedPaths(new Set());
    } else {
      setSelectedPaths(new Set(sorted.map(f => getEntryPath(f))));
    }
  }, [selectedPaths, sorted, getEntryPath]);

  const toggleItem = useCallback((idx: number, entry: DirEntry, extendSelect: boolean) => {
    const entryPath = getEntryPath(entry);
    
    if (extendSelect && lastSelectedIdx !== null) {
      // Range selection
      const start = Math.min(lastSelectedIdx, idx);
      const end = Math.max(lastSelectedIdx, idx);
      const rangePaths = sorted.slice(start, end + 1).map(f => getEntryPath(f));
      setSelectedPaths(prev => {
        const next = new Set(prev);
        rangePaths.forEach(p => next.add(p));
        return next;
      });
    } else {
      // Toggle single item
      setSelectedPaths(prev => {
        const next = new Set(prev);
        if (next.has(entryPath)) {
          next.delete(entryPath);
        } else {
          next.add(entryPath);
        }
        return next;
      });
    }
    setLastSelectedIdx(idx);
  }, [sorted, getEntryPath, lastSelectedIdx]);

  const clearSelection = useCallback(() => {
    setSelectedPaths(new Set());
    setSelectMode(false);
    setLastSelectedIdx(null);
  }, []);

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    const maxIdx = sorted.length - 1;
    
    if (e.key === 'Escape' && selectMode) {
      clearSelection();
      return;
    }
    
    if (e.key === ' ' && selectMode && selected != null && sorted[selected]) {
      e.preventDefault();
      toggleItem(selected, sorted[selected], false);
      return;
    }
    
    if (e.key === 'a' && (e.ctrlKey || e.metaKey) && selectMode) {
      e.preventDefault();
      toggleSelectAll();
      return;
    }
    
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setSelected(s => s == null ? 0 : Math.min(s + 1, maxIdx));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setSelected(s => s == null ? 0 : Math.max(s - 1, 0));
    } else if (e.key === 'Enter' && selected != null && sorted[selected]) {
      const entry = sorted[selected];
      if (selectMode) {
        toggleItem(selected, entry, false);
      } else {
        navigate(entry.name, entry.isDir);
      }
    } else if (e.key === 'Backspace') {
      goUp();
    }
  }, [sorted, selected, navigate, goUp, selectMode, toggleItem, clearSelection, toggleSelectAll]);

  const handleRowClick = useCallback((idx: number, entry: DirEntry, e: React.MouseEvent) => {
    if (selectMode) {
      toggleItem(idx, entry, e.shiftKey || e.ctrlKey || e.metaKey);
      onSelect?.(entry);
    } else {
      setSelected(idx);
      onSelect?.(entry);
    }
  }, [selectMode, toggleItem, onSelect]);

  const handleRowDoubleClick = useCallback((entry: DirEntry) => {
    if (!selectMode) {
      navigate(entry.name, entry.isDir);
    }
  }, [navigate, selectMode]);

  const isAllSelected = sorted.length > 0 && selectedPaths.size === sorted.length;

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      {/* Navigation bar */}
      <div style={{
        height: 32, display: 'flex', alignItems: 'center', gap: 4, padding: '0 8px',
        background: 'var(--win-surface)', borderBottom: '1px solid var(--win-stroke)', flexShrink: 0,
      }}>
        <NavBtn aria-label="Go back" disabled={histIdx <= 0} onClick={goBack}
          icon={<Icon name="chevron-right" size={11} style={{ transform: 'rotate(180deg)' }} />} />
        <NavBtn aria-label="Go forward" disabled={histIdx >= history.length - 1} onClick={goForward}
          icon={<Icon name="chevron-right" size={11} />} />
        <NavBtn aria-label="Go up" disabled={path === '/'} onClick={goUp}
          icon={<Icon name="chevron-right" size={11} style={{ transform: 'rotate(-90deg)' }} />} />
        <div style={{ width: 1, height: 14, background: 'var(--win-stroke-strong)', margin: '0 6px' }} />

        {/* Select Mode Toggle */}
        <button
          onClick={() => setSelectMode(!selectMode)}
          style={{
            display: 'flex', alignItems: 'center', gap: 4, padding: '2px 8px',
            fontSize: 11, borderRadius: 'var(--win-radius-sm)',
            background: selectMode ? 'var(--win-accent-default)' : 'transparent',
            color: selectMode ? '#fff' : 'var(--win-text-secondary)',
            border: selectMode ? 'none' : '1px solid var(--win-stroke)',
          }}
          onMouseEnter={e => {
            if (!selectMode) e.currentTarget.style.background = 'var(--win-subtle-hover)';
          }}
          onMouseLeave={e => {
            if (!selectMode) e.currentTarget.style.background = 'transparent';
          }}
        >
          <Icon name="check" size={10} />
          Select
        </button>

        {/* Breadcrumb */}
        <nav aria-label="Breadcrumb" style={{
          display: 'flex', alignItems: 'center', gap: 1, fontSize: 12,
          color: 'var(--win-text-secondary)', overflow: 'hidden', flex: 1,
        }}>
          <Crumb label="/" onClick={() => loadDir('/')} active={path === '/'} />
          {segments.map((seg, i) => (
            <span key={i} style={{ display: 'flex', alignItems: 'center' }}>
              {i > 0 && <span style={{ color: 'var(--win-text-tertiary)', margin: '0 1px' }}>/</span>}
              <Crumb label={seg} onClick={() => loadDir('/' + segments.slice(0, i + 1).join('/'))} active={i === segments.length - 1} />
            </span>
          ))}
        </nav>

        <span style={{ fontSize: 11, color: 'var(--win-text-tertiary)', flexShrink: 0 }}>
          {selectMode ? `${selectedPaths.size} selected` : `${files.length} items`}
        </span>
      </div>

      {/* Column headers */}
      <div role="row" style={{ display: 'flex', height: 26, borderBottom: '1px solid var(--win-stroke)', background: 'var(--win-bg)', flexShrink: 0 }}>
        {selectMode && (
          <div style={{ width: 30, flexShrink: 0, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Checkbox checked={isAllSelected} onChange={toggleSelectAll} />
          </div>
        )}
        <ColHeader flex={1} label="Name" col="name" sortKey={sortKey} sortDir={sortDir} onSort={handleSort} />
        <ColHeader width={90} label="Size" col="size" sortKey={sortKey} sortDir={sortDir} onSort={handleSort} />
        <ColHeader width={70} label="Type" />
        <ColHeader width={110} label="Modified" col="modified" sortKey={sortKey} sortDir={sortDir} onSort={handleSort} />
      </div>

      {/* File list */}
      <div
        ref={gridRef}
        role="grid"
        aria-label="File list"
        tabIndex={0}
        onKeyDown={handleKeyDown}
        style={{ flex: 1, overflow: 'auto', outline: 'none' }}
      >
        {loading ? (
          <CenterMsg><Spinner /><span>Loading…</span></CenterMsg>
        ) : error ? (
          <CenterMsg color="var(--win-danger)"><Icon name="alert" size={16} /><span>{error}</span></CenterMsg>
        ) : files.length === 0 ? (
          <CenterMsg>Empty directory</CenterMsg>
        ) : (
          <>
            {path !== '/' && (
              <FileRow
                entry={{ name: '..', isDir: true }}
                selected={false}
                selectMode={false}
                checked={false}
                onClick={() => goUp()}
                onDoubleClick={goUp}
              />
            )}
            {sorted.map((entry, i) => (
              <FileRow
                key={entry.name}
                entry={entry}
                selected={selected === i}
                selectMode={selectMode}
                checked={selectedPaths.has(getEntryPath(entry))}
                onClick={(e) => handleRowClick(i, entry, e)}
                onDoubleClick={() => handleRowDoubleClick(entry)}
              />
            ))}
          </>
        )}
      </div>

      {/* Batch Action Bar */}
      {selectMode && selectedPaths.size > 0 && (
        <BatchActionBar
          count={selectedPaths.size}
          onHash={() => onHashSelected?.(Array.from(selectedPaths))}
          onClear={clearSelection}
        />
      )}
    </div>
  );
}

// ── Batch Action Bar ─────────────────────────────────────────────────

function BatchActionBar({ count, onHash, onClear }: {
  count: number;
  onHash: () => void;
  onClear: () => void;
}) {
  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 8, padding: '6px 12px',
      background: 'var(--win-accent-default)', color: '#fff',
      borderTop: '1px solid var(--win-accent-hover)', flexShrink: 0,
    }}>
      <span style={{ fontSize: 12, fontWeight: 500 }}>{count} items selected</span>
      <div style={{ flex: 1 }} />
      <button onClick={onHash} style={{
        display: 'flex', alignItems: 'center', gap: 4, padding: '4px 12px',
        fontSize: 11, fontWeight: 600, borderRadius: 'var(--win-radius-sm)',
        background: '#fff', color: 'var(--win-accent-default)', border: 'none',
      }}
        onMouseEnter={e => (e.currentTarget.style.background = 'var(--win-subtle-hover)')}
        onMouseLeave={e => (e.currentTarget.style.background = '#fff')}>
        <Icon name="hash" size={12} />
        Hash Selected
      </button>
      <button onClick={onClear} style={{
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        width: 22, height: 22, borderRadius: 'var(--win-radius-sm)',
        color: '#fff', background: 'transparent',
      }}
        onMouseEnter={e => (e.currentTarget.style.background = 'rgba(255,255,255,0.2)')}
        onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}>
        <Icon name="close" size={12} />
      </button>
    </div>
  );
}

// ── Checkbox ─────────────────────────────────────────────────────────

function Checkbox({ checked, onChange }: { checked: boolean; onChange: () => void }) {
  return (
    <button
      role="checkbox"
      aria-checked={checked}
      onClick={(e) => { e.stopPropagation(); onChange(); }}
      style={{
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        width: 16, height: 16, borderRadius: 3,
        border: `1px solid ${checked ? 'var(--win-accent-default)' : 'var(--win-stroke-strong)'}`,
        background: checked ? 'var(--win-accent-default)' : 'transparent',
        cursor: 'pointer', padding: 0,
      }}
    >
      {checked && (
        <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
          <path d="M2 5L4 7L8 3" stroke="#fff" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      )}
    </button>
  );
}

// ── NavBtn ───────────────────────────────────────────────────────────

function NavBtn({ icon, disabled, onClick, 'aria-label': ariaLabel }: {
  icon: React.ReactNode; disabled?: boolean; onClick: () => void; 'aria-label': string;
}) {
  return (
    <button
      aria-label={ariaLabel}
      disabled={disabled}
      onClick={onClick}
      style={{
        display: 'flex', alignItems: 'center', justifyContent: 'center', width: 22, height: 22,
        borderRadius: 'var(--win-radius-sm)', opacity: disabled ? 0.3 : 1,
        transition: 'background 0.1s', flexShrink: 0, color: 'var(--win-text-secondary)',
      }}
      onMouseEnter={e => !disabled && (e.currentTarget.style.background = 'var(--win-subtle-hover)')}
      onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
    >
      {icon}
    </button>
  );
}

// ── Crumb ────────────────────────────────────────────────────────────

function Crumb({ label, onClick, active }: { label: string; onClick: () => void; active: boolean }) {
  return (
    <button
      role="link"
      aria-current={active ? 'page' : undefined}
      onClick={onClick}
      style={{
        cursor: 'pointer', padding: '1px 4px', borderRadius: 3, whiteSpace: 'nowrap',
        color: active ? 'var(--win-text)' : 'var(--win-text-secondary)',
        fontWeight: active ? 500 : 400,
      }}
      onMouseEnter={e => (e.currentTarget.style.background = 'var(--win-subtle-hover)')}
      onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
    >
      {label}
    </button>
  );
}

// ── ColHeader ────────────────────────────────────────────────────────

function ColHeader({ flex, width, label, col, sortKey, sortDir, onSort }: {
  flex?: number; width?: number; label: string; col?: SortKey;
  sortKey?: SortKey; sortDir?: SortDir; onSort?: (col: SortKey) => void;
}) {
  const active = col && sortKey === col;
  return (
    <div
      role="columnheader"
      aria-sort={active ? (sortDir === 'asc' ? 'ascending' : 'descending') : 'none'}
      onClick={() => col && onSort?.(col)}
      style={{
        flex, width, display: 'flex', alignItems: 'center', padding: '0 10px',
        fontSize: 11, fontWeight: 500,
        color: active ? 'var(--win-text)' : 'var(--win-text-tertiary)',
        cursor: col ? 'pointer' : 'default',
        borderRight: '1px solid var(--win-stroke)', userSelect: 'none',
      }}
      onMouseEnter={e => col && (e.currentTarget.style.color = 'var(--win-text-secondary)')}
      onMouseLeave={e => col && (e.currentTarget.style.color = active ? 'var(--win-text)' : 'var(--win-text-tertiary)')}
    >
      {label}
      {active && (
        <span style={{ marginLeft: 2, fontSize: 9, color: 'var(--win-accent)' }}>
          {sortDir === 'asc' ? '▲' : '▼'}
        </span>
      )}
    </div>
  );
}

// ── FileRow ──────────────────────────────────────────────────────────

interface FileRowProps {
  entry: DirEntry;
  selected: boolean;
  selectMode: boolean;
  checked: boolean;
  onClick: (e: React.MouseEvent) => void;
  onDoubleClick: () => void;
}

function FileRow({ entry, selected, selectMode, checked, onClick, onDoubleClick }: FileRowProps) {
  const isUp = entry.name === '..';
  const color = isUp ? 'var(--win-text-tertiary)' : getFileColor(entry.name);
  const ext = isUp ? '' : getFileExt(entry.name);

  return (
    <div
      role="row"
      aria-selected={selected}
      tabIndex={-1}
      onClick={onClick}
      onDoubleClick={onDoubleClick}
      style={{
        display: 'flex', height: 26, alignItems: 'center', padding: '0 10px', fontSize: 12,
        color: entry.isDir ? 'var(--win-text)' : 'var(--win-text-secondary)',
        background: selected ? 'rgba(0, 120, 212, 0.08)' : 'transparent',
        borderLeft: selected ? '2px solid var(--win-accent-default)' : '2px solid transparent',
        cursor: 'pointer',
      }}
      onMouseEnter={e => { if (!selected) e.currentTarget.style.background = 'var(--win-subtle-hover)'; }}
      onMouseLeave={e => { if (!selected) e.currentTarget.style.background = 'transparent'; }}
    >
      {selectMode && !isUp && (
        <div style={{ width: 30, flexShrink: 0, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Checkbox checked={checked} onChange={() => {}} />
        </div>
      )}
      <div role="gridcell" style={{ width: 20, flexShrink: 0, display: 'flex', justifyContent: 'center' }}>
        {isUp ? (
          <Icon name="folder" size={14} style={{ color: 'var(--win-text-tertiary)' }} />
        ) : entry.isDir ? (
          <Icon name="folder" size={14} style={{ color: 'var(--win-accent)' }} />
        ) : (
          <span style={{ fontSize: 9, fontWeight: 700, color, fontFamily: 'var(--win-font-mono)' }}>
            {ext || '·'}
          </span>
        )}
      </div>
      <div role="gridcell" style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', paddingLeft: 4 }}>
        {isUp ? '..' : entry.name}
      </div>
      <div role="gridcell" style={{ width: 90, flexShrink: 0, textAlign: 'right', fontSize: 11, color: 'var(--win-text-tertiary)', fontFamily: 'var(--win-font-mono)' }}>
        {isUp ? '' : entry.isDir ? '—' : entry.size != null ? fmtSize(entry.size) : ''}
      </div>
      <div role="gridcell" style={{ width: 70, flexShrink: 0, textAlign: 'left', fontSize: 10, color: 'var(--win-text-tertiary)', fontFamily: 'var(--win-font-mono)', textTransform: 'uppercase', paddingLeft: 6 }}>
        {isUp ? '' : entry.isDir ? 'DIR' : ext}
      </div>
      <div role="gridcell" style={{ width: 110, flexShrink: 0, textAlign: 'left', fontSize: 11, color: 'var(--win-text-tertiary)', paddingLeft: 6 }}>
        {isUp ? '' : entry.modified || ''}
      </div>
    </div>
  );
}

// ── CenterMsg ────────────────────────────────────────────────────────

function CenterMsg({ children, color }: { children: React.ReactNode; color?: string }) {
  return (
    <div style={{
      display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
      height: 100, gap: 6, color: color || 'var(--win-text-tertiary)', fontSize: 12,
    }}>
      {children}
    </div>
  );
}

// ── Spinner ──────────────────────────────────────────────────────────

function Spinner() {
  return (
    <div style={{
      width: 14, height: 14, border: '2px solid var(--win-stroke-strong)',
      borderTopColor: 'var(--win-accent)', borderRadius: '50%',
      animation: 'spin 0.6s linear infinite',
    }} />
  );
}

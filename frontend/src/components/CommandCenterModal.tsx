import { useState, useRef, useEffect, useMemo, useCallback } from 'react';
import { Icon } from './Icon';
import type { DiskSnapshot } from '../types';
import { useJobStore } from '../store/jobStore';

type IconName = 'disk' | 'folder' | 'file' | 'search' | 'extract' | 'analyze' | 'hash' | 'report' | 'settings' | 'preview' | 'bookmark' | 'close' | 'chevron-down' | 'chevron-right' | 'plus' | 'command' | 'terminal' | 'clock' | 'check' | 'alert' | 'copy' | 'open-external' | 'cancel' | 'lock' | 'info';

interface Props {
  open: boolean;
  onClose: () => void;
  disk: DiskSnapshot | null;
}

interface CmdDef {
  name: string;
  label: string;
  icon: IconName;
  args?: string;
  category: string;
}

const COMMANDS: CmdDef[] = [
  { name: 'disk.list', label: 'List Disks', icon: 'disk', category: 'System' },
  { name: 'disk.open', label: 'Open Disk', icon: 'disk', category: 'System' },
  { name: 'disk.info', label: 'Disk Info', icon: 'analyze', category: 'System' },
  { name: 'ext4.ls', label: 'List Directory', icon: 'folder', args: '/path', category: 'ext4' },
  { name: 'ext4.cat', label: 'Read File', icon: 'file', args: '/path', category: 'ext4' },
  { name: 'ext4.stat', label: 'File Info', icon: 'analyze', args: '/path', category: 'ext4' },
  { name: 'ext4.hash', label: 'Hash File', icon: 'hash', args: '/path', category: 'ext4' },
  { name: 'ext4.search', label: 'Search Files', icon: 'search', args: 'pattern', category: 'ext4' },
  { name: 'fat16.ls', label: 'List FAT16', icon: 'folder', category: 'FAT16' },
  { name: 'fat32.ls', label: 'List FAT32', icon: 'folder', category: 'FAT32' },
  { name: 'ntfs.ls', label: 'List NTFS', icon: 'folder', category: 'NTFS' },
  { name: 'workspace.mount', label: 'Mount Target', icon: 'disk', category: 'Workspace' },
  { name: 'workspace.navigate', label: 'Navigate', icon: 'folder', args: '/path', category: 'Workspace' },
  { name: 'workspace.bookmark', label: 'Add Bookmark', icon: 'bookmark', args: '/path', category: 'Workspace' },
];

export function CommandCenterModal({ open, onClose, disk }: Props) {
  const [query, setQuery] = useState('');
  const [selected, setSelected] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const submitJob = useJobStore(s => s.submitJob);

  const filtered = useMemo(() =>
    COMMANDS.filter(c =>
      c.name.toLowerCase().includes(query.toLowerCase()) ||
      c.label.toLowerCase().includes(query.toLowerCase()) ||
      c.category.toLowerCase().includes(query.toLowerCase())
    ), [query]);

  useEffect(() => {
    if (open) {
      setQuery('');
      setSelected(0);
      const t = setTimeout(() => inputRef.current?.focus(), 50);
      return () => clearTimeout(t);
    }
  }, [open]);

  useEffect(() => { setSelected(0); }, [query]);

  const execute = useCallback(async (cmd: CmdDef) => {
    const { Main } = await import('../lib/wails');
    submitJob(cmd.name, cmd.label);
    try {
      const result = await Main.ExecuteCommand(cmd.name, cmd.args ? { path: cmd.args } : {});
      if (result?.error) console.error('Command failed:', result.error);
    } catch (err) {
      console.error('Execute error:', err);
    }
    onClose();
  }, [onClose, submitJob]);

  const onKey = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') { e.preventDefault(); setSelected(s => Math.min(s + 1, filtered.length - 1)); }
    else if (e.key === 'ArrowUp') { e.preventDefault(); setSelected(s => Math.max(s - 1, 0)); }
    else if (e.key === 'Enter' && filtered[selected]) { execute(filtered[selected]); }
    else if (e.key === 'Escape') { onClose(); }
  }, [filtered, selected, execute, onClose]);

  if (!open) return null;

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label="Command Center"
      style={{
        position: 'fixed', inset: 0, zIndex: 1000, display: 'flex', justifyContent: 'center',
        paddingTop: '12vh', background: 'rgba(0, 0, 0, 0.3)', backdropFilter: 'blur(4px)',
        animation: 'fadeIn 0.1s ease',
      }}
      onClick={onClose}
    >
      <div
        style={{
          width: 480, maxHeight: 420, background: 'var(--win-surface)',
          border: '1px solid var(--win-stroke-strong)', borderRadius: 'var(--win-radius)',
          boxShadow: 'var(--win-shadow-flyout)', display: 'flex', flexDirection: 'column',
          overflow: 'hidden', animation: 'scaleIn 0.15s ease',
        }}
        onClick={e => e.stopPropagation()}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '10px 14px', borderBottom: '1px solid var(--win-stroke)' }}>
          <Icon name="terminal" size={16} style={{ color: 'var(--win-accent)', flexShrink: 0 }} />
          <input
            ref={inputRef}
            value={query}
            onChange={e => setQuery(e.target.value)}
            onKeyDown={onKey}
            placeholder="Type a command…"
            aria-label="Search commands"
            role="combobox"
            aria-expanded={true}
            aria-controls="command-list"
            aria-activedescendant={filtered[selected] ? `cmd-${filtered[selected].name}` : undefined}
            style={{ flex: 1, fontSize: 14, background: 'transparent', color: 'var(--win-text)' }}
          />
          <kbd style={kbd}>ESC</kbd>
        </div>

        {disk && (
          <div style={{ display: 'flex', gap: 6, padding: '6px 14px', borderBottom: '1px solid var(--win-stroke)' }}>
            <span style={pill}>
              <Icon name="disk" size={11} /> {disk.fileName}
            </span>
            {disk.partitions[disk.activePartition] && (
              <span style={{ ...pill, background: 'var(--win-subtle)', color: 'var(--win-text-secondary)' }}>
                P{disk.activePartition} {disk.partitions[disk.activePartition].fsType}
              </span>
            )}
          </div>
        )}

        <div ref={listRef} id="command-list" role="listbox" aria-label="Commands" style={{ flex: 1, overflow: 'auto', padding: '4px 0' }}>
          {filtered.length === 0 ? (
            <div style={{ padding: 20, textAlign: 'center', color: 'var(--win-text-tertiary)', fontSize: 13 }}>
              No matching commands
            </div>
          ) : (
            filtered.map((cmd, i) => (
              <div
                key={cmd.name}
                id={`cmd-${cmd.name}`}
                role="option"
                aria-selected={i === selected}
                onClick={() => execute(cmd)}
                onMouseEnter={() => setSelected(i)}
                style={{
                  display: 'flex', alignItems: 'center', gap: 10, padding: '6px 14px',
                  cursor: 'pointer', background: i === selected ? 'var(--win-accent-default)' : 'transparent',
                }}
              >
                <Icon name={cmd.icon} size={14}
                  style={{ color: i === selected ? '#fff' : 'var(--win-text-tertiary)', flexShrink: 0 }} />
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontSize: 13, color: i === selected ? '#fff' : 'var(--win-text)', fontWeight: i === selected ? 500 : 400 }}>
                    {cmd.label}
                  </div>
                  <div style={{ fontSize: 11, color: i === selected ? 'rgba(255,255,255,0.7)' : 'var(--win-text-tertiary)', fontFamily: 'var(--win-font-mono)' }}>
                    {cmd.name}
                  </div>
                </div>
                <span style={{
                  fontSize: 10, color: i === selected ? 'rgba(255,255,255,0.7)' : 'var(--win-text-tertiary)',
                  padding: '2px 6px', borderRadius: 3,
                  background: i === selected ? 'rgba(255,255,255,0.12)' : 'var(--win-subtle)',
                }}>
                  {cmd.category}
                </span>
              </div>
            ))
          )}
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: 14, padding: '6px 14px', borderTop: '1px solid var(--win-stroke)', fontSize: 11, color: 'var(--win-text-tertiary)' }}>
          <span><kbd style={kbd}>↑</kbd><kbd style={kbd}>↓</kbd> Navigate</span>
          <span><kbd style={kbd}>↵</kbd> Execute</span>
          <span><kbd style={kbd}>ESC</kbd> Close</span>
        </div>
      </div>
    </div>
  );
}

const kbd: React.CSSProperties = {
  display: 'inline-block', padding: '0 4px', borderRadius: 3,
  border: '1px solid var(--win-stroke-strong)', background: 'var(--win-card)',
  fontSize: 10, fontFamily: 'var(--win-font-mono)', lineHeight: '16px', marginRight: 2,
};

const pill: React.CSSProperties = {
  padding: '3px 8px', borderRadius: 4,
  background: 'rgba(0, 120, 212, 0.1)', color: 'var(--win-accent)',
  fontSize: 12, display: 'flex', alignItems: 'center', gap: 4,
};

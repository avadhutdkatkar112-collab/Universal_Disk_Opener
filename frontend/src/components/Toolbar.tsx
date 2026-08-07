import { Icon } from './Icon';
import type { DiskSnapshot } from '../types';

interface Props {
  disk: DiskSnapshot | null;
  onOpen: () => void;
  onCommandCenter: () => void;
  onHash?: () => void;
}

interface ToolbarItem {
  icon: any;
  label: string;
  action?: () => void;
  accent?: boolean;
  disabled?: boolean;
}

export function Toolbar({ disk, onOpen, onCommandCenter, onHash }: Props) {
  const items: ToolbarItem[] = disk
    ? [
        { icon: 'folder' as const, label: 'Browse' },
        { icon: 'search' as const, label: 'Search' },
        { icon: 'analyze' as const, label: 'Analyze' },
        { icon: 'extract' as const, label: 'Extract' },
        { icon: 'hash' as const, label: 'Hash', action: onHash },
        { icon: 'info' as const, label: 'Properties' },
      ]
    : [
        { icon: 'disk' as const, label: 'Open', action: onOpen, accent: true },
        { icon: 'terminal' as const, label: 'Command Center', action: onCommandCenter },
      ];

  return (
    <div style={{
      display: 'flex',
      alignItems: 'center',
      height: 40,
      padding: '0 8px',
      gap: 2,
      background: 'var(--win-surface)',
      borderBottom: '1px solid var(--win-stroke)',
      flexShrink: 0,
    }}>
      {items.map((item, i) => (
        <button
          key={i}
          onClick={item.action}
          disabled={item.disabled}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 5,
            padding: '5px 10px',
            borderRadius: 'var(--win-radius-sm)',
            fontSize: 12,
            color: item.accent ? '#fff' : item.disabled ? 'var(--win-text-disabled)' : 'var(--win-text-secondary)',
            background: item.accent ? 'var(--win-accent-default)' : 'transparent',
            opacity: item.disabled ? 0.5 : 1,
            cursor: item.disabled ? 'default' : 'pointer',
            transition: 'all 0.1s',
            flexShrink: 0,
          }}
          onMouseEnter={e => {
            if (!item.disabled && !item.accent) e.currentTarget.style.background = 'var(--win-subtle-hover)';
          }}
          onMouseLeave={e => {
            if (!item.accent) e.currentTarget.style.background = 'transparent';
          }}
        >
          <Icon
            name={item.icon}
            size={14}
            style={{ color: item.accent ? '#fff' : 'var(--win-text-tertiary)' }}
          />
          <span>{item.label}</span>
        </button>
      ))}

      <div style={{ flex: 1 }} />

      <button
        onClick={onCommandCenter}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 5,
          padding: '5px 10px',
          borderRadius: 'var(--win-radius-sm)',
          border: '1px solid var(--win-stroke)',
          fontSize: 12,
          color: 'var(--win-text-tertiary)',
          transition: 'all 0.1s',
          flexShrink: 0,
        }}
        onMouseEnter={e => {
          e.currentTarget.style.background = 'var(--win-subtle-hover)';
          e.currentTarget.style.borderColor = 'var(--win-stroke-strong)';
          e.currentTarget.style.color = 'var(--win-text-secondary)';
        }}
        onMouseLeave={e => {
          e.currentTarget.style.background = 'transparent';
          e.currentTarget.style.borderColor = 'var(--win-stroke)';
          e.currentTarget.style.color = 'var(--win-text-tertiary)';
        }}
      >
        <Icon name="terminal" size={13} />
        <span>Command Center</span>
        <kbd style={{
          padding: '0 4px',
          borderRadius: 3,
          border: '1px solid var(--win-stroke-strong)',
          background: 'var(--win-bg)',
          fontSize: 10,
          fontFamily: 'var(--win-font-mono)',
          lineHeight: '16px',
          marginLeft: 4,
        }}>Ctrl+K</kbd>
      </button>
    </div>
  );
}

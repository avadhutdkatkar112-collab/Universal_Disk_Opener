import { useState, useEffect, useRef, useCallback } from 'react';
import { Icon } from './Icon';

interface MenuDef {
  label: string;
  items: MenuItemDef[];
}

interface MenuItemDef {
  label?: string;
  icon?: string;
  shortcut?: string;
  action?: () => void;
  divider?: boolean;
  disabled?: boolean;
}

interface Props {
  onOpen: () => void;
  onCommandCenter: () => void;
  onClose?: () => void;
  onHash?: () => void;
}

export function MenuBar({ onOpen, onCommandCenter, onClose, onHash }: Props) {
  const [openMenu, setOpenMenu] = useState<string | null>(null);
  const [hovering, setHovering] = useState(false);
  const barRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (barRef.current && !barRef.current.contains(e.target as Node)) {
        setOpenMenu(null);
        setHovering(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  const menus: MenuDef[] = [
    {
      label: 'File',
      items: [
        { label: 'Open Evidence...', icon: 'disk', shortcut: 'Ctrl+O', action: onOpen },
        { label: 'Open Recent', icon: 'clock', disabled: true },
        { divider: true },
        { label: 'Close Evidence', icon: 'close', shortcut: 'Ctrl+W', action: onClose, disabled: !onClose },
        { divider: true },
        { label: 'Export...', icon: 'extract', disabled: true },
        { divider: true },
        { label: 'Exit', icon: 'cancel', shortcut: 'Alt+F4', disabled: true },
      ],
    },
    {
      label: 'View',
      items: [
        { label: 'Explorer', icon: 'folder' },
        { label: 'Investigate', icon: 'search' },
        { label: 'Examine', icon: 'preview' },
        { label: 'Timeline', icon: 'clock' },
        { label: 'Case', icon: 'report' },
      ],
    },
    {
      label: 'Actions',
      items: [
        { label: 'Calculate Hash', icon: 'hash', shortcut: 'Ctrl+H', action: onHash, disabled: !onHash },
        { divider: true },
        { label: 'Search', icon: 'search', disabled: true },
        { label: 'Extract File', icon: 'extract', disabled: true },
        { label: 'Bookmark', icon: 'bookmark', disabled: true },
      ],
    },
    {
      label: 'Tools',
      items: [
        { label: 'Command Center', icon: 'terminal', shortcut: 'Ctrl+K', action: onCommandCenter },
        { divider: true },
        { label: 'Hex Viewer', icon: 'preview', disabled: true },
        { label: 'Sigma Scanner', icon: 'analyze', disabled: true },
        { label: 'YARA Scanner', icon: 'analyze', disabled: true },
        { divider: true },
        { label: 'Settings', icon: 'settings', disabled: true },
      ],
    },
    {
      label: 'Help',
      items: [
        { label: 'Documentation', icon: 'info', disabled: true },
        { label: 'Keyboard Shortcuts', icon: 'command', disabled: true },
        { divider: true },
        { label: 'About Universal Container Explorer', icon: 'info', disabled: true },
      ],
    },
  ];

  const handleMenuClick = useCallback((label: string) => {
    if (openMenu === label) { setOpenMenu(null); setHovering(false); }
    else { setOpenMenu(label); setHovering(true); }
  }, [openMenu]);

  const handleMenuHover = useCallback((label: string) => {
    if (hovering) setOpenMenu(label);
  }, [hovering]);

  return (
    <div
      ref={barRef}
      role="menubar"
      aria-label="Application menu"
      style={{
        display: 'flex', alignItems: 'center', height: 28, background: 'var(--win-surface)',
        borderBottom: '1px solid var(--win-stroke)', flexShrink: 0, position: 'relative',
        zIndex: 100,
      }}
    >
      {menus.map(menu => (
        <div key={menu.label} style={{ position: 'relative' }}>
          <button
            role="menuitem"
            aria-haspopup="true"
            aria-expanded={openMenu === menu.label}
            onClick={() => handleMenuClick(menu.label)}
            onMouseEnter={() => handleMenuHover(menu.label)}
            style={{
              padding: '4px 10px', fontSize: 12, height: 28,
              color: openMenu === menu.label ? 'var(--win-text)' : 'var(--win-text-secondary)',
              background: openMenu === menu.label ? 'var(--win-subtle-hover)' : 'transparent',
              borderRadius: openMenu === menu.label ? 'var(--win-radius-sm)' : 0,
            }}
          >
            {menu.label}
          </button>

          {openMenu === menu.label && (
            <div
              role="menu"
              aria-label={`${menu.label} menu`}
              style={{
                position: 'absolute', top: 28, left: 0, minWidth: 220,
                background: 'var(--win-surface)', border: '1px solid var(--win-stroke-strong)',
                borderRadius: 'var(--win-radius)', boxShadow: 'var(--win-shadow-flyout)',
                padding: '4px 0', zIndex: 200, animation: 'scaleIn 0.08s ease',
              }}
            >
              {menu.items.map((item, i) =>
                item.divider ? (
                  <div key={i} role="separator" style={{ height: 1, background: 'var(--win-stroke)', margin: '4px 0' }} />
                ) : (
                  <button
                    key={i}
                    role="menuitem"
                    aria-disabled={item.disabled}
                    onClick={() => {
                      if (!item.disabled && item.action) item.action();
                      setOpenMenu(null);
                      setHovering(false);
                    }}
                    style={{
                      display: 'flex', alignItems: 'center', gap: 8, padding: '6px 12px',
                      width: '100%', fontSize: 12,
                      color: item.disabled ? 'var(--win-text-disabled)' : 'var(--win-text)',
                      opacity: item.disabled ? 0.5 : 1,
                      cursor: item.disabled ? 'default' : 'pointer',
                    }}
                    onMouseEnter={e => !item.disabled && (e.currentTarget.style.background = 'var(--win-subtle-hover)')}
                    onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                  >
                    {item.icon && <Icon name={item.icon as any} size={14} style={{ color: 'var(--win-text-tertiary)', flexShrink: 0 }} />}
                    {!item.icon && <span style={{ width: 14 }} />}
                    <span style={{ flex: 1, textAlign: 'left' }}>{item.label}</span>
                    {item.shortcut && (
                      <span style={{ color: 'var(--win-text-tertiary)', fontSize: 11, fontFamily: 'var(--win-font-mono)' }}>
                        {item.shortcut}
                      </span>
                    )}
                  </button>
                )
              )}
            </div>
          )}
        </div>
      ))}
    </div>
  );
}

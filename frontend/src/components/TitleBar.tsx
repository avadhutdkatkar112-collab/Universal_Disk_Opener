import { useState, useCallback } from 'react';

export function TitleBar() {
  const [maximized, setMaximized] = useState(false);

  const handleMinimize = useCallback(() => {
    window.runtime?.WindowMinimise();
  }, []);

  const handleMaximize = useCallback(async () => {
    window.runtime?.WindowToggleMaximise();
    try {
      const isMax = await window.runtime?.WindowIsMaximised();
      setMaximized(!!isMax);
    } catch {}
  }, []);

  const handleClose = useCallback(() => {
    window.runtime?.WindowClose();
  }, []);

  return (
    <div
      data-wails-drag
      style={{
        height: 32,
        display: 'flex',
        alignItems: 'center',
        background: 'var(--win-surface)',
        flexShrink: 0,
        userSelect: 'none',
        WebkitAppRegion: 'drag',
      } as React.CSSProperties}
    >
      <div style={{ flex: 1 }} />

      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          height: '100%',
          WebkitAppRegion: 'no-drag',
        } as React.CSSProperties}
      >
        {/* Minimize */}
        <button
          onClick={handleMinimize}
          style={{
            width: 46,
            height: '100%',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            transition: 'background 0.1s',
            color: 'var(--win-text)',
          }}
          onMouseEnter={e => (e.currentTarget.style.background = 'var(--win-subtle-hover)')}
          onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
        >
          <svg width="10" height="1" viewBox="0 0 10 1" fill="none">
            <rect width="10" height="1" fill="currentColor" />
          </svg>
        </button>

        {/* Maximize */}
        <button
          onClick={handleMaximize}
          style={{
            width: 46,
            height: '100%',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            transition: 'background 0.1s',
            color: 'var(--win-text)',
          }}
          onMouseEnter={e => (e.currentTarget.style.background = 'var(--win-subtle-hover)')}
          onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
        >
          {maximized ? (
            <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
              <rect x="2.5" y="0" width="7.5" height="7.5" rx="1" stroke="currentColor" strokeWidth="1" fill="none" />
              <rect x="0" y="2.5" width="7.5" height="7.5" rx="1" stroke="currentColor" strokeWidth="1" fill="var(--win-surface)" />
            </svg>
          ) : (
            <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
              <rect x="0.5" y="0.5" width="9" height="9" rx="1" stroke="currentColor" strokeWidth="1" fill="none" />
            </svg>
          )}
        </button>

        {/* Close */}
        <button
          onClick={handleClose}
          style={{
            width: 46,
            height: '100%',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            transition: 'background 0.1s',
            color: 'var(--win-text)',
          }}
          onMouseEnter={e => {
            e.currentTarget.style.background = '#C42B1C';
            e.currentTarget.style.color = '#fff';
          }}
          onMouseLeave={e => {
            e.currentTarget.style.background = 'transparent';
            e.currentTarget.style.color = 'var(--win-text)';
          }}
        >
          <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
            <path d="M1 1L9 9M9 1L1 9" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
          </svg>
        </button>
      </div>
    </div>
  );
}

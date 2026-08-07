import React, { useState } from 'react';
import { useEvidenceStore } from '../store/evidenceStore';

export const Sidebar: React.FC = () => {
  const session = useEvidenceStore();
  const [activeTab, setActiveTab] = useState<'evidence' | 'bookmarks' | 'notes'>('evidence');

  const formatSize = (bytes: number) => {
    if (bytes === 0) return '-';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
  };

  return (
    <div style={{ width: '240px', minWidth: '240px', backgroundColor: 'var(--win-surface)', borderRight: '1px solid var(--win-stroke)', display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div style={{ display: 'flex', borderBottom: '1px solid var(--win-stroke)' }}>
        {(['evidence', 'bookmarks', 'notes'] as const).map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            style={{
              flex: 1,
              padding: '8px 4px',
              fontSize: '11px',
              fontWeight: activeTab === tab ? '600' : '400',
              color: activeTab === tab ? 'var(--win-accent)' : 'var(--win-text-secondary)',
              backgroundColor: 'transparent',
              border: 'none',
              borderBottom: activeTab === tab ? '2px solid var(--win-accent)' : '2px solid transparent',
              cursor: 'pointer',
              textTransform: 'capitalize',
            }}
          >
            {tab}
          </button>
        ))}
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: '8px' }}>
        {activeTab === 'evidence' && (
          <div>
            {!session.isActive ? (
              <div style={{ fontSize: '12px', color: 'var(--win-text-secondary)', padding: '12px 4px', textAlign: 'center' }}>
                No evidence loaded
              </div>
            ) : (
              <>
                <div style={{ marginBottom: '12px', padding: '8px', backgroundColor: 'var(--win-bg)', borderRadius: 'var(--win-radius-sm)', border: '1px solid var(--win-stroke)' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '6px', marginBottom: '4px' }}>
                    <span style={{ fontSize: '12px', color: session.isReadOnly ? '#0F7B0F' : '#C42B1C' }}>
                      {session.isReadOnly ? '\uD83D\uDD12' : '\uD83D\uDD13'}
                    </span>
                    <span style={{ fontSize: '12px', fontWeight: '600', color: 'var(--win-text)' }}>
                      {session.fileName}
                    </span>
                  </div>
                  <div style={{ fontSize: '10px', color: 'var(--win-text-secondary)' }}>
                    {session.format} | {formatSize(session.totalSize)} | {session.isReadOnly ? 'Read-Only' : 'Read-Write'}
                  </div>
                  {session.sha256 && (
                    <div style={{ fontSize: '9px', color: 'var(--win-text-secondary)', fontFamily: 'var(--win-font-mono)', marginTop: '4px', wordBreak: 'break-all' }}>
                      SHA-256: {session.sha256.slice(0, 32)}...
                    </div>
                  )}
                </div>

                {session.partitions.length > 0 && (
                  <div>
                    <div style={{ fontSize: '10px', fontWeight: '600', color: 'var(--win-text-secondary)', textTransform: 'uppercase', letterSpacing: '0.5px', padding: '4px 0', marginBottom: '4px' }}>
                      Partitions
                    </div>
                    {session.partitions.map((part) => (
                      <div key={part.index}>
                        <div
                          onClick={() => session.selectPartition(part.index)}
                          style={{
                            padding: '6px 8px',
                            fontSize: '12px',
                            cursor: 'pointer',
                            borderRadius: 'var(--win-radius-sm)',
                            backgroundColor: session.selectedPartition === part.index ? 'var(--win-accent)' : 'transparent',
                            color: session.selectedPartition === part.index ? '#fff' : 'var(--win-text)',
                            display: 'flex',
                            alignItems: 'center',
                            gap: '6px',
                            marginBottom: '2px',
                          }}
                        >
                          <span style={{ fontSize: '10px' }}>
                            {session.selectedPartition === part.index ? '\u25BC' : '\u25B6'}
                          </span>
                          <span style={{ fontWeight: '500' }}>#{part.index}</span>
                          <span style={{ fontSize: '10px', color: session.selectedPartition === part.index ? 'rgba(255,255,255,0.8)' : 'var(--win-text-secondary)' }}>
                            {part.type}
                          </span>
                          <span style={{ marginLeft: 'auto', fontSize: '10px', color: session.selectedPartition === part.index ? 'rgba(255,255,255,0.7)' : 'var(--win-text-secondary)' }}>
                            {formatSize(part.size)}
                          </span>
                        </div>
                      </div>
                    ))}
                  </div>
                )}

                {session.filesystemName && (
                  <div style={{ marginTop: '12px' }}>
                    <div style={{ fontSize: '10px', fontWeight: '600', color: 'var(--win-text-secondary)', textTransform: 'uppercase', letterSpacing: '0.5px', padding: '4px 0', marginBottom: '4px' }}>
                      {session.filesystemName}
                    </div>
                    <div style={{ fontSize: '11px', color: 'var(--win-text-secondary)', padding: '4px 0' }}>
                      Path: <code style={{ fontSize: '10px' }}>{session.currentPath}</code>
                    </div>
                  </div>
                )}
              </>
            )}
          </div>
        )}

        {activeTab === 'bookmarks' && (
          <div>
            {session.bookmarks.length === 0 ? (
              <div style={{ fontSize: '12px', color: 'var(--win-text-secondary)', padding: '12px 4px', textAlign: 'center' }}>
                No bookmarks yet. Right-click files to bookmark.
              </div>
            ) : (
              session.bookmarks.map((bm) => (
                <div
                  key={bm.id}
                  style={{ padding: '6px 8px', fontSize: '12px', borderBottom: '1px solid var(--win-stroke)', cursor: 'pointer' }}
                  onClick={() => session.navigateTo(bm.path)}
                >
                  <div style={{ fontWeight: '500', color: 'var(--win-text)' }}>{bm.name}</div>
                  <div style={{ fontSize: '10px', color: 'var(--win-text-secondary)' }}>{bm.path}</div>
                  {bm.note && <div style={{ fontSize: '10px', color: 'var(--win-accent)', marginTop: '2px' }}>{bm.note}</div>}
                </div>
              ))
            )}
          </div>
        )}

        {activeTab === 'notes' && (
          <div style={{ fontSize: '12px', color: 'var(--win-text-secondary)', padding: '12px 4px', textAlign: 'center' }}>
            Notes feature coming soon.
          </div>
        )}
      </div>

      {session.isAnalyzing && (
        <div style={{ padding: '8px', borderTop: '1px solid var(--win-stroke)', backgroundColor: 'var(--win-bg)', fontSize: '11px', color: 'var(--win-accent)' }}>
          {session.analysisProgress}
        </div>
      )}
    </div>
  );
};

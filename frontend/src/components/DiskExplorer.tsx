import React, { useState } from 'react';
import { useEvidenceStore } from '../store/evidenceStore';
import { Inspector } from './Inspector';

export const DiskExplorer: React.FC = () => {
  const session = useEvidenceStore();
  const [selectedFile, setSelectedFile] = useState<any>(null);

  const formatSize = (bytes: number) => {
    if (bytes === 0) return '-';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
  };

  if (!session.isActive) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'var(--win-text-secondary)', fontSize: '13px' }}>
        No evidence loaded.
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', height: '100%', backgroundColor: 'var(--win-bg)', color: 'var(--win-text)', fontFamily: 'var(--win-font)' }}>
      {/* File listing */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        {/* Navigation toolbar */}
        <div style={{ padding: '6px 16px', borderBottom: '1px solid var(--win-stroke)', display: 'flex', gap: '4px', alignItems: 'center' }}>
          <button onClick={() => session.navigateBack()} disabled={session.historyIndex <= 0}
            style={{ padding: '4px 8px', backgroundColor: 'var(--win-surface)', border: '1px solid var(--win-stroke-strong)', borderRadius: 'var(--win-radius-sm)', cursor: session.historyIndex <= 0 ? 'default' : 'pointer', fontSize: '14px', opacity: session.historyIndex <= 0 ? 0.4 : 1 }}>
            &#8592;
          </button>
          <button onClick={() => session.navigateForward()} disabled={session.historyIndex >= session.pathHistory.length - 1}
            style={{ padding: '4px 8px', backgroundColor: 'var(--win-surface)', border: '1px solid var(--win-stroke-strong)', borderRadius: 'var(--win-radius-sm)', cursor: session.historyIndex >= session.pathHistory.length - 1 ? 'default' : 'pointer', fontSize: '14px', opacity: session.historyIndex >= session.pathHistory.length - 1 ? 0.4 : 1 }}>
            &#8594;
          </button>
          <button onClick={() => session.navigateUp()} disabled={session.currentPath === '/'}
            style={{ padding: '4px 8px', backgroundColor: 'var(--win-surface)', border: '1px solid var(--win-stroke-strong)', borderRadius: 'var(--win-radius-sm)', cursor: session.currentPath === '/' ? 'default' : 'pointer', fontSize: '14px', opacity: session.currentPath === '/' ? 0.4 : 1 }}>
            &#8593;
          </button>
          <span style={{ marginLeft: '8px', fontSize: '12px', fontFamily: 'var(--win-font-mono)', color: 'var(--win-text-secondary)' }}>
            {session.currentPath}
          </span>
        </div>

        {/* File listing */}
        <div style={{ flex: 1, overflow: 'auto', padding: '0' }}>
          {session.isAnalyzing ? (
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'var(--win-accent)', fontSize: '13px', gap: 8 }}>
              <div style={{ width: 14, height: 14, border: '2px solid var(--win-stroke-strong)', borderTopColor: 'var(--win-accent)', borderRadius: '50%', animation: 'spin 0.6s linear infinite' }} />
              {session.analysisProgress}
            </div>
          ) : session.currentNodes.length === 0 ? (
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'var(--win-text-secondary)', fontSize: '13px' }}>
              Directory is empty.
            </div>
          ) : (
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
              <thead>
                <tr style={{ color: 'var(--win-text-secondary)', borderBottom: '1px solid var(--win-stroke)', backgroundColor: 'var(--win-surface)' }}>
                  <th style={{ textAlign: 'left', padding: '8px 12px', fontWeight: '600' }}>Name</th>
                  <th style={{ textAlign: 'left', padding: '8px 12px', fontWeight: '600' }}>Type</th>
                  <th style={{ textAlign: 'right', padding: '8px 12px', fontWeight: '600' }}>Size</th>
                </tr>
              </thead>
              <tbody>
                {session.currentNodes.map((node, idx) => (
                  <tr
                    key={idx}
                    onClick={() => {
                      setSelectedFile(node);
                      if (node.isDir) {
                        session.navigateTo(node.path);
                      } else {
                        session.openFileForExamine(node.path);
                      }
                    }}
                    style={{
                      cursor: 'pointer',
                      borderBottom: '1px solid var(--win-stroke)',
                      backgroundColor: selectedFile?.path === node.path ? 'rgba(0,120,212,0.08)' : idx % 2 === 0 ? 'transparent' : 'rgba(0,0,0,0.02)',
                    }}
                  >
                    <td style={{ padding: '6px 12px', color: node.isDir ? 'var(--win-accent)' : 'var(--win-text)' }}>
                      {node.isDir ? '\uD83D\uDCC1' : '\uD83D\uDCC4'} {node.name}
                    </td>
                    <td style={{ padding: '6px 12px', color: 'var(--win-text-secondary)' }}>{node.isDir ? 'Directory' : 'File'}</td>
                    <td style={{ padding: '6px 12px', textAlign: 'right', color: 'var(--win-text-secondary)' }}>{formatSize(node.size)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>

      {/* Inspector side panel */}
      <Inspector selectedEntry={selectedFile} />
    </div>
  );
};

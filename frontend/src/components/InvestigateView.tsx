import React, { useState, useEffect } from 'react';
import { useEvidenceStore } from '../store/evidenceStore';

export const InvestigateView: React.FC = () => {
  const session = useEvidenceStore();
  const [scanComplete, setScanComplete] = useState(false);
  const [deepArtifacts, setDeepArtifacts] = useState<any[]>([]);

  useEffect(() => {
    if (session.isActive && session.currentNodes.length > 0 && !scanComplete) {
      performDeepScan();
    }
  }, [session.isActive, session.currentNodes.length]);

  const performDeepScan = async () => {
    setScanComplete(false);
    const w = window as any;

    const discovered: any[] = [];

    const windowsPaths = [
      { name: 'SYSTEM', type: 'registry', searchPath: '/Windows/System32/config/SYSTEM' },
      { name: 'SOFTWARE', type: 'registry', searchPath: '/Windows/System32/config/SOFTWARE' },
      { name: 'SAM', type: 'registry', searchPath: '/Windows/System32/config/SAM' },
      { name: 'SECURITY', type: 'registry', searchPath: '/Windows/System32/config/SECURITY' },
      { name: 'NTUSER.DAT', type: 'registry', searchPath: '/Users' },
      { name: 'Security.evtx', type: 'evtx', searchPath: '/Windows/System32/winevt/Logs/Security.evtx' },
      { name: 'System.evtx', type: 'evtx', searchPath: '/Windows/System32/winevt/Logs/System.evtx' },
      { name: 'Application.evtx', type: 'evtx', searchPath: '/Windows/System32/winevt/Logs/Application.evtx' },
      { name: 'PowerShell.evtx', type: 'evtx', searchPath: '/Windows/System32/winevt/Logs/Microsoft-Windows-PowerShell%4Operational.evtx' },
      { name: '$MFT', type: 'mft', searchPath: '/$MFT' },
      { name: '$LogFile', type: 'other', searchPath: '/$LogFile' },
      { name: '$UsnJrnl', type: 'other', searchPath: '/$UsnJrnl' },
      { name: 'Prefetch', type: 'prefetch', searchPath: '/Windows/Prefetch' },
      { name: 'Amcache.hve', type: 'registry', searchPath: '/Windows/appcompat/Programs/Amcache.hve' },
      { name: 'Shellbags', type: 'shellbag', searchPath: '/Users' },
    ];

    for (const artifact of windowsPaths) {
      try {
        const nodes = await w.go.ui.StorageHandler.ListDirectory(artifact.searchPath);
        if (nodes && nodes.length > 0) {
          discovered.push({
            name: artifact.name,
            type: artifact.type,
            path: artifact.searchPath,
            found: true,
            count: nodes.length,
          });
        }
      } catch {
        discovered.push({
          name: artifact.name,
          type: artifact.type,
          path: artifact.searchPath,
          found: false,
          count: 0,
        });
      }
    }

    setDeepArtifacts(discovered);
    setScanComplete(true);
  };

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'registry': return '\uD83D\uDDC3';
      case 'evtx': return '\uD83D\uDCCB';
      case 'mft': return '\uD83D\uDCC2';
      case 'prefetch': return '\u26A1';
      case 'shellbag': return '\uD83D\uDCC6';
      default: return '\uD83D\uDCC4';
    }
  };

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'registry': return 'var(--win-accent)';
      case 'evtx': return '#D83B01';
      case 'mft': return '#0F7B0F';
      case 'prefetch': return '#9D5D00';
      default: return 'var(--win-text-secondary)';
    }
  };

  const foundCount = deepArtifacts.filter(a => a.found).length;

  if (!session.isActive) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'var(--win-text-secondary)', fontSize: '13px' }}>
        Open evidence to begin artifact analysis.
      </div>
    );
  }

  return (
    <div style={{ padding: '16px', height: '100%', overflow: 'auto', backgroundColor: 'var(--win-bg)', color: 'var(--win-text)', fontFamily: 'var(--win-font)' }}>
      <div style={{ marginBottom: '16px' }}>
        <h2 style={{ margin: '0 0 4px 0', fontSize: '16px', fontWeight: '600' }}>Artifact Auto-Discovery</h2>
        <p style={{ margin: 0, fontSize: '12px', color: 'var(--win-text-secondary)' }}>
          Evidence: <strong>{session.fileName}</strong> | Format: {session.format} | Scanning filesystem for known forensic artifacts...
        </p>
      </div>

      {!scanComplete ? (
        <div style={{ padding: '24px', backgroundColor: 'var(--win-surface)', borderRadius: 'var(--win-radius)', border: '1px solid var(--win-stroke)', textAlign: 'center' }}>
          <div style={{ fontSize: '14px', marginBottom: '8px' }}>Scanning evidence for artifacts...</div>
          <div style={{ fontSize: '12px', color: 'var(--win-text-secondary)' }}>Checking known Windows artifact paths</div>
        </div>
      ) : (
        <>
          <div style={{ marginBottom: '16px', padding: '12px', backgroundColor: foundCount > 0 ? '#E8F5E9' : '#FDE7E9', borderRadius: 'var(--win-radius-sm)', border: `1px solid ${foundCount > 0 ? '#0F7B0F' : '#C42B1C'}`, fontSize: '13px' }}>
            {foundCount > 0 ? (
              <span><strong>{foundCount}</strong> artifact(s) discovered and ready for analysis.</span>
            ) : (
              <span>No standard Windows artifacts found. This may not be a Windows filesystem.</span>
            )}
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: '12px' }}>
            {deepArtifacts.map((artifact, idx) => (
              <div
                key={idx}
                style={{
                  padding: '12px',
                  backgroundColor: 'var(--win-surface)',
                  borderRadius: 'var(--win-radius-sm)',
                  border: `1px solid ${artifact.found ? getTypeColor(artifact.type) : 'var(--win-stroke)'}`,
                  opacity: artifact.found ? 1 : 0.5,
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '6px' }}>
                  <span style={{ fontSize: '16px' }}>{getTypeIcon(artifact.type)}</span>
                  <span style={{ fontWeight: '600', fontSize: '13px' }}>{artifact.name}</span>
                  <span style={{
                    marginLeft: 'auto',
                    fontSize: '10px',
                    padding: '2px 6px',
                    borderRadius: '8px',
                    backgroundColor: artifact.found ? '#E8F5E9' : '#FDE7E9',
                    color: artifact.found ? '#0F7B0F' : '#C42B1C',
                    fontWeight: '600',
                  }}>
                    {artifact.found ? 'FOUND' : 'NOT FOUND'}
                  </span>
                </div>
                <div style={{ fontSize: '11px', color: 'var(--win-text-secondary)', fontFamily: 'var(--win-font-mono)', wordBreak: 'break-all' }}>
                  {artifact.path}
                </div>
                {artifact.found && (
                  <div style={{ fontSize: '11px', color: 'var(--win-text-secondary)', marginTop: '4px' }}>
                    {artifact.count} item(s)
                  </div>
                )}
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
};

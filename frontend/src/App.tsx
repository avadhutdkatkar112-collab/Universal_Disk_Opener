import { useState, useCallback, useEffect } from 'react';
import './index.css';
import { TitleBar } from './components/TitleBar';
import { MenuBar } from './components/MenuBar';
import { StatusBar } from './components/StatusBar';
import { WelcomeDashboard } from './components/WelcomeDashboard';
import { CommandCenterModal } from './components/CommandCenterModal';
import { TaskExecutionHUD } from './components/TaskExecutionHUD';
import { HashModal } from './components/HashModal';
import { Sidebar } from './components/Sidebar';
import { DiskExplorer } from './components/DiskExplorer';
import { InvestigateView } from './components/InvestigateView';
import { ExamineView } from './components/ExamineView';
import { TimelineViewer } from './components/TimelineViewer';
import { ReportViewer } from './components/ReportViewer';
import { useEvidenceStore } from './store/evidenceStore';
import { useDiskStore } from './store/diskStore';
import { openDisk } from './lib/openDisk';

type HashModalMode = { type: 'single'; path: string } | { type: 'batch'; paths: string[] } | null;
type PrimaryMode = 'explorer' | 'investigate' | 'examine' | 'timeline' | 'case';

const modeConfig: Record<PrimaryMode, { label: string; icon: string }> = {
  explorer: { label: 'Explorer', icon: '\uD83D\uDCC2' },
  investigate: { label: 'Investigate', icon: '\uD83D\uDD0D' },
  examine: { label: 'Examine', icon: '\uD83D\uDD2C' },
  timeline: { label: 'Timeline', icon: '\uD83D\uDCC5' },
  case: { label: 'Case', icon: '\uD83D\uDCCB' },
};

export default function App() {
  const [cmdOpen, setCmdOpen] = useState(false);
  const { disk } = useDiskStore();
  const evidence = useEvidenceStore();
  const [hashModal, setHashModal] = useState<HashModalMode>(null);
  const [primaryMode, setPrimaryMode] = useState<PrimaryMode>('explorer');

  useEffect(() => {
    if (evidence.viewMode && evidence.viewMode !== primaryMode) {
      setPrimaryMode(evidence.viewMode);
    }
  }, [evidence.viewMode]);

  const handleMinimizeHash = useCallback(() => {
    setHashModal(null);
  }, []);

  const handleOpenFile = useCallback(() => {
    openDisk();
  }, []);

  const handleCloseEvidence = useCallback(() => {
    useEvidenceStore.getState().clearSession();
    useDiskStore.getState().setDisk(null);
  }, []);

  const handleHashSelected = useCallback((path?: string) => {
    if (path) {
      setHashModal({ type: 'single', path });
    } else {
      setHashModal({ type: 'single', path: '/' });
    }
  }, []);

  // Global keyboard shortcuts
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const mod = e.ctrlKey || e.metaKey;
      if (mod && e.key === 'o') {
        e.preventDefault();
        openDisk();
      } else if (mod && e.key === 'k') {
        e.preventDefault();
        setCmdOpen(prev => !prev);
      } else if (mod && e.key === 'w') {
        e.preventDefault();
        if (evidence.isActive) handleCloseEvidence();
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [evidence.isActive, handleCloseEvidence]);

  const renderMainContent = () => {
    if (!evidence.isActive) {
      return <WelcomeDashboard onOpen={handleOpenFile} />;
    }

    switch (primaryMode) {
      case 'explorer':
        return <DiskExplorer />;
      case 'investigate':
        return <InvestigateView />;
      case 'examine':
        return <ExamineView />;
      case 'timeline':
        return <TimelineViewer />;
      case 'case':
        return <ReportViewer />;
      default:
        return <DiskExplorer />;
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100vh', background: 'var(--win-bg)', overflow: 'hidden' }}>
      <TitleBar />
      <MenuBar
        onOpen={handleOpenFile}
        onCommandCenter={() => setCmdOpen(true)}
        onClose={evidence.isActive ? handleCloseEvidence : undefined}
        onHash={evidence.isActive ? () => handleHashSelected('/') : undefined}
      />

      {/* Evidence Banner — permanent READ-ONLY indicator */}
      {evidence.isActive && (
        <div style={{
          backgroundColor: '#E8F5E9',
          borderBottom: '2px solid #0F7B0F',
          padding: '6px 16px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          fontSize: '11px',
          fontFamily: 'var(--win-font)',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <span style={{
              backgroundColor: '#0F7B0F',
              color: '#fff',
              padding: '2px 8px',
              borderRadius: '10px',
              fontWeight: '600',
              fontSize: '10px',
            }}>
              READ ONLY
            </span>
            <span style={{ color: 'var(--win-text)' }}>
              <strong>{evidence.fileName}</strong>
            </span>
            <span style={{ color: 'var(--win-text-secondary)' }}>
              {evidence.format}
            </span>
            <span style={{ color: 'var(--win-text-secondary)' }}>
              {evidence.partitions.length} partition(s)
            </span>
            {evidence.sha256 && (
              <span style={{ color: 'var(--win-text-secondary)', fontFamily: 'var(--win-font-mono)' }}>
                SHA-256: {evidence.sha256.slice(0, 24)}...
              </span>
            )}
          </div>
          {evidence.isAnalyzing && (
            <span style={{ color: 'var(--win-accent)', fontWeight: '500' }}>{evidence.analysisProgress}</span>
          )}
        </div>
      )}

      {/* Primary Mode Tabs */}
      <div style={{
        display: 'flex',
        gap: '0',
        padding: '0 16px',
        backgroundColor: 'var(--win-bg)',
        borderBottom: '1px solid var(--win-stroke)',
      }}>
        {(Object.keys(modeConfig) as PrimaryMode[]).map((mode) => (
          <button
            key={mode}
            onClick={() => setPrimaryMode(mode)}
            style={{
              padding: '8px 16px',
              backgroundColor: primaryMode === mode ? 'var(--win-surface)' : 'transparent',
              color: primaryMode === mode ? 'var(--win-accent)' : 'var(--win-text-secondary)',
              border: 'none',
              borderBottom: primaryMode === mode ? '2px solid var(--win-accent)' : '2px solid transparent',
              cursor: 'pointer',
              fontWeight: primaryMode === mode ? '600' : '400',
              fontSize: '13px',
              fontFamily: 'var(--win-font)',
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
            }}
          >
            <span>{modeConfig[mode].icon}</span>
            {modeConfig[mode].label}
          </button>
        ))}
      </div>

      {/* Main Content Area: Sidebar + Content */}
      <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        {evidence.isActive && <Sidebar />}
        <div style={{ flex: 1, overflow: 'hidden' }}>
          {renderMainContent()}
        </div>
      </div>

      <StatusBar disk={disk} />
      <TaskExecutionHUD />
      <CommandCenterModal open={cmdOpen} onClose={() => setCmdOpen(false)} disk={disk} />

      {hashModal?.type === 'single' && (
        <HashModal
          filePath={hashModal.path}
          onClose={() => setHashModal(null)}
          onMinimize={handleMinimizeHash}
        />
      )}
      {hashModal?.type === 'batch' && (
        <HashModal
          filePaths={hashModal.paths}
          onClose={() => setHashModal(null)}
          onMinimize={handleMinimizeHash}
        />
      )}
    </div>
  );
}

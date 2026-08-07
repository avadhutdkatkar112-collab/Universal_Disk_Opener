import { useState, useCallback } from 'react';
import { ParseEVTXFile, ParseMFTFile, BuildUnifiedTimeline } from '../lib/wails';

interface EvtxMftViewerProps {
  onClose: () => void;
}

interface TimelineEntry {
  id: string;
  timestamp: string;
  source: string;
  event_type: string;
  title: string;
  description: string;
  path?: string;
  data?: Record<string, any>;
}

export default function EvtxMftViewer({ onClose }: EvtxMftViewerProps) {
  const [activeTab, setActiveTab] = useState<'evtx' | 'mft' | 'timeline'>('evtx');
  const [evtxPath, setEvtxPath] = useState('');
  const [mftPath, setMftPath] = useState('');
  const [registryPath, setRegistryPath] = useState('');
  const [startTime, setStartTime] = useState('');
  const [endTime, setEndTime] = useState('');
  
  const [evtxData, setEvtxData] = useState<any>(null);
  const [mftData, setMftData] = useState<any>(null);
  const [timelineData, setTimelineData] = useState<any>(null);
  
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleParseEVTX = useCallback(async () => {
    if (!evtxPath) return;
    setLoading(true);
    setError(null);
    try {
      const result = await ParseEVTXFile(evtxPath);
      setEvtxData(result);
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }, [evtxPath]);

  const handleParseMFT = useCallback(async () => {
    if (!mftPath) return;
    setLoading(true);
    setError(null);
    try {
      const result = await ParseMFTFile(mftPath);
      setMftData(result);
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }, [mftPath]);

  const handleBuildTimeline = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await BuildUnifiedTimeline(
        registryPath, evtxPath, mftPath, startTime, endTime
      );
      setTimelineData(result);
      setActiveTab('timeline');
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }, [registryPath, evtxPath, mftPath, startTime, endTime]);

  const formatTimestamp = (ts: string) => {
    if (!ts) return 'N/A';
    return new Date(ts).toLocaleString();
  };

  const getSourceColor = (source: string) => {
    switch (source) {
      case 'registry': return '#0078D4';
      case 'evtx': return '#107C10';
      case 'mft': return '#D83B01';
      default: return 'var(--win-text-secondary)';
    }
  };

  return (
    <div style={{
      position: 'fixed', top: 32, right: 0, bottom: 24, width: 800,
      background: 'var(--win-surface)', borderLeft: '1px solid var(--win-stroke)',
      boxShadow: '-4px 0 16px rgba(0, 0, 0, 0.08)', display: 'flex', flexDirection: 'column',
      zIndex: 350, animation: 'slideInRight 0.15s ease',
    }}>
      {/* Header */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 8, padding: '8px 12px',
        borderBottom: '1px solid var(--win-stroke)', flexShrink: 0,
      }}>
        <div style={{ flex: 1 }}>
          <div style={{ fontSize: 13, fontWeight: 500, color: 'var(--win-text)' }}>
            Forensic Artifact Analyzer
          </div>
          <div style={{ fontSize: 11, color: 'var(--win-text-tertiary)' }}>
            EVTX Event Logs, MFT Timeline, and Unified Correlation
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

      {/* Tabs */}
      <div style={{ display: 'flex', gap: 4, padding: '8px 12px', borderBottom: '1px solid var(--win-stroke)' }}>
        {(['evtx', 'mft', 'timeline'] as const).map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            style={{
              padding: '4px 12px', fontSize: 11,
              border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
              background: activeTab === tab ? 'var(--win-accent)' : 'var(--win-bg)',
              color: activeTab === tab ? 'white' : 'var(--win-text)',
            }}
          >
            {tab === 'evtx' ? 'EVTX Events' : tab === 'mft' ? 'MFT Records' : 'Unified Timeline'}
          </button>
        ))}
      </div>

      {/* Error Banner */}
      {error && (
        <div style={{
          padding: '8px 12px', background: 'rgba(232, 17, 35, 0.08)',
          color: 'var(--win-danger)', fontSize: 12,
        }}>
          {error}
        </div>
      )}

      {/* Content */}
      <div style={{ flex: 1, overflow: 'auto', padding: 12 }}>
        {/* EVTX Tab */}
        {activeTab === 'evtx' && (
          <div>
            <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
              <input
                type="text"
                placeholder="Path to .evtx file (e.g., /Windows/System32/winevt/Logs/Security.evtx)"
                value={evtxPath}
                onChange={e => setEvtxPath(e.target.value)}
                style={{
                  flex: 1, padding: '6px 10px', fontSize: 12,
                  border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
                  background: 'var(--win-bg)', color: 'var(--win-text)',
                  fontFamily: 'var(--win-font-mono)',
                }}
              />
              <button
                onClick={handleParseEVTX}
                disabled={loading || !evtxPath}
                style={{
                  padding: '6px 16px', fontSize: 12, fontWeight: 500,
                  border: '1px solid var(--win-accent)', borderRadius: 'var(--win-radius-sm)',
                  background: loading ? 'var(--win-stroke)' : 'var(--win-accent)',
                  color: 'white', cursor: loading ? 'wait' : 'pointer',
                }}
              >
                {loading ? 'Parsing...' : 'Parse EVTX'}
              </button>
            </div>

            {evtxData && (
              <div>
                <div style={{ fontSize: 12, color: 'var(--win-text-secondary)', marginBottom: 8 }}>
                  Total Events: {evtxData.total_events}
                </div>
                <div style={{ maxHeight: 400, overflow: 'auto' }}>
                  {evtxData.forensic_events?.map((event: any, idx: number) => (
                    <div key={idx} style={{
                      padding: '8px', marginBottom: 4,
                      background: 'var(--win-bg)', border: '1px solid var(--win-stroke)',
                      borderRadius: 'var(--win-radius-sm)',
                    }}>
                      <div style={{ display: 'flex', gap: 8, fontSize: 11 }}>
                        <span style={{ color: 'var(--win-accent)', fontWeight: 500 }}>
                          Event {event.EventRecordID}
                        </span>
                        <span style={{ color: 'var(--win-text-tertiary)' }}>
                          {formatTimestamp(event.TimeCreated)}
                        </span>
                      </div>
                      <div style={{ fontSize: 11, marginTop: 4 }}>
                        <strong>{event.ProviderName}</strong>
                        {event.Data?.UserID && <span> - User: {event.Data.UserID}</span>}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        {/* MFT Tab */}
        {activeTab === 'mft' && (
          <div>
            <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
              <input
                type="text"
                placeholder="Path to $MFT file (e.g., /\\$MFT)"
                value={mftPath}
                onChange={e => setMftPath(e.target.value)}
                style={{
                  flex: 1, padding: '6px 10px', fontSize: 12,
                  border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
                  background: 'var(--win-bg)', color: 'var(--win-text)',
                  fontFamily: 'var(--win-font-mono)',
                }}
              />
              <button
                onClick={handleParseMFT}
                disabled={loading || !mftPath}
                style={{
                  padding: '6px 16px', fontSize: 12, fontWeight: 500,
                  border: '1px solid var(--win-accent)', borderRadius: 'var(--win-radius-sm)',
                  background: loading ? 'var(--win-stroke)' : 'var(--win-accent)',
                  color: 'white', cursor: loading ? 'wait' : 'pointer',
                }}
              >
                {loading ? 'Parsing...' : 'Parse MFT'}
              </button>
            </div>

            {mftData && (
              <div>
                <div style={{ fontSize: 12, color: 'var(--win-text-secondary)', marginBottom: 8 }}>
                  Total Records: {mftData.total_records}
                </div>
                <pre style={{
                  padding: 12, background: 'var(--win-bg)',
                  border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
                  fontSize: 11, fontFamily: 'var(--win-font-mono)', overflow: 'auto',
                }}>
                  {JSON.stringify(mftData.summary, null, 2)}
                </pre>
              </div>
            )}
          </div>
        )}

        {/* Timeline Tab */}
        {activeTab === 'timeline' && (
          <div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 8, marginBottom: 12 }}>
              <input
                type="text"
                placeholder="Registry hive path (optional)"
                value={registryPath}
                onChange={e => setRegistryPath(e.target.value)}
                style={{
                  padding: '6px 10px', fontSize: 11,
                  border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
                  background: 'var(--win-bg)', color: 'var(--win-text)',
                  fontFamily: 'var(--win-font-mono)',
                }}
              />
              <input
                type="datetime-local"
                value={startTime}
                onChange={e => setStartTime(e.target.value)}
                style={{
                  padding: '6px 10px', fontSize: 11,
                  border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
                  background: 'var(--win-bg)', color: 'var(--win-text)',
                }}
              />
              <input
                type="datetime-local"
                value={endTime}
                onChange={e => setEndTime(e.target.value)}
                style={{
                  padding: '6px 10px', fontSize: 11,
                  border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
                  background: 'var(--win-bg)', color: 'var(--win-text)',
                }}
              />
            </div>

            <button
              onClick={handleBuildTimeline}
              disabled={loading}
              style={{
                padding: '6px 16px', fontSize: 12, fontWeight: 500,
                border: '1px solid var(--win-accent)', borderRadius: 'var(--win-radius-sm)',
                background: loading ? 'var(--win-stroke)' : 'var(--win-accent)',
                color: 'white', cursor: loading ? 'wait' : 'pointer',
                marginBottom: 12,
              }}
            >
              {loading ? 'Building Timeline...' : 'Build Unified Timeline'}
            </button>

            {timelineData && (
              <div>
                <div style={{ fontSize: 12, color: 'var(--win-text-secondary)', marginBottom: 8 }}>
                  Total Events: {timelineData.total_events} | 
                  Time Range: {formatTimestamp(timelineData.start_time)} - {formatTimestamp(timelineData.end_time)}
                </div>

                {/* Source Breakdown */}
                <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
                  {Object.entries(timelineData.statistics?.source_counts || {}).map(([source, count]) => (
                    <div key={source} style={{
                      padding: '4px 8px', fontSize: 10,
                      background: getSourceColor(source) + '20',
                      border: `1px solid ${getSourceColor(source)}`,
                      borderRadius: 'var(--win-radius-sm)',
                      color: getSourceColor(source),
                    }}>
                      {source}: {String(count)}
                    </div>
                  ))}
                </div>

                {/* Timeline Entries */}
                <div style={{ maxHeight: 400, overflow: 'auto' }}>
                  {timelineData.entries?.map((entry: TimelineEntry, idx: number) => (
                    <div key={idx} style={{
                      padding: '8px', marginBottom: 4,
                      background: 'var(--win-bg)', border: '1px solid var(--win-stroke)',
                      borderRadius: 'var(--win-radius-sm)',
                      borderLeft: `3px solid ${getSourceColor(entry.source)}`,
                    }}>
                      <div style={{ display: 'flex', gap: 8, fontSize: 11 }}>
                        <span style={{ 
                          padding: '1px 6px', fontSize: 9, fontWeight: 500,
                          background: getSourceColor(entry.source),
                          color: 'white', borderRadius: 3,
                        }}>
                          {entry.source.toUpperCase()}
                        </span>
                        <span style={{ color: 'var(--win-text)', fontWeight: 500 }}>
                          {entry.title}
                        </span>
                        <span style={{ color: 'var(--win-text-tertiary)', marginLeft: 'auto' }}>
                          {formatTimestamp(entry.timestamp)}
                        </span>
                      </div>
                      <div style={{ fontSize: 10, color: 'var(--win-text-secondary)', marginTop: 4 }}>
                        {entry.description}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

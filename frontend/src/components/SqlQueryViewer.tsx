import React, { useState } from 'react';
import { useTimeline } from '../hooks/useTimeline';
import { LoadTimelineToSQL, ExecuteSQLQuery } from '../lib/wails';

interface QueryResult {
  columns: string[];
  rows: Record<string, any>[];
  count: number;
  time_ms: number;
}

const PRESET_QUERIES = [
  { label: 'All Events', query: 'SELECT * FROM timeline_events ORDER BY timestamp DESC LIMIT 100' },
  { label: 'By Source', query: 'SELECT source, COUNT(*) as count FROM timeline_events GROUP BY source ORDER BY count DESC' },
  { label: 'Event Types', query: 'SELECT event_type, COUNT(*) as count FROM timeline_events GROUP BY event_type ORDER BY count DESC' },
  { label: 'Registry Events', query: "SELECT * FROM timeline_events WHERE source = 'registry' ORDER BY timestamp DESC LIMIT 50" },
  { label: 'EVTX Events', query: "SELECT * FROM timeline_events WHERE source = 'evtx' ORDER BY timestamp DESC LIMIT 50" },
  { label: 'MFT Events', query: "SELECT * FROM timeline_events WHERE source = 'mft' ORDER BY timestamp DESC LIMIT 50" },
  { label: 'Suspicious Paths', query: "SELECT * FROM timeline_events WHERE description LIKE '%cmd.exe%' OR description LIKE '%powershell%' OR description LIKE '%temp%'" },
  { label: 'Hourly Activity', query: "SELECT strftime('%H', timestamp) as hour, COUNT(*) as count FROM timeline_events GROUP BY hour ORDER BY hour" },
];

export const SqlQueryViewer: React.FC = () => {
  const { entries, loading: timelineLoading } = useTimeline();
  const [query, setQuery] = useState("SELECT source, COUNT(*) as event_count FROM timeline_events GROUP BY source ORDER BY event_count DESC");
  const [result, setResult] = useState<QueryResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loaded, setLoaded] = useState(false);

  const handleLoadData = async () => {
    setLoading(true);
    setError(null);
    try {
      const count = await LoadTimelineToSQL(entries);
      setLoaded(true);
      setResult({ columns: ['status'], rows: [{ status: `Loaded ${count} events into SQL database` }], count: 1, time_ms: 0 });
    } catch (err: any) {
      setError(err?.toString() || 'Failed to load data');
    } finally {
      setLoading(false);
    }
  };

  const handleExecute = async () => {
    if (!query.trim()) return;
    setLoading(true);
    setError(null);
    try {
      const res = await ExecuteSQLQuery(query);
      setResult(res);
    } catch (err: any) {
      setError(err?.toString() || 'Query failed');
      setResult(null);
    } finally {
      setLoading(false);
    }
  };

  const styles = {
    container: {
      padding: '16px',
      backgroundColor: 'var(--win-bg)',
      color: 'var(--win-text)',
      fontFamily: 'var(--win-font)',
      height: '100%',
      boxSizing: 'border-box' as const,
      display: 'flex',
      flexDirection: 'column' as const,
      gap: '12px',
    },
    header: {
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'center',
    },
    title: {
      fontSize: '18px',
      fontWeight: '600',
      margin: 0,
    },
    loadBtn: {
      padding: '8px 16px',
      backgroundColor: loaded ? 'var(--win-success)' : 'var(--win-accent)',
      color: '#FFFFFF',
      border: 'none',
      borderRadius: 'var(--win-radius-sm)',
      cursor: 'pointer',
      fontSize: '13px',
      fontWeight: '500',
    },
    queryArea: {
      display: 'flex',
      gap: '8px',
      alignItems: 'flex-start',
    },
    textarea: {
      flex: 1,
      minHeight: '80px',
      padding: '10px',
      backgroundColor: 'var(--win-surface)',
      border: '1px solid var(--win-stroke-strong)',
      borderRadius: 'var(--win-radius-sm)',
      color: 'var(--win-text)',
      fontFamily: 'var(--win-font-mono)',
      fontSize: '13px',
      resize: 'vertical' as const,
      outline: 'none',
    },
    runBtn: {
      padding: '8px 20px',
      backgroundColor: 'var(--win-accent)',
      color: '#FFFFFF',
      border: 'none',
      borderRadius: 'var(--win-radius-sm)',
      cursor: 'pointer',
      fontSize: '13px',
      fontWeight: '600',
      alignSelf: 'stretch',
    },
    presets: {
      display: 'flex',
      gap: '6px',
      flexWrap: 'wrap' as const,
    },
    presetBtn: {
      padding: '4px 10px',
      backgroundColor: 'var(--win-control)',
      color: 'var(--win-text-secondary)',
      border: '1px solid var(--win-stroke)',
      borderRadius: 'var(--win-radius-sm)',
      cursor: 'pointer',
      fontSize: '11px',
    },
    error: {
      color: 'var(--win-danger)',
      padding: '8px 12px',
      backgroundColor: '#FDE7E9',
      borderRadius: 'var(--win-radius-sm)',
      fontSize: '13px',
    },
    resultInfo: {
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'center',
      fontSize: '12px',
      color: 'var(--win-text-secondary)',
    },
    tableContainer: {
      flex: 1,
      overflow: 'auto' as const,
      backgroundColor: 'var(--win-surface)',
      borderRadius: 'var(--win-radius)',
      border: '1px solid var(--win-stroke)',
    },
    table: {
      width: '100%',
      borderCollapse: 'collapse' as const,
      fontSize: '12px',
    },
    th: {
      padding: '8px 12px',
      backgroundColor: 'var(--win-elevated)',
      borderBottom: '2px solid var(--win-stroke-strong)',
      fontWeight: '600',
      textAlign: 'left' as const,
      color: 'var(--win-text-secondary)',
      position: 'sticky' as const,
      top: 0,
    },
    td: {
      padding: '6px 12px',
      borderBottom: '1px solid var(--win-stroke)',
      fontFamily: 'var(--win-font-mono)',
    },
  };

  return (
    <div style={styles.container}>
      <div style={styles.header}>
        <h2 style={styles.title}>SQL Analytics Console</h2>
        <button
          onClick={handleLoadData}
          disabled={loading || timelineLoading || entries.length === 0}
          style={{
            ...styles.loadBtn,
            opacity: (loading || timelineLoading || entries.length === 0) ? 0.6 : 1,
          }}
        >
          {timelineLoading ? 'Loading Timeline...' : loaded ? 'Data Loaded' : `Load ${entries.length} Events`}
        </button>
      </div>

      <div style={styles.queryArea}>
        <textarea
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Enter SQL query..."
          style={styles.textarea}
        />
        <button
          onClick={handleExecute}
          disabled={loading || !loaded}
          style={{
            ...styles.runBtn,
            opacity: (loading || !loaded) ? 0.6 : 1,
          }}
        >
          {loading ? 'Running...' : 'Run Query'}
        </button>
      </div>

      <div style={styles.presets}>
        {PRESET_QUERIES.map((preset, idx) => (
          <button
            key={idx}
            onClick={() => setQuery(preset.query)}
            style={styles.presetBtn}
          >
            {preset.label}
          </button>
        ))}
      </div>

      {error && <div style={styles.error}>{error}</div>}

      {result && (
        <>
          <div style={styles.resultInfo}>
            <span>{result.count} row{result.count !== 1 ? 's' : ''} returned</span>
            <span>{result.time_ms.toFixed(2)}ms</span>
          </div>
          <div style={styles.tableContainer}>
            <table style={styles.table}>
              <thead>
                <tr>
                  {result.columns.map((col, idx) => (
                    <th key={idx} style={styles.th}>{col}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {result.rows.map((row, idx) => (
                  <tr key={idx}>
                    {result.columns.map((col, colIdx) => (
                      <td key={colIdx} style={styles.td}>
                        {row[col] !== null && row[col] !== undefined ? String(row[col]) : 'NULL'}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {!result && !error && (
        <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--win-text-tertiary)' }}>
          {loaded ? 'Enter a SQL query and click Run Query' : 'Click "Load Events" to populate the SQL database from your timeline'}
        </div>
      )}
    </div>
  );
};

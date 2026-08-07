import React, { useState, useMemo } from 'react';
import { useTimeline } from '../hooks/useTimeline';

export const TimelineViewer: React.FC = () => {
  const [regPath, setRegPath] = useState('');
  const [evtxPath, setEvtxPath] = useState('');
  const [mftPath, setMftPath] = useState('');
  
  const [searchQuery, setSearchQuery] = useState('');
  const [sourceFilter, setSourceFilter] = useState<'ALL' | 'REGISTRY' | 'EVTX' | 'MFT'>('ALL');
  const [exportPath, setExportPath] = useState('C:\\timeline_export.csv');

  const { entries, loading, error, buildTimeline } = useTimeline();

  const handleGenerate = () => {
    buildTimeline({
      registry_path: regPath,
      evtx_path: evtxPath,
      mft_path: mftPath,
    });
  };

  const filteredEntries = useMemo(() => {
    return entries.filter((entry) => {
      const src = entry.source.toUpperCase();
      const matchesSource = sourceFilter === 'ALL' || src === sourceFilter;
      const matchesQuery =
        searchQuery === '' ||
        entry.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
        (entry.path || '').toLowerCase().includes(searchQuery.toLowerCase()) ||
        entry.title.toLowerCase().includes(searchQuery.toLowerCase());
      return matchesSource && matchesQuery;
    });
  }, [entries, sourceFilter, searchQuery]);

  const handleExportCSV = async () => {
    if (!exportPath || filteredEntries.length === 0) return;
    try {
      const { ExportTimelineCSV } = await import('../lib/wails');
      await ExportTimelineCSV(filteredEntries, exportPath);
      alert('CSV Export completed successfully!');
    } catch (err: any) {
      alert('Export failed: ' + err?.toString());
    }
  };

  const handleExportJSON = async () => {
    const jsonPath = exportPath.replace(/\.csv$/, '.json');
    if (!jsonPath || filteredEntries.length === 0) return;
    try {
      const { ExportTimelineJSON } = await import('../lib/wails');
      await ExportTimelineJSON(filteredEntries, jsonPath);
      alert('JSON Export completed successfully!');
    } catch (err: any) {
      alert('Export failed: ' + err?.toString());
    }
  };

  const getSourceColor = (source: string) => {
    switch (source.toUpperCase()) {
      case 'REGISTRY': return 'var(--win-accent)';
      case 'EVTX': return '#6a1b9a';
      case 'MFT': return 'var(--win-success)';
      default: return 'var(--win-text-secondary)';
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
      overflow: 'auto',
    },
    title: {
      fontSize: '18px',
      fontWeight: '600',
      marginBottom: '16px',
      color: 'var(--win-text)',
    },
    inputGrid: {
      display: 'grid',
      gridTemplateColumns: '1fr 1fr 1fr auto',
      gap: '10px',
      marginBottom: '12px',
    },
    input: {
      padding: '8px 12px',
      backgroundColor: 'var(--win-surface)',
      border: '1px solid var(--win-stroke-strong)',
      borderRadius: 'var(--win-radius-sm)',
      color: 'var(--win-text)',
      fontSize: '13px',
      fontFamily: 'var(--win-font)',
      outline: 'none',
    },
    button: {
      padding: '8px 16px',
      backgroundColor: 'var(--win-accent)',
      color: '#FFFFFF',
      border: 'none',
      borderRadius: 'var(--win-radius-sm)',
      cursor: 'pointer',
      fontSize: '13px',
      fontWeight: '500',
    },
    buttonDisabled: {
      opacity: 0.6,
      cursor: 'not-allowed',
    },
    filterBar: {
      display: 'flex',
      gap: '10px',
      marginBottom: '12px',
      alignItems: 'center',
      backgroundColor: 'var(--win-surface)',
      padding: '10px 12px',
      borderRadius: 'var(--win-radius)',
      border: '1px solid var(--win-stroke)',
    },
    select: {
      padding: '6px 10px',
      backgroundColor: 'var(--win-surface)',
      border: '1px solid var(--win-stroke-strong)',
      borderRadius: 'var(--win-radius-sm)',
      color: 'var(--win-text)',
      fontSize: '13px',
      fontFamily: 'var(--win-font)',
      outline: 'none',
    },
    exportButton: {
      padding: '6px 12px',
      backgroundColor: 'var(--win-success)',
      color: '#FFFFFF',
      border: 'none',
      borderRadius: 'var(--win-radius-sm)',
      cursor: 'pointer',
      fontSize: '12px',
      fontWeight: '500',
    },
    exportButtonAlt: {
      padding: '6px 12px',
      backgroundColor: '#6a1b9a',
      color: '#FFFFFF',
      border: 'none',
      borderRadius: 'var(--win-radius-sm)',
      cursor: 'pointer',
      fontSize: '12px',
      fontWeight: '500',
    },
    error: {
      color: 'var(--win-danger)',
      marginBottom: '12px',
      padding: '8px 12px',
      backgroundColor: '#FDE7E9',
      borderRadius: 'var(--win-radius-sm)',
      fontSize: '13px',
    },
    tableContainer: {
      overflowY: 'auto' as const,
      maxHeight: 'calc(100vh - 280px)',
      backgroundColor: 'var(--win-surface)',
      borderRadius: 'var(--win-radius)',
      border: '1px solid var(--win-stroke)',
    },
    table: {
      width: '100%',
      borderCollapse: 'collapse' as const,
      textAlign: 'left' as const,
      fontSize: '13px',
    },
    th: {
      padding: '10px 12px',
      backgroundColor: 'var(--win-elevated)',
      borderBottom: '2px solid var(--win-stroke-strong)',
      fontWeight: '600',
      color: 'var(--win-text-secondary)',
      position: 'sticky' as const,
      top: 0,
    },
    td: {
      padding: '8px 12px',
      borderBottom: '1px solid var(--win-stroke)',
    },
    trEven: {
      backgroundColor: 'var(--win-surface)',
    },
    trOdd: {
      backgroundColor: 'var(--win-elevated)',
    },
    sourceBadge: (color: string) => ({
      display: 'inline-block',
      padding: '2px 8px',
      borderRadius: '10px',
      backgroundColor: color + '15',
      color: color,
      fontSize: '11px',
      fontWeight: '600',
    }),
    timestamp: {
      fontFamily: 'var(--win-font-mono)',
      fontSize: '12px',
      color: 'var(--win-accent)',
    },
    path: {
      fontFamily: 'var(--win-font-mono)',
      fontSize: '11px',
      color: 'var(--win-text-secondary)',
    },
  };

  return (
    <div style={styles.container}>
      <h2 style={styles.title}>Unified MACB Timeline Engine</h2>

      <div style={styles.inputGrid}>
        <input
          type="text"
          value={regPath}
          onChange={(e) => setRegPath(e.target.value)}
          placeholder="Registry Hive Path (SYSTEM, SOFTWARE...)"
          style={styles.input}
        />
        <input
          type="text"
          value={evtxPath}
          onChange={(e) => setEvtxPath(e.target.value)}
          placeholder="EVTX Log Path (Security.evtx...)"
          style={styles.input}
        />
        <input
          type="text"
          value={mftPath}
          onChange={(e) => setMftPath(e.target.value)}
          placeholder="$MFT File Path"
          style={styles.input}
        />
        <button 
          onClick={handleGenerate} 
          disabled={loading}
          style={{ ...styles.button, ...(loading ? styles.buttonDisabled : {}) }}
        >
          {loading ? 'Building...' : 'Correlate'}
        </button>
      </div>

      <div style={styles.filterBar}>
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder="Filter timeline entries by keyword..."
          style={{ ...styles.input, flex: 1 }}
        />
        <select
          value={sourceFilter}
          onChange={(e) => setSourceFilter(e.target.value as any)}
          style={styles.select}
        >
          <option value="ALL">All Sources</option>
          <option value="REGISTRY">Registry Only</option>
          <option value="EVTX">EVTX Only</option>
          <option value="MFT">MFT Only</option>
        </select>
        <input
          type="text"
          value={exportPath}
          onChange={(e) => setExportPath(e.target.value)}
          placeholder="Export file path"
          style={{ ...styles.input, width: '220px' }}
        />
        <button 
          onClick={handleExportCSV} 
          disabled={filteredEntries.length === 0}
          style={{ ...styles.exportButton, ...(filteredEntries.length === 0 ? styles.buttonDisabled : {}) }}
        >
          Export CSV
        </button>
        <button 
          onClick={handleExportJSON} 
          disabled={filteredEntries.length === 0}
          style={{ ...styles.exportButtonAlt, ...(filteredEntries.length === 0 ? styles.buttonDisabled : {}) }}
        >
          Export JSON
        </button>
      </div>

      {error && <div style={styles.error}>{error}</div>}

      <div style={styles.tableContainer}>
        <table style={styles.table}>
          <thead>
            <tr>
              <th style={styles.th}>Timestamp (UTC)</th>
              <th style={styles.th}>Source</th>
              <th style={styles.th}>Event Type</th>
              <th style={styles.th}>Title</th>
              <th style={styles.th}>Description</th>
              <th style={styles.th}>Path</th>
            </tr>
          </thead>
          <tbody>
            {filteredEntries.map((entry, idx) => (
              <tr key={entry.id || idx} style={idx % 2 === 0 ? styles.trEven : styles.trOdd}>
                <td style={{ ...styles.td, ...styles.timestamp }}>
                  {entry.timestamp ? new Date(entry.timestamp).toISOString().replace('T', ' ').slice(0, 23) + 'Z' : '-'}
                </td>
                <td style={styles.td}>
                  <span style={styles.sourceBadge(getSourceColor(entry.source))}>
                    {entry.source.toUpperCase()}
                  </span>
                </td>
                <td style={{ ...styles.td, fontSize: '12px' }}>{entry.event_type}</td>
                <td style={styles.td}>{entry.title}</td>
                <td style={styles.td}>{entry.description}</td>
                <td style={{ ...styles.td, ...styles.path }}>{entry.path || '-'}</td>
              </tr>
            ))}
            {filteredEntries.length === 0 && (
              <tr>
                <td colSpan={6} style={{ ...styles.td, textAlign: 'center', padding: '32px', color: 'var(--win-text-tertiary)' }}>
                  {loading ? 'Loading timeline...' : 'No entries found. Provide artifact paths and click Correlate.'}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
};

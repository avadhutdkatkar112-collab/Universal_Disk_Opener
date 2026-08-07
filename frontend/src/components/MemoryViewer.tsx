import React, { useState, useEffect } from 'react';
import { GetLiveProcesses, RunMemoryYaraScan } from '../lib/wails';

interface Socket {
  type: string;
  local_addr: string;
  remote_addr: string;
  state: string;
}

interface ProcessInfo {
  pid: number;
  ppid: number;
  name: string;
  path: string;
  command_line: string;
  username: string;
  create_time: string;
  open_sockets: Socket[];
  is_suspicious: boolean;
  flag_reason: string;
}

interface YaraMatch {
  rule_name: string;
  pid: number;
  process_name: string;
  matched_data: string;
  severity: string;
  tags: string[];
}

export const MemoryViewer: React.FC = () => {
  const [processes, setProcesses] = useState<ProcessInfo[]>([]);
  const [matches, setMatches] = useState<YaraMatch[]>([]);
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState('');

  const fetchMemoryState = async () => {
    setLoading(true);
    try {
      const procs = await GetLiveProcesses();
      setProcesses(procs || []);
      const yara = await RunMemoryYaraScan(procs || []);
      setMatches(yara || []);
    } catch (err) {
      console.error('Failed to capture memory state:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchMemoryState();
  }, []);

  const filteredProcs = processes.filter(p =>
    p.name.toLowerCase().includes(search.toLowerCase()) ||
    p.pid.toString().includes(search) ||
    p.command_line.toLowerCase().includes(search.toLowerCase())
  );

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
    refreshBtn: {
      padding: '8px 16px',
      backgroundColor: 'var(--win-accent)',
      color: '#FFFFFF',
      border: 'none',
      borderRadius: 'var(--win-radius-sm)',
      cursor: 'pointer',
      fontSize: '13px',
      fontWeight: '500',
    },
    alertBanner: {
      backgroundColor: '#FDE7E9',
      border: '1px solid #C42B1C',
      borderRadius: 'var(--win-radius-sm)',
      padding: '12px',
    },
    alertTitle: {
      color: '#C42B1C',
      fontWeight: '600',
      fontSize: '14px',
      margin: '0 0 8px 0',
    },
    alertItem: {
      color: '#C42B1C',
      fontSize: '12px',
      marginBottom: '4px',
    },
    searchInput: {
      width: '100%',
      padding: '8px 12px',
      backgroundColor: 'var(--win-surface)',
      border: '1px solid var(--win-stroke-strong)',
      borderRadius: 'var(--win-radius-sm)',
      color: 'var(--win-text)',
      fontSize: '13px',
      fontFamily: 'var(--win-font)',
      outline: 'none',
    },
    tableContainer: {
      flex: 1,
      overflow: 'auto' as const,
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
      padding: '8px 12px',
      borderBottom: '1px solid var(--win-stroke)',
    },
    suspiciousRow: {
      backgroundColor: '#FDE7E9',
    },
    socketItem: {
      fontSize: '10px',
      color: 'var(--win-accent)',
      fontFamily: 'var(--win-font-mono)',
    },
  };

  return (
    <div style={styles.container}>
      <div style={styles.header}>
        <h2 style={styles.title}>Live Memory & Volatile Process Triage</h2>
        <button onClick={fetchMemoryState} disabled={loading} style={styles.refreshBtn}>
          {loading ? 'Capturing...' : 'Refresh Snapshot'}
        </button>
      </div>

      {matches.length > 0 && (
        <div style={styles.alertBanner}>
          <h3 style={styles.alertTitle}>YARA In-Memory Detections ({matches.length})</h3>
          {matches.map((m, idx) => (
            <div key={idx} style={styles.alertItem}>
              <strong>[{m.severity}] {m.rule_name}</strong> &rarr; PID: {m.pid} ({m.process_name}) | Matched: <code>{m.matched_data}</code>
            </div>
          ))}
        </div>
      )}

      <input
        type="text"
        placeholder="Filter by PID, Name, or Command Line..."
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        style={styles.searchInput}
      />

      <div style={styles.tableContainer}>
        <table style={styles.table}>
          <thead>
            <tr>
              <th style={styles.th}>PID</th>
              <th style={styles.th}>PPID</th>
              <th style={styles.th}>Name</th>
              <th style={styles.th}>User</th>
              <th style={styles.th}>Created</th>
              <th style={styles.th}>Command Line</th>
              <th style={styles.th}>Connections</th>
            </tr>
          </thead>
          <tbody>
            {filteredProcs.map((p) => (
              <tr key={p.pid} style={p.is_suspicious ? styles.suspiciousRow : undefined}>
                <td style={{ ...styles.td, color: 'var(--win-accent)', fontFamily: 'var(--win-font-mono)' }}>{p.pid}</td>
                <td style={{ ...styles.td, color: 'var(--win-text-tertiary)' }}>{p.ppid}</td>
                <td style={{ ...styles.td, fontWeight: '600', color: p.is_suspicious ? '#C42B1C' : 'var(--win-text)' }}>
                  {p.name}
                  {p.is_suspicious && <span style={{ marginLeft: '6px', fontSize: '10px', color: '#C42B1C' }}>&#9888; {p.flag_reason}</span>}
                </td>
                <td style={styles.td}>{p.username || 'N/A'}</td>
                <td style={{ ...styles.td, color: 'var(--win-text-secondary)', fontFamily: 'var(--win-font-mono)' }}>{p.create_time}</td>
                <td style={{ ...styles.td, color: 'var(--win-text-secondary)', wordBreak: 'break-all', maxWidth: '300px' }}>{p.command_line || p.path}</td>
                <td style={styles.td}>
                  {p.open_sockets?.map((s, sIdx) => (
                    <div key={sIdx} style={styles.socketItem}>
                      {s.local_addr} &rarr; {s.remote_addr} ({s.state})
                    </div>
                  ))}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

import React, { useState, useEffect } from 'react';
import { useTimeline } from '../hooks/useTimeline';
import { RunSigmaScan, LoadSigmaRuleDirectory, GetSigmaRuleCount } from '../lib/wails';

interface Alert {
  rule_title: string;
  level: string;
  description: string;
  tags: string[];
  log_source: string;
  path: string;
  timestamp: string;
  matched_log: string;
}

export const SigmaViewer: React.FC = () => {
  const { entries, loading: timelineLoading } = useTimeline();
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [ruleCount, setRuleCount] = useState<number>(0);
  const [customPath, setCustomPath] = useState('');
  const [scanning, setScanning] = useState(false);

  useEffect(() => {
    GetSigmaRuleCount().then(setRuleCount).catch(() => setRuleCount(0));
  }, []);

  const handleLoadRules = async () => {
    if (!customPath) return;
    try {
      const count = await LoadSigmaRuleDirectory(customPath);
      setRuleCount(count);
    } catch (err) {
      console.error('Failed to load rules:', err);
    }
  };

  const handleScan = async () => {
    setScanning(true);
    try {
      const res = await RunSigmaScan(entries);
      setAlerts(res || []);
    } catch (err) {
      console.error(err);
    } finally {
      setScanning(false);
    }
  };

  const getLevelColor = (level: string) => {
    switch (level.toLowerCase()) {
      case 'critical': return { bg: '#C42B1C', color: '#fff' };
      case 'high': return { bg: '#D83B01', color: '#fff' };
      case 'medium': return { bg: '#9D5D00', color: '#fff' };
      case 'low': return { bg: '#0078D4', color: '#fff' };
      default: return { bg: 'var(--win-text-tertiary)', color: '#fff' };
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
      fontSize: '18px',
      fontWeight: '600',
      margin: 0,
    },
    controls: {
      display: 'flex',
      gap: '10px',
      alignItems: 'center',
    },
    input: {
      flex: 1,
      padding: '8px 12px',
      backgroundColor: 'var(--win-surface)',
      border: '1px solid var(--win-stroke-strong)',
      borderRadius: 'var(--win-radius-sm)',
      color: 'var(--win-text)',
      fontSize: '13px',
      fontFamily: 'var(--win-font)',
      outline: 'none',
    },
    loadBtn: {
      padding: '8px 16px',
      backgroundColor: 'var(--win-control)',
      color: 'var(--win-text)',
      border: '1px solid var(--win-stroke-strong)',
      borderRadius: 'var(--win-radius-sm)',
      cursor: 'pointer',
      fontSize: '13px',
    },
    scanBtn: {
      padding: '8px 20px',
      backgroundColor: '#C42B1C',
      color: '#FFFFFF',
      border: 'none',
      borderRadius: 'var(--win-radius-sm)',
      cursor: 'pointer',
      fontSize: '13px',
      fontWeight: '600',
    },
    ruleInfo: {
      fontSize: '12px',
      color: 'var(--win-text-secondary)',
      padding: '4px 0',
    },
    alertsContainer: {
      flex: 1,
      overflowY: 'auto' as const,
    },
    emptyState: {
      padding: '40px',
      textAlign: 'center' as const,
      color: 'var(--win-text-tertiary)',
      backgroundColor: 'var(--win-surface)',
      borderRadius: 'var(--win-radius)',
      border: '1px solid var(--win-stroke)',
    },
    alertCard: {
      backgroundColor: 'var(--win-surface)',
      borderLeft: '4px solid',
      padding: '12px 16px',
      marginBottom: '10px',
      borderRadius: 'var(--win-radius-sm)',
      border: '1px solid var(--win-stroke)',
    },
    alertHeader: {
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'center',
      marginBottom: '6px',
    },
    badge: (bg: string, color: string) => ({
      backgroundColor: bg,
      color: color,
      padding: '2px 8px',
      borderRadius: '10px',
      fontSize: '11px',
      fontWeight: '600',
    }),
    alertTitle: {
      fontSize: '14px',
      color: 'var(--win-text)',
      fontWeight: '600',
    },
    alertTime: {
      color: 'var(--win-text-tertiary)',
      fontSize: '12px',
      fontFamily: 'var(--win-font-mono)',
    },
    alertDesc: {
      color: 'var(--win-text-secondary)',
      marginBottom: '8px',
      fontSize: '13px',
    },
    matchedLog: {
      backgroundColor: 'var(--win-bg)',
      padding: '8px 10px',
      fontSize: '11px',
      fontFamily: 'var(--win-font-mono)',
      color: 'var(--win-accent)',
      borderRadius: 'var(--win-radius-sm)',
      marginBottom: '8px',
      wordBreak: 'break-all' as const,
    },
    tags: {
      display: 'flex',
      gap: '6px',
      flexWrap: 'wrap' as const,
    },
    tag: {
      backgroundColor: 'var(--win-control)',
      color: 'var(--win-text-secondary)',
      padding: '2px 8px',
      borderRadius: '10px',
      fontSize: '10px',
    },
  };

  return (
    <div style={styles.container}>
      <h2 style={styles.header}>Sigma Rule Threat Detection Engine</h2>

      <div style={styles.controls}>
        <input
          type="text"
          placeholder="Path to custom Sigma rules folder (e.g. C:\rules)"
          value={customPath}
          onChange={(e) => setCustomPath(e.target.value)}
          style={styles.input}
        />
        <button onClick={handleLoadRules} style={styles.loadBtn}>Load Rules</button>
        <button
          onClick={handleScan}
          disabled={scanning || timelineLoading || entries.length === 0}
          style={{
            ...styles.scanBtn,
            opacity: (scanning || timelineLoading || entries.length === 0) ? 0.6 : 1,
          }}
        >
          {scanning ? 'Scanning...' : `Run Sigma Scan (${ruleCount} Rules)`}
        </button>
      </div>

      <div style={styles.ruleInfo}>
        {ruleCount} active rules loaded | {entries.length} timeline events available
      </div>

      <div style={styles.alertsContainer}>
        {alerts.length === 0 ? (
          <div style={styles.emptyState}>
            {timelineLoading
              ? 'Loading timeline events...'
              : 'No Sigma detections triggered. Build a timeline and run a scan.'}
          </div>
        ) : (
          alerts.map((alert, idx) => {
            const levelColor = getLevelColor(alert.level);
            return (
              <div
                key={idx}
                style={{
                  ...styles.alertCard,
                  borderLeftColor: levelColor.bg,
                }}
              >
                <div style={styles.alertHeader}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                    <span style={styles.badge(levelColor.bg, levelColor.color)}>
                      {alert.level.toUpperCase()}
                    </span>
                    <span style={styles.alertTitle}>{alert.rule_title}</span>
                  </div>
                  <span style={styles.alertTime}>{alert.timestamp}</span>
                </div>
                <div style={styles.alertDesc}>{alert.description}</div>
                <div style={styles.matchedLog}>
                  Matched: {alert.matched_log}
                </div>
                <div style={styles.tags}>
                  {alert.tags?.map((tag, tIdx) => (
                    <span key={tIdx} style={styles.tag}>{tag}</span>
                  ))}
                </div>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
};

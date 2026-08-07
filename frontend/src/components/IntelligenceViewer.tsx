import React, { useState } from 'react';
import { useTimeline } from '../hooks/useTimeline';
import { AnalyzeFindings } from '../lib/wails';

interface Finding {
  level: string;
  category: string;
  title: string;
  description: string;
  source: string;
  path: string;
  formatted_ts: string;
}

export const IntelligenceViewer: React.FC = () => {
  const { entries, loading } = useTimeline();
  const [findings, setFindings] = useState<Finding[]>([]);
  const [analyzing, setAnalyzing] = useState(false);

  const runAnalysis = async () => {
    if (!entries || entries.length === 0) return;
    setAnalyzing(true);
    try {
      const res = await AnalyzeFindings(entries);
      setFindings(res || []);
    } catch (err) {
      console.error(err);
    } finally {
      setAnalyzing(false);
    }
  };

  const getLevelBadge = (level: string) => {
    switch (level) {
      case 'CRITICAL':
        return { bg: '#C42B1C', color: '#fff' };
      case 'HIGH':
        return { bg: '#D83B01', color: '#fff' };
      case 'MEDIUM':
        return { bg: '#9D5D00', color: '#fff' };
      default:
        return { bg: '#0078D4', color: '#fff' };
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
    header: {
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'center',
      marginBottom: '16px',
    },
    title: {
      fontSize: '18px',
      fontWeight: '600',
      color: 'var(--win-text)',
      margin: 0,
    },
    button: {
      padding: '8px 20px',
      backgroundColor: 'var(--win-accent)',
      color: '#FFFFFF',
      border: 'none',
      borderRadius: 'var(--win-radius-sm)',
      cursor: 'pointer',
      fontSize: '13px',
      fontWeight: '500',
      fontFamily: 'var(--win-font)',
    },
    buttonDisabled: {
      opacity: 0.6,
      cursor: 'not-allowed',
    },
    findingsContainer: {
      overflowY: 'auto' as const,
      maxHeight: 'calc(100vh - 180px)',
    },
    emptyState: {
      padding: '40px',
      textAlign: 'center' as const,
      color: 'var(--win-text-tertiary)',
      backgroundColor: 'var(--win-surface)',
      borderRadius: 'var(--win-radius)',
      border: '1px solid var(--win-stroke)',
    },
    findingCard: {
      backgroundColor: 'var(--win-surface)',
      borderLeft: '4px solid',
      padding: '12px 16px',
      marginBottom: '10px',
      borderRadius: 'var(--win-radius-sm)',
      border: '1px solid var(--win-stroke)',
    },
    findingHeader: {
      display: 'flex',
      alignItems: 'center',
      gap: '10px',
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
    category: {
      color: 'var(--win-accent)',
      fontWeight: '600',
      fontSize: '13px',
    },
    findingTitle: {
      fontSize: '14px',
      color: 'var(--win-text)',
      fontWeight: '600',
    },
    findingTime: {
      marginLeft: 'auto',
      color: 'var(--win-text-tertiary)',
      fontSize: '12px',
      fontFamily: 'var(--win-font-mono)',
    },
    findingDesc: {
      color: 'var(--win-text-secondary)',
      marginBottom: '6px',
      fontSize: '13px',
    },
    findingMeta: {
      color: 'var(--win-text-tertiary)',
      fontSize: '11px',
      fontFamily: 'var(--win-font-mono)',
    },
  };

  return (
    <div style={styles.container}>
      <div style={styles.header}>
        <h2 style={styles.title}>Automated Forensic Intelligence & Anomaly Engine</h2>
        <button
          onClick={runAnalysis}
          disabled={analyzing || loading || entries.length === 0}
          style={{
            ...styles.button,
            ...(analyzing || loading || entries.length === 0 ? styles.buttonDisabled : {}),
          }}
        >
          {analyzing ? 'Scanning...' : `Scan Timeline (${entries.length} Events)`}
        </button>
      </div>

      <div style={styles.findingsContainer}>
        {findings.length === 0 ? (
          <div style={styles.emptyState}>
            No findings generated. Build a timeline first, then run a scan to detect forensic anomalies.
          </div>
        ) : (
          findings.map((finding, idx) => {
            const badge = getLevelBadge(finding.level);
            return (
              <div
                key={idx}
                style={{
                  ...styles.findingCard,
                  borderLeftColor: badge.bg,
                }}
              >
                <div style={styles.findingHeader}>
                  <span style={styles.badge(badge.bg, badge.color)}>
                    {finding.level}
                  </span>
                  <span style={styles.category}>[{finding.category}]</span>
                  <span style={styles.findingTitle}>{finding.title}</span>
                  <span style={styles.findingTime}>{finding.formatted_ts}</span>
                </div>
                <div style={styles.findingDesc}>{finding.description}</div>
                <div style={styles.findingMeta}>
                  Source: {finding.source} | Path: {finding.path}
                </div>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
};

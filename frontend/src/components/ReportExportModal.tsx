import React, { useState } from 'react';
import { ExportChainOfCustodyReport } from '../lib/wails';

interface Props {
  caseDir: string;
  isOpen: boolean;
  onClose: () => void;
}

export const ReportExportModal: React.FC<Props> = ({ caseDir, isOpen, onClose }) => {
  const [examinerName, setExaminerName] = useState('Lead Examiner');
  const [outputPath, setOutputPath] = useState(`${caseDir}/reports/chain_of_custody.html`);
  const [isExporting, setIsExporting] = useState(false);
  const [exportMessage, setExportMessage] = useState<string | null>(null);

  if (!isOpen) return null;

  const handleExport = async () => {
    setIsExporting(true);
    setExportMessage(null);
    try {
      const res = await ExportChainOfCustodyReport({
        caseDir,
        examinerName,
        outputPath,
      });
      if (res.success) {
        setExportMessage(`Report generated: ${res.reportPath}`);
      } else {
        setExportMessage(`Error: ${res.errorMessage}`);
      }
    } catch (err: any) {
      setExportMessage(`Export failed: ${err?.toString() || err}`);
    } finally {
      setIsExporting(false);
    }
  };

  const styles = {
    overlay: {
      position: 'fixed' as const,
      top: 0, left: 0, right: 0, bottom: 0,
      backgroundColor: 'rgba(0,0,0,0.5)',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      zIndex: 1000,
    },
    modal: {
      backgroundColor: 'var(--win-surface)',
      border: '1px solid var(--win-stroke)',
      borderRadius: 'var(--win-radius)',
      padding: '24px',
      width: '480px',
      boxShadow: 'var(--win-shadow-flyout)',
    },
    title: {
      margin: '0 0 16px 0',
      fontSize: '16px',
      fontWeight: '600',
      color: 'var(--win-text)',
    },
    field: {
      marginBottom: '14px',
    },
    label: {
      display: 'block',
      fontSize: '12px',
      color: 'var(--win-text-secondary)',
      marginBottom: '4px',
    },
    input: {
      width: '100%',
      padding: '8px 12px',
      backgroundColor: 'var(--win-bg)',
      border: '1px solid var(--win-stroke-strong)',
      borderRadius: 'var(--win-radius-sm)',
      color: 'var(--win-text)',
      fontSize: '13px',
      fontFamily: 'var(--win-font)',
      outline: 'none',
      boxSizing: 'border-box' as const,
    },
    message: (success: boolean) => ({
      padding: '10px',
      marginBottom: '14px',
      borderRadius: 'var(--win-radius-sm)',
      fontSize: '12px',
      backgroundColor: success ? '#E8F5E9' : '#FDE7E9',
      color: success ? '#0F7B0F' : '#C42B1C',
    }),
    btnRow: {
      display: 'flex',
      justifyContent: 'flex-end',
      gap: '10px',
    },
  };

  const isSuccess = exportMessage?.startsWith('Report generated');

  return (
    <div style={styles.overlay} onClick={onClose}>
      <div style={styles.modal} onClick={e => e.stopPropagation()}>
        <h3 style={styles.title}>Export Chain of Custody Report</h3>

        <div style={styles.field}>
          <label style={styles.label}>Examiner / Analyst Name:</label>
          <input value={examinerName} onChange={e => setExaminerName(e.target.value)} style={styles.input} />
        </div>
        <div style={styles.field}>
          <label style={styles.label}>Output HTML Path:</label>
          <input value={outputPath} onChange={e => setOutputPath(e.target.value)} style={styles.input} />
        </div>

        {exportMessage && (
          <div style={styles.message(isSuccess || false)}>{exportMessage}</div>
        )}

        <div style={styles.btnRow}>
          <button onClick={onClose} style={{ padding: '8px 16px', backgroundColor: 'var(--win-control)', color: 'var(--win-text)', border: 'none', borderRadius: 'var(--win-radius-sm)', cursor: 'pointer', fontSize: '13px' }}>
            Close
          </button>
          <button onClick={handleExport} disabled={isExporting} style={{ padding: '8px 16px', backgroundColor: 'var(--win-accent)', color: '#fff', border: 'none', borderRadius: 'var(--win-radius-sm)', cursor: 'pointer', fontSize: '13px', fontWeight: '600' }}>
            {isExporting ? 'Generating...' : 'Generate Report'}
          </button>
        </div>
      </div>
    </div>
  );
};

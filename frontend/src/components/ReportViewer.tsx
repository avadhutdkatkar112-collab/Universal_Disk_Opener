import React, { useState } from 'react';
import { useTimeline } from '../hooks/useTimeline';
import { GenerateReport, SaveReportToFile } from '../lib/wails';

export const ReportViewer: React.FC = () => {
  const { entries } = useTimeline();
  const [caseName, setCaseName] = useState('INCIDENT-2026-08A');
  const [investigator, setInvestigator] = useState('Lead Forensic Examiner');
  const [savePath, setSavePath] = useState('C:\\report_output.html');
  const [htmlReport, setHtmlReport] = useState('');
  const [loading, setLoading] = useState(false);

  const handleGenerate = async () => {
    setLoading(true);
    try {
      const html = await GenerateReport(caseName, investigator, entries || [], [], [], []);
      setHtmlReport(html);
    } catch (err) {
      console.error('Failed to generate report:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    if (!htmlReport || !savePath) return;
    try {
      await SaveReportToFile(savePath, htmlReport);
      alert('Report saved to ' + savePath);
    } catch (err) {
      alert('Save failed: ' + err);
    }
  };

  const handlePrintPDF = () => {
    const w = window.open('', '_blank');
    if (w) {
      w.document.write(htmlReport);
      w.document.close();
      w.focus();
      w.print();
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
    title: {
      fontSize: '18px',
      fontWeight: '600',
      margin: 0,
    },
    controls: {
      display: 'flex',
      gap: '16px',
      alignItems: 'flex-end',
      backgroundColor: 'var(--win-surface)',
      padding: '16px',
      borderRadius: 'var(--win-radius)',
      border: '1px solid var(--win-stroke)',
    },
    field: {
      display: 'flex',
      flexDirection: 'column' as const,
      gap: '4px',
    },
    label: {
      fontSize: '12px',
      color: 'var(--win-text-secondary)',
      fontWeight: '500',
    },
    input: {
      padding: '8px 12px',
      backgroundColor: 'var(--win-bg)',
      border: '1px solid var(--win-stroke-strong)',
      borderRadius: 'var(--win-radius-sm)',
      color: 'var(--win-text)',
      fontSize: '13px',
      fontFamily: 'var(--win-font)',
      outline: 'none',
    },
    btnGroup: {
      display: 'flex',
      gap: '8px',
    },
    generateBtn: {
      padding: '8px 20px',
      backgroundColor: 'var(--win-accent)',
      color: '#FFFFFF',
      border: 'none',
      borderRadius: 'var(--win-radius-sm)',
      cursor: 'pointer',
      fontSize: '13px',
      fontWeight: '600',
    },
    saveBtn: {
      padding: '8px 16px',
      backgroundColor: 'var(--win-success)',
      color: '#FFFFFF',
      border: 'none',
      borderRadius: 'var(--win-radius-sm)',
      cursor: 'pointer',
      fontSize: '13px',
      fontWeight: '500',
    },
    printBtn: {
      padding: '8px 16px',
      backgroundColor: '#9D5D00',
      color: '#FFFFFF',
      border: 'none',
      borderRadius: 'var(--win-radius-sm)',
      cursor: 'pointer',
      fontSize: '13px',
      fontWeight: '500',
    },
    preview: {
      flex: 1,
      border: '1px solid var(--win-stroke)',
      borderRadius: 'var(--win-radius)',
      overflow: 'hidden',
    },
    emptyState: {
      flex: 1,
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      color: 'var(--win-text-tertiary)',
      border: '2px dashed var(--win-stroke)',
      borderRadius: 'var(--win-radius)',
      padding: '40px',
      textAlign: 'center' as const,
    },
  };

  return (
    <div style={styles.container}>
      <h2 style={styles.title}>Executive Forensic Report Generator</h2>

      <div style={styles.controls}>
        <div style={styles.field}>
          <label style={styles.label}>Case Identifier</label>
          <input value={caseName} onChange={e => setCaseName(e.target.value)} style={styles.input} />
        </div>
        <div style={styles.field}>
          <label style={styles.label}>Examiner Name</label>
          <input value={investigator} onChange={e => setInvestigator(e.target.value)} style={styles.input} />
        </div>
        <div style={styles.field}>
          <label style={styles.label}>Save Path</label>
          <input value={savePath} onChange={e => setSavePath(e.target.value)} style={styles.input} />
        </div>
        <div style={styles.btnGroup}>
          <button onClick={handleGenerate} disabled={loading} style={styles.generateBtn}>
            {loading ? 'Compiling...' : 'Generate Report'}
          </button>
          {htmlReport && (
            <>
              <button onClick={handleSave} style={styles.saveBtn}>Save HTML</button>
              <button onClick={handlePrintPDF} style={styles.printBtn}>Export PDF</button>
            </>
          )}
        </div>
      </div>

      {htmlReport ? (
        <div style={styles.preview}>
          <iframe
            title="Report Preview"
            srcDoc={htmlReport}
            style={{ width: '100%', height: '100%', border: 'none', backgroundColor: '#fff' }}
          />
        </div>
      ) : (
        <div style={styles.emptyState}>
          Configure case parameters and click <strong>Generate Report</strong> to assemble the executive summary.
        </div>
      )}
    </div>
  );
};

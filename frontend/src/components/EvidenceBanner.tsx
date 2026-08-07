import React from 'react';

interface Props {
  caseId: string;
  sourceHash: string;
  isVerified: boolean;
}

export const EvidenceBanner: React.FC<Props> = ({ caseId, sourceHash, isVerified }) => {
  return (
    <div style={{
      backgroundColor: isVerified ? '#E8F5E9' : '#FDE7E9',
      borderBottom: `2px solid ${isVerified ? '#0F7B0F' : '#C42B1C'}`,
      padding: '8px 20px',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between',
      fontSize: '12px',
      fontFamily: 'var(--win-font-mono)',
    }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
        <span style={{
          backgroundColor: isVerified ? '#0F7B0F' : '#C42B1C',
          color: '#fff',
          padding: '3px 10px',
          borderRadius: '10px',
          fontWeight: '600',
          fontSize: '11px',
        }}>
          EVIDENCE MODE ({isVerified ? 'READ-ONLY & VERIFIED' : 'UNVERIFIED'})
        </span>
        <span style={{ color: 'var(--win-text)' }}><strong>Case:</strong> {caseId}</span>
        <span style={{ color: 'var(--win-text-secondary)' }}>
          <strong>SHA-256:</strong>{' '}
          <code style={{ fontSize: '11px' }}>{sourceHash ? `${sourceHash.slice(0, 24)}...` : 'N/A'}</code>
        </span>
      </div>

      <div style={{ color: isVerified ? '#0F7B0F' : '#C42B1C', fontWeight: '500' }}>
        {isVerified ? 'Chain of Custody Intact' : 'Integrity Verification Required'}
      </div>
    </div>
  );
};

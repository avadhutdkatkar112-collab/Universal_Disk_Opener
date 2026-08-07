import { useState, useCallback } from 'react';
import { GetRegistryArtifacts } from '../lib/wails';

interface ArtifactViewerProps {
  diskPath: string;
  onClose: () => void;
}

interface ArtifactData {
  header?: {
    signature: string;
    primary_seq: number;
    secondary_seq: number;
    last_modified: string;
    major_version: number;
    minor_version: number;
    root_cell_offset: number;
    data_size: number;
    filename: string;
  };
  system_info?: {
    computer_name: string;
    os_version: string;
    current_build: string;
    install_date: string;
    registered_owner: string;
  };
  run_keys?: Array<{
    key_path: string;
    value_name: string;
    value_data: string;
    data_type: string;
    timestamp: string;
  }>;
  services?: Array<{
    name: string;
    image_path: string;
    start_type: number;
  }>;
  installed_software?: Array<{
    name: string;
    version: string;
    publisher: string;
  }>;
}

export default function ArtifactViewer({ diskPath: _diskPath, onClose }: ArtifactViewerProps) {
  const [hivePath, setHivePath] = useState<string>('');
  const [hiveType, setHiveType] = useState<string>('SYSTEM');
  const [artifacts, setArtifacts] = useState<ArtifactData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'overview' | 'keys' | 'values'>('overview');

  const handleParse = useCallback(async () => {
    if (!hivePath) return;
    
    setLoading(true);
    setError(null);
    try {
      const result = await GetRegistryArtifacts(hivePath, hiveType);
      setArtifacts(result as ArtifactData);
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }, [hivePath, hiveType]);

  return (
    <div style={{
      position: 'fixed', top: 32, right: 0, bottom: 24, width: 700,
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
            Registry Artifact Analyzer
          </div>
          <div style={{ fontSize: 11, color: 'var(--win-text-tertiary)' }}>
            Parse and analyze Windows Registry hive files
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

      {/* Input Controls */}
      <div style={{
        padding: '12px', borderBottom: '1px solid var(--win-stroke)',
        display: 'flex', gap: 8, flexShrink: 0,
      }}>
        <input
          type="text"
          placeholder="Path to registry hive (e.g., /Windows/System32/config/SYSTEM)"
          value={hivePath}
          onChange={e => setHivePath(e.target.value)}
          style={{
            flex: 1, padding: '6px 10px', fontSize: 12,
            border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
            background: 'var(--win-bg)', color: 'var(--win-text)',
            fontFamily: 'var(--win-font-mono)',
          }}
        />
        <select
          value={hiveType}
          onChange={e => setHiveType(e.target.value)}
          style={{
            padding: '6px 10px', fontSize: 12,
            border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
            background: 'var(--win-bg)', color: 'var(--win-text)',
          }}
        >
          <option value="SYSTEM">SYSTEM</option>
          <option value="SOFTWARE">SOFTWARE</option>
          <option value="SAM">SAM</option>
          <option value="NTUSER.DAT">NTUSER.DAT</option>
        </select>
        <button
          onClick={handleParse}
          disabled={loading || !hivePath}
          style={{
            padding: '6px 16px', fontSize: 12, fontWeight: 500,
            border: '1px solid var(--win-accent)', borderRadius: 'var(--win-radius-sm)',
            background: loading ? 'var(--win-stroke)' : 'var(--win-accent)',
            color: 'white', cursor: loading ? 'wait' : 'pointer',
          }}
        >
          {loading ? 'Parsing...' : 'Analyze'}
        </button>
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

      {/* Tabs */}
      {artifacts && (
        <div style={{ display: 'flex', gap: 4, padding: '8px 12px', borderBottom: '1px solid var(--win-stroke)' }}>
          {(['overview', 'keys', 'values'] as const).map(tab => (
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
              {tab.charAt(0).toUpperCase() + tab.slice(1)}
            </button>
          ))}
        </div>
      )}

      {/* Content */}
      <div style={{ flex: 1, overflow: 'auto', padding: 12 }}>
        {!artifacts && !loading && (
          <div style={{
            textAlign: 'center', padding: 40, color: 'var(--win-text-tertiary)',
          }}>
            <div style={{ fontSize: 13, marginBottom: 8 }}>
              Enter a path to a Windows Registry hive file
            </div>
            <div style={{ fontSize: 11 }}>
              Common locations:<br/>
              • SYSTEM: /Windows/System32/config/SYSTEM<br/>
              • SOFTWARE: /Windows/System32/config/SOFTWARE<br/>
              • SAM: /Windows/System32/config/SAM<br/>
              • NTUSER.DAT: /Users/[username]/NTUSER.DAT
            </div>
          </div>
        )}

        {loading && (
          <div style={{
            textAlign: 'center', padding: 40, color: 'var(--win-text-tertiary)',
          }}>
            <div style={{ fontSize: 13 }}>Parsing registry hive...</div>
          </div>
        )}

        {artifacts && activeTab === 'overview' && (
          <div>
            {/* Header Info */}
            {artifacts.header && (
              <div style={{
                background: 'var(--win-bg)', border: '1px solid var(--win-stroke)',
                borderRadius: 'var(--win-radius-sm)', padding: 12, marginBottom: 12,
              }}>
                <div style={{ fontSize: 12, fontWeight: 500, color: 'var(--win-text)', marginBottom: 8 }}>
                  Registry Header
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8, fontSize: 11 }}>
                  <div><span style={{ color: 'var(--win-text-tertiary)' }}>Signature:</span> {artifacts.header.signature}</div>
                  <div><span style={{ color: 'var(--win-text-tertiary)' }}>Version:</span> {artifacts.header.major_version}.{artifacts.header.minor_version}</div>
                  <div><span style={{ color: 'var(--win-text-tertiary)' }}>Last Modified:</span> {artifacts.header.last_modified}</div>
                  <div><span style={{ color: 'var(--win-text-tertiary)' }}>Data Size:</span> {(artifacts.header.data_size / 1024 / 1024).toFixed(1)} MB</div>
                </div>
              </div>
            )}

            {/* System Info */}
            {artifacts.system_info && (
              <div style={{
                background: 'var(--win-bg)', border: '1px solid var(--win-stroke)',
                borderRadius: 'var(--win-radius-sm)', padding: 12, marginBottom: 12,
              }}>
                <div style={{ fontSize: 12, fontWeight: 500, color: 'var(--win-text)', marginBottom: 8 }}>
                  System Information
                </div>
                <div style={{ fontSize: 11 }}>
                  <div><span style={{ color: 'var(--win-text-tertiary)' }}>Computer Name:</span> {artifacts.system_info.computer_name || 'N/A'}</div>
                  <div><span style={{ color: 'var(--win-text-tertiary)' }}>OS Version:</span> {artifacts.system_info.os_version || 'N/A'}</div>
                  <div><span style={{ color: 'var(--win-text-tertiary)' }}>Build:</span> {artifacts.system_info.current_build || 'N/A'}</div>
                  <div><span style={{ color: 'var(--win-text-tertiary)' }}>Registered Owner:</span> {artifacts.system_info.registered_owner || 'N/A'}</div>
                </div>
              </div>
            )}

            {/* Run Keys */}
            {artifacts.run_keys && artifacts.run_keys.length > 0 && (
              <div style={{
                background: 'var(--win-bg)', border: '1px solid var(--win-stroke)',
                borderRadius: 'var(--win-radius-sm)', padding: 12,
              }}>
                <div style={{ fontSize: 12, fontWeight: 500, color: 'var(--win-text)', marginBottom: 8 }}>
                  Autostart Entries ({artifacts.run_keys.length})
                </div>
                <div style={{ maxHeight: 200, overflow: 'auto' }}>
                  {artifacts.run_keys.map((entry, idx) => (
                    <div key={idx} style={{
                      padding: '4px 0', borderBottom: '1px solid var(--win-stroke)',
                      fontSize: 10,
                    }}>
                      <div style={{ color: 'var(--win-accent)' }}>{entry.value_name}</div>
                      <div style={{ color: 'var(--win-text-secondary)' }}>{entry.value_data}</div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        {artifacts && activeTab === 'keys' && (
          <div>
            <div style={{ fontSize: 12, color: 'var(--win-text-secondary)', marginBottom: 12 }}>
              Use the overview tab to navigate registry keys
            </div>
          </div>
        )}

        {artifacts && activeTab === 'values' && (
          <div>
            <div style={{ fontSize: 12, color: 'var(--win-text-secondary)', marginBottom: 12 }}>
              Registry values are displayed in the overview tab
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

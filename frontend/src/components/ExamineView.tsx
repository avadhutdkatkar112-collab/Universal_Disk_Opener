import React, { useState, useEffect, useMemo, useCallback } from 'react';
import { useEvidenceStore } from '../store/evidenceStore';
import { ReadFileChunk, ReadFileText } from '../lib/wails';
import CodeMirror from '@uiw/react-codemirror';
import { EditorView } from '@codemirror/view';
import { json } from '@codemirror/lang-json';
import { javascript } from '@codemirror/lang-javascript';
import { xml } from '@codemirror/lang-xml';
import { python } from '@codemirror/lang-python';
import { yaml } from '@codemirror/lang-yaml';
import { html } from '@codemirror/lang-html';
import { css } from '@codemirror/lang-css';
import { markdown } from '@codemirror/lang-markdown';
import { sql } from '@codemirror/lang-sql';
import { cpp } from '@codemirror/lang-cpp';
import { java } from '@codemirror/lang-java';
import { go } from '@codemirror/lang-go';
import { rust } from '@codemirror/lang-rust';
import { php } from '@codemirror/lang-php';
import { StreamLanguage } from '@codemirror/language';
import { shell } from '@codemirror/legacy-modes/mode/shell';
import { properties } from '@codemirror/legacy-modes/mode/properties';
import { dockerFile } from '@codemirror/legacy-modes/mode/dockerfile';
import { nginx } from '@codemirror/legacy-modes/mode/nginx';
import { toml } from '@codemirror/legacy-modes/mode/toml';

const TEXT_EXTENSIONS: Record<string, string> = {
  '.txt': 'text', '.log': 'text', '.cfg': 'text', '.conf': 'text',
  '.ini': 'properties', '.properties': 'properties', '.env': 'text', '.rc': 'text',
  '.md': 'markdown', '.markdown': 'markdown', '.mdx': 'markdown',
  '.json': 'json', '.jsonc': 'json', '.jsonl': 'json',
  '.js': 'javascript', '.jsx': 'javascript', '.ts': 'javascript', '.tsx': 'javascript',
  '.mjs': 'javascript', '.cjs': 'javascript',
  '.html': 'html', '.htm': 'html', '.xhtml': 'html',
  '.css': 'css', '.scss': 'css', '.less': 'css',
  '.xml': 'xml', '.xsl': 'xml', '.xsd': 'xml', '.svg': 'xml',
  '.py': 'python', '.pyw': 'python',
  '.yaml': 'yaml', '.yml': 'yaml',
  '.sql': 'sql',
  '.c': 'cpp', '.h': 'cpp', '.cpp': 'cpp', '.hpp': 'cpp', '.cc': 'cpp', '.cxx': 'cpp',
  '.java': 'java',
  '.go': 'go',
  '.rs': 'rust',
  '.php': 'php',
  '.sh': 'shell', '.bash': 'shell', '.zsh': 'shell', '.fish': 'shell',
  '.hcl': 'shell', '.tf': 'shell', '.tfvars': 'shell', '.nomad': 'shell',
  '.tfstate': 'json',
  '.dockerfile': 'dockerfile', '.docker': 'dockerfile',
  '.csv': 'text', '.tsv': 'text',
  '.proto': 'shell',
  '.graphql': 'text', '.gql': 'text',
  '.toml': 'toml',
  '.conf.nginx': 'nginx', '.nginx': 'nginx',
};

const TEXT_BASENAMES: Record<string, string> = {
  'makefile': 'text', 'cmakefile': 'text', 'dockerfile': 'dockerfile',
  'vagrantfile': 'ruby', 'gemfile': 'ruby', 'rakefile': 'ruby',
  'procfile': 'text', 'brewfile': 'text',
  'license': 'text', 'readme': 'text', 'changelog': 'text', 'authors': 'text',
  '.gitignore': 'text', '.gitattributes': 'text', '.editorconfig': 'text',
  '.dockerignore': 'text', '.eslintrc': 'javascript', '.prettierrc': 'json',
  '.babelrc': 'json', '.npmrc': 'text', '.nvmrc': 'text',
  'todo': 'text', 'copying': 'text', 'install': 'text',
};

function getLanguageExtension(ext: string, baseName: string) {
  const lang = TEXT_EXTENSIONS[ext] || TEXT_EXTENSIONS[ext.toLowerCase()]
    || TEXT_BASENAMES[baseName.toLowerCase()];
  switch (lang) {
    case 'json': return json();
    case 'javascript': return javascript();
    case 'html': return html();
    case 'css': return css();
    case 'xml': return xml();
    case 'python': return python();
    case 'yaml': return yaml();
    case 'sql': return sql();
    case 'cpp': return cpp();
    case 'java': return java();
    case 'go': return go();
    case 'rust': return rust();
    case 'php': return php();
    case 'markdown': return markdown();
    case 'shell': return StreamLanguage.define(shell);
    case 'properties': return StreamLanguage.define(properties);
    case 'dockerfile': return StreamLanguage.define(dockerFile);
    case 'nginx': return StreamLanguage.define(nginx);
    case 'toml': return StreamLanguage.define(toml);
    default: return [];
  }
}

function isTextFile(filePath: string): boolean {
  const baseName = filePath.split('/').pop() || '';
  const dotIdx = baseName.lastIndexOf('.');
  if (dotIdx > 0) {
    const ext = baseName.substring(dotIdx).toLowerCase();
    if (ext in TEXT_EXTENSIONS) return true;
  }
  if (baseName.toLowerCase() in TEXT_BASENAMES) return true;
  return false;
}

function getFileExt(filePath: string): string {
  const baseName = filePath.split('/').pop() || '';
  const dotIdx = baseName.lastIndexOf('.');
  return dotIdx === -1 ? '' : baseName.substring(dotIdx).toLowerCase();
}

function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

function getFileIcon(name: string, isDir: boolean): string {
  if (isDir) return '\u{1F4C1}';
  const ext = getFileExt(name);
  const base = name.toLowerCase();
  if (base === 'readme' || base === 'readme.md' || base === 'readme.txt') return '\u{1F4DD}';
  if (base === 'license' || base === 'license.md') return '\u{1F511}';
  if (base === 'changelog' || base === 'changelog.md') return '\u{1F4CB}';
  if (base.startsWith('makefile') || base.startsWith('cmake')) return '\u2699\uFE0F';
  if (base.includes('docker')) return '\u{1F433}';
  if (base.includes('.git')) return '\u{1F500}';
  if (ext === '.json') return '\u{1F4CB}';
  if (ext === '.yaml' || ext === '.yml') return '\u{1F4CB}';
  if (ext === '.md' || ext === '.markdown') return '\u{1F4DD}';
  if (ext === '.log') return '\u{1F4DC}';
  if (ext === '.js' || ext === '.ts' || ext === '.jsx' || ext === '.tsx') return '\u{1F4BB}';
  if (ext === '.py') return '\u{1F40D}';
  if (ext === '.go') return '\u{1F48E}';
  if (ext === '.rs') return '\u{1F980}';
  if (ext === '.html' || ext === '.css') return '\u{1F3A8}';
  if (ext === '.sh' || ext === '.bash') return '\u{1F4BB}';
  if (ext === '.xml') return '\u{1F4F0}';
  if (ext === '.sql') return '\u{1F5C4}';
  if (ext === '.java') return '\u2615';
  if (ext === '.php') return '\u{1F35A}';
  if (ext === '.c' || ext === '.h' || ext === '.cpp') return '\u{1F527}';
  if (ext === '.save' || ext === '.bak' || ext === '.old') return '\u{1F4BE}';
  return '\u{1F4C4}';
}

interface HexLine {
  offset: string;
  bytes: string[];
  ascii: string;
}

const darkTheme = EditorView.theme({
  '&': { backgroundColor: '#1e1e2e', color: '#cdd6f4', fontSize: '13px', height: '100%' },
  '.cm-content': { caretColor: '#f5e0dc', fontFamily: 'var(--win-font-mono, monospace)' },
  '.cm-cursor': { borderLeftColor: '#f5e0dc' },
  '.cm-selectionBackground': { backgroundColor: '#45475a !important' },
  '&.cm-focused .cm-selectionBackground': { backgroundColor: '#45475a !important' },
  '.cm-gutters': { backgroundColor: '#181825', color: '#6c7086', borderRight: '1px solid #313244' },
  '.cm-activeLineGutter': { backgroundColor: '#313244' },
  '.cm-activeLine': { backgroundColor: '#1e1e2e' },
  '.cm-matchingBracket': { backgroundColor: '#45475a', outline: '1px solid #6c7086' },
}, { dark: true });

const spinKeyframes = `@keyframes examineSpin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }`;
if (typeof document !== 'undefined' && !document.getElementById('examine-spin-style')) {
  const style = document.createElement('style');
  style.id = 'examine-spin-style';
  style.textContent = spinKeyframes;
  document.head.appendChild(style);
}

export const ExamineView: React.FC = () => {
  const session = useEvidenceStore();
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [hexData, setHexData] = useState<HexLine[]>([]);
  const [textContent, setTextContent] = useState<string | null>(null);
  const [isTextMode, setIsTextMode] = useState(false);
  const [inspectorOffset, setInspectorOffset] = useState(0);
  const [hoveredRow, setHoveredRow] = useState<number | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fileSize, setFileSize] = useState<number>(0);

  useEffect(() => {
    if (session.examineFilePath) {
      setSelectedFile(session.examineFilePath);
    }
  }, [session.examineFilePath]);

  useEffect(() => {
    if (selectedFile) {
      const isText = isTextFile(selectedFile);
      setIsTextMode(isText);
      setHexData([]);
      setTextContent(null);
      setError(null);
      setInspectorOffset(0);
      setFileSize(0);
      setHoveredRow(null);
      if (isText) {
        loadTextContent(selectedFile);
      } else {
        loadHexData(selectedFile);
      }
    }
  }, [selectedFile]);

  const loadHexData = async (path: string) => {
    setLoading(true);
    setError(null);
    try {
      const chunk = await ReadFileChunk(path, 0, 4096);
      const bytes = Array.from(chunk);
      setFileSize(bytes.length);
      const lines: HexLine[] = [];
      for (let i = 0; i < bytes.length; i += 16) {
        const slice = bytes.slice(i, i + 16);
        const hexBytes = slice.map(b => b.toString(16).toUpperCase().padStart(2, '0'));
        const ascii = slice.map(b => (b >= 32 && b < 127 ? String.fromCharCode(b) : '.')).join('');
        lines.push({
          offset: i.toString(16).toUpperCase().padStart(8, '0'),
          bytes: hexBytes,
          ascii,
        });
      }
      setHexData(lines);
    } catch (err: any) {
      setError(err?.toString() || 'Failed to read file');
      setHexData([]);
    } finally {
      setLoading(false);
    }
  };

  const loadTextContent = async (path: string) => {
    setLoading(true);
    setError(null);
    try {
      const text = await ReadFileText(path);
      setTextContent(text);
      setFileSize(new TextEncoder().encode(text).length);
    } catch (err: any) {
      setError(err?.toString() || 'Failed to read file');
      setTextContent(null);
    } finally {
      setLoading(false);
    }
  };

  const handleEntryClick = useCallback((node: any) => {
    if (node.isDir) {
      session.navigateTo(node.path);
      setSelectedFile(null);
      setHexData([]);
      setTextContent(null);
      setError(null);
      setFileSize(0);
    } else {
      setSelectedFile(node.path);
    }
  }, [session]);

  const getUInt16 = useCallback((offset: number) => {
    if (offset + 2 <= hexData.length * 16) {
      const lineIdx = Math.floor(offset / 16);
      const byteIdx = offset % 16;
      const b1 = parseInt(hexData[lineIdx]?.bytes[byteIdx] || '0', 16);
      const b2 = parseInt(hexData[lineIdx]?.bytes[byteIdx + 1] || '0', 16);
      return b1 | (b2 << 8);
    }
    return 0;
  }, [hexData]);

  const getUInt32 = useCallback((offset: number) => {
    return getUInt16(offset) | (getUInt16(offset + 2) << 16);
  }, [getUInt16]);

  const formatTimestamp = (ts: number) => {
    if (ts === 0) return 'N/A';
    try {
      const d = new Date(ts * 1000);
      if (isNaN(d.getTime())) return 'Invalid';
      return d.toISOString();
    } catch {
      return 'Invalid';
    }
  };

  const languageExtensions = useMemo(() => {
    if (!selectedFile) return [];
    const ext = getFileExt(selectedFile);
    const baseName = selectedFile.split('/').pop() || '';
    const lang = getLanguageExtension(ext, baseName);
    return Array.isArray(lang) ? lang : [lang];
  }, [selectedFile]);

  const folderName = useMemo(() => {
    if (!session.currentPath || session.currentPath === '/') return 'Root';
    const parts = session.currentPath.replace(/\/+$/, '').split('/').filter(Boolean);
    return parts[parts.length - 1] || 'Root';
  }, [session.currentPath]);

  if (!session.isActive) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'var(--win-text-secondary)', fontSize: '13px' }}>
        Open evidence and select a file to examine.
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', height: '100%', backgroundColor: 'var(--win-bg)', color: 'var(--win-text)', fontFamily: 'var(--win-font)' }}>
      {/* File browser sidebar */}
      <div style={{ width: '220px', borderRight: '1px solid var(--win-stroke)', overflow: 'auto', display: 'flex', flexDirection: 'column', flexShrink: 0 }}>
        {/* Directory header */}
        <div style={{ padding: '8px 12px 6px', borderBottom: '1px solid var(--win-stroke)', flexShrink: 0 }}>
          <div style={{ fontSize: '10px', fontWeight: '600', color: 'var(--win-text-tertiary)', textTransform: 'uppercase', letterSpacing: '0.5px', marginBottom: '2px' }}>
            Browsing
          </div>
          <div style={{ fontSize: '12px', fontWeight: '600', color: 'var(--win-text)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={session.currentPath}>
            {session.currentPath === '/' ? '/' : `${folderName}/`}
          </div>
        </div>

        <div style={{ flex: 1, overflow: 'auto', padding: '4px' }}>
          {/* Parent directory */}
          {session.currentPath !== '/' && (
            <div
              onClick={() => session.navigateUp()}
              style={{
                padding: '5px 8px',
                fontSize: '12px',
                cursor: 'pointer',
                borderRadius: 'var(--win-radius-sm)',
                color: 'var(--win-text-secondary)',
                marginBottom: '2px',
                display: 'flex',
                alignItems: 'center',
                gap: '6px',
              }}
              onMouseEnter={e => (e.currentTarget.style.backgroundColor = 'var(--win-subtle-hover)')}
              onMouseLeave={e => (e.currentTarget.style.backgroundColor = 'transparent')}
            >
              <span style={{ fontSize: '10px', opacity: 0.6 }}>&#9664;</span> ..
            </div>
          )}

          {/* Directories */}
          {session.currentNodes.filter(n => n.isDir).map((node, idx) => (
            <div
              key={`d-${idx}`}
              onClick={() => handleEntryClick(node)}
              style={{
                padding: '5px 8px',
                fontSize: '12px',
                cursor: 'pointer',
                borderRadius: 'var(--win-radius-sm)',
                color: 'var(--win-accent)',
                display: 'flex',
                alignItems: 'center',
                gap: '6px',
              }}
              onMouseEnter={e => (e.currentTarget.style.backgroundColor = 'var(--win-subtle-hover)')}
              onMouseLeave={e => (e.currentTarget.style.backgroundColor = 'transparent')}
            >
              <span style={{ fontSize: '14px', lineHeight: 1 }}>&#128193;</span>
              <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1 }}>{node.name}</span>
            </div>
          ))}

          {/* Separator */}
          {session.currentNodes.filter(n => n.isDir).length > 0 && session.currentNodes.filter(n => !n.isDir).length > 0 && (
            <div style={{ height: '1px', backgroundColor: 'var(--win-stroke)', margin: '4px 8px' }} />
          )}

          {/* Files */}
          {session.currentNodes.filter(n => !n.isDir).map((node, idx) => (
            <div
              key={`f-${idx}`}
              onClick={() => handleEntryClick(node)}
              style={{
                padding: '5px 8px',
                fontSize: '12px',
                cursor: 'pointer',
                borderRadius: 'var(--win-radius-sm)',
                backgroundColor: selectedFile === node.path ? 'var(--win-accent)' : 'transparent',
                color: selectedFile === node.path ? '#fff' : 'var(--win-text)',
                display: 'flex',
                alignItems: 'center',
                gap: '6px',
              }}
              onMouseEnter={e => { if (selectedFile !== node.path) e.currentTarget.style.backgroundColor = 'var(--win-subtle-hover)'; }}
              onMouseLeave={e => { if (selectedFile !== node.path) e.currentTarget.style.backgroundColor = 'transparent'; }}
            >
              <span style={{ fontSize: '13px', lineHeight: 1, flexShrink: 0 }}>{getFileIcon(node.name, false)}</span>
              <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1 }}>{node.name}</span>
              {node.size > 0 && (
                <span style={{ fontSize: '10px', color: selectedFile === node.path ? 'rgba(255,255,255,0.7)' : 'var(--win-text-tertiary)', flexShrink: 0 }}>
                  {formatSize(node.size)}
                </span>
              )}
            </div>
          ))}

          {/* Empty state */}
          {session.currentNodes.length === 0 && (
            <div style={{ fontSize: '11px', color: 'var(--win-text-tertiary)', padding: '16px 8px', textAlign: 'center' }}>
              {loading ? (
                <span style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '6px' }}>
                  <span style={{ display: 'inline-block', width: 12, height: 12, border: '2px solid var(--win-stroke-strong)', borderTopColor: 'var(--win-accent)', borderRadius: '50%', animation: 'examineSpin 0.6s linear infinite' }} />
                  Loading...
                </span>
              ) : 'Empty directory'}
            </div>
          )}
        </div>
      </div>

      {/* Content area */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        {/* Header bar with path + badges */}
        <div style={{ padding: '6px 12px', borderBottom: '1px solid var(--win-stroke)', display: 'flex', alignItems: 'center', gap: '6px', flexShrink: 0, minHeight: '32px' }}>
          {selectedFile ? (
            <>
              <span style={{ fontSize: '11px', color: 'var(--win-text-tertiary)' }}>Inspecting:</span>
              <code style={{ fontSize: '11px', color: 'var(--win-accent)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{selectedFile}</code>
              {isTextMode ? (
                <span style={{ fontSize: '9px', fontWeight: '600', color: '#00c850', padding: '1px 5px', backgroundColor: 'rgba(0,200,80,0.1)', borderRadius: '3px', flexShrink: 0 }}>TEXT</span>
              ) : hexData.length > 0 ? (
                <span style={{ fontSize: '9px', fontWeight: '600', color: '#f5a623', padding: '1px 5px', backgroundColor: 'rgba(245,166,35,0.1)', borderRadius: '3px', flexShrink: 0 }}>BINARY</span>
              ) : null}
              {fileSize > 0 && (
                <span style={{ fontSize: '10px', color: 'var(--win-text-tertiary)', marginLeft: 'auto', flexShrink: 0 }}>{formatSize(fileSize)}</span>
              )}
            </>
          ) : (
            <span style={{ fontSize: '12px', color: 'var(--win-text-tertiary)' }}>Select a file to examine</span>
          )}
        </div>

        {/* Main content */}
        <div style={{ flex: 1, overflow: 'auto', position: 'relative' }}>
          {loading ? (
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'var(--win-text-secondary)', gap: '8px' }}>
              <span style={{ display: 'inline-block', width: 20, height: 20, border: '2px solid var(--win-stroke-strong)', borderTopColor: 'var(--win-accent)', borderRadius: '50%', animation: 'examineSpin 0.6s linear infinite' }} />
              <span style={{ fontSize: '12px' }}>Reading file data...</span>
            </div>
          ) : error ? (
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100%', padding: '20px', gap: '8px' }}>
              <span style={{ fontSize: '20px', opacity: 0.5 }}>&#9888;</span>
              <span style={{ color: 'var(--win-danger)', fontSize: '12px', textAlign: 'center' }}>{error}</span>
            </div>
          ) : isTextMode ? (
            textContent !== null ? (
              <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
                <div style={{ flex: 1, minHeight: 0, overflow: 'auto' }}>
                  <CodeMirror
                    value={textContent}
                    readOnly={true}
                    editable={false}
                    extensions={[darkTheme, EditorView.lineWrapping, ...languageExtensions]}
                    basicSetup={{
                      lineNumbers: true,
                      highlightActiveLine: false,
                      highlightActiveLineGutter: false,
                      foldGutter: true,
                      bracketMatching: true,
                      closeBrackets: true,
                    }}
                  />
                </div>
              </div>
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'var(--win-text-tertiary)', gap: '6px' }}>
                <span style={{ fontSize: '20px', opacity: 0.4 }}>&#128196;</span>
                <span style={{ fontSize: '12px' }}>Select a file to view contents</span>
              </div>
            )
          ) : hexData.length > 0 ? (
            <div style={{ display: 'flex', height: '100%' }}>
              {/* Hex table */}
              <div style={{ flex: 1, overflow: 'auto' }}>
                <table style={{ borderCollapse: 'collapse', width: '100%', fontFamily: 'var(--win-font-mono)', fontSize: '12px' }}>
                  <thead>
                    <tr style={{ color: 'var(--win-text-secondary)', borderBottom: '1px solid var(--win-stroke)' }}>
                      <th style={{ textAlign: 'left', padding: '4px 8px', fontWeight: '600', position: 'sticky', top: 0, backgroundColor: 'var(--win-bg)', zIndex: 1 }}>Offset</th>
                      <th style={{ textAlign: 'left', padding: '4px 8px', fontWeight: '600', position: 'sticky', top: 0, backgroundColor: 'var(--win-bg)', zIndex: 1, letterSpacing: '0.5px' }}>
                        {'00 01 02 03 04 05 06 07  08 09 0A 0B 0C 0D 0E 0F'}
                      </th>
                      <th style={{ textAlign: 'left', padding: '4px 8px', fontWeight: '600', position: 'sticky', top: 0, backgroundColor: 'var(--win-bg)', zIndex: 1 }}>ASCII</th>
                    </tr>
                  </thead>
                  <tbody>
                    {hexData.map((line, idx) => (
                      <tr
                        key={idx}
                        style={{
                          borderBottom: '1px solid rgba(0,0,0,0.05)',
                          backgroundColor: hoveredRow === idx ? 'rgba(99,102,241,0.08)' : 'transparent',
                          transition: 'background-color 0.1s',
                        }}
                        onMouseEnter={() => setHoveredRow(idx)}
                        onMouseLeave={() => setHoveredRow(null)}
                      >
                        <td style={{ padding: '2px 8px', color: 'var(--win-accent)', whiteSpace: 'nowrap', userSelect: 'none' }}>{line.offset}</td>
                        <td style={{ padding: '2px 8px', whiteSpace: 'nowrap' }}>
                          {line.bytes.map((b, bi) => (
                            <span key={bi} style={{
                              marginRight: bi === 7 ? '12px' : '5px',
                              color: b === '00' ? 'var(--win-text-secondary)' : 'var(--win-text)',
                              opacity: b === '00' ? 0.4 : 1,
                              fontSize: '11px',
                            }}>{b}</span>
                          ))}
                        </td>
                        <td style={{ padding: '2px 8px', color: 'var(--win-text-secondary)', fontFamily: 'var(--win-font-mono)', fontSize: '11px', letterSpacing: '0.5px' }}>{line.ascii}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              {/* Data Inspector */}
              <div style={{ width: '240px', borderLeft: '1px solid var(--win-stroke)', padding: '10px 12px', overflow: 'auto', flexShrink: 0, backgroundColor: 'var(--win-surface)' }}>
                <div style={{ fontSize: '11px', fontWeight: '600', color: 'var(--win-text)', marginBottom: '10px', textTransform: 'uppercase', letterSpacing: '0.5px' }}>Data Inspector</div>

                <div style={{ marginBottom: '10px' }}>
                  <label style={{ fontSize: '10px', color: 'var(--win-text-secondary)', display: 'block', marginBottom: '3px' }}>Offset (hex)</label>
                  <input
                    value={inspectorOffset.toString(16).toUpperCase()}
                    onChange={e => {
                      const val = parseInt(e.target.value, 16);
                      if (!isNaN(val) && val >= 0) setInspectorOffset(Math.min(val, hexData.length * 16 - 1));
                    }}
                    style={{ width: '100%', padding: '5px 8px', fontSize: '11px', fontFamily: 'var(--win-font-mono)', backgroundColor: 'var(--win-bg)', border: '1px solid var(--win-stroke-strong)', borderRadius: 'var(--win-radius-sm)', color: 'var(--win-text)', outline: 'none', boxSizing: 'border-box' }}
                  />
                </div>

                <div style={{ fontSize: '10px', fontWeight: '600', color: 'var(--win-text-secondary)', marginBottom: '6px', textTransform: 'uppercase', letterSpacing: '0.5px' }}>
                  Interpreted Values
                </div>

                <div style={{ fontSize: '11px' }}>
                  {[
                    { label: 'UInt8', value: `0x${hexData[Math.floor(inspectorOffset / 16)]?.bytes[inspectorOffset % 16] || '00'}` },
                    { label: 'UInt16 (LE)', value: getUInt16(inspectorOffset).toString() },
                    { label: 'UInt32 (LE)', value: getUInt32(inspectorOffset).toString() },
                    { label: 'Int16 (LE)', value: (() => { const v = getUInt16(inspectorOffset); return v > 32767 ? (v - 65536).toString() : v.toString(); })() },
                    { label: 'Int32 (LE)', value: (() => { const v = getUInt32(inspectorOffset); return v > 2147483647 ? (v - 4294967296).toString() : v.toString(); })() },
                    { label: 'Float32', value: (() => { try { const buf = new ArrayBuffer(4); const view = new DataView(buf); const lo = getUInt16(inspectorOffset); const hi = getUInt16(inspectorOffset + 2); view.setUint16(0, lo, true); view.setUint16(2, hi, true); const f = view.getFloat32(0, true); return isFinite(f) ? f.toFixed(6) : 'N/A'; } catch { return 'N/A'; } })() },
                    { label: 'Timestamp', value: formatTimestamp(getUInt32(inspectorOffset)) },
                  ].map((item, idx) => (
                    <div key={idx} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '3px 0', borderBottom: '1px solid var(--win-stroke)' }}>
                      <span style={{ color: 'var(--win-text-secondary)', fontSize: '10px' }}>{item.label}</span>
                      <code style={{ fontSize: '10px', color: 'var(--win-text)' }}>{item.value}</code>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'var(--win-text-tertiary)', gap: '6px' }}>
              <span style={{ fontSize: '20px', opacity: 0.4 }}>&#128196;</span>
              <span style={{ fontSize: '12px' }}>Select a file to view contents</span>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

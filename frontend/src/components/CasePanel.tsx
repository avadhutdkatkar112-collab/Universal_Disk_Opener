import { useState, useCallback } from 'react';
import { useCaseManager, Bookmark, CaseNote } from '../hooks/useCaseManager';

interface CasePanelProps {
  onNavigate?: (path: string) => void;
}

export default function CasePanel({ onNavigate }: CasePanelProps) {
  const {
    caseData,
    addBookmark,
    removeBookmark,
    addNote,
    removeNote,
    exportCase,
    clearCase,
  } = useCaseManager();

  const [activeTab, setActiveTab] = useState<'bookmarks' | 'notes' | 'audit'>('bookmarks');
  const [newBookmarkPath, setNewBookmarkPath] = useState('');
  const [newBookmarkName, setNewBookmarkName] = useState('');
  const [newNoteTitle, setNewNoteTitle] = useState('');
  const [newNoteContent, setNewNoteContent] = useState('');
  const [showAddBookmark, setShowAddBookmark] = useState(false);
  const [showAddNote, setShowAddNote] = useState(false);

  const handleAddBookmark = useCallback(() => {
    if (newBookmarkPath && newBookmarkName) {
      addBookmark(newBookmarkPath, newBookmarkName);
      setNewBookmarkPath('');
      setNewBookmarkName('');
      setShowAddBookmark(false);
    }
  }, [newBookmarkPath, newBookmarkName, addBookmark]);

  const handleAddNote = useCallback(() => {
    if (newNoteTitle && newNoteContent) {
      addNote(newNoteTitle, newNoteContent);
      setNewNoteTitle('');
      setNewNoteContent('');
      setShowAddNote(false);
    }
  }, [newNoteTitle, newNoteContent, addNote]);

  return (
    <div style={{
      background: 'var(--win-surface)', border: '1px solid var(--win-stroke)',
      borderRadius: 'var(--win-radius-sm)', padding: 12, marginBottom: 12,
    }}>
      {/* Header */}
      <div style={{
        display: 'flex', justifyContent: 'space-between', alignItems: 'center',
        marginBottom: 12, paddingBottom: 8, borderBottom: '1px solid var(--win-stroke)',
      }}>
        <div>
          <div style={{ fontSize: 13, fontWeight: 500, color: 'var(--win-text)' }}>
            Case: {caseData.name}
          </div>
          <div style={{ fontSize: 11, color: 'var(--win-text-tertiary)' }}>
            {caseData.bookmarks.length} bookmarks · {caseData.notes.length} notes · {caseData.auditLog.length} events
          </div>
        </div>
        <div style={{ display: 'flex', gap: 6 }}>
          <button onClick={exportCase} style={{
            padding: '4px 8px', fontSize: 11,
            border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
            background: 'var(--win-bg)', color: 'var(--win-text)',
          }}>
            Export
          </button>
          <button onClick={clearCase} style={{
            padding: '4px 8px', fontSize: 11,
            border: '1px solid var(--win-danger)', borderRadius: 'var(--win-radius-sm)',
            background: 'transparent', color: 'var(--win-danger)',
          }}>
            Clear
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div style={{ display: 'flex', gap: 4, marginBottom: 12 }}>
        {(['bookmarks', 'notes', 'audit'] as const).map(tab => (
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

      {/* Bookmarks Tab */}
      {activeTab === 'bookmarks' && (
        <div>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
            <span style={{ fontSize: 11, color: 'var(--win-text-secondary)' }}>
              {caseData.bookmarks.length} bookmarks
            </span>
            <button onClick={() => setShowAddBookmark(!showAddBookmark)} style={{
              padding: '2px 8px', fontSize: 11,
              border: '1px solid var(--win-accent)', borderRadius: 'var(--win-radius-sm)',
              background: 'transparent', color: 'var(--win-accent)',
            }}>
              + Add
            </button>
          </div>

          {showAddBookmark && (
            <div style={{
              padding: 8, marginBottom: 8, background: 'var(--win-bg)',
              border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
            }}>
              <input
                type="text"
                placeholder="File path..."
                value={newBookmarkPath}
                onChange={e => setNewBookmarkPath(e.target.value)}
                style={{
                  width: '100%', padding: '4px 8px', fontSize: 11, marginBottom: 6,
                  border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
                  background: 'var(--win-surface)', color: 'var(--win-text)',
                }}
              />
              <input
                type="text"
                placeholder="Bookmark name..."
                value={newBookmarkName}
                onChange={e => setNewBookmarkName(e.target.value)}
                style={{
                  width: '100%', padding: '4px 8px', fontSize: 11, marginBottom: 6,
                  border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
                  background: 'var(--win-surface)', color: 'var(--win-text)',
                }}
              />
              <button onClick={handleAddBookmark} style={{
                padding: '4px 12px', fontSize: 11,
                border: '1px solid var(--win-accent)', borderRadius: 'var(--win-radius-sm)',
                background: 'var(--win-accent)', color: 'white',
              }}>
                Save
              </button>
            </div>
          )}

          {caseData.bookmarks.map((bookmark: Bookmark) => (
            <div key={bookmark.id} style={{
              padding: 8, marginBottom: 6, background: 'var(--win-bg)',
              border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
              borderLeft: `3px solid ${bookmark.color}`,
            }}>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <div>
                  <div style={{ fontSize: 12, fontWeight: 500, color: 'var(--win-text)' }}>
                    {bookmark.name}
                  </div>
                  <div style={{ fontSize: 10, color: 'var(--win-text-tertiary)', fontFamily: 'var(--win-font-mono)' }}>
                    {bookmark.path}
                  </div>
                </div>
                <div style={{ display: 'flex', gap: 4 }}>
                  {onNavigate && (
                    <button onClick={() => onNavigate(bookmark.path)} style={{
                      padding: '2px 6px', fontSize: 10,
                      border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
                      background: 'transparent', color: 'var(--win-text)',
                    }}>
                      Go
                    </button>
                  )}
                  <button onClick={() => removeBookmark(bookmark.id)} style={{
                    padding: '2px 6px', fontSize: 10,
                    border: '1px solid var(--win-danger)', borderRadius: 'var(--win-radius-sm)',
                    background: 'transparent', color: 'var(--win-danger)',
                  }}>
                    ✕
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Notes Tab */}
      {activeTab === 'notes' && (
        <div>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
            <span style={{ fontSize: 11, color: 'var(--win-text-secondary)' }}>
              {caseData.notes.length} notes
            </span>
            <button onClick={() => setShowAddNote(!showAddNote)} style={{
              padding: '2px 8px', fontSize: 11,
              border: '1px solid var(--win-accent)', borderRadius: 'var(--win-radius-sm)',
              background: 'transparent', color: 'var(--win-accent)',
            }}>
              + Add
            </button>
          </div>

          {showAddNote && (
            <div style={{
              padding: 8, marginBottom: 8, background: 'var(--win-bg)',
              border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
            }}>
              <input
                type="text"
                placeholder="Note title..."
                value={newNoteTitle}
                onChange={e => setNewNoteTitle(e.target.value)}
                style={{
                  width: '100%', padding: '4px 8px', fontSize: 11, marginBottom: 6,
                  border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
                  background: 'var(--win-surface)', color: 'var(--win-text)',
                }}
              />
              <textarea
                placeholder="Note content..."
                value={newNoteContent}
                onChange={e => setNewNoteContent(e.target.value)}
                rows={4}
                style={{
                  width: '100%', padding: '4px 8px', fontSize: 11, marginBottom: 6,
                  border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
                  background: 'var(--win-surface)', color: 'var(--win-text)',
                  resize: 'vertical',
                }}
              />
              <button onClick={handleAddNote} style={{
                padding: '4px 12px', fontSize: 11,
                border: '1px solid var(--win-accent)', borderRadius: 'var(--win-radius-sm)',
                background: 'var(--win-accent)', color: 'white',
              }}>
                Save
              </button>
            </div>
          )}

          {caseData.notes.map((note: CaseNote) => (
            <div key={note.id} style={{
              padding: 8, marginBottom: 6, background: 'var(--win-bg)',
              border: '1px solid var(--win-stroke)', borderRadius: 'var(--win-radius-sm)',
            }}>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <div style={{ fontSize: 12, fontWeight: 500, color: 'var(--win-text)' }}>
                  {note.title}
                </div>
                <button onClick={() => removeNote(note.id)} style={{
                  padding: '2px 6px', fontSize: 10,
                  border: '1px solid var(--win-danger)', borderRadius: 'var(--win-radius-sm)',
                  background: 'transparent', color: 'var(--win-danger)',
                }}>
                  ✕
                </button>
              </div>
              <div style={{ fontSize: 11, color: 'var(--win-text-secondary)', marginTop: 4 }}>
                {note.content.substring(0, 100)}...
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Audit Tab */}
      {activeTab === 'audit' && (
        <div style={{ maxHeight: 300, overflow: 'auto' }}>
          {caseData.auditLog.map((entry) => (
            <div key={entry.id} style={{
              padding: 6, marginBottom: 4, fontSize: 10,
              borderBottom: '1px solid var(--win-stroke)',
            }}>
              <div style={{ display: 'flex', gap: 8 }}>
                <span style={{ color: 'var(--win-accent)', fontWeight: 500 }}>
                  {entry.action}
                </span>
                <span style={{ color: 'var(--win-text-tertiary)' }}>
                  {new Date(entry.timestamp).toLocaleTimeString()}
                </span>
              </div>
              <div style={{ color: 'var(--win-text-secondary)', marginTop: 2 }}>
                {entry.target}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

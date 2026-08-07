import { useState, useCallback, useEffect } from 'react';

export interface Bookmark {
  id: string;
  path: string;
  name: string;
  notes: string;
  tags: string[];
  createdAt: string;
  color: string;
}

export interface CaseNote {
  id: string;
  title: string;
  content: string;
  relatedPaths: string[];
  createdAt: string;
  updatedAt: string;
  tags: string[];
}

export interface AuditEntry {
  id: string;
  action: string;
  target: string;
  timestamp: string;
  details: string;
}

export interface CaseData {
  name: string;
  description: string;
  createdAt: string;
  bookmarks: Bookmark[];
  notes: CaseNote[];
  auditLog: AuditEntry[];
  tags: string[];
}

const STORAGE_KEY = 'vhd-opener-case';

const defaultCase: CaseData = {
  name: 'New Investigation',
  description: '',
  createdAt: new Date().toISOString(),
  bookmarks: [],
  notes: [],
  auditLog: [],
  tags: ['evidence', 'suspicious', 'verified'],
};

export function useCaseManager() {
  const [caseData, setCaseData] = useState<CaseData>(() => {
    try {
      const saved = localStorage.getItem(STORAGE_KEY);
      return saved ? JSON.parse(saved) : defaultCase;
    } catch {
      return defaultCase;
    }
  });

  useEffect(() => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(caseData));
  }, [caseData]);

  const addAuditEntry = useCallback((action: string, target: string, details: string) => {
    setCaseData(prev => ({
      ...prev,
      auditLog: [
        {
          id: Date.now().toString(),
          action,
          target,
          timestamp: new Date().toISOString(),
          details,
        },
        ...prev.auditLog,
      ].slice(0, 1000), // Keep last 1000 entries
    }));
  }, []);

  const addBookmark = useCallback((path: string, name: string, notes: string = '', tags: string[] = [], color: string = '#0078D4') => {
    const bookmark: Bookmark = {
      id: Date.now().toString(),
      path,
      name,
      notes,
      tags,
      createdAt: new Date().toISOString(),
      color,
    };
    setCaseData(prev => ({
      ...prev,
      bookmarks: [...prev.bookmarks, bookmark],
    }));
    addAuditEntry('BOOKMARK_ADDED', path, `Bookmarked: ${name}`);
    return bookmark;
  }, [addAuditEntry]);

  const removeBookmark = useCallback((id: string) => {
    setCaseData(prev => ({
      ...prev,
      bookmarks: prev.bookmarks.filter(b => b.id !== id),
    }));
    addAuditEntry('BOOKMARK_REMOVED', id, 'Bookmark removed');
  }, [addAuditEntry]);

  const updateBookmark = useCallback((id: string, updates: Partial<Bookmark>) => {
    setCaseData(prev => ({
      ...prev,
      bookmarks: prev.bookmarks.map(b => b.id === id ? { ...b, ...updates } : b),
    }));
    addAuditEntry('BOOKMARK_UPDATED', id, 'Bookmark updated');
  }, [addAuditEntry]);

  const addNote = useCallback((title: string, content: string, relatedPaths: string[] = [], tags: string[] = []) => {
    const note: CaseNote = {
      id: Date.now().toString(),
      title,
      content,
      relatedPaths,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      tags,
    };
    setCaseData(prev => ({
      ...prev,
      notes: [...prev.notes, note],
    }));
    addAuditEntry('NOTE_ADDED', title, `Note created: ${title}`);
    return note;
  }, [addAuditEntry]);

  const updateNote = useCallback((id: string, updates: Partial<CaseNote>) => {
    setCaseData(prev => ({
      ...prev,
      notes: prev.notes.map(n => n.id === id ? { ...n, ...updates, updatedAt: new Date().toISOString() } : n),
    }));
    addAuditEntry('NOTE_UPDATED', id, 'Note updated');
  }, [addAuditEntry]);

  const removeNote = useCallback((id: string) => {
    setCaseData(prev => ({
      ...prev,
      notes: prev.notes.filter(n => n.id !== id),
    }));
    addAuditEntry('NOTE_REMOVED', id, 'Note removed');
  }, [addAuditEntry]);

  const addTag = useCallback((tag: string) => {
    setCaseData(prev => ({
      ...prev,
      tags: prev.tags.includes(tag) ? prev.tags : [...prev.tags, tag],
    }));
  }, []);

  const exportCase = useCallback(() => {
    const blob = new Blob([JSON.stringify(caseData, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `case-${caseData.name.replace(/\s+/g, '-').toLowerCase()}-${new Date().toISOString().split('T')[0]}.json`;
    a.click();
    URL.revokeObjectURL(url);
    addAuditEntry('CASE_EXPORTED', caseData.name, 'Case exported to JSON');
  }, [caseData, addAuditEntry]);

  const importCase = useCallback((json: string) => {
    try {
      const imported = JSON.parse(json) as CaseData;
      setCaseData(imported);
      addAuditEntry('CASE_IMPORTED', imported.name, 'Case imported from JSON');
      return true;
    } catch {
      return false;
    }
  }, [addAuditEntry]);

  const clearCase = useCallback(() => {
    setCaseData(defaultCase);
    addAuditEntry('CASE_CLEARED', '', 'Case data cleared');
  }, [addAuditEntry]);

  return {
    caseData,
    addBookmark,
    removeBookmark,
    updateBookmark,
    addNote,
    updateNote,
    removeNote,
    addTag,
    exportCase,
    importCase,
    clearCase,
    addAuditEntry,
  };
}

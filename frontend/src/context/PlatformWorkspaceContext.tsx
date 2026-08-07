import React, { createContext, useContext, useEffect, useState, ReactNode } from 'react';

declare global {
  interface Window {
    go?: {
      api?: {
        App: {
          OpenFileDialog(): Promise<string>;
          OpenFile(path: string): Promise<any>;
          GetDetailedDiskInfo(): Promise<any>;
          GetFilePreview(path: string): Promise<any>;
          ListDirectory(path: string): Promise<any[]>;
          GetDiskHash(): Promise<{ md5: string; sha256: string }>;
          ExecuteCommand(command: string, context: string): Promise<any>;
          GetSuggestions(query: string, context: string): Promise<any[]>;
          SelectPartition(index: number): Promise<any>;
          GetPartitions(): Promise<any[]>;
          GetEntry(path: string): Promise<any>;
          HashFile(vfsPath: string, verifyHash: string): Promise<any>;
          BatchHashFiles(vfsPaths: string[]): Promise<any>;
          PartitionHash(partitionIndex: number): Promise<any>;
          CompareHash(pathA: string, pathB: string): Promise<any>;
          StartPartitionHashJob(partitionIndex: number): Promise<any>;
          StartDirectoryHashJob(vfsPath: string): Promise<any>;
          GetHashJobStatus(jobID: string): Promise<any>;
          CancelHashJob(jobID: string): Promise<any>;
          StartBatchHashJob(vfsPaths: string[]): Promise<any>;
          ExtractFile(diskPath: string, localPath: string): Promise<void>;
          GetGrammarForExtension(ext: string): Promise<any>;
          GetFileInfoForPath(path: string): Promise<any>;
          BrowseLocalFS(dirPath: string): Promise<any>;
          GetLocalDrives(): Promise<any>;
          GetRecentFiles(): Promise<any[]>;
          ParseCommand(command: string, ctxStr: string): Promise<any>;
          GetCommandHistory(): Promise<any[]>;
          GetFavorites(): Promise<any[]>;
          AddFavorite(command: string, label: string): Promise<void>;
          RemoveFavorite(label: string): Promise<void>;
          GetRegistryArtifacts(hivePath: string, hiveType: string): Promise<any>;
          ParseEVTXFile(evtxPath: string): Promise<any>;
          ParseMFTFile(mftPath: string): Promise<any>;
          BuildUnifiedTimeline(...args: any[]): Promise<any>;
          ExportTimelineCSV(entries: any[], outputPath: string): Promise<void>;
          ExportTimelineJSON(entries: any[], outputPath: string): Promise<void>;
          AnalyzeFindings(entries: any[]): Promise<any>;
          LoadTimelineToSQL(entries: any[]): Promise<any>;
          ExecuteSQLQuery(query: string): Promise<any>;
          LoadSigmaRuleDirectory(dir: string): Promise<any>;
          RunSigmaScan(entries: any[]): Promise<any>;
          GetSigmaRuleCount(): Promise<any>;
          GetLiveProcesses(): Promise<any>;
          RunMemoryYaraScan(procs: any[]): Promise<any>;
          GetMemorySnapshot(): Promise<any>;
          GenerateReport(...args: any[]): Promise<any>;
          SaveReportToFile(filePath: string, htmlContent: string): Promise<void>;
          IngestEvidence(...args: any[]): Promise<any>;
          IngestEvidenceStruct(req: any): Promise<any>;
          VerifyEvidenceContainer(caseDir: string): Promise<any>;
          VerifyCaseIntegrity(caseDir: string): Promise<any>;
          LogAnalystAction(actor: string, action: string, metadata: string): Promise<void>;
          ExportChainOfCustodyReport(req: any): Promise<any>;
          GetWorkspaceState(): Promise<any>;
          Close(): Promise<void>;
        };
        WailsBridge: {
          GetWorkspaceState(): Promise<WorkspaceState>;
          MountDiskTarget(target: WorkspaceTarget): Promise<void>;
          ExecuteCapability(id: string, params: Record<string, any>): Promise<string>;
          ExecuteUCL(query: string): Promise<string>;
          CancelJob(jobID: string): Promise<boolean>;
          GetJobStatus(jobID: string): Promise<any>;
          AddBookmark(label: string, path: string): Promise<any>;
          RemoveBookmark(bookmarkID: string): Promise<void>;
          GetBookmarks(): Promise<any[]>;
          NavigateWorkspace(path: string): Promise<void>;
          OpenTab(path: string): Promise<void>;
          CloseTab(index: number): Promise<void>;
          SetActiveTab(index: number): Promise<void>;
        };
        StorageHandler: {
          MountDisk(imagePath: string): Promise<any>;
          MountPartition(index: number): Promise<any>;
          ListDirectory(path: string): Promise<any[]>;
          ReadFileChunk(path: string, offset: number, length: number): Promise<any>;
          ReadFile(path: string): Promise<any>;
        };
        SessionHandler: {
          MountDiskToSession(imagePath: string): Promise<any>;
        };
      };
    };
    runtime?: {
      WindowMinimise: () => void;
      WindowToggleMaximise: () => void;
      WindowClose: () => void;
      WindowIsMaximised: () => Promise<boolean>;
      EventsOn(eventName: string, callback: (data: any) => void): void;
      EventsOff(eventName: string): void;
    };
  }
}

export interface WorkspaceTarget {
  id: string;
  image_path: string;
  format: string;
  partitions: string[];
  is_encrypted: boolean;
}

export interface WorkspaceState {
  id: string;
  name: string;
  targets: WorkspaceTarget[];
  active_target_id: string;
  active_partition: string;
  bookmarks: any[];
  opened_tabs: string[];
  active_tab_index: number;
}

interface ActiveJob {
  id: string;
  progress: number;
  status: string;
  result?: any;
}

interface PlatformWorkspaceContextType {
  workspace: WorkspaceState | null;
  mountDisk: (target: WorkspaceTarget) => Promise<void>;
  executeCapability: (id: string, params: Record<string, any>) => Promise<string>;
  executeUCL: (query: string) => Promise<string>;
  cancelJob: (jobID: string) => Promise<boolean>;
  openDisk: (path: string) => Promise<void>;
  navigate: (path: string) => Promise<void>;
  addBookmark: (label: string, path: string) => Promise<void>;
  removeBookmark: (id: string) => Promise<void>;
  bookmarks: any[];
  activeJobs: Record<string, ActiveJob>;
}

const PlatformWorkspaceContext = createContext<PlatformWorkspaceContextType | undefined>(undefined);

export const PlatformWorkspaceProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [workspace, setWorkspace] = useState<WorkspaceState | null>(null);
  const [activeJobs, setActiveJobs] = useState<Record<string, ActiveJob>>({});
  const [bookmarks, setBookmarks] = useState<any[]>([]);

  useEffect(() => {
    if (window.go?.api?.WailsBridge) {
      window.go.api.WailsBridge.GetWorkspaceState()
        .then(setWorkspace)
        .catch((err) => console.error("Failed to load workspace:", err));

      window.go.api.WailsBridge.GetBookmarks()
        .then(setBookmarks)
        .catch(() => {});
    }

    if (window.runtime) {
      window.runtime.EventsOn("WORKSPACE_UPDATED", (newState: WorkspaceState) => {
        setWorkspace(newState);
      });

      window.runtime.EventsOn("JOB_PROGRESS", (payload: any) => {
        setActiveJobs((prev) => ({
          ...prev,
          [payload.job_id]: {
            id: payload.job_id,
            progress: payload.progress,
            status: 'RUNNING',
          },
        }));
      });

      window.runtime.EventsOn("JOB_COMPLETED", (payload: any) => {
        setActiveJobs((prev) => {
          const updated = { ...prev };
          delete updated[payload.job_id];
          return updated;
        });
      });

      window.runtime.EventsOn("JOB_FAILED", (payload: any) => {
        setActiveJobs((prev) => {
          const updated = { ...prev };
          delete updated[payload.job_id];
          return updated;
        });
      });
    }

    return () => {
      if (window.runtime) {
        window.runtime.EventsOff("WORKSPACE_UPDATED");
        window.runtime.EventsOff("JOB_PROGRESS");
        window.runtime.EventsOff("JOB_COMPLETED");
        window.runtime.EventsOff("JOB_FAILED");
      }
    };
  }, []);

  const mountDisk = async (target: WorkspaceTarget) => {
    if (window.go?.api?.WailsBridge) {
      await window.go.api.WailsBridge.MountDiskTarget(target);
      const state = await window.go.api.WailsBridge.GetWorkspaceState();
      setWorkspace(state);
    }
  };

  const executeCapability = async (id: string, params: Record<string, any>): Promise<string> => {
    if (window.go?.api?.WailsBridge) {
      return await window.go.api.WailsBridge.ExecuteCapability(id, params);
    }
    throw new Error("Wails bridge not initialized");
  };

  const executeUCL = async (query: string): Promise<string> => {
    if (window.go?.api?.WailsBridge) {
      return await window.go.api.WailsBridge.ExecuteUCL(query);
    }
    throw new Error("Wails bridge not initialized");
  };

  const cancelJob = async (jobID: string): Promise<boolean> => {
    if (window.go?.api?.WailsBridge) {
      return await window.go.api.WailsBridge.CancelJob(jobID);
    }
    return false;
  };

  const openDisk = async (path: string) => {
    if (window.go?.api?.App) {
      const result = await window.go.api.App.OpenFile(path);
      if (result && window.go?.api?.WailsBridge) {
        const target: WorkspaceTarget = {
          id: `target_${Date.now()}`,
          image_path: path,
          format: path.split('.').pop()?.toUpperCase() || 'UNKNOWN',
          partitions: result.partitions?.map((p: any) => p.filesystem) || [],
          is_encrypted: false,
        };
        await mountDisk(target);
      }
    }
  };

  const navigate = async (path: string) => {
    if (window.go?.api?.WailsBridge) {
      await window.go.api.WailsBridge.NavigateWorkspace(path);
    }
  };

  const addBookmark = async (label: string, path: string) => {
    if (window.go?.api?.WailsBridge) {
      const bm = await window.go.api.WailsBridge.AddBookmark(label, path);
      setBookmarks((prev) => [...prev, bm]);
    }
  };

  const removeBookmark = async (id: string) => {
    if (window.go?.api?.WailsBridge) {
      await window.go.api.WailsBridge.RemoveBookmark(id);
      setBookmarks((prev) => prev.filter((b) => b.id !== id));
    }
  };

  return (
    <PlatformWorkspaceContext.Provider
      value={{
        workspace,
        mountDisk,
        executeCapability,
        executeUCL,
        cancelJob,
        openDisk,
        navigate,
        addBookmark,
        removeBookmark,
        bookmarks,
        activeJobs,
      }}
    >
      {children}
    </PlatformWorkspaceContext.Provider>
  );
};

export const usePlatformWorkspace = () => {
  const context = useContext(PlatformWorkspaceContext);
  if (!context) {
    throw new Error("usePlatformWorkspace must be used within a PlatformWorkspaceProvider");
  }
  return context;
};

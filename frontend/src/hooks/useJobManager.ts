import { useEffect, useState, useCallback } from 'react';

export interface HashJobProgress {
  job_id: string;
  target_path: string;
  status: 'PENDING' | 'RUNNING' | 'COMPLETED' | 'FAILED';
  bytes_processed: number;
  total_bytes: number;
  percentage: number;
  throughput_mbps: number;
  eta_seconds: number;
  error?: string;
  result?: any;
}

export function useJobManager() {
  const [activeJobs, setActiveJobs] = useState<Record<string, HashJobProgress>>({});

  useEffect(() => {
    const rt = (window as any).runtime;
    if (!rt) return;

    const handleProgress = (data: HashJobProgress) => {
      setActiveJobs(prev => ({ ...prev, [data.job_id]: data }));
    };

    const handleComplete = (data: HashJobProgress) => {
      setActiveJobs(prev => ({ ...prev, [data.job_id]: data }));
    };

    rt.EventsOn('hash:progress', handleProgress);
    rt.EventsOn('hash:complete', handleComplete);

    return () => {
      rt.EventsOff('hash:progress');
      rt.EventsOff('hash:complete');
    };
  }, []);

  const getJob = useCallback((jobId: string): HashJobProgress | undefined => {
    return activeJobs[jobId];
  }, [activeJobs]);

  const isRunning = useCallback((jobId: string): boolean => {
    return activeJobs[jobId]?.status === 'RUNNING';
  }, [activeJobs]);

  const getCompletedJob = useCallback((jobId: string): HashJobProgress | undefined => {
    const job = activeJobs[jobId];
    return job?.status === 'COMPLETED' ? job : undefined;
  }, [activeJobs]);

  return { activeJobs, getJob, isRunning, getCompletedJob };
}

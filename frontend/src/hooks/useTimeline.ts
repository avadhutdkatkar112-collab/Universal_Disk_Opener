import { useState, useCallback } from 'react';
import { BuildUnifiedTimeline } from '../lib/wails';
import { timeline } from '../../wailsjs/go/models';

export type TimelineEntry = timeline.TimelineEntry;

export interface TimelineResult {
  total_events: number;
  start_time: string;
  end_time: string;
  statistics: {
    source_counts: Record<string, number>;
    type_counts: Record<string, number>;
    hourly_activity: Record<string, number>;
    daily_activity: Record<string, number>;
  };
  entries: TimelineEntry[];
}

interface UseTimelineReturn {
  result: TimelineResult | null;
  entries: TimelineEntry[];
  loading: boolean;
  error: string | null;
  buildTimeline: (params: {
    registry_path?: string;
    evtx_path?: string;
    mft_path?: string;
    start_time?: string;
    end_time?: string;
  }) => Promise<void>;
}

export function useTimeline(): UseTimelineReturn {
  const [result, setResult] = useState<TimelineResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const buildTimeline = useCallback(async (params: {
    registry_path?: string;
    evtx_path?: string;
    mft_path?: string;
    start_time?: string;
    end_time?: string;
  }) => {
    setLoading(true);
    setError(null);
    try {
      const data = await BuildUnifiedTimeline(
        params.registry_path || '',
        params.evtx_path || '',
        params.mft_path || '',
        params.start_time || '',
        params.end_time || ''
      );
      setResult(data);
    } catch (err: any) {
      setError(err?.toString() || 'Failed to build timeline');
    } finally {
      setLoading(false);
    }
  }, []);

  return {
    result,
    entries: result?.entries || [],
    loading,
    error,
    buildTimeline,
  };
}

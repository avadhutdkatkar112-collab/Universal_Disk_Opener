import { useState, useEffect } from 'react';

export interface Job {
  id: string;
  type: string;
  label: string;
  status: 'queued' | 'running' | 'completed' | 'failed' | 'cancelled';
  progress: number;
  createdAt: number;
}

interface JobState {
  jobs: Job[];
  submitJob: (type: string, label: string) => string;
  updateJob: (id: string, patch: Partial<Job>) => void;
  cancelJob: (id: string) => void;
  removeJob: (id: string) => void;
}

let nextId = 1;
let _jobs: Job[] = [];
let _listeners: Array<() => void> = [];

function _emit() {
  for (const fn of _listeners) fn();
}

function submitJob(type: string, label: string): string {
  const id = `job-${nextId++}`;
  _jobs = [
    ..._jobs,
    { id, type, label, status: 'running', progress: 0, createdAt: Date.now() },
  ];
  _emit();
  return id;
}

function updateJob(id: string, patch: Partial<Job>) {
  _jobs = _jobs.map((j) => (j.id === id ? { ...j, ...patch } : j));
  _emit();
}

function cancelJob(id: string) {
  _jobs = _jobs.map((j) => (j.id === id ? { ...j, status: 'cancelled' as const } : j));
  _emit();
  setTimeout(() => {
    _jobs = _jobs.filter((j) => j.id !== id);
    _emit();
  }, 2000);
}

function removeJob(id: string) {
  _jobs = _jobs.filter((j) => j.id !== id);
  _emit();
}

const _actions = { submitJob, updateJob, cancelJob, removeJob };

function getSnapshot(): JobState {
  return { jobs: _jobs, ..._actions };
}

export function useJobStore<T>(selector: (state: JobState) => T): T {
  const [, rerender] = useState(0);

  useEffect(() => {
    const listener = () => rerender((c) => c + 1);
    _listeners.push(listener);
    return () => {
      _listeners = _listeners.filter((fn) => fn !== listener);
    };
  }, []);

  return selector(getSnapshot());
}

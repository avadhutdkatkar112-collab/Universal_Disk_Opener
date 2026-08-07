import { useState, useEffect } from 'react';
import type { DiskSnapshot } from '../types';

interface DiskState {
  disk: DiskSnapshot | null;
  tab: string;
  setDisk: (disk: DiskSnapshot | null) => void;
  setTab: (tab: string) => void;
}

let _state: DiskState = { disk: null, tab: 'welcome', setDisk: () => {}, setTab: () => {} };
let _listeners: Array<() => void> = [];

function _emit() {
  for (const fn of _listeners) fn();
}

function setDisk(disk: DiskSnapshot | null) {
  _state = { ..._state, disk, tab: disk ? 'explorer' : 'welcome' };
  _emit();
}

function setTab(tab: string) {
  _state = { ..._state, tab };
  _emit();
}

_state.setDisk = setDisk;
_state.setTab = setTab;

function getSnapshot(): DiskState {
  return _state;
}

export function useDiskStore(): DiskState {
  const [, rerender] = useState(0);

  useEffect(() => {
    const listener = () => rerender((c) => c + 1);
    _listeners.push(listener);
    return () => {
      _listeners = _listeners.filter((fn) => fn !== listener);
    };
  }, []);

  return _state;
}

useDiskStore.getState = getSnapshot;

import React from 'react';

type IconName =
  | 'disk' | 'folder' | 'file' | 'search' | 'extract'
  | 'analyze' | 'hash' | 'report' | 'settings' | 'preview'
  | 'bookmark' | 'close' | 'chevron-down' | 'chevron-right'
  | 'plus' | 'command' | 'terminal' | 'clock' | 'check'
  | 'alert' | 'copy' | 'open-external' | 'cancel' | 'lock' | 'info';

const icons: Record<IconName, (props: { size?: number; className?: string; style?: React.CSSProperties }) => React.JSX.Element> = {
  disk: ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <circle cx="8" cy="8" r="6.5" stroke="currentColor" strokeWidth="1.2" />
      <circle cx="8" cy="8" r="2" stroke="currentColor" strokeWidth="1.2" />
      <circle cx="8" cy="8" r="0.75" fill="currentColor" />
    </svg>
  ),
  folder: ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <path d="M2 4C2 3.45 2.45 3 3 3H6.17a1 1 0 0 1 .71.29L8 4.41l4.29-4.29a1 1 0 0 1 .71-.29H13c.55 0 1 .45 1 1V12c0 .55-.45 1-1 1H3c-.55 0-1-.45-1-1V4Z" fill="currentColor" opacity="0.15" />
      <path d="M2 4C2 3.45 2.45 3 3 3H6.17a1 1 0 0 1 .71.29L8 4.41l4.29-4.29a1 1 0 0 1 .71-.29H13c.55 0 1 .45 1 1V12c0 .55-.45 1-1 1H3c-.55 0-1-.45-1-1V4Z" stroke="currentColor" strokeWidth="1.2" />
    </svg>
  ),
  file: ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <path d="M9.5 2H4.5C3.95 2 3.5 2.45 3.5 3V13C3.5 13.55 3.95 14 4.5 14H11.5C12.05 14 12.5 13.55 12.5 13V5.5L9.5 2Z" fill="currentColor" opacity="0.1" />
      <path d="M9.5 2H4.5C3.95 2 3.5 2.45 3.5 3V13C3.5 13.55 3.95 14 4.5 14H11.5C12.05 14 12.5 13.55 12.5 13V5.5L9.5 2Z" stroke="currentColor" strokeWidth="1.2" />
      <path d="M9.5 2V5.5H12.5" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
    </svg>
  ),
  search: ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <circle cx="7" cy="7" r="4.5" stroke="currentColor" strokeWidth="1.2" />
      <path d="M10.5 10.5L14 14" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
    </svg>
  ),
  extract: ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <path d="M8 2v8M4.5 6.5L8 10l3.5-3.5" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round" />
      <path d="M2 12v1.5A1.5 1.5 0 0 0 3.5 15h9a1.5 1.5 0 0 0 1.5-1.5V12" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
    </svg>
  ),
  analyze: ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <rect x="1.5" y="9" width="3" height="5" rx="0.5" stroke="currentColor" strokeWidth="1.2" />
      <rect x="6.5" y="5" width="3" height="9" rx="0.5" stroke="currentColor" strokeWidth="1.2" />
      <rect x="11.5" y="2" width="3" height="12" rx="0.5" stroke="currentColor" strokeWidth="1.2" />
    </svg>
  ),
  hash: ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <path d="M4.5 2L3 14M11.5 2L10 14M1 5.5h14M1 10.5h14" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
    </svg>
  ),
  report: ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <path d="M9 2H4.5C3.95 2 3.5 2.45 3.5 3V13C3.5 13.55 3.95 14 4.5 14H11.5C12.05 14 12.5 13.55 12.5 13V5.5L9 2Z" stroke="currentColor" strokeWidth="1.2" />
      <path d="M9 2V5.5H12.5M5 8h6M5 10.5h4" stroke="currentColor" strokeWidth="1" strokeLinecap="round" />
    </svg>
  ),
  settings: ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <circle cx="8" cy="8" r="2" stroke="currentColor" strokeWidth="1.2" />
      <path d="M8 1.5v2M8 12.5v2M1.5 8h2M12.5 8h2M3.34 3.34l1.41 1.41M11.25 11.25l1.41 1.41M3.34 12.66l1.41-1.41M11.25 4.75l1.41-1.41" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
    </svg>
  ),
  preview: ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <path d="M1 8s2.5-5 7-5 7 5 7 5-2.5 5-7 5-7-5-7-5Z" stroke="currentColor" strokeWidth="1.2" />
      <circle cx="8" cy="8" r="2.5" stroke="currentColor" strokeWidth="1.2" />
    </svg>
  ),
  bookmark: ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <path d="M4 2.5h8v11l-4-2.5L4 13.5v-11Z" stroke="currentColor" strokeWidth="1.2" strokeLinejoin="round" />
    </svg>
  ),
  close: ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <path d="M4 4l8 8M12 4l-8 8" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
    </svg>
  ),
  'chevron-down': ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <path d="M4 6l4 4 4-4" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  ),
  'chevron-right': ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <path d="M6 4l4 4-4 4" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  ),
  plus: ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <path d="M8 3v10M3 8h10" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
    </svg>
  ),
  command: ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <path d="M4 7a4 4 0 0 1 4-4h1a4 4 0 0 1 0 8H8" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
      <circle cx="4" cy="8" r="1" fill="currentColor" />
      <circle cx="12" cy="8" r="1" fill="currentColor" />
    </svg>
  ),
  terminal: ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <rect x="1.5" y="2.5" width="13" height="11" rx="1.5" stroke="currentColor" strokeWidth="1.2" />
      <path d="M4 6.5l2.5 2L4 10.5" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round" />
      <path d="M7.5 11h4" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
    </svg>
  ),
  clock: ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <circle cx="8" cy="8" r="6.5" stroke="currentColor" strokeWidth="1.2" />
      <path d="M8 4.5V8l2.5 2" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  ),
  check: ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <path d="M3 8.5l3.5 3.5 6.5-8" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  ),
  alert: ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <path d="M8 1.5L1.5 13.5h13L8 1.5Z" stroke="currentColor" strokeWidth="1.2" strokeLinejoin="round" />
      <path d="M8 6v3M8 11v.5" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
    </svg>
  ),
  copy: ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <rect x="5" y="5" width="9" height="9" rx="1.5" stroke="currentColor" strokeWidth="1.2" />
      <path d="M11 5V3.5A1.5 1.5 0 0 0 9.5 2h-6A1.5 1.5 0 0 0 2 3.5v6A1.5 1.5 0 0 0 3.5 11H5" stroke="currentColor" strokeWidth="1.2" />
    </svg>
  ),
  'open-external': ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <path d="M6 2H3.5A1.5 1.5 0 0 0 2 3.5v9A1.5 1.5 0 0 0 3.5 14h9a1.5 1.5 0 0 0 1.5-1.5V10" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
      <path d="M9 2h5v5M14 2L7 9" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  ),
  cancel: ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <circle cx="8" cy="8" r="6.5" stroke="currentColor" strokeWidth="1.2" />
      <path d="M5.5 5.5l5 5M10.5 5.5l-5 5" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
    </svg>
  ),
  lock: ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <rect x="3.5" y="7" width="9" height="7" rx="1.5" stroke="currentColor" strokeWidth="1.2" />
      <path d="M5.5 7V5a2.5 2.5 0 0 1 5 0v2" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
    </svg>
  ),
  info: ({ size = 14, className, style }) => (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} style={style}>
      <circle cx="8" cy="8" r="6.5" stroke="currentColor" strokeWidth="1.2" />
      <path d="M8 7v4M8 5v.5" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
    </svg>
  ),
};

interface IconProps {
  name: IconName;
  size?: number;
  className?: string;
  style?: React.CSSProperties;
}

export const Icon: React.FC<IconProps> = ({ name, size, className, style }) => {
  const renderer = icons[name];
  if (!renderer) return null;
  return renderer({ size, className, style });
};

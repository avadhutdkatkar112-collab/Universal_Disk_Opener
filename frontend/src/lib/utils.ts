export function fmtSize(bytes: number | null | undefined): string {
  if (bytes == null || isNaN(bytes) || bytes <= 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(k)), sizes.length - 1);
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

export function fmtNumber(n: number | null | undefined): string {
  if (n == null) return '0';
  return n.toLocaleString();
}

export function clsx(...args: (string | false | null | undefined)[]): string {
  return args.filter(Boolean).join(' ');
}

// Consistent auto units across the monitoring screen: B/KB/MB/GB for memory
// and storage, KB/s/MB/s/GB/s for network and disk I/O rates.
export function fmtBytes(n: number): string {
  if (!Number.isFinite(n) || n <= 0) return "0 B";
  if (n >= 1 << 30) return (n / (1 << 30)).toFixed(1) + " GB";
  if (n >= 1 << 20) return (n / (1 << 20)).toFixed(1) + " MB";
  if (n >= 1 << 10) return (n / (1 << 10)).toFixed(0) + " KB";
  return Math.round(n) + " B";
}

export function fmtRate(bytesPerSec: number): string {
  if (!Number.isFinite(bytesPerSec) || bytesPerSec < 0) return "—";
  if (bytesPerSec >= 1 << 30) return (bytesPerSec / (1 << 30)).toFixed(2) + " GB/s";
  if (bytesPerSec >= 1 << 20) return (bytesPerSec / (1 << 20)).toFixed(1) + " MB/s";
  if (bytesPerSec >= 1 << 10) return (bytesPerSec / (1 << 10)).toFixed(1) + " KB/s";
  return Math.round(bytesPerSec) + " B/s";
}

export function fmtUptime(sec: number): string {
  const d = Math.floor(sec / 86400);
  const h = Math.floor((sec % 86400) / 3600);
  const m = Math.floor((sec % 3600) / 60);
  return `${d}d ${h}h ${m}m`;
}

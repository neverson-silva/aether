import React from "react";
import { Card } from "./card";

export function MetricCard({
  label,
  value,
  icon,
  hint,
}: {
  label: string;
  value: string;
  icon: string;
  hint?: string;
}) {
  return (
    <Card variant="glass" className="flex flex-col gap-sm">
      <div className="flex items-center justify-between">
        <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">
          {label}
        </span>
        <span className="material-symbols-outlined text-[18px] text-primary">{icon}</span>
      </div>
      <p className="font-headline-sm text-headline-sm text-on-surface">{value}</p>
      {hint && <p className="font-body-sm text-body-sm text-on-surface-variant/70">{hint}</p>}
    </Card>
  );
}

export function fmtBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  const units = ["KiB", "MiB", "GiB", "TiB"];
  let v = bytes;
  let i = -1;
  do {
    v /= 1024;
    i++;
  } while (v >= 1024 && i < units.length - 1);
  return `${v.toFixed(1)} ${units[i]}`;
}

export function fmtDate(iso: string): string {
  if (!iso) return "";
  return new Date(iso).toLocaleDateString(undefined, { day: "2-digit", month: "short", year: "numeric" });
}

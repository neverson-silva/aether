import React from "react";
import { AppCard } from "./app-card";

export function AppStatCard({ icon, label, value, tone = "primary", onClick }: { icon: string; label: string; value: React.ReactNode; tone?: string; onClick?: () => void }) {
  const colors: Record<string, string> = {
    primary: "text-primary",
    success: "text-[#4ade80]",
    danger: "text-error",
    warning: "text-[#fbbf24]",
    info: "text-[#60a5fa]",
  };
  return (
    <AppCard className="p-md" onClick={onClick} hover={!!onClick}>
      <span className={`material-symbols-outlined text-2xl ${colors[tone] ?? colors.primary}`} style={{ fontVariationSettings: "'FILL' 1" }}>
        {icon}
      </span>
      <p className="font-headline-md text-headline-md font-bold text-on-surface mt-sm">{value}</p>
      <p className="font-label-caps text-label-caps text-on-surface-variant uppercase">{label}</p>
    </AppCard>
  );
}

export function AppInfoRow({ label, children }: { label: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-0.5">
      <span className="font-label-caps text-[10px] text-on-surface-variant/70 uppercase tracking-wider">{label}</span>
      <span className="font-code-md text-code-md text-on-surface truncate">{children}</span>
    </div>
  );
}

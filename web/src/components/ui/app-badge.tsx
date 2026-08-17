import React from "react";
import { cn } from "./cn";

const BADGE_TONES: Record<string, string> = {
  success: "text-[#4ade80] bg-[#4ade80]/10 border-[#4ade80]/20",
  danger: "text-error bg-error/10 border-error/20",
  warning: "text-[#fbbf24] bg-[#fbbf24]/10 border-[#fbbf24]/20",
  info: "text-[#60a5fa] bg-[#60a5fa]/10 border-[#60a5fa]/20",
  neutral: "text-on-surface-variant bg-surface-container-high/40 border-outline-variant/60",
  primary: "text-primary bg-primary/10 border-primary/30",
};

export function AppBadge({
  tone = "neutral",
  pulse,
  dot,
  children,
  className,
}: {
  tone?: keyof typeof BADGE_TONES | string;
  pulse?: boolean;
  dot?: boolean;
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 px-2 py-0.5 rounded border font-code-md text-code-md",
        BADGE_TONES[tone] ?? BADGE_TONES.neutral,
        className
      )}
    >
      {dot && <span className={cn("w-1.5 h-1.5 rounded-full bg-current", pulse && "status-pulse")} />}
      {children}
    </span>
  );
}

const STATUS_TONE: Record<string, string> = {
  ready: "success",
  running: "success",
  healthy: "success",
  issued: "success",
  valid: "success",
  active: "success",
  connected: "success",
  sync: "success",
  completed: "success",
  ok: "success",
  failed: "danger",
  error: "danger",
  degraded: "danger",
  disabled: "danger",
  queued: "warning",
  pending: "warning",
  building: "warning",
  provisioning: "warning",
  restarting: "warning",
  processing: "warning",
};

export function AppStatusBadge({ status, pulse, className }: { status: string; pulse?: boolean; className?: string }) {
  return (
    <AppBadge tone={STATUS_TONE[status] ?? "neutral"} dot pulse={pulse} className={className}>
      {status}
    </AppBadge>
  );
}

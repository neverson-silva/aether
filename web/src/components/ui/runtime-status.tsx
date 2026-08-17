import { cn } from "./cn";

type StatusMeta = {
  icon: string;
  tone: string;
  badge: string;
  anim: string;
  dots?: boolean;
  blocks?: boolean;
};

const ACTIVE = new Set(["queued", "preparing", "cloning", "building", "pulling", "starting", "running", "health_checking", "healthy", "deploying"]);

export function isDeploymentActive(status: string): boolean {
  return ACTIVE.has(status);
}

const STATUS: Record<string, StatusMeta> = {
  queued: { icon: "schedule", tone: "text-[#fbbf24]", badge: "bg-[#fbbf24]/10 border-[#fbbf24]/20", anim: "rt-pulse-slow", dots: true },
  preparing: { icon: "auto_awesome", tone: "text-[#a78bfa]", badge: "bg-[#a78bfa]/10 border-[#a78bfa]/20 rt-shimmer", anim: "" },
  cloning: { icon: "call_split", tone: "text-[#60a5fa]", badge: "bg-[#60a5fa]/10 border-[#60a5fa]/20 rt-shimmer", anim: "" },
  building: { icon: "deployed_code", tone: "text-[#fb923c]", badge: "bg-[#fb923c]/10 border-[#fb923c]/20", anim: "", blocks: true },
  pulling: { icon: "download", tone: "text-[#38bdf8]", badge: "bg-[#38bdf8]/10 border-[#38bdf8]/20 rt-shimmer", anim: "" },
  starting: { icon: "play_circle", tone: "text-[#4ade80]", badge: "bg-[#4ade80]/10 border-[#4ade80]/20", anim: "rt-spin-slow" },
  deploying: { icon: "rocket_launch", tone: "text-[#fb923c]", badge: "bg-[#fb923c]/10 border-[#fb923c]/20", anim: "rt-pulse-slow" },
  running: { icon: "monitor_heart", tone: "text-[#4ade80]", badge: "bg-[#4ade80]/10 border-[#4ade80]/20", anim: "rt-breath rt-ring" },
  health_checking: { icon: "monitor_heart", tone: "text-[#4ade80]", badge: "bg-[#4ade80]/10 border-[#4ade80]/20", anim: "rt-heartbeat" },
  healthy: { icon: "favorite", tone: "text-[#4ade80]", badge: "bg-[#4ade80]/10 border-[#4ade80]/20", anim: "rt-heartbeat" },
  ready: { icon: "check_circle", tone: "text-[#4ade80]", badge: "bg-[#4ade80]/10 border-[#4ade80]/20", anim: "rt-ready-in" },
  success: { icon: "check_circle", tone: "text-[#4ade80]", badge: "bg-[#4ade80]/10 border-[#4ade80]/20", anim: "rt-ready-in" },
  failed: { icon: "error", tone: "text-error", badge: "bg-error/10 border-error/20", anim: "rt-failed-in" },
  cancelled: { icon: "cancel", tone: "text-on-surface-variant", badge: "bg-surface-container-high/30 border-outline-variant/40", anim: "" },
  rolled_back: { icon: "undo", tone: "text-[#60a5fa]", badge: "bg-[#60a5fa]/10 border-[#60a5fa]/20", anim: "rt-rollback-in" },
  stopped: { icon: "stop_circle", tone: "text-on-surface-variant", badge: "bg-surface-container-high/30 border-outline-variant/40", anim: "" },
};

function Blocks() {
  return (
    <span className="rt-blocks" aria-hidden>
      <span />
      <span />
      <span />
    </span>
  );
}

export function DeploymentStatus({ status, className }: { status: string; className?: string }) {
  const meta = STATUS[status] ?? { icon: "radio_button_unchecked", tone: "text-on-surface-variant", badge: "bg-surface-container-high/30 border-outline-variant/40", anim: "" };
  return (
    <span className={cn("rt-badge", meta.tone, meta.badge, meta.anim, className)}>
      <span key={status} className={cn("rt-icon", meta.anim && "rt-fade-in")}>
        {meta.blocks ? <Blocks /> : <span className="material-symbols-outlined">{meta.icon}</span>}
      </span>
      <span className="capitalize">{status.replace(/_/g, " ")}</span>
      {meta.dots && (
        <span className="rt-dots" aria-hidden>
          <span />
          <span />
          <span />
        </span>
      )}
    </span>
  );
}

import { cn } from "./cn";

export function StatusPill({
  status,
  pulse,
}: {
  status: string;
  pulse?: boolean;
}) {
  const tone =
    status === "ready" || status === "running" || status === "issued" || status === "valid" || status === "active" || status === "connected" || status === "sync"
      ? "text-[#4ade80] bg-[#4ade80]/10 border-[#4ade80]/20"
      : status === "failed" || status === "error" || status === "degraded" || status === "disabled" || status === "stopped"
        ? "text-error bg-error/10 border-error/20"
        : status === "pending" || status === "queued" || status === "building" || status === "deploying" || status === "pending deploy" || status === "restarting"
          ? "text-[#fbbf24] bg-[#fbbf24]/10 border-[#fbbf24]/20"
          : "text-on-surface-variant bg-surface-container-high/40 border-outline-variant/60";
  return (
    <span
      className={cn(
        "inline-flex items-center gap-xs px-2 py-0.5 rounded border font-code-md text-code-md",
        tone
      )}
    >
      <span
        className={cn(
          "w-2 h-2 rounded-full bg-current",
          pulse && "status-pulse"
        )}
      />
      {status}
    </span>
  );
}

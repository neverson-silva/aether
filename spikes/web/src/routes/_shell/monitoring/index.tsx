import { createFileRoute } from "@tanstack/react-router";
import { useMemo, useRef, useState } from "react";
import { useHostEvents, useHostLogs, useHostStats } from "../../../api/hooks";
import { cn } from "../../../components/ui";

function fmtBytes(n: number): string {
  if (n >= 1 << 30) return (n / (1 << 30)).toFixed(1) + " GB";
  if (n >= 1 << 20) return (n / (1 << 20)).toFixed(1) + " MB";
  if (n >= 1 << 10) return (n / (1 << 10)).toFixed(1) + " KB";
  return String(n) + " B";
}

function fmtUptime(sec: number): string {
  const d = Math.floor(sec / 86400);
  const h = Math.floor((sec % 86400) / 3600);
  const m = Math.floor((sec % 3600) / 60);
  return `${d}d ${h}h ${m}m`;
}

function Sparkline({ data, color, max = 100 }: { data: number[]; color: string; max?: number }) {
  if (data.length < 2) return <div className="flex-1 flex items-center justify-center text-on-surface-variant/40 text-[11px]">collecting…</div>;
  const w = 100;
  const h = 36;
  const step = w / (data.length - 1);
  const pts = data.map((v, i) => `${(i * step).toFixed(1)},${(h - (Math.min(v, max) / max) * h).toFixed(1)}`);
  const path = "M" + pts.join(" L");
  return (
    <svg className="w-full h-9" viewBox={`0 0 ${w} ${h}`} preserveAspectRatio="none">
      <path d={path} fill="none" stroke={color} strokeWidth="0.5" vectorEffect="non-scaling-stroke" />
    </svg>
  );
}

function QuickStat({ label, icon, iconClass, value, sub, bar, barClass }: { label: string; icon: string; iconClass: string; value: string; sub?: string; bar?: number; barClass?: string }) {
  return (
    <div className="bg-surface-container border border-outline-variant rounded-lg p-md">
      <div className="flex justify-between items-start mb-sm">
        <span className="font-label-caps text-label-caps text-on-surface-variant">{label}</span>
        <span className={cn("material-symbols-outlined", iconClass)} style={{ fontSize: 18 }}>{icon}</span>
      </div>
      <div className="font-headline-sm text-headline-sm text-on-surface mb-xs truncate">{value}</div>
      {bar !== undefined && (
        <div className="w-full bg-surface-bright h-1 rounded-full overflow-hidden">
          <div className={cn("h-full rounded-full", barClass ?? "bg-primary")} style={{ width: `${Math.min(100, bar)}%` }} />
        </div>
      )}
      {sub && <div className="font-label-caps text-label-caps text-on-surface-variant/70 truncate">{sub}</div>}
    </div>
  );
}

const EVENT_ICON: Record<string, { icon: string; color: string }> = {
  "app.deployed": { icon: "check_circle", color: "bg-[#4ade80]" },
  "app.deploy_failed": { icon: "error", color: "bg-error" },
  "deployment.created": { icon: "schedule", color: "bg-[#fbbf24]" },
  "deployment.starting": { icon: "play_circle", color: "bg-primary" },
  "backup.finished": { icon: "backup", color: "bg-[#4ade80]" },
  "server.registered": { icon: "dns", color: "bg-primary" },
};

function Monitoring() {
  const { stats, history } = useHostStats();
  const events = useHostEvents();
  const [follow, setFollow] = useState(true);
  const logLines = useHostLogs(follow);
  const logRef = useRef<HTMLDivElement>(null);
  const scrollToBottom = () => {
    if (logRef.current) logRef.current.scrollTo({ top: logRef.current.scrollHeight });
  };

  const netSpeed = useMemo(() => {
    if (!stats) return "—";
    return `${fmtBytes(stats.net.rx_bytes)} ↓ · ${fmtBytes(stats.net.tx_bytes)} ↑`;
  }, [stats]);

  const filteredLogs = useMemo(() => logLines.filter((l) => l.line.trim()), [logLines]);

  return (
    <div className="space-y-lg">
      <div className="flex flex-wrap justify-between items-end gap-md mb-lg">
        <div className="min-w-0">
          <h1 className="font-display-lg text-[clamp(1.5rem,4vw,3rem)] leading-[1.1] text-on-surface mb-xs">Monitoring</h1>
          <p className="font-body-md text-body-md text-on-surface-variant">
            Host telemetry for <span className="font-code-md text-primary bg-primary/10 px-1 rounded">{stats?.hostname || "this host"}</span>
          </p>
        </div>
        <div className="flex gap-sm flex-wrap">
          <span className="px-md py-sm border border-outline-variant rounded bg-surface hover:border-primary text-on-surface font-body-sm transition-colors flex items-center gap-2">
            <span className="material-symbols-outlined" style={{ fontSize: 16 }}>dns</span>
            {stats?.os || "Loading…"}
          </span>
          <button
            onClick={() => setFollow((f) => !f)}
            className={cn(
              "px-md py-sm rounded font-body-sm font-semibold transition-colors flex items-center gap-2",
              follow ? "bg-primary text-on-primary hover:bg-primary-fixed" : "bg-surface border border-outline-variant text-on-surface-variant"
            )}
          >
            <span className="material-symbols-outlined" style={{ fontSize: 16 }}>{follow ? "pause" : "play_arrow"}</span>
            {follow ? "Live" : "Paused"}
          </button>
        </div>
      </div>

      <div className="grid grid-cols-12 gap-md">
        <div className="col-span-12 xl:col-span-8 flex flex-col gap-md">
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-md">
            <QuickStat label="Host CPU Load" icon="memory" iconClass="text-primary" value={stats ? `${stats.cpu_percent.toFixed(1)}%` : "—"} bar={stats?.cpu_percent} />
            <QuickStat label="Node Memory" icon="storage" iconClass="text-secondary" value={stats ? `${stats.mem_percent.toFixed(1)}%` : "—"} sub={stats ? `${fmtBytes(stats.mem_used)} / ${fmtBytes(stats.mem_total)}` : undefined} bar={stats?.mem_percent} barClass="bg-secondary" />
            <QuickStat label="Host Network I/O" icon="swap_vert" iconClass="text-tertiary" value={netSpeed} sub={stats ? `uptime ${fmtUptime(stats.uptime)}` : undefined} />
            <QuickStat label="Node Disk" icon="hard_drive" iconClass="text-outline" value={stats ? `${stats.disk.percent.toFixed(1)}%` : "—"} sub={stats ? `${fmtBytes(stats.disk.used)} / ${fmtBytes(stats.disk.total)}` : undefined} bar={stats?.disk.percent} barClass="bg-tertiary" />
          </div>

          <div className="bg-surface-container border border-outline-variant rounded-lg p-md flex-1 min-h-[260px] flex flex-col">
            <div className="flex justify-between items-center mb-md border-b border-outline-variant pb-sm">
              <h2 className="font-label-caps text-label-caps text-on-surface">Node Performance Metrics</h2>
              <div className="flex gap-sm">
                <span className="flex items-center gap-xs font-label-caps text-label-caps text-on-surface-variant"><span className="w-2 h-2 rounded-full bg-primary inline-block" /> CPU</span>
                <span className="flex items-center gap-xs font-label-caps text-label-caps text-on-surface-variant"><span className="w-2 h-2 rounded-full bg-secondary inline-block" /> RAM</span>
              </div>
            </div>
            <div className="flex-1 flex flex-col gap-2">
              <div className="flex items-center gap-2">
                <span className="font-code-md text-code-md text-primary w-10 shrink-0">CPU</span>
                <div className="flex-1 min-w-0 bg-surface-bright rounded border border-outline-variant/50 p-1">
                  <Sparkline data={history.map((p) => p.cpu)} color="#b0c6ff" />
                </div>
                <span className="font-code-md text-code-md text-on-surface-variant w-12 text-right shrink-0">{stats ? stats.cpu_percent.toFixed(0) + "%" : "—"}</span>
              </div>
              <div className="flex items-center gap-2">
                <span className="font-code-md text-code-md text-secondary w-10 shrink-0">RAM</span>
                <div className="flex-1 min-w-0 bg-surface-bright rounded border border-outline-variant/50 p-1">
                  <Sparkline data={history.map((p) => p.mem)} color="#c0c1ff" />
                </div>
                <span className="font-code-md text-code-md text-on-surface-variant w-12 text-right shrink-0">{stats ? stats.mem_percent.toFixed(0) + "%" : "—"}</span>
              </div>
            </div>
          </div>
        </div>

        <div className="col-span-12 xl:col-span-4 flex flex-col gap-md">
          <div className="bg-surface-container border border-outline-variant rounded-lg p-md flex-1">
            <div className="flex justify-between items-center mb-md border-b border-outline-variant pb-sm">
              <h2 className="font-label-caps text-label-caps text-on-surface">Global Events</h2>
              <span className="material-symbols-outlined text-primary" style={{ fontSize: 16 }}>filter_list</span>
            </div>
            <div className="space-y-sm max-h-[300px] overflow-y-auto sidebar-scroll">
              {(events.data ?? []).map((e, i) => {
                const meta = EVENT_ICON[e.type] ?? { icon: "circle", color: "bg-outline-variant" };
                return (
                  <div key={i} className="flex gap-sm">
                    <div className="flex flex-col items-center">
                      <span className={cn("w-2 h-2 rounded-full mt-1", meta.color)} />
                      {i < (events.data?.length ?? 0) - 1 && <div className="w-px h-full bg-outline-variant mt-1" />}
                    </div>
                    <div className="pb-sm min-w-0">
                      <div className="font-label-caps text-label-caps text-on-surface-variant mb-1">{e.ts}</div>
                      <div className="font-body-sm text-body-sm text-on-surface">{e.title || e.type}</div>
                      {e.detail && <div className="font-code-md text-code-md text-on-surface-variant mt-1 break-words">{e.detail}</div>}
                    </div>
                  </div>
                );
              })}
              {(events.data ?? []).length === 0 && (
                <p className="font-body-sm text-body-sm text-on-surface-variant/60 text-center py-lg">No events yet.</p>
              )}
            </div>
          </div>
        </div>

        <div className="col-span-12 bg-surface-container border border-outline-variant rounded-lg flex flex-col h-[400px]">
          <div className="flex flex-wrap items-center justify-between gap-sm p-sm border-b border-outline-variant bg-surface-container-high rounded-t-lg">
            <div className="flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-primary pulse-live inline-block" />
              <h2 className="font-label-caps text-label-caps text-on-surface">System Events &amp; Host Logs</h2>
            </div>
            <div className="flex items-center gap-xs">
              <span className="px-2 py-1 bg-surface border border-outline-variant rounded font-label-caps text-label-caps text-on-surface-variant">
                {stats?.hostname || "host"}
              </span>
            </div>
          </div>
          <div ref={logRef} onScroll={() => { if (logRef.current && logRef.current.scrollTop + logRef.current.clientHeight > logRef.current.scrollHeight - 60) scrollToBottom(); }} className="flex-1 bg-[#0A0A0A] p-sm overflow-y-auto font-code-md text-code-md leading-relaxed rounded-b-lg">
            {filteredLogs.map((l, i) => (
              <div key={i} className="whitespace-pre-wrap break-all text-on-surface/85">{l.line}</div>
            ))}
            {filteredLogs.length === 0 && (
              <p className="text-on-surface-variant/50 py-lg text-center">Waiting for host logs…</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

export const Route = createFileRoute("/_shell/monitoring/")({
  component: Monitoring,
});

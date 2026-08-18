import { createFileRoute } from "@tanstack/react-router";
import { useMemo, useRef, useState } from "react";
import {
  useHostEvents,
  useHostLogs,
  useMonitoring,
  useMonitoringHistory,
  useResourceHistory,
  type MonitoringWindow,
} from "../../../hooks";
import { cn } from "../../../components/ui";
import { LineChart } from "./-components/LineChart";
import { ResourcesTable } from "./-components/ResourcesTable";
import { fmtBytes, fmtRate, fmtUptime } from "./-components/format";

const COLORS = {
  host: "#b0c6ff",
  aether: "#c0c1ff",
  user: "#ffb599",
  system: "#8c90a1",
  netRx: "#4ade80",
  netTx: "#c0c1ff",
};

const WINDOWS: { id: MonitoringWindow; label: string }[] = [
  { id: "5m", label: "5m" },
  { id: "15m", label: "15m" },
  { id: "1h", label: "1h" },
  { id: "6h", label: "6h" },
  { id: "24h", label: "24h" },
  { id: "7d", label: "7d" },
];

function Card({ label, icon, value, sub, bar, barClass }: { label: string; icon: string; value: string; sub?: string; bar?: number; barClass?: string }) {
  return (
    <div className="bg-surface-container border border-outline-variant rounded-lg p-md">
      <div className="flex justify-between items-start mb-sm">
        <span className="font-label-caps text-label-caps text-on-surface-variant">{label}</span>
        <span className="material-symbols-outlined text-on-surface-variant" style={{ fontSize: 18 }}>{icon}</span>
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

function SkeletonCard() {
  return (
    <div className="bg-surface-container border border-outline-variant rounded-lg p-md animate-pulse">
      <div className="h-3 w-20 bg-surface-bright rounded mb-md" />
      <div className="h-6 w-28 bg-surface-bright rounded mb-sm" />
      <div className="h-1 w-full bg-surface-bright rounded" />
    </div>
  );
}

function DistributionCard({
  title,
  color,
  agg,
  detail,
}: {
  title: string;
  color: string;
  agg: { cpu_of_host: number; mem_usage: number; net_rx_rate: number; net_tx_rate: number; storage_usage: number; available: boolean; running_count: number; count: number } | undefined;
  detail: string;
}) {
  if (!agg || !agg.available) {
    return (
      <div className="bg-surface-container border border-outline-variant rounded-lg p-md">
        <div className="flex items-center gap-2 mb-sm">
          <span className="w-2 h-2 rounded-full" style={{ background: color }} />
          <span className="font-label-caps text-label-caps text-on-surface">{title}</span>
        </div>
        <p className="font-body-sm text-body-sm text-on-surface-variant/60">unavailable</p>
      </div>
    );
  }
  return (
    <div className="bg-surface-container border border-outline-variant rounded-lg p-md">
      <div className="flex items-center gap-2 mb-md">
        <span className="w-2 h-2 rounded-full" style={{ background: color }} />
        <span className="font-label-caps text-label-caps text-on-surface">{title}</span>
      </div>
      <div className="grid grid-cols-2 gap-md">
        <div>
          <div className="font-label-caps text-label-caps text-on-surface-variant/60">CPU</div>
          <div className="font-headline-sm text-headline-sm text-on-surface">{agg.cpu_of_host.toFixed(1)}%</div>
          <div className="h-1 w-full bg-surface-bright rounded-full overflow-hidden mt-xs">
            <div className="h-full rounded-full" style={{ width: `${Math.min(100, agg.cpu_of_host)}%`, background: color }} />
          </div>
        </div>
        <div>
          <div className="font-label-caps text-label-caps text-on-surface-variant/60">Memory</div>
          <div className="font-headline-sm text-headline-sm text-on-surface">{fmtBytes(agg.mem_usage)}</div>
        </div>
        <div>
          <div className="font-label-caps text-label-caps text-on-surface-variant/60">Storage</div>
          <div className="font-headline-sm text-headline-sm text-on-surface">{fmtBytes(agg.storage_usage)}</div>
          <p className="font-label-caps text-label-caps text-on-surface-variant/40 mt-0.5">incl. stopped containers</p>
        </div>
        <div>
          <div className="font-label-caps text-label-caps text-on-surface-variant/60">Network</div>
          <div className="font-headline-sm text-headline-sm text-on-surface">
            <span className="text-[#4ade80]">↓{fmtRate(agg.net_rx_rate)}</span>{" "}
            <span className="text-on-surface-variant">↑{fmtRate(agg.net_tx_rate)}</span>
          </div>
        </div>
      </div>
      <p className="font-label-caps text-label-caps text-on-surface-variant/50 mt-sm">{detail}</p>
    </div>
  );
}

function ChartPanel({ title, subtitle, legend, children }: { title: string; subtitle: string; legend: { label: string; color: string }[]; children: React.ReactNode }) {
  return (
    <div className="bg-surface-container border border-outline-variant rounded-lg p-md flex flex-col">
      <div className="flex flex-wrap justify-between items-center gap-sm mb-md border-b border-outline-variant pb-sm">
        <div>
          <h2 className="font-label-caps text-label-caps text-on-surface">{title}</h2>
          <p className="font-body-sm text-body-sm text-on-surface-variant/60 mt-0.5">{subtitle}</p>
        </div>
        <div className="flex gap-sm">
          {legend.map((l) => (
            <span key={l.label} className="flex items-center gap-xs font-label-caps text-label-caps text-on-surface-variant">
              <span className="w-2 h-2 rounded-full inline-block" style={{ background: l.color }} />
              {l.label}
            </span>
          ))}
        </div>
      </div>
      <div className="flex-1 min-h-[140px]">{children}</div>
    </div>
  );
}

function Monitoring() {
  const { snapshot, connected } = useMonitoring();
  const events = useHostEvents();
  const [follow, setFollow] = useState(true);
  const logLines = useHostLogs(follow);
  const logRef = useRef<HTMLDivElement>(null);
  const [window, setWindow] = useState<MonitoringWindow>("1h");
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const history = useMonitoringHistory(window);
  const resourceHistory = useResourceHistory(selectedId, window);

  const scrollToBottom = () => {
    if (logRef.current) logRef.current.scrollTo({ top: logRef.current.scrollHeight });
  };

  const stats = snapshot?.host;
  const points = history.data ?? [];
  const selected = useMemo(() => snapshot?.resources.find((r) => r.id === selectedId) ?? null, [snapshot, selectedId]);

  const filteredLogs = useMemo(() => logLines.filter((l) => l.line.trim()), [logLines]);

  return (
    <div className="space-y-lg">
      <div className="flex flex-wrap justify-between items-end gap-md">
        <div className="min-w-0">
          <h1 className="font-display-lg text-[clamp(1.5rem,4vw,3rem)] leading-[1.1] text-on-surface mb-xs">Monitoring</h1>
          <p className="font-body-md text-body-md text-on-surface-variant">
            Host telemetry for <span className="font-code-md text-primary bg-primary/10 px-1 rounded">{stats?.hostname || "this host"}</span>
          </p>
        </div>
        <div className="flex gap-sm flex-wrap items-center">
          <span className="px-md py-sm border border-outline-variant rounded bg-surface text-on-surface font-body-sm flex items-center gap-2">
            <span className={cn("w-2 h-2 rounded-full", connected ? "bg-[#4ade80]" : "bg-error")} />
            {connected ? "Live" : "Reconnecting…"}
          </span>
          <span
            className={cn(
              "px-md py-sm border rounded bg-surface font-body-sm flex items-center gap-2",
              stats?.source && stats.source !== "runtime" ? "border-[#4ade80]/30 text-[#4ade80]" : "border-outline-variant text-on-surface-variant",
            )}
            title={
              stats?.source && stats.source !== "runtime"
                ? "Metrics collected by the host agent running natively on the host machine"
                : "No host agent detected — metrics come from the container runtime (VM). Install scripts/host-agent.sh on the host."
            }
          >
            <span className="material-symbols-outlined" style={{ fontSize: 16 }}>monitor_heart</span>
            {stats?.source && stats.source !== "runtime" ? "Host: real machine" : "Host: runtime (VM)"}
          </span>
          <span className="px-md py-sm border border-outline-variant rounded bg-surface text-on-surface-variant font-body-sm flex items-center gap-2">
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
            {follow ? "Live logs" : "Paused"}
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-md">
        {!snapshot ? (
          <>
            <SkeletonCard />
            <SkeletonCard />
            <SkeletonCard />
            <SkeletonCard />
          </>
        ) : (
          <>
            <Card
              label="Host CPU"
              icon="memory"
              value={`${stats!.cpu_percent.toFixed(1)}%`}
              sub={`${stats!.cpu_cores} vCPU · load ${(stats!.load?.[0] ?? 0).toFixed(2)}`}
              bar={stats!.cpu_percent}
            />
            <Card
              label="Host Memory"
              icon="data_usage"
              value={`${stats!.mem_percent.toFixed(1)}%`}
              sub={`${fmtBytes(stats!.mem_used)} / ${fmtBytes(stats!.mem_total)}`}
              bar={stats!.mem_percent}
              barClass="bg-secondary"
            />
            <Card
              label="Host Storage"
              icon="hard_drive"
              value={`${stats!.disk_percent.toFixed(1)}%`}
              sub={`${fmtBytes(stats!.disk_used)} / ${fmtBytes(stats!.disk_total)}`}
              bar={stats!.disk_percent}
              barClass="bg-tertiary"
            />
            <Card
              label="Host Network"
              icon="swap_vert"
              value={`↓${fmtRate(stats!.net_rx_rate)} ↑${fmtRate(stats!.net_tx_rate)}`}
              sub={`uptime ${fmtUptime(stats!.uptime)}`}
            />
          </>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-md">
        <DistributionCard
          title="Aether"
          color={COLORS.aether}
          agg={snapshot?.aether}
          detail={`${snapshot?.aether?.running_count ?? 0} active / ${snapshot?.aether?.count ?? 0} resources`}
        />
        <DistributionCard
          title="User Workloads"
          color={COLORS.user}
          agg={snapshot?.user}
          detail={`${snapshot?.user?.running_count ?? 0} active / ${snapshot?.user?.count ?? 0} resources`}
        />
        <DistributionCard
          title="System / unaccounted"
          color={COLORS.system}
          agg={snapshot?.system}
          detail="Host minus Aether and user attribution"
        />
      </div>
      <p className="font-body-sm text-body-sm text-on-surface-variant/60 -mt-md">
        Approximate reconciliation: host used memory excludes filesystem cache, while container working-set memory includes part of it, so the three
        buckets may not sum exactly to the host.
      </p>

      <div className="bg-surface-container border border-outline-variant rounded-lg p-md">
        <div className="flex flex-wrap justify-between items-center gap-sm mb-md border-b border-outline-variant pb-sm">
          <div>
            <h2 className="font-label-caps text-label-caps text-on-surface">Trends</h2>
            <p className="font-body-sm text-body-sm text-on-surface-variant/60 mt-0.5">
              History is persisted in the platform database; windows 5m through 7d.
            </p>
          </div>
          <div className="flex gap-xs">
            {WINDOWS.map((w) => (
              <button
                key={w.id}
                onClick={() => setWindow(w.id)}
                className={cn(
                  "px-md py-sm rounded font-label-caps text-label-caps transition-colors border",
                  window === w.id ? "bg-primary/10 text-primary border-primary/30" : "border-outline-variant/50 text-on-surface-variant hover:text-on-surface"
                )}
              >
                {w.label}
              </button>
            ))}
          </div>
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-md">
          <ChartPanel
            title="CPU"
            subtitle="How has consumption evolved?"
            legend={[
              { label: "Host", color: COLORS.host },
              { label: "Aether", color: COLORS.aether },
              { label: "User", color: COLORS.user },
            ]}
          >
            <LineChart
              unit={(v) => v.toFixed(0) + "%"}
              series={[
                { name: "Host", color: COLORS.host, points: points.map((p) => p.host_cpu) },
                { name: "Aether", color: COLORS.aether, points: points.map((p) => p.aether_cpu) },
                { name: "User", color: COLORS.user, points: points.map((p) => p.user_cpu) },
              ]}
            />
          </ChartPanel>
          <ChartPanel
            title="Memory"
            subtitle="Are we approaching saturation?"
            legend={[
              { label: "Host", color: COLORS.host },
              { label: "Aether", color: COLORS.aether },
              { label: "User", color: COLORS.user },
            ]}
          >
            <LineChart
              unit={(v) => v.toFixed(0) + "%"}
              series={[
                { name: "Host", color: COLORS.host, points: points.map((p) => p.host_mem) },
                { name: "Aether", color: COLORS.aether, points: points.map((p) => p.aether_mem_pct) },
                { name: "User", color: COLORS.user, points: points.map((p) => p.user_mem_pct) },
              ]}
            />
          </ChartPanel>
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-md mt-md">
          <ChartPanel
            title="Network"
            subtitle="Any traffic spikes?"
            legend={[
              { label: "Received", color: COLORS.netRx },
              { label: "Sent", color: COLORS.netTx },
            ]}
          >
            <LineChart
              unit={fmtRate}
              series={[
                { name: "RX", color: COLORS.netRx, points: points.map((p) => p.net_rx) },
                { name: "TX", color: COLORS.netTx, points: points.map((p) => p.net_tx) },
              ]}
            />
          </ChartPanel>
          <ChartPanel
            title="Distribution"
            subtitle="Who is consuming the resources?"
            legend={[
              { label: "Aether", color: COLORS.aether },
              { label: "User", color: COLORS.user },
            ]}
          >
            <LineChart
              unit={(v) => v.toFixed(0) + "%"}
              series={[
                { name: "Aether", color: COLORS.aether, points: points.map((p) => p.aether_cpu) },
                { name: "User", color: COLORS.user, points: points.map((p) => p.user_cpu) },
              ]}
            />
          </ChartPanel>
        </div>
      </div>

      <ResourcesTable resources={snapshot?.resources ?? []} selectedId={selectedId} onSelect={setSelectedId} />

      {selected && (
        <div className="bg-surface-container border border-outline-variant rounded-lg p-md">
          <div className="flex flex-wrap justify-between items-center gap-sm mb-md border-b border-outline-variant pb-sm">
            <div className="flex items-center gap-2">
              <span className="material-symbols-outlined text-on-surface-variant" style={{ fontSize: 18 }}>apps</span>
              <h2 className="font-label-caps text-label-caps text-on-surface">{selected.name}</h2>
              <span className={cn("px-2 py-0.5 rounded border font-label-caps text-label-caps capitalize", selected.state === "running" ? "bg-[#4ade80]/10 text-[#4ade80] border-[#4ade80]/20" : "bg-outline/10 text-on-surface-variant border-outline-variant/30")}>
                {selected.state}
              </span>
            </div>
            <button onClick={() => setSelectedId(null)} className="font-label-caps text-label-caps text-primary hover:text-primary-fixed">
              Close
            </button>
          </div>
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-md">
            <div className="bg-surface rounded-lg border border-outline-variant/50 p-md">
              <div className="font-label-caps text-label-caps text-on-surface-variant mb-sm">CPU ({window})</div>
              {resourceHistory.data && resourceHistory.data.length >= 2 ? (
                <LineChart unit={(v) => v.toFixed(1) + "%"} series={[{ name: "CPU", color: COLORS.user, points: resourceHistory.data.map((p) => p.cpu) }]} />
              ) : (
                <p className="font-body-sm text-body-sm text-on-surface-variant/50">Collecting history…</p>
              )}
            </div>
            <div className="bg-surface rounded-lg border border-outline-variant/50 p-md">
              <div className="font-label-caps text-label-caps text-on-surface-variant mb-sm">Memory ({window})</div>
              {resourceHistory.data && resourceHistory.data.length >= 2 ? (
                <LineChart unit={fmtBytes} series={[{ name: "Mem", color: COLORS.aether, points: resourceHistory.data.map((p) => p.mem) }]} />
              ) : (
                <p className="font-body-sm text-body-sm text-on-surface-variant/50">Collecting history…</p>
              )}
            </div>
          </div>
        </div>
      )}

      <div className="bg-surface-container border border-outline-variant rounded-lg flex flex-col h-[400px]">
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
        <div
          ref={logRef}
          onScroll={() => {
            if (logRef.current && logRef.current.scrollTop + logRef.current.clientHeight > logRef.current.scrollHeight - 60) scrollToBottom();
          }}
          className="flex-1 bg-[#0A0A0A] p-sm overflow-y-auto font-code-md text-code-md leading-relaxed rounded-b-lg"
        >
          {filteredLogs.map((l, i) => (
            <div key={i} className="whitespace-pre-wrap break-all text-on-surface/85">{l.line}</div>
          ))}
          {filteredLogs.length === 0 && <p className="text-on-surface-variant/50 py-lg text-center">Waiting for host logs…</p>}
        </div>
      </div>

      <div className="bg-surface-container border border-outline-variant rounded-lg p-md">
        <div className="flex justify-between items-center mb-md border-b border-outline-variant pb-sm">
          <h2 className="font-label-caps text-label-caps text-on-surface">Global Events</h2>
          <span className="material-symbols-outlined text-primary" style={{ fontSize: 16 }}>filter_list</span>
        </div>
        <div className="space-y-sm max-h-[240px] overflow-y-auto sidebar-scroll">
          {(events.data ?? []).length === 0 && <p className="font-body-sm text-body-sm text-on-surface-variant/60 text-center py-lg">No events yet.</p>}
          {(events.data ?? []).map((e, i) => (
            <div key={i} className="flex gap-sm">
              <div className="flex flex-col items-center">
                <span className="w-2 h-2 rounded-full mt-1 bg-outline-variant" />
                {i < (events.data?.length ?? 0) - 1 && <div className="w-px h-full bg-outline-variant mt-1" />}
              </div>
              <div className="pb-sm min-w-0">
                <div className="font-label-caps text-label-caps text-on-surface-variant mb-1">{e.ts}</div>
                <div className="font-body-sm text-body-sm text-on-surface">{e.title || e.type}</div>
                {e.detail && <div className="font-code-md text-code-md text-on-surface-variant mt-1 break-words">{e.detail}</div>}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

export const Route = createFileRoute("/_shell/monitoring/")({
  component: Monitoring,
});

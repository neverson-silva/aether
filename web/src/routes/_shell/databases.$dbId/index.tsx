import { createFileRoute, useParams, useRouter } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import {
  useDatabaseDetail,
  useDatabaseStats,
  useDeleteDatabase,
  useDatabaseDeploy,
  useDatabaseRebuild,
  useDatabaseStart,
  useDatabaseStop,
  useDatabaseDeployments,
} from "../../../hooks";
import type { DatabaseDeployment } from "../../../hooks/use-database-deployments";
import { getServer } from "../../../api/client";
import { TechIcon } from "../../../components/TechIcon";
import { Button, Card, ConfirmDialog, StatusPill, DeploymentStatus, useToast } from "../../../components/ui";
import { DbTerminal } from "./-components/DbTerminal";
import { DatabaseDeploymentLogModal } from "./-components/DatabaseDeploymentLogModal";
import { DomainsPanel } from "../../../components/DomainsPanel";
import { BackupTab } from "./-components/BackupTab";

type Tab = "overview" | "deployments" | "domains" | "previews" | "terminal" | "metrics" | "logs" | "settings" | "backup";

const TABS: { id: Tab; label: string; icon: string }[] = [
  { id: "overview", label: "Overview", icon: "dashboard" },
  { id: "deployments", label: "Deployments", icon: "rocket_launch" },
  { id: "domains", label: "Domains", icon: "language" },
  { id: "previews", label: "Previews", icon: "preview" },
  { id: "terminal", label: "Terminal", icon: "terminal" },
  { id: "metrics", label: "Metrics", icon: "monitor_heart" },
  { id: "logs", label: "Logs", icon: "terminal" },
  { id: "settings", label: "Settings", icon: "settings" },
  { id: "backup", label: "Backup", icon: "backup" },
];

function dbPill(status: string): { status: string; pulse: boolean } {
  if (status === "ready" || status === "running") return { status: "running", pulse: true };
  if (status === "creating" || status === "starting" || status === "provisioning") return { status: "pending deploy", pulse: true };
  if (status === "failed" || status === "error") return { status: "error", pulse: false };
  if (status === "stopped") return { status: "stopped", pulse: false };
  return { status, pulse: false };
}

function fmtBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KiB`;
  if (n < 1024 * 1024 * 1024) return `${(n / (1024 * 1024)).toFixed(1)} MiB`;
  return `${(n / (1024 * 1024 * 1024)).toFixed(2)} GiB`;
}

function DatabaseDetail() {
  const { dbId } = useParams({ strict: false }) as { dbId: string };
  const router = useRouter();
  const { data } = useDatabaseDetail(dbId);
  const deleteDb = useDeleteDatabase();
  const deploy = useDatabaseDeploy();
  const rebuild = useDatabaseRebuild();
  const start = useDatabaseStart();
  const stop = useDatabaseStop();
  const { data: deployments } = useDatabaseDeployments(dbId);
  const { toast } = useToast();
  const [tab, setTab] = useState<Tab>("overview");
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [copied, setCopied] = useState(false);
  const [logDep, setLogDep] = useState<DatabaseDeployment | null>(null);
  const [logLines, setLogLines] = useState<string[]>([]);
  const [logFollow, setLogFollow] = useState(true);
  const logRef = useRef<HTMLDivElement>(null);
  const { data: stats, isLoading: loadingStats, refetch: reloadStats } = useDatabaseStats(dbId);

  const db = data?.database;
  const running = db?.status === "running" || db?.status === "ready";

  useEffect(() => {
    if (!logFollow) return;
    const es = new EventSource(`/api/v1/databases/${dbId}/logs?follow=1`, { withCredentials: true });
    es.addEventListener("log", (e: MessageEvent) => {
      setLogLines((prev) => [...prev.slice(-800), String(e.data)]);
    });
    return () => es.close();
  }, [dbId, logFollow]);

  useEffect(() => {
    if (logFollow) logRef.current?.scrollTo({ top: logRef.current.scrollHeight });
  }, [logLines, logFollow]);

  const copyDsn = async () => {
    if (!data?.dsn) return;
    await navigator.clipboard.writeText(data.dsn);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  const maskedDsn = (dsn: string) => dsn.replace(/\/\/[^@/]+@/, "//***@");

  const run = (fn: () => Promise<unknown>, okMsg: string) => {
    fn().then(
      () => toast(okMsg),
      (e) => toast(e instanceof Error ? e.message : "operation failed", "error")
    );
  };

  return (
    <div className="space-y-lg">
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-6 mb-8">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-3 mb-2">
            <TechIcon name={db?.engine} size={32} className="text-primary shrink-0" />
            <h2 className="font-display-lg text-[clamp(1.5rem,4vw,3rem)] leading-[1.1] text-on-surface truncate">{db?.name ?? "Database"}</h2>
            {db && <StatusPill status={dbPill(db.status).status} pulse={dbPill(db.status).pulse} />}
          </div>
          <p className="font-body-md text-body-md text-on-surface-variant">
            {db?.engine} {db?.version} · port :{db?.port}
          </p>
        </div>
        <div className="flex items-center gap-3 flex-wrap">
          <Button
            onClick={() => router.navigate({ to: "/studio/$dbId", params: { dbId } })}
          >
            <span className="material-symbols-outlined text-[18px]">table_view</span>
            Open Studio
          </Button>
          <Button
            onClick={() => {
              if (data?.dsn) window.open(`http://${data.public_host || "127.0.0.1"}:${db?.port}`, "_blank");
              else toast("Add a domain to open the URL", "error");
            }}
          >
            <span className="material-symbols-outlined text-[18px]">open_in_new</span>
            Open port
          </Button>
          <Button variant="danger" onClick={() => setConfirmDelete(true)}>
            <span className="material-symbols-outlined text-[18px]">delete</span>
            Delete
          </Button>
        </div>
      </div>

      <div className="flex items-center gap-sm border-b border-outline-variant overflow-x-auto">
        {TABS.map((t) => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={`flex items-center gap-sm px-md py-2.5 font-label-caps text-label-caps uppercase border-b-2 -mb-px transition-colors whitespace-nowrap ${
              tab === t.id ? "border-primary text-primary" : "border-transparent text-on-surface-variant hover:text-on-surface"
            }`}
          >
            <span className="material-symbols-outlined text-[16px]">{t.icon}</span>
            {t.label}
          </button>
        ))}
      </div>

      {tab === "overview" && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-1 flex flex-col gap-6">
            <div className="bg-surface-container-lowest border border-outline-variant rounded-xl p-6">
              <div className="flex items-center justify-between mb-6">
                <div>
                  <h3 className="font-headline-sm text-headline-sm text-on-surface">Deploy Settings</h3>
                  <p className="text-body-sm text-on-surface-variant">Deploy, rebuild or control this database</p>
                </div>
                <span className="px-3 py-1.5 bg-surface-container-high border border-outline-variant rounded text-body-sm font-medium text-on-surface-variant uppercase">
                  {db?.engine}
                </span>
              </div>
              <div className="flex flex-wrap items-center gap-3">
                <Button className="bg-on-surface text-surface hover:bg-on-surface/90" leftIcon="rocket_launch" onClick={() => run(() => deploy.mutateAsync(dbId), "Deploy started")}>
                  Deploy
                </Button>
                <Button variant="subtle" leftIcon="refresh" onClick={() => run(() => rebuild.mutateAsync(dbId), "Rebuild started")}>
                  Rebuild
                </Button>
                {running ? (
                  <Button variant="danger" leftIcon="stop_circle" onClick={() => run(() => stop.mutateAsync(dbId), "Database stopped")}>
                    Stop
                  </Button>
                ) : (
                  <Button variant="success" leftIcon="play_arrow" onClick={() => run(() => start.mutateAsync(dbId), "Database started")}>
                    Start
                  </Button>
                )}
              </div>
            </div>

            <Card>
              <h3 className="font-label-caps text-label-caps text-on-surface-variant mb-6 uppercase">Database details</h3>
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <span className="block font-label-caps text-label-caps text-on-surface-variant mb-1">Engine</span>
                    <span className="font-body-md text-body-md text-on-surface">{db?.engine}</span>
                  </div>
                  <div>
                    <span className="block font-label-caps text-label-caps text-on-surface-variant mb-1">Version</span>
                    <span className="font-body-md text-body-md text-on-surface">{db?.version}</span>
                  </div>
                  <div>
                    <span className="block font-label-caps text-label-caps text-on-surface-variant mb-1">Port</span>
                    <span className="font-body-md text-body-md text-on-surface">:{db?.port}</span>
                  </div>
                  <div>
                    <span className="block font-label-caps text-label-caps text-on-surface-variant mb-1">DB name</span>
                    <span className="font-body-md text-body-md text-on-surface">{db?.db_name || db?.name}</span>
                  </div>
                  <div>
                    <span className="block font-label-caps text-label-caps text-on-surface-variant mb-1">User</span>
                    <span className="font-body-md text-body-md text-on-surface">{db?.user || "aether"}</span>
                  </div>
                  <div>
                    <span className="block font-label-caps text-label-caps text-on-surface-variant mb-1">Memory</span>
                    <span className="font-body-md text-body-md text-on-surface">{db?.mem_mb ? `${db.mem_mb} MiB` : "default"}</span>
                  </div>
                </div>
              </div>
            </Card>

            <div className="grid grid-cols-2 gap-4">
              <div className="bg-surface border border-outline-variant rounded-xl p-4 flex flex-col justify-between">
                <span className="font-label-caps text-label-caps text-on-surface-variant">CPU Usage</span>
                <div className="mt-4">
                  <span className="font-headline-sm text-headline-sm text-on-surface">{stats?.stats?.cpu_percent?.toFixed(0) ?? "—"}%</span>
                  <div className="w-full h-1 bg-surface-container-high mt-2 rounded-full overflow-hidden">
                    <div className="h-full bg-primary rounded-full" style={{ width: `${Math.min(100, stats?.stats?.cpu_percent ?? 0)}%` }} />
                  </div>
                </div>
              </div>
              <div className="bg-surface border border-outline-variant rounded-xl p-4 flex flex-col justify-between">
                <span className="font-label-caps text-label-caps text-on-surface-variant">Memory</span>
                <div className="mt-4">
                  <span className="font-headline-sm text-headline-sm text-on-surface">{stats?.stats?.mem_bytes != null ? fmtBytes(stats.stats.mem_bytes) : "—"}</span>
                  <div className="w-full h-1 bg-surface-container-high mt-2 rounded-full overflow-hidden">
                    <div className="h-full bg-secondary rounded-full" style={{ width: `${Math.min(100, stats?.stats?.mem_limit ? ((stats.stats.mem_bytes ?? 0) / stats.stats.mem_limit) * 100 : 0)}%` }} />
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div className="lg:col-span-2 flex flex-col gap-6">
            <div className="bg-surface border border-outline-variant rounded-xl p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Connection string</h3>
                <Button variant="ghost" onClick={copyDsn} disabled={!data?.dsn}>
                  <span className="material-symbols-outlined text-[16px]">{copied ? "check" : "content_copy"}</span>
                  {copied ? "Copied" : "Copy"}
                </Button>
              </div>
              <div className="bg-surface-container-lowest border border-outline-variant rounded-lg p-md font-code-md text-code-md text-on-surface break-all">
                {data?.dsn ? maskedDsn(data.dsn) : "Loading..."}
              </div>
              <p className="font-code-md text-code-md text-on-surface-variant/60 mt-sm">
                The connection string is only shown here and in the environment variables of this project's services.
              </p>
            </div>

            <div className="bg-surface border border-outline-variant rounded-xl p-6">
              <h3 className="font-label-caps text-label-caps text-on-surface-variant mb-4 uppercase">Status</h3>
              <div className="flex items-center gap-md">
                <StatusPill status={dbPill(db?.status ?? "").status} pulse={dbPill(db?.status ?? "").pulse} />
                <span className="font-body-sm text-body-sm text-on-surface-variant capitalize">{db?.status ?? "—"}</span>
                <Button variant="ghost" onClick={() => void reloadStats()} disabled={loadingStats}>
                  {loadingStats ? "Loading..." : "Refresh"}
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}

      {tab === "deployments" && (
        <Card>
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">Deployments</h2>
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="border-b border-outline-variant font-label-caps text-label-caps text-on-surface-variant/60 uppercase">
                  <th className="px-sm py-2">#</th>
                  <th className="px-sm py-2">Status</th>
                  <th className="px-sm py-2">Trigger</th>
                  <th className="px-sm py-2">Container</th>
                  <th className="px-sm py-2">Started</th>
                  <th className="px-sm py-2">Error</th>
                </tr>
              </thead>
              <tbody>
                {(deployments ?? []).map((d) => (
                  <tr key={d.id} onClick={() => setLogDep(d)} className="border-b border-outline-variant/40 hover:bg-surface-container-high transition-colors cursor-pointer">
                    <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">#{d.number}</td>
                    <td className="px-sm py-2">
                      <DeploymentStatus status={d.status} />
                    </td>
                    <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{d.trigger}</td>
                    <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant/60 max-w-[220px] truncate">{d.container_id || "—"}</td>
                    <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant/60">
                      {d.created_at ? new Date(d.created_at).toLocaleString(undefined, { day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit" }) : "—"}
                    </td>
                    <td className="px-sm py-2">
                      {d.error ? (
                        <span className="font-code-md text-code-md text-error/80 max-w-[180px] truncate inline-block align-middle" title={d.error}>{d.error}</span>
                      ) : (
                        "—"
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {(deployments ?? []).length === 0 && (
            <p className="font-body-sm text-body-sm text-on-surface-variant p-sm">
              No deployments yet. Use Deploy to start the database container from the official image.
            </p>
          )}
        </Card>
      )}

      <DatabaseDeploymentLogModal dbId={dbId} deployment={logDep} onClose={() => setLogDep(null)} />

      {tab === "previews" && (
        <Card>
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">Previews</h2>
          <p className="font-body-sm text-body-sm text-on-surface-variant">
            Deploy previews are available for git-based services. Databases are provisioned from official images and do not support previews.
          </p>
        </Card>
      )}

      {tab === "terminal" && <DbTerminal dbId={dbId} />}

      {tab === "domains" && <DomainsPanel kind="databases" id={dbId} />}

      {tab === "logs" && (
        <Card>
          <div className="flex items-center justify-between mb-md">
            <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Container logs</h2>
            <div className="flex items-center gap-sm">
              <span className={`font-code-md text-code-md ${logFollow ? "text-[#4ade80]" : "text-on-surface-variant/60"}`}>
                {logFollow ? "● Live" : "○ Paused"}
              </span>
              <Button variant="ghost" onClick={() => setLogFollow((f) => !f)}>
                <span className="material-symbols-outlined text-[16px]">{logFollow ? "pause" : "play_arrow"}</span>
                {logFollow ? "Pause" : "Resume"}
              </Button>
            </div>
          </div>
          <div
            ref={logRef}
            className="bg-surface-container-lowest border border-outline-variant rounded-lg p-md font-code-md text-code-md text-on-surface overflow-auto h-[480px] whitespace-pre-wrap sidebar-scroll"
          >
            {logLines.length === 0 && <span className="text-on-surface-variant/60">Waiting for logs...</span>}
            {logLines.map((line, i) => (
              <div key={i}>{line}</div>
            ))}
          </div>
        </Card>
      )}

      {tab === "metrics" && (
        <Card>
          <div className="flex items-center justify-between mb-md">
            <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Metrics</h2>
            <StatusPill status={stats?.state ?? db?.status ?? "unknown"} pulse={stats?.state === "running"} />
          </div>
          {stats ? (
            <div className="grid grid-cols-2 md:grid-cols-4 gap-md">
              <div className="bg-surface-container-low rounded-lg p-md">
                <p className="font-label-caps text-label-caps text-on-surface-variant uppercase">CPU</p>
                <p className="font-headline-sm text-headline-sm text-on-surface">{stats.stats?.cpu_percent?.toFixed(2) ?? "—"}%</p>
              </div>
              <div className="bg-surface-container-low rounded-lg p-md">
                <p className="font-label-caps text-label-caps text-on-surface-variant uppercase">Memory</p>
                <p className="font-headline-sm text-headline-sm text-on-surface">{stats.stats?.mem_bytes != null ? fmtBytes(stats.stats.mem_bytes) : "—"}</p>
              </div>
              <div className="bg-surface-container-low rounded-lg p-md">
                <p className="font-label-caps text-label-caps text-on-surface-variant uppercase">Limit</p>
                <p className="font-headline-sm text-headline-sm text-on-surface">{stats.stats?.mem_limit != null ? fmtBytes(stats.stats.mem_limit) : "—"}</p>
              </div>
              <div className="bg-surface-container-low rounded-lg p-md">
                <p className="font-label-caps text-label-caps text-on-surface-variant uppercase">PIDs</p>
                <p className="font-headline-sm text-headline-sm text-on-surface">{stats.stats?.mem_percent != null ? `${stats.stats.mem_percent.toFixed(1)}%` : "—"}</p>
              </div>
            </div>
          ) : (
            <div className="text-center py-lg">
              <p className="font-body-sm text-body-sm text-on-surface-variant">{loadingStats ? "Loading metrics..." : "No live metrics available (no active container)."}</p>
            </div>
          )}
        </Card>
      )}

      {tab === "backup" && <BackupTab dbId={dbId} dbName={db?.name} />}

      {tab === "settings" && (
        <div className="grid grid-cols-1 xl:grid-cols-3 gap-lg">
          <Card>
            <div className="flex items-center justify-between mb-md">
              <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Danger zone</h2>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mb-md">
              Delete this database, its container and credentials. The data volume is kept on disk.
            </p>
            <Button variant="danger" onClick={() => setConfirmDelete(true)}>
              <span className="material-symbols-outlined text-[16px]">delete</span>
              Delete database
            </Button>
          </Card>
        </div>
      )}

      <ConfirmDialog
        open={confirmDelete}
        onClose={() => setConfirmDelete(false)}
        onConfirm={() =>
          deleteDb.mutate(dbId, {
            onSuccess: () => {
              toast("Database deleted");
              window.location.href = `/projects/${db?.project_id}`;
            },
            onError: (e) => toast(e.message, "error"),
          })
        }
        title="Delete database"
        description={`Remove ${db?.name} and its container? The volume is kept.`}
        confirmLabel="Delete"
        danger
      />
    </div>
  );
}

export const Route = createFileRoute("/_shell/databases/$dbId/")({
  component: DatabaseDetail,
});

import { createFileRoute, useParams, useRouter } from "@tanstack/react-router";
import { useEffect, useMemo, useState, type ComponentProps } from "react";
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
import { AlertDialog, Badge, Button, Card, EmptyState, LogViewer, RuntimeStatus, Skeleton, useToast } from "@aether/design-system";
import { ArrowSquareOut, Check, Copy, Database as DatabaseIcon, Gear, Globe, Play, RocketLaunch, Stop, Table, TerminalWindow, Trash, X } from "@phosphor-icons/react";
import { DbTerminal } from "./-components/DbTerminal";
import { DatabaseDeploymentLogModal } from "./-components/DatabaseDeploymentLogModal";
import { DomainsPanel } from "../../../components/DomainsPanel";
import { BackupTab } from "./-components/BackupTab";
import { isRuntimeLive, mapRuntimeStatus } from "../../../lib/runtime-status";
import { toDeploymentLogLines } from "../../../lib/deployment-log-lines";

type Tab = "overview" | "deployments" | "domains" | "previews" | "terminal" | "metrics" | "logs" | "settings" | "backup";
type DesignIcon = NonNullable<ComponentProps<typeof Button>["icon"]>;
const designIcon = (icon: typeof RocketLaunch) => icon as unknown as DesignIcon;

const TABS: { id: Tab; label: string; icon: typeof DatabaseIcon }[] = [
  { id: "overview", label: "Overview", icon: DatabaseIcon },
  { id: "deployments", label: "Deployments", icon: RocketLaunch },
  { id: "domains", label: "Domains", icon: Globe },
  { id: "previews", label: "Previews", icon: Globe },
  { id: "terminal", label: "Terminal", icon: TerminalWindow },
  { id: "metrics", label: "Metrics", icon: Gear },
  { id: "logs", label: "Logs", icon: TerminalWindow },
  { id: "settings", label: "Settings", icon: Gear },
  { id: "backup", label: "Backup", icon: DatabaseIcon },
];

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
  const { add } = useToast();
  const [tab, setTab] = useState<Tab>("overview");
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [copied, setCopied] = useState(false);
  const [logDep, setLogDep] = useState<DatabaseDeployment | null>(null);
  const [logLines, setLogLines] = useState<string[]>([]);
  const [logFollow, setLogFollow] = useState(true);
  const { data: stats, isLoading: loadingStats, refetch: reloadStats } = useDatabaseStats(dbId);

  const db = data?.database;
  const running = db?.status === "running" || db?.status === "ready";
  const runtimeStatus = mapRuntimeStatus(stats?.state ?? db?.status);

  useEffect(() => {
    if (!logFollow) return;
    const es = new EventSource(`/api/v1/databases/${dbId}/logs?follow=1`, { withCredentials: true });
    es.addEventListener("log", (e: MessageEvent) => {
      setLogLines((prev) => [...prev.slice(-800), String(e.data)]);
    });
    return () => es.close();
  }, [dbId, logFollow]);

  const formattedLogLines = useMemo(
    () => toDeploymentLogLines(logLines.join("\n")),
    [logLines],
  );

  const copyDsn = async () => {
    if (!data?.dsn) return;
    await navigator.clipboard.writeText(data.dsn);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  const maskedDsn = (dsn: string) => dsn.replace(/\/\/[^@/]+@/, "//***@");

  const run = (fn: () => Promise<unknown>, okMsg: string) => {
    fn().then(
      () => add({ title: okMsg, tone: "success" }),
      (e) => add({ title: "Operation failed", description: e instanceof Error ? e.message : "The operation failed.", tone: "error" })
    );
  };

  return (
    <div className="space-y-lg">
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-6 mb-8">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-3 mb-2">
            <TechIcon name={db?.engine} size={32} className="text-primary shrink-0" />
            <h2 className="font-display-lg text-[clamp(1.5rem,4vw,3rem)] leading-[1.1] text-on-surface truncate">{db?.name ?? "Database"}</h2>
            {db && <RuntimeStatus status={runtimeStatus} live={isRuntimeLive(runtimeStatus)} />}
          </div>
          <p className="font-body-md text-body-md text-on-surface-variant">
            {db?.engine} {db?.version} · port :{db?.port}
          </p>
        </div>
        <div className="flex items-center gap-3 flex-wrap">
          <Button
            onClick={() => router.navigate({ to: "/studio/$dbId", params: { dbId } })}
          >
            <Table size={18} />
            Open Studio
          </Button>
          <Button
            onClick={() => {
              if (data?.dsn) window.open(`http://${data.public_host || "127.0.0.1"}:${db?.port}`, "_blank");
              else add({ title: "Unable to open port", description: "Add a domain to open the URL.", tone: "error" });
            }}
          >
            <ArrowSquareOut size={18} />
            Open port
          </Button>
          <Button variant="danger" onClick={() => setConfirmDelete(true)}>
            <Trash size={18} />
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
            <t.icon size={16} />
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
                <Button icon={designIcon(RocketLaunch)} onClick={() => run(() => deploy.mutateAsync(dbId), "Deploy started")}>
                  Deploy
                </Button>
                <Button variant="secondary" icon={designIcon(ArrowSquareOut)} onClick={() => run(() => rebuild.mutateAsync(dbId), "Rebuild started")}>
                  Rebuild
                </Button>
                {running ? (
                <Button variant="destructive-ghost" icon={designIcon(Stop)} onClick={() => run(() => stop.mutateAsync(dbId), "Database stopped")}>
                    Stop
                  </Button>
                ) : (
                  <Button variant="success" icon={designIcon(Play)} onClick={() => run(() => start.mutateAsync(dbId), "Database started")}>
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
                  {copied ? <Check size={16} /> : <Copy size={16} />}
                  {copied ? "Copied" : "Copy"}
                </Button>
              </div>
              <div className="bg-surface-container-lowest border border-outline-variant rounded-lg p-md font-code-md text-code-md text-on-surface break-all">
                {data?.dsn ? maskedDsn(data.dsn) : <Skeleton variant="text" aria-label="Loading connection string" />}
              </div>
              <p className="font-code-md text-code-md text-on-surface-variant/60 mt-sm">
                The connection string is only shown here and in the environment variables of this project's services.
              </p>
            </div>

            <div className="bg-surface border border-outline-variant rounded-xl p-6">
              <h3 className="font-label-caps text-label-caps text-on-surface-variant mb-4 uppercase">Status</h3>
              <div className="flex items-center gap-md">
                <RuntimeStatus status={runtimeStatus} live={isRuntimeLive(runtimeStatus)} />
              <Button variant="ghost" onClick={() => void reloadStats()} loading={loadingStats} disabled={loadingStats}>
                  Refresh
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
                      <Badge tone={d.status === "completed" ? "success" : d.status === "failed" ? "danger" : "neutral"}>{d.status}</Badge>
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
            <EmptyState title="No deployments yet" description="Use Deploy to start the database container from the official image." className="border-0" />
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
        <LogViewer
          lines={formattedLogLines}
          followTail={logFollow}
          onFollowTailChange={setLogFollow}
        />
      )}

      {tab === "metrics" && (
        <Card>
          <div className="flex items-center justify-between mb-md">
            <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Metrics</h2>
            <RuntimeStatus status={runtimeStatus} live={isRuntimeLive(runtimeStatus)} />
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
            loadingStats ? <div className="space-y-sm py-lg" aria-label="Loading metrics"><Skeleton variant="card" /><Skeleton variant="card" /></div> : <EmptyState title="No live metrics available" description="Metrics appear when a container is active." className="border-0" />
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
              Delete this database, its container, credentials and data volume. This permanently removes its stored data.
            </p>
            <Button variant="danger" onClick={() => setConfirmDelete(true)}>
              <Trash size={16} />
              Delete database
            </Button>
          </Card>
        </div>
      )}

      <AlertDialog
        trigger={<span />}
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        onConfirm={() =>
          deleteDb.mutate(dbId, {
            onSuccess: () => {
              add({ title: "Database deleted", tone: "success" });
              window.location.href = `/projects/${db?.project_id}`;
            },
            onError: (e) => add({ title: "Delete failed", description: e.message, tone: "error" }),
          })
        }
        title="Delete database"
        description={`Remove ${db?.name}, its container and data volume? This permanently deletes all stored data.`}
        confirmLabel="Delete"
      />
    </div>
  );
}

export const Route = createFileRoute("/_shell/databases/$dbId/")({
  component: DatabaseDetail,
});

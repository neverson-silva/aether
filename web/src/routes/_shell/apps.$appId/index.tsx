import { Metric } from "./-components/Metric";
import { LiveLogs } from "./-components/LiveLogs";
import { createFileRoute } from "@tanstack/react-router";
import { useEffect, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useParams } from "@tanstack/react-router";
import { CronJobs } from "./-components/CronJobs";
import { Autopilot } from "./-components/Autopilot";
import { Workers } from "./-components/Workers";
import { Previews } from "./-components/Previews";
import { Terminal } from "./-components/Terminal";
import { ComposeTab } from "./-components/ComposeTab";
import {
  useAddDomain,
  useAppDetail,
  useAppDetailSecrets,
  useAppRebuild,
  useAppRestart,
  useAppStart,
  useAppStates,
  useAppStop,
  useDeleteApp,
  useDeleteEnv,
  useDeploy,
  useDeployments,
  useUpdateApp,
  useDomains,
  useNetQ,
  useRemoveDomain,
  useGenerateFreeDomain,
  useRollback,
  useSetEnv,
  useSetWebhook,
  useStats,
  useTimeline,
} from "@/hooks";
import {
  Button,
  Card,
  CodeBlock,
  ConfirmDialog,
  Field,
  Input,
  Modal,
  Select,
  Spinner,
  StatusPill,
  Table,
  cn,
  DeploymentStatus,
  fmtBytes,
  fmtDate,
  isDeploymentActive,
  useToast,
} from "../../../components/ui";
import { EnvEditorModal } from "../../../components/EnvEditorModal";
import { DeploymentsTab } from "./-components/DeploymentsTab";


const domainSchema = z.object({
  host: z.string().min(1, "Host is required").regex(/^[a-z0-9.-]+\.[a-z]{2,}$/, "Invalid domain"),
  https: z.boolean(),
});

const webhookSchema = z.object({
  secret: z.string().min(1, "Secret is required"),
});

const TABS = ["overview", "deployments", "compose", "logs", "metrics", "settings", "cron", "workers", "previews", "terminal"] as const;
type Tab = (typeof TABS)[number];

function stateTone(state: string): string {
  return state === "running" ? "active" : state === "no_container" ? "disabled" : "pending";
}

function domainPill(status: string): string {
  switch (status) {
    case "ACTIVE":
      return "active";
    case "ERROR":
      return "error";
    case "PROVISIONING":
      return "pending";
    default:
      return "disabled";
  }
}

function AppDetail() {
  const { appId } = useParams({ strict: false }) as { appId: string };
  const [envEditorOpen, setEnvEditorOpen] = useState(false);
  const { data: detail } = useAppDetail(appId);
  const { data: detailSecrets } = useAppDetailSecrets(appId, envEditorOpen);
  const envEditorVars = useMemo(
    () => (detailSecrets?.env ?? detail?.env ?? []).map((e) => ({ key: e.name, value: e.value, is_secret: e.secret })),
    [detailSecrets, detail]
  );
  const { data: deployments } = useDeployments(appId);
  const { data: domains } = useDomains(appId);
  const { data: stats } = useStats(appId);
  const { data: timeline } = useTimeline(appId);
  const { data: states } = useAppStates();
  const { data: netq } = useNetQ();

  const deploy = useDeploy(appId);
  const rebuild = useAppRebuild();
  const restart = useAppRestart();
  const start = useAppStart();
  const stop = useAppStop();
  const rollback = useRollback(appId);
  const deleteApp = useDeleteApp();
  const setEnv = useSetEnv(appId);
  const deleteEnv = useDeleteEnv(appId);
  const addDomain = useAddDomain(appId);
  const removeDomain = useRemoveDomain(appId);
  const generateFreeDomain = useGenerateFreeDomain(appId);
  const setWebhook = useSetWebhook(appId);
  const updateApp = useUpdateApp(appId);

  const { toast } = useToast();
  const [tab, setTab] = useState<Tab>("overview");
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [confirmRollback, setConfirmRollback] = useState(false);
  const [webhookModal, setWebhookModal] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [editName, setEditName] = useState("");
  const [editPort, setEditPort] = useState(0);
  const [copied, setCopied] = useState(false);
  const [copiedInt, setCopiedInt] = useState(false);
  const [autodeploy, setAutodeploy] = useState(false);
  const [viewLogsDep, setViewLogsDep] = useState<string | null>(null);

  const app = detail?.app;

  useEffect(() => {
    if (editOpen && app) {
      setEditName(app.name);
      setEditPort(app.port);
    }
  }, [editOpen, app]);
  const latest = deployments?.[0];
  const state = states?.[appId] ?? "unknown";
  const running = state === "running";
  const buildPhase = latest ? ["queued", "building", "starting", "health_checking"].includes(latest.status) : false;

  const domainForm = useForm<z.infer<typeof domainSchema>>({
    resolver: zodResolver(domainSchema),
    defaultValues: { host: "", https: false },
  });
  const webhookForm = useForm<z.infer<typeof webhookSchema>>({
    resolver: zodResolver(webhookSchema),
    defaultValues: { secret: "" },
  });

  if (!app) return <Spinner label="Loading application..." />;

  const liveURL = (() => {
    if (domains?.length) return `${domains[0].https ? "https" : "http"}://${domains[0].host}`;
    const probe = (netq ?? []).find((n) => n.app_id === app.id);
    return probe?.addr ? `http://${probe.addr}` : null;
  })();

  const copyURL = () => {
    if (!liveURL) return;
    navigator.clipboard.writeText(liveURL).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    });
  };


  const submitDomain = async (values: z.infer<typeof domainSchema>) => {
    try {
      await addDomain.mutateAsync({ host: values.host.toLowerCase(), https: values.https });
      toast("Domain added");
      domainForm.reset();
    } catch (err) {
      toast(err instanceof Error ? err.message : "error adding domain", "error");
    }
  };

  const submitWebhook = async (values: z.infer<typeof webhookSchema>) => {
    try {
      await setWebhook.mutateAsync(values.secret);
      toast("Webhook secret saved");
      setWebhookModal(false);
      webhookForm.reset();
    } catch (err) {
      toast(err instanceof Error ? err.message : "error saving webhook", "error");
    }
  };

  const run = (fn: () => Promise<unknown>, okMsg: string) => {    fn().then(
      () => toast(okMsg),
      (e) => toast(e instanceof Error ? e.message : "operation failed", "error")
    );
  };

  return (
    <div className="space-y-lg">
      {/* Header */}      <div className="flex flex-col md:flex-row md:items-end justify-between gap-6 mb-8">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-3">
            <span className="material-symbols-outlined text-primary text-[32px] shrink-0">
              {app.source_type === "git" ? "code" : "webhook"}
            </span>
            <h2 className="font-display-lg text-[clamp(1.5rem,4vw,3rem)] leading-[1.1] text-on-surface truncate">{app.name}</h2>
            <div className="px-3 py-1 rounded-full bg-surface-container border border-outline-variant flex items-center gap-2 shrink-0">
              <div className={`w-2 h-2 rounded-full ${latest?.status === "failed" ? "bg-error" : buildPhase ? "bg-[#fbbf24]" : running ? "bg-[#4ade80]" : "bg-on-surface-variant/50"} ${buildPhase || running ? "pulse-dot" : ""}`} />
              <span className="font-code-md text-code-md text-on-surface">{latest?.status === "failed" ? "Failed" : buildPhase ? "Building" : running ? "Running" : state === "no_container" ? "No container" : "Stopped"}</span>
            </div>
            <p className="md:ml-4 font-body-md text-body-md text-on-surface-variant">:{app.port}</p>
          </div>
          <p className="font-body-md text-body-md text-on-surface-variant mt-1 max-w-[42rem]">
            {app.source_type === "image" ? app.image : app.git_url}
          </p>
        </div>
        <div className="flex items-center gap-3 flex-wrap">
          <span className="px-3 py-1 rounded-full bg-surface-container-high border border-outline-variant font-code-md text-[11px] text-on-surface uppercase tracking-wider">
            {app.source_type === "git" ? (app.git_branch || "Git") : "OCI Image"}
          </span>
          <div className="flex gap-2">
            <button className="text-on-surface-variant hover:text-primary transition-colors" onClick={() => setEditOpen(true)} title="Edit">
              <span className="material-symbols-outlined text-[18px]">edit</span>
            </button>
            <button className="text-on-surface-variant hover:text-error transition-colors" onClick={() => setConfirmDelete(true)} title="Delete">
              <span className="material-symbols-outlined text-[18px]">delete</span>
            </button>
          </div>
        </div>
      </div>

      <div className="border-b border-outline-variant mb-8 flex gap-6 overflow-x-auto">
        {TABS.map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`font-label-caps text-label-caps pb-3 px-1 whitespace-nowrap capitalize transition-colors ${
              tab === t ? "text-primary border-b-2 border-primary" : "text-on-surface-variant hover:text-on-surface"
            }`}
          >
            {t}
          </button>
        ))}
      </div>

      {tab === "overview" && (
        <>
          {/* Deploy Settings */}
          <div className="bg-surface-container-lowest border border-outline-variant rounded-xl p-6 mb-8">
            <div className="flex items-center justify-between mb-6">
              <div>
                <h3 className="font-headline-sm text-headline-sm text-on-surface">Deploy Settings</h3>
                <p className="text-body-sm text-on-surface-variant">Deploy, rebuild or control this service</p>
              </div>
              <span className="px-3 py-1.5 bg-surface-container-high border border-outline-variant rounded text-body-sm font-medium text-on-surface-variant">
                {app.source_type === "git" ? "Git" : "Image"}
              </span>
            </div>
            <div className="flex flex-wrap items-center gap-3">
              <Button className="bg-on-surface text-surface hover:bg-on-surface/90" leftIcon="rocket_launch" onClick={() => run(() => deploy.mutateAsync(undefined), "Deploy started")}>
                Deploy
              </Button>
              <Button variant="subtle" leftIcon="refresh" onClick={() => run(() => rebuild.mutateAsync(app.id), "Rebuild started")}>
                Rebuild
              </Button>
              {running ? (
                <Button variant="danger" leftIcon="stop_circle" onClick={() => run(() => stop.mutateAsync(app.id), "Service stopped")}>
                  Stop
                </Button>
              ) : (
                <Button variant="success" leftIcon="play_arrow" onClick={() => run(() => start.mutateAsync(app.id), "Service started")}>
                  Start
                </Button>
              )}
              <Button variant="subtle" leftIcon="terminal" onClick={() => setTab("terminal")}>
                Open Terminal
              </Button>
              <Button variant="subtle" leftIcon="open_in_new" onClick={() => {
                if (liveURL) window.open(liveURL, "_blank");
                else toast("Add a domain to open the URL", "error");
              }}>
                Visit URL
              </Button>
              <div className="ml-auto flex items-center gap-3">
                <span className="text-body-sm text-on-surface-variant">Autodeploy</span>
                <button
                  onClick={() => setAutodeploy((v) => !v)}
                  className={`w-10 h-5 rounded-full relative cursor-pointer transition-colors ${autodeploy ? "bg-primary" : "bg-surface-container-high border border-outline-variant"}`}
                >
                  <div className={`absolute top-1 w-3 h-3 bg-on-primary rounded-full transition-all ${autodeploy ? "right-1 bg-on-primary" : "left-1 bg-on-surface-variant/60"}`} />
                </button>
              </div>
            </div>
          </div>

          {/* Provider */}
          <div className="bg-surface-container-lowest border border-outline-variant rounded-xl p-8 space-y-8 mb-8">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="font-headline-sm text-headline-sm text-on-surface">Provider</h3>
                <p className="text-body-sm text-on-surface-variant">Source of this service's code or image</p>
              </div>
              <span className="material-symbols-outlined text-on-surface-variant">link</span>
            </div>
            <div className="flex gap-6 border-b border-outline-variant pb-4">
              {[["code", "Git"], ["docker", "Image"], ["upload", "Upload"]].map(([icon, label]) => (
                <button
                  key={label}
                  className={`flex items-center gap-2 text-body-sm font-medium pb-4 -mb-[17px] transition-colors ${
                    (label === "Git" && app.source_type === "git") || (label === "Image" && app.source_type === "image")
                      ? "text-primary border-b-2 border-primary font-bold"
                      : "text-on-surface-variant hover:text-on-surface"
                  }`}
                >
                  <span className="material-symbols-outlined text-[18px]">{icon}</span>
                  {label}
                </button>
              ))}
            </div>
            <div className="grid grid-cols-1 gap-6">
              <div className="space-y-2">
                <label className="block font-label-caps text-label-caps text-on-surface-variant">
                  {app.source_type === "git" ? "Repository" : "Image"}
                </label>
                <div className="w-full p-3 bg-surface-container border border-outline-variant rounded flex justify-between items-center">
                  <span className="font-code-md text-code-md text-on-surface truncate">{app.source_type === "git" ? app.git_url || "—" : app.image}</span>
                  <span className="material-symbols-outlined text-on-surface-variant">expand_more</span>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-6">
                <div className="space-y-2">
                  <label className="block font-label-caps text-label-caps text-on-surface-variant">Branch</label>
                  <div className="w-full p-3 bg-surface-container border border-outline-variant rounded flex justify-between items-center">
                    <span className="font-code-md text-code-md text-on-surface">{app.git_branch || "main"}</span>
                    <span className="material-symbols-outlined text-on-surface-variant">unfold_more</span>
                  </div>
                </div>
                <div className="space-y-2">
                  <label className="block font-label-caps text-label-caps text-on-surface-variant">Port</label>
                  <div className="w-full p-3 bg-surface-container border border-outline-variant rounded flex justify-between items-center">
                    <span className="font-code-md text-code-md text-on-surface">:{app.port}</span>
                    <span className="material-symbols-outlined text-on-surface-variant">unfold_more</span>
                  </div>
                </div>
              </div>
            </div>
            <div className="flex justify-end pt-4">
              <Button onClick={() => setTab("settings")}>Configure</Button>
            </div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-1 flex flex-col gap-6">
            <div className="bg-surface border border-outline-variant rounded-xl p-6">
              <h3 className="font-label-caps text-label-caps text-on-surface-variant mb-6 uppercase tracking-wider">Service Details</h3>
              <div className="space-y-4">
                <div>
                  <span className="block font-body-sm text-body-sm text-on-surface-variant mb-1">Live URL</span>
                  <div className="flex items-center justify-between p-3 bg-surface-container-low border border-outline-variant rounded-md group">
                    <span className="font-code-md text-code-md text-primary truncate">{liveURL ?? "—"}</span>
                    <button onClick={copyURL} className="text-on-surface-variant group-hover:text-primary transition-colors" title={copied ? "Copied!" : "Copy"}>
                      <span className="material-symbols-outlined text-[18px]">{copied ? "check" : "content_copy"}</span>
                    </button>
                  </div>
                </div>
                <div>
                  <span className="block font-body-sm text-body-sm text-on-surface-variant mb-1">
                    Internal Host
                    <span className="material-symbols-outlined text-[14px] text-on-surface-variant/50 ml-1 cursor-help" title="Hostname interno na rede container-to-container. Outros serviços do mesmo projeto alcançam este serviço por este nome.">help</span>
                  </span>
                  <div className="flex items-center justify-between p-3 bg-surface-container-low border border-outline-variant rounded-md group">
                    <span className="font-code-md text-code-md text-on-surface truncate">{detail?.internal_host ?? "—"}</span>
                    <button
                      onClick={() => {
                        if (!detail?.internal_host) return;
                        navigator.clipboard.writeText(detail.internal_host).then(() => {
                          setCopiedInt(true);
                          setTimeout(() => setCopiedInt(false), 1500);
                        });
                      }}
                      className="text-on-surface-variant group-hover:text-primary transition-colors"
                      title={copiedInt ? "Copied!" : "Copy"}
                    >
                      <span className="material-symbols-outlined text-[18px]">{copiedInt ? "check" : "content_copy"}</span>
                    </button>
                  </div>
                  {detail?.internal_network && (
                    <p className="font-code-md text-[11px] text-on-surface-variant/60 mt-1">network: {detail.internal_network}</p>
                  )}
                </div>
                <div>
                  <span className="block font-body-sm text-body-sm text-on-surface-variant mb-1">Source</span>
                  <div className="flex items-center justify-between p-3 bg-surface-container-low border border-outline-variant rounded-md group">
                    <span className="font-code-md text-code-md text-on-surface truncate flex items-center gap-2">
                      <span className="material-symbols-outlined text-[16px]">code</span>
                      {app.source_type === "image" ? app.image : app.git_url}
                    </span>
                    <a href={app.source_type === "git" ? app.git_url : undefined} target="_blank" rel="noreferrer" className="text-on-surface-variant group-hover:text-primary transition-colors">
                      <span className="material-symbols-outlined text-[18px]">open_in_new</span>
                    </a>
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-4 pt-4 border-t border-outline-variant mt-4">
                  <div>
                    <span className="block font-label-caps text-label-caps text-on-surface-variant mb-1">Type</span>
                    <span className="font-body-md text-body-md text-on-surface">{app.source_type === "image" ? "OCI Image" : "Git"}</span>
                  </div>
                  <div>
                    <span className="block font-label-caps text-label-caps text-on-surface-variant mb-1">Port</span>
                    <span className="font-body-md text-body-md text-on-surface">:{app.port}</span>
                  </div>
                  <div>
                    <span className="block font-label-caps text-label-caps text-on-surface-variant mb-1">Health</span>
                    <span className={`font-body-md text-body-md ${stats?.state === "running" ? "text-[#4ade80]" : "text-on-surface-variant"}`}>
                      {stats?.state === "running" ? "healthy" : "—"}
                    </span>
                  </div>
                  <div>
                    <span className="block font-label-caps text-label-caps text-on-surface-variant mb-1">Memory</span>
                    <span className="font-body-md text-body-md text-on-surface">{fmtBytes(stats?.stats?.MemBytes ?? 0)}</span>
                  </div>
                </div>
                {(domains ?? []).length > 0 && (
                  <div className="pt-4 border-t border-outline-variant mt-4">
                    <span className="block font-body-sm text-body-sm text-on-surface-variant mb-1">Domains</span>
                    <div className="flex flex-wrap gap-sm">
                      {(domains ?? []).map((d) => (
                        <a
                          key={d.host}
                          href={`${d.https ? "https" : "http"}://${d.host}`}
                          target="_blank"
                          rel="noreferrer"
                          className="px-2 py-1 rounded bg-surface-container-low border border-outline-variant font-code-md text-code-md text-primary hover:border-primary/50 transition-colors"
                        >
                          {d.host}
                        </a>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="bg-surface border border-outline-variant rounded-xl p-4 flex flex-col justify-between">
                <span className="font-label-caps text-label-caps text-on-surface-variant">CPU Usage</span>
                <div className="mt-4">
                  <span className="font-headline-sm text-headline-sm text-on-surface">{stats?.stats?.CPUPercent?.toFixed(0) ?? "0"}%</span>
                  <div className="w-full h-1 bg-surface-container-high mt-2 rounded-full overflow-hidden">
                    <div className="h-full bg-primary rounded-full" style={{ width: `${Math.min(100, stats?.stats?.CPUPercent ?? 0)}%` }} />
                  </div>
                </div>
              </div>
              <div className="bg-surface border border-outline-variant rounded-xl p-4 flex flex-col justify-between">
                <span className="font-label-caps text-label-caps text-on-surface-variant">Memory</span>
                <div className="mt-4">
                    <span className="font-headline-sm text-headline-sm text-on-surface">{fmtBytes(stats?.stats?.MemBytes ?? 0)}</span>
                  <div className="w-full h-1 bg-surface-container-high mt-2 rounded-full overflow-hidden">
                    <div
                      className="h-full bg-secondary rounded-full"
                      style={{ width: `${Math.min(100, stats?.stats?.MemLimit ? (stats.stats.MemBytes / stats.stats.MemLimit) * 100 : 0)}%` }}
                    />
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div className="lg:col-span-2 flex flex-col gap-6">
            <div className={cn("bg-surface border border-outline-variant rounded-xl p-6 relative overflow-hidden", latest && isDeploymentActive(latest.status) && "rt-bg-glow")}>
              <div className="flex items-center justify-between mb-6 relative z-10">
                <h3 className="font-label-caps text-label-caps text-on-surface-variant uppercase tracking-wider">Latest Deployment</h3>
                {latest && (
                  <span className="px-2 py-1 rounded bg-surface-container border border-outline-variant font-code-md text-[11px] text-on-surface-variant">
                    {fmtDate(latest.created_at)}
                  </span>
                )}
              </div>
              {latest ? (
                <div className="flex items-start gap-4 relative z-10">
                  <div className="w-10 h-10 rounded-full bg-surface-container border border-outline-variant flex items-center justify-center shrink-0">
                    <span className="material-symbols-outlined text-primary">commit</span>
                  </div>
                  <div className="flex-1">
                    <h4 className="font-body-md text-body-md text-on-surface font-semibold mb-1">
                      Deployment #{latest.number} · {latest.commit ? latest.commit.slice(0, 8) : app.git_branch || "image"}
                    </h4>
                    <div className="flex items-center gap-3 font-code-md text-code-md text-on-surface-variant">
                      <span>{app.git_branch || "image"}</span>
                      <span className="w-1 h-1 rounded-full bg-outline-variant" />
                      <span>{latest.image_ref}</span>
                    </div>
                    <div className="mt-6 flex gap-4">
                      <DeploymentStatus status={latest.status} />
                    </div>
                  </div>
                  <button
                    onClick={() => { setTab("logs"); setViewLogsDep(latest?.id ?? ""); }}
                    className="px-3 py-1.5 border border-outline-variant rounded hover:bg-surface-container font-body-sm text-body-sm transition-colors"
                  >
                    View Logs
                  </button>
                </div>
              ) : (
                <p className="font-body-sm text-body-sm text-on-surface-variant relative z-10">No deployments yet.</p>
              )}
            </div>

            <div className="bg-[#050505] border border-outline-variant rounded-xl flex flex-col h-[400px] overflow-hidden relative">
              <div className="bg-[#0A0A0A] border-b border-[#1F1F1F] p-3 flex justify-between items-center z-10">
                <div className="flex items-center gap-3">
                  <span className="material-symbols-outlined text-on-surface-variant text-[18px]">terminal</span>
                  <span className="font-label-caps text-label-caps text-on-surface-variant">Live Logs</span>
                </div>
                <div className="flex gap-2">
                  <div className="w-3 h-3 rounded-full bg-[#1F1F1F]" />
                  <div className="w-3 h-3 rounded-full bg-[#1F1F1F]" />
                  <div className="w-3 h-3 rounded-full bg-[#1F1F1F]" />
                </div>
              </div>
              <div className="flex-1 overflow-hidden">
                <LiveLogs appId={app.id} />
              </div>
            </div>
          </div>
        </div>
        </>
      )}

      {tab === "compose" && <ComposeTab appID={app.id} />}

      {tab === "deployments" && (
        <DeploymentsTab
          appId={app.id}
          deployments={(deployments ?? []) as never}
          onRollback={() => run(() => rollback.mutateAsync(undefined), "Rollback started")}
        />
      )}

      {tab === "logs" && (
        <Card>
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">Live Logs</h2>
          <LiveLogs appId={app.id} />
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mt-lg mb-md">Event timeline</h2>
          <div className="space-y-1 max-h-[260px] overflow-y-auto sidebar-scroll">
            {(timeline ?? []).map((e, i) => (
              <div key={e.id} className="flex items-stretch gap-sm font-code-md text-code-md">
                <div className="flex flex-col items-center w-3">
                  <span className={cn("rt-node mt-1.5", i === 0 ? "bg-[#4ade80]" : "bg-outline-variant/40")} />
                </div>
                <div className="flex items-center gap-sm min-w-0">
                  <span className="text-on-surface-variant/50 shrink-0">{fmtDate(e.ts)}</span>
                  <span className="text-primary truncate">{e.type}</span>
                </div>
              </div>
            ))}
            {latest && isDeploymentActive(latest.status) && (
              <div className="flex gap-sm items-center pl-1.5">
                <span className="rt-live-dot" />
                <span className="font-code-md text-code-md text-on-surface-variant">live</span>
              </div>
            )}
          </div>
        </Card>
      )}

      {tab === "metrics" && (
        <Card>
          <div className="flex items-center justify-between mb-md">
            <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Metrics</h2>
            <StatusPill status={stats?.state ?? "unknown"} pulse={stats?.state === "running"} />
          </div>
          {stats?.stats ? (
            <div className="grid grid-cols-2 md:grid-cols-4 gap-md">
              <Metric label="CPU" value={`${stats.stats.CPUPercent?.toFixed(2) ?? 0}%`} icon="speed" />
              <Metric label="Memory" value={fmtBytes(stats.stats.MemBytes ?? 0)} icon="memory" />
              <Metric label="Limit" value={fmtBytes(stats.stats.MemLimit ?? 0)} icon="data_usage" />
              <Metric label="PIDs" value={String(stats.stats.Pids ?? 0)} icon="track_changes" />
            </div>
          ) : (
            <p className="font-body-sm text-body-sm text-on-surface-variant">No active container.</p>
          )}
        </Card>
      )}

      {tab === "settings" && (
        <div className="grid grid-cols-1 xl:grid-cols-3 gap-lg">
          <Card>
            <div className="flex items-center justify-between mb-md">
              <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Resources</h2>
            </div>
            <p className="font-label-caps text-label-caps text-on-surface-variant/60 uppercase mb-sm">CPU</p>
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-sm mb-md">
              {["0.25", "0.5", "1", "2"].map((c) => (
                <button
                  key={c}
                  onClick={() => updateApp.mutate({ resources: { cpus: c } as never })}
                  className={`px-sm py-2 rounded border font-code-md text-code-md transition-colors ${String(detail?.app.resources?.cpus ?? "") === c ? "border-primary bg-primary/10 text-primary" : "border-outline-variant text-on-surface-variant hover:border-primary/40"}`}
                >
                  {c}
                </button>
              ))}
            </div>
            <p className="font-label-caps text-label-caps text-on-surface-variant/60 uppercase mb-sm">Memory</p>
            <div className="grid grid-cols-2 sm:grid-cols-3 gap-sm mb-md">
              {[{ l: "256 MB", v: 256 }, { l: "512 MB", v: 512 }, { l: "1 GB", v: 1024 }, { l: "2 GB", v: 2048 }, { l: "4 GB", v: 4096 }, { l: "∞", v: 0 }].map((m) => (
                <button
                  key={m.l}
                  onClick={() => updateApp.mutate({ resources: { mem_mb: m.v } as never })}
                  className={`px-sm py-2 rounded border font-code-md text-code-md transition-colors ${detail?.app.resources?.mem_mb === m.v ? "border-primary bg-primary/10 text-primary" : "border-outline-variant text-on-surface-variant hover:border-primary/40"}`}
                >
                  {m.l}
                </button>
              ))}
            </div>
            <p className="font-code-md text-code-md text-on-surface-variant/60">Applies on the next deploy. CPU accepts decimals (0.5) or millicores (500m).</p>
          </Card>
          <Card>
            <div className="flex items-center justify-between mb-md">
              <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Image retention</h2>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mb-md">
              Keep the N most recent built images for this app (git builds). Older images are deleted from the internal registry and local storage. 0 = use global policy (default 5).
            </p>
            <div className="flex items-center gap-sm">
              <Input icon="photo_library" type="number" min={0} placeholder="5" defaultValue={detail?.app.image_retention || 0}
                onChange={(e) => {
                  const n = parseInt(e.target.value, 10);
                  updateApp.mutate({ image_retention: isFinite(n) ? n : 0 });
                }}
              />
            </div>
            <p className="font-code-md text-code-md text-on-surface-variant/60 mt-xs">Global policy: AETHER_IMAGE_RETENTION (default 5, 0 = disabled)</p>
          </Card>
          <Card>
            <div className="flex items-center justify-between mb-md">
              <div>
                <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Service Variables</h2>
                <p className="font-body-sm text-body-sm text-on-surface-variant/70 mt-xs">Only available to this service.</p>
              </div>
              <Button variant="ghost" onClick={() => setEnvEditorOpen(true)}>
                <span className="material-symbols-outlined text-[14px]">terminal</span>
                Open editor
              </Button>
            </div>
            <div className="space-y-sm mb-md">
              {(detail?.env ?? []).map((e) => (
                <div key={e.name} className="flex items-center justify-between gap-sm p-sm rounded border border-outline-variant/60">
                  <div className="min-w-0">
                    <p className="font-code-md text-code-md text-on-surface">{e.name}</p>
                    <p className="font-code-md text-code-md text-on-surface-variant/60 truncate">
                      {e.secret ? "••••••••" : e.value}
                    </p>
                  </div>
                  <div className="flex items-center gap-sm shrink-0">
                    {e.secret && <StatusPill status="secret" />}
                    <button
                      onClick={() => deleteEnv.mutate(e.name)}
                      className="material-symbols-outlined text-[16px] text-on-surface-variant hover:text-error transition-colors"
                    >
                      close
                    </button>
                  </div>
                </div>
              ))}
              {(detail?.env ?? []).length === 0 && (
                <p className="font-body-sm text-body-sm text-on-surface-variant">No variables defined.</p>
              )}
            </div>
            <Button variant="ghost" className="w-full" onClick={() => setEnvEditorOpen(true)}>
              <span className="material-symbols-outlined text-[16px]">add</span>
              Add variable
            </Button>
          </Card>

          <EnvEditorModal
            open={envEditorOpen}
            onClose={() => setEnvEditorOpen(false)}
            title={`Service variables · ${app.name}`}
            description="Variables injected only into this service. They override environment variables."
            vars={envEditorVars}
            onSave={async (entries) => {
              for (const [key, entry] of Object.entries(entries)) {
                await setEnv.mutateAsync({ name: key, value: entry.value, secret: entry.secret });
              }
              return { saved: Object.keys(entries).length };
            }}
          />

          <Card>
            <div className="flex items-center justify-between mb-md">
              <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Domains</h2>
              <button onClick={() => generateFreeDomain.mutate(undefined, { onSuccess: () => toast("Free domain generated") })} className="text-primary font-body-sm text-body-sm hover:text-primary-fixed-dim transition-colors flex items-center gap-1">
                <span className="material-symbols-outlined text-[16px]">auto_awesome</span>
                Generate Free Domain
              </button>
            </div>
            <form onSubmit={domainForm.handleSubmit(submitDomain)} className="space-y-sm mb-md" noValidate>
              <div className="space-y-xs">
                {domainForm.formState.errors.host && (
                  <p className="font-body-sm text-body-sm text-error">{domainForm.formState.errors.host.message}</p>
                )}
                <Input icon="language" placeholder="app.example.com" {...domainForm.register("host")} />
              </div>
              <div className="flex items-center gap-md">
                <label className="flex items-center gap-sm cursor-pointer select-none flex-1">
                  <input type="checkbox" className="w-4 h-4 rounded-sm bg-surface border-outline-variant text-primary" {...domainForm.register("https")} />
                  <span className="font-body-sm text-body-sm text-on-surface-variant">HTTPS (Let's Encrypt)</span>
                </label>
                <Button type="submit">Add</Button>
              </div>
            </form>
            <div className="space-y-sm">
              {(domains ?? []).map((d) => (
                <div key={d.id} className="flex items-center justify-between gap-sm p-sm rounded border border-outline-variant/60">
                  <div className="min-w-0">
                    <p className="font-code-md text-code-md text-on-surface truncate">{d.host}</p>
                    <div className="flex items-center gap-sm mt-xs flex-wrap">
                      <StatusPill status={domainPill(d.status)} pulse={d.status === "PROVISIONING"} />
                      {d.https && <span className="font-code-md text-[10px] text-on-surface-variant/60">SSL · {d.cert_status || "requested"}</span>}
                      <span className="font-code-md text-[10px] text-on-surface-variant/60">Port · {d.container_port}</span>
                    </div>
                  </div>
                  <button
                    onClick={() => removeDomain.mutate(d.host)}
                    className="material-symbols-outlined text-[16px] text-on-surface-variant hover:text-error transition-colors shrink-0"
                  >
                    close
                  </button>
                </div>
              ))}
              {(domains ?? []).length === 0 && (
                <p className="font-body-sm text-body-sm text-on-surface-variant">No domains linked.</p>
              )}
            </div>
          </Card>

          <div className="space-y-lg">
            <Autopilot appID={app.id} />
            <Card>
              <div className="flex items-center justify-between mb-md">
                <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Webhook GitHub</h2>
                <button onClick={() => setWebhookModal(true)} className="text-primary font-body-sm text-body-sm hover:text-primary-fixed-dim transition-colors">
                  Configure
                </button>
              </div>
              <p className="font-body-sm text-body-sm text-on-surface-variant mb-sm">
                {app.source_type === "git"
                  ? "POST /api/v1/webhooks/github/{appID} with the X-Hub-Signature-256 header."
                  : "Available for applications with a git source."}
              </p>
              {app.source_type === "git" && (
                <CodeBlock>{`POST /api/v1/webhooks/github/${app.id}\nX-Hub-Signature-256: sha256=<hmac>`}</CodeBlock>
              )}
            </Card>
          </div>
        </div>
      )}

      {tab === "cron" && <CronJobs appID={app.id} />}
      {tab === "workers" && <Workers appID={app.id} />}
      {tab === "previews" && <Previews appID={app.id} />}
      {tab === "terminal" && <Terminal appID={app.id} />}

      <Modal open={webhookModal} onClose={() => setWebhookModal(false)} title="Webhook secret">
        <form onSubmit={webhookForm.handleSubmit(submitWebhook)} className="space-y-lg" noValidate>
          <Field label="Secret (HMAC)" hint={webhookForm.formState.errors.secret?.message}>
            <Input icon="key" placeholder="whsec-..." {...webhookForm.register("secret")} />
          </Field>
          <div className="flex justify-end gap-md">
            <Button type="button" variant="ghost" onClick={() => setWebhookModal(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={webhookForm.formState.isSubmitting}>
              Save
            </Button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog
        open={confirmDelete}
        onClose={() => setConfirmDelete(false)}
        onConfirm={() =>
          deleteApp.mutate(app.id, {
            onSuccess: () => {
              toast("Application deleted");
              window.location.href = "/apps";
            },
            onError: (e) => toast(e.message, "error"),
          })
        }
        title="Delete application"
        description={`Remove ${app.name} and all deployments? Active containers will be stopped. Type the service name to confirm.`}
        confirmLabel="Delete"
        danger
        requireType={app.name}
      />
      <ConfirmDialog
        open={confirmRollback}
        onClose={() => setConfirmRollback(false)}
        onConfirm={() =>
          rollback.mutate(undefined, {
            onSuccess: () => toast("Rollback started"),
            onError: (e) => toast(e.message, "error"),
          })
        }
        title="Rollback"
        description="Restore the previous ready deployment of this application?"
        confirmLabel="Rollback"
      />

      <Modal open={editOpen} onClose={() => setEditOpen(false)} title="Edit service">
        <div className="flex flex-col gap-lg">
          <Field label="Service name">
            <Input value={editName} onChange={(e) => setEditName(e.target.value)} placeholder="my-service" />
          </Field>
          <Field label="Port">
            <Input type="number" value={String(editPort)} onChange={(e) => setEditPort(parseInt(e.target.value) || 0)} placeholder="8080" />
          </Field>
          <div className="flex justify-end gap-md border-t border-outline-variant pt-lg">
            <Button variant="ghost" onClick={() => setEditOpen(false)}>Cancel</Button>
            <Button
              onClick={() => {
                const body: Record<string, unknown> = {};
                if (editName.trim() && editName.trim() !== app.name) body.name = editName.trim();
                if (editPort > 0 && editPort !== app.port) body.port = editPort;
                if (Object.keys(body).length) {
                  updateApp.mutate(body as never, {
                    onSuccess: () => {
                      toast("Service updated");
                      setEditOpen(false);
                    },
                    onError: (e) => toast(e instanceof Error ? e.message : "failed", "error"),
                  });
                } else {
                  setEditOpen(false);
                }
              }}
            >
              Save
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}

export const Route = createFileRoute("/_shell/apps/$appId/")({
  component: AppDetail,
});

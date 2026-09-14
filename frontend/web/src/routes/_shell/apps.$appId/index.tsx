import { createFileRoute, useParams } from "@tanstack/react-router";
import { useEffect, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { NativeSelect } from "@aether/design-system";
import { Button, Dialog, Field, Input, useToast, type VariableRow } from "@aether/design-system";
import type { App } from "@/api/types";
import { useDomains, useSaveServiceSource, useServiceAction, useServiceConnection, useServiceContainers, useServiceDeployments, useServiceDetails, useServiceEnvironment, useServiceSource, useServiceStats, useServiceTimeline, useSetEnv, useSetWebhook, useSourceControlConnections, useSourceControlRepositories } from "@/hooks";
import { useDeleteEnv as useRemoveEnv } from "@/hooks/use-delete-env";
import { useUpdateService as usePatchService } from "@/hooks/use-update-service";
import { isRuntimeLive, mapRuntimeStatus } from "@/lib/runtime-status";
import { BackupTab } from "../databases.$dbId/-components/BackupTab";
import { ComposeTab } from "./-components/ComposeTab";
import { CronJobs } from "./-components/CronJobs";
import { DeploymentLogModal } from "./-components/DeploymentLogModal";
import { DeploymentsTab } from "./-components/DeploymentsTab";
import { DomainsPanel } from "../../../components/DomainsPanel";
import { ServiceHeader } from "./-components/ServiceHeader";
import { ServiceLogsTab, ServiceMetricsTab, ServiceVariablesTab } from "./-components/ServiceRuntimeTabs";
import { ServiceOverviewTab } from "./-components/ServiceOverviewTab";
import { ServiceSettingsTab } from "./-components/ServiceSettingsTab";
import { Terminal } from "./-components/Terminal";
import { SERVICE_TABS, type ServiceTab } from "./-components/service-tabs";
import { extractReturnTo, normalizeDetailsSearch, readDetailsReturnTo } from "./-details-search";
import type { Deployment } from "@/api/types";

const detailSearchSchema = z.preprocess(normalizeDetailsSearch, z.object({
  tab: z.enum(SERVICE_TABS).optional(),
  returnTo: z.string().optional(),
}));

const webhookSchema = z.object({ secret: z.string().min(1, "Secret is required") });
const maskedSecretValue = "••••••••";

function safeExternalURL(value: string | null | undefined): string | null {
  if (!value) return null;
  try {
    const url = new URL(value);
    return url.protocol === "http:" || url.protocol === "https:" ? url.toString() : null;
  } catch {
    return null;
  }
}

function parseProviderPaths(value: string) { return value.split(/[\n,]/).map((item) => item.trim()).filter(Boolean); }

function parseEnv(): VariableRow[] | undefined {
  const pasted = window.prompt("Paste .env content (KEY=value per line):");
  if (!pasted) return;
  return pasted.split("\n").flatMap((raw, index) => {
    const line = raw.trim();
    if (!line || line.startsWith("#")) return [];
    const match = line.match(/^([A-Za-z_][A-Za-z0-9_]*)=(.*)$/);
    if (!match) return [];
    return [{ id: `imported-${index}-${match[1]}`, key: match[1], value: match[2].replace(/^"|"$/g, ""), secret: /password|secret|key|token/i.test(match[1]) }];
  });
}

async function copyText(value: string) {
  if (navigator.clipboard?.writeText) return navigator.clipboard.writeText(value);
  const textarea = document.createElement("textarea");
  textarea.value = value;
  textarea.style.position = "fixed";
  textarea.style.opacity = "0";
  document.body.appendChild(textarea);
  textarea.focus();
  textarea.select();
  const copied = document.execCommand("copy");
  textarea.remove();
  if (!copied) throw new Error("Copy command failed");
}

function isDeploymentActive(status: string) { return ["queued", "building", "starting", "health_checking"].includes(status); }

function formatBytes(bytes: number) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(1)} KiB`;
  return `${(bytes / 1024 ** 2).toFixed(1)} MiB`;
}

function canonicalApp(service: NonNullable<ReturnType<typeof useServiceDetails>["data"]>): App {
  return {
    id: service.id,
    org_id: service.org_id,
    project_id: service.project_id,
    name: service.name,
    source_type: service.kind === "app" && service.spec?.source_type === "git" ? "git" : "image",
    image: service.spec?.image ?? (service.kind === "compose" ? "Docker Compose stack" : service.spec?.engine ?? service.kind),
    git_url: service.spec?.git_url ?? "",
    git_branch: service.spec?.git_branch ?? "",
    dockerfile: service.spec?.dockerfile ?? "",
    build_type: service.spec?.build_type ?? service.kind,
    preview_domain: "",
    server_id: "",
    cluster_id: "",
    environment_id: service.environment_id ?? "",
    port: service.spec?.port ?? 0,
    storage_mb: service.spec?.storage_mb ?? 0,
    resources: { cpus: service.spec?.cpus ?? "0", mem_mb: service.spec?.mem_mb ?? 0 },
    image_retention: service.spec?.image_retention ?? 0,
    health_check: { enabled: false, path: "/", interval_ms: 0, timeout_ms: 0, retries: 0 },
    volumes: (service.volumes ?? []).map((volume) => ({ name: volume.name, mount_path: volume.mount_path })),
    created_at: service.created_at,
    updated_at: service.updated_at,
  };
}

function AppDetail() {
  const { appId } = useParams({ strict: false }) as { appId: string };
  const search = Route.useSearch();
  const navigate = Route.useNavigate();
  const { add } = useToast();
  const serviceQuery = useServiceDetails(appId, Boolean(appId));
  const service = serviceQuery.data;
  const serviceId = service?.id ?? appId;
  const runtimeId = service?.spec_id ?? appId;
  const app = service ? canonicalApp(service) : undefined;
  const { data: serviceEnvironment } = useServiceEnvironment(serviceId, Boolean(service), search.tab === "variables");
  const { data: serviceConnection, isError: serviceConnectionError } = useServiceConnection(serviceId, Boolean(service && service.kind === "database"));
  const canManageSource = service?.capabilities.can_manage_source === true;
  const { data: source, isLoading: sourceLoading, isError: sourceError } = useServiceSource(serviceId, canManageSource, canManageSource);
  const { data: sourceConnections } = useSourceControlConnections(canManageSource);
  const sourceConnection = sourceConnections?.find((item) => item.id === source?.connection_id);
  const { data: providerRepositories } = useSourceControlRepositories(sourceConnection?.installation_id);
  const { data: deployments } = useServiceDeployments(serviceId, Boolean(service));
  const { data: domains } = useDomains("services", serviceId);
  const { data: stats } = useServiceStats(serviceId, Boolean(service));
  const { data: containers } = useServiceContainers(serviceId, Boolean(service));
  const { data: timeline } = useServiceTimeline(serviceId, Boolean(service));
  const setEnv = useSetEnv(serviceId, Boolean(service));
  const deleteEnv = useRemoveEnv(serviceId, Boolean(service));
  const setWebhook = useSetWebhook(serviceId, Boolean(service));
  const saveSource = useSaveServiceSource(serviceId, Boolean(service));
  const updateService = usePatchService();
  const serviceDeploy = useServiceAction("deploy");
  const serviceStart = useServiceAction("start");
  const serviceStop = useServiceAction("stop");
  const serviceRestart = useServiceAction("restart");
  const serviceDelete = useServiceAction("delete");
  const [variables, setVariables] = useState<VariableRow[]>([]);
  const [connectionVisible, setConnectionVisible] = useState(false);
  const [providerEditing, setProviderEditing] = useState(false);
  const [providerRepositoryId, setProviderRepositoryId] = useState("");
  const [providerBranch, setProviderBranch] = useState("");
  const [providerRootDirectory, setProviderRootDirectory] = useState("");
  const [providerWatchPaths, setProviderWatchPaths] = useState("");
  const [providerWatchRootFiles, setProviderWatchRootFiles] = useState(false);
  const [providerAutoDeploy, setProviderAutoDeploy] = useState(false);
  const [autodeploy, setAutodeploy] = useState(false);
  const [logContainer, setLogContainer] = useState("");
  const [webhookModal, setWebhookModal] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [editName, setEditName] = useState("");
  const [editPort, setEditPort] = useState(0);
  const [editBuildType, setEditBuildType] = useState("buildpacks");
  const [copied, setCopied] = useState(false);
  const [connectionCopied, setConnectionCopied] = useState(false);
  const [viewLogsDep, setViewLogsDep] = useState<string | null>(null);
  const requestedTab: ServiceTab = search.tab ?? "overview";
  const visibleTabs = service ? SERVICE_TABS.filter((item) => (item !== "compose" || service.kind === "compose") && (item !== "domains" || service.capabilities.can_manage_domains) && (item !== "logs" || service.capabilities.can_view_logs) && (item !== "metrics" || service.capabilities.can_view_metrics) && (item !== "cron" || service.capabilities.can_manage_schedules) && (item !== "terminal" || service.capabilities.can_open_terminal) && (item !== "settings" || service.capabilities.can_build || service.kind === "compose") && (item !== "backup" || service.capabilities.can_manage_backups)) : SERVICE_TABS;
  const tab = visibleTabs.includes(requestedTab) ? requestedTab : "overview";
  const envEditorVars = useMemo(() => (serviceEnvironment?.env ?? []).map((entry) => ({ key: entry.name, value: entry.secret && entry.value === "" ? maskedSecretValue : entry.value, is_secret: entry.secret })), [serviceEnvironment]);
  const variableKey = useMemo(() => envEditorVars.map((entry) => `${entry.key}:${entry.value.length}:${entry.is_secret}`).join("|"), [envEditorVars]);
  const latest = deployments?.slice().sort((a, b) => b.number - a.number)[0];
  const containerState = stats?.state && stats.state !== "unknown" ? stats.state : containers?.[0]?.status ?? service?.status ?? "unknown";
  const activeDeployment = latest && isDeploymentActive(latest.status) ? latest.status : undefined;
  const runtimeStatus = mapRuntimeStatus(containerState, activeDeployment);
  const running = containerState === "running" || containerState === "healthy";
  const returnTo = readDetailsReturnTo(search.returnTo);
  const serviceLogsEndpoint = service ? `/api/v1/services/${service.id}/logs${logContainer ? `?container=${encodeURIComponent(logContainer)}` : ""}` : undefined;
  const overviewLogsEndpoint = service ? `/api/v1/services/${service.id}/logs?running=1` : undefined;
  const liveURL = domains?.length ? safeExternalURL(`${domains[0].https ? "https" : "http"}://${domains[0].host}`) : null;
  const webhookForm = useForm<z.infer<typeof webhookSchema>>({ resolver: zodResolver(webhookSchema), defaultValues: { secret: "" } });

  useEffect(() => { setVariables(envEditorVars.map((entry, index) => ({ id: `service-${index}-${entry.key}`, key: entry.key, value: entry.value, secret: entry.is_secret }))); }, [envEditorVars]);
  useEffect(() => { if (source) { setAutodeploy(source.auto_deploy); setProviderAutoDeploy(source.auto_deploy); } }, [source]);
  useEffect(() => { if (editOpen && app) { setEditName(app.name); setEditPort(app.port); setEditBuildType(app.build_type || "buildpacks"); } }, [editOpen, app]);
  useEffect(() => { if (typeof window !== "undefined") { const params = new URLSearchParams(window.location.search); if (params.has("kind")) { params.delete("kind"); if (returnTo) params.set("returnTo", returnTo); const query = params.toString(); window.history.replaceState(window.history.state, "", `${window.location.pathname}${query ? `?${query}` : ""}${window.location.hash}`); } } }, [returnTo]);

  if (serviceQuery.isLoading) return <div className="min-h-48 animate-pulse rounded-xl border border-outline-variant bg-surface-container" aria-label="Loading service" />;
  if (serviceQuery.isError || !service || !app) return <div className="rounded-xl border border-outline-variant bg-surface-container p-6 text-body-md text-on-surface-variant" role="alert">The requested service could not be loaded.</div>;

  const setTab = (nextTab: ServiceTab) => { void navigate({ search: (previous: typeof search) => ({ ...previous, tab: nextTab }) }); };
  const run = (operation: () => Promise<unknown>, successMessage: string) => { operation().then(() => add({ title: successMessage, tone: "success" })).catch((error) => add({ title: "Operation failed", description: error instanceof Error ? error.message : "Try again later.", tone: "error" })); };
  const updateAutodeploy = (next: boolean) => {
    if (!source?.connection_id || !source.repository_id) { add({ title: "Source control configuration is incomplete", description: "Reconnect the repository before enabling autodeploy.", tone: "error" }); return; }
    setAutodeploy(next);
    saveSource.mutate({ connection_id: source.connection_id, repository_id: source.repository_id, repository_owner: source.repository_owner, repository_name: source.repository_name, repository_full_name: source.repository_full_name, default_branch: source.default_branch, branch: source.branch, auto_deploy: next, root_directory: source.root_directory, environment_template_path: source.environment_template_path, watch_paths: source.watch_paths ?? [], ignore_paths: source.ignore_paths ?? [], watch_root_files: source.watch_root_files }, { onSuccess: () => add({ title: next ? "Autodeploy enabled" : "Autodeploy disabled", tone: "success" }), onError: (error) => { setAutodeploy(source.auto_deploy); add({ title: "Could not update autodeploy", description: error.message, tone: "error" }); } });
  };
  const openProviderEditor = () => { if (!source) return; setProviderBranch(source.branch || source.default_branch || "main"); setProviderRepositoryId(source.repository_id); setProviderRootDirectory(source.root_directory || "/"); setProviderWatchPaths((source.watch_paths ?? []).join(", ")); setProviderWatchRootFiles(source.watch_root_files); setProviderAutoDeploy(source.auto_deploy); setProviderEditing(true); };
  const saveProvider = () => {
    if (!source) return;
    const repository = providerRepositories?.find((item) => item.id === providerRepositoryId) ?? { id: source.repository_id, owner: source.repository_owner, name: source.repository_name, full_name: source.repository_full_name, default_branch: source.default_branch };
    saveSource.mutate({ connection_id: source.connection_id, repository_id: repository.id, repository_owner: repository.owner, repository_name: repository.name, repository_full_name: repository.full_name, default_branch: repository.default_branch, branch: providerBranch.trim() || repository.default_branch || "main", auto_deploy: providerAutoDeploy, root_directory: providerRootDirectory.trim() || "/", environment_template_path: source.environment_template_path, watch_paths: parseProviderPaths(providerWatchPaths), ignore_paths: parseProviderPaths(source.ignore_paths.join(",")), watch_root_files: providerWatchRootFiles }, { onSuccess: () => { setAutodeploy(providerAutoDeploy); setProviderEditing(false); add({ title: "Provider settings saved", tone: "success" }); }, onError: (error) => add({ title: "Could not save provider settings", description: error.message, tone: "error" }) });
  };
  const exportVariables = () => { const content = variables.filter((variable) => variable.key.trim()).map((variable) => `${variable.key}=${variable.secret && variable.value === maskedSecretValue ? "" : variable.value}`).join("\n"); void copyText(content).then(() => add({ title: "Variables copied", tone: "success" })).catch(() => add({ title: "Variables could not be copied", tone: "error" })); };
  const saveVariables = async () => { try { const entries = new Map(variables.filter((variable) => variable.key.trim()).map((variable) => [variable.key.trim(), { value: variable.value, secret: Boolean(variable.secret) }])); for (const entry of envEditorVars) if (!entries.has(entry.key)) await deleteEnv.mutateAsync(entry.key); for (const [name, value] of entries) { if (value.secret && value.value === maskedSecretValue) continue; await setEnv.mutateAsync({ name, value: value.value, secret: value.secret }); } add({ title: "Variables saved", description: `${entries.size} variable(s) updated.`, tone: "success" }); } catch (error) { add({ title: "Variables could not be saved", description: error instanceof Error ? error.message : "Try again later.", tone: "error" }); } };
  const copyURL = () => { if (!liveURL) return; void copyText(liveURL).then(() => { setCopied(true); window.setTimeout(() => setCopied(false), 1500); }).catch(() => add({ title: "Could not copy URL", tone: "error" })); };
  const copyConnection = () => { if (!serviceConnection?.dsn) return; void copyText(serviceConnection.dsn).then(() => { setConnectionCopied(true); window.setTimeout(() => setConnectionCopied(false), 1500); add({ title: "Connection string copied", tone: "success" }); }).catch(() => add({ title: "Could not copy connection string", tone: "error" })); };
  const submitWebhook = async (values: z.infer<typeof webhookSchema>) => { try { await setWebhook.mutateAsync(values.secret); setWebhookModal(false); webhookForm.reset(); add({ title: "Webhook secret saved", tone: "success" }); } catch (error) { add({ title: "Could not save webhook secret", description: error instanceof Error ? error.message : "Try again later.", tone: "error" }); } };
  const saveService = () => { const update: Record<string, unknown> = {}; if (editName.trim() && editName.trim() !== app.name) update.name = editName.trim(); if (editPort > 0 && editPort !== app.port) update.port = editPort; if (service.kind === "app" && editBuildType !== app.build_type) update.build_type = editBuildType; if (!Object.keys(update).length) { setEditOpen(false); return; } updateService.mutate({ serviceId: service.id, update }, { onSuccess: () => { setEditOpen(false); add({ title: "Service updated", tone: "success" }); }, onError: (error) => add({ title: "Could not update service", description: error instanceof Error ? error.message : "Try again later.", tone: "error" }) }); };

  return (
  <div className="space-y-lg">
    <ServiceHeader app={app} service={service} runtimeId={runtimeId} runtimeStatus={runtimeStatus} isLive={isRuntimeLive(runtimeStatus)} visibleTabs={visibleTabs} tab={tab} onTabChange={setTab} onEdit={() => setEditOpen(true)} deletePending={serviceDelete.isPending} onDelete={() => serviceDelete.mutate(service.id, { onSuccess: () => { add({ title: "Service deleted", tone: "success" }); window.location.href = returnTo; }, onError: (error) => add({ title: "Could not delete service", description: error.message, tone: "error" }) })} />
    {tab === "overview" ? <ServiceOverviewTab app={app} service={service} runtimeId={runtimeId} liveURL={liveURL} source={source} repositories={providerRepositories} sourceLoading={sourceLoading} sourceError={sourceError} canManageSource={canManageSource} providerEditing={providerEditing} providerRepositoryId={providerRepositoryId} providerBranch={providerBranch} providerRootDirectory={providerRootDirectory} providerWatchPaths={providerWatchPaths} providerWatchRootFiles={providerWatchRootFiles} providerAutoDeploy={providerAutoDeploy} savingSource={saveSource.isPending} autodeploy={autodeploy} runtimeStats={stats} domains={domains} latest={latest} logsID={service.id} overviewLogsEndpoint={overviewLogsEndpoint} running={running} restarting={serviceRestart.isPending} stopping={serviceStop.isPending} starting={serviceStart.isPending} copied={copied} connectionVisible={connectionVisible} connectionDSN={serviceConnection?.dsn} connectionError={serviceConnectionError} connectionCopied={connectionCopied} onOpenTerminal={() => setTab("terminal")} onVisit={() => liveURL ? window.open(liveURL, "_blank", "noopener,noreferrer") : add({ title: "Add a domain to open the URL", tone: "error" })} onDeploy={() => serviceDeploy.mutate(service.id)} onRestart={() => run(() => serviceRestart.mutateAsync(service.id), "Service restarted")} onStop={() => run(() => serviceStop.mutateAsync(service.id), "Service stopped")} onStart={() => run(() => serviceStart.mutateAsync(service.id), "Service started")} onAutodeployChange={updateAutodeploy} onProviderEdit={openProviderEditor} onProviderCancel={() => setProviderEditing(false)} onProviderSave={saveProvider} onRepositoryChange={setProviderRepositoryId} onBranchChange={setProviderBranch} onRootDirectoryChange={setProviderRootDirectory} onWatchPathsChange={setProviderWatchPaths} onWatchRootFilesChange={setProviderWatchRootFiles} onProviderAutoDeployChange={setProviderAutoDeploy} onCopyURL={copyURL} onShowConnection={() => setConnectionVisible(true)} onCopyConnection={copyConnection} onViewDeploymentLogs={setViewLogsDep} /> : null}
    {tab === "variables" ? <ServiceVariablesTab variableKey={`${variableKey}:${variables.length}`} variables={variables} onChange={setVariables} onImport={parseEnv} onExport={exportVariables} onSave={() => void saveVariables()} saving={setEnv.isPending || deleteEnv.isPending} /> : null}
    {tab === "compose" ? <ComposeTab appID={app.id} initialCompose={service.spec?.compose} canonicalService exportID={service.kind === "app" ? service.spec_id ?? app.id : undefined} showRuntimeExports={service.kind !== "compose"} /> : null}
    {tab === "deployments" ? <DeploymentsTab appId={app.id} serviceId={service.id} deployments={(deployments ?? []) as Deployment[]} onRollback={() => add({ title: "Rollback is only available for application deployments", tone: "info" })} /> : null}
    {tab === "logs" ? <ServiceLogsTab serviceId={service.id} enabled endpoint={serviceLogsEndpoint} containers={containers} logContainer={logContainer} onContainerChange={setLogContainer} timeline={timeline} latest={latest} deploymentActive={Boolean(activeDeployment)} /> : null}
    {tab === "metrics" ? <ServiceMetricsTab runtimeStatus={runtimeStatus} live={isRuntimeLive(runtimeStatus)} stats={stats} /> : null}
    {tab === "settings" ? <ServiceSettingsTab app={app} service={service} runtimeId={runtimeId} onUpdate={(update) => updateService.mutate({ serviceId: service.id, update })} onOpenWebhook={() => setWebhookModal(true)} /> : null}
    {tab === "cron" ? <CronJobs appID={app.id} canonicalService /> : null}
    {tab === "terminal" ? <Terminal serviceId={service.id} /> : null}
    {tab === "domains" ? <DomainsPanel kind="services" id={service.id} /> : null}
    {tab === "backup" && service.kind === "database" ? <BackupTab dbId={service.id} dbName={service.name} /> : null}
    <Dialog open={webhookModal} onOpenChange={setWebhookModal} title="Webhook secret" trigger={<span />}><form onSubmit={webhookForm.handleSubmit(submitWebhook)} className="space-y-lg" noValidate><Field label="Secret (HMAC)" error={webhookForm.formState.errors.secret?.message}><Input placeholder="whsec-..." {...webhookForm.register("secret")} /></Field><div className="flex justify-end gap-md"><Button type="button" variant="ghost" onClick={() => setWebhookModal(false)}>Cancel</Button><Button type="submit" loading={webhookForm.formState.isSubmitting}>Save</Button></div></form></Dialog>
    <DeploymentLogModal appId={app.id} serviceId={service.id} deploymentId={viewLogsDep} onClose={() => setViewLogsDep(null)} />
    <Dialog open={editOpen} onOpenChange={setEditOpen} title="Edit service" trigger={<span />}><div className="flex flex-col gap-lg"><Field label="Service name"><Input value={editName} onChange={(event) => setEditName(event.target.value)} placeholder="my-service" /></Field>{editBuildType !== "compose" ? <Field label="Port"><Input type="number" value={String(editPort)} onChange={(event) => setEditPort(Number.parseInt(event.target.value, 10) || 0)} placeholder="8080" /></Field> : null}{service.kind === "app" ? <Field label="Build type"><NativeSelect value={editBuildType} onChange={(event) => setEditBuildType(event.target.value)} options={[{ label: "Dockerfile", value: "dockerfile" }, { label: "SmartBuild (CNB)", value: "buildpacks" }, { label: "Custom", value: "custom" }, { label: "Compose", value: "compose" }]} /></Field> : null}<div className="flex justify-end gap-md border-t border-outline-variant pt-lg"><Button variant="ghost" onClick={() => setEditOpen(false)}>Cancel</Button><Button onClick={saveService} loading={updateService.isPending}>Save</Button></div></div></Dialog>
  </div>);
}

export const Route = createFileRoute("/_shell/apps/$appId/")({ validateSearch: detailSearchSchema, component: AppDetail });

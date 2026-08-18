import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "./client";
import {
  apiDelete,
  apiGet,
  apiPatch,
  apiPost,
  apiPut,
  getServer,
  setServer,
} from "./client";
import type {
  ApiKey,
  App,
  AppDetail,
  Backup,
  CronJob,
  Database,
  Deployment,
  Domain,
  ResolvedVariable,
  LoginResponse,
  Me,
  Member,
  AuditLog,
  OrgDetail,
  OrgMember,
  NotificationChannel,
  Preview,
  Project,
  S3Destination,
  Stats,
  Template,
  TimelineEvent,
  Worker,
} from "./types";

export const qk = {
  me: ["me"] as const,
  members: ["members"] as const,
  keys: ["api-keys"] as const,
  composes: ["composes"] as const,
  projects: ["projects"] as const,
  apps: ["apps"] as const,
  app: (id: string) => ["app", id] as const,
  deployments: (id: string) => ["deployments", id] as const,
  domains: (id: string) => ["domains", id] as const,
  timeline: (id: string) => ["timeline", id] as const,
  backups: ["backups"] as const,
  stats: (id: string) => ["stats", id] as const,
};

export function useLogin() {
  return useMutation({
    mutationFn: async ({
      email,
      password,
      server,
    }: {
      email: string;
      password: string;
      server: string;
    }) => {
      setServer(server);
      const data = await apiPost<LoginResponse>("/api/v1/auth/login", {
        email,
        password,
      });
      return data;
    },
  });
}

export function useMe() {
  return useQuery({ queryKey: qk.me, queryFn: () => apiGet<Me>("/api/v1/me") });
}

export function useMembers() {
  return useQuery({
    queryKey: qk.members,
    queryFn: () => apiGet<Member[]>("/api/v1/members"),
  });
}

export function useOrgDetail(orgId: string) {
  return useQuery({
    queryKey: ["org", orgId],
    enabled: !!orgId,
    queryFn: () => apiGet<OrgDetail>(`/api/v1/organizations/${orgId}`),
  });
}

export function useOrgMembers(orgId: string) {
  return useQuery({
    queryKey: ["org-members", orgId],
    enabled: !!orgId,
    queryFn: () => apiGet<OrgMember[]>(`/api/v1/organizations/${orgId}/members`),
  });
}

export function useOrgAudit(orgId: string) {
  return useQuery({
    queryKey: ["org-audit", orgId],
    enabled: !!orgId,
    queryFn: () => apiGet<AuditLog[]>(`/api/v1/organizations/${orgId}/audit`),
  });
}

export function useUpdateMember() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ userID, role }: { userID: string; role: string }) =>
      apiPut(`/api/v1/members/${userID}`, { role }),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.members }),
  });
}

export function useAddMember() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { email: string; password: string; name: string; role: string }) =>
      apiPost("/api/v1/members", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.members }),
  });
}

export function useApiKeys() {
  return useQuery({ queryKey: qk.keys, queryFn: () => apiGet<ApiKey[]>("/api/v1/api-keys") });
}

export function useCreateApiKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; scopes: string[] }) =>
      apiPost<{ key: string }>("/api/v1/api-keys", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.keys }),
  });
}

export function useDeleteApiKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/api-keys/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.keys }),
  });
}

export function useProjects() {
  return useQuery({ queryKey: qk.projects, queryFn: () => apiGet<Project[]>("/api/v1/projects") });
}

export function useCreateProject() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => apiPost<Project>("/api/v1/projects", { name }),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.projects }),
  });
}

export function useApps() {
  return useQuery({ queryKey: qk.apps, queryFn: () => apiGet<App[]>("/api/v1/apps") });
}

export function useProjectApps(projectId: string) {
  return useQuery({
    queryKey: ["apps", "project", projectId],
    queryFn: () => apiGet<App[]>(`/api/v1/apps?project_id=${projectId}`),
    enabled: !!projectId,
  });
}

export function useAppDetail(id: string) {
  return useQuery({
    queryKey: qk.app(id),
    queryFn: () => apiGet<AppDetail>(`/api/v1/apps/${id}`),
    enabled: !!id,
    refetchInterval: 8000,
  });
}

export function useCreateApp() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { projectID: string; payload: Partial<App> }) =>
      apiPost<App>(`/api/v1/projects/${body.projectID}/apps`, body.payload),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.apps }),
  });
}

export function useDeleteApp() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/apps/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.apps }),
  });
}

export function useDeploy(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => apiPost<Deployment>(`/api/v1/apps/${appID}/deploy`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.deployments(appID) });
      qc.invalidateQueries({ queryKey: qk.apps });
    },
  });
}

export function useRollback(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => apiPost<Deployment>(`/api/v1/apps/${appID}/rollback`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.deployments(appID) });
      qc.invalidateQueries({ queryKey: qk.apps });
    },
  });
}

export function useDeployments(appID: string) {
  return useQuery({
    queryKey: qk.deployments(appID),
    queryFn: () => apiGet<Deployment[]>(`/api/v1/apps/${appID}/deployments`),
    enabled: !!appID,
    refetchInterval: 8000,
  });
}

export function useStats(appID: string) {
  return useQuery({
    queryKey: qk.stats(appID),
    queryFn: () => apiGet<Stats>(`/api/v1/apps/${appID}/stats`),

    enabled: !!appID,
  });
}

export function useSetEnv(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; value: string; secret: boolean }) =>
      apiPut(`/api/v1/apps/${appID}/env`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.app(appID) }),
  });
}

export function useDeleteEnv(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => apiDelete(`/api/v1/apps/${appID}/env/${name}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.app(appID) }),
  });
}

export function useDomains(appID: string) {
  return useQuery({
    queryKey: qk.domains(appID),
    queryFn: () => apiGet<Domain[]>(`/api/v1/apps/${appID}/domains`),
    enabled: !!appID,
  });
}

export function useAddDomain(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { host: string; https: boolean; container_port?: number; path?: string; internal_path?: string; strip_path?: boolean }) =>
      apiPost(`/api/v1/apps/${appID}/domains`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.domains(appID) }),
  });
}

export function useGenerateFreeDomain(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (https?: boolean) => apiPost<Domain>(`/api/v1/apps/${appID}/domains/generate`, { https: https ?? true }),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.domains(appID) }),
  });
}

export function useRemoveDomain(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (host: string) => apiDelete(`/api/v1/apps/${appID}/domains/${host}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.domains(appID) }),
  });
}

export function useTimeline(appID: string) {
  return useQuery({
    queryKey: qk.timeline(appID),
    queryFn: () => apiGet<TimelineEvent[]>(`/api/v1/apps/${appID}/timeline`),

    enabled: !!appID,
  });
}

export function useBackups() {
  return useQuery({ queryKey: qk.backups, queryFn: () => apiGet<Backup[]>("/api/v1/backups") });
}

export function useCreateBackup() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => apiPost<Backup>("/api/v1/backups"),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.backups }),
  });
}

export function useRestoreBackup() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiPost(`/api/v1/backups/${id}/restore`),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.backups }),
  });
}

export function useSetWebhook(appID: string) {
  return useMutation({
    mutationFn: (secret: string) => apiPut(`/api/v1/apps/${appID}/webhook`, { secret }),
  });
}

export { getServer, setServer };

export function useDatabases() {
  return useQuery({ queryKey: ["databases"], queryFn: () => apiGet<Database[]>("/api/v1/databases") });
}

export function useCreateDatabase() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { project_id: string; name: string; engine: string; version?: string; mem_mb?: number; storage_mb?: number }) =>
      apiPost<Database>("/api/v1/databases", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["databases"] }),
  });
}

export function useDeleteDatabase() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/databases/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["databases"] }),
  });
}

export function useBackupDatabase() {
  return useMutation({ mutationFn: (id: string) => apiPost(`/api/v1/databases/${id}/backup`) });
}

export function useCronJobs(appID: string) {
  return useQuery({
    queryKey: ["cron-jobs", appID],
    queryFn: () => apiGet<CronJob[]>(`/api/v1/apps/${appID}/cron-jobs`),
    enabled: !!appID,
  });
}

export function useCreateCronJob(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; schedule: string; command: string }) =>
      apiPost<CronJob>(`/api/v1/apps/${appID}/cron-jobs`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["cron-jobs", appID] }),
  });
}

export function useDeleteCronJob(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/cron-jobs/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["cron-jobs", appID] }),
  });
}

export function useWorkers(appID: string) {
  return useQuery({
    queryKey: ["workers", appID],
    queryFn: () => apiGet<Worker[]>(`/api/v1/apps/${appID}/workers`),
    enabled: !!appID,
  });
}

export function useCreateWorker(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; command: string }) =>
      apiPost<Worker>(`/api/v1/apps/${appID}/workers`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["workers", appID] }),
  });
}

export function useWorkerAction() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, action }: { id: string; action: "start" | "stop" }) =>
      apiPost(`/api/v1/workers/${id}/${action}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["workers"] });
    },
  });
}

export function useDeleteWorker(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/workers/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["workers", appID] }),
  });
}

export function usePreviews(appID: string) {
  return useQuery({
    queryKey: ["previews", appID],
    queryFn: () => apiGet<Preview[]>(`/api/v1/apps/${appID}/previews`),
    enabled: !!appID,
  });
}

export function useCreatePreview(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (branch: string) => apiPost<Preview>(`/api/v1/apps/${appID}/previews`, { branch }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["previews", appID] }),
  });
}

export function useDeletePreview(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/previews/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["previews", appID] }),
  });
}

export function useTemplates() {
  return useQuery({ queryKey: ["templates"], queryFn: () => apiGet<TemplateItem[]>(`/api/v1/templates`) });
}

export function useInstallTemplate() {
  return useMutation({
    mutationFn: (body: { id: string; project_id: string; name?: string }) =>
      apiPost(`/api/v1/templates/${body.id}/install`, body),
  });
}

export function useS3Destinations() {
  return useQuery({ queryKey: ["s3"], queryFn: () => apiGet<S3Destination[]>("/api/v1/s3-destinations") });
}

export function useCreateS3() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: unknown) => apiPost<S3Destination>("/api/v1/s3-destinations", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["s3"] }),
  });
}

export function useDeleteS3() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/s3-destinations/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["s3"] }),
  });
}

export function useNotificationChannels() {
  return useQuery({ queryKey: ["channels"], queryFn: () => apiGet<NotificationChannel[]>("/api/v1/notification-channels") });
}

export function useCreateChannel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: unknown) => apiPost<NotificationChannel>("/api/v1/notification-channels", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["channels"] }),
  });
}

export function useDeleteChannel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/notification-channels/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["channels"] }),
  });
}

export function useVolumeBackup() {
  return useMutation({
    mutationFn: ({ appID, name, destination_id }: { appID: string; name: string; destination_id: string }) =>
      apiPost(`/api/v1/apps/${appID}/volumes/${name}/backup`, { destination_id }),
  });
}

export function useTOTP() {
  return {
    enroll: () => apiPost<{ secret: string; uri: string }>("/api/v1/auth/totp/enroll"),
    verify: (code: string) => apiPost("/api/v1/auth/totp/verify", { code }),
    disable: () => apiDelete("/api/v1/auth/totp"),
  };
}

export function useExportOrg() {
  return useMutation({
    mutationFn: async () => {
      const server = getServer();
      const res = await fetch(server + "/api/v1/org/export", { credentials: "include" });
      if (!res.ok) throw new ApiError(res.status, "export falhou");
      return res.text();
    },
  });
}

export function useImportOrg() {
  return useMutation({
    mutationFn: async (yaml: string) => {
      const server = getServer();
      const res = await fetch(server + "/api/v1/org/import", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/yaml" },
        body: yaml,
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new ApiError(res.status, body.error || "import falhou");
      }
    },
  });
}

export interface RegistrySettings {
  id: string;
  enabled: boolean;
  host: string;
  port: number;
  container_id: string;
  status: string;
}

export interface RegistryImage {
  repo: string;
  tags: string[];
  size: number;
}

export function useRegistrySettings() {
  return useQuery({ queryKey: ["registry"], queryFn: () => apiGet<RegistrySettings>("/api/v1/registry") });
}

export function useRegistryImages() {
  return useQuery({
    queryKey: ["registry", "images"],
    queryFn: () => apiGet<RegistryImage[]>("/api/v1/registry/images"),
    enabled: false,
  });
}

export function useToggleRegistry() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (enabled: boolean) => apiPost<RegistrySettings>("/api/v1/registry", { enabled }),
    onSuccess: (data) => {
      qc.setQueryData(["registry"], data);
      qc.invalidateQueries({ queryKey: ["registry", "images"] });
    },
  });
}

export interface OutWebhook {
  id: string;
  org_id: string;
  name: string;
  url: string;
  events: string[];
  enabled: boolean;
  created_at: string;
}

export const WEBHOOK_EVENTS = [
  "deployment.started",
  "deployment.ready",
  "deployment.failed",
  "backup.started",
  "backup.finished",
  "backup.failed",
];

export function useOutWebhooks() {
  return useQuery({ queryKey: ["out-webhooks"], queryFn: () => apiGet<OutWebhook[]>("/api/v1/webhooks") });
}

export function useCreateOutWebhook() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; url: string; secret: string; events: string[] }) =>
      apiPost<OutWebhook>("/api/v1/webhooks", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["out-webhooks"] }),
  });
}

export function useDeleteOutWebhook() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/webhooks/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["out-webhooks"] }),
  });
}

export interface AppPolicy {
  app_id: string;
  enabled: boolean;
  cpu_min: number;
  cpu_max: number;
  mem_min_mb: number;
  mem_max_mb: number;
  scale_up_pct: number;
  scale_down_pct: number;
  cooldown_min: number;
}

export interface AutopilotEvent {
  id: string;
  app_id: string;
  action: string;
  detail: string;
  created_at: string;
}

export interface GitOpsConfig {
  id: string;
  org_id: string;
  name: string;
  repo_url: string;
  branch: string;
  path: string;
  target_org_id: string;
  apply_mode: string;
  last_sha: string;
  last_status: string;
  drift_added: number;
  drift_changed: number;
  drift_removed: number;
  last_sync: string;
  created_at: string;
}

export interface RegistryMirror {
  id: string;
  name: string;
  source: string;
  dest: string;
  dest_tls_verify: boolean;
  tags_filter: string;
  schedule: string;
  last_run: string;
  status: string;
  created_at: string;
}

export interface NetQStat {
  app_id: string;
  name: string;
  addr: string;
  samples: { at: string; ms: number; ok: boolean; h3: boolean }[];
  p50_ms: number;
  p95_ms: number;
  uptime_pct: number;
  http3: boolean;
}

export interface Snapshot {
  id: string;
  org_id: string;
  app_id: string;
  volume: string;
  name: string;
  size: number;
  chunks: number;
  dedup_saved: number;
  created_at: string;
}

export function usePolicy(appId: string) {
  return useQuery({ queryKey: ["policy", appId], queryFn: () => apiGet<AppPolicy>(`/api/v1/apps/${appId}/policy`) });
}

export function useSavePolicy(appId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (p: AppPolicy) => apiPut<AppPolicy>(`/api/v1/apps/${appId}/policy`, p),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["policy", appId] }),
  });
}

export function usePolicyEvents(appId: string) {
  return useQuery({ queryKey: ["policy-events", appId], queryFn: () => apiGet<AutopilotEvent[]>(`/api/v1/apps/${appId}/policy/events`) });
}

export function useGitOps() {
  return useQuery({ queryKey: ["gitops"], queryFn: () => apiGet<GitOpsConfig[]>("/api/v1/gitops") });
}

export function useCreateGitOps() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; repo_url: string; branch?: string; path?: string; apply_mode?: string }) =>
      apiPost<GitOpsConfig>("/api/v1/gitops", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["gitops"] }),
  });
}

export function useSyncGitOps() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiPost<GitOpsConfig>(`/api/v1/gitops/${id}/sync`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["gitops"] }),
  });
}

export function useDeleteGitOps() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/gitops/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["gitops"] }),
  });
}

export function useMirrors() {
  return useQuery({ queryKey: ["mirrors"], queryFn: () => apiGet<RegistryMirror[]>("/api/v1/mirrors") });
}

export function useCreateMirror() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; source: string; dest: string; dest_tls_verify?: boolean; tags_filter?: string; schedule?: string }) =>
      apiPost<RegistryMirror>("/api/v1/mirrors", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["mirrors"] }),
  });
}

export function useRunMirror() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiPost(`/api/v1/mirrors/${id}/run`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["mirrors"] }),
  });
}

export function useDeleteMirror() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/mirrors/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["mirrors"] }),
  });
}

export function useNetQ() {
  return useQuery({ queryKey: ["netq"], queryFn: () => apiGet<NetQStat[]>("/api/v1/network/quality"), refetchInterval: 15000 });
}

export interface HostStats {
  cpu_percent: number;
  cpu_cores: number;
  mem_total: number;
  mem_used: number;
  mem_percent: number;
  net: { rx_bytes: number; tx_bytes: number };
  disk: { read_bytes: number; write_bytes: number; total: number; used: number; percent: number };
  uptime: number;
  load: number[];
  hostname: string;
  os: string;
}

export function useHostStats() {
  const [stats, setStats] = useState<HostStats | null>(null);
  const [history, setHistory] = useState<{ cpu: number; mem: number }[]>([]);
  useEffect(() => {
    let active = true;
    const server = localStorage.getItem("aether_server") || "";
    const es = new EventSource(server + "/api/v1/host/stats/stream", { withCredentials: true });
    es.addEventListener("stats", (ev) => {
      if (!active) return;
      try {
        const s = JSON.parse((ev as MessageEvent).data) as HostStats;
        setStats(s);
        setHistory((prev) => [...prev.slice(-59), { cpu: s.cpu_percent, mem: s.mem_percent }]);
      } catch {}
    });
    return () => { active = false; es.close(); };
  }, []);
  return { stats, history };
}

export function useHostEvents() {
  return useQuery({ queryKey: ["host-events"], queryFn: () => apiGet<{ ts: string; type: string; title: string; detail: string }[]>("/api/v1/host/events") });
}

export function useHostLogs(follow: boolean) {
  const [lines, setLines] = useState<{ line: string }[]>([]);
  useEffect(() => {
    if (!follow) return;
    let active = true;
    const server = localStorage.getItem("aether_server") || "";
    const es = new EventSource(server + "/api/v1/host/logs?follow=1", { withCredentials: true });
    es.addEventListener("log", (ev) => {
      if (!active) return;
      try {
        const l = JSON.parse((ev as MessageEvent).data) as { line: string };
        setLines((prev) => [...prev, l].slice(-400));
      } catch {}
    });
    return () => { active = false; es.close(); };
  }, [follow]);
  return lines;
}

export function useSnapshots() {
  return useQuery({ queryKey: ["snapshots"], queryFn: () => apiGet<Snapshot[]>("/api/v1/snapshots") });
}

export function useCreateSnapshot() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { app_id: string; volume: string; name: string }) => apiPost<Snapshot>("/api/v1/snapshots", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["snapshots"] }),
  });
}

export function useRestoreSnapshot() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { id: string; volume?: string }) => apiPost(`/api/v1/snapshots/${body.id}/restore`, { volume: body.volume }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["snapshots"] }),
  });
}

export function useDeleteSnapshot() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/snapshots/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["snapshots"] }),
  });
}

export interface Branding {
  org_id: string;
  name: string;
  logo_url: string;
  primary_color: string;
  accent_color: string;
  dark_mode: boolean;
  updated_at: string;
}

export interface PipelineStage {
  name: string;
  image: string;
  commands: string[];
}

export interface Pipeline {
  id: string;
  org_id: string;
  app_id: string;
  name: string;
  trigger: string;
  stages: PipelineStage[];
  enabled: boolean;
  created_at: string;
}

export interface PipelineRun {
  id: string;
  pipeline_id: string;
  status: string;
  trigger: string;
  log: string;
  started_at: string;
  finished_at: string;
}

export interface ClusterServer {
  id: string;
  name: string;
  host: string;
  status: string;
  version: string;
  labels: string[];
  load: number;
  cluster_id: string;
  last_heartbeat: string;
}

export interface Cluster {
  id: string;
  org_id: string;
  name: string;
  labels: string[];
  created_at: string;
  servers: ClusterServer[];
}

export interface OIDCProvider {
  id: string;
  org_id: string;
  name: string;
  issuer: string;
  client_id: string;
  scopes: string;
  enabled: boolean;
  created_at: string;
}

export function useBranding() {
  return useQuery({ queryKey: ["branding"], queryFn: () => apiGet<Branding>("/api/v1/branding") });
}

export function useSaveBranding() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (b: Partial<Branding>) => apiPut<Branding>("/api/v1/branding", b),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["branding"] }),
  });
}

export function usePipelines() {
  return useQuery({ queryKey: ["pipelines"], queryFn: () => apiGet<Pipeline[]>("/api/v1/pipelines") });
}

export function useCreatePipeline() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { app_id: string; name: string; trigger: string; stages: PipelineStage[] }) =>
      apiPost<Pipeline>("/api/v1/pipelines", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["pipelines"] }),
  });
}

export function useDeletePipeline() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/pipelines/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["pipelines"] }),
  });
}

export function useRunPipeline() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiPost<PipelineRun>(`/api/v1/pipelines/${id}/run`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["pipeline-runs"] }),
  });
}

export function usePipelineRuns(pipelineId: string) {
  return useQuery({
    queryKey: ["pipeline-runs", pipelineId],
    queryFn: () => apiGet<PipelineRun[]>(`/api/v1/pipelines/${pipelineId}/runs`),
  });
}

export function useClusters() {
  return useQuery({ queryKey: ["clusters"], queryFn: () => apiGet<Cluster[]>("/api/v1/clusters") });
}

export function useCreateCluster() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; labels?: string[] }) => apiPost<Cluster>("/api/v1/clusters", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["clusters"] }),
  });
}

export function useDeleteCluster() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/clusters/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["clusters"] }),
  });
}

export function useClusterAddServer() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { cluster_id: string; server_id: string }) =>
      apiPost(`/api/v1/clusters/${body.cluster_id}/servers`, { server_id: body.server_id }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["clusters"] }),
  });
}

export function useClusterRemoveServer() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { cluster_id: string; server_id: string }) =>
      apiDelete(`/api/v1/clusters/${body.cluster_id}/servers/${body.server_id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["clusters"] }),
  });
}

export function useSSO() {
  return useQuery({ queryKey: ["sso"], queryFn: () => apiGet<OIDCProvider[]>("/api/v1/sso") });
}

export function useCreateSSO() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; issuer: string; client_id: string; client_secret?: string; scopes?: string }) =>
      apiPost<OIDCProvider>("/api/v1/sso", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["sso"] }),
  });
}

export function useSSOAuthURL() {
  return useMutation({
    mutationFn: (id: string) => apiPost<{ url: string }>(`/api/v1/sso/${id}/auth-url`),
  });
}

export function useDeleteSSO() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/sso/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["sso"] }),
  });
}

export function useServers() {
  return useQuery({ queryKey: ["servers"], queryFn: () => apiGet<ClusterServer[]>("/api/v1/servers") });
}

export interface SystemSummary {
  health_pct: number;
  deployments: number;
  traffic_bytes: number;
  io_bytes: number;
  cpu_pct: number;
  mem_pct: number;
  io_pct: number;
  apps: { id: string; name: string; cpu_pct: number; mem_pct: number; net_rx_bytes: number; net_tx_bytes: number }[];
  projects: { id: string; name: string; apps: number; env: string; status: string; last_deploy: string }[];
}

export interface AllCronJob {
  id: string;
  app_id: string;
  app_name: string;
  name: string;
  schedule: string;
  command: string;
  enabled: boolean;
  last_run: string;
  next_run: string;
  status: string;
}

export interface CertInfo {
  id: string;
  app_id: string;
  app_name: string;
  host: string;
  https: boolean;
  cert_status: string;
  created_at: string;
}

export function useSystemSummary() {
  return useQuery({ queryKey: ["system-summary"], queryFn: () => apiGet<SystemSummary>("/api/v1/system/summary") });
}

export function useAllCronJobs() {
  return useQuery({ queryKey: ["cron-jobs-all"], queryFn: () => apiGet<AllCronJob[]>("/api/v1/cron-jobs") });
}

export function useCertificates() {
  return useQuery({ queryKey: ["certificates"], queryFn: () => apiGet<CertInfo[]>("/api/v1/certificates") });
}

export function useAppStates() {
  const [states, setStates] = useState<Record<string, string>>({});
  useEffect(() => {
    let active = true;
    apiGet<Record<string, string>>("/api/v1/apps/states")
      .then((s) => { if (active) setStates(s); })
      .catch(() => {});
    const server = localStorage.getItem("aether_server") || "";
    const es = new EventSource(server + "/api/v1/apps/states/stream", { withCredentials: true });
    es.addEventListener("state", (ev) => {
      try {
        const s = JSON.parse((ev as MessageEvent).data) as { app_id: string; state: string };
        setStates((prev) => ({ ...prev, [s.app_id]: s.state }));
      } catch {}
    });
    return () => { active = false; es.close(); };
  }, []);
  return { data: states };
}

export function usePresence(scope: string) {
  const [count, setCount] = useState(0);
  useEffect(() => {
    if (!scope) return;
    let active = true;
    apiPost("/api/v1/presence/join", { scope }).catch(() => {});
    const beat = setInterval(() => {
      apiPost("/api/v1/presence/heartbeat", { scope }).catch(() => {});
    }, 30000);
    const tick = setInterval(() => {
      apiGet<{ count: number }>("/api/v1/presence/count?scope=" + encodeURIComponent(scope))
        .then((r) => { if (active) setCount(r.count); })
        .catch(() => {});
    }, 10000);
    return () => {
      active = false;
      clearInterval(beat);
      clearInterval(tick);
      apiPost("/api/v1/presence/leave", { scope }).catch(() => {});
    };
  }, [scope]);
  return count;
}

export function useAppStart() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: (id: string) => apiPost(`/api/v1/apps/${id}/start`), onSuccess: () => qc.invalidateQueries({ queryKey: ["app-states"] }) });
}

export function useAppStop() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: (id: string) => apiPost(`/api/v1/apps/${id}/stop`), onSuccess: () => qc.invalidateQueries({ queryKey: ["app-states"] }) });
}

export function useAppRestart() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: (id: string) => apiPost(`/api/v1/apps/${id}/restart`), onSuccess: () => qc.invalidateQueries({ queryKey: ["app-states"] }) });
}

export function useAppRebuild() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: (id: string) => apiPost(`/api/v1/apps/${id}/rebuild`), onSuccess: () => qc.invalidateQueries({ queryKey: ["app-states"] }) });
}

export interface Environment {
  id: string;
  project_id: string;
  name: string;
  slug: string;
  description: string;
  color: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface EnvSummary extends Environment {
  apps: number;
  status: string;
  last_deploy: string;
}

export function useEnvironments(projectId: string) {
  return useQuery({
    queryKey: ["environments", projectId],
    queryFn: () => apiGet<EnvSummary[]>(`/api/v1/projects/${projectId}/environments`),
  });
}

export function useCreateEnvironment(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; description?: string; color?: string }) =>
      apiPost<Environment>(`/api/v1/projects/${projectId}/environments`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["environments", projectId] }),
  });
}

export function useUpdateEnvironment(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { environmentID: string; name: string; description?: string; color?: string }) =>
      apiPatch<Environment>(`/api/v1/projects/${projectId}/environments/${body.environmentID}`, {
        name: body.name, description: body.description, color: body.color,
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["environments", projectId] }),
  });
}

export function useDeleteEnvironment(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (environmentID: string) =>
      apiDelete(`/api/v1/projects/${projectId}/environments/${environmentID}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["environments", projectId] }),
  });
}

export function useSetDefaultEnvironment(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (environmentID: string) =>
      apiPost(`/api/v1/projects/${projectId}/environments/${environmentID}/default`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["environments", projectId] }),
  });
}

export interface EnvironmentVariable {
  id: string;
  project_id: string;
  environment_id: string;
  key: string;
  value: string;
  is_secret: boolean;
  created_at: string;
  updated_at: string;
}

export interface VariableAudit {
  id: string;
  project_id: string;
  environment_id: string;
  action: string;
  key: string;
  previous_value: string;
  created_at: string;
}

export function useEnvVars(projectId: string, environmentId: string | null) {
  return useQuery({
    queryKey: ["env-vars", projectId, environmentId],
    enabled: !!projectId && !!environmentId,
    queryFn: async () => {
      const data = await apiGet<{ variables: EnvironmentVariable[] }>(`/api/v1/projects/${projectId}/environments/${environmentId}/variables?secrets=1`);
      return data.variables ?? [];
    },
  });
}

export function useReplaceEnvVars(projectId: string, environmentId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (entries: Record<string, { value: string; secret: boolean }>) =>
      apiPut(`/api/v1/projects/${projectId}/environments/${environmentId}/variables`, entries),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["env-vars", projectId, environmentId] });
      qc.invalidateQueries({ queryKey: ["apps"] });
    },
  });
}

export function useEnvVarAudit(projectId: string, environmentId: string | null) {
  return useQuery({
    queryKey: ["env-vars-audit", projectId, environmentId],
    queryFn: () => apiGet<VariableAudit[]>(`/api/v1/projects/${projectId}/environments/${environmentId}/variables/audit`),
    enabled: !!environmentId,
  });
}

export function useAppDetailSecrets(id: string, enabled: boolean) {
  return useQuery({
    queryKey: [...qk.app(id), "secrets"],
    queryFn: () => apiGet<AppDetail>(`/api/v1/apps/${id}?secrets=1`),
    enabled,
  });
}

export function useEffectiveVariables(appID: string) {
  return useQuery({
    queryKey: ["apps", appID, "effective-variables"],
    queryFn: () => apiGet<{ variables: ResolvedVariable[] }>(`/api/v1/apps/${appID}/variables/effective`),
    enabled: !!appID,
  });
}

export interface ProjectVariable {
  id: string;
  project_id: string;
  key: string;
  value: string;
  is_secret: boolean;
  created_at: string;
  updated_at: string;
}

export function useProjectVars(projectId: string) {
  return useQuery({
    queryKey: ["project-vars", projectId],
    queryFn: async () => {
      const data = await apiGet<{ variables: ProjectVariable[] }>(`/api/v1/projects/${projectId}/variables?secrets=1`);
      return data.variables ?? [];
    },
  });
}

export function useReplaceProjectVars(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (entries: Record<string, { value: string; secret: boolean }>) =>
      apiPut(`/api/v1/projects/${projectId}/variables`, entries),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["project-vars", projectId] });
      qc.invalidateQueries({ queryKey: ["apps"] });
    },
  });
}

export interface NotificationItem {
  id: string;
  org_id: string;
  type: string;
  message: string;
  payload: string;
  read: boolean;
  created_at: string;
}

export function useNotifications() {
  return useQuery({ queryKey: ["notifications"], queryFn: () => apiGet<NotificationItem[]>("/api/v1/notifications") });
}

export function useUnreadCount() {
  return useQuery({ queryKey: ["notifications-unread"], queryFn: () => apiGet<{ count: number }>("/api/v1/notifications/unread-count") });
}

export function useMarkRead() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiPost(`/api/v1/notifications/${id}/read`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["notifications"] });
      qc.invalidateQueries({ queryKey: ["notifications-unread"] });
    },
  });
}

export function useMarkAllRead() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => apiPost("/api/v1/notifications/read-all"),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["notifications"] });
      qc.invalidateQueries({ queryKey: ["notifications-unread"] });
    },
  });
}

export interface TemplateItem {
  id: string;
  name: string;
  description: string;
  category: string;
  tags: string[];
  icon: string;
  version: string;
  definition: string;
  compose_yaml?: string;
  readme: string;
  homepage: string;
  github: string;
  license: string;
  installs: number;
  featured: boolean;
  verified: boolean;
  updated_at: string;
}

export function useTemplatesFiltered(params: { category?: string; q?: string; featured?: boolean }) {
  const qs = new URLSearchParams();
  if (params.category) qs.set("category", params.category);
  if (params.q) qs.set("q", params.q);
  if (params.featured) qs.set("featured", "true");
  return useQuery({
    queryKey: ["templates", params],
    queryFn: () => apiGet<TemplateItem[]>(`/api/v1/templates${qs.toString() ? "?" + qs.toString() : ""}`),
  });
}

export function useTrendingTemplates() {
  return useQuery({ queryKey: ["templates-trending"], queryFn: () => apiGet<TemplateItem[]>("/api/v1/templates?trending=true") });
}

export function useTemplateCategories() {
  return useQuery({ queryKey: ["templates-categories"], queryFn: () => apiGet<string[]>("/api/v1/templates?categories=true") });
}

export interface DeployCompare {
  image: { from: string; to: string };
  commit: { from: string; to: string };
  status_a: string;
  status_b: string;
  env_added: string[];
  env_removed: string[];
  env_changed: string[];
}

export function useDeployCompare(appID: string, a: string | null, b: string | null) {
  return useQuery({
    queryKey: ["deploy-compare", appID, a, b],
    enabled: !!(a && b),
    queryFn: () => apiGet<DeployCompare>(`/api/v1/apps/${appID}/deployments/compare?a=${a}&b=${b}`),
  });
}

export interface DeploymentLog {
  number: number;
  status: string;
  error: string;
  content: string;
}

export function useDeploymentLog(appID: string, depID: string | null) {
  return useQuery({
    queryKey: ["deploy-log", appID, depID],
    enabled: !!depID,
    refetchInterval: depID ? 2500 : false,
    queryFn: () => apiGet<DeploymentLog>(`/api/v1/apps/${appID}/deployments/${depID}/log`),
  });
}

export interface ComposeValidation {
  valid: boolean;
  services: { name: string; image: string; build: string; ports: string[]; volumes: string[]; restart: string }[];
  volumes: string[];
  networks: string[];
  errors: string[];
  warnings: string[];
  depends_on: Record<string, string[]>;
  total_ports: number;
}

export function useValidateCompose(content: string) {
  return useQuery({
    queryKey: ["compose-validate", content],
    enabled: content.trim().length > 5,
    queryFn: () => apiPost<ComposeValidation>("/api/v1/compose/validate", { content }),
    staleTime: 500,
  });
}

export function useCreateCompose() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { project_id: string; name: string; content: string }) => apiPost("/api/v1/compose", body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.composes });
    },
  });
}

export interface AlertRule {
  id: string;
  org_id: string;
  name: string;
  metric: string;
  threshold: number;
  window_s: number;
  severity: string;
  enabled: boolean;
  target_app: string;
  created_at: string;
}

export interface AlertEvent {
  id: string;
  org_id: string;
  rule_id: string;
  app_id: string;
  app_name: string;
  severity: string;
  message: string;
  value: number;
  threshold: number;
  metric: string;
  created_at: string;
  resolved_at: string;
}

export function useAlertRules() {
  return useQuery({ queryKey: ["alert-rules"], queryFn: () => apiGet<AlertRule[]>("/api/v1/alerts/rules") });
}

export function useSetAlertRuleEnabled() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { id: string; enabled: boolean }) => apiPatch(`/api/v1/alerts/rules/${body.id}`, { enabled: body.enabled }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alert-rules"] }),
  });
}

export function useCreateAlertRule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: Partial<AlertRule>) => apiPost("/api/v1/alerts/rules", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alert-rules"] }),
  });
}

export function useDeleteAlertRule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/alerts/rules/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alert-rules"] }),
  });
}

export function useAlertEvents(limit = 30) {
  return useQuery({ queryKey: ["alert-events", limit], queryFn: () => apiGet<AlertEvent[]>(`/api/v1/alerts/events?limit=${limit}`) });
}

export function useResolveAlert() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiPost(`/api/v1/alerts/events/${id}/resolve`, {}),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alert-events"] }),
  });
}

export interface SnapshotSchedule {
  id: string;
  org_id: string;
  app_id: string;
  volume: string;
  name_prefix: string;
  cron: string;
  retention: number;
  enabled: boolean;
  last_run: string;
  next_run: string;
  created_at: string;
}

export function useSnapshotSchedules() {
  return useQuery({ queryKey: ["snap-schedules"], queryFn: () => apiGet<SnapshotSchedule[]>("/api/v1/snapshots/schedules") });
}

export function useCreateSnapshotSchedule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: Partial<SnapshotSchedule>) => apiPost<SnapshotSchedule>("/api/v1/snapshots/schedules", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["snap-schedules"] }),
  });
}

export function useDeleteSnapshotSchedule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/snapshots/schedules/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["snap-schedules"] }),
  });
}

export function useUpdateApp(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name?: string; image_retention?: number; port?: number; resources?: { cpus?: string; mem_mb?: number } }) => apiPatch(`/api/v1/apps/${appID}`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["app", appID] }),
  });
}

export function useDatabaseDetail(dbId: string) {
  return useQuery({
    queryKey: ["database", dbId],
    queryFn: () => apiGet<{ database: Database; dsn: string }>(`/api/v1/databases/${dbId}`),

  });
}

export function useAppCompose(appID: string) {
  return useQuery({
    queryKey: ["app-compose", appID],
    enabled: !!appID,
    queryFn: () => apiGet<{ compose: string }>(`/api/v1/apps/${appID}/compose`),
  });
}

export function useDeploymentCompose(appID: string, depID: string | null) {
  return useQuery({
    queryKey: ["dep-compose", appID, depID],
    enabled: !!depID,
    queryFn: () => apiGet<{ number: number; hash: string; compose: string }>(`/api/v1/apps/${appID}/deployments/${depID}/compose`),
  });
}

export interface ComposeStack {
  id: string;
  org_id: string;
  project_id: string;
  name: string;
  compose: string;
  status: string;
  created_at: string;
}

export function useComposeStacks() {
  return useQuery({ queryKey: qk.composes, queryFn: () => apiGet<ComposeStack[]>("/api/v1/compose") });
}

export function useComposeStack(id: string) {
  return useQuery({ queryKey: ["compose", id], enabled: !!id, queryFn: () => apiGet<ComposeStack>(`/api/v1/compose/${id}`) });
}

export function useComposeUp() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiPost(`/api/v1/compose/${id}/up`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["compose"] }),
  });
}

export function useComposeDown() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiPost(`/api/v1/compose/${id}/down`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["compose"] }),
  });
}

export function useDeleteCompose() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/compose/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["compose"] }),
  });
}

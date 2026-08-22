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

export interface OutWebhook {
  id: string;
  org_id: string;
  name: string;
  url: string;
  events: string[];
  enabled: boolean;
  created_at: string;
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

export type ResourceOwner = "aether" | "user" | "system" | "unknown";

export interface MonitoringResource {
  id: string;
  name: string;
  owner: ResourceOwner;
  service_type: string;
  service_id: string;
  project_id: string;
  state: string;
  active: boolean;
  cpu_percent: number;
  cpu_of_host: number;
  mem_usage: number;
  mem_limit: number;
  mem_percent: number;
  net_input: number;
  net_output: number;
  net_rx_rate: number;
  net_tx_rate: number;
  has_net_rate: boolean;
  block_input: number;
  block_output: number;
  block_rx_rate: number;
  block_tx_rate: number;
  has_block_rate: boolean;
  storage: number | null;
}

export interface MonitoringAggregate {
  cpu_of_host: number;
  mem_usage: number;
  mem_percent: number;
  net_rx_rate: number;
  net_tx_rate: number;
  storage_usage: number;
  count: number;
  running_count: number;
  available: boolean;
}

export interface MonitoringHost {
  cpu_percent: number;
  cpu_cores: number;
  runtime_cores: number;
  mem_total: number;
  mem_used: number;
  mem_percent: number;
  disk_total: number;
  disk_used: number;
  disk_percent: number;
  net_rx_rate: number;
  net_tx_rate: number;
  load: number[];
  uptime: number;
  hostname: string;
  os: string;
  source: string;
}

export interface MonitoringCollector {
  collect_count: number;
  error_count: number;
  last_collect_ms: number;
  resources: number;
  with_stats: number;
  last_error?: string;
  up_since: string;
}

export interface MonitoringSnapshot {
  ts: string;
  host: MonitoringHost;
  aether: MonitoringAggregate;
  user: MonitoringAggregate;
  system: MonitoringAggregate;
  resources: MonitoringResource[];
  collector: MonitoringCollector;
}

export interface MonitoringHistoryPoint {
  ts: number;
  host_cpu: number;
  host_mem: number;
  aether_cpu: number;
  aether_mem: number;
  aether_mem_pct: number;
  user_cpu: number;
  user_mem: number;
  user_mem_pct: number;
  net_rx: number;
  net_tx: number;
}

export interface MonitoringResourcePoint {
  ts: number;
  cpu: number;
  mem: number;
  net_rx: number;
  net_tx: number;
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

export type EnvSummary = Environment;

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

export interface ProjectVariable {
  id: string;
  project_id: string;
  key: string;
  value: string;
  is_secret: boolean;
  created_at: string;
  updated_at: string;
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

export interface EventEnvelope {
  id: string;
  type: string;
  version: number;
  ts: string;
  org_id?: string;
  project_id?: string;
  app_id?: string;
  resource_type?: string;
  resource_id?: string;
  correlation_id?: string;
  causation_id?: string;
  seq: number;
  message?: string;
  payload?: Record<string, unknown>;
}

export interface RealtimeInbound {
  op: "event" | "subscribed" | "unsubscribed" | "dropped" | "pong" | "error";
  ev?: EventEnvelope;
  replay?: boolean;
  n?: number;
  code?: string;
  message?: string;
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

export interface DeployCompare {
  image: { from: string; to: string };
  commit: { from: string; to: string };
  status_a: string;
  status_b: string;
  env_added: string[];
  env_removed: string[];
  env_changed: string[];
}

export interface DeploymentLog {
  number: number;
  status: string;
  error: string;
  content: string;
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

export interface ComposeStack {
  id: string;
  org_id: string;
  project_id: string;
  name: string;
  compose: string;
  status: string;
  created_at: string;
}

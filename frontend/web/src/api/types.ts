export interface User {
  id: string;
  email: string;
  name: string;
  created_at: string;
}

export interface Org {
  id: string;
  name: string;
  role: string;
}

export interface OrgDetail {
  id: string;
  slug: string;
  name: string;
  description: string;
  avatar: string;
  color: string;
  owner_user_id: string;
  role?: string;
}

export interface OrgMember {
  org_id: string;
  user_id: string;
  email: string;
  name: string;
  role: string;
  projects: string[];
}

export interface AuditLog {
  id: string;
  org_id: string;
  user_id: string;
  action: string;
  resource_type: string;
  resource_id: string;
  details: string;
  created_at: string;
}

export interface Me {
  id: string;
  email: string;
  name: string;
  global_role?: string;
  org: Org;
  organizations?: OrgRoleView[];
}

export interface OrgRoleView {
  id: string;
  slug: string;
  name: string;
  color?: string;
  role: string;
}

export interface Member {
  user_id: string;
  email: string;
  name: string;
  role: string;
}

export interface ApiKey {
  id: string;
  name: string;
  scopes: string[];
  created_at: string;
}

export interface Project {
  id: string;
  org_id: string;
  name: string;
  slug?: string;
  description?: string;
  color?: string;
  created_at: string;
}

export interface Resources {
  cpus: string;
  mem_mb: number;
}

export interface HealthCheck {
  enabled: boolean;
  path: string;
  interval_ms: number;
  timeout_ms: number;
  retries: number;
}

export interface Volume {
  name: string;
  mount_path: string;
}

export interface App {
  image_retention?: number;
  id: string;
  org_id: string;
  project_id: string;
  name: string;
  source_type: "image" | "git";
  image: string;
  git_url: string;
  git_branch: string;
  dockerfile: string;
  build_type: string;
  preview_domain: string;
  server_id: string;
  cluster_id: string;
  environment_id: string;
  port: number;
  resources: Resources;
  health_check: HealthCheck;
  volumes: Volume[];
  latest_deployment?: {
    status: string;
  };
  created_at: string;
  updated_at: string;
}

export interface EnvVar {
  app_id: string;
  name: string;
  value: string;
  secret: boolean;
}

export interface AppDetail {
  app: App;
  env: EnvVar[];
  internal_host?: string;
  internal_network?: string;
}

export interface ResolvedVariable {
  key: string;
  value: string;
  source: string;
  secret: boolean;
}

export interface Deployment {
  id: string;
  app_id: string;
  number: number;
  status: string;
  commit: string;
  image_ref: string;
  container_id: string;
  error: string;
  created_at: string;
  started_at: string;
  finished_at: string;
}

export interface ContainerStats {
  cpu_percent: number;
  mem_bytes: number;
  mem_limit: number;
  mem_percent: number;
}

export interface Stats {
  state: string;
  stats: ContainerStats;
}

export interface Domain {
  id: string;
  app_id: string;
  service_type: string;
  server_id: string;
  host: string;
  https: boolean;
  path: string;
  internal_path: string;
  strip_path: boolean;
  container_port: number;
  status: string;
  cert_status: string;
  created_at: string;
  updated_at: string;
}

export interface HostInfo {
  public_ip: string;
  free_domain_base: string;
}

export interface TimelineEvent {
  id: string;
  aggregate_type: string;
  aggregate_id: string;
  sequence: number;
  type: string;
  payload: string;
  ts: string;
}

export interface Backup {
  id: string;
  path: string;
  size: number;
  created_at: string;
  kind?: string;
  dest?: string;
  app_id?: string;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export interface Database {
  id: string;
  org_id: string;
  project_id: string;
  name: string;
  engine: string;
  version: string;
  port: number;
  db_name: string;
  mem_mb: number;
  storage_mb: number;
  status: string;
  container_id: string;
  user: string;
  created_at: string;
}

export interface CronJob {
  id: string;
  app_id: string;
  name: string;
  schedule: string;
  command: string;
  enabled: boolean;
  last_run: string;
  next_run: string;
}

export interface Worker {
  id: string;
  app_id: string;
  name: string;
  command: string;
  replicas: number;
  enabled: boolean;
  status: string;
  container_id: string;
}

export interface Preview {
  id: string;
  app_id: string;
  branch: string;
  deployment_id: string;
  container_id: string;
  domain: string;
  status: string;
  created_at: string;
}

export interface Template {
  id: string;
  name: string;
  description: string;
  category: string;
  icon: string;
  version: string;
  definition: string;
}

export type DestinationType = "aws" | "cloudflare-r2" | "minio" | "custom-s3" | "google-drive";
export type OAuthStatus = "" | "connecting" | "connected" | "reauth_required" | "error";

export interface S3Destination {
  id: string;
  org_id: string;
  name: string;
  type: DestinationType;
  endpoint: string;
  bucket: string;
  region: string;
  account_id: string;
  oauth_status: OAuthStatus;
  oauth_email: string;
  google_client_id: string;
  created_at: string;
  updated_at: string;
}

export interface NotificationChannel {
  id: string;
  org_id: string;
  name: string;
  type: string;
  enabled: boolean;
  created_at: string;
}

export interface StudioMeta {
  engine: string;
  version: string;
  status: string;
  schemas: number;
  tables: number;
  views: number;
  functions: number;
}

export interface StudioObject {
  name: string;
  type: string;
}

export interface StudioObjectSummary {
  tables: number;
  views: number;
  mat_views: number;
  functions: number;
  procedures: number;
  triggers: number;
  sequences: number;
  types: number;
  extensions: number;
}

export interface StudioColumn {
  name: string;
  type: string;
  nullable: boolean;
  default?: string | null;
  primary_key: boolean;
  unique: boolean;
  identity: string;
  generated: string;
  collation?: string | null;
  comment: string;
}

export interface StudioIndex {
  name: string;
  method: string;
  unique: boolean;
  columns: string[];
  predicate: string;
  primary: boolean;
}

export interface StudioConstraint {
  name: string;
  type: string;
  column: string;
  ref_table?: string;
  ref_column?: string;
  definition: string;
}

export interface StudioForeignKey {
  name: string;
  columns: string[];
  ref_table: string;
  ref_columns: string[];
  on_delete: string;
  on_update: string;
}

export interface StudioTrigger {
  name: string;
  event: string;
  timing: string;
  function: string;
  enabled: string;
}

export interface StudioTableDetail {
  schema: string;
  name: string;
  type: string;
  owner: string;
  columns: StudioColumn[];
  indexes: StudioIndex[];
  constraints: StudioConstraint[];
  foreign_keys: StudioForeignKey[];
  triggers: StudioTrigger[];
}

export interface StudioQueryError {
  message: string;
  position?: number;
  line?: number;
  column?: number;
  suggestion?: string;
}

export interface StudioQueryResult {
  columns: string[];
  rows: unknown[][];
  row_count: number;
  duration_ms: number;
  read_only: boolean;
  truncated: boolean;
  message?: string;
  error?: StudioQueryError;
}

export interface StudioExecResult {
  message: string;
  command_tag: string;
  duration_ms: number;
}

export interface BackupSchedule {
  type: "hourly" | "daily" | "weekly" | "biweekly" | "custom";
  minute?: number;
  at?: string;
  day_of_week?: string;
  start_date?: string;
  cron?: string;
  timezone: string;
}

export interface BackupRetention {
  type: "all" | "latest";
}

export interface BackupConfig {
  id: string;
  database_id: string;
  enabled: boolean;
  destination_id: string;
  path_prefix: string;
  schedule: BackupSchedule;
  retention: BackupRetention;
  next_run_at?: string | null;
}

export interface BackupJob {
  id: string;
  database_id: string;
  status: string;
  trigger: string;
  engine: string;
  engine_version: string;
  format: string;
  size_bytes: number;
  checksum: string;
  storage_key: string;
  error_code: string;
  error_message: string;
  started_at?: string | null;
  completed_at?: string | null;
}

export interface RestoreJob {
  id: string;
  backup_id: string;
  target_database_id: string;
  status: string;
  error_code: string;
  error_message: string;
  started_at?: string | null;
  completed_at?: string | null;
}

export interface PreflightCheck {
  name: string;
  ok: boolean;
  message: string;
}

export interface PreflightResult {
  compatible: boolean;
  ready: boolean;
  checks: PreflightCheck[];
}

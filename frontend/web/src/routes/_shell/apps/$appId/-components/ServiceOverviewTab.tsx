import {
  Check,
  Code,
  Copy,
  GitBranch,
  MagnifyingGlass,
} from "@phosphor-icons/react";
import { Badge, Button, Card, EmptyState } from "@aether/design-system";
import type {
  App,
  Deployment,
  Domain,
  ServiceSummary,
  Stats,
} from "@/api/types";
import type {
  ServiceSource,
  SourceControlRepository,
} from "@/hooks/use-source-control";
import { LiveLogs } from "./LiveLogs";
import { ServiceControlPanel } from "./ServiceControlPanel";
import { ServiceProviderPanel } from "./ServiceProviderPanel";

export function ServiceOverviewTab({
  app,
  service,
  runtimeId,
  liveURL,
  source,
  repositories,
  sourceLoading,
  sourceError,
  providerEditing,
  providerRepositoryId,
  providerBranch,
  providerRootDirectory,
  providerWatchPaths,
  providerWatchRootFiles,
  providerAutoDeploy,
  savingSource,
  autodeploy,
  canManageSource,
  runtimeStats,
  domains,
  latest,
  logsID,
  overviewLogsEndpoint,
  running,
  restarting,
  stopping,
  starting,
  copied,
  connectionVisible,
  connectionDSN,
  connectionError,
  connectionCopied,
  onOpenTerminal,
  onVisit,
  onDeploy,
  onRestart,
  onStop,
  onStart,
  onAutodeployChange,
  onProviderEdit,
  onProviderCancel,
  onProviderSave,
  onRepositoryChange,
  onBranchChange,
  onRootDirectoryChange,
  onWatchPathsChange,
  onWatchRootFilesChange,
  onProviderAutoDeployChange,
  onCopyURL,
  onShowConnection,
  onCopyConnection,
  onViewDeploymentLogs,
}: {
  app: App;
  service: ServiceSummary;
  runtimeId: string;
  liveURL: string | null;
  source?: ServiceSource;
  repositories?: SourceControlRepository[];
  sourceLoading: boolean;
  sourceError: boolean;
  canManageSource: boolean;
  providerEditing: boolean;
  providerRepositoryId: string;
  providerBranch: string;
  providerRootDirectory: string;
  providerWatchPaths: string;
  providerWatchRootFiles: boolean;
  providerAutoDeploy: boolean;
  savingSource: boolean;
  autodeploy: boolean;
  runtimeStats?: Stats;
  domains?: Domain[];
  latest?: Deployment;
  logsID: string;
  overviewLogsEndpoint?: string;
  running: boolean;
  restarting: boolean;
  stopping: boolean;
  starting: boolean;
  copied: boolean;
  connectionVisible: boolean;
  connectionDSN?: string;
  connectionError: boolean;
  connectionCopied: boolean;
  onOpenTerminal: () => void;
  onVisit: () => void;
  onDeploy: () => void;
  onRestart: () => void;
  onStop: () => void;
  onStart: () => void;
  onAutodeployChange: (checked: boolean) => void;
  onProviderEdit: () => void;
  onProviderCancel: () => void;
  onProviderSave: () => void;
  onRepositoryChange: (value: string) => void;
  onBranchChange: (value: string) => void;
  onRootDirectoryChange: (value: string) => void;
  onWatchPathsChange: (value: string) => void;
  onWatchRootFilesChange: (value: boolean) => void;
  onProviderAutoDeployChange: (value: boolean) => void;
  onCopyURL: () => void;
  onShowConnection: () => void;
  onCopyConnection: () => void;
  onViewDeploymentLogs: (id: string) => void;
}) {
  return (
    <>
      <ServiceControlPanel
        running={running}
        autodeploy={autodeploy}
        autodeployDisabled={sourceLoading || savingSource}
        restarting={restarting}
        stopping={stopping}
        starting={starting}
        onOpenTerminal={onOpenTerminal}
        onVisit={onVisit}
        onDeploy={onDeploy}
        onRestart={onRestart}
        onStop={onStop}
        onStart={onStart}
        onAutodeployChange={onAutodeployChange}
        canManageSource={canManageSource}
      />
      {canManageSource ? (
        <>
          {sourceError ? (
            <div
              role="alert"
              className="mb-8 rounded border border-error/40 bg-error/10 px-md py-sm font-body-sm text-body-sm text-error"
            >
              Unable to load source control settings.
            </div>
          ) : null}
          <ServiceProviderPanel
            app={app}
            source={source}
            repositories={repositories}
            editing={providerEditing}
            repositoryId={providerRepositoryId}
            branch={providerBranch}
            rootDirectory={providerRootDirectory}
            watchPaths={providerWatchPaths}
            watchRootFiles={providerWatchRootFiles}
            autoDeploy={providerAutoDeploy}
            saving={savingSource}
            onRepositoryChange={onRepositoryChange}
            onBranchChange={onBranchChange}
            onRootDirectoryChange={onRootDirectoryChange}
            onWatchPathsChange={onWatchPathsChange}
            onWatchRootFilesChange={onWatchRootFilesChange}
            onAutoDeployChange={onProviderAutoDeployChange}
            onEdit={onProviderEdit}
            onCancel={onProviderCancel}
            onSave={onProviderSave}
          />
        </>
      ) : null}
      <div className="grid grid-cols-1 gap-lg lg:grid-cols-3">
        <div className="flex flex-col gap-lg">
          <ServiceDetailsCard
            app={app}
            service={service}
            liveURL={liveURL}
            domains={domains}
            runtimeStats={runtimeStats}
            connectionVisible={connectionVisible}
            connectionDSN={connectionDSN}
            connectionError={connectionError}
            connectionCopied={connectionCopied}
            copied={copied}
            onCopyURL={onCopyURL}
            onShowConnection={onShowConnection}
            onCopyConnection={onCopyConnection}
          />
          {service.runtime?.containers?.length ? (
            <Card variant="default">
              <div className="mb-md flex items-center justify-between">
                <div>
                  <p className="font-label-caps text-label-caps text-primary">
                    Runtime
                  </p>
                  <h3 className="mt-xs font-body-md text-body-md font-semibold text-on-surface">
                    Containers
                  </h3>
                </div>
                <span className="rounded-full bg-surface-container-high px-sm py-1 font-code-md text-code-md text-on-surface-variant">
                  {service.runtime.containers.length}
                </span>
              </div>
              <div className="space-y-sm">
                {service.runtime.containers.map((container, index) => (
                  <div
                    key={`${container.id || container.name}-${index}`}
                    className="flex items-center justify-between gap-md rounded-lg border border-outline-variant/60 bg-surface-container-low px-sm py-2"
                  >
                    <span className="min-w-0 truncate font-code-md text-code-md text-on-surface">
                      {container.name || container.id}
                    </span>
                    <Badge
                      tone={
                        container.status === "running" ? "success" : "neutral"
                      }
                      dot
                    >
                      {container.status}
                    </Badge>
                  </div>
                ))}
              </div>
            </Card>
          ) : null}
          <div className="grid grid-cols-1 gap-md sm:grid-cols-2">
            <UsageCard
              label="CPU usage"
              value={`${runtimeStats?.stats?.cpu_percent?.toFixed(0) ?? "0"}%`}
              percent={runtimeStats?.stats?.cpu_percent ?? 0}
            />
            <UsageCard
              label="Memory"
              value={formatBytes(runtimeStats?.stats?.mem_bytes ?? 0)}
              percent={
                runtimeStats?.stats?.mem_limit
                  ? ((runtimeStats.stats.mem_bytes ?? 0) /
                      runtimeStats.stats.mem_limit) *
                    100
                  : 0
              }
            />
          </div>
        </div>
        <div className="flex flex-col gap-lg lg:col-span-2">
          <LatestDeployment
            latest={latest}
            branch={app.git_branch}
            onViewLogs={onViewDeploymentLogs}
          />
          <Card variant="default" padding="none" className="overflow-hidden">
            <div className="flex flex-wrap items-center justify-between gap-sm border-b border-outline-variant bg-surface-container-low px-lg py-md">
              <div>
                <p className="font-label-caps text-label-caps text-primary">
                  Observability
                </p>
                <h3 className="mt-xs font-body-md text-body-md font-semibold text-on-surface">
                  Live logs
                </h3>
              </div>
              <span className="inline-flex items-center gap-xs rounded-full border border-status-success/30 bg-status-success-container/20 px-sm py-1 font-label-caps text-label-caps text-status-success">
                <span className="size-1.5 rounded-full bg-status-success" />
                Streaming
              </span>
            </div>
            <div className="p-sm">
              <LiveLogs
                serviceId={logsID}
                enabled={Boolean(service)}
                endpoint={overviewLogsEndpoint}
              />
            </div>
          </Card>
        </div>
      </div>
    </>
  );
}

function ServiceDetailsCard({
  app,
  service,
  liveURL,
  domains,
  runtimeStats,
  connectionVisible,
  connectionDSN,
  connectionError,
  connectionCopied,
  copied,
  onCopyURL,
  onShowConnection,
  onCopyConnection,
}: Pick<
  Parameters<typeof ServiceOverviewTab>[0],
  | "app"
  | "service"
  | "liveURL"
  | "domains"
  | "runtimeStats"
  | "connectionVisible"
  | "connectionDSN"
  | "connectionError"
  | "connectionCopied"
  | "copied"
  | "onCopyURL"
  | "onShowConnection"
  | "onCopyConnection"
>) {
  return (
    <Card variant="elevated">
      <div className="mb-lg flex items-start justify-between gap-md">
        <div>
          <p className="font-label-caps text-label-caps text-primary">
            Configuration
          </p>
          <h3 className="mt-xs font-headline-sm text-headline-sm font-semibold text-on-surface">
            Service details
          </h3>
        </div>
        <span className="rounded-lg bg-primary/10 p-sm text-primary">
          <Code size={18} aria-hidden="true" />
        </span>
      </div>
      <div className="space-y-md">
        <CopyField
          label="Live URL"
          value={liveURL ?? "—"}
          copied={copied}
          onCopy={onCopyURL}
        />
        <div>
          {service.kind === "database" ? (
            <>
              <span className="mb-1 block font-body-sm text-body-sm text-on-surface-variant">
                Connection
              </span>
              {!connectionVisible ? (
                <Button variant="ghost" onClick={onShowConnection}>
                  Show connection
                </Button>
              ) : connectionDSN ? (
                <CopyField
                  value={maskConnection(connectionDSN)}
                  copied={connectionCopied}
                  onCopy={onCopyConnection}
                />
              ) : (
                <span className="font-body-sm text-body-sm text-on-surface-variant">
                  {connectionError
                    ? "Connection string unavailable"
                    : "Loading connection..."}
                </span>
              )}
            </>
          ) : null}
        </div>
        <div>
          <span className="mb-1 block font-body-sm text-body-sm text-on-surface-variant">
            Internal host{" "}
            <MagnifyingGlass
              size={14}
              className="ml-1 inline text-muted-foreground/50"
            />
          </span>
          <Value value="—" />
        </div>
        <div>
          <span className="mb-1 block font-body-sm text-body-sm text-on-surface-variant">
            Source
          </span>
          <Value
            value={app.source_type === "image" ? app.image : app.git_url}
            icon={Code}
          />
        </div>
        <div className="grid grid-cols-2 gap-md border-t border-outline-variant pt-md">
          <Stat
            label="Type"
            value={app.source_type === "image" ? "OCI Image" : "Git"}
          />
          <Stat label="Port" value={`:${app.port}`} />
          <Stat
            label="Health"
            value={runtimeStats?.state === "running" ? "healthy" : "—"}
          />
          <Stat
            label="Memory"
            value={formatBytes(runtimeStats?.stats?.mem_bytes ?? 0)}
          />
        </div>
        {domains?.length ? (
          <div className="border-t border-outline-variant pt-md">
            <span className="mb-1 block font-body-sm text-body-sm text-on-surface-variant">
              Domains
            </span>
            <div className="flex flex-wrap gap-sm">
              {domains.map((domain, index) => (
                <a
                  key={`${domain.id || domain.host}-${domain.path}-${index}`}
                  href={`${domain.https ? "https" : "http"}://${domain.host}`}
                  target="_blank"
                  rel="noreferrer"
                  className="rounded-md border border-outline-variant bg-surface-container-low px-sm py-1.5 font-code-md text-code-md text-primary transition-[background-color,border-color] duration-150 hover:border-primary/50 hover:bg-primary/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  {domain.host}
                </a>
              ))}
            </div>
          </div>
        ) : null}
      </div>
    </Card>
  );
}

function LatestDeployment({
  latest,
  branch,
  onViewLogs,
}: {
  latest?: Deployment;
  branch: string;
  onViewLogs: (id: string) => void;
}) {
  return (
    <section className="relative overflow-hidden rounded-2xl border border-outline-variant bg-gradient-to-br from-surface-container to-surface-container-low p-lg shadow-md">
      <div className="mb-lg flex items-center justify-between gap-md">
        <div>
          <p className="font-label-caps text-label-caps text-primary">
            Delivery
          </p>
          <h3 className="mt-xs font-headline-sm text-headline-sm font-semibold text-on-surface">
            Latest deployment
          </h3>
        </div>
        {latest ? (
          <Badge tone={deploymentTone(latest.status)} dot live>
            {latest.status === "ready" ? "Success" : latest.status}
          </Badge>
        ) : null}
      </div>
      {latest ? (
        <div className="flex flex-col gap-lg sm:flex-row sm:items-center">
          <div className="flex min-w-0 flex-1 items-start gap-md">
            <div className="flex size-11 shrink-0 items-center justify-center rounded-xl border border-primary/20 bg-primary/10 text-primary">
              <GitBranch size={20} aria-hidden="true" />
            </div>
            <div className="min-w-0">
              <h4 className="font-body-md text-body-md font-semibold text-on-surface">
                Deployment #{latest.number}
              </h4>
              <p className="mt-1 truncate font-code-md text-code-md text-on-surface-variant">
                {latest.commit ? latest.commit.slice(0, 8) : branch || "image"}{" "}
                <span className="text-on-surface-variant/50">·</span>{" "}
                {latest.image_ref || "image"}
              </p>
              <div className="mt-md flex flex-wrap items-center gap-sm font-code-md text-code-md text-on-surface-variant">
                <span>{branch || "image"}</span>
                <span className="text-on-surface-variant/50">·</span>
                <span>
                  {new Date(latest.created_at).toLocaleString(undefined, {
                    day: "2-digit",
                    month: "short",
                    hour: "2-digit",
                    minute: "2-digit",
                  })}
                </span>
              </div>
            </div>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => onViewLogs(latest.id)}
          >
            View logs
          </Button>
        </div>
      ) : (
        <EmptyState
          title="No deployments yet"
          description="Deploy the service to create its first deployment."
          className="border-0"
        />
      )}
    </section>
  );
}

function CopyField({
  label,
  value,
  copied,
  onCopy,
}: {
  label?: string;
  value: string;
  copied: boolean;
  onCopy: () => void;
}) {
  return (
    <div>
      {label ? (
        <span className="mb-1 block font-body-sm text-body-sm text-on-surface-variant">
          {label}
        </span>
      ) : null}
      <div className="group flex items-center justify-between gap-sm rounded-lg border border-outline-variant bg-surface-container-low p-sm">
        <span className="min-w-0 truncate font-code-md text-code-md text-primary">
          {value}
        </span>
        <button
          type="button"
          onClick={onCopy}
          className="shrink-0 rounded-md p-1 text-on-surface-variant outline-none transition-[background-color,color,transform] duration-150 hover:bg-primary/10 hover:text-primary active:scale-[0.96] focus-visible:ring-2 focus-visible:ring-ring"
          title={copied ? "Copied!" : "Copy"}
          aria-label={copied ? "Copied" : `Copy ${label ?? "value"}`}
        >
          {copied ? (
            <Check size={18} aria-hidden="true" />
          ) : (
            <Copy size={18} aria-hidden="true" />
          )}
        </button>
      </div>
    </div>
  );
}

function Value({ value, icon: Icon }: { value: string; icon?: typeof Code }) {
  return (
    <div className="flex items-center gap-2 rounded border border-outline-variant bg-surface-container-low p-3">
      <span className="truncate font-code-md text-code-md text-on-surface">
        {Icon ? <Icon size={16} className="mr-2 inline" /> : null}
        {value}
      </span>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <span className="mb-1 block font-label-caps text-label-caps text-on-surface-variant">
        {label}
      </span>
      <span className="font-body-md text-body-md text-on-surface">{value}</span>
    </div>
  );
}
function UsageCard({
  label,
  value,
  percent,
}: {
  label: string;
  value: string;
  percent: number;
}) {
  return (
    <div className="flex flex-col justify-between rounded-xl border border-outline-variant bg-surface-container p-md">
      <div className="flex items-center justify-between gap-sm">
        <span className="font-label-caps text-label-caps text-on-surface-variant">
          {label}
        </span>
        <span className="font-code-md text-code-md text-primary">
          {Math.round(percent)}%
        </span>
      </div>
      <div className="mt-lg">
        <span className="font-headline-sm text-headline-sm font-semibold text-on-surface">
          {value}
        </span>
        <div className="mt-sm h-1.5 w-full overflow-hidden rounded-full bg-surface-container-high">
          <div
            className="h-full rounded-full bg-primary transition-[width] duration-300"
            style={{ width: `${Math.min(100, Math.max(0, percent))}%` }}
          />
        </div>
      </div>
    </div>
  );
}
function deploymentTone(
  status: string,
): "danger" | "success" | "warning" | "neutral" {
  if (status === "failed") return "danger";
  if (status === "ready" || status === "running") return "success";
  if (["queued", "building", "starting", "health_checking"].includes(status))
    return "warning";
  return "neutral";
}
function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(1)} KiB`;
  return `${(bytes / 1024 ** 2).toFixed(1)} MiB`;
}
function maskConnection(value: string): string {
  return value.replace(/:\/\/([^:]+):([^@]+)@/, "://$1:••••••••@");
}

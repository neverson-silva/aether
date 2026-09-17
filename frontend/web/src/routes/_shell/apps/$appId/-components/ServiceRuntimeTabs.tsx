import { Database, Gauge, HardDrives, Target } from "@phosphor-icons/react";
import { Button, Card, NativeSelect, RuntimeStatus, VariableEditor, type RuntimeStatusValue, type VariableRow } from "@aether/design-system";
import type { Deployment, Stats, TimelineEvent } from "@/api/types";
import type { ServiceContainer } from "@/hooks/use-service-containers";
import { LiveLogs } from "./LiveLogs";
import { Metric } from "./Metric";

export function ServiceVariablesTab({
  variableKey,
  variables,
  onChange,
  onImport,
  onExport,
  onSave,
  saving,
}: {
  variableKey: string;
  variables: VariableRow[];
  onChange: (variables: VariableRow[]) => void;
  onImport: () => VariableRow[] | undefined;
  onExport: () => void;
  onSave: () => void;
  saving: boolean;
}) {
  return (
    <Card>
      <div className="mb-md">
        <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Service Variables</h2>
        <p className="mt-xs font-body-sm text-body-sm text-on-surface-variant/70">Variables injected only into this service. They override environment variables.</p>
      </div>
      <VariableEditor
        key={variableKey}
        variables={variables}
        onChange={onChange}
        onImport={onImport}
        onExport={onExport}
        className="max-h-[min(42rem,calc(100dvh-18rem))]"
      />
      <div className="mt-md flex justify-end">
        <Button type="button" onClick={onSave} loading={saving} loadingLabel="Saving variables">Save variables</Button>
      </div>
    </Card>
  );
}

export function ServiceLogsTab({
  serviceId,
  enabled,
  endpoint,
  containers,
  logContainer,
  onContainerChange,
  timeline,
  latest,
  deploymentActive,
}: {
  serviceId: string;
  enabled: boolean;
  endpoint?: string;
  containers?: ServiceContainer[];
  logContainer: string;
  onContainerChange: (value: string) => void;
  timeline?: TimelineEvent[];
  latest?: Deployment;
  deploymentActive: boolean;
}) {
  return (
    <Card>
      <div className="flex flex-wrap items-center justify-between gap-md">
        <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Live Logs</h2>
        {(containers?.length ?? 0) > 1 ? (
          <NativeSelect
            aria-label="Log container"
            value={logContainer}
            onChange={(event) => onContainerChange(event.target.value)}
            options={[{ label: "All containers", value: "" }, ...(containers ?? []).map((container) => ({ label: container.name, value: container.id }))]}
          />
        ) : null}
      </div>
      <div className="mt-md">
        <LiveLogs serviceId={serviceId} enabled={enabled} endpoint={endpoint} />
      </div>
      <h2 className="mb-md mt-lg font-label-caps text-label-caps text-on-surface-variant uppercase">Event timeline</h2>
      <div className="sidebar-scroll max-h-[260px] space-y-1 overflow-y-auto">
        {(timeline ?? []).map((event, index) => (
          <div key={`${event.id || event.sequence || event.ts}-${index}`} className="flex items-stretch gap-sm font-code-md text-code-md">
            <div className="flex w-3 flex-col items-center">
              <span className={`mt-1.5 size-2 shrink-0 rounded-full ${index === 0 ? "bg-status-success" : "bg-outline-variant/40"}`} />
            </div>
            <div className="flex min-w-0 items-center gap-sm">
              <span className="shrink-0 text-on-surface-variant/50">{new Date(event.ts).toLocaleString(undefined, { day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit" })}</span>
              <span className="truncate text-primary">{event.type}</span>
            </div>
          </div>
        ))}
        {latest && deploymentActive ? <div className="flex items-center gap-sm pl-1.5"><span className="size-2 rounded-full bg-status-success" /><span className="font-code-md text-code-md text-on-surface-variant">live</span></div> : null}
      </div>
    </Card>
  );
}

export function ServiceMetricsTab({ runtimeStatus, live, stats }: { runtimeStatus: RuntimeStatusValue; live: boolean; stats?: Stats }) {
  return (
    <Card>
      <div className="mb-md flex items-center justify-between">
        <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Metrics</h2>
        <RuntimeStatus status={runtimeStatus} live={live} />
      </div>
      {stats?.stats ? (
        <div className="grid grid-cols-2 gap-md md:grid-cols-4">
          <Metric label="CPU" value={`${stats.stats.cpu_percent?.toFixed(2) ?? 0}%`} icon={Gauge} />
          <Metric label="Memory" value={formatBytes(stats.stats.mem_bytes ?? 0)} icon={HardDrives} />
          <Metric label="Limit" value={formatBytes(stats.stats.mem_limit ?? 0)} icon={Database} />
          <Metric label="Mem %" value={`${stats.stats.mem_percent?.toFixed(1) ?? 0}%`} icon={Target} />
        </div>
      ) : <p className="font-body-sm text-body-sm text-on-surface-variant">No active container.</p>}
    </Card>
  );
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(1)} KiB`;
  return `${(bytes / 1024 ** 2).toFixed(1)} MiB`;
}

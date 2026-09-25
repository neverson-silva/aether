import {
  Button,
  Card,
  EmptyState,
  Marker,
  Progress,
  RuntimeStatus,
  Skeleton,
  Typography,
} from '@aether/elisyum-ds'
import {
  ActivityIcon,
  ChartLineUp,
  Cpu,
  HardDrives,
  Memory,
  Pulse,
} from '@phosphor-icons/react'
import type { MonitoringAggregate, MonitoringResource } from '../hooks/types'
import { useMonitoring } from '../hooks/use-monitoring'

export function MonitoringField() {
  const query = useMonitoring()
  const snapshot = query.snapshot
  const resources = Array.isArray(snapshot?.resources) ? snapshot.resources : []
  const ownedResources = resources.filter(
    (resource) => resource.owner === 'aether' || resource.owner === 'user',
  )
  if (query.error)
    return (
      <div className="grid gap-6">
        <MonitoringHeader connected={query.connected} />
        <div className="flex flex-wrap items-center justify-between gap-4 rounded-xl border border-danger/45 bg-danger-soft px-4 py-4 text-supporting text-danger-strong">
          <span className="flex items-center gap-3">
            <Pulse size={18} />
            {query.error}
          </span>
          <Button
            tone="neutral"
            onClick={() => window.location.reload()}
          >
            Reconnect stream
          </Button>
        </div>
      </div>
    )
  if (!snapshot)
    return (
      <div className="grid gap-7">
        <MonitoringHeader connected={query.connected} />
        <MonitoringSkeleton />
      </div>
    )
  return (
    <div className="grid gap-7">
      <MonitoringHeader connected={query.connected} />
      <section aria-label="Live telemetry">
        <Card className="grid gap-4 rounded-2xl border-border-subtle bg-surface-1 p-4 md:grid-cols-4 md:gap-0">
          <TelemetryCell
            icon={<Cpu size={16} />}
            label="HOST CPU"
            value={snapshot ? `${snapshot.host.cpu_percent.toFixed(1)}%` : '—'}
            detail={query.connected ? 'Streaming now' : 'Waiting for stream'}
            tone={query.connected ? 'success' : 'warning'}
          />
          <TelemetryCell
            icon={<Memory size={16} />}
            label="HOST MEMORY"
            value={snapshot ? `${snapshot.host.mem_percent.toFixed(1)}%` : '—'}
            detail={
              snapshot
                ? `${formatBytes(snapshot.host.mem_used)} of ${formatBytes(snapshot.host.mem_total)}`
                : 'Memory telemetry'
            }
            tone="accent"
          />
          <TelemetryCell
            icon={<HardDrives size={16} />}
            label="HOST STORAGE"
            value={snapshot ? `${snapshot.host.disk_percent.toFixed(1)}%` : '—'}
            detail={
              snapshot
                ? `${formatBytes(snapshot.host.disk_used)} of ${formatBytes(snapshot.host.disk_total)}`
                : 'Storage telemetry'
            }
            tone="warning"
          />
          <TelemetryCell
            icon={<HardDrives size={16} />}
            label="RESOURCES"
            value={snapshot ? String(ownedResources.length) : '—'}
            detail={
              snapshot
                ? `${snapshot.collector.with_stats} with runtime stats`
                : 'Collector pending'
            }
            tone="neutral"
          />
        </Card>
      </section>
      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(18rem,0.45fr)]">
        <Card className="grid gap-4 rounded-2xl border-border-subtle bg-surface-1 p-5">
          <div className="flex items-end justify-between gap-4">
            <div className="grid gap-1">
              <span className="text-label tracking-[0.12em] text-text-subtle">
                PRESSURE DISTRIBUTION
              </span>
              <Typography
                as="h2"
                role="section-title"
              >
                Runtime allocation
              </Typography>
            </div>
            <RuntimeStatus
              status={query.connected ? 'healthy' : 'degraded'}
              label={query.connected ? 'Streaming' : 'Disconnected'}
            />
          </div>
          <div className="grid gap-2">
            <AggregateLine
              label="Platform"
              aggregate={snapshot?.aether}
            />
            <AggregateLine
              label="User workloads"
              aggregate={snapshot?.user}
            />
            <AggregateLine
              label="System"
              aggregate={snapshot?.system}
            />
          </div>
        </Card>
        <Card className="grid content-start gap-5 rounded-2xl border-border-subtle bg-surface-1 p-5">
          <span className="text-label tracking-[0.12em] text-text-subtle">
            COLLECTOR
          </span>
          <Fact
            label="STATUS"
            value={query.connected ? 'Receiving events' : 'Waiting'}
          />
          <Fact
            label="LAST COLLECT"
            value={snapshot ? `${snapshot.collector.last_collect_ms} ms` : '—'}
          />
          <Fact
            label="UP SINCE"
            value={snapshot?.collector.up_since || '—'}
          />
          <Fact
            label="COLLECTED"
            value={snapshot ? `${snapshot.collector.collect_count} cycles` : '—'}
          />
        </Card>
      </div>
      <StorageDistribution snapshot={snapshot} />
      <section className="grid gap-4">
        <div className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-3 text-label tracking-[0.12em] text-text-subtle">
            <ChartLineUp size={17} />
            RESOURCE STREAM
          </div>
          <span className="font-technical text-log text-text-tertiary">
            {ownedResources.length} resources
          </span>
        </div>
        {ownedResources.length ? (
          <Card className="grid gap-2 rounded-2xl border-border-subtle bg-surface-1 p-3">
            {ownedResources.slice(0, 100).map((resource) => (
              <ResourceRow
                key={resource.id}
                resource={resource}
              />
            ))}
          </Card>
        ) : (
          <EmptyState
            title="Waiting for Aether resources"
            description="Services created in Aether will appear here when the live collector reports them."
            icon={<ActivityIcon size={28} />}
          />
        )}
      </section>
    </div>
  )
}

function MonitoringSkeleton() {
  return (
    <div
      className="grid gap-4"
      aria-label="Loading monitoring data"
      aria-busy="true"
    >
      <Skeleton className="h-32 rounded-2xl" />
      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(18rem,0.45fr)]">
        <Skeleton className="h-80 rounded-2xl" />
        <Skeleton className="h-80 rounded-2xl" />
      </div>
      <Skeleton className="h-56 rounded-2xl" />
      <div className="grid gap-3">
        <Skeleton className="h-6 w-48" />
        <Skeleton className="h-64 rounded-2xl" />
      </div>
    </div>
  )
}

function MonitoringHeader({ connected }: { connected: boolean }) {
  return (
    <header className="grid gap-3">
      <div className="flex items-center gap-3 text-label tracking-[0.16em] text-text-subtle">
        <ActivityIcon size={17} />
        OBSERVABILITY / LIVE
        <Marker tone={connected ? 'success' : 'warning'} />
        {connected ? 'STREAMING' : 'RECONNECTING'}
      </div>
      <Typography
        as="h1"
        role="page-title"
      >
        Telemetry field
      </Typography>
      <Typography
        className="max-w-2xl"
        role="supporting"
      >
        Live resource pressure with a clear separation between host, platform and user
        load.
      </Typography>
    </header>
  )
}

function TelemetryCell({
  detail,
  icon,
  label,
  tone,
  value,
}: {
  detail: string
  icon: React.ReactNode
  label: string
  tone: 'neutral' | 'accent' | 'success' | 'warning'
  value: string
}) {
  return (
    <div className="grid gap-2 md:px-5 md:first:pl-1 md:not-first:border-l md:not-first:border-border-subtle">
      <div className="flex items-center gap-2">
        <Marker tone={tone} />
        {icon}
        <span className="text-label text-text-subtle">{label}</span>
      </div>
      <span className="font-technical text-metric text-text-primary">{value}</span>
      <span className="truncate text-supporting text-text-secondary">{detail}</span>
    </div>
  )
}

function AggregateLine({
  label,
  aggregate,
}: {
  label: string
  aggregate?: MonitoringAggregate
}) {
  return (
    <div className="grid gap-3 rounded-xl bg-surface-2 px-3 py-4 sm:grid-cols-[minmax(0,1fr)_auto_minmax(8rem,0.7fr)] sm:items-center">
      <span className="text-supporting text-text-secondary">{label}</span>
      <span className="grid justify-items-end gap-1 font-technical text-code text-text-primary">
        <span>CPU {aggregate?.cpu_of_host.toFixed(1) ?? '—'}%</span>
        <span className="text-log text-text-secondary">
          RAM {formatBytes(aggregate?.mem_usage ?? 0)}
        </span>
      </span>
      <Progress
        label={`${label} CPU`}
        value={aggregate?.cpu_of_host ?? 0}
      />
    </div>
  )
}

function StorageDistribution({
  snapshot,
}: {
  snapshot: {
    host: { disk_used: number; disk_total: number }
    aether: MonitoringAggregate
    user: MonitoringAggregate
  } | null
}) {
  const totalUsed = snapshot?.host.disk_used ?? 0
  const totalCapacity = snapshot?.host.disk_total ?? 0
  return (
    <Card className="grid gap-5 rounded-2xl border-border-subtle bg-surface-1 p-5">
      <div className="flex items-end justify-between gap-4">
        <div className="grid gap-1">
          <span className="text-label tracking-[0.12em] text-text-subtle">
            STORAGE DISTRIBUTION
          </span>
          <Typography
            as="h2"
            role="section-title"
          >
            Disk usage
          </Typography>
        </div>
        <span className="font-technical text-log text-text-secondary">
          {snapshot
            ? `${formatBytes(totalUsed)} of ${formatBytes(totalCapacity)}`
            : 'Waiting for stream'}
        </span>
      </div>
      <div className="grid gap-3 md:grid-cols-3">
        <StorageLine
          label="Aether"
          value={snapshot?.aether.storage_usage}
          total={totalCapacity}
        />
        <StorageLine
          label="User workloads"
          value={snapshot?.user.storage_usage}
          total={totalCapacity}
        />
        <StorageLine
          label="Host in use"
          value={snapshot?.host.disk_used}
          total={totalCapacity}
        />
      </div>
      <span className="text-supporting text-text-tertiary">
        Storage values are live runtime usage; Aether and user values may not sum
        exactly to host usage.
      </span>
    </Card>
  )
}

function StorageLine({
  label,
  value,
  total,
}: {
  label: string
  value?: number
  total: number
}) {
  const amount = value ?? 0
  const percentage = total > 0 ? Math.min(100, (amount / total) * 100) : 0
  return (
    <div className="grid gap-2 rounded-xl bg-surface-2 p-4">
      <div className="flex items-center justify-between gap-3">
        <span className="text-supporting text-text-secondary">{label}</span>
        <span className="font-technical text-code text-text-primary">
          {formatBytes(amount)}
        </span>
      </div>
      <Progress
        label={`${label} storage usage`}
        value={percentage}
      />
      <span className="font-technical text-log text-text-tertiary">
        {percentage.toFixed(1)}% of host capacity
      </span>
    </div>
  )
}

function ResourceRow({ resource }: { resource: MonitoringResource }) {
  return (
    <div className="grid gap-3 rounded-xl bg-surface-2 p-4 md:grid-cols-[minmax(0,1fr)_auto_auto] md:items-center">
      <div className="grid min-w-0 gap-1">
        <span className="truncate text-supporting text-text-primary">
          {resource.name}
        </span>
        <span className="truncate font-technical text-log text-text-subtle">
          {resource.owner.toUpperCase()} · {resource.service_type} ·{' '}
          {resource.project_id}
        </span>
      </div>
      <div className="flex items-center gap-3 font-technical text-log text-text-secondary">
        <span>CPU {resource.cpu_percent.toFixed(1)}%</span>
        <span>MEM {resource.mem_percent.toFixed(1)}%</span>
      </div>
      <RuntimeStatus
        status={resource.active ? 'healthy' : 'stopped'}
        label={resource.state}
      />
    </div>
  )
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid gap-1">
      <span className="text-label tracking-[0.1em] text-text-subtle">{label}</span>
      <span className="break-words font-technical text-code text-text-secondary">
        {value}
      </span>
    </div>
  )
}

function formatBytes(bytes: number) {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'
  if (bytes < 1024) return `${bytes.toFixed(0)} B`
  if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(1)} KiB`
  if (bytes < 1024 ** 3) return `${(bytes / 1024 ** 2).toFixed(1)} MiB`
  if (bytes < 1024 ** 4) return `${(bytes / 1024 ** 3).toFixed(1)} GiB`
  return `${(bytes / 1024 ** 4).toFixed(1)} TiB`
}

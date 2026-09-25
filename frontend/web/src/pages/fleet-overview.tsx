import {
  Button,
  Card,
  EmptyState,
  Marker,
  RuntimeStatus,
  Skeleton,
  Typography,
} from '@aether/elisyum-ds'
import {
  ArrowRight,
  ArrowsClockwise,
  CheckCircle,
  CircleNotch,
  CloudWarning,
  Cpu,
  Database,
  GlobeHemisphereWest,
  HardDrives,
  Memory,
  RocketLaunch,
  WarningCircle,
} from '@phosphor-icons/react'
import type { ReactNode } from 'react'
import type { Project, ServiceSummary } from '../api/types'
import { useMonitoring } from '../hooks/use-monitoring'
import { useProjects } from '../hooks/use-projects'
import { useServices } from '../hooks/use-service-details'

function runtimeStatus(status: string) {
  if (status === 'running') return 'healthy' as const
  if (status === 'pending' || status === 'deploying') return 'deploying' as const
  if (status === 'degraded') return 'degraded' as const
  if (status === 'failed') return 'failed' as const
  if (status === 'stopped') return 'stopped' as const
  return 'unknown' as const
}

function operationalState(projects: Project[], services: ServiceSummary[]) {
  const degraded = services.filter(
    (service) => service.status === 'degraded' || service.status === 'failed',
  ).length
  const deploying = services.filter(
    (service) => service.status === 'pending' || service.status === 'deploying',
  ).length
  const running = services.filter((service) => service.status === 'running').length

  if (projects.length === 0 && services.length === 0)
    return {
      kind: 'unconfigured',
      title: 'No projects yet',
      detail: 'Create a project to establish your first operational boundary.',
      icon: <GlobeHemisphereWest size={21} />,
      tone: 'neutral' as const,
    }
  if (degraded > 0)
    return {
      kind: 'degraded',
      title: `${degraded} ${degraded === 1 ? 'service needs' : 'services need'} attention`,
      detail: 'One or more services are degraded or have failed.',
      icon: <WarningCircle size={21} />,
      tone: 'warning' as const,
    }
  if (deploying > 0)
    return {
      kind: 'deploying',
      title: 'Services are deploying',
      detail: `${deploying} service${deploying === 1 ? ' is' : 's are'} being deployed.`,
      icon: <CircleNotch size={21} />,
      tone: 'accent' as const,
    }
  if (running > 0)
    return {
      kind: 'operational',
      title: 'Services are running',
      detail: `${running} of ${services.length} services currently running.`,
      icon: <CheckCircle size={21} />,
      tone: 'success' as const,
    }
  if (services.length === 0)
    return {
      kind: 'ready',
      title: 'Projects are ready',
      detail: 'There are no services in this organization yet.',
      icon: <RocketLaunch size={21} />,
      tone: 'neutral' as const,
    }
  if (services.every((service) => service.status === 'stopped'))
    return {
      kind: 'stopped',
      title: 'Services are stopped',
      detail: `${services.length} service${services.length === 1 ? ' is' : 's are'} currently stopped.`,
      icon: <WarningCircle size={21} />,
      tone: 'neutral' as const,
    }
  return {
    kind: 'unknown',
    title: 'Service status unavailable',
    detail: 'The service inventory does not contain a recognized runtime state.',
    icon: <CloudWarning size={21} />,
    tone: 'neutral' as const,
  }
}

function formatBytes(bytes: number) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KiB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MiB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GiB`
}

export function FleetOverview({ onNavigate }: { onNavigate: (path: string) => void }) {
  const projectsQuery = useProjects()
  const servicesQuery = useServices()
  const monitoring = useMonitoring()
  const projects = projectsQuery.data ?? []
  const services = servicesQuery.data ?? []
  const isLoading = projectsQuery.isLoading || servicesQuery.isLoading
  const isError = projectsQuery.isError || servicesQuery.isError
  const snapshot = monitoring.snapshot
  const state = operationalState(projects, services)
  const serviceCount = services.length
  const runningCount = services.filter((service) => service.status === 'running').length
  const statusTone = isError ? 'warning' : state.tone

  return (
    <div className="grid w-full max-w-none content-start gap-6 xl:gap-8">
      <header className="flex flex-wrap items-end justify-between gap-5">
        <div className="grid gap-2">
          <div className="flex flex-wrap items-center gap-2 text-label tracking-[0.14em] text-text-subtle">
            <span>CONTROL PLANE</span>
            <span aria-hidden="true">/</span>
            <span>ORGANIZATION</span>
            <span className="ml-2 inline-flex items-center gap-2 tracking-normal">
              <Marker tone={monitoring.connected ? 'success' : 'warning'} />
              {monitoring.connected ? 'Monitoring live' : 'Monitoring reconnecting'}
            </span>
          </div>
          <Typography
            as="h1"
            role="page-title"
          >
            Infrastructure overview
          </Typography>
          <Typography
            className="max-w-2xl"
            role="supporting"
          >
            Operational status, runtime capacity and projects across your organization.
          </Typography>
        </div>
        <Button onClick={() => onNavigate('/services/new')}>
          <RocketLaunch size={17} />
          Deploy service
        </Button>
      </header>

      <section aria-label="Operational status">
        <Card className="grid gap-5 rounded-2xl border-border-subtle bg-surface-1 p-4 sm:p-5 xl:p-6">
          {isLoading ? (
            <OperationalSkeleton />
          ) : isError ? (
            <div className="flex flex-wrap items-center justify-between gap-4">
              <div className="flex items-start gap-3">
                <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-surface-2 text-warning">
                  <WarningCircle size={21} />
                </span>
                <div className="grid gap-1">
                  <Typography
                    as="h2"
                    role="section-title"
                  >
                    Operational summary unavailable
                  </Typography>
                  <Typography role="supporting">
                    Projects or services could not be loaded from the control plane.
                  </Typography>
                </div>
              </div>
              <Button
                tone="neutral"
                onClick={() => {
                  void Promise.all([projectsQuery.refetch(), servicesQuery.refetch()])
                }}
              >
                <ArrowsClockwise size={17} />
                Retry
              </Button>
            </div>
          ) : (
            <>
              <div className="flex flex-wrap items-start justify-between gap-4">
                <div className="flex min-w-0 items-start gap-3">
                  <span
                    className={`grid size-11 shrink-0 place-items-center rounded-xl ${statusSurface(statusTone)}`}
                  >
                    {state.icon}
                  </span>
                  <div className="grid min-w-0 gap-1">
                    <span className="text-label tracking-[0.12em] text-text-subtle">
                      OPERATIONAL STATUS
                    </span>
                    <Typography
                      as="h2"
                      role="section-title"
                    >
                      {state.title}
                    </Typography>
                    <Typography role="supporting">{state.detail}</Typography>
                  </div>
                </div>
                <span className="inline-flex items-center gap-2 rounded-full border border-border-subtle bg-surface-2 px-3 py-1.5 text-supporting text-text-secondary">
                  <Marker tone={statusTone} />
                  {statusLabel(state.kind)}
                </span>
              </div>
              <div className="grid gap-4 border-t border-border-subtle pt-4 sm:grid-cols-3 sm:gap-0">
                <SummaryMetric
                  label="SERVICES"
                  value={String(serviceCount)}
                  detail="Registered in projects"
                  icon={<Database size={17} />}
                />
                <SummaryMetric
                  label="RUNNING"
                  value={String(runningCount)}
                  detail="Current service state"
                  icon={<RocketLaunch size={17} />}
                />
                <SummaryMetric
                  label="PROJECTS"
                  value={String(projects.length)}
                  detail="In this organization"
                  icon={<GlobeHemisphereWest size={17} />}
                />
              </div>
              {projects.length === 0 && serviceCount === 0 ? (
                <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl bg-surface-2 p-4">
                  <Typography role="supporting">
                    Your workspace is ready for its first project.
                  </Typography>
                  <Button onClick={() => onNavigate('/projects/new')}>
                    Create project <ArrowRight size={16} />
                  </Button>
                </div>
              ) : serviceCount === 0 ? (
                <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl bg-surface-2 p-4">
                  <Typography role="supporting">
                    Add a service to begin deploying workloads.
                  </Typography>
                  <Button onClick={() => onNavigate('/services/new')}>
                    Create service <ArrowRight size={16} />
                  </Button>
                </div>
              ) : null}
            </>
          )}
        </Card>
      </section>

      <div className="grid min-w-0 gap-5 xl:grid-cols-[minmax(0,1.25fr)_minmax(19rem,0.75fr)] xl:gap-6">
        <section aria-label="Projects">
          <Card className="grid content-start gap-4 rounded-2xl border-border-subtle bg-surface-1 p-4 sm:p-5">
            <div className="flex items-end justify-between gap-4">
              <div className="grid gap-1">
                <span className="text-label tracking-[0.12em] text-text-subtle">
                  ORGANIZATION SCOPE
                </span>
                <Typography
                  as="h2"
                  role="section-title"
                >
                  Projects
                </Typography>
              </div>
              <button
                className="inline-flex items-center gap-2 rounded-lg px-2 py-1 text-supporting text-action-strong transition-colors hover:bg-action-soft hover:text-text-primary"
                onClick={() => onNavigate('/projects')}
                type="button"
              >
                All projects <ArrowRight size={16} />
              </button>
            </div>
            {isLoading ? (
              <ProjectSkeleton />
            ) : isError ? (
              <div className="rounded-xl bg-surface-2 p-4 text-supporting text-text-secondary">
                Projects are unavailable until the control plane reconnects.
              </div>
            ) : projects.length > 0 ? (
              <div className="grid content-start gap-2">
                {projects.map((project) => (
                  <ProjectRow
                    key={project.id}
                    project={project}
                    serviceCount={
                      services.filter((service) => service.project_id === project.id)
                        .length
                    }
                    onOpen={() => onNavigate(`/projects/${project.id}`)}
                  />
                ))}
              </div>
            ) : (
              <EmptyState
                title="No projects"
                description="Create a project to establish the first operational boundary."
                action={
                  <Button onClick={() => onNavigate('/projects/new')}>
                    Create project
                  </Button>
                }
              />
            )}
          </Card>
        </section>

        <section aria-label="Resource utilization">
          <Card className="grid content-start gap-4 rounded-2xl border-border-subtle bg-surface-1 p-4 sm:p-5">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div className="grid gap-1">
                <span className="text-label tracking-[0.12em] text-text-subtle">
                  HOST
                </span>
                <Typography
                  as="h2"
                  role="section-title"
                >
                  Resource utilization
                </Typography>
              </div>
              <span className="inline-flex items-center gap-2 text-log text-text-secondary">
                <Marker tone={monitoring.connected ? 'success' : 'warning'} />
                {monitoring.connected ? 'Live' : 'Reconnecting'}
              </span>
            </div>
            {!snapshot ? (
              monitoring.error ? (
                <div className="grid gap-3 rounded-xl bg-surface-2 p-4">
                  <div className="flex items-center gap-2 text-text-secondary">
                    <CloudWarning size={18} />
                    <span className="text-supporting">Host telemetry unavailable</span>
                  </div>
                  <Typography role="supporting">
                    The monitoring stream has not returned a host snapshot.
                  </Typography>
                </div>
              ) : (
                <ResourceSkeleton />
              )
            ) : (
              <>
                <div className="grid gap-2 sm:grid-cols-3">
                  <ResourceMetric
                    label="CPU"
                    value={`${snapshot.host.cpu_percent.toFixed(1)}%`}
                    icon={<Cpu size={17} />}
                  />
                  <ResourceMetric
                    label="Memory"
                    value={`${formatBytes(snapshot.host.mem_used)} / ${formatBytes(snapshot.host.mem_total)}`}
                    icon={<Memory size={17} />}
                  />
                  <ResourceMetric
                    label="Storage"
                    value={`${formatBytes(snapshot.host.disk_used)} / ${formatBytes(snapshot.host.disk_total)}`}
                    icon={<HardDrives size={17} />}
                  />
                </div>
              </>
            )}
          </Card>
        </section>
      </div>
    </div>
  )
}

function statusSurface(tone: 'neutral' | 'accent' | 'success' | 'warning') {
  if (tone === 'success') return 'bg-success/15 text-success'
  if (tone === 'warning') return 'bg-warning/15 text-warning'
  if (tone === 'accent') return 'bg-action-soft text-action-strong'
  return 'bg-surface-2 text-text-secondary'
}

function statusLabel(kind: string) {
  if (kind === 'operational') return 'Running'
  if (kind === 'degraded') return 'Attention required'
  if (kind === 'deploying') return 'In progress'
  if (kind === 'unconfigured') return 'Not configured'
  if (kind === 'ready') return 'Ready for services'
  if (kind === 'stopped') return 'Stopped'
  return 'Status unknown'
}

function SummaryMetric({
  label,
  value,
  detail,
  icon,
}: {
  label: string
  value: string
  detail: string
  icon: ReactNode
}) {
  return (
    <div className="grid gap-1.5 sm:px-4 sm:first:pl-0 sm:not-first:border-l sm:not-first:border-border-subtle">
      <span className="flex items-center gap-2 text-label tracking-[0.1em] text-text-subtle">
        {icon}
        {label}
      </span>
      <span className="font-technical text-metric text-text-primary">{value}</span>
      <span className="text-log text-text-secondary">{detail}</span>
    </div>
  )
}

function ProjectRow({
  project,
  serviceCount,
  onOpen,
}: {
  project: Project
  serviceCount: number
  onOpen: () => void
}) {
  return (
    <button
      aria-label={`Open ${project.name}`}
      className="group flex min-w-0 items-center gap-3 rounded-xl border border-transparent bg-surface-2 px-3 py-3 text-left transition-colors hover:border-border-default hover:bg-surface-3"
      onClick={onOpen}
      type="button"
    >
      <span className="grid size-9 shrink-0 place-items-center rounded-lg bg-surface-1 text-text-tertiary">
        <GlobeHemisphereWest size={17} />
      </span>
      <span className="grid min-w-0 flex-1 gap-1">
        <span className="truncate text-supporting text-text-primary">
          {project.name}
        </span>
        <span className="truncate text-log text-text-subtle">
          {serviceCount} {serviceCount === 1 ? 'service' : 'services'}
        </span>
      </span>
      <span className="shrink-0 text-log text-text-tertiary">Open</span>
      <ArrowRight
        aria-hidden="true"
        className="shrink-0 text-text-tertiary transition-transform group-hover:translate-x-0.5"
        size={17}
      />
    </button>
  )
}

function ResourceMetric({
  label,
  value,
  icon,
}: {
  label: string
  value: string
  icon: ReactNode
}) {
  return (
    <div className="grid min-w-0 gap-2 rounded-xl bg-surface-2 p-3">
      <span className="flex items-center gap-2 text-supporting text-text-secondary">
        {icon}
        {label}
      </span>
      <span className="truncate font-technical text-code text-text-primary">
        {value}
      </span>
    </div>
  )
}

function OperationalSkeleton() {
  return (
    <div className="grid gap-5">
      <div className="flex items-center gap-3">
        <Skeleton className="size-11 rounded-xl" />
        <div className="grid flex-1 gap-2">
          <Skeleton className="h-3 w-32" />
          <Skeleton className="h-6 w-56" />
          <Skeleton className="h-4 w-72 max-w-full" />
        </div>
      </div>
      <div className="grid gap-4 border-t border-border-subtle pt-4 sm:grid-cols-3">
        <Skeleton className="h-16" />
        <Skeleton className="h-16" />
        <Skeleton className="h-16" />
      </div>
    </div>
  )
}

function ProjectSkeleton() {
  return (
    <div className="grid gap-2">
      {Array.from({ length: 3 }, (_, index) => (
        <Skeleton
          className="h-[3.75rem] rounded-xl"
          key={index}
        />
      ))}
    </div>
  )
}

function ResourceSkeleton() {
  return (
    <div className="grid gap-2 sm:grid-cols-3">
      <Skeleton className="h-[4.5rem] rounded-xl" />
      <Skeleton className="h-[4.5rem] rounded-xl" />
      <Skeleton className="h-[4.5rem] rounded-xl" />
    </div>
  )
}

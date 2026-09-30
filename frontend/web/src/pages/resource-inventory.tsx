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
  GlobeHemisphereWest,
  Package,
  Plus,
  Stack,
} from '@phosphor-icons/react'
import { type ReactNode, useState } from 'react'
import type { Project, ServiceSummary } from '../api/types'
import { useProjects } from '../hooks/use-projects'
import { useServices } from '../hooks/use-service-details'

export function ResourceInventory({
  kind,
  onNavigate,
}: {
  kind: 'projects' | 'services'
  onNavigate: (path: string) => void
  resourcePath?: string
}) {
  return kind === 'projects' ? (
    <ProjectInventory onNavigate={onNavigate} />
  ) : (
    <ServiceInventory onNavigate={onNavigate} />
  )
}

function ProjectInventory({ onNavigate }: { onNavigate: (path: string) => void }) {
  const query = useProjects()
  const projects = query.data ?? []
  const [selectedId, setSelectedId] = useState<string>()
  const selected = projects.find((project) => project.id === selectedId) ?? projects[0]
  const hasProjects = projects.length > 0
  const hasMultipleProjects = projects.length >= 2

  return (
    <div className="grid gap-7">
      <InventoryHeader
        eyebrow="PROJECT SYSTEM / BOUNDARIES"
        title="Project field"
        description="Operational boundaries for environments, delivery ownership and team scope."
        action={
          <Button onClick={() => onNavigate('/projects/new')}>
            <Plus size={17} />
            Create project
          </Button>
        }
      />
      {query.isError ? (
        <EmptyState
          title="Project field unavailable"
          description="The control plane did not return the project collection."
          action={
            <Button
              tone="neutral"
              onClick={() => query.refetch()}
            >
              Retry request
            </Button>
          }
        />
      ) : null}
      {query.isLoading ? <InventorySkeleton /> : null}
      {!query.isLoading && !query.isError && !hasProjects ? (
        <EmptyState
          className="max-w-3xl justify-self-start px-8 py-14 sm:px-12"
          title="No projects yet"
          description="Projects group your services, environments and deployments."
          icon={<Stack size={28} />}
          action={
            <Button onClick={() => onNavigate('/projects/new')}>
              <Plus size={17} />
              Create project
            </Button>
          }
        />
      ) : null}
      {!query.isLoading && !query.isError && projects.length === 1 ? (
        <SingleProjectOverview
          project={projects[0]}
          onNavigate={onNavigate}
        />
      ) : null}
      {!query.isLoading && !query.isError && hasMultipleProjects ? (
        <div className="grid gap-5 xl:grid-cols-[minmax(0,1.2fr)_minmax(20rem,0.8fr)]">
          <section className="grid min-w-0 content-start gap-4">
            <InventoryLead
              icon={<Stack size={18} />}
              label={`${projects.length} PROJECTS IN SCOPE`}
              meta="SORT / RECENT"
            />
            <Card className="grid gap-2 rounded-2xl border-border-subtle bg-surface-1 p-3">
              {projects.map((project) => (
                <ProjectLine
                  key={project.id}
                  project={project}
                  selected={project.id === selected?.id}
                  onSelect={() => setSelectedId(project.id)}
                />
              ))}
            </Card>
          </section>
          <ProjectInspector
            project={selected}
            onNavigate={onNavigate}
          />
        </div>
      ) : null}
    </div>
  )
}

function ServiceInventory({ onNavigate }: { onNavigate: (path: string) => void }) {
  const query = useServices()
  const projectsQuery = useProjects()
  const services = query.data ?? []
  const [selectedId, setSelectedId] = useState<string>()
  const selected = services.find((service) => service.id === selectedId) ?? services[0]
  const projectName = (projectId: string) =>
    projectsQuery.data?.find((project) => project.id === projectId)?.name ?? projectId

  return (
    <div className="grid gap-7">
      <InventoryHeader
        eyebrow="SERVICE SYSTEM / RUNTIME"
        title="Service field"
        description="Runtime resources with lifecycle, source and the next safe operation visible at a glance."
        action={
          <Button onClick={() => onNavigate('/services/new')}>
            <Plus size={17} />
            Create service
          </Button>
        }
      />
      <div className="grid gap-5 xl:grid-cols-[minmax(0,1.2fr)_minmax(20rem,0.8fr)]">
        <section className="grid min-w-0 content-start gap-4">
          <InventoryLead
            icon={<Package size={18} />}
            label={`${services.length} SERVICES IN SCOPE`}
            meta="SORT / RECENT"
          />
          {query.isError ? (
            <EmptyState
              title="Service field unavailable"
              description="The control plane did not return the service collection."
              action={
                <Button
                  tone="neutral"
                  onClick={() => query.refetch()}
                >
                  Retry request
                </Button>
              }
            />
          ) : null}
          {query.isLoading ? <InventorySkeleton /> : null}
          {!(query.isLoading || query.isError || services.length) ? (
            <EmptyState
              title="No services in scope"
              description="Create a service to establish the first runtime surface."
              icon={<Package size={28} />}
              action={
                <Button onClick={() => onNavigate('/services/new')}>
                  <Plus size={17} />
                  Create service
                </Button>
              }
            />
          ) : null}
          {services.length ? (
            <Card className="grid gap-2 rounded-2xl border-border-subtle bg-surface-1 p-3">
              {services.map((service) => (
                <ServiceLine
                  key={service.id}
                  projectName={projectName(service.project_id)}
                  service={service}
                  selected={service.id === selected?.id}
                  onSelect={() => setSelectedId(service.id)}
                />
              ))}
            </Card>
          ) : null}
        </section>
        <ServiceInspector
          projectName={selected ? projectName(selected.project_id) : undefined}
          service={selected}
          onNavigate={onNavigate}
        />
      </div>
    </div>
  )
}

function InventoryHeader({
  eyebrow,
  title,
  description,
  action,
}: {
  eyebrow: string
  title: string
  description: string
  action: ReactNode
}) {
  return (
    <header className="flex flex-wrap items-end justify-between gap-5">
      <div className="grid gap-3">
        <span className="text-label tracking-[0.16em] text-text-subtle">{eyebrow}</span>
        <Typography
          as="h1"
          role="page-title"
        >
          {title}
        </Typography>
        <Typography
          className="max-w-2xl"
          role="supporting"
        >
          {description}
        </Typography>
      </div>
      {action}
    </header>
  )
}

function InventoryLead({
  icon,
  label,
  meta,
}: {
  icon: ReactNode
  label: string
  meta?: string
}) {
  return (
    <div className="flex items-center justify-between gap-4">
      <div className="flex items-center gap-3 text-label tracking-[0.12em] text-text-subtle">
        <Marker tone="accent" />
        {icon}
        {label}
      </div>
      {meta ? <span className="font-technical text-log text-text-subtle">{meta}</span> : null}
    </div>
  )
}

function ProjectLine({
  project,
  selected,
  onSelect,
}: {
  project: Project
  selected: boolean
  onSelect: () => void
}) {
  return (
    <button
      aria-pressed={selected}
      className={`group relative grid min-h-[5.25rem] grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-4 rounded-xl border px-4 py-3 text-left transition-[background-color,border-color,transform] duration-[var(--ely-duration-fast)] focus-visible:outline-2 focus-visible:outline-focus ${selected ? 'border-action/65 bg-action/10 before:absolute before:inset-y-3 before:left-0 before:w-0.5 before:rounded-full before:bg-action' : 'border-transparent bg-surface-2 hover:-translate-y-px hover:border-border-default hover:bg-surface-3'}`}
      onClick={onSelect}
      type="button"
    >
      <span className={`grid size-10 place-items-center rounded-xl bg-surface-1 ${selected ? 'text-action-strong' : 'text-text-tertiary'}`}>
        <GlobeHemisphereWest size={19} />
      </span>
      <span className="grid min-w-0 gap-1">
        <span className="truncate text-supporting text-text-primary">
          {project.name}
        </span>
        <span className="truncate font-technical text-log text-text-subtle">
          {project.slug || project.id}
        </span>
      </span>
      <ArrowRight
        className="text-text-tertiary transition-transform group-hover:translate-x-0.5"
        size={18}
      />
    </button>
  )
}

function SingleProjectOverview({
  project,
  onNavigate,
}: {
  project: Project
  onNavigate: (path: string) => void
}) {
  return (
    <section className="grid max-w-4xl gap-4">
      <InventoryLead
        icon={<Stack size={18} />}
        label="1 PROJECT IN SCOPE"
      />
      <Card className="grid gap-7 rounded-2xl border-border-subtle bg-surface-1 p-5 sm:p-7">
        <div className="grid gap-5 sm:grid-cols-[auto_minmax(0,1fr)_auto] sm:items-start">
          <span className="grid size-12 place-items-center rounded-xl bg-surface-2 text-text-tertiary">
            <GlobeHemisphereWest size={22} />
          </span>
          <div className="grid min-w-0 gap-1">
            <Typography
              as="h2"
              role="section-title"
            >
              {project.name}
            </Typography>
            <span className="truncate font-technical text-log text-text-subtle">
              {project.slug || project.id}
            </span>
            <p className="mt-2 text-supporting text-text-tertiary">
              {project.description || 'No description supplied'}
            </p>
          </div>
          <ArrowRight className="hidden text-text-tertiary sm:block" size={20} />
        </div>
        <div className="flex flex-wrap items-end justify-between gap-5 border-t border-border-subtle pt-5">
          <Fact
            label="CREATED"
            value={formatProjectDate(project.created_at)}
          />
          <Button onClick={() => onNavigate(`/projects/${project.id}`)}>
            Open project <ArrowRight size={16} />
          </Button>
        </div>
      </Card>
    </section>
  )
}

function ServiceLine({
  projectName,
  service,
  selected,
  onSelect,
}: {
  projectName: string
  service: ServiceSummary
  selected: boolean
  onSelect: () => void
}) {
  const status = service.status || 'unknown'
  const source = service.spec?.source_type || service.kind
  return (
    <button
      aria-pressed={selected}
      className={`relative grid min-h-[5.25rem] grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-4 rounded-xl border px-4 py-3 text-left transition-[background-color,border-color,transform] duration-[var(--ely-duration-fast)] focus-visible:outline-2 focus-visible:outline-focus ${selected ? 'border-action/65 bg-action/10 before:absolute before:inset-y-3 before:left-0 before:w-0.5 before:rounded-full before:bg-action' : 'border-transparent bg-surface-2 hover:-translate-y-px hover:border-border-default hover:bg-surface-3'}`}
      onClick={onSelect}
      type="button"
    >
      <span
        className={`grid size-10 place-items-center rounded-xl bg-surface-1 ${selected ? 'text-action-strong' : 'text-text-tertiary'}`}
      >
        <Package size={19} />
      </span>
      <span className="grid min-w-0 gap-1">
        <span className="truncate text-supporting text-text-primary">
          {service.name}
        </span>
        <span className="truncate font-technical text-log text-text-subtle">
          {source.toUpperCase()} / {projectName}
        </span>
      </span>
      <RuntimeStatus
        className="hidden sm:flex"
        label={status}
        status={toRuntimeStatus(status)}
      />
    </button>
  )
}

function ProjectInspector({
  project,
  onNavigate,
}: {
  project?: Project
  onNavigate: (path: string) => void
}) {
  return (
    <aside className="grid h-full content-start gap-6 rounded-2xl border border-border-subtle bg-surface-1 p-5">
      <div className="grid gap-2">
        <span className="text-label tracking-[0.12em] text-text-subtle">
          PROJECT INSPECTOR
        </span>
        <Typography
          as="h2"
          role="section-title"
        >
          {project?.name ?? 'Awaiting project'}
        </Typography>
        <Typography role="supporting">
          {project
            ? 'Identity and lifecycle context remain attached to this project boundary.'
            : 'Select a project from the field to inspect it.'}
        </Typography>
      </div>
      {project ? (
        <div className="grid gap-5 rounded-xl bg-surface-2 p-4">
          <div className="grid gap-4">
            <Fact
              label="SLUG"
              value={project.slug || 'UNSET'}
            />
            <Fact
              label="DESCRIPTION"
              value={project.description || 'No description supplied'}
            />
            <Fact label="CREATED" value={formatProjectDate(project.created_at)} />
          </div>
          <Button onClick={() => onNavigate(`/projects/${project.id}`)}>
            Open project <ArrowRight size={16} />
          </Button>
        </div>
      ) : (
        <div className="rounded-xl bg-surface-2 p-4 text-supporting text-text-tertiary">
          No project has been returned.
        </div>
      )}
    </aside>
  )
}

function ServiceInspector({
  projectName,
  service,
  onNavigate,
}: {
  projectName?: string
  service?: ServiceSummary
  onNavigate: (path: string) => void
}) {
  const source = service?.spec?.source_type || service?.kind || 'unknown'
  return (
    <aside className="grid h-full content-start gap-6 rounded-2xl border border-border-subtle bg-surface-1 p-5">
      <div className="grid gap-2">
        <span className="text-label tracking-[0.12em] text-text-subtle">
          SERVICE INSPECTOR
        </span>
        <Typography
          as="h2"
          role="section-title"
        >
          {service?.name ?? 'Awaiting service'}
        </Typography>
        <Typography role="supporting">
          {service
            ? 'Runtime identity and source context remain attached to this service.'
            : 'Select a service from the field to inspect it.'}
        </Typography>
      </div>
      {service ? (
        <div className="grid gap-5 rounded-xl bg-surface-2 p-4">
          <div className="grid gap-4">
            <Fact
              label="SOURCE"
              value={source}
            />
            <Fact
              label="PROJECT"
              value={projectName ?? service.project_id}
            />
            <Fact
              label="UPDATED"
              value={service.updated_at || '—'}
            />
          </div>
          <Button onClick={() => onNavigate(`/services/${service.id}?from=services`)}>
            Open service <ArrowRight size={16} />
          </Button>
        </div>
      ) : (
        <div className="rounded-xl bg-surface-2 p-4 text-supporting text-text-tertiary">
          No service has been returned.
        </div>
      )}
    </aside>
  )
}

function InventorySkeleton() {
  return (
    <div className="grid gap-2">
      {Array.from({ length: 6 }, (_, index) => (
        <Skeleton
          className="h-20 rounded-xl"
          key={index}
        />
      ))}
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

function formatProjectDate(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return 'Unknown'
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)
}

function toRuntimeStatus(
  status?: string,
): 'healthy' | 'deploying' | 'degraded' | 'failed' | 'stopped' | 'unknown' {
  if (status === 'ready' || status === 'running') return 'healthy'
  if (status === 'building' || status === 'starting') return 'deploying'
  if (status === 'failed' || status === 'error' || status === 'exited') return 'failed'
  if (status === 'stopped') return 'stopped'
  return 'unknown'
}

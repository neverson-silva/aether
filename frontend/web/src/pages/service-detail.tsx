import {
  ArrowLeft,
  CheckCircle,
  CopySimple,
  Cube,
  DotsThree,
  Eye,
  EyeSlash,
  Globe,
  MagnifyingGlass,
  PencilSimple,
  Play,
  Plus,
  Repeat,
  Stop,
  TerminalWindow,
  Trash,
  X,
} from '@phosphor-icons/react'
import { FitAddon } from '@xterm/addon-fit'
import { SearchAddon } from '@xterm/addon-search'
import { Unicode11Addon } from '@xterm/addon-unicode11'
import { WebLinksAddon } from '@xterm/addon-web-links'
import { Terminal as XTerm } from '@xterm/xterm'
import { type ReactNode, useCallback, useEffect, useRef, useState } from 'react'
import '@xterm/xterm/css/xterm.css'
import {
  AlertDialog,
  Button,
  CodeEditorLite,
  DropdownMenu,
  EmptyState,
  Field,
  Input,
  LogViewer,
  RuntimeStatus,
  Select,
  Slider,
  Switch,
  showToast,
  Tabs,
  Textarea,
  Typography,
} from '@aether/elisyum-ds'
import { useNavigate, useParams, useSearch } from '@tanstack/react-router'
import { getServer } from '../api/client'
import type { Deployment, Domain, ServiceSummary, Stats } from '../api/types'
import { useAddDomain } from '../hooks/use-add-domain'
import { useAppCompose } from '../hooks/use-app-compose'
import { useCronJobs } from '../hooks/use-cron-jobs'
import { useDeleteEnv } from '../hooks/use-delete-env'
import { useDomains } from '../hooks/use-domains'
import { usePolicy } from '../hooks/use-policy'
import { usePolicyEvents } from '../hooks/use-policy-events'
import { useRemoveDomain } from '../hooks/use-remove-domain'
import { useSavePolicy } from '../hooks/use-save-policy'
import { useServiceAction } from '../hooks/use-service-action'
import { useServiceCancelDeployment } from '../hooks/use-service-cancel-deployment'
import { useServiceConnection } from '../hooks/use-service-connection'
import { useServiceContainers } from '../hooks/use-service-containers'
import { useServiceDeploymentLog } from '../hooks/use-service-deployment-log'
import { useServiceDeployments } from '../hooks/use-service-deployments'
import { useServiceDetails } from '../hooks/use-service-details'
import { useServiceEnvironment } from '../hooks/use-service-environment'
import { useServiceStats } from '../hooks/use-service-stats'
import { useSetEnv } from '../hooks/use-set-env'
import { useSetWebhook } from '../hooks/use-set-webhook'
import {
  type ServiceSourceInput,
  selectGitHubConnection,
  useSaveServiceSource,
  useServiceSource,
  useSourceControlBranches,
  useSourceControlConnections,
  useSourceControlRepositories,
} from '../hooks/use-source-control'
import { useUpdateDomain } from '../hooks/use-update-domain'
import { useUpdateService } from '../hooks/use-update-service'
import { ServiceBackups } from './service-backups'

export function ServiceDetail() {
  const { serviceId } = useParams({ from: '/_shell/services/$serviceId' })
  const { from, projectId, tab } = useSearch({ from: '/_shell/services/$serviceId' })
  const navigate = useNavigate()
  const serviceQuery = useServiceDetails(serviceId)
  const deploymentsQuery = useServiceDeployments(serviceId)
  const cancelDeployment = useServiceCancelDeployment(serviceId)
  const statsQuery = useServiceStats(
    serviceId,
    serviceQuery.data?.capabilities.can_view_metrics ?? false,
  )
  const [selectedDeploymentId, setSelectedDeploymentId] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState(tab ?? 'overview')
  const deployment =
    deploymentsQuery.data?.find((item) => item.id === selectedDeploymentId) ??
    deploymentsQuery.data?.[0]
  const logQuery = useServiceDeploymentLog(serviceId, deployment?.id ?? null)
  const returnTo =
    from === 'project' && projectId
      ? () => navigate({ to: `/projects/${projectId}` })
      : () => navigate({ to: '/services' })
  useEffect(() => {
    setActiveTab(tab ?? 'overview')
  }, [tab])

  if (serviceQuery.isLoading) return <DetailSkeleton />
  if (!serviceQuery.data || serviceQuery.isError)
    return (
      <EmptyState
        title="Service unavailable"
        description="The control plane did not return this service record."
        action={
          <Button
            tone="neutral"
            onClick={() => serviceQuery.refetch()}
          >
            Retry request
          </Button>
        }
      />
    )

  const service = serviceQuery.data
  const runtimeContainers = service.runtime?.containers ?? []
  const displayedStatus = service.status
  const lines = (logQuery.data?.content ?? '')
    .split('\n')
    .filter(Boolean)
    .map((line, index) => ({
      id: `${deployment?.id ?? 'log'}-${index}`,
      level: 'OUT',
      message: line,
    }))
  const items = [
    {
      value: 'overview',
      label: 'Overview',
      content: (
        <div className="w-full">
          <OverviewPanel
            service={service}
            stats={statsQuery.data}
            telemetryLoading={statsQuery.isLoading}
            telemetryError={statsQuery.isError}
            deployments={deploymentsQuery.data ?? []}
            containers={runtimeContainers}
          />
        </div>
      ),
    },
    {
      value: 'deployments',
      label: 'Deployments',
      content: (
        <div className="w-full">
          <DeploymentsPanel
            deployments={deploymentsQuery.data ?? []}
            selectedId={deployment?.id}
            onSelect={setSelectedDeploymentId}
            onCancel={(deploymentId) =>
              cancelDeployment.mutate(deploymentId, {
                onSuccess: () =>
                  showToast('Deployment cancellation requested.', 'success'),
                onError: (error) =>
                  showToast(`Could not cancel deployment: ${error.message}`, 'error'),
              })
            }
            cancellingDeploymentId={cancelDeployment.variables}
            cancelPending={cancelDeployment.isPending}
            log={lines}
          />
        </div>
      ),
    },
    {
      value: 'variables',
      label: 'Variables',
      content: (
        <div className="w-full">
          <VariablesPanel serviceId={service.id} />
        </div>
      ),
    },
    ...(service.kind === 'compose' &&
    service.capabilities.can_edit_compose &&
    service.spec_id
      ? [
          {
            value: 'compose',
            label: 'Compose',
            content: <ComposePanel composeId={service.spec_id} />,
          },
        ]
      : []),
    ...(service.capabilities.can_manage_domains
      ? [
          {
            value: 'domains',
            label: 'Domains',
            content: (
              <div className="w-full max-w-6xl">
                <DomainsPanel
                  defaultPort={service.spec?.port}
                  serviceId={service.id}
                />
              </div>
            ),
          },
        ]
      : []),
    ...(service.capabilities.can_view_logs
      ? [
          {
            value: 'logs',
            label: 'Logs',
            content: <LiveLogsPanel serviceId={service.id} />,
          },
        ]
      : []),
    ...(service.capabilities.can_build || service.kind === 'compose'
      ? [
          {
            value: 'settings',
            label: 'Settings',
            content: (
              <div className="w-full max-w-6xl">
                <SettingsPanel service={service} />
              </div>
            ),
          },
        ]
      : []),
    ...(service.capabilities.can_manage_schedules
      ? [
          {
            value: 'cron',
            label: 'Schedules',
            content: (
              <div className="w-full max-w-6xl">
                <CronPanel serviceId={service.id} />
              </div>
            ),
          },
        ]
      : []),
    ...(service.capabilities.can_open_terminal
      ? [
          {
            value: 'terminal',
            label: 'Terminal',
            content: <TerminalPanel serviceId={service.id} />,
          },
        ]
      : []),
    ...(service.kind === 'database' && service.capabilities.can_manage_backups
      ? [
          {
            value: 'backup',
            label: 'Backups',
            content: (
              <div className="w-full max-w-6xl">
                <ServiceBackups
                  serviceId={service.id}
                  serviceName={service.name}
                />
              </div>
            ),
          },
        ]
      : []),
  ]

  return (
    <div className="grid gap-6">
      <header className="grid gap-4 pb-5">
        <button
          className="flex w-fit items-center gap-2 text-label text-text-tertiary transition-colors hover:text-text-primary"
          onClick={returnTo}
          type="button"
        >
          <ArrowLeft size={16} />
          {from === 'project' ? 'Project details' : 'Service field'}
        </button>
        <div className="flex flex-wrap items-end justify-between gap-6">
          <div className="grid gap-2">
            <span className="text-label tracking-[0.16em] text-text-subtle">
              SERVICE / {service.kind.toUpperCase()}
            </span>
            <div className="flex flex-wrap items-center gap-3">
              <Typography
                as="h1"
                role="page-title"
              >
                {service.name}
              </Typography>
              <RuntimeStatus
                label={displayedStatus}
                status={toRuntimeStatus(displayedStatus)}
              />
            </div>
            <p className="text-supporting text-text-tertiary">
              {serviceContext(service)}
            </p>
          </div>
          <ServiceActions
            service={service}
            deployments={deploymentsQuery.data ?? []}
            containers={runtimeContainers}
            onDeleted={returnTo}
            onDeploymentQueued={(deploymentId) => {
              setSelectedDeploymentId(deploymentId)
              setActiveTab('deployments')
            }}
            onOpenTerminal={() => setActiveTab('terminal')}
            onOpenStudio={() => {
              void navigate({
                to: '/studio/$databaseId',
                params: { databaseId: service.spec_id ?? service.id },
                search: {
                  returnTo: `${window.location.pathname}${window.location.search}${window.location.hash}`,
                },
              })
            }}
          />
        </div>
      </header>
      <Tabs
        className="gap-4"
        items={items}
        value={activeTab}
        onValueChange={(value) => {
          setActiveTab(value)
          void navigate({
            to: `/services/${serviceId}`,
            search: { from, projectId, tab: value },
          })
        }}
      />
    </div>
  )
}

function ServiceActions({
  service,
  deployments,
  containers,
  onDeleted,
  onDeploymentQueued,
  onOpenTerminal,
  onOpenStudio,
}: {
  service: ServiceSummary
  deployments: Deployment[]
  containers: NonNullable<ServiceSummary['runtime']>['containers']
  onDeleted: () => void
  onDeploymentQueued: (deploymentId: string) => void
  onOpenTerminal: () => void
  onOpenStudio: () => void
}) {
  const action = useServiceAction('start')
  const deploy = useServiceAction('deploy')
  const stop = useServiceAction('stop')
  const remove = useServiceAction('delete')
  const [deleteOpen, setDeleteOpen] = useState(false)
  const neverDeployed =
    service.status === 'pending' && deployments.length === 0 && containers.length === 0
  const starting = service.status === 'starting'
  const stopping = service.status === 'stopping'
  const transitioning = starting || stopping
  const canShowStart =
    service.status !== 'running' &&
    service.status !== 'pending' &&
    !stopping &&
    service.capabilities.can_start
  const operationPending =
    deploy.isPending || stop.isPending || action.isPending || transitioning
  const menuItems = [
    ...(service.kind === 'database'
      ? [
          {
            label: (
              <span className="flex w-full items-center gap-2">
                <Cube size={16} />
                Open Studio
              </span>
            ),
            onSelect: onOpenStudio,
          },
        ]
      : []),
    ...(service.status === 'running' || stopping
      ? [
          {
            label: (
              <span className="flex w-full items-center gap-2">
                <Stop size={16} />
                {stopping || stop.isPending ? 'Stopping…' : 'Stop service'}
              </span>
            ),
            disabled: operationPending,
            onSelect: () => stop.mutate(service.id),
          },
        ]
      : []),
    ...(canShowStart
      ? [
          {
            label: (
              <span className="flex w-full items-center gap-2">
                <Play size={16} />
                {starting || action.isPending ? 'Starting…' : 'Start service'}
              </span>
            ),
            disabled: operationPending,
            onSelect: () => action.mutate(service.id),
          },
        ]
      : []),
    {
      label: (
        <span className="flex w-full items-center gap-2">
          <Trash size={16} />
          Delete service
        </span>
      ),
      danger: true,
      disabled: operationPending,
      onSelect: () => setDeleteOpen(true),
    },
  ]
  return (
    <div className="flex flex-wrap items-center gap-2">
      {service.capabilities.can_deploy ? (
        <Button
          disabled={operationPending && !deploy.isPending}
          loading={deploy.isPending}
          onClick={() =>
            deploy.mutate(service.id, {
              onSuccess: (response) => {
                const result = response?.result
                if (
                  result &&
                  typeof result === 'object' &&
                  'deployment_id' in result &&
                  typeof result.deployment_id === 'string'
                )
                  onDeploymentQueued(result.deployment_id)
              },
            })
          }
        >
          <Repeat size={16} />
          {deploy.isPending
            ? 'Deploying…'
            : neverDeployed
              ? 'Deploy service'
              : 'Deploy'}
        </Button>
      ) : null}
      {service.capabilities.can_open_terminal ? (
        <Button
          tone="ghost"
          onClick={onOpenTerminal}
        >
          <TerminalWindow size={16} />
          Open terminal
        </Button>
      ) : null}
      <>
        <DropdownMenu
          trigger={
            <DotsThree
              size={21}
              weight="bold"
            />
          }
          triggerClassName="inline-flex min-h-9 cursor-pointer items-center justify-center rounded-lg px-2 text-text-tertiary transition-colors hover:bg-surface-2 hover:text-text-primary focus-visible:outline-2 focus-visible:outline-focus"
          items={menuItems}
        />
        <AlertDialog
          description={
            <span className="grid gap-4">
              <span>
                Delete <strong className="text-text-primary">{service.name}</strong> and
                its runtime state from this project.
              </span>
              <span className="rounded-xl border border-danger/40 bg-danger-soft/30 p-4 text-label text-danger-strong">
                <span className="block tracking-[0.12em]">DANGER ZONE</span>
                <span className="mt-1 block text-supporting text-danger-strong/90">
                  This permanently removes the service, deployments and runtime
                  connection.
                </span>
              </span>
            </span>
          }
          confirmLabel={remove.isPending ? 'Deleting…' : 'Delete service'}
          onConfirm={() => remove.mutate(service.id, { onSuccess: onDeleted })}
          onOpenChange={setDeleteOpen}
          open={deleteOpen}
          title={
            <span className="flex items-center gap-3">
              <span className="grid size-9 place-items-center rounded-lg bg-danger-soft text-danger-strong">
                <Trash size={18} />
              </span>
              <span>Delete service?</span>
            </span>
          }
        />
      </>
    </div>
  )
}

function OverviewPanel({
  service,
  stats,
  telemetryLoading,
  telemetryError,
  deployments,
  containers,
}: {
  service: ServiceSummary
  stats?: Stats
  telemetryLoading: boolean
  telemetryError: boolean
  deployments: Deployment[]
  containers: NonNullable<ServiceSummary['runtime']>['containers']
}) {
  const neverDeployed =
    service.status === 'pending' && deployments.length === 0 && containers.length === 0
  const telemetryAvailable =
    Boolean(stats?.stats) &&
    stats?.state !== 'unknown' &&
    (stats?.containers?.length ?? 0) > 0
  const latestDeployment = deployments[0]
  const configuration = [
    ...(service.spec?.build_type
      ? [{ label: 'BUILD', value: service.spec.build_type }]
      : []),
    ...(service.spec?.engine
      ? [
          {
            label: 'ENGINE',
            value: `${service.spec.engine}${service.spec.version ? ` ${service.spec.version}` : ''}`,
          },
        ]
      : []),
    ...(service.spec?.image ? [{ label: 'IMAGE', value: service.spec.image }] : []),
    ...(service.spec?.compose_file
      ? [{ label: 'COMPOSE FILE', value: service.spec.compose_file }]
      : []),
    ...(service.spec?.git_url
      ? [{ label: 'SOURCE', value: service.spec.git_url }]
      : []),
    ...(service.spec?.port
      ? [{ label: 'PORT', value: String(service.spec.port) }]
      : []),
    ...(service.spec?.cpus
      ? [{ label: 'CPU LIMIT', value: `${service.spec.cpus} vCPU` }]
      : []),
    ...(service.spec?.mem_mb
      ? [{ label: 'MEMORY LIMIT', value: `${service.spec.mem_mb} MB` }]
      : []),
    ...(service.spec?.storage_mb
      ? [{ label: 'STORAGE LIMIT', value: `${service.spec.storage_mb} MB` }]
      : []),
  ]
  return (
    <div className="grid content-start gap-5">
      <section
        aria-label="Runtime summary"
        className="grid gap-3 rounded-xl border border-border-subtle bg-surface-1 p-4"
      >
        <div className="flex flex-wrap items-center justify-between gap-x-5 gap-y-2">
          <Typography
            as="h2"
            role="section-title"
          >
            Runtime
          </Typography>
          {latestDeployment ? (
            <span className="text-label text-text-tertiary">Latest deployment</span>
          ) : (
            <span className="text-label text-text-tertiary">No deployment yet</span>
          )}
        </div>
        <div className="grid grid-cols-2 gap-3 border-t border-border-subtle pt-3 sm:grid-cols-3 sm:gap-5">
          <RuntimeMetric
            label="CPU"
            value={
              telemetryLoading
                ? 'Loading…'
                : telemetryAvailable
                  ? `${stats!.stats.cpu_percent.toFixed(1)}%`
                  : '—'
            }
            detail={
              telemetryLoading
                ? 'Collecting telemetry'
                : telemetryError
                  ? 'Telemetry request failed'
                  : telemetryAvailable
                    ? 'Current usage'
                    : 'No telemetry'
            }
          />
          <RuntimeMetric
            label="Memory"
            value={
              telemetryLoading
                ? 'Loading…'
                : telemetryAvailable
                  ? `${stats!.stats.mem_percent.toFixed(1)}%`
                  : '—'
            }
            detail={
              telemetryLoading
                ? 'Collecting telemetry'
                : telemetryError
                  ? 'Telemetry request failed'
                  : telemetryAvailable
                    ? `${formatRuntimeBytes(stats!.stats.mem_bytes)}${stats!.stats.mem_limit > 0 ? ` of ${formatRuntimeBytes(stats!.stats.mem_limit)}` : ' used'}`
                    : 'No telemetry'
            }
          />
          <RuntimeMetric
            label="Containers"
            value={String(containers.length)}
            detail="Attached to runtime"
          />
        </div>
        {latestDeployment ? (
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 border-t border-border-subtle pt-3">
            <RuntimeStatus
              label={latestDeployment.status}
              status={toRuntimeStatus(latestDeployment.status)}
            />
            <span className="text-label text-text-tertiary">
              RUN / {String(latestDeployment.number).padStart(3, '0')}
            </span>
            <span className="min-w-0 flex-1 truncate font-technical text-log text-text-secondary">
              {latestDeployment.commit ||
                latestDeployment.image_ref ||
                latestDeployment.id}
            </span>
            <time
              className="shrink-0 font-technical text-log text-text-tertiary"
              dateTime={latestDeployment.created_at}
            >
              {formatTimelineTimestamp(latestDeployment.created_at)}
            </time>
          </div>
        ) : null}
      </section>
      <section className="grid gap-3">
        <div className="flex flex-wrap items-baseline justify-between gap-2">
          <Typography
            as="h2"
            role="section-title"
          >
            Runtime topology
          </Typography>
          <span className="font-technical text-log text-text-subtle">
            {containers.length} {containers.length === 1 ? 'CONTAINER' : 'CONTAINERS'}
          </span>
        </div>
        {containers.length ? (
          <div className="grid gap-1">
            {containers.map((container) => (
              <div
                className="flex min-w-0 flex-wrap items-center justify-between gap-x-4 gap-y-2 rounded-lg bg-surface-1 px-3 py-2.5"
                key={container.id}
              >
                <div className="flex min-w-0 items-center gap-3">
                  <Cube
                    aria-hidden="true"
                    className="shrink-0 text-text-tertiary"
                    size={17}
                  />
                  <span className="truncate font-technical text-log text-text-primary">
                    {container.name}
                  </span>
                </div>
                <RuntimeStatus
                  label={container.status}
                  status={
                    container.healthy === false
                      ? 'degraded'
                      : toRuntimeStatus(container.status)
                  }
                />
              </div>
            ))}
          </div>
        ) : (
          <div className="grid gap-2 border-y border-border-subtle py-4">
            <Typography
              as="h3"
              role="section-title"
            >
              {neverDeployed ? 'No runtime yet' : 'No containers attached'}
            </Typography>
            <p className="text-supporting text-text-tertiary">
              {neverDeployed
                ? 'Deploy this service to create its first runtime container.'
                : 'The runtime has not reported any attached containers.'}
            </p>
          </div>
        )}
      </section>
      {configuration.length ? (
        <section className="grid gap-3">
          <Typography
            as="h2"
            role="section-title"
          >
            Configuration
          </Typography>
          <div className="grid grid-cols-[repeat(auto-fit,minmax(min(100%,16rem),1fr))] gap-x-8">
            {configuration.map((item) => (
              <DefinitionValue
                key={item.label}
                label={item.label}
                value={item.value}
              />
            ))}
          </div>
        </section>
      ) : null}
      {service.kind === 'database' ? (
        <DatabaseConnection serviceId={service.id} />
      ) : null}
      {service.capabilities.can_manage_source &&
      (service.spec?.source_type === 'git' || Boolean(service.spec?.git_url)) ? (
        <GitProviderPanel serviceId={service.id} />
      ) : null}
    </div>
  )
}

function DatabaseConnection({ serviceId }: { serviceId: string }) {
  const [revealed, setRevealed] = useState(false)
  const connection = useServiceConnection(serviceId, revealed)
  const dsn = connection.data?.dsn
  const maskedDsn = dsn?.replace(/:\/\/([^:/?#]+):([^@]+)@/, '://$1:••••••@')
  const copyConnection = async () => {
    if (!dsn) return
    try {
      await navigator.clipboard.writeText(dsn)
      showToast('Connection string copied.', 'success')
    } catch {
      showToast('Could not copy the connection string.', 'error')
    }
  }
  return (
    <section className="grid gap-3 border-t border-border-subtle pt-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <Typography
            as="h2"
            role="section-title"
          >
            Connection string
          </Typography>
          <p className="text-supporting text-text-tertiary">
            Use these credentials to connect to this database.
          </p>
        </div>
        <div className="flex items-center gap-2">
          {revealed && dsn ? (
            <Button
              size="sm"
              tone="neutral"
              onClick={() => void copyConnection()}
            >
              <CopySimple size={15} />
              Copy
            </Button>
          ) : null}
          <Button
            disabled={connection.isLoading}
            size="sm"
            tone="ghost"
            onClick={() =>
              revealed && connection.isError
                ? void connection.refetch()
                : setRevealed((value) => !value)
            }
          >
            {revealed ? <EyeSlash size={15} /> : <Eye size={15} />}
            {connection.isLoading
              ? 'Loading…'
              : revealed && connection.isError
                ? 'Retry'
                : revealed
                  ? 'Hide'
                  : 'Show connection string'}
          </Button>
        </div>
      </div>
      {revealed ? (
        <div
          aria-live="polite"
          className="min-w-0 rounded-lg border border-border-subtle bg-surface-1 px-3 py-2.5 font-technical text-log text-text-secondary"
        >
          {connection.isLoading ? (
            'Loading connection string…'
          ) : connection.isError ? (
            <span className="text-danger-strong">Connection string unavailable.</span>
          ) : maskedDsn ? (
            <span className="break-all">{maskedDsn}</span>
          ) : (
            'Connection string unavailable.'
          )}
        </div>
      ) : null}
    </section>
  )
}

function RuntimeMetric({
  label,
  value,
  detail,
}: {
  label: string
  value: string
  detail: string
}) {
  return (
    <div className="grid gap-1">
      <span className="text-label tracking-[0.08em] text-text-subtle">
        {label.toUpperCase()}
      </span>
      <span
        aria-live={label === 'CPU' || label === 'Memory' ? 'polite' : undefined}
        className="font-technical text-code text-text-primary"
      >
        {value}
      </span>
      <span className="text-label text-text-tertiary">{detail}</span>
    </div>
  )
}

function DefinitionValue({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid min-w-0 gap-1 border-b border-border-subtle py-3">
      <span className="text-label tracking-[0.08em] text-text-subtle">{label}</span>
      <span className="break-all font-technical text-log text-text-secondary">
        {value}
      </span>
    </div>
  )
}

function serviceContext(service: ServiceSummary) {
  if (service.kind === 'database') {
    const engine = service.spec?.engine
    const displayEngine = engine
      ? ({
          mariadb: 'MariaDB',
          mongodb: 'MongoDB',
          mysql: 'MySQL',
          postgres: 'PostgreSQL',
          postgresql: 'PostgreSQL',
          redis: 'Redis',
          mssql: 'SQL Server',
        }[engine.toLowerCase()] ?? engine)
      : 'Managed database'
    return `${displayEngine}${service.spec?.version ? ` ${service.spec.version}` : ''} · Database`
  }
  if (service.kind === 'compose') return 'Compose stack'
  return service.spec?.source_type === 'git' || service.spec?.git_url
    ? 'Application · Git source'
    : service.spec?.image
      ? `Application · ${service.spec.image}`
      : 'Application'
}

function formatRuntimeBytes(value: number) {
  if (value < 1024) return `${value} B`
  const units = ['KiB', 'MiB', 'GiB', 'TiB']
  let size = value / 1024
  let unit = 0
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024
    unit += 1
  }
  return `${size.toFixed(1)} ${units[unit]}`
}

function GitProviderPanel({ serviceId }: { serviceId: string }) {
  const source = useServiceSource(serviceId, true)
  const connections = useSourceControlConnections()
  const connection = selectGitHubConnection(connections.data)
  const repositories = useSourceControlRepositories(connection?.installation_id)
  const [editing, setEditing] = useState(false)
  const [repositoryId, setRepositoryId] = useState('')
  const [branch, setBranch] = useState('')
  const [rootDirectory, setRootDirectory] = useState('')
  const [autoDeploy, setAutoDeploy] = useState(false)
  const selectedRepository = repositories.data?.find((item) => item.id === repositoryId)
  const branches = useSourceControlBranches(
    repositoryId || source.data?.repository_id,
    connection?.installation_id,
  )
  const save = useSaveServiceSource(serviceId, true)
  useEffect(() => {
    if (source.data) {
      setRepositoryId(source.data.repository_id)
      setBranch(source.data.branch || source.data.default_branch)
      setRootDirectory(source.data.root_directory)
      setAutoDeploy(source.data.auto_deploy)
    }
  }, [source.data])
  const submit = () => {
    if (!(source.data && selectedRepository)) return
    const payload: ServiceSourceInput = {
      connection_id: source.data.connection_id || connection?.id || '',
      repository_id: selectedRepository.id,
      repository_owner: selectedRepository.owner,
      repository_name: selectedRepository.name,
      repository_full_name: selectedRepository.full_name,
      default_branch: selectedRepository.default_branch,
      branch: branch || selectedRepository.default_branch,
      auto_deploy: autoDeploy,
      root_directory: rootDirectory,
      environment_template_path: source.data.environment_template_path,
      watch_paths: source.data.watch_paths,
      ignore_paths: source.data.ignore_paths,
      watch_root_files: source.data.watch_root_files,
      compose_file: source.data.compose_file,
    }
    save.mutate(payload, { onSuccess: () => setEditing(false) })
  }
  if (source.isLoading)
    return <section className="h-40 animate-pulse rounded-2xl bg-surface-1" />
  if (source.isError || !source.data)
    return (
      <section className="grid gap-2 border-y border-border-subtle py-5">
        <Typography
          as="h2"
          role="section-title"
        >
          Source control
        </Typography>
        <p className="text-supporting text-text-tertiary">
          Git provider settings are not available for this service.
        </p>
      </section>
    )
  const currentRepository = repositories.data?.find(
    (item) => item.id === source.data?.repository_id,
  )
  return (
    <section className="grid gap-4 border-y border-border-subtle py-5">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <Typography
            as="h2"
            role="section-title"
          >
            Source control
          </Typography>
          <p className="text-supporting text-text-tertiary">
            Git provider and deployment source for this service.
          </p>
        </div>
        <span className="font-technical text-log uppercase text-text-subtle">
          {connection?.provider ?? 'Git provider'}
        </span>
      </div>
      <div className="grid gap-x-8 gap-y-4 sm:grid-cols-2">
        <SourceValue
          label="Provider"
          value={connection?.external_account_name || connection?.provider || 'GitHub'}
        />
        <div className="grid gap-2">
          <span className="text-label tracking-[0.08em] text-text-subtle">
            REPOSITORY
          </span>
          {editing ? (
            <Select
              value={repositoryId}
              onChange={(event) => {
                setRepositoryId(event.target.value)
                const repo = repositories.data?.find(
                  (item) => item.id === event.target.value,
                )
                if (repo) {
                  setBranch(repo.default_branch)
                }
              }}
            >
              <option value="">Choose repository</option>
              {(repositories.data ?? []).map((repo) => (
                <option
                  key={repo.id}
                  value={repo.id}
                >
                  {repo.full_name}
                </option>
              ))}
            </Select>
          ) : (
            <span className="font-technical text-code text-text-primary">
              {source.data.repository_full_name || currentRepository?.full_name || '—'}
            </span>
          )}
        </div>
        <div className="grid gap-2">
          <span className="text-label tracking-[0.08em] text-text-subtle">BRANCH</span>
          {editing ? (
            <Select
              value={branch}
              onChange={(event) => setBranch(event.target.value)}
            >
              <option value="">Choose branch</option>
              {(branches.data ?? []).map((item) => (
                <option
                  key={item.name}
                  value={item.name}
                >
                  {item.name}
                </option>
              ))}
            </Select>
          ) : (
            <span className="font-technical text-code text-text-primary">
              {source.data.branch || source.data.default_branch || 'main'}
            </span>
          )}
        </div>
        <SourceValue
          label="Root directory"
          value={source.data.root_directory || '/'}
        />
        <SourceValue
          label="Watch paths"
          value={
            source.data.watch_paths.length
              ? source.data.watch_paths.join(', ')
              : 'All service files'
          }
        />
        <div className="flex items-center gap-3">
          <span className="text-label tracking-[0.08em] text-text-subtle">
            AUTODEPLOY
          </span>
          {editing ? (
            <Switch
              checked={autoDeploy}
              onChange={(event) => setAutoDeploy(event.target.checked)}
            />
          ) : (
            <span className="font-technical text-code text-text-primary">
              {source.data.auto_deploy ? 'Enabled' : 'Disabled'}
            </span>
          )}
        </div>
      </div>
      {editing ? (
        <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto_auto]">
          <Field
            id="source-root"
            label="Root directory"
          >
            <Input
              value={rootDirectory}
              onChange={(event) => setRootDirectory(event.target.value)}
            />
          </Field>
          <Button
            tone="ghost"
            onClick={() => setEditing(false)}
          >
            Cancel
          </Button>
          <Button
            disabled={save.isPending || !repositoryId}
            onClick={submit}
          >
            {save.isPending ? 'Saving…' : 'Save source'}
          </Button>
        </div>
      ) : (
        <Button
          className="w-fit"
          tone="ghost"
          onClick={() => setEditing(true)}
        >
          Edit source
        </Button>
      )}
    </section>
  )
}

function SourceValue({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid gap-2">
      <span className="text-label tracking-[0.08em] text-text-subtle">
        {label.toUpperCase()}
      </span>
      <span className="font-technical text-code text-text-primary">{value}</span>
    </div>
  )
}

function DeploymentsPanel({
  deployments,
  selectedId,
  onSelect,
  onCancel,
  cancellingDeploymentId,
  cancelPending,
  log,
}: {
  deployments: Deployment[]
  selectedId?: string
  onSelect: (id: string) => void
  onCancel: (id: string) => void
  cancellingDeploymentId?: string
  cancelPending: boolean
  log: Array<{ id: string; level: string; message: string }>
}) {
  return (
    <div className="grid min-h-0 gap-8 xl:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)]">
      <section className="grid content-start gap-2">
        <div className="flex items-center justify-between pb-3">
          <Typography
            as="h2"
            role="section-title"
          >
            Deployment history
          </Typography>
          <span className="font-technical text-log text-text-subtle">
            {deployments.length} RUNS
          </span>
        </div>
        {deployments.length ? (
          deployments.map((deployment) => (
            <div
              className={`grid gap-2 rounded-xl border p-4 transition-[background-color,border-color] ${deployment.id === selectedId ? 'border-action/60 bg-surface-2' : 'border-transparent bg-surface-1 hover:border-border-default hover:bg-surface-2'}`}
              key={deployment.id}
            >
              <button
                aria-pressed={deployment.id === selectedId}
                className="grid w-full gap-2 text-left outline-none focus-visible:ring-2 focus-visible:ring-focus"
                onClick={() => onSelect(deployment.id)}
                type="button"
              >
                <div className="flex items-center justify-between gap-3">
                  <span className="font-technical text-code text-text-primary">
                    RUN / {String(deployment.number).padStart(3, '0')}
                  </span>
                  <RuntimeStatus
                    label={deployment.status}
                    status={toRuntimeStatus(deployment.status)}
                  />
                </div>
                <span className="truncate font-technical text-log text-text-subtle">
                  {deployment.commit || deployment.image_ref || deployment.id}
                </span>
                <span className="font-technical text-log text-text-tertiary">
                  {deployment.created_at}
                </span>
              </button>
              {isDeploymentCancelable(deployment.status) ? (
                <div className="flex justify-end border-t border-border-subtle pt-2">
                  <Button
                    aria-label={`Cancel deployment ${deployment.number}`}
                    disabled={cancelPending}
                    loading={cancelPending && cancellingDeploymentId === deployment.id}
                    onClick={() => onCancel(deployment.id)}
                    size="sm"
                    tone="ghost"
                  >
                    <X size={15} />
                    Cancel deployment
                  </Button>
                </div>
              ) : null}
            </div>
          ))
        ) : (
          <EmptyState
            title="No deployments yet"
            description="The first deployment will establish the service runtime history."
          />
        )}
      </section>
      <section className="grid min-h-0 gap-3 xl:max-h-[calc(100dvh-20rem)] xl:grid-rows-[auto_minmax(0,1fr)]">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <CheckCircle
              size={18}
              className="text-success"
            />
            <Typography
              as="h2"
              role="section-title"
            >
              Deployment output
            </Typography>
          </div>
          {selectedId ? (
            <span className="font-technical text-log text-text-subtle">
              {selectedId}
            </span>
          ) : null}
        </div>
        <LogViewer
          className="min-h-0 xl:h-full xl:!max-h-none"
          follow
          lines={log}
        />
      </section>
    </div>
  )
}

type VariableDraft = {
  id: string
  name: string
  value: string
  originalName: string
  originalValue: string
  secret: boolean
  writeOnly: boolean
  isNew?: boolean
}

const MASKED_VALUE = '••••••••••••••••'

function VariablesPanel({ serviceId }: { serviceId: string }) {
  const variables = useServiceEnvironment(serviceId, true, true)
  const save = useSetEnv(serviceId, true)
  const remove = useDeleteEnv(serviceId, true)
  const [rows, setRows] = useState<VariableDraft[]>([])
  const [mode, setMode] = useState<'form' | 'raw'>('form')
  const [raw, setRaw] = useState('')
  const [rawRevealed, setRawRevealed] = useState(false)
  const [protectedRawLines, setProtectedRawLines] = useState<Set<number>>(new Set())
  const [revealed, setRevealed] = useState<Set<string>>(new Set())
  const [deletedNames, setDeletedNames] = useState<Set<string>>(new Set())
  const [query, setQuery] = useState('')
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)
  const hydrate = useRef(true)
  useEffect(() => {
    if (!(variables.data && hydrate.current)) return
    setRows(
      variables.data.env.map((item, index) => ({
        id: `${item.name}-${index}`,
        name: item.name,
        value: item.value,
        originalName: item.name,
        originalValue: item.value,
        secret: item.secret,
        writeOnly: item.secret && !item.value,
      })),
    )
    hydrate.current = false
  }, [variables.data])
  const dirty =
    deletedNames.size > 0 ||
    rows.some(
      (row) =>
        row.isNew || row.name !== row.originalName || row.value !== row.originalValue,
    )
  const duplicateNames = new Set(
    rows
      .map((row) => row.name.trim())
      .filter((name, index, names) => name && names.indexOf(name) !== index),
  )
  const filteredRows = rows.filter((row) =>
    row.name.toLowerCase().includes(query.toLowerCase()),
  )
  const changeRow = (id: string, change: Partial<VariableDraft>) => {
    setSaved(false)
    setRows((current) =>
      current.map((row) => (row.id === id ? { ...row, ...change } : row)),
    )
  }
  const addRow = () => {
    const id = `new-${Date.now()}`
    setRows((current) => [
      ...current,
      {
        id,
        name: '',
        value: '',
        originalName: '',
        originalValue: '',
        secret: false,
        writeOnly: false,
        isNew: true,
      },
    ])
    setRevealed((current) => new Set(current).add(id))
    setQuery('')
  }
  const deleteRow = (row: VariableDraft) => {
    setRows((current) => current.filter((item) => item.id !== row.id))
    if (!row.isNew) setDeletedNames((current) => new Set(current).add(row.originalName))
    setSaved(false)
  }
  const buildRaw = (maskValues: boolean) => {
    const protectedLines = new Set<number>()
    const text = rows
      .map((row, index) => {
        const hidden = row.writeOnly || maskValues
        if (hidden) protectedLines.add(index)
        return `${row.name}=${hidden ? MASKED_VALUE : row.value}`
      })
      .join('\n')
    setProtectedRawLines(protectedLines)
    return text
  }
  const toRaw = () => {
    setRaw(buildRaw(true))
    setRawRevealed(false)
    setMode('raw')
  }
  const toggleRawVisibility = () => {
    const next = !rawRevealed
    setRaw(buildRaw(!next))
    setRawRevealed(next)
  }
  const fromRaw = (text: string) => {
    const parsed = text
      .split('\n')
      .map((line, index) => parseEnvLine(line, index, protectedRawLines, rows))
      .filter((item): item is VariableDraft => item !== null)
    setRows(parsed)
    setRaw(text)
    setSaved(false)
  }
  const switchMode = (next: 'form' | 'raw') => {
    if (next === mode) return
    if (next === 'raw') toRaw()
    else {
      fromRaw(raw)
      setMode('form')
    }
  }
  const commit = async () => {
    setError('')
    setSaved(false)
    if (duplicateNames.size) {
      setError(`Duplicate variable key: ${Array.from(duplicateNames).join(', ')}`)
      return
    }
    const invalid = rows.find(
      (row) => !/^[A-Za-z_][A-Za-z0-9_]*$/.test(row.name.trim()),
    )
    if (invalid) {
      setError(`Invalid variable key: ${invalid.name || 'empty key'}`)
      return
    }
    try {
      for (const name of deletedNames) await remove.mutateAsync(name)
      for (const row of rows) {
        if (row.writeOnly && !row.value && !row.isNew) continue
        if (!row.isNew && row.name !== row.originalName)
          await remove.mutateAsync(row.originalName)
        if (
          row.isNew ||
          row.name !== row.originalName ||
          row.value !== row.originalValue
        )
          await save.mutateAsync({
            name: row.name.trim(),
            value: row.value,
            secret: row.secret,
          })
      }
      setSaved(true)
      setDeletedNames(new Set())
      setRevealed(new Set())
      hydrate.current = true
      await variables.refetch()
    } catch {
      setError('Changes could not be saved. Your draft is still intact.')
    }
  }
  if (variables.isLoading)
    return <div className="h-56 animate-pulse rounded-2xl bg-surface-1" />
  return (
    <section className="grid gap-5">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div className="grid gap-1">
          <Typography
            as="h2"
            role="section-title"
          >
            Environment variables
          </Typography>
          <p className="text-supporting text-text-tertiary">
            Configuration injected into this service at runtime.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <div
            className="flex rounded-lg border border-border-subtle bg-surface-1 p-1"
            role="group"
            aria-label="Editor mode"
          >
            <button
              className={`rounded-md px-3 py-1.5 text-label transition-colors ${mode === 'form' ? 'bg-surface-2 text-text-primary' : 'text-text-tertiary hover:text-text-primary'}`}
              onClick={() => switchMode('form')}
              type="button"
            >
              Form
            </button>
            <button
              className={`rounded-md px-3 py-1.5 text-label transition-colors ${mode === 'raw' ? 'bg-surface-2 text-text-primary' : 'text-text-tertiary hover:text-text-primary'}`}
              onClick={() => switchMode('raw')}
              type="button"
            >
              &lt;/&gt; Raw
            </button>
          </div>
          <Button
            disabled={!dirty || save.isPending || remove.isPending}
            onClick={() => void commit()}
          >
            {save.isPending || remove.isPending
              ? 'Saving…'
              : saved
                ? 'Saved'
                : 'Save changes'}
          </Button>
        </div>
      </div>
      {error ? (
        <div
          className="rounded-lg border border-danger/40 bg-danger-soft px-3 py-2 text-supporting text-danger-strong"
          role="alert"
        >
          {error}
        </div>
      ) : null}
      {mode === 'raw' ? (
        <>
          <RawVariablesEditor
            revealed={rawRevealed}
            value={raw}
            onChange={fromRaw}
            onToggleReveal={toggleRawVisibility}
          />
          {duplicateNames.size ? (
            <p
              className="text-supporting text-danger-strong"
              role="alert"
            >
              Duplicate variable key: {Array.from(duplicateNames).join(', ')}
            </p>
          ) : null}
        </>
      ) : (
        <div className="grid overflow-hidden rounded-2xl border border-border-subtle bg-surface-1">
          <div className="grid grid-cols-12 items-center gap-4 border-b border-border-subtle px-4 py-2.5 text-left text-label tracking-[0.08em] text-text-subtle">
            <span className="col-span-4">KEY</span>
            <span className="col-span-7">VALUE</span>
            <div className="col-span-1 flex items-center justify-end">
              <label className="flex items-center gap-2 normal-case tracking-normal">
                <MagnifyingGlass size={15} />
                <input
                  aria-label="Filter variables"
                  className="w-28 bg-transparent text-label text-text-primary outline-none placeholder:text-text-tertiary"
                  onChange={(event) => setQuery(event.target.value)}
                  placeholder="Filter"
                  value={query}
                />
              </label>
            </div>
          </div>
          {filteredRows.map((row) => (
            <VariableRow
              key={row.id}
              row={row}
              revealed={revealed.has(row.id)}
              duplicate={duplicateNames.has(row.name.trim())}
              onChange={changeRow}
              onReveal={() =>
                setRevealed((current) => {
                  const next = new Set(current)
                  if (next.has(row.id)) next.delete(row.id)
                  else next.add(row.id)
                  return next
                })
              }
              onDelete={() => deleteRow(row)}
            />
          ))}
          <button
            className="flex min-h-12 items-center gap-2 px-4 text-label text-text-secondary transition-colors hover:bg-surface-2 focus-visible:outline-2 focus-visible:outline-focus"
            onClick={addRow}
            type="button"
          >
            <Plus size={16} />
            Add variable
          </button>
          {!filteredRows.length && rows.length ? (
            <p className="px-4 py-6 text-center text-supporting text-text-tertiary">
              No variables match this filter.
            </p>
          ) : null}
          {!rows.length ? (
            <EmptyState
              title="No variables configured"
              description="Add a variable to establish service runtime configuration."
            />
          ) : null}
        </div>
      )}
    </section>
  )
}

function VariableRow({
  row,
  revealed,
  duplicate,
  onChange,
  onReveal,
  onDelete,
}: {
  row: VariableDraft
  revealed: boolean
  duplicate: boolean
  onChange: (id: string, change: Partial<VariableDraft>) => void
  onReveal: () => void
  onDelete: () => void
}) {
  return (
    <div
      className={`grid grid-cols-12 items-center gap-4 border-b border-border-subtle bg-surface-1 px-4 py-2.5 text-left transition-colors hover:bg-surface-2 ${duplicate ? 'bg-danger-soft/30' : ''}`}
    >
      <div className="col-span-4 min-w-0">
        <Input
          autoFocus={row.isNew}
          aria-label="Variable key"
          className={`!border-border-subtle !bg-surface-2 hover:!border-border-default hover:!bg-surface-2 focus:!border-focus focus:!ring-focus/25 ${duplicate ? '!border-danger' : ''}`}
          placeholder="VARIABLE_NAME"
          value={row.name}
          onChange={(event) => onChange(row.id, { name: event.target.value })}
        />
        {duplicate ? (
          <span className="text-label text-danger-strong">Duplicate key</span>
        ) : null}
      </div>
      <div className="col-span-7 flex min-w-0 items-center gap-2">
        <Input
          aria-label={`Value for ${row.name || 'new variable'}`}
          className="!border-border-subtle !bg-surface-2 hover:!border-border-default hover:!bg-surface-2 focus:!border-focus focus:!ring-focus/25"
          disabled={!(revealed || row.isNew)}
          placeholder={
            row.writeOnly ? 'Stored securely · enter to replace' : MASKED_VALUE
          }
          type={revealed || row.isNew ? 'text' : 'password'}
          value={revealed || row.isNew ? row.value : MASKED_VALUE}
          onChange={(event) => onChange(row.id, { value: event.target.value })}
        />
        <button
          aria-label={revealed ? `Hide ${row.name}` : `Reveal ${row.name}`}
          className="grid size-8 shrink-0 cursor-pointer place-items-center rounded-md text-text-tertiary transition-colors hover:bg-surface-3 hover:text-text-primary focus-visible:outline-2 focus-visible:outline-focus"
          onClick={onReveal}
          title={revealed ? 'Hide value' : 'Reveal value'}
          type="button"
        >
          {revealed ? <EyeSlash size={17} /> : <Eye size={17} />}
        </button>
      </div>
      <button
        aria-label={`Delete ${row.name || 'new variable'}`}
        className="col-span-1 grid size-8 cursor-pointer place-items-center rounded-md text-text-tertiary transition-colors hover:bg-danger-soft hover:text-danger focus-visible:outline-2 focus-visible:outline-focus"
        onClick={onDelete}
        title="Delete variable"
        type="button"
      >
        <X size={17} />
      </button>
    </div>
  )
}

function RawVariablesEditor({
  revealed,
  value,
  onChange,
  onToggleReveal,
}: {
  revealed: boolean
  value: string
  onChange: (value: string) => void
  onToggleReveal: () => void
}) {
  return (
    <div className="grid gap-2">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <span className="text-label tracking-[0.08em] text-text-subtle">
            .ENV EDITOR
          </span>
          <span className="font-technical text-log text-text-tertiary">
            {revealed ? 'Values revealed' : 'Values masked by default'}
          </span>
        </div>
        <button
          aria-label={revealed ? 'Mask raw values' : 'Reveal raw values'}
          className="grid size-8 cursor-pointer place-items-center rounded-md text-text-tertiary transition-colors hover:bg-surface-2 hover:text-text-primary focus-visible:outline-2 focus-visible:outline-focus"
          onClick={onToggleReveal}
          title={revealed ? 'Mask values' : 'Reveal values'}
          type="button"
        >
          {revealed ? <EyeSlash size={17} /> : <Eye size={17} />}
        </button>
      </div>
      <CodeEditorLite
        aria-label="Raw environment variables"
        className="min-h-[22rem] !bg-code-canvas"
        language="dotenv"
        onChange={(event) => onChange(event.target.value)}
        value={value}
      />
    </div>
  )
}

function parseEnvLine(
  line: string,
  index: number,
  protectedLines: Set<number>,
  previous: VariableDraft[],
): VariableDraft | null {
  if (!line.trim() || line.trimStart().startsWith('#')) return null
  const separator = line.indexOf('=')
  if (separator < 1)
    return {
      id: `raw-${index}`,
      name: line.trim(),
      value: '',
      originalName: '',
      originalValue: '',
      secret: false,
      writeOnly: false,
      isNew: true,
    }
  const name = line.slice(0, separator).trim()
  const value = line.slice(separator + 1)
  const existing =
    previous.find((row) => row.name === name || row.originalName === name) ??
    (protectedLines.has(index) ? previous[index] : undefined)
  const protectedValue = protectedLines.has(index) && value === MASKED_VALUE
  return {
    id: existing?.id ?? `raw-${index}`,
    name,
    value: protectedValue ? (existing?.value ?? '') : value,
    originalName: existing?.originalName ?? '',
    originalValue: existing?.originalValue ?? '',
    secret: existing?.secret ?? false,
    writeOnly: protectedValue ? (existing?.writeOnly ?? false) : false,
    isNew: !existing?.originalName,
  }
}

function ComposePanel({ composeId }: { composeId: string }) {
  const compose = useAppCompose(composeId)
  const content = compose.data?.compose ?? ''
  return (
    <section className="grid gap-4">
      <Typography
        as="h2"
        role="section-title"
      >
        Compose definition
      </Typography>
      <p className="text-supporting text-text-tertiary">
        The live compose definition for this stack.
      </p>
      {compose.isLoading ? (
        <div className="grid min-h-[28rem] place-items-center rounded-lg border border-border-subtle bg-code-canvas font-technical text-code text-text-tertiary">
          Loading compose definition…
        </div>
      ) : compose.isError ? (
        <div className="grid min-h-48 place-items-center rounded-lg border border-border-subtle bg-code-canvas font-technical text-code text-danger">
          The compose definition could not be loaded.
        </div>
      ) : content ? (
        <YamlCodeBlock content={content} />
      ) : (
        <div className="grid min-h-48 place-items-center rounded-lg border border-border-subtle bg-code-canvas font-technical text-code text-text-tertiary">
          No compose definition returned.
        </div>
      )}
    </section>
  )
}

function YamlCodeBlock({ content }: { content: string }) {
  return (
    <div
      aria-label="YAML compose definition"
      className="max-h-[42rem] overflow-auto rounded-lg border border-border-subtle bg-code-canvas p-4 font-technical text-code leading-6 text-text-secondary"
    >
      <pre className="m-0">
        <code>
          {content.split('\n').map((line, index) => (
            <div
              className="grid min-w-max grid-cols-[3rem_minmax(0,1fr)]"
              key={`${index}-${line}`}
            >
              <span className="select-none pr-4 text-right text-text-subtle">
                {index + 1}
              </span>
              <span className="whitespace-pre">{highlightYamlLine(line)}</span>
            </div>
          ))}
        </code>
      </pre>
    </div>
  )
}

function highlightYamlLine(line: string): ReactNode {
  const keyMatch = line.match(/^(\s*(?:-\s+)?)([^:#\s][^:]*?)(\s*:)(.*)$/)
  if (!keyMatch) return highlightYamlValue(line)
  return (
    <>
      {keyMatch[1]}
      <span className="text-action">{keyMatch[2]}</span>
      <span className="text-text-tertiary">{keyMatch[3]}</span>
      {highlightYamlValue(keyMatch[4])}
    </>
  )
}

function highlightYamlValue(value: string): ReactNode {
  const tokenPattern =
    /("(?:\\.|[^"])*"|'[^']*'|#[^\n]*|\b(?:true|false|null|yes|no)\b|-?\d+(?:\.\d+)?|[{}[\],])/g
  const parts: ReactNode[] = []
  let cursor = 0
  let match: RegExpExecArray | null
  let tokenIndex = 0
  while ((match = tokenPattern.exec(value)) !== null) {
    if (match.index > cursor) parts.push(value.slice(cursor, match.index))
    const token = match[0]
    const className = token.startsWith('#')
      ? 'text-text-subtle'
      : token.startsWith('"') || token.startsWith("'")
        ? 'text-warning'
        : /^(true|false|null|yes|no)$/.test(token)
          ? 'text-action'
          : /^-?\d/.test(token)
            ? 'text-success'
            : 'text-text-tertiary'
    parts.push(
      <span
        className={className}
        key={`${token}-${tokenIndex}`}
      >
        {token}
      </span>,
    )
    cursor = match.index + token.length
    tokenIndex += 1
  }
  if (cursor < value.length) parts.push(value.slice(cursor))
  return <>{parts}</>
}
function DomainsPanel({
  serviceId,
  defaultPort,
}: {
  serviceId: string
  defaultPort?: number
}) {
  const domains = useDomains('services', serviceId)
  const add = useAddDomain('services', serviceId)
  const update = useUpdateDomain('services', serviceId)
  const remove = useRemoveDomain('services', serviceId)
  const [editing, setEditing] = useState<Domain | null>(null)
  const [host, setHost] = useState('')
  const [port, setPort] = useState(defaultPort ? String(defaultPort) : '')
  const [https, setHttps] = useState(true)
  const reset = () => {
    setEditing(null)
    setHost('')
    setPort(defaultPort ? String(defaultPort) : '')
    setHttps(true)
  }
  const beginEdit = (domain: Domain) => {
    setEditing(domain)
    setHost(domain.host)
    setPort(String(domain.container_port))
    setHttps(domain.https)
  }
  const submit = () => {
    const containerPort = Number(port)
    if (
      !(host.trim() && Number.isInteger(containerPort)) ||
      containerPort < 1 ||
      containerPort > 65535
    )
      return
    const body = { host: host.trim(), https, container_port: containerPort }
    if (editing) update.mutate({ domainID: editing.id, body }, { onSuccess: reset })
    else add.mutate(body, { onSuccess: reset })
  }
  return (
    <section className="grid gap-5">
      <div>
        <Typography
          as="h2"
          role="section-title"
        >
          Domains
        </Typography>
        <p className="text-supporting text-text-tertiary">
          Public routing attached to this service.
        </p>
      </div>
      <div className="grid gap-2">
        {(domains.data ?? []).map((domain) => (
          <DomainRow
            domain={domain}
            key={domain.id}
            onEdit={() => beginEdit(domain)}
            onRemove={() => remove.mutate(domain.host)}
          />
        ))}
        {!domains.data?.length ? (
          <EmptyState
            title="No domains attached"
            description="Add a hostname when this service is ready for public traffic."
          />
        ) : null}
      </div>
      <div className="grid gap-4 rounded-2xl border border-border-subtle bg-surface-1 p-4">
        <div className="flex items-center justify-between gap-3">
          <Typography
            as="h3"
            role="section-title"
          >
            {editing ? 'Edit domain' : 'Add domain'}
          </Typography>
          {editing ? (
            <Button
              tone="ghost"
              onClick={reset}
            >
              <X size={16} />
              Cancel
            </Button>
          ) : null}
        </div>
        <div className="grid gap-4 sm:grid-cols-[minmax(0,1fr)_12rem] sm:items-end">
          <Field
            id="service-domain"
            label="Hostname"
          >
            <Input
              value={host}
              onChange={(event) => setHost(event.target.value)}
              placeholder="api.example.com"
            />
          </Field>
          <Field
            id="service-domain-port"
            label="Internal port"
          >
            <Input
              inputMode="numeric"
              min="1"
              max="65535"
              type="number"
              value={port}
              onChange={(event) => setPort(event.target.value)}
              placeholder="3000"
            />
          </Field>
        </div>
        <label className="flex cursor-pointer items-center gap-3 text-supporting text-text-secondary">
          <Switch
            checked={https}
            onChange={(event) => setHttps(event.target.checked)}
          />
          <span>
            <span className="block text-label text-text-primary">
              SSL with Let’s Encrypt
            </span>
            <span className="block text-log text-text-tertiary">
              Provision and renew a certificate for this hostname.
            </span>
          </span>
        </label>
        <Button
          className="w-fit"
          disabled={!(host.trim() && port) || add.isPending || update.isPending}
          onClick={submit}
        >
          {editing ? 'Save domain' : 'Add domain'}
        </Button>
      </div>
    </section>
  )
}
function DomainRow({
  domain,
  onEdit,
  onRemove,
}: {
  domain: Domain
  onEdit: () => void
  onRemove: () => void
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-4 rounded-xl bg-surface-1 p-4">
      <div className="flex items-center gap-3">
        <Globe
          size={18}
          className="text-action-strong"
        />
        <div className="grid gap-1">
          <span className="text-supporting text-text-primary">{domain.host}</span>
          <span className="font-technical text-log text-text-tertiary">
            {domain.https ? 'HTTPS · Let’s Encrypt' : 'HTTP'} · internal port{' '}
            {domain.container_port} · {domain.status.toLowerCase()}
          </span>
        </div>
      </div>
      <div className="flex items-center gap-2">
        <Button
          tone="ghost"
          onClick={onEdit}
        >
          <PencilSimple size={16} />
          Edit
        </Button>
        <Button
          tone="ghost"
          onClick={onRemove}
        >
          <Trash size={16} />
          Remove
        </Button>
      </div>
    </div>
  )
}
function LiveLogsPanel({ serviceId }: { serviceId: string }) {
  const [lines, setLines] = useState<
    Array<{ id: string; level: string; message: string }>
  >([])
  useEffect(() => {
    const server = getServer() || window.location.origin
    const url = new URL(`/api/v1/services/${serviceId}/logs`, server)
    url.searchParams.set('running', '1')
    url.searchParams.set('follow', '1')
    const stream = new EventSource(url, { withCredentials: true })
    stream.onmessage = (event) =>
      setLines((current) => [
        ...current.slice(-499),
        { id: `${Date.now()}-${current.length}`, level: 'OUT', message: event.data },
      ])
    return () => stream.close()
  }, [serviceId])
  return (
    <section className="grid gap-4">
      <div>
        <Typography
          as="h2"
          role="section-title"
        >
          Live logs
        </Typography>
        <p className="text-supporting text-text-tertiary">
          Runtime output arrives through the service log stream.
        </p>
      </div>
      <LogViewer
        className="w-full lg:w-[110%]"
        follow
        lines={lines}
      />
    </section>
  )
}
function SettingsPanel({ service }: { service: ServiceSummary }) {
  const update = useUpdateService()
  const policyEnabled = service.kind === 'app' && service.capabilities.can_build
  const policy = usePolicy(service.id, policyEnabled)
  const policyEvents = usePolicyEvents(service.id, policyEnabled)
  const savePolicy = useSavePolicy(service.id)
  const [name, setName] = useState(service.name)
  const [port, setPort] = useState(service.spec?.port ? String(service.spec.port) : '')
  const [buildType, setBuildType] = useState(service.spec?.build_type ?? 'buildpacks')
  const [cpus, setCpus] = useState(
    Number.parseFloat(service.spec?.cpus ?? '0.5') || 0.5,
  )
  const [memory, setMemory] = useState(service.spec?.mem_mb || 256)
  const [storage, setStorage] = useState(service.spec?.storage_mb || 0)
  const [retention, setRetention] = useState(service.spec?.image_retention || 0)
  const [saved, setSaved] = useState(false)
  useEffect(() => {
    setName(service.name)
    setPort(service.spec?.port ? String(service.spec.port) : '')
    setBuildType(service.spec?.build_type ?? 'buildpacks')
    setCpus(Number.parseFloat(service.spec?.cpus ?? '0.5') || 0.5)
    setMemory(service.spec?.mem_mb || 256)
    setStorage(service.spec?.storage_mb || 0)
    setRetention(service.spec?.image_retention || 0)
  }, [service])
  const save = () => {
    setSaved(false)
    const resources =
      service.kind === 'app'
        ? { cpus: String(cpus), mem_mb: memory, storage_mb: storage }
        : { mem_mb: memory, storage_mb: storage }
    const updatePayload = {
      name: name.trim(),
      ...(port ? { port: Number(port) } : {}),
      ...(service.kind === 'app'
        ? {
            build_type: buildType as 'dockerfile' | 'buildpacks' | 'custom' | 'compose',
            image_retention: retention,
            resources,
          }
        : {}),
      ...(service.kind === 'database' ? { resources } : {}),
    }
    update.mutate(
      { serviceId: service.id, update: updatePayload },
      { onSuccess: () => setSaved(true) },
    )
  }
  const currentPolicy = policy.data ?? {
    app_id: service.id,
    enabled: false,
    cpu_min: 0.25,
    cpu_max: 4,
    mem_min_mb: 256,
    mem_max_mb: 2048,
    scale_up_pct: 80,
    scale_down_pct: 15,
    cooldown_min: 10,
  }
  const savePolicySettings = (enabled: boolean) =>
    savePolicy.mutate({ ...currentPolicy, enabled, app_id: service.id })
  return (
    <section className="grid gap-6">
      <header className="grid gap-2">
        <div className="flex items-center gap-3">
          <span className="grid size-10 place-items-center rounded-xl border border-action/30 bg-action-soft text-action-strong">
            <Cube size={20} />
          </span>
          <div>
            <Typography
              as="h2"
              role="section-title"
            >
              Service settings
            </Typography>
            <p className="text-supporting text-text-tertiary">
              Shape the runtime contract without leaving the operational surface.
            </p>
          </div>
        </div>
      </header>
      <div className="grid gap-4 xl:grid-cols-[minmax(0,1.1fr)_minmax(20rem,0.9fr)]">
        <SettingsCard
          eyebrow="IDENTITY"
          title="Service contract"
          description="The stable identity and network contract used by the runtime."
        >
          <div className="grid gap-4 sm:grid-cols-2">
            <Field
              id="service-name"
              label="Service name"
              required
            >
              <Input
                value={name}
                onChange={(event) => {
                  setName(event.target.value)
                  setSaved(false)
                }}
              />
            </Field>
            <Field
              id="service-port"
              label="Container port"
            >
              <Input
                inputMode="numeric"
                value={port}
                onChange={(event) => {
                  setPort(event.target.value)
                  setSaved(false)
                }}
                placeholder="Optional"
              />
            </Field>
          </div>
          {service.kind === 'app' ? (
            <Field
              id="service-build-type"
              label="Build strategy"
            >
              <Select
                value={buildType}
                onChange={(event) => {
                  setBuildType(event.target.value)
                  setSaved(false)
                }}
              >
                <option value="buildpacks">SmartBuild (CNB)</option>
                <option value="dockerfile">Dockerfile</option>
                <option value="custom">Custom commands</option>
              </Select>
            </Field>
          ) : null}
          <div className="flex flex-wrap items-center justify-between gap-3 border-t border-border-subtle pt-4">
            <span className="text-supporting text-text-tertiary">
              {saved
                ? 'Settings saved to the service contract.'
                : 'Changes apply on the next deploy.'}
            </span>
            <Button
              className="w-fit"
              disabled={!name.trim() || update.isPending}
              onClick={save}
            >
              {update.isPending ? 'Saving…' : saved ? 'Saved' : 'Save settings'}
            </Button>
          </div>
        </SettingsCard>
        {service.kind === 'app' ? (
          <SettingsCard
            eyebrow="REGISTRY"
            title="Image retention"
            description="Keep the most recent built images for this service. Set 0 to use the organization policy."
          >
            <Field
              id="image-retention"
              label="Retained images"
            >
              <Input
                inputMode="numeric"
                min="0"
                type="number"
                value={retention}
                onChange={(event) => {
                  setRetention(
                    Math.max(0, Number.parseInt(event.target.value, 10) || 0),
                  )
                  setSaved(false)
                }}
              />
            </Field>
            <div className="rounded-xl border border-border-subtle bg-surface-2 p-3">
              <span className="text-label text-text-primary">Organization policy</span>
              <p className="mt-1 text-supporting text-text-tertiary">
                A value of 0 delegates retention to the global registry policy.
              </p>
            </div>
          </SettingsCard>
        ) : null}
      </div>
      {service.kind === 'app' || service.kind === 'database' ? (
        <SettingsCard
          eyebrow="RESOURCES"
          title="Runtime allocation"
          description="Tune the resources reserved for the next runtime. Adjustments are applied on the next deploy."
        >
          <div
            className={`grid gap-5 ${service.kind === 'app' ? 'md:grid-cols-3' : 'md:grid-cols-2'}`}
          >
            {service.kind === 'app' ? (
              <ResourceSlider
                label="CPU"
                displayValue={`${cpus} vCPU`}
                min="0.25"
                max="2"
                step="0.25"
                current={cpus}
                onChange={(value) => {
                  setCpus(value)
                  setSaved(false)
                }}
              />
            ) : null}
            <ResourceSlider
              label="Memory"
              displayValue={
                memory >= 1024 && memory % 1024 === 0
                  ? `${memory / 1024} GB`
                  : `${memory} MB`
              }
              min="256"
              max="2048"
              step="256"
              current={memory}
              onChange={(value) => {
                setMemory(value)
                setSaved(false)
              }}
            />
            <ResourceSlider
              label="Storage"
              displayValue={
                storage
                  ? `${storage >= 1024 ? `${storage / 1024} GB` : `${storage} MB`}`
                  : 'Unlimited'
              }
              min="0"
              max="102400"
              step="1024"
              current={storage}
              onChange={(value) => {
                setStorage(value)
                setSaved(false)
              }}
            />
          </div>
        </SettingsCard>
      ) : null}
      {policyEnabled ? (
        <SettingsCard
          eyebrow="AUTOPILOT"
          title="Resource Autopilot"
          description="Let the control plane adjust runtime resources in response to sustained pressure."
        >
          <div className="flex flex-wrap items-center justify-between gap-4 rounded-xl border border-border-subtle bg-surface-2 p-4">
            <div>
              <span className="text-supporting font-medium text-text-primary">
                Automatic resource adjustments
              </span>
              <p className="mt-1 text-supporting text-text-tertiary">
                Checks runtime pressure every 60 seconds.
              </p>
            </div>
            <Switch
              checked={Boolean(policy.data?.enabled)}
              onChange={(event) => savePolicySettings(event.target.checked)}
            />
          </div>
          <div className="grid gap-3 sm:grid-cols-4">
            <SettingsFact
              label="SCALE UP"
              value={`${currentPolicy.scale_up_pct}% memory`}
            />
            <SettingsFact
              label="SCALE DOWN"
              value={`${currentPolicy.scale_down_pct}% memory`}
            />
            <SettingsFact
              label="MEMORY LIMIT"
              value={`${currentPolicy.mem_min_mb}–${currentPolicy.mem_max_mb} MB`}
            />
            <SettingsFact
              label="CPU LIMIT"
              value={`${currentPolicy.cpu_min}–${currentPolicy.cpu_max} vCPU`}
            />
          </div>
          {policyEvents.data?.length ? (
            <div className="grid gap-2 border-t border-border-subtle pt-4">
              <span className="text-label tracking-[0.08em] text-text-subtle">
                RECENT ACTIONS
              </span>
              {policyEvents.data.slice(0, 3).map((event) => (
                <div
                  className="flex flex-wrap items-center justify-between gap-3 rounded-lg bg-surface-2 px-3 py-2"
                  key={event.id}
                >
                  <span className="text-supporting text-action-strong">
                    {event.action}
                  </span>
                  <span className="text-supporting text-text-tertiary">
                    {event.detail}
                  </span>
                </div>
              ))}
            </div>
          ) : null}
        </SettingsCard>
      ) : null}
      {service.capabilities.can_manage_source &&
      (service.spec?.source_type === 'git' || Boolean(service.spec?.git_url)) ? (
        <GitProviderPanel serviceId={service.id} />
      ) : null}
      {service.kind === 'app' && service.capabilities.can_build ? (
        <WebhookSettings serviceId={service.id} />
      ) : null}
    </section>
  )
}

function SettingsCard({
  eyebrow,
  title,
  description,
  children,
}: {
  eyebrow: string
  title: string
  description: string
  children: ReactNode
}) {
  return (
    <section className="grid gap-5 rounded-2xl border border-border-subtle bg-surface-1 p-5 sm:p-6">
      <div className="grid gap-1">
        <span className="text-label tracking-[0.14em] text-action-strong">
          {eyebrow}
        </span>
        <Typography
          as="h3"
          role="section-title"
        >
          {title}
        </Typography>
        <p className="text-supporting text-text-tertiary">{description}</p>
      </div>
      {children}
    </section>
  )
}

function ResourceSlider({
  label,
  displayValue,
  min,
  max,
  step,
  onChange,
  current,
}: {
  label: string
  displayValue: string
  min: string
  max: string
  step: string
  onChange: (value: number) => void
  current: number
}) {
  return (
    <label className="grid gap-3">
      <div className="flex items-center justify-between gap-3">
        <span className="text-label text-text-secondary">{label}</span>
        <span className="font-technical text-log text-action-strong">
          {displayValue}
        </span>
      </div>
      <Slider
        min={min}
        max={max}
        step={step}
        value={current}
        onChange={(event) => onChange(Number(event.target.value))}
      />
    </label>
  )
}

function SettingsFact({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid gap-1 rounded-xl bg-surface-2 p-3">
      <span className="text-label tracking-[0.08em] text-text-subtle">{label}</span>
      <span className="font-technical text-log text-text-primary">{value}</span>
    </div>
  )
}

function WebhookSettings({ serviceId }: { serviceId: string }) {
  const webhook = useSetWebhook(serviceId, true)
  const [secret, setSecret] = useState('')
  const [saved, setSaved] = useState(false)
  return (
    <SettingsCard
      eyebrow="INTEGRATION"
      title="GitHub webhook"
      description="Accept signed deployment events from the connected repository."
    >
      <div className="grid gap-4 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-end">
        <Field
          id="github-webhook-secret"
          label="Signing secret"
        >
          <Input
            type="password"
            value={secret}
            onChange={(event) => {
              setSecret(event.target.value)
              setSaved(false)
            }}
            placeholder="Enter a new webhook secret"
          />
        </Field>
        <Button
          className="w-fit"
          disabled={!secret.trim() || webhook.isPending}
          onClick={() =>
            webhook.mutate(secret, {
              onSuccess: () => {
                setSecret('')
                setSaved(true)
              },
            })
          }
        >
          {webhook.isPending ? 'Saving…' : saved ? 'Saved' : 'Save secret'}
        </Button>
      </div>
      <p className="font-technical text-log text-text-tertiary">
        POST /api/v1/webhooks/github/{serviceId}
      </p>
    </SettingsCard>
  )
}
function CronPanel({ serviceId }: { serviceId: string }) {
  const jobs = useCronJobs(serviceId, true)
  return (
    <section className="grid gap-5">
      <div>
        <Typography
          as="h2"
          role="section-title"
        >
          Scheduled jobs
        </Typography>
        <p className="text-supporting text-text-tertiary">
          Schedules attached to this service.
        </p>
      </div>
      {jobs.data?.length ? (
        <div className="grid gap-2">
          {jobs.data.map((job) => (
            <div
              className="grid gap-1 rounded-xl bg-surface-1 p-4"
              key={job.id}
            >
              <span className="text-supporting text-text-primary">{job.name}</span>
              <span className="font-technical text-log text-text-tertiary">
                {job.schedule} · {job.command}
              </span>
            </div>
          ))}
        </div>
      ) : (
        <EmptyState
          title="No scheduled jobs"
          description="Cron jobs will appear here when configured for this service."
        />
      )}
    </section>
  )
}
const TERMINAL_SHELLS = [
  { id: 'sh', label: '/bin/sh', command: '/bin/sh' },
  { id: 'bash', label: 'bash', command: 'bash' },
  { id: 'ash', label: 'ash', command: 'ash' },
]

function TerminalPanel({ serviceId }: { serviceId: string }) {
  const containers = useServiceContainers(serviceId)
  const active = containers.data?.find((item) => item.status === 'running')?.name
  const [shell, setShell] = useState(TERMINAL_SHELLS[0])
  const [connectionState, setConnectionState] = useState<
    'connected' | 'connecting' | 'reconnecting' | 'disconnected'
  >('connecting')
  const [searchOpen, setSearchOpen] = useState(false)
  const [search, setSearch] = useState('')
  const socket = useRef<WebSocket | null>(null)
  const retry = useRef<ReturnType<typeof setTimeout> | null>(null)
  const terminalRef = useRef<XTerm | null>(null)
  const fitRef = useRef<FitAddon | null>(null)
  const searchRef = useRef<SearchAddon | null>(null)
  const hostRef = useRef<HTMLDivElement | null>(null)

  const resize = useCallback(() => {
    const fit = fitRef.current
    const connection = socket.current
    if (!(fit && connection) || connection.readyState !== WebSocket.OPEN) return
    fit.fit()
    const terminal = terminalRef.current
    if (terminal)
      connection.send(
        JSON.stringify({ type: 'resize', cols: terminal.cols, rows: terminal.rows }),
      )
  }, [])

  useEffect(() => {
    const host = hostRef.current
    if (!host) return
    const codeCanvasColor = () =>
      getComputedStyle(host).getPropertyValue('--ely-color-code-canvas').trim()
    const terminal = new XTerm({
      allowProposedApi: true,
      convertEol: true,
      cursorBlink: true,
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
      fontSize: 13,
      lineHeight: 1.45,
      scrollback: 5000,
      theme: {
        background: codeCanvasColor(),
        foreground: '#d7dde8',
        cursor: '#b0c6ff',
        selectionBackground: '#33436e',
        black: '#101316',
        brightBlack: '#657080',
        blue: '#8ea7ff',
        brightBlue: '#b0c6ff',
        cyan: '#82d7e8',
        brightCyan: '#a4e7f4',
        green: '#7bd9a5',
        brightGreen: '#a1edbd',
        magenta: '#d7a5ff',
        brightMagenta: '#e8c7ff',
        red: '#f28b9b',
        brightRed: '#ffadb8',
        white: '#d7dde8',
        brightWhite: '#f7f9fc',
        yellow: '#e9c879',
        brightYellow: '#f4d98f',
      },
    })
    const fit = new FitAddon()
    const searchAddon = new SearchAddon()
    terminal.loadAddon(fit)
    terminal.loadAddon(searchAddon)
    terminal.loadAddon(new Unicode11Addon())
    terminal.loadAddon(new WebLinksAddon())
    terminal.open(host)
    terminalRef.current = terminal
    fitRef.current = fit
    searchRef.current = searchAddon
    const observer = new ResizeObserver(resize)
    observer.observe(host)
    const themeObserver = new MutationObserver(() => {
      const background = codeCanvasColor()
      if (background) terminal.options.theme = { ...terminal.options.theme, background }
    })
    themeObserver.observe(document.documentElement, {
      attributes: true,
      subtree: true,
      attributeFilter: ['data-theme'],
    })
    requestAnimationFrame(resize)
    return () => {
      observer.disconnect()
      themeObserver.disconnect()
      terminal.dispose()
      terminalRef.current = null
      fitRef.current = null
      searchRef.current = null
    }
  }, [resize])

  useEffect(() => {
    const terminal = terminalRef.current
    if (!(terminal && active)) return
    let disposed = false
    const connect = () => {
      if (disposed) return
      setConnectionState('connecting')
      terminal.clear()
      terminal.writeln(`Connecting to ${active} using ${shell.label}…`)
      const server = getServer() || window.location.origin
      const address = new URL(`/api/v1/ws/terminal/${serviceId}`, server)
      address.protocol = address.protocol === 'https:' ? 'wss:' : 'ws:'
      address.searchParams.set('container', active)
      address.searchParams.set('shell', shell.command)
      const connection = new WebSocket(address)
      connection.binaryType = 'arraybuffer'
      socket.current = connection
      connection.onopen = () => {
        setConnectionState('connected')
        terminal.clear()
        terminal.focus()
        resize()
      }
      connection.onmessage = (event) => {
        if (typeof event.data === 'string') terminal.write(event.data)
        else terminal.write(new Uint8Array(event.data))
      }
      connection.onclose = () => {
        if (!disposed) {
          setConnectionState('reconnecting')
          terminal.writeln('\r\nConnection lost. Reconnecting…')
          retry.current = setTimeout(connect, 1500)
        }
      }
      connection.onerror = () => setConnectionState('disconnected')
    }
    if (!active)
      terminal.writeln('No runtime container is available for a terminal session.')
    else connect()
    const data = terminal.onData((value) => {
      if (socket.current?.readyState === WebSocket.OPEN)
        socket.current.send(new TextEncoder().encode(value))
    })
    return () => {
      disposed = true
      data.dispose()
      if (retry.current) clearTimeout(retry.current)
      socket.current?.close()
      socket.current = null
    }
  }, [active, resize, serviceId, shell])

  useEffect(() => {
    if (search) searchRef.current?.findNext(search)
  }, [search])

  const runSearch = (backward = false) => {
    if (!search) return
    if (backward) searchRef.current?.findPrevious(search)
    else searchRef.current?.findNext(search)
  }
  const statusLabel =
    connectionState === 'connected'
      ? 'Connected'
      : connectionState === 'reconnecting'
        ? 'Reconnecting'
        : connectionState === 'connecting'
          ? 'Connecting'
          : 'Disconnected'
  const statusTone =
    connectionState === 'connected'
      ? 'bg-success'
      : connectionState === 'disconnected'
        ? 'bg-danger'
        : 'bg-warning'

  return (
    <section className="grid min-w-0 gap-5">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <Typography
            as="h2"
            role="section-title"
          >
            Terminal
          </Typography>
          <p className="text-supporting text-text-tertiary">
            Open a live shell against the selected runtime container.
          </p>
        </div>
        <div className="flex items-center gap-3 font-technical text-log text-text-subtle">
          <span className={`size-2 rounded-full ${statusTone}`} />
          {statusLabel}
          <span className="text-text-tertiary">{active ?? 'NO CONTAINER'}</span>
        </div>
      </div>
      <div className="min-w-0 overflow-hidden rounded-2xl border border-border-default bg-surface-1 shadow-[0_18px_48px_rgba(0,0,0,0.18)]">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border-subtle bg-surface-2 px-4 py-3">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-label text-text-subtle">SHELL</span>
            {TERMINAL_SHELLS.map((option) => (
              <button
                className={`cursor-pointer rounded-lg px-3 py-1.5 text-label transition-colors focus-visible:outline-2 focus-visible:outline-focus ${shell.id === option.id ? 'bg-action-soft text-action-strong' : 'text-text-secondary hover:bg-surface-3 hover:text-text-primary'}`}
                key={option.id}
                onClick={() => setShell(option)}
                type="button"
              >
                {option.label}
              </button>
            ))}
          </div>
          <div className="flex items-center gap-2">
            {searchOpen ? (
              <div className="flex items-center gap-2 rounded-lg border border-border-default bg-surface-1 px-2">
                <MagnifyingGlass
                  className="text-text-tertiary"
                  size={15}
                />
                <input
                  autoFocus
                  aria-label="Search terminal"
                  className="w-40 bg-transparent py-1.5 text-label text-text-primary outline-none placeholder:text-text-tertiary"
                  onChange={(event) => setSearch(event.target.value)}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter') runSearch(event.shiftKey)
                    if (event.key === 'Escape') {
                      setSearchOpen(false)
                      setSearch('')
                    }
                  }}
                  placeholder="Search output"
                  value={search}
                />
              </div>
            ) : null}
            <button
              aria-label="Search terminal output"
              className="grid size-8 cursor-pointer place-items-center rounded-lg text-text-tertiary transition-colors hover:bg-surface-3 hover:text-text-primary focus-visible:outline-2 focus-visible:outline-focus"
              onClick={() => setSearchOpen((value) => !value)}
              title="Search output"
              type="button"
            >
              <MagnifyingGlass size={17} />
            </button>
          </div>
        </div>
        <div className="h-[min(60vh,36rem)] min-h-[20rem] min-w-0 bg-code-canvas p-4">
          <div
            className="h-full min-w-0"
            ref={hostRef}
          />
        </div>
        <div className="flex items-center justify-between gap-3 border-t border-border-subtle bg-surface-2 px-4 py-2.5">
          <span className="font-technical text-log text-text-subtle">
            {shell.command}
          </span>
          <span className="text-label text-text-tertiary">
            Interactive session · resize aware
          </span>
        </div>
      </div>
    </section>
  )
}
function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid gap-1">
      <span className="text-label tracking-[0.1em] text-text-subtle">{label}</span>
      <span className="font-technical text-code text-text-secondary">{value}</span>
    </div>
  )
}
function formatTimelineTimestamp(timestamp: string) {
  const date = new Date(timestamp)
  if (Number.isNaN(date.getTime())) return timestamp
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)
}
function toRuntimeStatus(
  status?: string,
): 'healthy' | 'deploying' | 'degraded' | 'failed' | 'stopped' | 'unknown' {
  if (status === 'running' || status === 'ready' || status === 'healthy')
    return 'healthy'
  if (
    status === 'deploying' ||
    status === 'building' ||
    status === 'starting' ||
    status === 'stopping'
  )
    return 'deploying'
  if (status === 'degraded' || status === 'exited') return 'degraded'
  if (status === 'failed' || status === 'error') return 'failed'
  if (status === 'stopped') return 'stopped'
  return 'unknown'
}
function isDeploymentCancelable(status: string) {
  return ['queued', 'building', 'starting', 'health_checking'].includes(status)
}
function DetailSkeleton() {
  return (
    <div className="grid gap-8 animate-pulse">
      <div className="h-48 rounded-2xl bg-surface-1" />
      <div className="h-72 rounded-2xl bg-surface-1" />
    </div>
  )
}

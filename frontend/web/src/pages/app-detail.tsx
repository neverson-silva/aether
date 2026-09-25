import {
  Button,
  Card,
  EmptyState,
  LogViewer,
  Marker,
  RuntimeStatus,
  Skeleton,
  Tabs,
  Timeline,
  Typography,
} from '@aether/elisyum-ds'
import {
  ArrowLeft,
  GitBranch,
  Package,
  Play,
  Repeat,
  RocketLaunch,
  Stop,
} from '@phosphor-icons/react'
import { useNavigate, useParams } from '@tanstack/react-router'
import { useState } from 'react'
import type { Deployment } from '../api/types'
import { useAppDetail } from '../hooks/use-app-detail'
import { useAppRestart } from '../hooks/use-app-restart'
import { useAppStart } from '../hooks/use-app-start'
import { useAppStop } from '../hooks/use-app-stop'
import { useDeploymentLog } from '../hooks/use-deployment-log'
import { useDeployments } from '../hooks/use-deployments'

export function AppDetail() {
  const { appId } = useParams({ from: '/_shell/apps/$appId' })
  const navigate = useNavigate()
  const detail = useAppDetail(appId)
  const deployments = useDeployments(appId)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const deployment =
    deployments.data?.find((item) => item.id === selectedId) ?? deployments.data?.[0]
  const log = useDeploymentLog(appId, deployment?.id ?? null)
  const start = useAppStart()
  const stop = useAppStop()
  const restart = useAppRestart()
  const app = detail.data?.app

  if (detail.isLoading)
    return (
      <div className="grid gap-4">
        <Skeleton className="h-48 rounded-2xl" />
        <Skeleton className="h-72 rounded-2xl" />
      </div>
    )
  if (detail.isError || !app)
    return (
      <EmptyState
        title="Service unavailable"
        description="The control plane did not return this service record."
        action={
          <Button
            tone="neutral"
            onClick={() => detail.refetch()}
          >
            Retry request
          </Button>
        }
      />
    )
  const running =
    app.latest_deployment?.status === 'running' ||
    app.latest_deployment?.status === 'ready'
  const status = toRuntimeStatus(app.latest_deployment?.status)
  const lines = (log.data?.content ?? '')
    .split('\n')
    .filter(Boolean)
    .map((line, index) => ({
      id: `${deployment?.id ?? 'log'}-${index}`,
      level: 'OUT',
      message: line,
    }))
  const timeline = (deployments.data ?? [])
    .slice(0, 8)
    .map((item) => ({
      id: item.id,
      title: `Delivery ${item.number}`,
      description: item.commit || item.image_ref || 'Deployment artifact',
      timestamp: item.created_at,
      status: item.status,
      tone: toTone(item.status),
    }))
  return (
    <div className="grid gap-6">
      <header className="grid gap-5">
        <button
          className="flex w-fit items-center gap-2 text-label text-text-tertiary transition-colors hover:text-text-primary"
          onClick={() => navigate({ to: '/apps' })}
          type="button"
        >
          <ArrowLeft size={16} />
          Service field
        </button>
        <div className="flex flex-wrap items-end justify-between gap-5">
          <div className="grid gap-3">
            <div className="flex items-center gap-3 text-label tracking-[0.16em] text-text-subtle">
              <Package size={17} />
              SERVICE / {app.source_type.toUpperCase()}
            </div>
            <Typography
              as="h1"
              role="page-title"
            >
              {app.name}
            </Typography>
            <Typography role="supporting">
              A deployment surface for source, runtime state and delivery output.
            </Typography>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              tone="neutral"
              onClick={() => (running ? stop.mutate(app.id) : start.mutate(app.id))}
              disabled={start.isPending || stop.isPending}
            >
              {running ? <Stop size={16} /> : <Play size={16} />}
              {running ? 'Stop service' : 'Start service'}
            </Button>
            <Button
              onClick={() => restart.mutate(app.id)}
              disabled={restart.isPending}
            >
              <Repeat size={16} />
              Restart service
            </Button>
          </div>
        </div>
        <div className="grid gap-1 font-technical text-log text-text-subtle sm:grid-cols-3">
          <span>SERVICE ID / {app.id}</span>
          <span>PROJECT / {app.project_id}</span>
          <span>SOURCE / {app.source_type === 'git' ? app.git_url : app.image}</span>
        </div>
      </header>
      <Tabs
        items={[
          {
            value: 'overview',
            label: 'Overview',
            content: (
              <Overview
                app={app}
                status={status}
              />
            ),
          },
          {
            value: 'delivery',
            label: 'Delivery',
            content: (
              <Delivery
                deployments={deployments.data ?? []}
                selectedId={deployment?.id}
                onSelect={setSelectedId}
                lines={lines}
              />
            ),
          },
          {
            value: 'history',
            label: 'History',
            content: timeline.length ? (
              <Timeline items={timeline} />
            ) : (
              <EmptyState
                title="No delivery history"
                description="A delivery will establish the service timeline."
              />
            ),
          },
        ]}
      />
    </div>
  )
}

function Overview({
  app,
  status,
}: {
  app: NonNullable<ReturnType<typeof useAppDetail>['data']>['app']
  status: ReturnType<typeof toRuntimeStatus>
}) {
  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1.3fr)_minmax(19rem,0.7fr)]">
      <section className="grid gap-4">
        <SignalBand
          items={[
            {
              label: 'RUNTIME',
              value: app.latest_deployment?.status?.toUpperCase() ?? 'UNKNOWN',
              detail: app.latest_deployment?.status ?? 'No delivery',
              tone: status === 'healthy' ? 'success' : 'warning',
            },
            {
              label: 'PORT',
              value: String(app.port),
              detail: 'Public application port',
              tone: 'accent',
            },
            {
              label: 'MEMORY',
              value: `${app.resources.mem_mb} MB`,
              detail: 'Runtime allocation',
              tone: 'neutral',
            },
          ]}
        />
        <Card className="grid gap-4 rounded-2xl border-border-subtle bg-surface-1 p-5">
          <div className="flex items-center gap-3">
            <GitBranch
              size={19}
              className="text-action-strong"
            />
            <Typography
              as="h2"
              role="section-title"
            >
              Source contract
            </Typography>
          </div>
          <div className="grid gap-2 rounded-xl bg-surface-2 p-4 sm:grid-cols-2">
            <Fact
              label="SOURCE"
              value={app.source_type === 'git' ? app.git_url : app.image}
            />
            <Fact
              label="BRANCH"
              value={app.git_branch || 'N/A'}
            />
            <Fact
              label="BUILD"
              value={app.build_type || 'dockerfile'}
            />
            <Fact
              label="SERVER"
              value={app.server_id || 'Not assigned'}
            />
          </div>
        </Card>
      </section>
      <Card className="grid content-start gap-4 rounded-2xl border-border-subtle bg-surface-1 p-5">
        <div className="flex items-center gap-3">
          <RocketLaunch
            size={19}
            className="text-action-strong"
          />
          <Typography
            as="h2"
            role="section-title"
          >
            Release posture
          </Typography>
        </div>
        <RuntimeStatus
          status={status}
          label={app.latest_deployment?.status ?? 'No delivery'}
        />
        <Fact
          label="ENVIRONMENT"
          value={app.environment_id}
        />
        <Fact
          label="UPDATED"
          value={app.updated_at}
        />
      </Card>
    </div>
  )
}

function SignalBand({
  items,
}: {
  items: Array<{
    label: string
    value: string
    detail: string
    tone: 'neutral' | 'accent' | 'success' | 'warning'
  }>
}) {
  return (
    <div className="grid gap-4 rounded-2xl border border-border-subtle bg-surface-1 p-4 md:grid-cols-3 md:gap-0">
      {items.map((item) => (
        <div
          className="grid gap-2 md:px-5 md:first:pl-1 md:not-first:border-l md:not-first:border-border-subtle"
          key={item.label}
        >
          <div className="flex items-center gap-2">
            <Marker tone={item.tone} />
            <span className="text-label text-text-subtle">{item.label}</span>
          </div>
          <span className="font-technical text-metric text-text-primary">
            {item.value}
          </span>
          <span className="text-supporting text-text-secondary">{item.detail}</span>
        </div>
      ))}
    </div>
  )
}

function Delivery({
  deployments,
  selectedId,
  onSelect,
  lines,
}: {
  deployments: Deployment[]
  selectedId?: string
  onSelect: (id: string) => void
  lines: Array<{ id: string; level: string; message: string }>
}) {
  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,0.75fr)_minmax(0,1.25fr)]">
      <Card className="grid content-start gap-2 rounded-2xl border-border-subtle bg-surface-1 p-3">
        <div className="flex items-center justify-between px-2 pb-2">
          <Typography
            as="h2"
            role="section-title"
          >
            Deployments
          </Typography>
          <span className="font-technical text-log text-text-subtle">
            {deployments.length} runs
          </span>
        </div>
        {deployments.length ? (
          deployments.map((item) => (
            <button
              aria-pressed={item.id === selectedId}
              className={`grid gap-2 rounded-xl p-4 text-left transition-colors ${item.id === selectedId ? 'bg-action-soft' : 'bg-surface-2 hover:bg-surface-3'}`}
              key={item.id}
              onClick={() => onSelect(item.id)}
              type="button"
            >
              <div className="flex items-center justify-between gap-3">
                <span className="font-technical text-code text-text-primary">
                  RUN / {String(item.number).padStart(3, '0')}
                </span>
                <RuntimeStatus
                  status={toRuntimeStatus(item.status)}
                  label={item.status}
                />
              </div>
              <span className="truncate font-technical text-log text-text-subtle">
                {item.commit || item.image_ref || item.id}
              </span>
            </button>
          ))
        ) : (
          <EmptyState
            title="No deployments"
            description="The first release will appear here."
          />
        )}
      </Card>
      <Card className="grid gap-4 rounded-2xl border-border-subtle bg-surface-1 p-5">
        <div className="flex items-center gap-3">
          <RocketLaunch
            size={19}
            className="text-action-strong"
          />
          <Typography
            as="h2"
            role="section-title"
          >
            Delivery output
          </Typography>
        </div>
        <LogViewer lines={lines} />
      </Card>
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
function toRuntimeStatus(
  status?: string,
): 'healthy' | 'deploying' | 'degraded' | 'failed' | 'stopped' | 'unknown' {
  if (status === 'running' || status === 'ready') return 'healthy'
  if (status === 'building' || status === 'deploying' || status === 'starting')
    return 'deploying'
  if (status === 'degraded') return 'degraded'
  if (status === 'failed' || status === 'error') return 'failed'
  if (status === 'stopped') return 'stopped'
  return 'unknown'
}
function toTone(status: string) {
  return status === 'failed'
    ? ('danger' as const)
    : status === 'running' || status === 'ready'
      ? ('success' as const)
      : status === 'building' || status === 'deploying'
        ? ('accent' as const)
        : ('neutral' as const)
}

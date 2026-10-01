import {
  Button,
  Card,
  EmptyState,
  Field,
  Input,
  Marker,
  RuntimeStatus,
  Skeleton,
  Switch,
  Tabs,
  Typography,
} from '@aether/elisyum-ds'
import {
  ArrowLeft,
  CopySimple,
  Database as DatabaseIcon,
  Eye,
  EyeSlash,
  HardDrives,
  Play,
  Repeat,
  ShieldCheck,
  Stop,
  WarningCircle,
} from '@phosphor-icons/react'
import { useNavigate, useParams } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import type { BackupJob, Database, DatabaseConnectionDetails } from '../api/types'
import { useDatabaseBackupConfig } from '../hooks/use-database-backup-config'
import { useDatabaseBackups } from '../hooks/use-database-backups'
import { useDatabaseDeploy } from '../hooks/use-database-deploy'
import { useDatabaseDetail } from '../hooks/use-database-detail'
import { useRevealDatabaseConnection } from '../hooks/use-reveal-database-connection'
import { useServiceAction } from '../hooks/use-service-action'
import { useServiceDetails } from '../hooks/use-service-details'
import { useUpdateDatabaseNetworkAccess } from '../hooks/use-update-database-network-access'

export function DatabaseDetail() {
  const { dbId } = useParams({ from: '/_shell/databases/$dbId' })
  const navigate = useNavigate()
  const detail = useDatabaseDetail(dbId)
  const database = detail.data?.database
  const targetId = database?.service_id ?? dbId
  const serviceDetail = useServiceDetails(targetId, Boolean(database?.service_id))
  const serviceStatus = serviceDetail.data?.status ?? database?.status ?? 'unknown'
  const backups = useDatabaseBackups(targetId, 20)
  const backupConfig = useDatabaseBackupConfig(targetId)
  const deploy = useDatabaseDeploy()
  const action = serviceStatus === 'running' ? 'stop' : 'start'
  const lifecycle = useServiceAction(action)

  if (detail.isLoading)
    return (
      <div className="grid gap-4">
        <Skeleton className="h-48 rounded-2xl" />
        <Skeleton className="h-72 rounded-2xl" />
      </div>
    )
  if (detail.isError || !database || !detail.data)
    return (
      <EmptyState
        title="Database unavailable"
        description="The control plane did not return this database record."
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
  const record = detail.data

  const transitioning = serviceStatus === 'starting' || serviceStatus === 'stopping'
  const actionLabel =
    serviceStatus === 'starting'
      ? 'Starting…'
      : serviceStatus === 'stopping'
        ? 'Stopping…'
        : action === 'stop'
          ? 'Stop database'
          : 'Start database'
  return (
    <div className="grid gap-6">
      <header className="grid gap-5">
        <button
          className="flex w-fit items-center gap-2 text-label text-text-tertiary transition-colors hover:text-text-primary"
          onClick={() => navigate({ to: '/databases' })}
          type="button"
        >
          <ArrowLeft size={16} />
          Database field
        </button>
        <div className="flex flex-wrap items-end justify-between gap-5">
          <div className="grid gap-3">
            <div className="flex items-center gap-3 text-label tracking-[0.16em] text-text-subtle">
              <DatabaseIcon size={17} />
              DATABASE / {database.engine.toUpperCase()}
            </div>
            <Typography
              as="h1"
              role="page-title"
            >
              {database.name}
            </Typography>
            <Typography role="supporting">
              A managed data runtime with lifecycle, connection and recovery context in
              one place.
            </Typography>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              tone="neutral"
              onClick={() => lifecycle.mutate(targetId)}
              disabled={transitioning || lifecycle.isPending || serviceDetail.isLoading}
            >
              {action === 'stop' ? <Stop size={16} /> : <Play size={16} />}
              {lifecycle.isPending
                ? `${action === 'stop' ? 'Stopping' : 'Starting'}…`
                : actionLabel}
            </Button>
            <Button
              onClick={() => deploy.mutate(database.id)}
              disabled={deploy.isPending || transitioning}
            >
              <Repeat size={16} />
              Deploy revision
            </Button>
          </div>
        </div>
        <div className="grid gap-1 font-technical text-log text-text-subtle sm:grid-cols-3">
          <span>DATABASE ID / {database.id}</span>
          <span>PROJECT / {database.project_id}</span>
          <span>PUBLIC HOST / {database.public_access ? record.public_host || 'Unavailable' : 'PRIVATE'}</span>
        </div>
      </header>
      <Tabs
        items={[
          {
            value: 'overview',
            label: 'Overview',
            content: (
              <Overview
                database={database}
                status={serviceStatus}
                publicHost={database.public_access ? record.public_host : ''}
              />
            ),
          },
          {
            value: 'backups',
            label: 'Backups',
            content: (
              <BackupSurface
                backups={backups.data?.items ?? []}
                configCount={backupConfig.data?.length ?? 0}
                loading={backups.isLoading || backupConfig.isLoading}
              />
            ),
          },
          {
            value: 'settings',
            label: 'Settings',
            content: (
              <NetworkAccessSettings
                database={database}
                publicHost={record.public_host}
              />
            ),
          },
        ]}
      />
    </div>
  )
}

function Overview({
  database,
  status,
  publicHost,
}: {
  database: Database
  status: string
  publicHost: string
}) {
  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1.35fr)_minmax(19rem,0.65fr)]">
      <section className="grid gap-4">
        <SignalBand
          items={[
            {
              label: 'STATE',
              value: status.toUpperCase(),
              detail: status,
              tone:
                status === 'running'
                  ? 'success'
                  : status === 'failed' || status === 'degraded'
                    ? 'warning'
                    : 'neutral',
            },
            {
              label: 'ENGINE',
              value: `${database.engine} ${database.version}`,
              detail: 'Managed database runtime',
              tone: 'accent',
            },
            {
              label: 'STORAGE',
              value: `${database.storage_mb} MB`,
              detail: 'Allocated capacity',
              tone: 'neutral',
            },
          ]}
        />
        <Card className="grid gap-4 rounded-2xl border-border-subtle bg-surface-1 p-5">
          <div className="flex items-center gap-3">
            <HardDrives
              size={19}
              className="text-action-strong"
            />
            <Typography
              as="h2"
              role="section-title"
            >
              Runtime contract
            </Typography>
          </div>
          <div className="grid gap-4 rounded-xl bg-surface-2 p-4 sm:grid-cols-2">
            <Fact
              label="DATABASE NAME"
              value={database.db_name}
            />
            <Fact
              label="INTERNAL PORT"
              value={String(database.internal_port)}
            />
            <Fact
              label="MEMORY"
              value={`${database.mem_mb} MB`}
            />
            <Fact
              label="CONTAINER"
              value={database.container_id || 'Not assigned'}
            />
            <Fact
              label="PUBLIC HOST"
              value={publicHost || 'Private network'}
            />
            <Fact
              label="CREATED"
              value={database.created_at}
            />
          </div>
        </Card>
      </section>
      <Card className="grid content-start gap-4 rounded-2xl border-border-subtle bg-surface-1 p-5">
        <div className="flex items-center gap-3">
          <ShieldCheck
            size={19}
            className="text-success"
          />
          <Typography
            as="h2"
            role="section-title"
          >
            Protection posture
          </Typography>
        </div>
        <div className="grid gap-3 rounded-xl bg-surface-2 p-4">
          <span className="text-supporting text-text-secondary">
            Recovery configuration is attached to this database resource.
          </span>
          <RuntimeStatus
            status="healthy"
            label="Recovery context available"
          />
        </div>
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

function BackupSurface({
  backups,
  configCount,
  loading,
}: {
  backups: BackupJob[]
  configCount: number
  loading: boolean
}) {
  if (loading) return <Skeleton className="h-72 rounded-2xl" />
  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(18rem,0.55fr)]">
      <Card className="grid gap-4 rounded-2xl border-border-subtle bg-surface-1 p-5">
        <div className="flex items-end justify-between gap-3">
          <div className="grid gap-1">
            <span className="text-label tracking-[0.12em] text-text-subtle">
              RECOVERY HISTORY
            </span>
            <Typography
              as="h2"
              role="section-title"
            >
              Database backups
            </Typography>
          </div>
          <span className="font-technical text-log text-text-subtle">
            {backups.length} records
          </span>
        </div>
        {backups.length ? (
          <div className="grid gap-2">
            {backups.map((backup) => (
              <BackupRow
                backup={backup}
                key={backup.id}
              />
            ))}
          </div>
        ) : (
          <EmptyState
            title="No backups recorded"
            description="The first protected operation will create a recovery record for this database."
            icon={<ShieldCheck size={28} />}
          />
        )}
      </Card>
      <Card className="grid content-start gap-4 rounded-2xl border-border-subtle bg-surface-1 p-5">
        <span className="text-label tracking-[0.12em] text-text-subtle">
          AUTOMATION
        </span>
        <Fact
          label="CONFIGURATIONS"
          value={String(configCount)}
        />
        <Typography role="supporting">
          Backup schedules and destinations remain attached to the database boundary.
        </Typography>
      </Card>
    </div>
  )
}

function BackupRow({ backup }: { backup: BackupJob }) {
  const status =
    backup.status === 'completed' || backup.status === 'success'
      ? 'healthy'
      : backup.status === 'failed'
        ? 'failed'
        : 'deploying'
  return (
    <div className="grid gap-3 rounded-xl bg-surface-2 p-4 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center">
      <div className="grid min-w-0 gap-1">
        <span className="truncate text-supporting text-text-primary">
          {backup.trigger} · {backup.engine} {backup.engine_version}
        </span>
        <span className="font-technical text-log text-text-subtle">
          {backup.completed_at || backup.started_at || backup.id} ·{' '}
          {formatBytes(backup.size_bytes)}
        </span>
      </div>
      <RuntimeStatus
        status={status}
        label={backup.status}
      />
    </div>
  )
}

function NetworkAccessSettings({
  database,
  publicHost,
}: {
  database: Database
  publicHost: string
}) {
  const update = useUpdateDatabaseNetworkAccess(database.id)
  const [publicAccess, setPublicAccess] = useState(database.public_access)
  const [externalPort, setExternalPort] = useState(
    String(database.external_port || database.internal_port),
  )
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    setPublicAccess(database.public_access)
    setExternalPort(String(database.external_port || database.internal_port))
  }, [database.external_port, database.internal_port, database.public_access])

  const save = async () => {
    setError('')
    setMessage('')
    try {
      await update.mutateAsync({
        public_access: publicAccess,
        external_port: publicAccess ? Number(externalPort) : 0,
      })
      setMessage('Network access saved. Deploy a revision to apply the change.')
    } catch (value) {
      setError(value instanceof Error ? value.message : 'Network settings could not be saved.')
    }
  }

  return (
    <div className="grid gap-4">
      <Card className="grid gap-5 rounded-2xl border-border-subtle bg-surface-1 p-5">
        <div className="grid gap-1">
          <Typography as="h2" role="section-title">
            Network Access
          </Typography>
          <Typography role="supporting">
            Internal access is always available to services in this environment.
          </Typography>
        </div>
        <div className="grid gap-2 rounded-xl bg-surface-2 p-4 sm:grid-cols-2">
          <Fact label="INTERNAL HOST" value={database.internal_host} />
          <Fact label="INTERNAL PORT" value={String(database.internal_port)} />
          <Fact label="DATABASE" value={database.db_name} />
          <Fact
            label="ENVIRONMENT NETWORK"
            value={`aether-env-${(database.environment_id || database.project_id).replaceAll('-', '')}`}
          />
        </div>
        <div className="grid gap-4 border-t border-border-subtle pt-4">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div className="grid gap-1">
              <span className="text-supporting text-text-primary">Public Access</span>
              <span className="text-supporting text-text-secondary">
                {publicAccess
                  ? 'External TCP access is enabled for this database.'
                  : 'Keep this database accessible only to services inside the Aether private network.'}
              </span>
            </div>
            <Switch
              aria-label="Public access"
              checked={publicAccess}
              onChange={(event) => setPublicAccess(event.target.checked)}
            />
          </div>
          {publicAccess ? (
            <div className="grid gap-3 sm:max-w-sm">
              <Field id="database-external-port" label="External port" required>
                <Input
                  inputMode="numeric"
                  max={65535}
                  min={1024}
                  type="number"
                  value={externalPort}
                  onChange={(event) => setExternalPort(event.target.value)}
                />
              </Field>
              <Typography role="supporting">
                {publicHost || 'Public host'}:{externalPort} → {database.internal_host}:{database.internal_port}
              </Typography>
              <Typography role="supporting">
                This exposes a database TCP port directly. HTTP WAF rules do not protect database protocols.
              </Typography>
            </div>
          ) : null}
          {error ? <span className="text-supporting text-danger">{error}</span> : null}
          {message ? <span className="text-supporting text-success">{message}</span> : null}
          <div className="flex flex-wrap items-center justify-between gap-3">
            <Typography role="supporting">
              Changes are saved now and take effect on the next deploy.
            </Typography>
            <Button
              disabled={update.isPending || (publicAccess && (Number(externalPort) < 1024 || Number(externalPort) > 65535))}
              onClick={() => void save()}
            >
              {update.isPending ? 'Saving…' : 'Save network settings'}
            </Button>
          </div>
        </div>
      </Card>
      <ConnectionCredentials
        database={database}
        publicAccess={publicAccess}
        publicHost={publicHost}
      />
    </div>
  )
}

function ConnectionCredentials({
  database,
  publicAccess,
  publicHost,
}: {
  database: Database
  publicAccess: boolean
  publicHost: string
}) {
  return (
    <div className="grid gap-4 xl:grid-cols-2">
      <CredentialCard database={database} scope="internal" />
      {publicAccess ? (
        <CredentialCard
          database={database}
          scope="external"
          host={publicHost}
          available={database.public_access}
        />
      ) : null}
    </div>
  )
}

function CredentialCard({
  database,
  scope,
  host,
  available = true,
}: {
  database: Database
  scope: 'internal' | 'external'
  host?: string
  available?: boolean
}) {
  const reveal = useRevealDatabaseConnection(database.id)
  const [credentials, setCredentials] = useState<DatabaseConnectionDetails | null>(null)
  const [passwordVisible, setPasswordVisible] = useState(false)
  const [error, setError] = useState('')
  const title = scope === 'internal' ? 'Internal Credentials' : 'External Credentials'

  const revealCredentials = async () => {
    setError('')
    try {
      setCredentials(await reveal.mutateAsync(scope))
    } catch (value) {
      setError(value instanceof Error ? value.message : 'Credentials could not be revealed.')
    }
  }

  const copy = async (value: string) => {
    await navigator.clipboard.writeText(value)
  }

  return (
    <Card className="grid content-start gap-4 rounded-2xl border-border-subtle bg-surface-1 p-5">
      <div className="grid gap-1">
        <Typography as="h2" role="section-title">{title}</Typography>
        <Typography role="supporting">
          {scope === 'internal'
            ? 'Use this connection from services attached to the same environment.'
            : `Connect from outside Aether through ${host || 'the configured public host'}.`}
        </Typography>
      </div>
      {credentials ? (
        <div className="grid gap-3 rounded-xl bg-surface-2 p-4">
          <Fact label="HOST" value={credentials.host} />
          <Fact label="PORT" value={String(credentials.port)} />
          <Fact label="DATABASE" value={credentials.database} />
          <Fact label="USERNAME" value={credentials.username} />
          <div className="flex items-end gap-2">
            <div className="min-w-0 flex-1">
              <Fact label="PASSWORD" value={passwordVisible ? credentials.password : '••••••••••••'} />
            </div>
            <Button tone="neutral" onClick={() => setPasswordVisible((visible) => !visible)}>
              {passwordVisible ? <EyeSlash size={16} /> : <Eye size={16} />}
              {passwordVisible ? 'Hide' : 'Show'}
            </Button>
          </div>
          <Fact label="CONNECTION URL" value={credentials.url} />
          <div className="flex flex-wrap gap-2">
            <Button tone="neutral" onClick={() => void copy(credentials.host)}><CopySimple size={15} /> Copy host</Button>
            <Button tone="neutral" onClick={() => void copy(credentials.url)}><CopySimple size={15} /> Copy URL</Button>
            <Button tone="neutral" onClick={() => void copy(credentials.username)}><CopySimple size={15} /> Copy username</Button>
            <Button tone="neutral" onClick={() => void copy(credentials.password)}><CopySimple size={15} /> Copy password</Button>
          </div>
        </div>
      ) : (
        <div className="grid justify-items-start gap-2">
          <Button
            tone="neutral"
            disabled={reveal.isPending || !available}
            onClick={() => void revealCredentials()}
          >
            {reveal.isPending ? 'Loading credentials…' : 'Reveal credentials'}
          </Button>
          {!available ? (
            <Typography role="supporting">
              Save network settings before revealing external credentials.
            </Typography>
          ) : null}
        </div>
      )}
      {error ? <span className="text-supporting text-danger">{error}</span> : null}
    </Card>
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
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KiB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MiB`
}

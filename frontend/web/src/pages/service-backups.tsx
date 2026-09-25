import {
  AlertDialog,
  Badge,
  Button,
  Card,
  EmptyState,
  Field,
  Input,
  Modal,
  RuntimeStatus,
  Select,
  Skeleton,
  showToast,
  Typography,
} from '@aether/elisyum-ds'
import {
  Archive,
  ArrowCounterClockwise,
  CheckCircle,
  CloudArrowUp,
  FileArrowUp,
  Gear,
  HardDrives,
  Plus,
  Trash,
  WarningCircle,
  XCircle,
} from '@phosphor-icons/react'
import { useQueryClient } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import type {
  BackupConfig,
  BackupJob,
  BackupSchedule,
  RestoreJob,
  S3Destination,
} from '../api/types'
import { useDatabaseBackupCancel } from '../hooks/use-database-backup-cancel'
import { useDatabaseBackupConfig } from '../hooks/use-database-backup-config'
import { useDatabaseBackupNow } from '../hooks/use-database-backup-now'
import { useDatabaseBackupPreflight } from '../hooks/use-database-backup-preflight'
import { useDatabaseBackupRestore } from '../hooks/use-database-backup-restore'
import { useDatabaseBackups } from '../hooks/use-database-backups'
import {
  useDatabaseUploadRestore,
  useRestoreJob,
} from '../hooks/use-database-upload-restore'
import { useDeleteDatabaseBackupConfig } from '../hooks/use-delete-database-backup-config'
import { useS3Destinations } from '../hooks/use-s3-destinations'
import { useUpsertDatabaseBackupConfig } from '../hooks/use-upsert-database-backup-config'

const activeBackupStatuses = new Set([
  'queued',
  'preparing',
  'running',
  'uploading',
  'verifying',
  'cancelling',
])
const scheduleTypes: Array<{ value: BackupSchedule['type']; label: string }> = [
  { value: 'hourly', label: 'Hourly' },
  { value: 'daily', label: 'Daily' },
  { value: 'weekly', label: 'Weekly' },
  { value: 'biweekly', label: 'Every 15 days' },
  { value: 'custom', label: 'Custom cron' },
]
const weekdays = [
  'sunday',
  'monday',
  'tuesday',
  'wednesday',
  'thursday',
  'friday',
  'saturday',
]
const supportedTimezones = (() => {
  try {
    return (
      (
        Intl as unknown as { supportedValuesOf?: (key: string) => string[] }
      ).supportedValuesOf?.('timeZone') ?? [
        'UTC',
        'America/Sao_Paulo',
        'America/New_York',
        'Europe/London',
      ]
    )
  } catch {
    return ['UTC', 'America/Sao_Paulo', 'America/New_York', 'Europe/London']
  }
})()
const cronField =
  /^(?:[?*]|\d+|[A-Za-z]{3})(?:-(?:\d+|[A-Za-z]{3}))?(?:\/\d+)?(?:,(?:[?*]|\d+|[A-Za-z]{3})(?:-(?:\d+|[A-Za-z]{3}))?(?:\/\d+)?)*$/

function formatBytes(bytes: number) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(1)} KiB`
  if (bytes < 1024 ** 3) return `${(bytes / 1024 ** 2).toFixed(1)} MiB`
  return `${(bytes / 1024 ** 3).toFixed(2)} GiB`
}

function formatDate(value?: string | null) {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '—' : date.toLocaleString()
}

function scheduleDescription(schedule: BackupSchedule) {
  if (schedule.type === 'hourly')
    return `Every hour at :${String(schedule.minute ?? 0).padStart(2, '0')}`
  if (schedule.type === 'daily') return `Every day at ${schedule.at ?? '03:00'}`
  if (schedule.type === 'weekly')
    return `Every ${schedule.day_of_week ?? 'sunday'} at ${schedule.at ?? '03:00'}`
  if (schedule.type === 'biweekly')
    return `Every 15 days from ${schedule.start_date ?? 'today'} at ${schedule.at ?? '03:00'}`
  return `Cron ${schedule.cron ?? '0 3 * * *'} (${schedule.timezone})`
}

function statusTone(status: string): 'success' | 'danger' | 'warning' | 'neutral' {
  if (status === 'completed') return 'success'
  if (status === 'failed') return 'danger'
  if (activeBackupStatuses.has(status)) return 'warning'
  return 'neutral'
}

export function ServiceBackups({
  serviceId,
  serviceName,
}: {
  serviceId: string
  serviceName: string
}) {
  const configs = useDatabaseBackupConfig(serviceId)
  const backups = useDatabaseBackups(serviceId, 50)
  const destinations = useS3Destinations()
  const backupNow = useDatabaseBackupNow(serviceId)
  const cancelBackup = useDatabaseBackupCancel(serviceId)
  const removeConfig = useDeleteDatabaseBackupConfig(serviceId)
  const [editing, setEditing] = useState<BackupConfig | null | undefined>(undefined)
  const [restoreTarget, setRestoreTarget] = useState<BackupJob | null>(null)
  const [uploadOpen, setUploadOpen] = useState(false)
  const destinationsById = new Map(
    (destinations.data ?? []).map((destination) => [destination.id, destination]),
  )

  const onBackupNow = (configId: string) =>
    backupNow.mutate(configId, {
      onSuccess: () => showToast('Backup queued.', 'success'),
      onError: (error) =>
        showToast(
          error instanceof Error ? error.message : 'Could not queue the backup.',
          'error',
        ),
    })

  return (
    <section className="grid min-w-0 gap-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div className="grid gap-1">
          <Typography
            as="h2"
            role="section-title"
          >
            Database backups
          </Typography>
          <p className="text-supporting text-text-tertiary">
            Schedule protected recovery points, run them on demand, or restore from an
            existing backup.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button
            tone="neutral"
            onClick={() => setUploadOpen(true)}
          >
            <FileArrowUp size={16} />
            Restore from file
          </Button>
          <Button onClick={() => setEditing(null)}>
            <Plus size={16} />
            Configure backup
          </Button>
        </div>
      </div>

      {configs.isLoading || destinations.isLoading ? (
        <Skeleton className="h-36 rounded-2xl" />
      ) : configs.isError || destinations.isError ? (
        <div
          className="rounded-xl border border-danger/35 bg-danger-soft/20 p-4 text-supporting text-danger-strong"
          role="alert"
        >
          Backup configuration could not be loaded. Refresh the page and try again.
        </div>
      ) : null}

      {!(configs.isLoading || configs.isError) && configs.data?.length === 0 ? (
        <Card className="grid gap-4 rounded-2xl border-border-subtle bg-surface-1 p-5 sm:grid-cols-[auto_minmax(0,1fr)_auto] sm:items-center">
          <span className="grid size-11 place-items-center rounded-xl bg-action-soft text-action-strong">
            <CloudArrowUp size={22} />
          </span>
          <div className="grid gap-1">
            <Typography
              as="h3"
              role="section-title"
            >
              Automated backups are not configured
            </Typography>
            <p className="text-supporting text-text-tertiary">
              Choose an S3 destination and schedule. You can also import a backup file
              to restore this database.
            </p>
            {!destinations.data?.length ? (
              <span className="text-label text-text-tertiary">
                Add an S3 destination in Storage before scheduling backups.
              </span>
            ) : null}
          </div>
          <Button
            disabled={!destinations.data?.length}
            onClick={() => setEditing(null)}
          >
            <Gear size={16} />
            Set up schedule
          </Button>
        </Card>
      ) : null}

      {configs.data?.length ? (
        <div className="grid gap-3">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <span className="text-label tracking-[0.12em] text-text-subtle">
              SCHEDULES · {configs.data.length}
            </span>
            <Button
              tone="neutral"
              onClick={() => setEditing(null)}
            >
              <Plus size={16} />
              Add schedule
            </Button>
          </div>
          {configs.data.map((config) => (
            <BackupScheduleCard
              config={config}
              destination={destinationsById.get(config.destination_id)}
              key={config.id}
              busy={backupNow.isPending}
              onBackup={() => onBackupNow(config.id)}
              onEdit={() => setEditing(config)}
              onDelete={() =>
                removeConfig.mutate(config.id, {
                  onSuccess: () =>
                    showToast(
                      'Backup schedule removed. Existing backup files are kept.',
                      'success',
                    ),
                  onError: (error) =>
                    showToast(
                      error instanceof Error
                        ? error.message
                        : 'Could not remove the schedule.',
                      'error',
                    ),
                })
              }
            />
          ))}
        </div>
      ) : null}

      <Card className="grid min-w-0 gap-0 overflow-hidden rounded-2xl border-border-subtle bg-surface-1 p-0">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border-subtle px-4 py-4 sm:px-5">
          <div className="flex items-center gap-3">
            <Archive
              className="text-action-strong"
              size={18}
            />
            <div>
              <Typography
                as="h3"
                role="section-title"
              >
                Backup history
              </Typography>
              <p className="text-label text-text-tertiary">
                {backups.data?.length ?? 0} recovery points
              </p>
            </div>
          </div>
          <Button
            tone="ghost"
            disabled={!configs.data?.length || backupNow.isPending}
            loading={backupNow.isPending}
            onClick={() => configs.data?.[0] && onBackupNow(configs.data[0].id)}
          >
            <CloudArrowUp size={16} />
            Run backup now
          </Button>
        </div>
        {backups.isLoading ? (
          <div className="grid gap-2 p-4">
            <Skeleton className="h-16 rounded-xl" />
            <Skeleton className="h-16 rounded-xl" />
          </div>
        ) : backups.isError ? (
          <div
            className="p-5 text-supporting text-danger-strong"
            role="alert"
          >
            Backup history could not be loaded.
          </div>
        ) : backups.data?.length ? (
          <div className="divide-y divide-border-subtle">
            {backups.data.map((backup) => (
              <BackupHistoryRow
                backup={backup}
                key={backup.id}
                onCancel={() =>
                  cancelBackup.mutate(backup.id, {
                    onSuccess: () =>
                      showToast('Backup cancellation requested.', 'info'),
                    onError: (error) =>
                      showToast(
                        error instanceof Error
                          ? error.message
                          : 'Could not cancel the backup.',
                        'error',
                      ),
                  })
                }
                onRestore={() => setRestoreTarget(backup)}
                canceling={cancelBackup.isPending}
              />
            ))}
          </div>
        ) : (
          <div className="p-4">
            <EmptyState
              title="No backups recorded"
              description={
                configs.data?.length
                  ? 'Run a backup now or wait for the next scheduled run.'
                  : 'Configure a schedule to create protected recovery points for this database.'
              }
              icon={<CloudArrowUp size={28} />}
            />
          </div>
        )}
      </Card>

      {editing !== undefined && destinations.data ? (
        <BackupConfigDialog
          config={editing}
          destinations={destinations.data}
          serviceId={serviceId}
          onClose={() => setEditing(undefined)}
        />
      ) : null}
      {restoreTarget ? (
        <BackupRestoreDialog
          backup={restoreTarget}
          serviceId={serviceId}
          serviceName={serviceName}
          onClose={() => setRestoreTarget(null)}
        />
      ) : null}
      {uploadOpen ? (
        <UploadRestoreDialog
          serviceId={serviceId}
          serviceName={serviceName}
          onClose={() => setUploadOpen(false)}
        />
      ) : null}
    </section>
  )
}

function BackupScheduleCard({
  config,
  destination,
  busy,
  onBackup,
  onEdit,
  onDelete,
}: {
  config: BackupConfig
  destination?: S3Destination
  busy: boolean
  onBackup: () => void
  onEdit: () => void
  onDelete: () => void
}) {
  return (
    <Card className="grid min-w-0 gap-4 rounded-2xl border-border-subtle bg-surface-1 p-4 sm:p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="grid min-w-0 gap-2">
          <div className="flex flex-wrap items-center gap-2">
            <Typography
              as="h3"
              role="section-title"
            >
              {scheduleDescription(config.schedule)}
            </Typography>
            <Badge tone={config.enabled ? 'success' : 'neutral'}>
              {config.enabled ? 'Active' : 'Disabled'}
            </Badge>
          </div>
          <p className="text-supporting text-text-tertiary">
            {destination?.name ?? 'Destination unavailable'}
            {destination?.bucket ? ` · ${destination.bucket}` : ''}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button
            size="sm"
            tone="neutral"
            onClick={onEdit}
          >
            <Gear size={15} />
            Edit
          </Button>
          <Button
            size="sm"
            disabled={busy || !config.enabled}
            loading={busy}
            onClick={onBackup}
          >
            <CloudArrowUp size={15} />
            Backup now
          </Button>
          <AlertDialog
            trigger={<Trash size={16} />}
            triggerClassName="grid size-9 cursor-pointer place-items-center rounded-lg text-text-tertiary transition-colors hover:bg-danger-soft hover:text-danger-strong focus-visible:outline-2 focus-visible:outline-focus"
            title="Remove backup schedule?"
            description="Scheduled backups will stop. Existing backup files remain in the destination."
            confirmLabel="Remove schedule"
            onConfirm={onDelete}
          />
        </div>
      </div>
      <div className="grid gap-2 border-t border-border-subtle pt-3 sm:grid-cols-3">
        <ScheduleFact
          label="RETENTION"
          value={config.retention.type === 'latest' ? 'Latest only' : 'Keep all'}
        />
        <ScheduleFact
          label="NEXT RUN"
          value={formatDate(config.next_run_at)}
        />
        <ScheduleFact
          label="PATH PREFIX"
          value={config.path_prefix || '—'}
        />
      </div>
    </Card>
  )
}

function ScheduleFact({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid min-w-0 gap-1">
      <span className="text-label tracking-[0.08em] text-text-subtle">{label}</span>
      <span className="truncate font-technical text-log text-text-secondary">
        {value}
      </span>
    </div>
  )
}

function BackupHistoryRow({
  backup,
  onCancel,
  onRestore,
  canceling,
}: {
  backup: BackupJob
  onCancel: () => void
  onRestore: () => void
  canceling: boolean
}) {
  const active = activeBackupStatuses.has(backup.status)
  const tone = statusTone(backup.status)
  return (
    <div className="grid min-w-0 gap-3 px-4 py-4 transition-colors hover:bg-surface-2/50 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center sm:px-5">
      <div className="grid min-w-0 gap-1">
        <div className="flex flex-wrap items-center gap-2">
          <Badge tone={tone}>{backup.status}</Badge>
          <span className="font-technical text-log text-text-primary">
            {formatDate(backup.completed_at ?? backup.started_at)} · {backup.trigger}
          </span>
        </div>
        <span className="truncate font-technical text-log text-text-tertiary">
          {backup.format.toUpperCase()}
          {backup.size_bytes > 0 ? ` · ${formatBytes(backup.size_bytes)}` : ''}
          {backup.engine ? ` · ${backup.engine} ${backup.engine_version}` : ''}
        </span>
        {backup.checksum ? (
          <span className="truncate font-technical text-log text-text-subtle">
            SHA-256 · {backup.checksum}
          </span>
        ) : null}
        {backup.error_message ? (
          <span className="text-label text-danger-strong">
            {backup.error_code ? `${backup.error_code} · ` : ''}
            {backup.error_message}
          </span>
        ) : null}
      </div>
      <div className="flex flex-wrap gap-2 sm:justify-end">
        {active ? (
          <Button
            size="sm"
            tone="neutral"
            disabled={canceling}
            loading={canceling}
            onClick={onCancel}
          >
            Cancel
          </Button>
        ) : null}
        {backup.status === 'completed' ? (
          <Button
            size="sm"
            tone="neutral"
            onClick={onRestore}
          >
            <ArrowCounterClockwise size={15} />
            Restore
          </Button>
        ) : null}
      </div>
    </div>
  )
}

function BackupConfigDialog({
  config,
  destinations,
  serviceId,
  onClose,
}: {
  config: BackupConfig | null
  destinations: S3Destination[]
  serviceId: string
  onClose: () => void
}) {
  const save = useUpsertDatabaseBackupConfig(serviceId)
  const [destinationId, setDestinationId] = useState(
    config?.destination_id ?? destinations[0]?.id ?? '',
  )
  const [pathPrefix, setPathPrefix] = useState(config?.path_prefix ?? 'databases')
  const [scheduleType, setScheduleType] = useState<BackupSchedule['type']>(
    config?.schedule.type ?? 'daily',
  )
  const [minute, setMinute] = useState(config?.schedule.minute ?? 0)
  const [time, setTime] = useState(config?.schedule.at ?? '03:00')
  const [day, setDay] = useState(config?.schedule.day_of_week ?? 'sunday')
  const [startDate, setStartDate] = useState(
    config?.schedule.start_date ?? new Date().toISOString().slice(0, 10),
  )
  const [cron, setCron] = useState(config?.schedule.cron ?? '0 3 * * *')
  const [timezone, setTimezone] = useState(config?.schedule.timezone ?? 'UTC')
  const [retention, setRetention] = useState<'all' | 'latest'>(
    config?.retention.type ?? 'all',
  )
  const [error, setError] = useState('')
  const cronValid =
    cron.trim().split(/\s+/).length === 5 &&
    cron
      .trim()
      .split(/\s+/)
      .every((field) => cronField.test(field))
  const selectedDestination = destinations.find(
    (destination) => destination.id === destinationId,
  )

  const submit = async () => {
    setError('')
    try {
      await save.mutateAsync({
        id: config?.id,
        enabled: true,
        destination_id: destinationId,
        path_prefix: pathPrefix.trim(),
        schedule: {
          type: scheduleType,
          minute,
          at: time,
          day_of_week: day,
          start_date: startDate,
          cron,
          timezone,
        },
        retention: { type: retention },
      })
      showToast('Backup schedule saved.', 'success')
      onClose()
    } catch (cause) {
      const message =
        cause instanceof Error ? cause.message : 'Could not save the backup schedule.'
      setError(message)
      showToast(message, 'error')
    }
  }

  return (
    <Modal
      open
      trigger={<span />}
      triggerClassName="hidden"
      hideCancel
      onOpenChange={(open) => !open && onClose()}
      title={config ? 'Edit backup schedule' : 'Configure automated backups'}
      description="Choose where recovery points are stored and when they are created."
      popupClassName="w-[min(48rem,calc(100vw-2rem))] max-h-[90dvh] overflow-y-auto"
      footer={
        <div className="flex gap-2">
          <Button
            tone="ghost"
            onClick={onClose}
          >
            Cancel
          </Button>
          <Button
            disabled={
              !(destinationId && pathPrefix.trim()) ||
              (scheduleType === 'custom' && !cronValid)
            }
            loading={save.isPending}
            onClick={() => void submit()}
          >
            Save schedule
          </Button>
        </div>
      }
    >
      <div className="grid gap-5">
        {error ? (
          <div
            className="rounded-lg border border-danger/35 bg-danger-soft/20 px-3 py-2 text-supporting text-danger-strong"
            role="alert"
          >
            {error}
          </div>
        ) : null}
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="backup-destination"
            label="Backup destination"
            required
          >
            <Select
              value={destinationId}
              onChange={(event) => setDestinationId(event.target.value)}
            >
              <option value="">Choose a destination</option>
              {destinations.map((destination) => (
                <option
                  key={destination.id}
                  value={destination.id}
                >
                  {destination.name}
                </option>
              ))}
            </Select>
          </Field>
          <Field
            id="backup-prefix"
            label="Path prefix"
            required
          >
            <Input
              value={pathPrefix}
              onChange={(event) => setPathPrefix(event.target.value)}
              placeholder="databases/production"
            />
          </Field>
        </div>
        {selectedDestination ? (
          <div className="grid gap-3 rounded-xl border border-border-subtle bg-surface-2 p-3 sm:grid-cols-3">
            <ScheduleFact
              label="BUCKET"
              value={selectedDestination.bucket || '—'}
            />
            <ScheduleFact
              label="REGION"
              value={selectedDestination.region || '—'}
            />
            <ScheduleFact
              label="ENDPOINT"
              value={selectedDestination.endpoint || 'Managed destination'}
            />
          </div>
        ) : null}
        <fieldset className="grid gap-3">
          <legend className="text-label text-text-secondary">Schedule</legend>
          <div className="flex flex-wrap gap-2">
            {scheduleTypes.map((type) => (
              <button
                aria-pressed={scheduleType === type.value}
                className={`rounded-lg border px-3 py-2 text-label transition-colors focus-visible:outline-2 focus-visible:outline-focus ${scheduleType === type.value ? 'border-action/50 bg-action-soft text-action-strong' : 'border-border-default text-text-secondary hover:bg-surface-2 hover:text-text-primary'}`}
                key={type.value}
                onClick={() => setScheduleType(type.value)}
                type="button"
              >
                {type.label}
              </button>
            ))}
          </div>
          {scheduleType === 'hourly' ? (
            <Field
              id="backup-minute"
              label="Minute past the hour"
            >
              <Input
                max={59}
                min={0}
                type="number"
                value={minute}
                onChange={(event) => setMinute(Number(event.target.value))}
              />
            </Field>
          ) : null}
          {scheduleType === 'daily' ||
          scheduleType === 'weekly' ||
          scheduleType === 'biweekly' ? (
            <div className="grid gap-4 sm:grid-cols-2">
              {scheduleType === 'weekly' ? (
                <Field
                  id="backup-day"
                  label="Day of week"
                >
                  <Select
                    value={day}
                    onChange={(event) => setDay(event.target.value)}
                  >
                    {weekdays.map((weekday) => (
                      <option
                        key={weekday}
                        value={weekday}
                      >
                        {weekday[0].toUpperCase() + weekday.slice(1)}
                      </option>
                    ))}
                  </Select>
                </Field>
              ) : null}
              {scheduleType === 'biweekly' ? (
                <Field
                  id="backup-start-date"
                  label="Start date"
                >
                  <Input
                    type="date"
                    value={startDate}
                    onChange={(event) => setStartDate(event.target.value)}
                  />
                </Field>
              ) : null}
              <Field
                id="backup-time"
                label="Run time"
              >
                <Input
                  type="time"
                  value={time}
                  onChange={(event) => setTime(event.target.value)}
                />
              </Field>
            </div>
          ) : null}
          {scheduleType === 'custom' ? (
            <Field
              id="backup-cron"
              label="Cron expression"
              description={
                cronValid
                  ? 'Use a standard five-field cron expression.'
                  : 'Enter a valid five-field cron expression, for example */15 * * * *.'
              }
            >
              <Input
                aria-invalid={!cronValid}
                value={cron}
                onChange={(event) => setCron(event.target.value)}
                placeholder="0 3 * * *"
              />
            </Field>
          ) : null}
          <Field
            id="backup-timezone"
            label="Timezone"
          >
            <Select
              value={timezone}
              onChange={(event) => setTimezone(event.target.value)}
            >
              {supportedTimezones.map((zone) => (
                <option
                  key={zone}
                  value={zone}
                >
                  {zone}
                </option>
              ))}
            </Select>
          </Field>
        </fieldset>
        <fieldset className="grid gap-2">
          <legend className="text-label text-text-secondary">Retention</legend>
          {(
            [
              {
                value: 'all',
                label: 'Keep all backups',
                detail: 'Retain every recovery point.',
              },
              {
                value: 'latest',
                label: 'Keep only the latest backup',
                detail:
                  'Replace the previous recovery point after each successful run.',
              },
            ] as const
          ).map((option) => (
            <label
              className={`flex cursor-pointer items-start gap-3 rounded-xl border p-3 ${retention === option.value ? 'border-action/50 bg-action-soft/40' : 'border-border-subtle bg-surface-1'}`}
              key={option.value}
            >
              <input
                checked={retention === option.value}
                className="mt-1 accent-action"
                name="backup-retention"
                onChange={() => setRetention(option.value)}
                type="radio"
              />
              <span className="grid gap-0.5">
                <span className="text-supporting text-text-primary">
                  {option.label}
                </span>
                <span className="text-label text-text-tertiary">{option.detail}</span>
              </span>
            </label>
          ))}
        </fieldset>
      </div>
    </Modal>
  )
}

function BackupRestoreDialog({
  backup,
  serviceId,
  serviceName,
  onClose,
}: {
  backup: BackupJob
  serviceId: string
  serviceName: string
  onClose: () => void
}) {
  const preflight = useDatabaseBackupPreflight(serviceId, backup.id, true)
  const restore = useDatabaseBackupRestore(serviceId)
  const [confirmation, setConfirmation] = useState('')
  const [error, setError] = useState('')
  const checksReady = preflight.data?.ready === true
  const submit = async () => {
    setError('')
    try {
      const result = await restore.mutateAsync(backup.id)
      showToast(
        result.status === 'completed'
          ? 'Database restore completed.'
          : 'Database restore queued.',
        'success',
      )
      onClose()
    } catch (cause) {
      const message =
        cause instanceof Error ? cause.message : 'Could not restore this backup.'
      setError(message)
      showToast(message, 'error')
    }
  }
  return (
    <Modal
      open
      trigger={<span />}
      triggerClassName="hidden"
      hideCancel
      onOpenChange={(open) => !open && onClose()}
      title="Restore database backup"
      description="Review compatibility checks before replacing the current database contents."
      popupClassName="w-[min(42rem,calc(100vw-2rem))] max-h-[90dvh] overflow-y-auto"
      footer={
        <div className="flex gap-2">
          <Button
            tone="ghost"
            disabled={restore.isPending}
            onClick={onClose}
          >
            Cancel
          </Button>
          <Button
            tone="danger"
            disabled={!checksReady || confirmation !== serviceName || restore.isPending}
            loading={restore.isPending}
            onClick={() => void submit()}
          >
            Restore backup
          </Button>
        </div>
      }
    >
      <div className="grid gap-4">
        <div className="grid gap-3 rounded-xl border border-border-subtle bg-surface-2 p-3 sm:grid-cols-3">
          <ScheduleFact
            label="BACKUP DATE"
            value={formatDate(backup.completed_at ?? backup.started_at)}
          />
          <ScheduleFact
            label="DATABASE"
            value={serviceName}
          />
          <ScheduleFact
            label="ENGINE"
            value={`${backup.engine} ${backup.engine_version}`}
          />
        </div>
        <section
          className="grid gap-2"
          aria-label="Restore preflight checks"
        >
          <div className="flex items-center justify-between">
            <span className="text-label tracking-[0.1em] text-text-subtle">
              PREFLIGHT
            </span>
            {preflight.isLoading ? (
              <span className="text-label text-text-tertiary">Checking…</span>
            ) : preflight.data ? (
              <Badge tone={preflight.data.ready ? 'success' : 'warning'}>
                {preflight.data.ready ? 'Ready' : 'Blocked'}
              </Badge>
            ) : null}
          </div>
          {preflight.isError ? (
            <p
              className="text-supporting text-danger-strong"
              role="alert"
            >
              Could not check restore compatibility. Try again before proceeding.
            </p>
          ) : null}
          {preflight.data?.checks.map((check) => (
            <div
              className="flex flex-wrap items-center gap-2 rounded-lg bg-surface-2 px-3 py-2"
              key={check.name}
            >
              {check.ok ? (
                <CheckCircle
                  className="text-success"
                  size={17}
                />
              ) : (
                <XCircle
                  className="text-danger-strong"
                  size={17}
                />
              )}
              <span className="text-label text-text-primary">{check.name}</span>
              <span className="ml-auto text-label text-text-tertiary">
                {check.message}
              </span>
            </div>
          ))}
        </section>
        <div className="grid gap-1 rounded-xl border border-warning/35 bg-warning-soft/20 p-3">
          <span className="flex items-center gap-2 text-supporting text-warning-strong">
            <WarningCircle size={17} />
            This replaces the current database data.
          </span>
          <span className="text-label text-text-secondary">
            The restore may cause downtime and cannot be automatically undone. Type the
            database name to confirm.
          </span>
        </div>
        {error ? (
          <p
            className="text-supporting text-danger-strong"
            role="alert"
          >
            {error}
          </p>
        ) : null}
        <Field
          id="restore-confirmation"
          label={`Type “${serviceName}” to confirm`}
        >
          <Input
            autoComplete="off"
            value={confirmation}
            onChange={(event) => setConfirmation(event.target.value)}
          />
        </Field>
      </div>
    </Modal>
  )
}

type UploadPhase =
  | 'pick'
  | 'uploading'
  | 'validate'
  | 'ready'
  | 'restoring'
  | 'terminal'

function UploadRestoreDialog({
  serviceId,
  serviceName,
  onClose,
}: {
  serviceId: string
  serviceName: string
  onClose: () => void
}) {
  const input = useRef<HTMLInputElement>(null)
  const queryClient = useQueryClient()
  const restoreFlow = useDatabaseUploadRestore(serviceId)
  const [phase, setPhase] = useState<UploadPhase>('pick')
  const [selected, setSelected] = useState<RestoreJob | null>(null)
  const [fileName, setFileName] = useState('')
  const [progress, setProgress] = useState<{ loaded: number; total: number } | null>(
    null,
  )
  const [confirmation, setConfirmation] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const finalRestore = useRestoreJob(serviceId, selected?.id ?? null)
  const job =
    finalRestore.data &&
    ['uploading', 'validating', 'ready', 'queued'].includes(finalRestore.data.status)
      ? selected
      : (finalRestore.data ?? selected)
  const running =
    job &&
    [
      'queued',
      'uploading',
      'validating',
      'ready',
      'preparing',
      'downloading',
      'restoring',
    ].includes(job.status)
  const done = job && ['completed', 'failed', 'cancelled'].includes(job.status)

  const selectFile = async (file: File) => {
    setFileName(file.name)
    setProgress(null)
    setError('')
    setBusy(true)
    try {
      const created = await restoreFlow.createRestore.mutateAsync(file.name)
      setSelected(created)
      setPhase('uploading')
      const uploaded = await restoreFlow.uploadRestore.mutateAsync({
        restoreId: created.id,
        file,
        onProgress: (loaded, total) => setProgress({ loaded, total }),
      })
      setSelected(uploaded)
      setPhase(uploaded.status === 'ready' ? 'ready' : 'validate')
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : 'Could not upload the restore file.',
      )
      setPhase('terminal')
    } finally {
      setBusy(false)
    }
  }

  const validate = async () => {
    if (!selected) return
    setBusy(true)
    setError('')
    try {
      const validated = await restoreFlow.validateRestore.mutateAsync(selected.id)
      setSelected(validated)
      setPhase(validated.status === 'ready' ? 'ready' : 'terminal')
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : 'The backup file could not be validated.',
      )
    } finally {
      setBusy(false)
    }
  }

  const startRestore = async () => {
    if (!selected || confirmation !== serviceName) return
    setBusy(true)
    setError('')
    try {
      const started = await restoreFlow.startRestore.mutateAsync(selected.id)
      setSelected(started)
      setPhase('restoring')
      showToast('Restore started.', 'info')
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Could not start the restore.')
    } finally {
      setBusy(false)
    }
  }

  const cancelRestore = async () => {
    if (!selected) return
    setBusy(true)
    try {
      await restoreFlow.cancelRestore.mutateAsync(selected.id)
      await queryClient.invalidateQueries({
        queryKey: ['database-restore-jobs', 'service', serviceId],
      })
      showToast('Uploaded restore cancelled.', 'info')
      onClose()
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Could not cancel the restore.')
    } finally {
      setBusy(false)
    }
  }

  const close = () => {
    if (busy) return
    onClose()
  }

  return (
    <Modal
      open
      trigger={<span />}
      triggerClassName="hidden"
      hideCancel
      onOpenChange={(open) => !open && close()}
      title="Restore from backup file"
      description={`Import a database dump into ${serviceName}. The current data is not replaced until you confirm the restore.`}
      popupClassName="w-[min(44rem,calc(100vw-2rem))] max-h-[90dvh] overflow-y-auto"
      footer={
        <div className="flex gap-2">
          {phase === 'ready' && job?.status === 'ready' ? (
            <Button
              tone="ghost"
              disabled={busy}
              onClick={() => void cancelRestore()}
            >
              Discard upload
            </Button>
          ) : null}
          <Button
            tone="ghost"
            disabled={busy}
            onClick={close}
          >
            {done ? 'Close' : 'Cancel'}
          </Button>
          {phase === 'validate' ? (
            <Button
              disabled={busy}
              loading={busy}
              onClick={() => void validate()}
            >
              Validate file
            </Button>
          ) : null}
          {phase === 'ready' && job?.status === 'ready' ? (
            <Button
              tone="danger"
              disabled={busy || confirmation !== serviceName}
              loading={busy}
              onClick={() => void startRestore()}
            >
              Restore database
            </Button>
          ) : null}
        </div>
      }
    >
      <div className="grid gap-4">
        {error ? (
          <div
            className="rounded-lg border border-danger/35 bg-danger-soft/20 px-3 py-2 text-supporting text-danger-strong"
            role="alert"
          >
            {error}
          </div>
        ) : null}
        {phase === 'pick' ? (
          <button
            className="grid min-h-48 cursor-pointer place-items-center gap-2 rounded-xl border border-dashed border-border-default bg-surface-2 p-6 text-center transition-colors hover:border-action/50 hover:bg-surface-3 focus-visible:outline-2 focus-visible:outline-focus"
            onClick={() => input.current?.click()}
            onDragOver={(event) => event.preventDefault()}
            onDrop={(event) => {
              event.preventDefault()
              const file = event.dataTransfer.files[0]
              if (file) void selectFile(file)
            }}
            type="button"
          >
            <input
              accept=".dump,.backup,.sql,.gz,.bak,.dmp,.tar"
              className="hidden"
              onChange={(event) => {
                const file = event.target.files?.[0]
                if (file) void selectFile(file)
                event.currentTarget.value = ''
              }}
              ref={input}
              type="file"
            />
            <FileArrowUp
              className="text-action-strong"
              size={34}
            />
            <span className="text-supporting text-text-primary">
              Drop a backup file here or browse to upload
            </span>
            <span className="text-label text-text-tertiary">
              PostgreSQL, MySQL/MariaDB, SQL Server and Oracle formats
            </span>
          </button>
        ) : null}
        {phase === 'uploading' ? (
          <div className="grid gap-3">
            <div className="flex flex-wrap justify-between gap-2">
              <span className="truncate text-supporting text-text-primary">
                {fileName}
              </span>
              <span className="font-technical text-log text-text-tertiary">
                {progress
                  ? `${formatBytes(progress.loaded)} / ${progress.total ? formatBytes(progress.total) : '—'}`
                  : 'Preparing upload…'}
              </span>
            </div>
            <div className="h-2 overflow-hidden rounded-full bg-surface-3">
              <div
                className="h-full bg-action transition-[width]"
                style={{
                  width: progress?.total
                    ? `${Math.min(100, (progress.loaded / progress.total) * 100)}%`
                    : '18%',
                }}
              />
            </div>
            <span className="text-label text-text-tertiary">
              Uploading file. Keep this window open.
            </span>
          </div>
        ) : null}
        {phase === 'validate' && job ? (
          <div className="grid gap-3 rounded-xl bg-surface-2 p-4">
            <div className="flex items-center gap-3">
              <FileArrowUp
                className="text-action-strong"
                size={20}
              />
              <span className="truncate text-supporting text-text-primary">
                {job.source_filename || fileName}
              </span>
            </div>
            <div className="grid gap-3 sm:grid-cols-3">
              <ScheduleFact
                label="SIZE"
                value={formatBytes(job.source_size)}
              />
              <ScheduleFact
                label="FORMAT"
                value={job.source_format || 'Pending validation'}
              />
              <ScheduleFact
                label="TARGET"
                value={serviceName}
              />
            </div>
            <p className="text-label text-text-tertiary">
              Validate the file before making it eligible for restore.
            </p>
          </div>
        ) : null}
        {phase === 'ready' && job ? (
          <div className="grid gap-4">
            <div className="grid gap-3 sm:grid-cols-3">
              <ScheduleFact
                label="FILE"
                value={job.source_filename || fileName}
              />
              <ScheduleFact
                label="SIZE"
                value={formatBytes(job.source_size)}
              />
              <ScheduleFact
                label="FORMAT"
                value={job.source_format}
              />
            </div>
            <div className="grid gap-1 rounded-xl border border-warning/35 bg-warning-soft/20 p-3">
              <span className="flex items-center gap-2 text-supporting text-warning-strong">
                <WarningCircle size={17} />
                This replaces the current database data.
              </span>
              <span className="text-label text-text-secondary">
                The restore may cause downtime and cannot be automatically undone. Type
                the database name to continue.
              </span>
            </div>
            <Field
              id="upload-restore-confirmation"
              label={`Type “${serviceName}” to confirm`}
            >
              <Input
                autoComplete="off"
                value={confirmation}
                onChange={(event) => setConfirmation(event.target.value)}
              />
            </Field>
          </div>
        ) : null}
        {phase === 'restoring' && running ? (
          <div className="flex items-center gap-3 rounded-xl border border-action/35 bg-action-soft/40 p-4">
            <HardDrives
              className="text-action-strong"
              size={20}
            />
            <span className="text-supporting text-text-primary">
              Restore is {job?.status}. This view updates as the operation progresses.
            </span>
          </div>
        ) : null}
        {done && job ? (
          <div
            className={`flex items-start gap-3 rounded-xl border p-4 ${job.status === 'completed' ? 'border-success/35 bg-success-soft/30 text-success-strong' : 'border-danger/35 bg-danger-soft/20 text-danger-strong'}`}
          >
            {job.status === 'completed' ? (
              <CheckCircle size={19} />
            ) : (
              <XCircle size={19} />
            )}
            <div className="grid gap-1">
              <span className="text-supporting">
                {job.status === 'completed'
                  ? 'Restore completed successfully.'
                  : job.error_message || `Restore ${job.status}.`}
              </span>
              {job.error_code ? (
                <span className="font-technical text-log">{job.error_code}</span>
              ) : null}
            </div>
          </div>
        ) : null}
        {finalRestore.isError && selected ? (
          <span className="text-label text-text-tertiary">
            Live restore status is temporarily unavailable.
          </span>
        ) : null}
      </div>
    </Modal>
  )
}

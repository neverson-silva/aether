import {
  AlertDialog,
  Button,
  Card,
  EmptyState,
  Field,
  Input,
  Marker,
  Modal,
  RuntimeStatus,
  Select,
  Skeleton,
  showToast,
  Typography,
} from '@aether/elisyum-ds'
import {
  Archive,
  CheckCircle,
  CloudArrowUp,
  FloppyDisk,
  PencilSimple,
  Plus,
  Trash,
  WifiHigh,
  XCircle,
} from '@phosphor-icons/react'
import { useEffect, useState } from 'react'
import type { DestinationType, S3Destination } from '../api/types'
import { useApps } from '../hooks/use-apps'
import { useCreateS3 } from '../hooks/use-create-s3'
import { useCreateSnapshotSchedule } from '../hooks/use-create-snapshot-schedule'
import { useDeleteS3 } from '../hooks/use-delete-s3'
import { useDeleteSnapshotSchedule } from '../hooks/use-delete-snapshot-schedule'
import { useGoogleConnect } from '../hooks/use-google-connect'
import { useGoogleDisconnect } from '../hooks/use-google-disconnect'
import { useS3Destinations } from '../hooks/use-s3-destinations'
import { useSnapshotSchedules } from '../hooks/use-snapshot-schedules'
import { useSnapshots } from '../hooks/use-snapshots'
import { useTestS3 } from '../hooks/use-test-s3'
import { useUpdateS3 } from '../hooks/use-update-s3'

type DestinationForm = {
  name: string
  type: DestinationType
  endpoint: string
  bucket: string
  region: string
  account_id: string
  access_key: string
  secret_key: string
  google_client_id: string
  google_client_secret: string
}

const emptyForm: DestinationForm = {
  name: '',
  type: 'custom-s3',
  endpoint: '',
  bucket: '',
  region: 'us-east-1',
  account_id: '',
  access_key: '',
  secret_key: '',
  google_client_id: '',
  google_client_secret: '',
}
const providerNames: Record<DestinationType, string> = {
  aws: 'Amazon S3',
  'cloudflare-r2': 'Cloudflare R2',
  minio: 'MinIO',
  'custom-s3': 'S3 compatible',
  'google-drive': 'Google Drive',
}

export function StorageField() {
  const destinations = useS3Destinations()
  const snapshots = useSnapshots()
  const schedules = useSnapshotSchedules()
  const apps = useApps()
  const create = useCreateS3()
  const update = useUpdateS3()
  const remove = useDeleteS3()
  const test = useTestS3()
  const googleConnect = useGoogleConnect()
  const googleDisconnect = useGoogleDisconnect()
  const createScheduleMutation = useCreateSnapshotSchedule()
  const deleteScheduleMutation = useDeleteSnapshotSchedule()
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<S3Destination | null>(null)
  const [form, setForm] = useState<DestinationForm>(emptyForm)
  const [testedId, setTestedId] = useState<string | null>(null)
  const [testErrorId, setTestErrorId] = useState<string | null>(null)
  const [testError, setTestError] = useState('')

  useEffect(() => {
    const url = new URL(window.location.href)
    if (url.searchParams.get('oauth') !== 'google-drive') return
    const status = url.searchParams.get('status') || ''
    if (status === 'connected')
      showToast('Google Drive connected successfully.', 'success')
    else if (status.startsWith('error:'))
      showToast(`Google Drive connection failed: ${status.slice(6)}`, 'error')
    else showToast('Google Drive connection status updated.', 'info')
    url.searchParams.delete('oauth')
    url.searchParams.delete('status')
    window.history.replaceState({}, '', `${url.pathname}${url.search}${url.hash}`)
  }, [])

  useEffect(() => {
    if (!open) return
    if (!editing) {
      setForm(emptyForm)
      return
    }
    setForm({
      name: editing.name,
      type: editing.type,
      endpoint: editing.endpoint || '',
      bucket: editing.bucket || '',
      region: editing.region || 'us-east-1',
      account_id: editing.account_id || '',
      access_key: '',
      secret_key: '',
      google_client_id: editing.google_client_id || '',
      google_client_secret: '',
    })
  }, [editing, open])

  const setField = (field: keyof DestinationForm, value: string) =>
    setForm((current) => ({ ...current, [field]: value }))
  const close = () => {
    setOpen(false)
    setEditing(null)
  }
  const endpoint =
    form.type === 'aws'
      ? `https://s3.${form.region || 'us-east-1'}.amazonaws.com`
      : form.type === 'cloudflare-r2' && form.account_id
        ? `https://${form.account_id}.r2.cloudflarestorage.com`
        : form.endpoint

  const submit = async () => {
    const body: Record<string, string> = {
      name: form.name,
      type: form.type,
      bucket: form.bucket,
      region: form.region,
      account_id: form.account_id,
    }
    if (form.type !== 'google-drive') {
      body.endpoint = endpoint
      if (form.access_key || form.secret_key || !editing) {
        body.access_key = form.access_key
        body.secret_key = form.secret_key
      }
    } else {
      if (form.google_client_id || !editing)
        body.google_client_id = form.google_client_id
      if (form.google_client_secret || !editing)
        body.google_client_secret = form.google_client_secret
    }
    try {
      if (editing) {
        await update.mutateAsync({ id: editing.id, body })
        showToast(`Destination “${form.name}” updated successfully.`, 'success')
      } else {
        const created = await create.mutateAsync(body)
        showToast(`Destination “${form.name}” created successfully.`, 'success')
        if (form.type === 'google-drive') {
          showToast('Redirecting to Google Drive authorization…', 'info')
          await googleConnect.mutateAsync(created.id)
        }
      }
      close()
    } catch (error) {
      showToast(
        error instanceof Error
          ? error.message
          : 'Could not save the destination. Try again.',
        'error',
      )
    }
  }

  const runTest = (id: string) => {
    setTestedId(null)
    setTestErrorId(null)
    setTestError('')
    test.mutate(id, {
      onSuccess: () => {
        setTestedId(id)
        showToast('Connection verified successfully.', 'success')
      },
      onError: (error) => {
        setTestErrorId(id)
        setTestError(
          error instanceof Error
            ? error.message
            : 'The connection test failed. Verify the endpoint and credentials.',
        )
        showToast(
          error instanceof Error
            ? error.message
            : 'The connection test failed. Verify the endpoint and credentials.',
          'error',
        )
      },
    })
  }

  const connectGoogle = (destination: S3Destination) => {
    showToast(`Connecting “${destination.name}” to Google Drive…`, 'info')
    googleConnect.mutate(destination.id, {
      onError: (error) =>
        showToast(
          error instanceof Error
            ? error.message
            : 'Could not start Google Drive authorization.',
          'error',
        ),
    })
  }

  const disconnectGoogle = (destination: S3Destination) => {
    googleDisconnect.mutate(destination.id, {
      onSuccess: () =>
        showToast(`Google Drive disconnected from “${destination.name}”.`, 'success'),
      onError: (error) =>
        showToast(
          error instanceof Error ? error.message : 'Could not disconnect Google Drive.',
          'error',
        ),
    })
  }

  const deleteDestination = (destination: S3Destination) => {
    remove.mutate(destination.id, {
      onSuccess: () =>
        showToast(`Destination “${destination.name}” deleted.`, 'success'),
      onError: (error) =>
        showToast(
          error instanceof Error ? error.message : 'Could not delete the destination.',
          'error',
        ),
    })
  }

  const createSchedule = (body: {
    app_id: string
    volume: string
    name_prefix: string
    cron: string
    retention: number
    enabled: boolean
  }) => {
    createScheduleMutation.mutate(body, {
      onSuccess: () => showToast('Snapshot schedule created successfully.', 'success'),
      onError: (error) =>
        showToast(
          error instanceof Error
            ? error.message
            : 'Could not create the snapshot schedule.',
          'error',
        ),
    })
  }

  const deleteSchedule = (id: string) => {
    deleteScheduleMutation.mutate(id, {
      onSuccess: () => showToast('Snapshot schedule deleted.', 'success'),
      onError: (error) =>
        showToast(
          error instanceof Error
            ? error.message
            : 'Could not delete the snapshot schedule.',
          'error',
        ),
    })
  }

  return (
    <div className="grid gap-7">
      <header className="flex flex-wrap items-end justify-between gap-5">
        <div className="grid gap-3">
          <div className="flex items-center gap-3 text-label tracking-[0.16em] text-text-subtle">
            <CloudArrowUp size={18} />
            PROTECTION / DESTINATIONS
          </div>
          <Typography
            as="h1"
            role="page-title"
          >
            Storage field
          </Typography>
          <Typography
            className="max-w-2xl"
            role="supporting"
          >
            External destinations and snapshots used to protect operational data.
          </Typography>
        </div>
        <Modal
          open={open}
          onOpenChange={(value) => {
            setOpen(value)
            if (!value) setEditing(null)
          }}
          trigger={
            <>
              <Plus size={17} />
              New destination
            </>
          }
          title={editing ? 'Edit destination' : 'New destination'}
          description="Declare the storage boundary used by backups and artifacts."
          footer={
            <Button
              disabled={
                !form.name.trim() ||
                create.isPending ||
                update.isPending ||
                googleConnect.isPending
              }
              onClick={() => void submit()}
            >
              <FloppyDisk size={16} />
              {create.isPending || update.isPending
                ? 'Saving...'
                : editing
                  ? 'Save destination'
                  : 'Create destination'}
            </Button>
          }
        >
          <DestinationForm
            editing={editing}
            form={form}
            setField={setField}
            onConnect={() => editing && connectGoogle(editing)}
            connecting={googleConnect.isPending}
          />
        </Modal>
      </header>
      <section className="grid gap-4">
        <div className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <Marker tone="accent" />
            <span className="text-label tracking-[0.12em] text-text-subtle">
              DESTINATION INVENTORY / {destinations.data?.length ?? 0}
            </span>
          </div>
          <span className="font-technical text-log text-text-subtle">
            CREDENTIALS REMAIN PROTECTED
          </span>
        </div>
        {destinations.isLoading ? (
          <Skeleton className="h-64 rounded-2xl" />
        ) : destinations.isError ? (
          <EmptyState
            title="Storage field unavailable"
            description="The control plane did not return storage destinations."
            action={
              <Button
                tone="neutral"
                onClick={() => destinations.refetch()}
              >
                Retry request
              </Button>
            }
          />
        ) : destinations.data?.length ? (
          <Card className="grid gap-2 rounded-2xl border-border-subtle bg-surface-1 p-3">
            {destinations.data.map((destination) => (
              <DestinationRow
                key={destination.id}
                destination={destination}
                tested={testedId === destination.id}
                testing={test.isPending && test.variables === destination.id}
                testError={testErrorId === destination.id ? testError : ''}
                onTest={() => runTest(destination.id)}
                onEdit={() => {
                  setEditing(destination)
                  setOpen(true)
                }}
                onDelete={() => deleteDestination(destination)}
                onConnect={() => connectGoogle(destination)}
                onDisconnect={() => disconnectGoogle(destination)}
              />
            ))}
          </Card>
        ) : (
          <EmptyState
            title="No destinations configured"
            description="Create a destination to make backups and artifacts durable."
            icon={<CloudArrowUp size={28} />}
            action={
              <Button
                onClick={() => {
                  setEditing(null)
                  setOpen(true)
                }}
              >
                <Plus size={17} />
                Create destination
              </Button>
            }
          />
        )}
      </section>
      <section className="grid gap-4 xl:grid-cols-2">
        <SnapshotWorkplane
          title="Snapshots"
          icon={<Archive size={18} />}
          items={(snapshots.data ?? []).map((item) => ({
            id: item.id,
            name: item.name,
            meta: `${item.volume} · ${formatBytes(item.size)}`,
            state: item.created_at,
          }))}
          loading={snapshots.isLoading}
        />
        <ScheduleWorkplane
          apps={apps.data ?? []}
          schedules={schedules.data ?? []}
          loading={schedules.isLoading}
          creating={createScheduleMutation.isPending}
          onCreate={createSchedule}
          onDelete={deleteSchedule}
        />
      </section>
    </div>
  )
}

function DestinationForm({
  form,
  setField,
  editing,
  onConnect,
  connecting,
}: {
  form: DestinationForm
  setField: (field: keyof DestinationForm, value: string) => void
  editing: S3Destination | null
  onConnect: () => void
  connecting: boolean
}) {
  const customEndpoint = form.type === 'minio' || form.type === 'custom-s3'
  const google = form.type === 'google-drive'
  return (
    <div className="grid gap-4">
      <Field
        id="destination-name"
        label="Name"
        required
      >
        <Input
          value={form.name}
          onChange={(event) => setField('name', event.target.value)}
          placeholder="production-backups"
        />
      </Field>
      <Field
        id="destination-type"
        label="Destination type"
        required
      >
        <Select
          value={form.type}
          onChange={(event) => setField('type', event.target.value as DestinationType)}
        >
          {(Object.keys(providerNames) as DestinationType[]).map((type) => (
            <option
              key={type}
              value={type}
            >
              {providerNames[type]}
            </option>
          ))}
        </Select>
      </Field>
      <div className="grid gap-4 rounded-xl bg-surface-2 p-4">
        <Typography role="supporting">
          {google
            ? 'Connect Google Drive through the organization OAuth boundary.'
            : 'Credentials are used only by the control plane and are never rendered in inventory.'}
        </Typography>
        {!google ? (
          <>
            <Field
              id="destination-bucket"
              label="Bucket"
            >
              <Input
                value={form.bucket}
                onChange={(event) => setField('bucket', event.target.value)}
                placeholder="aether-backups"
              />
            </Field>
            {customEndpoint ? (
              <Field
                id="destination-endpoint"
                label="Endpoint"
              >
                <Input
                  value={form.endpoint}
                  onChange={(event) => setField('endpoint', event.target.value)}
                  placeholder="https://storage.example.com"
                />
              </Field>
            ) : null}
            {form.type === 'cloudflare-r2' ? (
              <Field
                id="destination-account"
                label="Account ID"
              >
                <Input
                  value={form.account_id}
                  onChange={(event) => setField('account_id', event.target.value)}
                  placeholder="Account identifier"
                />
              </Field>
            ) : null}
            {form.type === 'aws' ? (
              <Field
                id="destination-region"
                label="Region"
              >
                <Input
                  value={form.region}
                  onChange={(event) => setField('region', event.target.value)}
                  placeholder="us-east-1"
                />
              </Field>
            ) : null}
            <div className="grid gap-4 sm:grid-cols-2">
              <Field
                id="destination-access"
                label="Access key"
              >
                <Input
                  value={form.access_key}
                  onChange={(event) => setField('access_key', event.target.value)}
                />
              </Field>
              <Field
                id="destination-secret"
                label="Secret key"
              >
                <Input
                  type="password"
                  value={form.secret_key}
                  onChange={(event) => setField('secret_key', event.target.value)}
                />
              </Field>
            </div>
          </>
        ) : (
          <>
            <div className="grid gap-2 rounded-lg border border-border-subtle bg-surface-1 p-3">
              <span className="text-label text-text-tertiary">REDIRECT URI</span>
              <code className="break-all font-technical text-log text-action-strong">
                {window.location.origin}/api/v1/s3-destinations/google/callback
              </code>
              <span className="text-supporting text-text-tertiary">
                Register this exact URI in your Google OAuth client.
              </span>
            </div>
            <Field
              id="google-client-id"
              label="Google client ID"
            >
              <Input
                value={form.google_client_id}
                onChange={(event) => setField('google_client_id', event.target.value)}
              />
            </Field>
            <Field
              id="google-client-secret"
              label={
                editing
                  ? 'Google client secret (leave blank to keep)'
                  : 'Google client secret'
              }
            >
              <Input
                type="password"
                value={form.google_client_secret}
                onChange={(event) =>
                  setField('google_client_secret', event.target.value)
                }
              />
            </Field>
            <Field
              id="google-bucket"
              label="Bucket name"
              description="Backups and artifacts will be stored inside this Google Drive folder."
            >
              <Input
                value={form.bucket}
                onChange={(event) => setField('bucket', event.target.value)}
                placeholder="aether-backups"
              />
            </Field>
            {editing ? (
              <Button
                tone="neutral"
                disabled={connecting}
                onClick={onConnect}
              >
                {connecting
                  ? 'Connecting…'
                  : editing.oauth_status === 'reauth_required'
                    ? 'Reconnect Google Drive'
                    : 'Connect Google Drive'}
              </Button>
            ) : null}
          </>
        )}
      </div>
    </div>
  )
}

function DestinationRow({
  destination,
  tested,
  testing,
  testError,
  onTest,
  onEdit,
  onDelete,
  onConnect,
  onDisconnect,
}: {
  destination: S3Destination
  tested: boolean
  testing: boolean
  testError: string
  onTest: () => void
  onEdit: () => void
  onDelete: () => void
  onConnect: () => void
  onDisconnect: () => void
}) {
  const google = destination.type === 'google-drive'
  return (
    <div className="grid gap-4 rounded-xl border border-transparent bg-surface-2 p-4 transition-[background-color,border-color] hover:border-border-default hover:bg-surface-3 sm:grid-cols-[minmax(0,1fr)_auto]">
      <div className="grid min-w-0 gap-2">
        <div className="flex items-center gap-3">
          <span className="grid size-10 place-items-center rounded-xl bg-surface-1 text-text-tertiary">
            <CloudArrowUp size={19} />
          </span>
          <div className="grid min-w-0 gap-1">
            <span className="truncate text-supporting text-text-primary">
              {destination.name}
            </span>
            <span className="truncate font-technical text-log text-text-subtle">
              {providerNames[destination.type]} /{' '}
              {google ? 'OAuth managed' : destination.endpoint || 'Endpoint pending'}
            </span>
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-3 pl-[3.25rem] font-technical text-log text-text-subtle">
          <span>BUCKET / {destination.bucket || 'UNSET'}</span>
          {tested ? (
            <span className="flex items-center gap-1 text-success">
              <CheckCircle size={14} /> CONNECTION VERIFIED
            </span>
          ) : testError ? (
            <span className="flex items-center gap-1 text-danger">
              <XCircle size={14} /> {testError}
            </span>
          ) : null}
        </div>
      </div>
      <div className="flex flex-wrap items-center justify-end gap-2">
        <RuntimeStatus
          status={
            google
              ? destination.oauth_status === 'connected'
                ? 'healthy'
                : 'degraded'
              : 'healthy'
          }
          label={google ? destination.oauth_status || 'NOT CONNECTED' : 'CONFIGURED'}
        />
        {google ? (
          <Button
            size="sm"
            tone="ghost"
            onClick={
              destination.oauth_status === 'connected' ? onDisconnect : onConnect
            }
          >
            {destination.oauth_status === 'connected' ? 'Disconnect' : 'Connect'}
          </Button>
        ) : (
          <Button
            size="sm"
            tone="ghost"
            disabled={testing}
            onClick={onTest}
          >
            <WifiHigh size={15} />
            {testing ? 'Testing...' : 'Test'}
          </Button>
        )}
        <Button
          size="sm"
          tone="ghost"
          onClick={onEdit}
        >
          <PencilSimple size={15} />
          Edit
        </Button>
        <AlertDialog
          trigger={<Trash size={16} />}
          title={`Delete ${destination.name}?`}
          description="This removes the destination configuration. Existing stored data is not deleted."
          confirmLabel="Delete destination"
          onConfirm={onDelete}
        />
      </div>
    </div>
  )
}

function SnapshotWorkplane({
  title,
  icon,
  items,
  loading,
}: {
  title: string
  icon: React.ReactNode
  items: Array<{ id: string; name: string; meta: string; state: string }>
  loading: boolean
}) {
  return (
    <section className="grid gap-4">
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <span className="text-action-strong">{icon}</span>
          <Typography
            as="h2"
            role="section-title"
          >
            {title}
          </Typography>
        </div>
        <span className="font-technical text-log text-text-subtle">
          {items.length} ITEMS
        </span>
      </div>
      {loading ? (
        <Skeleton className="h-40 rounded-2xl" />
      ) : items.length ? (
        <Card className="grid gap-2 rounded-2xl border-border-subtle bg-surface-1 p-3">
          {items.map((item) => (
            <div
              className="grid gap-1 rounded-xl bg-surface-2 p-4"
              key={item.id}
            >
              <span className="text-supporting text-text-primary">{item.name}</span>
              <span className="font-technical text-log text-text-subtle">
                {item.meta} · {item.state}
              </span>
            </div>
          ))}
        </Card>
      ) : (
        <EmptyState
          title={`No ${title.toLowerCase()} available`}
          description="The control plane will surface this protection state when it exists."
        />
      )}
    </section>
  )
}

function ScheduleWorkplane({
  apps,
  schedules,
  loading,
  creating,
  onCreate,
  onDelete,
}: {
  apps: Array<{ id: string; name: string }>
  schedules: Array<{
    id: string
    volume: string
    name_prefix: string
    cron: string
    enabled: boolean
  }>
  loading: boolean
  creating: boolean
  onCreate: (body: {
    app_id: string
    volume: string
    name_prefix: string
    cron: string
    retention: number
    enabled: boolean
  }) => void
  onDelete: (id: string) => void
}) {
  const [appId, setAppId] = useState('')
  const [volume, setVolume] = useState('')
  const [cron, setCron] = useState('@daily')
  const [retention, setRetention] = useState('7')
  const submit = () => {
    if (!(appId && volume.trim())) return
    onCreate({
      app_id: appId,
      volume: volume.trim(),
      name_prefix: 'scheduled',
      cron,
      retention: Number(retention) || 7,
      enabled: true,
    })
    setVolume('')
  }
  return (
    <section className="grid gap-4">
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <span className="text-action-strong">
            <CheckCircle size={18} />
          </span>
          <Typography
            as="h2"
            role="section-title"
          >
            Snapshot schedules
          </Typography>
        </div>
        <span className="font-technical text-log text-text-subtle">
          {schedules.length} ITEMS
        </span>
      </div>
      <Card className="grid gap-4 rounded-2xl border-border-subtle bg-surface-1 p-4">
        <div className="grid gap-3">
          <Select
            aria-label="Service"
            value={appId}
            onChange={(event) => setAppId(event.target.value)}
          >
            <option value="">Choose service</option>
            {apps.map((app) => (
              <option
                key={app.id}
                value={app.id}
              >
                {app.name}
              </option>
            ))}
          </Select>
          <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto_auto]">
            <Input
              aria-label="Volume"
              placeholder="Volume path"
              value={volume}
              onChange={(event) => setVolume(event.target.value)}
            />
            <Select
              aria-label="Frequency"
              value={cron}
              onChange={(event) => setCron(event.target.value)}
            >
              <option value="@daily">Daily</option>
              <option value="@weekly">Weekly</option>
              <option value="@hourly">Hourly</option>
              <option value="0 3 * * *">Daily at 03:00</option>
              <option value="30 22 * * 5">Weekly Friday</option>
            </Select>
            <Button
              disabled={!(appId && volume.trim()) || creating}
              onClick={submit}
            >
              <Plus size={16} />
              {creating ? 'Scheduling...' : 'Schedule'}
            </Button>
          </div>
        </div>
        {loading ? (
          <Skeleton className="h-24 rounded-xl" />
        ) : schedules.length ? (
          <div className="grid gap-2">
            {schedules.map((schedule) => (
              <div
                className="flex flex-wrap items-center gap-3 rounded-xl bg-surface-2 p-3"
                key={schedule.id}
              >
                <Archive
                  className={schedule.enabled ? 'text-success' : 'text-text-tertiary'}
                  size={16}
                />
                <span className="min-w-0 flex-1 truncate text-supporting text-text-primary">
                  {schedule.volume}
                </span>
                <span className="font-technical text-log text-text-subtle">
                  {schedule.cron}
                </span>
                <AlertDialog
                  trigger={<Trash size={15} />}
                  title={`Delete ${schedule.volume} schedule?`}
                  description="This removes the schedule. Existing snapshots remain available."
                  confirmLabel="Delete schedule"
                  onConfirm={() => onDelete(schedule.id)}
                />
              </div>
            ))}
          </div>
        ) : (
          <EmptyState
            title="No schedules yet"
            description="Create a schedule to automate snapshots for a service."
          />
        )}
      </Card>
    </section>
  )
}

function formatBytes(bytes: number) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KiB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MiB`
}

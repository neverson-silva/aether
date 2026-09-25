import {
  Badge,
  Button,
  Card,
  CopyButton,
  EmptyState,
  Field,
  Input,
  Modal,
  RuntimeStatus,
  Skeleton,
  Typography,
} from '@aether/elisyum-ds'
import {
  ArrowsClockwise,
  CloudArrowUp,
  MagnifyingGlass,
  Package,
  Play,
  Plus,
  TrashSimple,
} from '@phosphor-icons/react'
import { useState } from 'react'
import type { RegistryImage, RegistryMirror } from '../hooks/types'
import { useCreateMirror } from '../hooks/use-create-mirror'
import { useDeleteMirror } from '../hooks/use-delete-mirror'
import { useMirrors } from '../hooks/use-mirrors'
import { useRegistryImages } from '../hooks/use-registry-images'
import { useRegistrySettings } from '../hooks/use-registry-settings'
import { useRunMirror } from '../hooks/use-run-mirror'
import { useToggleRegistry } from '../hooks/use-toggle-registry'

export function RegistryField() {
  const settings = useRegistrySettings()
  const images = useRegistryImages(Boolean(settings.data?.enabled))
  const mirrors = useMirrors()
  const toggle = useToggleRegistry()
  const createMirror = useCreateMirror()
  const runMirror = useRunMirror()
  const deleteMirror = useDeleteMirror()
  const [search, setSearch] = useState('')
  const [imagesRequested, setImagesRequested] = useState(false)
  const [mirrorOpen, setMirrorOpen] = useState(false)
  const [deletingMirror, setDeletingMirror] = useState<RegistryMirror | null>(null)
  const [name, setName] = useState('')
  const [source, setSource] = useState('')
  const [destination, setDestination] = useState('')
  const [schedule, setSchedule] = useState('')

  if (settings.isLoading) return <PlatformSkeleton />
  if (settings.isError || !settings.data)
    return (
      <EmptyState
        title="Registry context unavailable"
        description="The control plane did not return registry state for this organization."
        action={
          <Button
            tone="neutral"
            onClick={() => settings.refetch()}
          >
            Retry request
          </Button>
        }
      />
    )

  const registry = settings.data
  const loadImages = () => {
    setImagesRequested(true)
    void images.refetch()
  }
  const filteredImages = filterImages(images.data ?? [], search)
  const submitMirror = () => {
    if (!(name.trim() && source.trim() && destination.trim())) return
    createMirror.mutate(
      {
        name: name.trim(),
        source: source.trim(),
        dest: destination.trim(),
        schedule: schedule.trim() || undefined,
      },
      {
        onSuccess: () => {
          setMirrorOpen(false)
          setName('')
          setSource('')
          setDestination('')
          setSchedule('')
        },
      },
    )
  }
  const confirmDelete = () => {
    if (!deletingMirror) return
    deleteMirror.mutate(deletingMirror.id, { onSuccess: () => setDeletingMirror(null) })
  }

  return (
    <div className="grid min-w-0 gap-8">
      <header className="flex min-w-0 flex-wrap items-end justify-between gap-5">
        <div className="grid min-w-0 gap-3">
          <div className="flex items-center gap-3 text-label tracking-[0.16em] text-text-subtle">
            <Package size={17} />
            PLATFORM / IMAGE SUPPLY
          </div>
          <Typography
            as="h1"
            role="page-title"
          >
            Registry field
          </Typography>
          <Typography role="supporting">
            Private image storage, runtime reachability and controlled replication for
            this organization.
          </Typography>
        </div>
        <Button
          tone="neutral"
          disabled={toggle.isPending}
          onClick={() => toggle.mutate(!registry.enabled)}
        >
          {toggle.isPending
            ? 'Updating…'
            : registry.enabled
              ? 'Disable registry'
              : 'Enable registry'}
        </Button>
      </header>
      <RegistrySummary registry={registry} />
      <section
        className="grid min-w-0 gap-4"
        aria-labelledby="stored-images-title"
      >
        <div className="flex min-w-0 flex-wrap items-end justify-between gap-4">
          <div className="grid gap-1">
            <div className="flex items-center gap-3">
              <Typography
                as="h2"
                id="stored-images-title"
                role="section-title"
              >
                Stored images
              </Typography>
              <span className="font-technical text-log text-text-tertiary">
                {images.data?.length ?? 0}{' '}
                {images.data?.length === 1 ? 'image' : 'images'}
              </span>
            </div>
            <Typography role="supporting">
              Images available to the control plane runtime.
            </Typography>
          </div>
          <div className="flex w-full flex-wrap items-center justify-end gap-2 sm:w-auto">
            <div className="relative min-w-[14rem] flex-1 sm:flex-none">
              <MagnifyingGlass
                aria-hidden="true"
                className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-text-tertiary"
                size={16}
              />
              <Input
                aria-label="Search stored images"
                className="pl-9"
                onChange={(event) => setSearch(event.target.value)}
                placeholder="Search images"
                value={search}
              />
            </div>
            <Button
              tone="neutral"
              disabled={!registry.enabled || images.isFetching}
              onClick={loadImages}
            >
              <ArrowsClockwise
                className={images.isFetching ? 'motion-safe:animate-spin' : ''}
                size={16}
              />
              {images.isFetching ? 'Refreshing…' : 'Refresh index'}
            </Button>
          </div>
        </div>
        {!registry.enabled ? (
          <EmptyState
            title="Registry is disabled"
            description="Enable the registry to inspect stored images."
            icon={<CloudArrowUp size={28} />}
          />
        ) : images.isFetching ? (
          <Card className="grid gap-2 rounded-2xl border-border-subtle bg-surface-1 p-3">
            <Skeleton className="h-14 rounded-xl" />
            <Skeleton className="h-14 rounded-xl" />
            <Skeleton className="h-14 rounded-xl" />
          </Card>
        ) : filteredImages.length ? (
          <ImageCollection images={filteredImages} />
        ) : images.data?.length ? (
          <EmptyState
            title="No matching images"
            description="Try a different repository, tag or image identifier."
          />
        ) : (
          <EmptyState
            title={imagesRequested ? 'No images indexed' : 'No images indexed yet'}
            description="Images stored by deployed services will appear here."
            icon={<CloudArrowUp size={28} />}
          />
        )}
      </section>
      <section
        className="grid min-w-0 gap-4"
        aria-labelledby="registry-mirrors-title"
      >
        <div className="flex min-w-0 flex-wrap items-end justify-between gap-4">
          <div className="grid gap-1">
            <Typography
              as="h2"
              id="registry-mirrors-title"
              role="section-title"
            >
              Registry mirrors
            </Typography>
            <Typography role="supporting">
              Move selected image namespaces between registries on demand or on a
              declared schedule.
            </Typography>
          </div>
          <Button onClick={() => setMirrorOpen(true)}>
            <Plus size={17} />
            Create mirror
          </Button>
        </div>
        {mirrors.isLoading ? (
          <Skeleton className="h-24 rounded-2xl" />
        ) : mirrors.data?.length ? (
          <Card className="grid min-w-0 gap-2 rounded-2xl border-border-subtle bg-surface-1 p-3">
            {mirrors.data.map((mirror) => (
              <MirrorRow
                key={mirror.id}
                mirror={mirror}
                onRun={() => runMirror.mutate(mirror.id)}
                onDelete={() => setDeletingMirror(mirror)}
                running={runMirror.isPending}
              />
            ))}
          </Card>
        ) : (
          <Card className="rounded-2xl border-border-subtle bg-surface-1 p-5">
            <div className="grid gap-1">
              <span className="text-supporting font-medium text-text-primary">
                No mirror routes
              </span>
              <span className="text-supporting text-text-tertiary">
                Create a mirror when an image namespace needs controlled replication.
              </span>
            </div>
          </Card>
        )}
      </section>
      <Modal
        open={mirrorOpen}
        onOpenChange={setMirrorOpen}
        title="Create registry mirror"
        description="Define a source and destination for controlled image replication."
        triggerClassName="hidden"
        trigger={<span />}
        footer={
          <Button
            disabled={
              createMirror.isPending ||
              !name.trim() ||
              !source.trim() ||
              !destination.trim()
            }
            onClick={submitMirror}
          >
            {createMirror.isPending ? 'Creating…' : 'Create mirror'}
          </Button>
        }
      >
        <div className="grid gap-4">
          <Field
            id="mirror-name"
            label="Name"
            required
          >
            <Input
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="Production mirror"
            />
          </Field>
          <Field
            id="mirror-source"
            label="Source registry"
            required
          >
            <Input
              value={source}
              onChange={(event) => setSource(event.target.value)}
              placeholder="registry.example.com/team"
            />
          </Field>
          <Field
            id="mirror-destination"
            label="Destination registry"
            required
          >
            <Input
              value={destination}
              onChange={(event) => setDestination(event.target.value)}
              placeholder="registry.internal/team"
            />
          </Field>
          <Field
            id="mirror-schedule"
            label="Schedule"
          >
            <Input
              value={schedule}
              onChange={(event) => setSchedule(event.target.value)}
              placeholder="0 * * * *"
            />
          </Field>
          {createMirror.isError ? (
            <p
              className="text-supporting text-danger"
              role="alert"
            >
              The mirror could not be created. Verify the registry endpoints.
            </p>
          ) : null}
        </div>
      </Modal>
      <Modal
        open={Boolean(deletingMirror)}
        onOpenChange={(open) => !open && setDeletingMirror(null)}
        title="Delete registry mirror"
        description="This removes the replication route and its schedule."
        triggerClassName="hidden"
        trigger={<span />}
        footer={
          <Button
            disabled={deleteMirror.isPending}
            onClick={confirmDelete}
            tone="danger"
          >
            {deleteMirror.isPending ? 'Deleting…' : 'Delete mirror'}
          </Button>
        }
      >
        <p className="text-body text-text-secondary">
          Delete{' '}
          <span className="font-technical text-text-primary">
            {deletingMirror?.name}
          </span>
          ?
        </p>
      </Modal>
    </div>
  )
}

function RegistrySummary({
  registry,
}: {
  registry: {
    enabled: boolean
    host: string
    port: number
    container_id: string
    status: string
  }
}) {
  const status = toRuntimeStatus(registry.status)
  const label = registry.status || (registry.enabled ? 'running' : 'stopped')
  return (
    <Card className="grid min-w-0 gap-5 rounded-2xl border-border-subtle bg-surface-1 p-5">
      <div className="flex min-w-0 flex-wrap items-start justify-between gap-4">
        <div className="flex min-w-0 items-center gap-3">
          <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-surface-2 text-action">
            <Package size={19} />
          </span>
          <div className="grid min-w-0 gap-1">
            <Typography
              as="h2"
              role="section-title"
            >
              Private image registry
            </Typography>
            <span className="truncate font-technical text-log text-text-secondary">
              {registry.host}:{registry.port}
            </span>
          </div>
        </div>
        <RuntimeStatus
          status={status}
          label={label}
        />
      </div>
      <div className="grid min-w-0 gap-4 border-t border-border-subtle pt-4 sm:grid-cols-3 sm:gap-0">
        <Definition
          label="STATUS"
          value={label}
        />
        <Definition
          label="CONTAINER"
          value={registry.container_id || 'Not assigned'}
        />
        <Definition
          label="ENDPOINT"
          value={`${registry.host}:${registry.port}`}
        />
      </div>
    </Card>
  )
}

function ImageCollection({ images }: { images: RegistryImage[] }) {
  return (
    <Card className="min-w-0 overflow-hidden rounded-2xl border-border-subtle bg-surface-1">
      <div className="hidden grid-cols-[minmax(0,2.2fr)_minmax(7rem,0.8fr)_7rem_minmax(7rem,1fr)_auto] gap-4 border-b border-border-subtle px-4 py-3 text-label tracking-[0.08em] text-text-tertiary md:grid">
        <span>REPOSITORY</span>
        <span>TAG</span>
        <span>SIZE</span>
        <span>IMAGE ID</span>
        <span className="sr-only">ACTIONS</span>
      </div>
      <div className="divide-y divide-border-subtle">
        {images.map((image) => (
          <ImageRow
            image={image}
            key={`${image.id}-${image.repo}-${image.tag}`}
          />
        ))}
      </div>
    </Card>
  )
}

function ImageRow({ image }: { image: RegistryImage }) {
  const reference = `${image.repo}:${image.tag}`
  return (
    <div className="grid min-w-0 gap-3 px-4 py-4 transition-colors hover:bg-surface-2 md:grid-cols-[minmax(0,2.2fr)_minmax(7rem,0.8fr)_7rem_minmax(7rem,1fr)_auto] md:items-center md:gap-4">
      <div className="grid min-w-0 gap-1">
        <span
          className="truncate text-supporting font-medium text-text-primary"
          title={reference}
        >
          {image.repo}
        </span>
        <span
          className="truncate font-technical text-log text-text-tertiary md:hidden"
          title={reference}
        >
          {reference}
        </span>
      </div>
      <span
        className="truncate font-technical text-log text-text-secondary"
        title={image.tag}
      >
        {image.tag}
      </span>
      <span className="font-technical text-log text-text-secondary">
        {formatBytes(image.size)}
      </span>
      <span
        className="truncate font-technical text-log text-text-tertiary"
        title={image.id}
      >
        {image.id}
      </span>
      <div className="flex items-center justify-end gap-1">
        <CopyButton
          label={`Copy ${reference}`}
          value={reference}
        />
        <CopyButton
          label={`Copy image ID ${image.id}`}
          value={image.id}
        />
      </div>
    </div>
  )
}

function Definition({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid min-w-0 gap-1 sm:px-4 sm:first:pl-0 sm:last:pr-0">
      <span className="text-label text-text-tertiary">{label}</span>
      <span
        className="truncate font-technical text-log text-text-secondary"
        title={value}
      >
        {value}
      </span>
    </div>
  )
}

function MirrorRow({
  mirror,
  onRun,
  onDelete,
  running,
}: {
  mirror: RegistryMirror
  onRun: () => void
  onDelete: () => void
  running: boolean
}) {
  const status = mirror.status.toLowerCase()
  return (
    <div className="grid min-w-0 gap-4 rounded-xl bg-surface-2 p-4 lg:grid-cols-[minmax(0,1fr)_auto_auto] lg:items-center">
      <div className="grid min-w-0 gap-1">
        <span className="truncate text-supporting text-text-primary">
          {mirror.name}
        </span>
        <span className="truncate font-technical text-log text-text-subtle">
          {mirror.source} → {mirror.dest}
        </span>
        <span className="font-technical text-log text-text-tertiary">
          {mirror.schedule || 'On demand'} · Last run {mirror.last_run || 'Never'}
        </span>
      </div>
      <Badge
        tone={
          status === 'success' || status === 'ready'
            ? 'success'
            : status === 'failed' || status === 'error'
              ? 'danger'
              : 'neutral'
        }
      >
        {mirror.status}
      </Badge>
      <div className="flex justify-end gap-2">
        <Button
          aria-label={`Run ${mirror.name}`}
          disabled={running}
          onClick={onRun}
          tone="ghost"
        >
          <Play size={17} />
        </Button>
        <Button
          aria-label={`Delete ${mirror.name}`}
          onClick={onDelete}
          tone="ghost"
        >
          <TrashSimple size={17} />
        </Button>
      </div>
    </div>
  )
}

function filterImages(images: RegistryImage[], search: string) {
  const term = search.trim().toLowerCase()
  if (!term) return images
  return images.filter((image) =>
    `${image.repo}:${image.tag} ${image.id}`.toLowerCase().includes(term),
  )
}

function toRuntimeStatus(
  status: string,
): 'healthy' | 'deploying' | 'degraded' | 'failed' | 'stopped' | 'unknown' {
  if (status === 'running' || status === 'healthy') return 'healthy'
  if (status === 'starting' || status === 'deploying') return 'deploying'
  if (status === 'failed' || status === 'error') return 'failed'
  if (status === 'stopped' || status === 'disabled') return 'stopped'
  return 'unknown'
}

function formatBytes(value: number) {
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${Math.round(value / 1024)} KB`
  if (value < 1024 * 1024 * 1024) return `${(value / (1024 * 1024)).toFixed(1)} MB`
  return `${(value / (1024 * 1024 * 1024)).toFixed(1)} GB`
}

function PlatformSkeleton() {
  return (
    <div className="grid gap-4">
      <Skeleton className="h-32 rounded-2xl" />
      <Skeleton className="h-72 rounded-2xl" />
      <Skeleton className="h-48 rounded-2xl" />
    </div>
  )
}

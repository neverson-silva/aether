import {
  Badge,
  Button,
  Card,
  CodeEditorLite,
  EmptyState,
  Modal,
  RuntimeStatus,
  Skeleton,
  Typography,
} from '@aether/elisyum-ds'
import {
  ArrowClockwise,
  FileCode,
  FolderOpen,
  LockKey,
  Play,
  TrashSimple,
} from '@phosphor-icons/react'
import { useEffect, useMemo, useState } from 'react'
import { useTraefikFile } from '../hooks/use-traefik-file'
import {
  useDeleteTraefikFile,
  useRestartTraefik,
  useSaveTraefikFile,
} from '../hooks/use-traefik-file-actions'
import { useTraefikFiles, useTraefikStatus } from '../hooks/use-traefik-filesystem'

export function TraefikField() {
  const files = useTraefikFiles()
  const status = useTraefikStatus()
  const [selectedPath, setSelectedPath] = useState('traefik.yml')
  const [content, setContent] = useState('')
  const [deleteOpen, setDeleteOpen] = useState(false)
  const file = useTraefikFile(selectedPath)
  const save = useSaveTraefikFile()
  const remove = useDeleteTraefikFile()
  const restart = useRestartTraefik()
  const selectedEntry = useMemo(
    () => files.data?.find((entry) => entry.path === selectedPath),
    [files.data, selectedPath],
  )

  useEffect(() => setContent(file.data?.content ?? ''), [file.data?.content])
  if (files.isLoading || status.isLoading) return <TraefikSkeleton />
  if (files.isError || status.isError || !status.data)
    return (
      <EmptyState
        title="Ingress context unavailable"
        description="The control plane did not return the protected Traefik file system."
        action={
          <Button
            tone="neutral"
            onClick={() => {
              void files.refetch()
              void status.refetch()
            }}
          >
            Retry request
          </Button>
        }
      />
    )
  const entries = files.data ?? []
  const onSave = () => save.mutate({ path: selectedPath, content })
  const onDelete = () =>
    remove.mutate(selectedPath, {
      onSuccess: () => {
        setDeleteOpen(false)
        setSelectedPath('traefik.yml')
      },
    })
  return (
    <div className="grid gap-7">
      <header className="flex flex-wrap items-end justify-between gap-5">
        <div className="grid gap-3">
          <div className="flex items-center gap-3 text-label tracking-[0.16em] text-text-subtle">
            <FileCode size={17} />
            PLATFORM / INGRESS CONTROL
          </div>
          <Typography
            as="h1"
            role="page-title"
          >
            Traefik file field
          </Typography>
          <Typography role="supporting">
            Inspect protected routing state, edit dynamic configuration and keep ingress
            changes explicit.
          </Typography>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button
            tone="neutral"
            onClick={() => {
              void files.refetch()
              void status.refetch()
            }}
          >
            <ArrowClockwise size={16} />
            Refresh
          </Button>
          <Button
            disabled={restart.isPending}
            onClick={() => restart.mutate()}
          >
            <Play size={16} />
            {restart.isPending ? 'Restarting…' : 'Restart Traefik'}
          </Button>
        </div>
      </header>
      <div className="flex items-center gap-3 rounded-xl border border-warning/35 bg-warning-soft px-4 py-3 text-supporting text-text-secondary">
        <LockKey
          size={19}
          className="shrink-0 text-warning"
        />
        <span>
          Invalid configuration can interrupt routing. Protected certificate material is
          never displayed or editable.
        </span>
      </div>
      <div className="grid min-h-[620px] gap-4 xl:grid-cols-[minmax(17rem,0.7fr)_minmax(0,1.8fr)]">
        <Card className="grid content-start gap-4 rounded-2xl border-border-subtle bg-surface-1 p-3">
          <div className="flex items-center justify-between gap-3 px-2 pb-2">
            <div>
              <span className="text-label tracking-[0.12em] text-text-subtle">
                FILES
              </span>
              <p className="font-technical text-log text-text-tertiary">
                {entries.length} entries
              </p>
            </div>
            <FolderOpen
              size={20}
              className="text-action"
            />
          </div>
          <div className="grid gap-1">
            {entries.map((entry) => (
              <button
                aria-current={selectedPath === entry.path ? 'page' : undefined}
                className={`flex items-center gap-3 rounded-xl px-3 py-3 text-left transition-colors ${selectedPath === entry.path ? 'border border-action/45 bg-surface-2 text-text-primary' : 'border border-transparent text-text-secondary hover:border-border-default hover:bg-surface-2'} ${entry.type === 'directory' ? 'cursor-default' : 'cursor-pointer'}`}
                key={entry.path}
                onClick={() => entry.type === 'file' && setSelectedPath(entry.path)}
                type="button"
              >
                {entry.protected ? (
                  <LockKey
                    className="shrink-0 text-warning"
                    size={17}
                  />
                ) : (
                  <FileCode
                    className="shrink-0 text-action"
                    size={17}
                  />
                )}
                <span className="min-w-0 flex-1 truncate font-technical text-log">
                  {entry.name}
                </span>
                {entry.type === 'file' ? (
                  <span className="font-technical text-log text-text-tertiary">
                    {formatBytes(entry.size)}
                  </span>
                ) : null}
              </button>
            ))}
          </div>
        </Card>
        <Card className="grid min-h-0 content-start gap-4 rounded-2xl border-border-subtle bg-surface-1 p-5">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="min-w-0">
              <span className="block truncate font-technical text-code text-text-primary">
                {selectedEntry?.name ?? selectedPath}
              </span>
              <span className="text-supporting text-text-tertiary">
                {selectedEntry?.protected
                  ? 'Protected certificate storage'
                  : 'Scoped to the Traefik state directory'}
              </span>
            </div>
            <div className="flex items-center gap-2">
              {file.data?.redacted ? <Badge tone="warning">Redacted</Badge> : null}
              <Badge tone={file.data?.editable ? 'success' : 'neutral'}>
                {file.data?.editable ? 'Editable' : 'Read only'}
              </Badge>
            </div>
          </div>
          {file.isLoading ? (
            <Skeleton className="min-h-[28rem] rounded-xl" />
          ) : file.isError ? (
            <EmptyState
              title="File unavailable"
              description="Select another entry or refresh the protected file system."
              action={
                <Button
                  tone="neutral"
                  onClick={() => file.refetch()}
                >
                  Retry request
                </Button>
              }
            />
          ) : (
            <>
              <CodeEditorLite
                aria-label="Traefik file content"
                className="min-h-[28rem] !bg-code-canvas"
                language="yaml"
                onChange={(event) => setContent(event.target.value)}
                readOnly={!file.data?.editable}
                value={content}
              />
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="flex items-center gap-3 text-supporting text-text-tertiary">
                  <RuntimeStatus
                    status={status.data.state === 'running' ? 'healthy' : 'failed'}
                    label={`Traefik ${status.data.state}`}
                  />
                  {file.data?.redacted ? (
                    <span>Secrets are intentionally omitted.</span>
                  ) : null}
                </div>
                <div className="flex gap-2">
                  {file.data?.editable && selectedPath !== 'traefik.yml' ? (
                    <Button
                      tone="ghost"
                      onClick={() => setDeleteOpen(true)}
                    >
                      <TrashSimple size={16} />
                      Delete
                    </Button>
                  ) : null}
                  <Button
                    disabled={!file.data?.editable || save.isPending}
                    onClick={onSave}
                  >
                    {save.isPending ? 'Saving…' : 'Save changes'}
                  </Button>
                </div>
              </div>
            </>
          )}
        </Card>
      </div>
      <Modal
        trigger={<span className="hidden" />}
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Delete Traefik file"
        description="This may remove routes that depend on the selected dynamic configuration."
        footer={
          <Button
            disabled={remove.isPending}
            onClick={onDelete}
            tone="danger"
          >
            {remove.isPending ? 'Deleting…' : 'Delete file'}
          </Button>
        }
      >
        <p className="text-body text-text-secondary">
          Delete{' '}
          <span className="font-technical text-text-primary">
            {selectedEntry?.name ?? selectedPath}
          </span>
          ?
        </p>
      </Modal>
    </div>
  )
}

function TraefikSkeleton() {
  return (
    <div className="grid gap-4">
      <Skeleton className="h-40 rounded-2xl" />
      <div className="grid gap-4 xl:grid-cols-2">
        <Skeleton className="h-[32rem] rounded-2xl" />
        <Skeleton className="h-[32rem] rounded-2xl" />
      </div>
    </div>
  )
}

function formatBytes(size: number) {
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KiB`
  return `${(size / (1024 * 1024)).toFixed(1)} MiB`
}

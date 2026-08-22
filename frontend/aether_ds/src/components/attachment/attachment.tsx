import { Paperclip, Trash, UploadSimple } from '@phosphor-icons/react'
import { useId, useRef, useState } from 'react'
import { Button } from '../button/button'

export interface AttachmentItem {
  id: string
  name: string
  size?: number
  status?: 'uploading' | 'complete' | 'error'
  progress?: number
  error?: string
}
export interface AttachmentProps {
  label?: string
  description?: string
  accept?: string
  multiple?: boolean
  disabled?: boolean
  items?: AttachmentItem[]
  onFilesChange?: (files: File[]) => void
  onRemove?: (item: AttachmentItem) => void
  onRetry?: (item: AttachmentItem) => void
}

function formatSize(size = 0) {
  if (!size) return ''
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${Math.round(size / 1024)} KB`
  return `${(size / (1024 * 1024)).toFixed(1)} MB`
}

export function Attachment({
  accept,
  description,
  disabled,
  items = [],
  label = 'Attachments',
  multiple = true,
  onFilesChange,
  onRemove,
  onRetry,
}: AttachmentProps) {
  const inputId = useId()
  const inputRef = useRef<HTMLInputElement>(null)
  const [dragging, setDragging] = useState(false)
  const receiveFiles = (files: FileList | null) => {
    if (files?.length) onFilesChange?.(Array.from(files))
  }
  return (
    <div className="space-y-3">
      <div>
        <div className="text-body-sm font-semibold text-foreground">
          {label}
        </div>
        {description ? (
          <div className="mt-1 text-body-sm text-muted-foreground">
            {description}
          </div>
        ) : null}
      </div>
      <button
        type="button"
        disabled={disabled}
        onClick={() => inputRef.current?.click()}
        onDragEnter={(event) => {
          event.preventDefault()
          setDragging(true)
        }}
        onDragOver={(event) => event.preventDefault()}
        onDragLeave={() => setDragging(false)}
        onDrop={(event) => {
          event.preventDefault()
          setDragging(false)
          receiveFiles(event.dataTransfer.files)
        }}
        className={`flex min-h-28 w-full flex-col items-center justify-center gap-2 rounded-lg border border-dashed px-4 py-5 text-center transition-colors ${dragging ? 'border-primary bg-primary/10' : 'border-border bg-surface-card hover:border-primary/60'} disabled:cursor-not-allowed disabled:opacity-50`}
      >
        <UploadSimple size={22} className="text-primary" aria-hidden="true" />
        <span className="text-body-sm font-semibold text-foreground">
          Drop files here or browse
        </span>
        <span className="text-body-sm text-muted-foreground">
          {accept || 'Any supported file type'}
        </span>
      </button>
      <input
        ref={inputRef}
        id={inputId}
        type="file"
        accept={accept}
        multiple={multiple}
        disabled={disabled}
        className="sr-only"
        onChange={(event) => receiveFiles(event.target.files)}
      />
      {items.length ? (
        <ul className="space-y-2" aria-label="Attached files">
          {items.map((item) => (
            <li
              key={item.id}
              className="flex items-center gap-3 rounded-md border border-border bg-surface-card px-3 py-2"
            >
              <Paperclip
                size={17}
                className="shrink-0 text-muted-foreground"
                aria-hidden="true"
              />
              <span className="min-w-0 flex-1">
                <span className="block truncate text-body-sm text-foreground">
                  {item.name}
                </span>
                <span className="block text-body-sm text-muted-foreground">
                  {item.error || formatSize(item.size)}
                </span>
                {item.status === 'uploading' ? (
                  <span className="mt-1 block h-1 overflow-hidden rounded-full bg-surface-container">
                    <span
                      className="block h-full rounded-full bg-primary"
                      style={{ width: `${item.progress ?? 0}%` }}
                    />
                  </span>
                ) : null}
              </span>
              {item.status === 'error' && onRetry ? (
                <Button size="sm" variant="quiet" onClick={() => onRetry(item)}>
                  Retry
                </Button>
              ) : null}
              <button
                type="button"
                aria-label={`Remove ${item.name}`}
                onClick={() => onRemove?.(item)}
                className="text-muted-foreground hover:text-status-danger"
              >
                <Trash size={17} aria-hidden="true" />
              </button>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  )
}

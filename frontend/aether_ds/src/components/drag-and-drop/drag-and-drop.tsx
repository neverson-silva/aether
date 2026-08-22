import { UploadSimple } from '@phosphor-icons/react'
import { type ReactNode, useState } from 'react'
export interface DragAndDropProps {
  children?: ReactNode
  onDrop?: (files: File[]) => void
  onReorder?: (from: number, to: number) => void
  invalid?: boolean
  disabled?: boolean
  label?: string
  items?: string[]
  renderItem?: (item: string, index: number) => ReactNode
  onCancel?: () => void
}
export function DragAndDrop({
  children,
  disabled,
  invalid,
  label = 'Drop files or items here',
  onDrop,
  onReorder,
  items = [],
  renderItem,
  onCancel,
}: DragAndDropProps) {
  const [over, setOver] = useState(false)
  const [dragIndex, setDragIndex] = useState<number | null>(null)
  return (
    <section
      aria-label={label}
      onDragOver={(event) => {
        event.preventDefault()
        if (!disabled) setOver(true)
      }}
      onDragLeave={() => setOver(false)}
      onDrop={(event) => {
        event.preventDefault()
        setOver(false)
        if (disabled) return
        const files = Array.from(event.dataTransfer.files)
        if (files.length) onDrop?.(files)
      }}
      className={`rounded-lg border-2 border-dashed p-6 transition-colors ${over ? 'border-primary bg-primary/10' : invalid ? 'border-status-danger bg-status-danger-container/10' : 'border-border'} ${disabled ? 'pointer-events-none opacity-50' : ''}`}
    >
      <div className="mb-4 flex items-center gap-2 text-body-sm text-muted-foreground">
        <UploadSimple size={18} />
        {label}
      </div>
      {children ?? (
        <div className="text-body-sm text-muted-foreground">
          Choose a file or drag it into this area.
        </div>
      )}
      {onReorder ? (
        <div className="mt-4 space-y-1">
          {items.map((item, index) => (
            <button
              type="button"
              key={item}
              draggable
              onDragStart={() => setDragIndex(index)}
              onDragEnd={() => setDragIndex(null)}
              onDrop={(event) => {
                event.preventDefault()
                if (dragIndex !== null) onReorder(dragIndex, index)
                setDragIndex(null)
              }}
              onKeyDown={(event) => {
                if (event.key === 'ArrowUp' && index > 0)
                  onReorder(index, index - 1)
                if (event.key === 'ArrowDown' && index < items.length - 1)
                  onReorder(index, index + 1)
              }}
              className="rounded-md border border-border bg-surface-card p-3 text-body-sm outline-none hover:bg-surface-container focus-visible:ring-2 focus-visible:ring-primary"
            >
              {renderItem?.(item, index) ?? item}
            </button>
          ))}
        </div>
      ) : null}
      {onCancel ? (
        <button
          type="button"
          onClick={onCancel}
          className="mt-3 text-body-sm text-muted-foreground hover:text-foreground"
        >
          Cancel
        </button>
      ) : null}
    </section>
  )
}

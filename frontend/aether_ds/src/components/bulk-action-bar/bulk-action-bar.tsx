import { X } from '@phosphor-icons/react'
import type { ReactNode } from 'react'
export interface BulkAction {
  id: string
  label: string
  destructive?: boolean
  disabled?: boolean
  onSelect?: () => void
}
export interface BulkActionBarProps {
  selectedCount: number
  actions: BulkAction[]
  onClear?: () => void
  pending?: boolean
  partialFailure?: ReactNode
}
export function BulkActionBar({
  actions,
  onClear,
  partialFailure,
  pending,
  selectedCount,
}: BulkActionBarProps) {
  if (!selectedCount) return null
  return (
    <div
      role="toolbar"
      aria-label="Bulk actions"
      className="sticky bottom-4 z-20 flex flex-wrap items-center gap-3 rounded-lg border border-primary/30 bg-surface-modal p-3 text-foreground shadow-lg"
    >
      <span className="text-body-sm font-semibold">
        {selectedCount} selected
      </span>
      <div className="flex flex-wrap gap-2">
        {actions.map((action) => (
          <button
            type="button"
            key={action.id}
            disabled={pending || action.disabled}
            onClick={action.onSelect}
            className={`rounded-md px-3 py-1.5 text-body-sm ${action.destructive ? 'border border-status-danger/40 text-status-danger' : 'border border-border hover:bg-surface-container'} disabled:opacity-50`}
          >
            {pending ? 'Working...' : action.label}
          </button>
        ))}
      </div>
      {partialFailure ? (
        <span className="text-body-sm text-status-danger">
          {partialFailure}
        </span>
      ) : null}
      <button
        type="button"
        onClick={onClear}
        aria-label="Clear selection"
        className="ml-auto text-muted-foreground hover:text-foreground"
      >
        <X size={18} />
      </button>
    </div>
  )
}

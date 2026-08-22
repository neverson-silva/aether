import { X } from '@phosphor-icons/react'
import type { ReactNode } from 'react'
export interface FilterOption {
  id: string
  label: string
  value: string
  displayValue?: ReactNode
}
export interface FilterBarProps {
  filters: FilterOption[]
  onRemove?: (id: string) => void
  onClear?: () => void
  activeCount?: number
  children?: ReactNode
  loading?: boolean
}
export function FilterBar({
  activeCount = 0,
  children,
  filters,
  loading,
  onClear,
  onRemove,
}: FilterBarProps) {
  return (
    <div className="flex min-w-0 flex-wrap items-center gap-2 rounded-lg border border-border bg-surface-card p-2">
      {loading ? (
        <span className="text-body-sm text-muted-foreground">
          Loading filters...
        </span>
      ) : null}
      {filters.map((filter) => (
        <span
          key={filter.id}
          className="inline-flex max-w-full items-center gap-1 rounded-full border border-primary/30 bg-primary/10 px-2.5 py-1 text-body-sm text-primary"
        >
          <span className="truncate">
            {filter.label}: {filter.displayValue ?? filter.value}
          </span>
          <button
            type="button"
            aria-label={`Remove ${filter.label} filter`}
            onClick={() => onRemove?.(filter.id)}
            className="rounded-full p-0.5 hover:bg-primary/15"
          >
            <X size={14} aria-hidden="true" />
          </button>
        </span>
      ))}
      {children}
      <button
        type="button"
        disabled={!activeCount}
        onClick={onClear}
        className="ml-auto rounded-md px-2 py-1 text-body-sm text-muted-foreground transition-colors hover:bg-surface-container hover:text-foreground disabled:opacity-50"
      >
        Clear all{activeCount ? ` (${activeCount})` : ''}
      </button>
    </div>
  )
}

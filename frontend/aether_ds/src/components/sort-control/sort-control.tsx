import { ArrowDown, ArrowsDownUp, ArrowUp } from '@phosphor-icons/react'
export interface SortControlProps {
  value?: 'none' | 'asc' | 'desc'
  label?: string
  onValueChange?: (value: 'none' | 'asc' | 'desc') => void
}
export function SortControl({
  label = 'Sort',
  onValueChange,
  value = 'none',
}: SortControlProps) {
  const next = value === 'none' ? 'asc' : value === 'asc' ? 'desc' : 'none'
  const Icon =
    value === 'asc' ? ArrowUp : value === 'desc' ? ArrowDown : ArrowsDownUp
  return (
    <button
      type="button"
      aria-label={`${label}: ${value}`}
      onClick={() => onValueChange?.(next)}
      className="inline-flex items-center gap-2 rounded-md border border-border bg-surface-card px-3 py-2 text-body-sm text-foreground transition-colors hover:bg-surface-container focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
    >
      <Icon size={16} aria-hidden="true" />
      {label}
    </button>
  )
}

import type { ReactNode } from 'react'
import { Button } from './button'
import { Select } from './select'

export interface SavedViewOption {
  id: string
  label: ReactNode
}
export interface SavedViewProps {
  views: SavedViewOption[]
  value?: string
  onValueChange?: (value: string) => void
  onSave?: () => void
  className?: string
}
export function SavedView({
  className = '',
  onSave,
  onValueChange,
  value,
  views,
}: SavedViewProps) {
  return (
    <div
      className={`flex flex-wrap items-center gap-2 border-y border-border-subtle bg-surface-1 py-2 ${className}`}
    >
      <span className="text-overline text-text-tertiary">View</span>
      <Select
        aria-label="Saved view"
        className="min-h-9 max-w-56"
        onChange={(event) => onValueChange?.(event.target.value)}
        value={value}
      >
        <option value="">Saved views</option>
        {views.map((view) => (
          <option
            key={view.id}
            value={view.id}
          >
            {view.label}
          </option>
        ))}
      </Select>
      <Button
        disabled={!onSave}
        onClick={onSave}
        size="sm"
        tone="ghost"
      >
        Save current view
      </Button>
    </div>
  )
}

import type { ReactNode } from 'react'
import type { UseFormRegisterReturn } from 'react-hook-form'
import { Input } from './input'

export interface FilterOption {
  id: string
  label: ReactNode
  value?: string
  onRemove?: () => void
}
export interface FilterBarProps {
  filters?: FilterOption[]
  inputProps?: UseFormRegisterReturn
  placeholder?: string
  actions?: ReactNode
  className?: string
}

export function FilterBar({
  actions,
  className = '',
  filters = [],
  inputProps,
  placeholder = 'Filter resources',
}: FilterBarProps) {
  return (
    <div
      className={`flex flex-wrap items-center gap-2 border-y border-border-subtle bg-surface-1 px-3 py-2 transition-[border-color,box-shadow] motion-reduce:transition-none focus-within:border-focus focus-within:ring-2 focus-within:ring-focus/25 ${className}`}
    >
      <Input
        aria-label={placeholder}
        className="min-h-9 max-w-64 border-0 bg-transparent px-0 shadow-none focus-visible:ring-0"
        placeholder={placeholder}
        {...inputProps}
      />
      {filters.map((filter) => (
        <button
          aria-label={`Remove filter ${String(filter.label)}${filter.value ? ` ${filter.value}` : ''}`}
          className="cursor-pointer rounded-md border-l-2 border-action bg-action-soft px-2.5 py-1 text-label text-action-strong transition-[background-color,border-color,transform] duration-[var(--ely-duration-fast)] hover:border-action-strong hover:bg-selection motion-safe:active:scale-[var(--ely-motion-press-scale)] focus-visible:outline-2 focus-visible:outline-focus"
          key={filter.id}
          onClick={filter.onRemove}
          type="button"
        >
          {filter.label}
          {filter.value ? `: ${filter.value}` : ''} <span aria-hidden="true">×</span>
        </button>
      ))}
      {actions ? <div className="ml-auto flex gap-2">{actions}</div> : null}
    </div>
  )
}

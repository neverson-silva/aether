import { Check, MagnifyingGlass } from '@phosphor-icons/react'
import { useMemo, useState, type ReactNode } from 'react'
import { Input } from './input'

export interface ResourceExplorerItem {
  id: string
  label: ReactNode
  group?: ReactNode
  description?: ReactNode
}
export interface MultiSelectResourceExplorerProps {
  items: ResourceExplorerItem[]
  value?: string[]
  defaultValue?: string[]
  onValueChange?: (value: string[]) => void
  className?: string
}

export function MultiSelectResourceExplorer({
  className = '',
  defaultValue = [],
  items,
  onValueChange,
  value: controlledValue,
}: MultiSelectResourceExplorerProps) {
  const [query, setQuery] = useState('')
  const [uncontrolledValue, setUncontrolledValue] = useState(defaultValue)
  const value = controlledValue ?? uncontrolledValue
  const filtered = useMemo(
    () =>
      items.filter((item) =>
        `${item.label} ${item.description ?? ''}`
          .toLowerCase()
          .includes(query.toLowerCase()),
      ),
    [items, query],
  )
  const toggle = (id: string) => {
    const next = value.includes(id)
      ? value.filter((item) => item !== id)
      : [...value, id]
    if (controlledValue === undefined) setUncontrolledValue(next)
    onValueChange?.(next)
  }
  return (
    <div
      className={`grid gap-3 border-y border-border-subtle bg-surface-1 py-3 ${className}`}
    >
      <div className="relative px-3">
        <MagnifyingGlass
          aria-hidden="true"
          className="pointer-events-none absolute left-6 top-3 text-text-tertiary"
          size={16}
        />
        <Input
          aria-label="Search resources"
          className="pl-9"
          onChange={(event) => setQuery(event.target.value)}
          placeholder="Search resources"
          value={query}
        />
      </div>
      <fieldset
        aria-label="Resources"
        className="grid gap-px"
      >
        {filtered.map((item) => {
          const selected = value.includes(item.id)
          return (
            <button
              aria-pressed={selected}
              className={`flex cursor-pointer items-center gap-3 border-l-2 px-3 py-2 text-left transition-[background-color,border-color,transform] duration-[var(--ely-duration-fast)] hover:bg-surface-2 focus-visible:outline-2 focus-visible:outline-focus motion-safe:active:scale-[var(--ely-motion-press-scale)] ${selected ? 'border-action bg-action-soft' : 'border-transparent'}`}
              key={item.id}
              onClick={() => toggle(item.id)}
              type="button"
            >
              <span
                aria-hidden="true"
                className={`grid size-4 place-items-center rounded border transition-[background-color,border-color,transform] duration-[var(--ely-duration-fast)] ${selected ? 'border-action bg-action text-on-action motion-safe:scale-110' : 'border-border-default'}`}
              >
                {selected ? <Check size={12} /> : null}
              </span>
              <span className="min-w-0 flex-1">
                <span className="block truncate text-supporting text-text-primary">
                  {item.label}
                </span>
                {item.description ? (
                  <span className="block truncate text-log text-text-tertiary">
                    {item.description}
                  </span>
                ) : null}
              </span>
            </button>
          )
        })}
        {filtered.length === 0 ? (
          <p className="px-3 py-4 text-center text-supporting text-text-tertiary">
            No resources match this search.
          </p>
        ) : null}
      </fieldset>
    </div>
  )
}

import type { ReactElement } from 'react'
import {
  CommandPalette,
  type CommandPaletteItem,
} from '../command-palette/command-palette'
export interface SpotlightProps {
  trigger: ReactElement
  items: CommandPaletteItem[]
  placeholder?: string
  scopes?: { id: string; label: string }[]
  activeScope?: string
  allowedItemIds?: string[]
  recentItemIds?: string[]
  onScopeChange?: (scope: string) => void
}
export function Spotlight({
  activeScope,
  allowedItemIds,
  items,
  onScopeChange,
  recentItemIds = [],
  scopes = [],
  ...props
}: SpotlightProps) {
  const allowed = allowedItemIds ? new Set(allowedItemIds) : null
  const recent = new Map(recentItemIds.map((id, index) => [id, index]))
  const filtered = items
    .filter((item) => !allowed || allowed.has(item.id))
    .filter((item) => !activeScope || item.group === activeScope)
    .slice()
    .sort(
      (a, b) =>
        (recent.get(a.id) ?? Number.MAX_SAFE_INTEGER) -
        (recent.get(b.id) ?? Number.MAX_SAFE_INTEGER),
    )
  const trigger = scopes.length ? (
    <div className="space-y-2">
      <div className="flex gap-1" role="tablist" aria-label="Search scope">
        {scopes.map((scope) => (
          <button
            key={scope.id}
            type="button"
            role="tab"
            aria-selected={activeScope === scope.id}
            onClick={() => onScopeChange?.(scope.id)}
            className="rounded-md px-2 py-1 text-label-caps text-muted-foreground hover:bg-surface-container aria-selected:bg-primary/15 aria-selected:text-primary"
          >
            {scope.label}
          </button>
        ))}
      </div>
      {props.trigger}
    </div>
  ) : (
    props.trigger
  )
  return <CommandPalette {...props} trigger={trigger} items={filtered} />
}

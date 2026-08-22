import { CheckSquare, Square } from '@phosphor-icons/react'
import type { ReactNode } from 'react'
import { useState } from 'react'
import {
  ResourceTree,
  type ResourceTreeNode,
} from '../resource-tree/resource-tree'
export interface ResourceExplorerItem {
  id: string
  label: string
  status?: string
  type?: string
}
export interface MultiSelectResourceExplorerProps {
  nodes: ResourceTreeNode[]
  items: ResourceExplorerItem[]
  selectedIds?: string[]
  onSelectionChange?: (ids: string[]) => void
  onSearch?: (query: string) => void
  preview?: ReactNode
}
export function MultiSelectResourceExplorer({
  items,
  nodes,
  onSearch,
  onSelectionChange,
  preview,
  selectedIds = [],
}: MultiSelectResourceExplorerProps) {
  const [query, setQuery] = useState('')
  const toggle = (id: string) =>
    onSelectionChange?.(
      selectedIds.includes(id)
        ? selectedIds.filter((item) => item !== id)
        : [...selectedIds, id],
    )
  return (
    <div className="grid min-h-80 gap-4 lg:grid-cols-[14rem_1fr_16rem]">
      <aside className="rounded-lg border border-border bg-surface-card p-3">
        <ResourceTree nodes={nodes} />
      </aside>
      <section className="overflow-hidden rounded-lg border border-border">
        <header className="border-b border-border p-3">
          <input
            value={query}
            onChange={(event) => {
              setQuery(event.target.value)
              onSearch?.(event.target.value)
            }}
            placeholder="Search resources"
            aria-label="Search resources"
            className="h-9 w-full rounded-md border border-border bg-surface-background px-3 text-body-sm outline-none focus:border-primary"
          />
        </header>
        <div className="divide-y divide-border">
          {items
            .filter((item) =>
              item.label.toLowerCase().includes(query.toLowerCase()),
            )
            .map((item) => (
              <button
                type="button"
                key={item.id}
                onClick={() => toggle(item.id)}
                className="flex w-full items-center gap-3 p-3 text-start hover:bg-surface-container"
              >
                <span className="text-primary">
                  {selectedIds.includes(item.id) ? (
                    <CheckSquare size={18} />
                  ) : (
                    <Square size={18} />
                  )}
                </span>
                <span className="min-w-0 flex-1 text-body-sm text-foreground">
                  {item.label}
                </span>
                <span className="text-body-sm text-muted-foreground">
                  {item.status}
                </span>
              </button>
            ))}
        </div>
      </section>
      <aside className="rounded-lg border border-border bg-surface-card p-4">
        {preview ?? (
          <div className="text-body-sm text-muted-foreground">
            Select a resource to preview details.
          </div>
        )}
      </aside>
    </div>
  )
}

import { CaretDown, CaretRight, Folder, Package } from '@phosphor-icons/react'
import { useState, type ReactNode } from 'react'

export interface ResourceTreeItem {
  id: string
  label: ReactNode
  children?: ResourceTreeItem[]
  kind?: 'folder' | 'resource'
}
export interface ResourceTreeProps {
  items: ResourceTreeItem[]
  selectedId?: string
  onSelect?: (id: string) => void
  className?: string
}

function TreeNode({
  item,
  level = 1,
  onSelect,
  selectedId,
}: {
  item: ResourceTreeItem
  level?: number
  onSelect?: (id: string) => void
  selectedId?: string
}) {
  const [open, setOpen] = useState(false)
  const hasChildren = Boolean(item.children?.length)
  return (
    <li
      aria-expanded={hasChildren ? open : undefined}
      aria-level={level}
      aria-selected={selectedId === item.id}
      className="grid gap-1"
      role="treeitem"
      tabIndex={-1}
    >
      <div
        className={`relative flex items-center gap-1 px-2 py-1.5 text-supporting transition-[background-color,color] duration-[var(--ely-duration-fast)] motion-reduce:transition-none ${selectedId === item.id ? 'bg-action-soft text-action-strong before:absolute before:inset-y-1 before:left-0 before:w-0.5 before:rounded-full before:bg-action' : 'text-text-secondary hover:bg-surface-2'}`}
      >
        <button
          aria-expanded={hasChildren ? open : undefined}
          aria-label={open ? `Collapse ${item.label}` : `Expand ${item.label}`}
          className="grid size-6 shrink-0 cursor-pointer place-items-center rounded text-text-tertiary transition-[background-color,color,transform] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:bg-surface-3 disabled:cursor-not-allowed disabled:opacity-45 focus-visible:outline-2 focus-visible:outline-focus motion-safe:active:scale-[var(--ely-motion-press-scale)]"
          disabled={!hasChildren}
          onClick={() => setOpen((current) => !current)}
          type="button"
        >
          {hasChildren ? (
            open ? (
              <CaretDown size={14} />
            ) : (
              <CaretRight size={14} />
            )
          ) : null}
        </button>
        <button
          aria-current={selectedId === item.id ? 'true' : undefined}
          className="flex min-h-8 min-w-0 flex-1 cursor-pointer items-center gap-2 rounded text-left focus-visible:outline-2 focus-visible:outline-focus"
          onClick={() => onSelect?.(item.id)}
          type="button"
        >
          {item.kind === 'folder' || hasChildren ? (
            <Folder
              aria-hidden="true"
              size={16}
            />
          ) : (
            <Package
              aria-hidden="true"
              size={16}
            />
          )}
          <span className="truncate">{item.label}</span>
        </button>
      </div>
      {open && hasChildren ? (
        <ul
          className="ml-4 grid gap-1 border-l border-border-subtle pl-2 transition-[opacity,transform] duration-[var(--ely-duration-standard)] ease-ely-out data-[starting-style]:-translate-y-1 data-[starting-style]:opacity-0 data-[ending-style]:opacity-0 motion-reduce:transition-opacity"
          role="group"
        >
          {item.children?.map((child) => (
            <TreeNode
              item={child}
              key={child.id}
              level={level + 1}
              onSelect={onSelect}
              selectedId={selectedId}
            />
          ))}
        </ul>
      ) : null}
    </li>
  )
}

export function ResourceTree({
  className = '',
  items,
  onSelect,
  selectedId,
}: ResourceTreeProps) {
  return (
    <ul
      aria-label="Resource tree"
      className={`grid gap-1 ${className}`}
      role="tree"
    >
      {items.map((item) => (
        <TreeNode
          item={item}
          key={item.id}
          onSelect={onSelect}
          selectedId={selectedId}
        />
      ))}
    </ul>
  )
}

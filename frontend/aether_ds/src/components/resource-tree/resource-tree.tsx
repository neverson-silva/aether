import { CaretDown, CaretRight, Cube, Folder } from '@phosphor-icons/react'
import { type ReactNode, useState } from 'react'
export interface ResourceTreeNode {
  id: string
  label: string
  type?: 'folder' | 'resource'
  children?: ResourceTreeNode[]
  badge?: ReactNode
  disabled?: boolean
}
export interface ResourceTreeProps {
  nodes: ResourceTreeNode[]
  selectedId?: string
  onSelect?: (node: ResourceTreeNode) => void
  loadingIds?: string[]
  empty?: ReactNode
}
function TreeNode({
  loadingIds,
  node,
  onSelect,
  selectedId,
}: {
  node: ResourceTreeNode
  selectedId?: string
  onSelect?: (node: ResourceTreeNode) => void
  loadingIds: string[]
}) {
  const [expanded, setExpanded] = useState(false)
  const hasChildren = Boolean(node.children?.length)
  return (
    <li>
      <div
        className={`flex items-center gap-1 rounded-md px-2 py-1.5 text-body-sm transition-colors ${selectedId === node.id ? 'bg-primary/10 text-primary' : 'text-foreground hover:bg-surface-container'} ${node.disabled ? 'pointer-events-none opacity-50' : ''}`}
        style={{ paddingLeft: `${8 + (node.type === 'resource' ? 16 : 0)}px` }}
      >
        {hasChildren ? (
          <button
            type="button"
            aria-label={`${expanded ? 'Collapse' : 'Expand'} ${node.label}`}
            onClick={() => setExpanded(!expanded)}
            className="shrink-0 text-muted-foreground"
          >
            {expanded ? <CaretDown size={14} /> : <CaretRight size={14} />}
          </button>
        ) : (
          <span className="w-3.5" />
        )}
        <button
          type="button"
          onClick={() => onSelect?.(node)}
          className="flex min-w-0 flex-1 items-center gap-2 text-start"
        >
          <span className="text-muted-foreground">
            {node.type === 'resource' ? (
              <Cube size={16} />
            ) : (
              <Folder size={16} />
            )}
          </span>
          <span className="min-w-0 flex-1 truncate">{node.label}</span>
          {node.badge}
        </button>
      </div>
      {loadingIds.includes(node.id) ? (
        <div className="ml-8 text-label-caps text-muted-foreground">
          Loading...
        </div>
      ) : null}
      {expanded && node.children ? (
        <ul className="ml-4 border-l border-border pl-1">
          {node.children.map((child) => (
            <TreeNode
              key={child.id}
              node={child}
              selectedId={selectedId}
              onSelect={onSelect}
              loadingIds={loadingIds}
            />
          ))}
        </ul>
      ) : null}
    </li>
  )
}
export function ResourceTree({
  empty = 'No resources found.',
  loadingIds = [],
  nodes,
  onSelect,
  selectedId,
}: ResourceTreeProps) {
  return nodes.length ? (
    <nav aria-label="Resource tree">
      <ul className="space-y-1">
        {nodes.map((node) => (
          <TreeNode
            key={node.id}
            node={node}
            selectedId={selectedId}
            onSelect={onSelect}
            loadingIds={loadingIds}
          />
        ))}
      </ul>
    </nav>
  ) : (
    <div className="rounded-lg border border-dashed border-border p-8 text-center text-body-sm text-muted-foreground">
      {empty}
    </div>
  )
}

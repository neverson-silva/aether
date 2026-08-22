import { Menu } from '@base-ui/react/menu'
import { Check, DotsThree, Star } from '@phosphor-icons/react'
import type { ReactNode } from 'react'
export interface SavedViewItem {
  id: string
  name: string
  owner?: string
  shared?: boolean
  favorite?: boolean
}
export interface SavedViewProps {
  views: SavedViewItem[]
  value?: string
  onValueChange?: (id: string) => void
  onRename?: (id: string) => void
  onDuplicate?: (id: string) => void
  onDelete?: (id: string) => void
  onFavorite?: (id: string) => void
  trigger?: ReactNode
}
export function SavedView({
  onDelete,
  onDuplicate,
  onFavorite,
  onRename,
  onValueChange,
  trigger,
  value,
  views,
}: SavedViewProps) {
  const current = views.find((view) => view.id === value)
  return (
    <div className="flex items-center gap-2">
      <select
        value={value ?? ''}
        onChange={(event) => onValueChange?.(event.target.value)}
        aria-label="Saved view"
        className="h-10 max-w-56 rounded-md border border-border bg-surface-card px-3 text-body-sm text-foreground outline-none focus:border-primary"
      >
        <option value="">{trigger ?? 'Select saved view'}</option>
        {views.map((view) => (
          <option key={view.id} value={view.id}>
            {view.name}
          </option>
        ))}
      </select>
      {current ? (
        <>
          {current.favorite ? (
            <Star size={16} weight="fill" className="text-status-warning" />
          ) : null}
          <Menu.Root>
            <Menu.Trigger className="rounded-md p-2 text-muted-foreground hover:bg-surface-container">
              <DotsThree size={18} />
            </Menu.Trigger>
            <Menu.Portal>
              <Menu.Positioner className="z-50" sideOffset={4}>
                <Menu.Popup className="min-w-40 rounded-lg border border-border bg-surface-popover p-1 shadow-lg">
                  <Menu.Item
                    onClick={() => onRename?.(current.id)}
                    className="cursor-pointer rounded px-3 py-2 text-body-sm data-[highlighted]:bg-surface-container"
                  >
                    Rename
                  </Menu.Item>
                  <Menu.Item
                    onClick={() => onDuplicate?.(current.id)}
                    className="cursor-pointer rounded px-3 py-2 text-body-sm data-[highlighted]:bg-surface-container"
                  >
                    Duplicate
                  </Menu.Item>
                  <Menu.Item
                    onClick={() => onFavorite?.(current.id)}
                    className="cursor-pointer rounded px-3 py-2 text-body-sm data-[highlighted]:bg-surface-container"
                  >
                    {current.favorite ? 'Unfavorite' : 'Favorite'}
                  </Menu.Item>
                  <Menu.Item
                    onClick={() => onDelete?.(current.id)}
                    className="cursor-pointer rounded px-3 py-2 text-body-sm text-status-danger data-[highlighted]:bg-surface-container"
                  >
                    Delete
                  </Menu.Item>
                </Menu.Popup>
              </Menu.Positioner>
            </Menu.Portal>
          </Menu.Root>
          {current.shared ? (
            <span className="text-label-caps text-muted-foreground">
              Shared
            </span>
          ) : null}
          <Check size={14} className="sr-only" />
        </>
      ) : null}
    </div>
  )
}

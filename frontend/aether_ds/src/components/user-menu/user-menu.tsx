import { Menu } from '@base-ui/react/menu'
import { CaretDown, Check, UserCircle } from '@phosphor-icons/react'
import type { ReactNode } from 'react'

export interface WorkspaceOption {
  id: string
  label: string
  description?: string
}
export interface UserMenuProps {
  user: { name: string; email?: string; avatar?: ReactNode }
  workspaces?: WorkspaceOption[]
  currentWorkspace?: string
  onWorkspaceChange?: (id: string) => void
  onProfile?: () => void
  onSignOut?: () => void
}

export function UserMenu({
  currentWorkspace,
  onProfile,
  onSignOut,
  onWorkspaceChange,
  user,
  workspaces = [],
}: UserMenuProps) {
  return (
    <Menu.Root>
      <Menu.Trigger className="flex items-center gap-2 rounded-md px-2 py-1.5 text-start outline-none transition-colors hover:bg-surface-container focus-visible:ring-2 focus-visible:ring-primary">
        <span className="flex size-11 shrink-0 items-center justify-center overflow-hidden rounded-full bg-primary/15 text-primary">
          {user.avatar ?? <UserCircle size={20} aria-hidden="true" />}
        </span>
        <span className="hidden min-w-0 text-body-sm md:block">
          <span className="block truncate font-semibold text-foreground">
            {user.name}
          </span>
          {user.email ? (
            <span className="block truncate text-body-sm text-muted-foreground">
              {user.email}
            </span>
          ) : null}
        </span>
        <CaretDown
          size={14}
          className="text-muted-foreground"
          aria-hidden="true"
        />
      </Menu.Trigger>
      <Menu.Portal>
        <Menu.Positioner className="z-50" sideOffset={6} align="end">
          <Menu.Popup className="min-w-64 rounded-lg border border-border bg-surface-popover p-1 text-foreground shadow-lg outline-none data-[starting-style]:scale-95 data-[starting-style]:opacity-0 data-[ending-style]:scale-95 data-[ending-style]:opacity-0 transition-[transform,opacity] duration-150">
            <div className="border-b border-border px-3 py-2">
              <div className="text-body-sm font-semibold">{user.name}</div>
              {user.email ? (
                <div className="text-body-sm text-muted-foreground">
                  {user.email}
                </div>
              ) : null}
            </div>
            {workspaces.length ? (
              <Menu.Group>
                <Menu.GroupLabel className="px-3 pb-1 pt-3 text-label-caps text-muted-foreground">
                  Workspace
                </Menu.GroupLabel>
                {workspaces.map((workspace) => (
                  <Menu.Item
                    key={workspace.id}
                    onClick={() => onWorkspaceChange?.(workspace.id)}
                    className="flex cursor-pointer items-center gap-2 rounded-md px-3 py-2 text-body-sm outline-none data-[highlighted]:bg-surface-container"
                  >
                    <span className="min-w-0 flex-1">
                      <span className="block truncate">{workspace.label}</span>
                      {workspace.description ? (
                        <span className="block truncate text-body-sm text-muted-foreground">
                          {workspace.description}
                        </span>
                      ) : null}
                    </span>
                    {workspace.id === currentWorkspace ? (
                      <Check
                        size={16}
                        className="text-primary"
                        aria-hidden="true"
                      />
                    ) : null}
                  </Menu.Item>
                ))}
              </Menu.Group>
            ) : null}
            <Menu.Separator className="my-1 h-px bg-border" />
            <Menu.Item
              onClick={onProfile}
              className="cursor-pointer rounded-md px-3 py-2 text-body-sm outline-none data-[highlighted]:bg-surface-container"
            >
              Profile settings
            </Menu.Item>
            <Menu.Item
              onClick={onSignOut}
              className="cursor-pointer rounded-md px-3 py-2 text-body-sm text-destructive outline-none data-[highlighted]:bg-surface-container"
            >
              Sign out
            </Menu.Item>
          </Menu.Popup>
        </Menu.Positioner>
      </Menu.Portal>
    </Menu.Root>
  )
}

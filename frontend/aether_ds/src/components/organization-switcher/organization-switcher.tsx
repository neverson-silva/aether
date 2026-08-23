import { Menu } from '@base-ui/react/menu'
import { CaretDown, Check, Plus } from '@phosphor-icons/react'

export interface OrganizationOption {
  id: string
  label: string
  description?: string
}

export interface OrganizationSwitcherProps {
  options: OrganizationOption[]
  value?: string
  onValueChange?: (id: string) => void
  onCreate?: () => void
}

export function OrganizationSwitcher({
  onCreate,
  onValueChange,
  options,
  value,
}: OrganizationSwitcherProps) {
  const current = options.find((option) => option.id === value) ?? options[0]

  return (
    <Menu.Root>
      <Menu.Trigger
        aria-label="Select organization"
        className="inline-flex min-w-0 max-w-32 items-center gap-1 rounded-md border-0 bg-transparent px-2 py-1.5 text-body-sm font-semibold text-foreground outline-none transition-colors hover:bg-surface-container focus-visible:ring-2 focus-visible:ring-primary md:max-w-56"
      >
        <span className="min-w-0 truncate">{current?.label ?? 'Organization'}</span>
        <CaretDown size={14} aria-hidden="true" />
      </Menu.Trigger>
      <Menu.Portal>
        <Menu.Positioner className="z-50" sideOffset={6}>
          <Menu.Popup
            style={{ backgroundColor: 'var(--semantic-surface-popover)', opacity: 1 }}
            className="min-w-64 max-w-80 rounded-lg border border-border bg-surface-popover p-1 shadow-lg"
          >
            <div className="px-3 py-2 text-label-caps text-muted-foreground">Organizations</div>
            <div className="max-h-80 overflow-y-auto">
              {options.map((option) => (
                <Menu.Item
                  key={option.id}
                  onClick={() => onValueChange?.(option.id)}
                  className="flex min-w-0 cursor-pointer items-center gap-2 rounded-md px-3 py-2 text-body-sm outline-none data-[highlighted]:bg-surface-container"
                >
                  <span className="min-w-0 flex-1">
                    <span className="block truncate">{option.label}</span>
                    {option.description ? (
                      <span className="block truncate text-body-sm text-muted-foreground">{option.description}</span>
                    ) : null}
                  </span>
                  {option.id === value ? <Check size={16} className="shrink-0 text-primary" aria-label="Selected" /> : null}
                </Menu.Item>
              ))}
            </div>
            {onCreate ? (
              <>
                <div className="my-1 border-t border-border" />
                <Menu.Item
                  onClick={onCreate}
                  className="flex cursor-pointer items-center gap-2 rounded-md px-3 py-2 text-body-sm text-foreground outline-none data-[highlighted]:bg-surface-container"
                >
                  <Plus size={16} aria-hidden="true" />
                  Create organization
                </Menu.Item>
              </>
            ) : null}
          </Menu.Popup>
        </Menu.Positioner>
      </Menu.Portal>
    </Menu.Root>
  )
}

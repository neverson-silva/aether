import type { ReactNode } from 'react'
import { Menu as BaseMenu } from '@base-ui/react/menu'

export interface DropdownMenuItem {
  label: ReactNode
  onSelect?: () => void
  disabled?: boolean
  danger?: boolean
}
export interface DropdownMenuProps {
  trigger: ReactNode
  items: DropdownMenuItem[]
  footer?: ReactNode
  onFooterSelect?: () => void
  title?: ReactNode
  className?: string
  triggerClassName?: string
  open?: boolean
  onOpenChange?: (open: boolean) => void
}

export function DropdownMenu({
  className = '',
  footer,
  items,
  onFooterSelect,
  title,
  trigger,
  triggerClassName = 'inline-flex min-h-9 cursor-pointer items-center rounded-lg px-2 text-text-secondary transition-[background-color,color] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:bg-surface-2 hover:text-text-primary focus-visible:outline-2 focus-visible:outline-focus',
  open,
  onOpenChange,
}: DropdownMenuProps) {
  return (
    <BaseMenu.Root onOpenChange={onOpenChange} open={open}>
      <BaseMenu.Trigger className={triggerClassName}>
        {trigger}
      </BaseMenu.Trigger>
      <BaseMenu.Portal>
        <BaseMenu.Positioner className="z-50" sideOffset={6}>
          <BaseMenu.Popup
            className={`z-40 min-w-52 origin-[var(--transform-origin)] rounded-lg border border-border-default bg-overlay p-1 shadow-elevation-2 transition-[opacity,transform] duration-[var(--ely-duration-popover)] ease-ely-out data-[ending-style]:scale-[.98] data-[ending-style]:opacity-0 data-[starting-style]:scale-[.98] data-[starting-style]:opacity-0 motion-reduce:transition-opacity ${className}`}
          >
            {title ? (
              <div className="border-b border-border-subtle px-3 py-2 text-supporting text-text-tertiary">
                {title}
              </div>
            ) : null}
            {items.map((item, index) => (
              <BaseMenu.Item
                className={`flex w-full cursor-pointer rounded-md px-3 py-2 text-left text-supporting outline-none transition-[background-color,color,transform] duration-[var(--ely-duration-fast)] motion-reduce:transition-none data-[highlighted]:bg-surface-2 data-[highlighted]:text-text-primary data-[disabled]:cursor-not-allowed data-[disabled]:opacity-60 ${item.danger ? 'text-danger-strong' : 'text-text-secondary'}`}
                disabled={item.disabled}
                key={index}
                label={typeof item.label === 'string' ? item.label : undefined}
                onClick={item.onSelect}
              >
                {item.label}
              </BaseMenu.Item>
            ))}
            {footer ? (
              <BaseMenu.Item
                className="flex w-full cursor-pointer items-center rounded-md px-3 py-2 text-left text-supporting text-action-strong outline-none transition-[background-color,color,transform] duration-[var(--ely-duration-fast)] motion-reduce:transition-none data-[highlighted]:bg-surface-2 data-[highlighted]:text-action focus-visible:outline-2 focus-visible:outline-focus"
                onClick={onFooterSelect}
              >
                {footer}
              </BaseMenu.Item>
            ) : null}
          </BaseMenu.Popup>
        </BaseMenu.Positioner>
      </BaseMenu.Portal>
    </BaseMenu.Root>
  )
}

import { ContextMenu as BaseContextMenu } from '@base-ui/react/context-menu'
import type { ReactNode } from 'react'
import type { DropdownMenuItem } from './dropdown-menu'

export interface ContextMenuProps {
  children: ReactNode
  items: DropdownMenuItem[]
  className?: string
}

export function ContextMenu({ children, className = '', items }: ContextMenuProps) {
  return (
    <BaseContextMenu.Root>
      <BaseContextMenu.Trigger className={`block ${className}`}>
        {children}
      </BaseContextMenu.Trigger>
      <BaseContextMenu.Portal>
        <BaseContextMenu.Positioner>
          <BaseContextMenu.Popup className="z-40 min-w-52 origin-[var(--transform-origin)] rounded-lg border border-border-default bg-overlay p-1 shadow-elevation-2 transition-[opacity,transform] duration-[var(--ely-duration-popover)] ease-ely-out data-[ending-style]:scale-[.98] data-[ending-style]:opacity-0 data-[starting-style]:scale-[.98] data-[starting-style]:opacity-0 motion-reduce:transition-opacity">
            {items.map((item, index) => (
              <BaseContextMenu.Item
                className={`flex w-full cursor-pointer rounded-md px-3 py-2 text-left text-supporting outline-none transition-colors motion-reduce:transition-none data-[highlighted]:bg-surface-2 data-[disabled]:cursor-not-allowed data-[disabled]:opacity-45 ${item.danger ? 'text-danger' : 'text-text-secondary'}`}
                disabled={item.disabled}
                key={index}
                onClick={item.onSelect}
              >
                {item.label}
              </BaseContextMenu.Item>
            ))}
          </BaseContextMenu.Popup>
        </BaseContextMenu.Positioner>
      </BaseContextMenu.Portal>
    </BaseContextMenu.Root>
  )
}

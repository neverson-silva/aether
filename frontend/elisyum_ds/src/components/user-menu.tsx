import type { ReactNode } from 'react'
import { CaretDown, CaretUp } from '@phosphor-icons/react'
import { useState } from 'react'
import { Avatar } from './avatar'
import { DropdownMenu } from './dropdown-menu'

export interface UserMenuProps {
  name: string
  email?: string
  items?: { label: ReactNode; onSelect?: () => void; danger?: boolean }[]
  compact?: boolean
}

export function UserMenu({ compact = false, email, items = [], name }: UserMenuProps) {
  const [open, setOpen] = useState(false)
  const displayName = compact ? name.split(/\s+/)[0] || name : name
  return (
    <DropdownMenu
      items={items}
      onOpenChange={setOpen}
      open={open}
      triggerClassName="inline-flex min-h-10 cursor-pointer items-center rounded-xl px-2 transition-[background-color,color,transform] duration-[var(--ely-duration-fast)] ease-ely-out hover:bg-surface-2 active:bg-surface-3 motion-safe:active:scale-[var(--ely-motion-press-scale)] focus-visible:outline-2 focus-visible:outline-focus"
      trigger={
        <span className="flex items-center gap-2">
          <Avatar
            name={name}
            size={compact ? 'md' : 'sm'}
          />
          <span className="hidden text-supporting font-medium text-text-primary sm:inline">{displayName}</span>
          {compact ? (open ? <CaretUp aria-hidden="true" className="hidden text-text-tertiary sm:block" size={15} weight="bold" /> : <CaretDown aria-hidden="true" className="hidden text-text-tertiary sm:block" size={15} weight="bold" />) : null}
        </span>
      }
      title={email ?? name}
    />
  )
}

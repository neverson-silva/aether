import type { ReactNode } from 'react'
import { Collapsible } from '@base-ui/react/collapsible'

export interface CollapsibleProps {
  title: ReactNode
  children: ReactNode
  defaultOpen?: boolean
  open?: boolean
  onOpenChange?: (open: boolean) => void
  className?: string
}

export function ElisyumCollapsible({
  children,
  className = '',
  defaultOpen,
  onOpenChange,
  open,
  title,
}: CollapsibleProps) {
  return (
    <Collapsible.Root
      className={`overflow-hidden rounded-lg border border-border-subtle bg-surface-1 ${className}`}
      defaultOpen={defaultOpen}
      onOpenChange={onOpenChange}
      open={open}
    >
      <Collapsible.Trigger className="flex w-full cursor-pointer items-center justify-between px-4 py-3 text-left text-body text-text-primary transition-[background-color,transform] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:bg-surface-2 focus-visible:outline-2 focus-visible:outline-focus motion-safe:active:scale-[var(--ely-motion-press-scale)] motion-reduce:active:scale-100 disabled:cursor-not-allowed">
        {title}
        <span
          aria-hidden="true"
          className="text-action-strong transition-transform duration-[var(--ely-duration-fast)] ease-ely-out motion-reduce:transition-none data-[state=open]:rotate-45"
        >
          +
        </span>
      </Collapsible.Trigger>
      <Collapsible.Panel className="h-[var(--collapsible-panel-height)] overflow-hidden border-t border-border-subtle px-4 py-3 text-supporting text-text-secondary transition-[height,opacity] duration-[var(--ely-duration-standard)] ease-ely-out data-[ending-style]:h-0 data-[ending-style]:opacity-0 data-[starting-style]:h-0 data-[starting-style]:opacity-0 motion-reduce:transition-opacity">
        {children}
      </Collapsible.Panel>
    </Collapsible.Root>
  )
}

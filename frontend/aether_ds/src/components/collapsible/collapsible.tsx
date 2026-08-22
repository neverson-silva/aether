import { Collapsible as BaseCollapsible } from '@base-ui/react/collapsible'
import type { ReactNode } from 'react'
export interface CollapsibleProps {
  title: ReactNode
  children: ReactNode
  open?: boolean
  defaultOpen?: boolean
  onOpenChange?: (open: boolean) => void
  disabled?: boolean
}
export function Collapsible({
  children,
  defaultOpen,
  disabled,
  onOpenChange,
  open,
  title,
}: CollapsibleProps) {
  return (
    <BaseCollapsible.Root
      open={open}
      defaultOpen={defaultOpen}
      onOpenChange={onOpenChange}
      disabled={disabled}
    >
      <BaseCollapsible.Trigger className="flex w-full items-center justify-between rounded-md border border-border bg-surface-card px-4 py-3 text-start font-semibold hover:bg-surface-container disabled:opacity-50">
        {title}
        <span aria-hidden="true">⌄</span>
      </BaseCollapsible.Trigger>
      <BaseCollapsible.Panel className="overflow-hidden border-x border-b border-border p-4 text-body-sm text-muted-foreground">
        {children}
      </BaseCollapsible.Panel>
    </BaseCollapsible.Root>
  )
}

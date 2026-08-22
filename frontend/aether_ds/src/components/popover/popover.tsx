import { Popover as BasePopover } from '@base-ui/react/popover'
import type { ReactElement, ReactNode } from 'react'

export interface PopoverProps {
  trigger: ReactElement
  children: ReactNode
  title?: string
  description?: string
  side?: 'top' | 'right' | 'bottom' | 'left'
}

export function Popover({
  children,
  description,
  side = 'bottom',
  title,
  trigger,
}: PopoverProps) {
  return (
    <BasePopover.Root>
      <BasePopover.Trigger render={trigger} />
      <BasePopover.Portal>
        <BasePopover.Positioner
          side={side}
          align="start"
          sideOffset={8}
          className="z-50"
        >
          <BasePopover.Popup className="min-w-64 rounded-lg border border-border bg-surface-popover p-4 text-foreground shadow-lg outline-none data-[starting-style]:translate-y-1 data-[starting-style]:opacity-0 data-[ending-style]:translate-y-1 data-[ending-style]:opacity-0 transition-[transform,opacity] duration-200">
            {title ? (
              <BasePopover.Title className="font-semibold">
                {title}
              </BasePopover.Title>
            ) : null}
            {description ? (
              <BasePopover.Description className="mt-1 text-body-sm text-muted-foreground">
                {description}
              </BasePopover.Description>
            ) : null}
            <div className={title || description ? 'mt-4' : ''}>{children}</div>
          </BasePopover.Popup>
        </BasePopover.Positioner>
      </BasePopover.Portal>
    </BasePopover.Root>
  )
}

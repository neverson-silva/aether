import { Popover as BasePopover } from '@base-ui/react/popover'
import { useRef, useState, type ReactNode } from 'react'

export interface HoverCardProps {
  trigger: ReactNode
  children: ReactNode
  title?: ReactNode
}
export function HoverCard({ children, title, trigger }: HoverCardProps) {
  const [open, setOpen] = useState(false)
  const closeTimer = useRef<number | undefined>(undefined)
  const reveal = () => {
    if (closeTimer.current) window.clearTimeout(closeTimer.current)
    setOpen(true)
  }
  const dismiss = () => {
    closeTimer.current = window.setTimeout(() => setOpen(false), 120)
  }
  return (
    <BasePopover.Root
      open={open}
      onOpenChange={setOpen}
    >
      <BasePopover.Trigger
        className="inline-flex cursor-pointer rounded-md transition-[background-color,color,transform] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:bg-action-soft focus-visible:outline-2 focus-visible:outline-focus motion-safe:active:scale-[var(--ely-motion-press-scale)] motion-reduce:active:scale-100"
        onBlur={dismiss}
        onFocus={reveal}
        onPointerEnter={reveal}
        onPointerLeave={dismiss}
      >
        {trigger}
      </BasePopover.Trigger>
      <BasePopover.Portal>
        <BasePopover.Positioner
          sideOffset={8}
          onPointerEnter={reveal}
          onPointerLeave={dismiss}
        >
          <BasePopover.Popup className="z-30 min-w-56 origin-[var(--transform-origin)] rounded-lg border border-border-default bg-overlay p-4 text-text-primary shadow-elevation-2 transition-[opacity,transform] duration-[var(--ely-duration-popover)] ease-ely-out motion-reduce:transition-none data-[ending-style]:scale-[.98] data-[ending-style]:opacity-0 data-[starting-style]:scale-[.98] data-[starting-style]:opacity-0">
            <BasePopover.Title className="mb-2 text-label">{title}</BasePopover.Title>
            <div className="text-supporting text-text-secondary">{children}</div>
          </BasePopover.Popup>
        </BasePopover.Positioner>
      </BasePopover.Portal>
    </BasePopover.Root>
  )
}

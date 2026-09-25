import { useMemo, useRef, type ReactNode } from 'react'
import { Popover as BasePopover } from '@base-ui/react/popover'

export interface PopoverProps {
  trigger: ReactNode
  children: ReactNode
  title?: ReactNode
  className?: string
  open?: boolean
  onOpenChange?: (open: boolean) => void
  triggerClassName?: string
  side?: 'top' | 'right' | 'bottom' | 'left'
  align?: 'start' | 'center' | 'end'
  sideOffset?: number
  collisionPadding?: number
  placement?: 'standard' | 'anchor-overlap-end' | 'anchor-overlap-center'
}

export function Popover({ align = 'center', children, className = '', collisionPadding = 12, onOpenChange, open, placement = 'standard', side = 'bottom', sideOffset = 8, title, trigger, triggerClassName }: PopoverProps) {
  const triggerRef = useRef<HTMLButtonElement>(null)
  const portalContainer = typeof document === 'undefined' ? null : document.getElementById('overlay-root') ?? document.body
  const overlapAnchor = useMemo(() => ({
    getBoundingClientRect: () => {
      const rect = triggerRef.current?.getBoundingClientRect()
      if (!rect) return new DOMRect()
      const top = rect.top + rect.height * 0.7
      return new DOMRect(rect.left, top, rect.width, 0)
    },
  }), [])
  const isOverlapPlacement = placement !== 'standard'
  const overlapAlign = placement === 'anchor-overlap-center' ? 'center' : 'end'
  return (
    <BasePopover.Root onOpenChange={onOpenChange} open={open}>
      <BasePopover.Trigger ref={triggerRef} className={triggerClassName ?? 'inline-flex min-h-9 cursor-pointer items-center rounded-md px-3 text-label text-action-strong transition-[background-color,color,transform] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:bg-action-soft focus-visible:outline-2 focus-visible:outline-focus motion-safe:active:scale-[var(--ely-motion-press-scale)] motion-reduce:active:scale-100'}>
        {trigger}
      </BasePopover.Trigger>
      <BasePopover.Portal container={portalContainer}>
        <BasePopover.Positioner align={isOverlapPlacement ? overlapAlign : align} anchor={isOverlapPlacement ? overlapAnchor : undefined} collisionAvoidance={isOverlapPlacement ? { align: 'shift', fallbackAxisSide: 'none', side: 'shift' } : undefined} collisionPadding={collisionPadding} positionMethod={isOverlapPlacement ? 'fixed' : undefined} side={isOverlapPlacement ? 'bottom' : side} sideOffset={isOverlapPlacement ? 0 : sideOffset}>
          <BasePopover.Popup
            className={`z-[calc(var(--ely-z-popover)+1)] min-w-56 origin-[var(--transform-origin)] rounded-lg border border-border-default bg-overlay p-4 text-text-primary shadow-elevation-2 transition-[opacity,transform] duration-[var(--ely-duration-popover)] ease-ely-out data-[ending-style]:scale-[.98] data-[ending-style]:opacity-0 data-[starting-style]:scale-[.98] data-[starting-style]:opacity-0 motion-reduce:transition-opacity ${className}`}
          >
            {title ? (
              <BasePopover.Title className="mb-2 text-label">{title}</BasePopover.Title>
            ) : null}
            <div className="text-supporting text-text-secondary">{children}</div>
          </BasePopover.Popup>
        </BasePopover.Positioner>
      </BasePopover.Portal>
    </BasePopover.Root>
  )
}

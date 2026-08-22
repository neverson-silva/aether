import { PreviewCard } from '@base-ui/react/preview-card'
import type { ReactElement, ReactNode } from 'react'

export interface HoverCardProps {
  trigger: ReactElement
  children: ReactNode
  openDelay?: number
  title?: ReactNode
  description?: ReactNode
  footer?: ReactNode
}

export function HoverCard({
  children,
  openDelay = 300,
  trigger,
  title,
  description,
  footer,
}: HoverCardProps) {
  return (
    <PreviewCard.Root>
      <PreviewCard.Trigger render={trigger} delay={openDelay} />
      <PreviewCard.Portal>
        <PreviewCard.Positioner side="top" sideOffset={8} className="z-50">
          <PreviewCard.Popup className="w-72 rounded-lg border border-border bg-surface-popover p-4 text-foreground shadow-lg outline-none data-[starting-style]:translate-y-1 data-[starting-style]:opacity-0 data-[ending-style]:translate-y-1 data-[ending-style]:opacity-0 transition-[transform,opacity] duration-200">
            {title ? <div className="font-semibold">{title}</div> : null}
            {description ? (
              <div className="mt-1 text-body-sm text-muted-foreground">
                {description}
              </div>
            ) : null}
            {children}
            {footer ? (
              <div className="mt-3 border-t border-border pt-3">{footer}</div>
            ) : null}
          </PreviewCard.Popup>
        </PreviewCard.Positioner>
      </PreviewCard.Portal>
    </PreviewCard.Root>
  )
}

import { ScrollArea as BaseScrollArea } from '@base-ui/react/scroll-area'
import type { ReactNode } from 'react'
export interface ScrollAreaProps {
  children: ReactNode
  className?: string
  orientation?: 'vertical' | 'horizontal' | 'both'
  height?: string
}
export function ScrollArea({
  children,
  className = '',
  height = 'h-64',
  orientation = 'vertical',
}: ScrollAreaProps) {
  return (
    <BaseScrollArea.Root className={`${height} overflow-hidden ${className}`}>
      <BaseScrollArea.Viewport
        className={`${orientation === 'horizontal' ? 'overflow-x-auto overflow-y-hidden' : orientation === 'both' ? 'overflow-auto' : 'overflow-y-auto'} h-full w-full`}
      >
        <BaseScrollArea.Content>{children}</BaseScrollArea.Content>
      </BaseScrollArea.Viewport>
      <BaseScrollArea.Scrollbar
        orientation="vertical"
        className="flex w-2 bg-surface-container p-0.5"
      >
        <BaseScrollArea.Thumb className="flex-1 rounded-full bg-muted-foreground/50" />
      </BaseScrollArea.Scrollbar>
    </BaseScrollArea.Root>
  )
}

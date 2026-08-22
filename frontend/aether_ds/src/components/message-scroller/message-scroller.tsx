import type { ReactNode } from 'react'
import { Message, type MessageProps } from '../message/message'
export interface MessageScrollerItem extends Omit<MessageProps, 'className'> {
  id: string
}
export interface MessageScrollerProps {
  items: MessageScrollerItem[]
  maxHeight?: number
  empty?: ReactNode
  className?: string
}
export function MessageScroller({
  className = '',
  empty = 'No messages.',
  items,
  maxHeight = 360,
}: MessageScrollerProps) {
  return (
    <div
      className={`space-y-2 overflow-y-auto pr-1 ${className}`}
      style={{ maxHeight }}
      aria-live="polite"
    >
      {items.length ? (
        items.map(({ id, ...item }) => <Message key={id} {...item} />)
      ) : (
        <div className="rounded-lg border border-dashed border-border p-8 text-center text-body-sm text-muted-foreground">
          {empty}
        </div>
      )}
    </div>
  )
}

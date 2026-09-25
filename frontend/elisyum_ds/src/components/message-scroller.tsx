import type { ReactNode } from 'react'
import { Message, type MessageProps } from './message'
import { ScrollArea } from './scroll-area'

export interface MessageScrollerItem extends MessageProps {
  id: string
}
export interface MessageScrollerProps {
  messages: MessageScrollerItem[]
  className?: string
  children?: ReactNode
}
export function MessageScroller({
  children,
  className = '',
  messages,
}: MessageScrollerProps) {
  return (
    <ScrollArea
      aria-label="Messages"
      className={`grid gap-4 border-y border-border-subtle bg-surface-1 p-4 ${className}`}
      role="log"
    >
      {messages.map((message) => (
        <Message
          {...message}
          key={message.id}
        />
      ))}
      {children}
    </ScrollArea>
  )
}

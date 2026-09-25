import type { ReactNode } from 'react'
import { Avatar } from './avatar'
import { Bubble } from './bubble'

export interface MessageProps {
  author: string
  children: ReactNode
  timestamp?: ReactNode
  align?: 'start' | 'end'
  className?: string
}
export function Message({
  align = 'start',
  author,
  children,
  className = '',
  timestamp,
}: MessageProps) {
  return (
    <article
      aria-label={`Message from ${author}`}
      className={`flex gap-3 ${align === 'end' ? 'flex-row-reverse' : ''} ${className}`}
    >
      <Avatar
        name={author}
        size="sm"
      />
      <div className={`grid gap-1 ${align === 'end' ? 'justify-items-end' : ''}`}>
        <div className="flex items-center gap-2">
          <strong className="text-label text-text-primary">{author}</strong>
          {timestamp ? (
            <time className="text-log text-text-tertiary">{timestamp}</time>
          ) : null}
        </div>
        <Bubble align={align}>{children}</Bubble>
      </div>
    </article>
  )
}

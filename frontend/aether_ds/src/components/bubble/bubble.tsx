import type { ReactNode } from 'react'
import { tv } from 'tailwind-variants'
export interface BubbleProps {
  children: ReactNode
  author?: ReactNode
  timestamp?: ReactNode
  side?: 'start' | 'end'
  tone?: 'neutral' | 'primary' | 'success' | 'danger'
  avatar?: ReactNode
  className?: string
}
const bubble = tv({
  base: 'flex max-w-[min(36rem,90%)] gap-2',
  variants: {
    side: { start: 'self-start', end: 'self-end flex-row-reverse' },
    tone: {
      neutral: 'bg-surface-container',
      primary: 'bg-primary text-primary-foreground',
      success: 'bg-status-success text-status-success-foreground',
      danger: 'bg-status-danger text-status-danger-foreground',
    },
  },
  defaultVariants: { side: 'start', tone: 'neutral' },
})
export function Bubble({
  author,
  avatar,
  children,
  className,
  side = 'start',
  timestamp,
  tone = 'neutral',
}: BubbleProps) {
  return (
    <div className={bubble({ side, className })}>
      <div className="shrink-0">{avatar}</div>
      <div className={`min-w-0 rounded-xl px-4 py-3 ${bubble({ tone })}`}>
        <div className="text-body-md">{children}</div>
        {author || timestamp ? (
          <div className="mt-2 flex gap-2 text-label-caps opacity-70">
            {author ? <span>{author}</span> : null}
            {timestamp ? <span>{timestamp}</span> : null}
          </div>
        ) : null}
      </div>
    </div>
  )
}

import type { ReactNode } from 'react'
export interface BubbleProps {
  children: ReactNode
  tone?: 'neutral' | 'accent'
  align?: 'start' | 'end'
  className?: string
}
export function Bubble({
  align = 'start',
  children,
  className = '',
  tone = 'neutral',
}: BubbleProps) {
  return (
    <div
      className={`max-w-[75%] rounded-xl px-4 py-3 text-body transition-[background-color,transform] duration-[var(--ely-duration-fast)] motion-reduce:transition-none ${align === 'end' ? 'ml-auto' : ''} ${tone === 'accent' ? 'bg-action-soft text-action-strong' : 'bg-surface-2 text-text-secondary'} ${className}`}
    >
      {children}
    </div>
  )
}

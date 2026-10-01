import { useEffect, useState } from 'react'
import type { ReactNode } from 'react'
import { Button } from './button'

export interface BulkAction {
  id: string
  label: ReactNode
  onSelect?: () => void
  tone?: 'accent' | 'danger' | 'neutral'
}
export interface BulkActionBarProps {
  count: number
  actions: BulkAction[]
  className?: string
  layout?: 'floating' | 'inline'
  onClear?: () => void
}
export function BulkActionBar({
  actions,
  className = '',
  count,
  layout = 'floating',
  onClear,
}: BulkActionBarProps) {
  const [rendered, setRendered] = useState(count > 0)
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    let frame: number | undefined
    let timeout: number | undefined
    if (count > 0) {
      setRendered(true)
      frame = window.requestAnimationFrame(() => setVisible(true))
    } else {
      setVisible(false)
      timeout = window.setTimeout(() => setRendered(false), 220)
    }
    return () => {
      if (frame !== undefined) window.cancelAnimationFrame(frame)
      if (timeout !== undefined) window.clearTimeout(timeout)
    }
  }, [count])

  if (!rendered) return null
  const layoutClassName =
    layout === 'inline'
      ? 'flex w-full min-w-0 max-w-full flex-wrap items-center justify-between gap-3 rounded-2xl border border-border-subtle/70 bg-surface-1/90 p-3 shadow-[0_18px_50px_rgb(0_0_0_/_0.32),0_4px_14px_rgb(0_0_0_/_0.18)] backdrop-blur-xl transition-[opacity,transform] duration-[var(--ely-duration-overlay)] ease-ely-out motion-reduce:transition-opacity'
      : 'fixed inset-x-3 bottom-20 z-50 flex w-auto min-w-0 max-w-full flex-wrap items-center gap-3 rounded-2xl border border-border-subtle/70 bg-surface-1/90 p-3 shadow-[0_18px_50px_rgb(0_0_0_/_0.32),0_4px_14px_rgb(0_0_0_/_0.18)] backdrop-blur-xl transform-gpu transition-[opacity,transform] duration-[var(--ely-duration-overlay)] ease-ely-out motion-reduce:transform-none motion-reduce:transition-opacity md:static md:inset-auto md:bottom-auto md:mb-4 md:w-full md:max-w-full md:transform-none md:transition-none'
  const visibilityClassName =
    layout === 'inline'
      ? visible
        ? 'opacity-100'
        : 'opacity-0'
      : visible
        ? 'translate-y-0 scale-100 opacity-100'
        : 'translate-y-3 scale-[0.97] opacity-0'
  return (
    <section
      aria-live="polite"
      className={`${layoutClassName} ${visibilityClassName} ${className}`}
      aria-label="Bulk actions"
    >
      <span className="shrink-0 border-l-2 border-action pl-2 text-supporting text-text-secondary">
        {count} selected
      </span>
      <div className="ml-auto flex min-w-0 max-w-full flex-wrap items-center justify-end gap-2">
        {onClear ? (
          <Button onClick={onClear} size="sm" tone="ghost">
            Clear selection
          </Button>
        ) : null}
        {actions.map((action) => (
          <Button
            key={action.id}
            onClick={action.onSelect}
            size="sm"
            tone={action.tone}
          >
            {action.label}
          </Button>
        ))}
      </div>
    </section>
  )
}

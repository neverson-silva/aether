import type { CSSProperties, ReactNode } from 'react'
export interface ResizableProps {
  first: ReactNode
  second: ReactNode
  value?: number
  onValueChange?: (value: number) => void
  className?: string
}
export function Resizable({
  className = '',
  first,
  onValueChange,
  second,
  value = 50,
}: ResizableProps) {
  return (
    <div className={`grid gap-3 ${className}`}>
      <div
        className="grid min-h-24 grid-cols-[var(--ely-split)_1fr] border-y border-border-subtle bg-surface-1"
        style={{ '--ely-split': `${value}%` } as CSSProperties}
      >
        <div className="min-w-0 overflow-auto border-r border-border-subtle p-4">
          {first}
        </div>
        <div className="min-w-0 overflow-auto p-4">{second}</div>
      </div>
      <label className="grid gap-1 text-supporting text-text-tertiary">
        Panel split
        <span className="flex items-center gap-3">
          <input
            aria-label="Panel split"
            className="h-2 flex-1 cursor-pointer accent-action transition-[filter,opacity] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-45"
            max={80}
            min={20}
            onChange={(event) => onValueChange?.(Number(event.target.value))}
            type="range"
            value={value}
          />
          <output className="min-w-10 text-right font-technical text-log text-text-secondary">
            {value}%
          </output>
        </span>
      </label>
    </div>
  )
}

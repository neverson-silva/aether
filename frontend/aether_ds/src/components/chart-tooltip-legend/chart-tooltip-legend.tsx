import type { ReactNode } from 'react'
export interface ChartTooltipItem {
  label: string
  value: ReactNode
  color?: string
  unit?: string
}
export interface ChartTooltipLegendProps {
  title?: string
  items: ChartTooltipItem[]
  activeId?: string
  onItemChange?: (label: string) => void
}
export function ChartTooltipLegend({
  activeId,
  items,
  onItemChange,
  title,
}: ChartTooltipLegendProps) {
  return (
    <div className="rounded-lg border border-border bg-surface-popover p-3 shadow-md">
      {title ? (
        <div className="mb-2 text-body-sm font-semibold text-foreground">
          {title}
        </div>
      ) : null}
      <div className="space-y-2">
        {items.map((item) => (
          <button
            type="button"
            key={item.label}
            onClick={() => onItemChange?.(item.label)}
            className={`flex w-full items-center justify-between gap-6 rounded-md px-2 py-1 text-start text-body-sm transition-colors hover:bg-surface-container ${activeId && activeId !== item.label ? 'opacity-50' : ''}`}
          >
            <span className="inline-flex items-center gap-2 text-muted-foreground">
              <span
                className="size-2 rounded-full"
                style={{
                  backgroundColor:
                    item.color ?? 'var(--semantic-action-primary)',
                }}
              />
              {item.label}
            </span>
            <span className="font-mono text-foreground">
              {item.value}
              {item.unit ? ` ${item.unit}` : ''}
            </span>
          </button>
        ))}
      </div>
    </div>
  )
}

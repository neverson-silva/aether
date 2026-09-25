import type { ReactNode } from 'react'

export interface ChartTooltipItem {
  label: ReactNode
  value: ReactNode
  color?: string
}
export interface ChartTooltipLegendProps {
  items: ChartTooltipItem[]
  className?: string
}

export function ChartTooltipLegend({ className = '', items }: ChartTooltipLegendProps) {
  return (
    <ul
      aria-label="Chart legend"
      className={`m-0 flex list-none flex-wrap gap-x-4 gap-y-2 p-0 ${className}`}
    >
      {items.map((item) => (
        <li
          className="flex items-center gap-2 text-supporting text-text-secondary"
          key={String(item.label)}
        >
          <span
            aria-hidden="true"
            className="size-2 rounded-full"
            style={{ backgroundColor: item.color ?? 'var(--ely-color-action)' }}
          />
          {item.label}
          <strong className="font-medium text-text-primary">{item.value}</strong>
        </li>
      ))}
    </ul>
  )
}

import type { ReactNode } from 'react'

export interface ChartSeries {
  label: string
  values: number[]
  color?: string
}
export interface ChartProps {
  series: ChartSeries[]
  labels?: string[]
  height?: number
  title?: ReactNode
  className?: string
}

export function Chart({
  className = '',
  height = 180,
  labels = [],
  series,
  title,
}: ChartProps) {
  if (!series.length || series.every((item) => item.values.length === 0))
    return (
      <div
        aria-label={typeof title === 'string' ? title : 'Chart'}
        className={`grid min-h-32 place-items-center rounded-lg border border-dashed border-border-default bg-surface-1 px-4 text-center text-supporting text-text-tertiary ${className}`}
        role="img"
      >
        No data available.
      </div>
    )
  const max = Math.max(1, ...series.flatMap((item) => item.values))
  const width = 520
  const padding = 20
  const chartWidth = width - padding * 2
  const chartHeight = height - padding * 2
  return (
    <div
      className={`grid gap-3 border-y border-border-subtle bg-surface-1 py-4 ${className}`}
    >
      {title ? (
        <div className="flex flex-wrap items-center justify-between gap-3 px-4">
          <h3 className="text-label text-text-secondary">{title}</h3>
          <div className="flex flex-wrap gap-3">
            {series.map((item, seriesIndex) => (
              <span
                className="inline-flex items-center gap-2 text-log text-text-tertiary"
                key={item.label}
              >
                <span
                  aria-hidden="true"
                  className={`size-1.5 rounded-full ${seriesIndex ? 'bg-info' : 'bg-action'}`}
                />
                {item.label}
              </span>
            ))}
          </div>
        </div>
      ) : null}
      <svg
        aria-label={typeof title === 'string' ? title : 'Chart'}
        className="w-full"
        role="img"
        viewBox={`0 0 ${width} ${height}`}
        preserveAspectRatio="none"
      >
        <line
          aria-hidden="true"
          stroke="var(--ely-color-outline-subtle)"
          x1={padding}
          x2={width - padding}
          y1={padding + chartHeight * 0.25}
          y2={padding + chartHeight * 0.25}
        />
        <line
          aria-hidden="true"
          stroke="var(--ely-color-outline-subtle)"
          x1={padding}
          x2={width - padding}
          y1={padding + chartHeight * 0.5}
          y2={padding + chartHeight * 0.5}
        />
        <line
          aria-hidden="true"
          stroke="var(--ely-color-outline-subtle)"
          x1={padding}
          x2={width - padding}
          y1={padding + chartHeight * 0.75}
          y2={padding + chartHeight * 0.75}
        />
        <line
          aria-hidden="true"
          stroke="var(--ely-color-outline-subtle)"
          x1={padding}
          x2={width - padding}
          y1={height - padding}
          y2={height - padding}
        />
        {series.map((item, seriesIndex) => {
          const step =
            item.values.length > 1 ? chartWidth / (item.values.length - 1) : chartWidth
          const points = item.values
            .map(
              (itemValue, index) =>
                `${padding + step * index},${padding + chartHeight - (itemValue / max) * chartHeight}`,
            )
            .join(' ')
          return (
            <polyline
              aria-label={item.label}
              fill="none"
              key={item.label}
              points={points}
              stroke={
                item.color ??
                (seriesIndex ? 'var(--ely-color-info)' : 'var(--ely-color-action)')
              }
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth="3"
            />
          )
        })}
      </svg>
      {labels.length ? (
        <div className="flex justify-between px-4 text-log text-text-subtle">
          {labels.map((label) => (
            <span key={label}>{label}</span>
          ))}
        </div>
      ) : null}
    </div>
  )
}

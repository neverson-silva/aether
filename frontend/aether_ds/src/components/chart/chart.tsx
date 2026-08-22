import type { HTMLAttributes } from 'react'
export interface ChartSeries {
  id: string
  label: string
  color?: string
  values: number[]
}
export interface ChartProps extends HTMLAttributes<HTMLDivElement> {
  series: ChartSeries[]
  labels?: string[]
  type?:
    | 'line'
    | 'area'
    | 'bar'
    | 'donut'
    | 'scatter'
    | 'gauge'
    | 'stacked'
    | 'composed'
  height?: number
  empty?: boolean
  loading?: boolean
  error?: string
  legend?: boolean
}
export function Chart({
  className = '',
  empty,
  error,
  height = 220,
  labels = [],
  legend = true,
  loading,
  series,
  type = 'line',
  ...props
}: ChartProps) {
  if (loading)
    return (
      <div
        className={`animate-pulse rounded-lg bg-surface-container ${className}`}
        style={{ height }}
      />
    )
  if (error)
    return (
      <div
        className={`flex items-center justify-center rounded-lg border border-status-danger/30 bg-status-danger-container/10 p-6 text-body-sm text-status-danger ${className}`}
        style={{ height }}
      >
        {error}
      </div>
    )
  if (empty || !series.length)
    return (
      <div
        className={`flex items-center justify-center rounded-lg border border-dashed border-border p-6 text-body-sm text-muted-foreground ${className}`}
        style={{ height }}
      >
        No data available
      </div>
    )
  const max = Math.max(...series.flatMap((item) => item.values), 1)
  const points = (values: number[]) =>
    values
      .map(
        (value, index) =>
          `${(index / Math.max(values.length - 1, 1)) * 100},${100 - (value / max) * 90}`,
      )
      .join(' ')
  const stackedMax = Math.max(
    ...labels.map((_, index) =>
      series.reduce((sum, item) => sum + (item.values[index] ?? 0), 0),
    ),
    max,
  )
  const isBar = type === 'bar' || type === 'stacked'
  const renderBars = (item: ChartSeries, index: number) =>
    item.values.map((value, valueIndex) => {
      const previous =
        type === 'stacked'
          ? series
              .slice(0, index)
              .reduce(
                (sum, current) => sum + (current.values[valueIndex] ?? 0),
                0,
              )
          : 0
      const chartMax = type === 'stacked' ? stackedMax : max
      return (
        <rect
          key={`${item.id}-${valueIndex}`}
          x={`${(valueIndex / item.values.length) * 100 + (type === 'stacked' ? 0 : index * 2)}`}
          y={100 - ((value + previous) / chartMax) * 90}
          width={Math.max(
            2,
            type === 'stacked'
              ? 60 / item.values.length
              : 60 / item.values.length / series.length,
          )}
          height={(value / chartMax) * 90}
          fill={item.color ?? 'var(--semantic-action-primary)'}
          rx="1"
        />
      )
    })
  return (
    <div
      className={`rounded-lg border border-border bg-surface-card p-4 ${className}`}
      {...props}
    >
      <svg
        viewBox="0 0 100 100"
        preserveAspectRatio="none"
        className="w-full"
        style={{ height }}
        role="img"
        aria-label={`${type} chart`}
      >
        {series.map((item, index) =>
          isBar ? (
            renderBars(item, index)
          ) : type === 'gauge' ? (
            <g key={item.id}>
              <path
                d="M 20 80 A 30 30 0 0 1 80 80"
                fill="none"
                stroke="currentColor"
                strokeOpacity=".2"
                strokeWidth="8"
              />
              <path
                d="M 20 80 A 30 30 0 0 1 80 80"
                fill="none"
                stroke={item.color ?? 'var(--semantic-action-primary)'}
                strokeWidth="8"
                pathLength="100"
                strokeDasharray={`${Math.min(100, ((item.values[0] ?? 0) / max) * 100)} 100`}
              />
            </g>
          ) : type === 'scatter' ? (
            item.values.map((value, valueIndex) => (
              <circle
                key={`${item.id}-${valueIndex}`}
                cx={`${(valueIndex / Math.max(item.values.length - 1, 1)) * 100}`}
                cy={100 - (value / max) * 90}
                r="1.7"
                fill={item.color ?? 'var(--semantic-action-primary)'}
              />
            ))
          ) : (
            <polyline
              key={item.id}
              points={points(item.values)}
              fill={
                type === 'area' || type === 'composed'
                  ? (item.color ?? 'var(--semantic-action-primary)')
                  : 'none'
              }
              fillOpacity={type === 'area' || type === 'composed' ? 0.15 : 0}
              stroke={item.color ?? 'var(--semantic-action-primary)'}
              strokeWidth="1.5"
              vectorEffect="non-scaling-stroke"
            />
          ),
        )}
      </svg>
      {legend ? (
        <div className="mt-3 flex flex-wrap gap-4">
          {series.map((item) => (
            <span
              key={item.id}
              className="inline-flex items-center gap-2 text-body-sm text-muted-foreground"
            >
              <span
                className="size-2 rounded-full"
                style={{
                  backgroundColor:
                    item.color ?? 'var(--semantic-action-primary)',
                }}
              />
              {item.label}
            </span>
          ))}
        </div>
      ) : null}
      {labels.length ? (
        <div className="mt-2 flex justify-between text-label-caps text-muted-foreground">
          {labels.map((label) => (
            <span key={label}>{label}</span>
          ))}
        </div>
      ) : null}
    </div>
  )
}

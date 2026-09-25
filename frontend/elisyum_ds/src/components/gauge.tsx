import type { HTMLAttributes } from 'react'

export interface GaugeProps extends Omit<HTMLAttributes<HTMLDivElement>, 'title'> {
  value: number
  max?: number
  label: string
  unit?: string
}

export function Gauge({
  className = '',
  label,
  max = 100,
  unit = '%',
  value,
  ...props
}: GaugeProps) {
  const safeMax = Math.max(1, max)
  const percentage = Math.min(100, Math.max(0, (value / safeMax) * 100))
  return (
    <div
      {...props}
      aria-label={label}
      aria-valuemax={safeMax}
      aria-valuemin={0}
      aria-valuenow={Math.min(safeMax, Math.max(0, value))}
      className={`grid justify-items-center gap-2 ${className}`}
      role="meter"
    >
      <div className="relative grid size-32 place-items-center">
        <svg
          aria-hidden="true"
          className="absolute inset-0 size-full -rotate-90"
        >
          <circle
            className="stroke-surface-3"
            cx="50%"
            cy="50%"
            fill="none"
            pathLength="100"
            r="44%"
            strokeWidth="8"
          />
          <circle
            className="stroke-action transition-[stroke-dashoffset] duration-[var(--ely-duration-standard)] ease-ely-out motion-reduce:transition-none"
            cx="50%"
            cy="50%"
            fill="none"
            pathLength="100"
            r="44%"
            strokeDasharray="100"
            strokeDashoffset={100 - percentage}
            strokeLinecap="round"
            strokeWidth="8"
          />
        </svg>
        <div className="grid size-24 place-items-center rounded-full bg-surface-1">
          <span className="font-technical text-metric text-text-primary">
            {value}
            {unit}
          </span>
        </div>
      </div>
      <span className="text-supporting text-text-tertiary">{label}</span>
    </div>
  )
}

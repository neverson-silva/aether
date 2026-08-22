import type { HTMLAttributes } from 'react'
export interface GaugeProps extends HTMLAttributes<HTMLDivElement> {
  value: number
  min?: number
  max?: number
  size?: number
  label?: string
  status?: 'default' | 'success' | 'warning' | 'danger'
}
export function Gauge({
  className = '',
  label,
  max = 100,
  min = 0,
  size = 160,
  status = 'default',
  value,
  ...props
}: GaugeProps) {
  const progress = Math.min(1, Math.max(0, (value - min) / (max - min)))
  const radius = 52
  const circumference = Math.PI * radius
  const colors = {
    default: 'text-primary',
    success: 'text-status-success',
    warning: 'text-status-warning',
    danger: 'text-status-danger',
  }
  return (
    <div
      className={`relative inline-flex items-center justify-center ${colors[status]} ${className}`}
      style={{ width: size, height: size / 2 + 16 }}
      {...props}
    >
      <svg viewBox="0 0 120 70" className="h-full w-full" aria-hidden="true">
        <path
          d="M 8 60 A 52 52 0 0 1 112 60"
          fill="none"
          stroke="currentColor"
          strokeOpacity=".2"
          strokeWidth="10"
          strokeLinecap="round"
        />
        <path
          d="M 8 60 A 52 52 0 0 1 112 60"
          fill="none"
          stroke="currentColor"
          strokeWidth="10"
          strokeLinecap="round"
          strokeDasharray={circumference}
          strokeDashoffset={circumference * (1 - progress)}
        />
      </svg>
      <div className="absolute bottom-0 text-center">
        <div className="text-headline-sm font-semibold text-foreground">
          {label ?? value}
        </div>
        <div className="text-label-caps text-muted-foreground">
          {min} to {max}
        </div>
      </div>
    </div>
  )
}

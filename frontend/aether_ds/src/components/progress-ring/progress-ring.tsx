import type { HTMLAttributes } from 'react'
export interface ProgressRingProps extends HTMLAttributes<HTMLDivElement> {
  value?: number
  max?: number
  size?: number
  strokeWidth?: number
  label?: string
  indeterminate?: boolean
  status?: 'default' | 'success' | 'warning' | 'danger'
}
export function ProgressRing({
  className = '',
  indeterminate,
  label,
  max = 100,
  size = 64,
  status = 'default',
  strokeWidth = 6,
  value = 0,
  ...props
}: ProgressRingProps) {
  const radius = (size - strokeWidth) / 2
  const circumference = 2 * Math.PI * radius
  const progress = Math.min(1, Math.max(0, value / max))
  const colors = {
    default: 'text-primary',
    success: 'text-status-success',
    warning: 'text-status-warning',
    danger: 'text-status-danger',
  }
  return (
    <div
      className={`relative inline-flex items-center justify-center ${colors[status]} ${className}`}
      style={{ width: size, height: size }}
      {...props}
    >
      <svg
        viewBox={`0 0 ${size} ${size}`}
        className={indeterminate ? 'animate-spin' : ''}
        aria-hidden="true"
      >
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke="currentColor"
          strokeOpacity=".2"
          strokeWidth={strokeWidth}
        />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke="currentColor"
          strokeWidth={strokeWidth}
          strokeLinecap="round"
          strokeDasharray={circumference}
          strokeDashoffset={
            indeterminate
              ? circumference * 0.75
              : circumference * (1 - progress)
          }
          transform={`rotate(-90 ${size / 2} ${size / 2})`}
        />
      </svg>
      <span className="absolute text-label-caps text-foreground">
        {label ?? `${Math.round(progress * 100)}%`}
      </span>
      <span className="sr-only">
        {label ?? `${Math.round(progress * 100)} percent`}
      </span>
    </div>
  )
}

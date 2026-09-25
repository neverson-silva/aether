import type { SVGAttributes } from 'react'

export interface ProgressRingProps
  extends Omit<SVGAttributes<SVGSVGElement>, 'aria-label'> {
  value?: number
  max?: number
  label?: string
  size?: number
  strokeWidth?: number
}

export function ProgressRing({
  className = '',
  label = 'Progress',
  max = 100,
  size = 48,
  strokeWidth = 4,
  value = 0,
  ...props
}: ProgressRingProps) {
  const radius = (size - strokeWidth) / 2
  const circumference = 2 * Math.PI * radius
  const percentage = Math.min(100, Math.max(0, (value / max) * 100))
  return (
    <svg
      {...props}
      aria-label={label}
      aria-valuemax={max}
      aria-valuemin={0}
      aria-valuenow={value}
      className={className}
      height={size}
      role="progressbar"
      viewBox={`0 0 ${size} ${size}`}
      width={size}
    >
      <circle
        aria-hidden="true"
        cx={size / 2}
        cy={size / 2}
        fill="none"
        r={radius}
        stroke="var(--ely-color-surface-3)"
        strokeWidth={strokeWidth}
      />
      <circle
        aria-hidden="true"
        className="transition-[stroke-dashoffset] duration-[var(--ely-duration-standard)] ease-ely-out motion-reduce:transition-none"
        cx={size / 2}
        cy={size / 2}
        fill="none"
        pathLength="100"
        r={radius}
        stroke="var(--ely-color-action)"
        strokeDasharray="100"
        strokeDashoffset={100 - percentage}
        strokeLinecap="round"
        strokeWidth={strokeWidth}
        transform={`rotate(-90 ${size / 2} ${size / 2})`}
      />
    </svg>
  )
}

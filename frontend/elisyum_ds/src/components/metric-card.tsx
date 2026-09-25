import type { ReactNode } from 'react'
import { Marker } from './marker'
import type { BadgeTone } from './badge'
import { Card } from './card'

export interface MetricCardProps {
  label: ReactNode
  value: ReactNode
  trend?: ReactNode
  tone?: BadgeTone
  status?: ReactNode
  className?: string
}

export function MetricCard({
  className = '',
  label,
  status,
  tone = 'neutral',
  trend,
  value,
}: MetricCardProps) {
  return (
    <Card className={`grid gap-4 ${className}`}>
      <div className="flex items-center justify-between gap-3">
        <span className="text-label text-text-tertiary">{label}</span>
        {status ? (
          <span className="inline-flex items-center gap-2 text-log text-text-secondary">
            <Marker
              aria-hidden="true"
              tone={tone === 'accent' ? 'accent' : tone}
            />
            {status}
          </span>
        ) : null}
      </div>
      <div className="font-technical text-metric text-text-primary">{value}</div>
      {trend ? (
        <div className="text-supporting text-text-secondary">{trend}</div>
      ) : null}
    </Card>
  )
}

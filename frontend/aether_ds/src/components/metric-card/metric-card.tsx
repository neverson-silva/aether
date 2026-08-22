import type { ReactNode } from 'react'
import { tv, type VariantProps } from 'tailwind-variants'

const metricCard = tv({
  base: 'rounded-lg border p-5',
  variants: {
    status: {
      default: 'border-border bg-surface-card',
      success: 'border-status-success/30 bg-status-success-container/10',
      warning: 'border-status-warning/30 bg-status-warning-container/10',
      danger: 'border-status-danger/30 bg-status-danger-container/10',
    },
  },
  defaultVariants: { status: 'default' },
})
export interface MetricCardProps extends VariantProps<typeof metricCard> {
  label: string
  value: ReactNode
  unit?: string
  delta?: ReactNode
  period?: string
  trend?: 'up' | 'down' | 'flat'
  target?: ReactNode
  footer?: ReactNode
  loading?: boolean
  empty?: boolean
}
export function MetricCard({
  delta,
  empty,
  footer,
  label,
  loading,
  period,
  status,
  target,
  trend,
  unit,
  value,
}: MetricCardProps) {
  return (
    <article className={metricCard({ status })}>
      {loading ? (
        <div className="space-y-3">
          <div className="h-3 w-24 animate-pulse rounded bg-surface-container" />
          <div className="h-8 w-32 animate-pulse rounded bg-surface-container" />
        </div>
      ) : empty ? (
        <div className="text-body-sm text-muted-foreground">No metric data</div>
      ) : (
        <>
          <div className="flex items-start justify-between gap-4">
            <div>
              <div className="text-body-sm text-muted-foreground">{label}</div>
              <div className="mt-2 text-headline-sm font-semibold text-foreground">
                {value}
                {unit ? (
                  <span className="ml-1 text-body-sm font-normal text-muted-foreground">
                    {unit}
                  </span>
                ) : null}
              </div>
            </div>
            {target ? (
              <div className="text-end text-body-sm text-muted-foreground">
                Target
                <span className="block font-semibold text-foreground">
                  {target}
                </span>
              </div>
            ) : null}
          </div>
          {delta || period ? (
            <div className="mt-3 flex items-center gap-2 text-body-sm">
              <span
                className={
                  trend === 'down'
                    ? 'text-status-danger'
                    : trend === 'up'
                      ? 'text-status-success'
                      : 'text-muted-foreground'
                }
              >
                {delta}
              </span>
              <span className="text-muted-foreground">{period}</span>
            </div>
          ) : null}
          {footer ? (
            <div className="mt-4 border-t border-border pt-3">{footer}</div>
          ) : null}
        </>
      )}
    </article>
  )
}

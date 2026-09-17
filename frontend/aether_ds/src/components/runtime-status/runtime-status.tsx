import { tv, type VariantProps } from 'tailwind-variants'

const statusDot = tv({
  base: 'size-2 rounded-full',
  variants: {
    status: {
      healthy: 'bg-status-success',
      deploying: 'bg-primary',
      degraded: 'bg-status-warning',
      failed: 'bg-status-danger',
      paused: 'bg-muted-foreground',
      unknown: 'bg-muted-foreground',
      validating: 'bg-primary',
      offline: 'bg-status-danger',
    },
  },
  defaultVariants: { status: 'unknown' },
})
export type RuntimeStatusValue =
  | 'healthy'
  | 'deploying'
  | 'degraded'
  | 'failed'
  | 'paused'
  | 'unknown'
  | 'validating'
  | 'offline'
export interface RuntimeStatusProps extends VariantProps<typeof statusDot> {
  label?: string
  live?: boolean
}
export function RuntimeStatus({
  label,
  live,
  status = 'unknown',
}: RuntimeStatusProps) {
  const text = label ?? status[0].toUpperCase() + status.slice(1)
  return (
    <span className="inline-flex items-center gap-2 text-body-sm text-foreground">
      <span className={`${statusDot({ status })} ${live ? 'ring-2 ring-current/20' : ''}`} />
      {text}
    </span>
  )
}

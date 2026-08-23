import type { HTMLAttributes, ReactNode } from 'react'
export interface InputGroupProps
  extends Omit<HTMLAttributes<HTMLDivElement>, 'prefix'> {
  prefix?: ReactNode
  suffix?: ReactNode
  unit?: string
  action?: ReactNode
}
export function InputGroup({
  action,
  children,
  className = '',
  prefix,
  suffix,
  unit,
  ...props
}: InputGroupProps) {
  return (
    <div
      className={`flex items-center rounded-md border border-border bg-surface-control focus-within:border-primary focus-within:ring-2 focus-within:ring-ring/20 ${className}`}
      {...props}
    >
      {prefix ? (
        <span className="pl-3 text-muted-foreground">{prefix}</span>
      ) : null}
      <div className="min-w-0 flex-1">{children}</div>
      {unit ? (
        <span className="pr-3 text-body-sm text-muted-foreground">{unit}</span>
      ) : null}
      {suffix ? (
        <span className="pr-3 text-muted-foreground">{suffix}</span>
      ) : null}
      {action}
    </div>
  )
}

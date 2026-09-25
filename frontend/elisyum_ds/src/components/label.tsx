import type { LabelHTMLAttributes } from 'react'

export type LabelProps = LabelHTMLAttributes<HTMLLabelElement>

export function Label({ className = '', ...props }: LabelProps) {
  return (
    <label
      {...props}
      className={`text-label text-text-secondary ${className}`}
    />
  )
}

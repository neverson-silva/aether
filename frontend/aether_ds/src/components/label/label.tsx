import type { LabelHTMLAttributes } from 'react'
import { tv, type VariantProps } from 'tailwind-variants'

const label = tv({
  base: 'inline-flex items-center gap-1 font-semibold',
  variants: {
    disabled: {
      true: 'text-muted-foreground opacity-60',
      false: 'text-foreground',
    },
    size: { sm: 'text-body-sm', md: 'text-body-md' },
  },
  defaultVariants: { disabled: false, size: 'sm' },
})
export interface LabelProps
  extends LabelHTMLAttributes<HTMLLabelElement>,
    VariantProps<typeof label> {
  required?: boolean
  optional?: boolean
  htmlFor: string
}
export function Label({
  children,
  className = '',
  disabled,
  optional,
  required,
  size,
  htmlFor,
  ...props
}: LabelProps) {
  return (
    <label
      htmlFor={htmlFor}
      className={label({ disabled, size, className })}
      {...props}
    >
      {children}
      {required ? (
        <span className="text-destructive" aria-hidden="true">
          *
        </span>
      ) : optional ? (
        <span className="font-normal text-muted-foreground">(optional)</span>
      ) : null}
    </label>
  )
}

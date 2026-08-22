import type { HTMLAttributes } from 'react'
import { tv, type VariantProps } from 'tailwind-variants'

const typography = tv({
  base: 'm-0',
  variants: {
    level: {
      display: 'font-display-lg text-display-lg',
      heading: 'font-semibold text-headline-sm',
      body: 'text-body-md',
      small: 'text-body-sm',
      label: 'font-label-caps text-label-caps uppercase',
      code: 'font-mono text-body-sm',
    },
    tone: {
      default: 'text-foreground',
      muted: 'text-muted-foreground',
      primary: 'text-primary',
      danger: 'text-destructive',
      success: 'text-status-success',
    },
    weight: {
      regular: 'font-normal',
      medium: 'font-medium',
      semibold: 'font-semibold',
      bold: 'font-bold',
    },
    align: { start: 'text-start', center: 'text-center', end: 'text-end' },
    truncate: { true: 'truncate', false: '' },
  },
  defaultVariants: {
    level: 'body',
    tone: 'default',
    weight: 'regular',
    align: 'start',
    truncate: false,
  },
})

type TypographyElement =
  | 'p'
  | 'span'
  | 'h1'
  | 'h2'
  | 'h3'
  | 'h4'
  | 'h5'
  | 'h6'
  | 'code'
  | 'label'
export interface TypographyProps
  extends HTMLAttributes<HTMLElement>,
    VariantProps<typeof typography> {
  as?: TypographyElement
}
export function Typography({
  as = 'p',
  className = '',
  level,
  tone,
  weight,
  align,
  truncate,
  ...props
}: TypographyProps) {
  const Component = as
  return (
    <Component
      className={typography({
        level,
        tone,
        weight,
        align,
        truncate,
        className,
      })}
      {...props}
    />
  )
}

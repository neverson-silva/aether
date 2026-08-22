import type { HTMLAttributes } from 'react'
import { tv, type VariantProps } from 'tailwind-variants'

const stack = tv({
  base: 'flex flex-col',
  variants: {
    gap: { none: 'gap-0', sm: 'gap-2', md: 'gap-4', lg: 'gap-6', xl: 'gap-10' },
    align: {
      start: 'items-start',
      center: 'items-center',
      end: 'items-end',
      stretch: 'items-stretch',
    },
  },
  defaultVariants: { gap: 'md', align: 'stretch' },
})
const inline = tv({
  base: 'flex flex-row',
  variants: {
    gap: { none: 'gap-0', sm: 'gap-2', md: 'gap-4', lg: 'gap-6', xl: 'gap-10' },
    align: {
      start: 'items-start',
      center: 'items-center',
      end: 'items-end',
      baseline: 'items-baseline',
    },
    wrap: { true: 'flex-wrap', false: 'flex-nowrap' },
  },
  defaultVariants: { gap: 'md', align: 'center', wrap: false },
})
const grid = tv({
  base: 'grid',
  variants: {
    columns: {
      one: 'grid-cols-1',
      two: 'grid-cols-1 md:grid-cols-2',
      three: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3',
      four: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-4',
    },
    gap: { sm: 'gap-2', md: 'gap-4', lg: 'gap-6' },
  },
  defaultVariants: { columns: 'one', gap: 'md' },
})

type LayoutProps = HTMLAttributes<HTMLDivElement>
export type StackProps = LayoutProps & VariantProps<typeof stack>
export type InlineProps = LayoutProps & VariantProps<typeof inline>
export type GridProps = LayoutProps & VariantProps<typeof grid>

export function Box({ className = '', ...props }: LayoutProps) {
  return <div className={className} {...props} />
}
export function Stack({ className = '', gap, align, ...props }: StackProps) {
  return <div className={stack({ gap, align, className })} {...props} />
}
export function Inline({
  className = '',
  gap,
  align,
  wrap,
  ...props
}: InlineProps) {
  return <div className={inline({ gap, align, wrap, className })} {...props} />
}
export function Grid({ className = '', columns, gap, ...props }: GridProps) {
  return <div className={grid({ columns, gap, className })} {...props} />
}
export function Container({ className = '', ...props }: LayoutProps) {
  return (
    <div
      className={`mx-auto w-full max-w-7xl px-4 md:px-8 ${className}`}
      {...props}
    />
  )
}
export function Bleed({ className = '', ...props }: LayoutProps) {
  return <div className={`-mx-4 md:-mx-8 ${className}`} {...props} />
}
export function Divider({ className = '', ...props }: LayoutProps) {
  return (
    <hr className={`h-px w-full border-0 bg-border ${className}`} {...props} />
  )
}
export function VisuallyHidden({ className = '', ...props }: LayoutProps) {
  return (
    <span
      className={`absolute h-px w-px overflow-hidden border-0 p-0 [clip:rect(0,0,0,0)] whitespace-nowrap ${className}`}
      {...props}
    />
  )
}

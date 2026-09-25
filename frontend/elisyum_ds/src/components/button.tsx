import type { ButtonHTMLAttributes, ReactNode } from 'react'
import { tv, type VariantProps } from 'tailwind-variants'

const buttonVariants = tv({
  base: 'inline-flex cursor-pointer items-center justify-center gap-2 rounded-lg border font-medium transition-[background-color,border-color,box-shadow,color,opacity,transform] duration-[var(--ely-duration-fast)] ease-ely-out motion-reduce:transition-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus disabled:cursor-not-allowed disabled:opacity-45',
  variants: {
    tone: {
      accent: 'border-action bg-action text-on-action hover:bg-action-strong',
          neutral:
            'border-border-default bg-surface-2 text-text-primary hover:border-border-strong hover:bg-surface-3',
          success: 'border-success-action bg-success-action text-on-action hover:brightness-110',
          danger: 'border-danger bg-danger text-on-action hover:brightness-110',
      ghost:
        'border-transparent text-text-secondary hover:border-border-subtle hover:bg-surface-2 hover:text-text-primary',
    },
    size: {
      sm: 'min-h-8 px-3 text-label',
      md: 'min-h-9 px-4 text-label',
      lg: 'min-h-11 px-5 text-body',
    },
  },
  defaultVariants: {
    tone: 'accent',
    size: 'md',
  },
})

export type ButtonTone = NonNullable<VariantProps<typeof buttonVariants>['tone']>
export type ButtonSize = NonNullable<VariantProps<typeof buttonVariants>['size']>

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  tone?: ButtonTone
  size?: ButtonSize
  loading?: boolean
  children: ReactNode
}

export function Button({
  className = '',
  disabled,
  loading = false,
  size = 'md',
  tone = 'accent',
  type = 'button',
  children,
  ...props
}: ButtonProps) {
  return (
    <button
      {...props}
      type={type}
      disabled={disabled || loading}
      className={buttonVariants({ className, size, tone })}
    >
      {loading ? (
        <span
          aria-hidden="true"
          className="size-3 motion-safe:animate-spin rounded-full border-2 border-current border-t-transparent"
        />
      ) : null}
      {children}
    </button>
  )
}

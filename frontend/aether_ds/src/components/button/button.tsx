import { Button as BaseButton } from '@base-ui/react/button'
import type { Icon } from '@phosphor-icons/react'
import { forwardRef, type ButtonHTMLAttributes } from 'react'
import { tv } from 'tailwind-variants'

const button = tv({
  base: 'inline-flex items-center justify-center gap-2 font-semibold transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-50',
  variants: {
    variant: {
      primary:
        'bg-primary text-primary-foreground shadow-sm hover:bg-primary/90 active:bg-primary/80',
      secondary:
        'border border-primary bg-button-secondary text-primary hover:bg-button-secondary-hover',
      ghost: 'text-button-quiet-foreground hover:bg-button-quiet-hover',
      danger:
        '!bg-[var(--semantic-action-danger)] !text-[var(--semantic-status-danger-content-on-action)] shadow-sm hover:!bg-[var(--semantic-action-danger-hover)] active:!bg-[var(--semantic-action-danger-active)]',
      success:
        'bg-button-success text-button-success-foreground shadow-sm hover:bg-button-success-hover active:bg-button-success-active',
      quiet: 'text-button-quiet-foreground hover:bg-button-quiet-hover',
      outline:
        'border border-primary bg-transparent text-primary hover:bg-button-secondary-hover',
    },
    size: {
      sm: 'h-8 rounded-md px-3 text-body-sm',
      md: 'h-10 rounded-md px-4 text-body-md',
      lg: 'h-12 rounded-lg px-5 text-body-md',
    },
  },
  defaultVariants: {
    variant: 'primary',
    size: 'md',
  },
})

export type ButtonVariant =
  | 'primary'
  | 'secondary'
  | 'ghost'
  | 'danger'
  | 'success'
  | 'quiet'
  | 'outline'
export type ButtonSize = 'sm' | 'md' | 'lg'

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant
  size?: ButtonSize
  loading?: boolean
  loadingLabel?: string
  icon?: Icon
  iconPosition?: 'start' | 'end'
  fullWidth?: boolean
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button({
  children,
  className = '',
  disabled,
  loading = false,
  loadingLabel,
  icon: IconComponent,
  iconPosition = 'start',
  fullWidth = false,
  size = 'md',
  type = 'button',
  variant = 'primary',
  ...props
}, ref) {
  return (
    <BaseButton
      className={button({
        variant,
        size,
        className: `${fullWidth ? 'w-full' : ''} ${loading ? 'disabled:!opacity-100' : ''} ${className}`,
      })}
      type={type}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      ref={ref}
      {...props}
    >
      {loading ? (
        <span
          className="size-4 animate-spin rounded-full border-2 border-current border-t-transparent"
          aria-hidden="true"
        />
      ) : IconComponent && iconPosition === 'start' ? (
        <IconComponent size={size === 'sm' ? 16 : 18} aria-hidden="true" />
      ) : null}
      {children}
      {!loading && IconComponent && iconPosition === 'end' ? (
        <IconComponent size={size === 'sm' ? 16 : 18} aria-hidden="true" />
      ) : null}
      {loadingLabel && loading ? (
        <span className="sr-only">{loadingLabel}</span>
      ) : null}
    </BaseButton>
  )
})

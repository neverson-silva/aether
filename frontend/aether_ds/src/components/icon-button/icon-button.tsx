import { Button as BaseButton } from '@base-ui/react/button'
import type { Icon } from '@phosphor-icons/react'
import { forwardRef, type ButtonHTMLAttributes } from 'react'

export interface IconButtonProps
  extends ButtonHTMLAttributes<HTMLButtonElement> {
  icon: Icon
  label: string
  size?: 'sm' | 'md' | 'lg'
  pressed?: boolean
  loading?: boolean
}
export const IconButton = forwardRef<HTMLButtonElement, IconButtonProps>(function IconButton({
  icon: IconComponent,
  label: accessibleLabel,
  className = '',
  disabled,
  loading,
  pressed,
  size = 'md',
  type = 'button',
  ...props
}, ref) {
  const sizes = { sm: 'size-8', md: 'size-10', lg: 'size-12' }
  const iconSizes = { sm: 16, md: 20, lg: 24 }
  return (
    <BaseButton
      type={type}
      className={`inline-flex ${sizes[size]} items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-surface-container hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 ${className}`}
      aria-label={accessibleLabel}
      aria-pressed={pressed}
      aria-busy={loading || undefined}
      disabled={disabled || loading}
      ref={ref}
      {...props}
    >
      {loading ? (
        <span
          className="size-4 animate-spin rounded-full border-2 border-current border-t-transparent"
          aria-hidden="true"
        />
      ) : (
        <IconComponent size={iconSizes[size]} aria-hidden="true" />
      )}
    </BaseButton>
  )
})

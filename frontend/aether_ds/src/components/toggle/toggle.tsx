import { Toggle as BaseToggle } from '@base-ui/react/toggle'
import type { ButtonHTMLAttributes } from 'react'
export interface ToggleProps
  extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'value'> {
  pressed?: boolean
  defaultPressed?: boolean
  onPressedChange?: (pressed: boolean) => void
  size?: 'sm' | 'md' | 'lg'
  value?: string
}
export function Toggle({ className = '', size = 'md', ...props }: ToggleProps) {
  const sizes = { sm: 'h-8 px-2', md: 'h-10 px-3', lg: 'h-12 px-4' }
  return (
    <BaseToggle
      className={`inline-flex items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-surface-container data-[pressed]:bg-primary/10 data-[pressed]:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50 ${sizes[size]} ${className}`}
      {...props}
    />
  )
}

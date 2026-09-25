import { tv, type VariantProps } from 'tailwind-variants'
import type { ButtonHTMLAttributes } from 'react'

const toggleVariants = tv({
  base: 'inline-flex min-h-9 cursor-pointer items-center justify-center rounded-md px-3 text-label transition-[background-color,box-shadow,color] duration-[var(--ely-duration-fast)] ease-ely-out motion-reduce:transition-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus disabled:cursor-not-allowed disabled:opacity-45',
  variants: {
    pressed: {
      true: 'bg-action-soft text-action-strong shadow-elevation-1',
      false: 'text-text-secondary hover:bg-surface-2 hover:text-text-primary',
    },
  },
  defaultVariants: { pressed: false },
})

export interface ToggleProps
  extends ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof toggleVariants> {
  pressed?: boolean
  onPressedChange?: (pressed: boolean) => void
}

export function Toggle({
  className = '',
  onClick,
  onPressedChange,
  pressed = false,
  type = 'button',
  ...props
}: ToggleProps) {
  return (
    <button
      {...props}
      aria-pressed={pressed}
      type={type}
      onClick={(event) => {
        onPressedChange?.(!pressed)
        onClick?.(event)
      }}
      className={toggleVariants({ className, pressed })}
    />
  )
}

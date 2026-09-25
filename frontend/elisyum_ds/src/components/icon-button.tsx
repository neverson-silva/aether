import type { ButtonHTMLAttributes, ReactNode } from 'react'

export interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  label: string
  children: ReactNode
  size?: 'sm' | 'md' | 'lg'
}

export function IconButton({
  className = '',
  label,
  size = 'md',
  children,
  type = 'button',
  ...props
}: IconButtonProps) {
  const sizeClass = size === 'sm' ? 'size-8' : size === 'lg' ? 'size-11' : 'size-9'
  return (
    <button
      {...props}
      aria-label={label}
      type={type}
      className={`inline-flex ${sizeClass} cursor-pointer items-center justify-center rounded-lg border border-transparent text-text-secondary transition-[background-color,border-color,color,transform] duration-[var(--ely-duration-fast)] ease-ely-out hover:border-border-subtle hover:bg-surface-2 hover:text-text-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus disabled:cursor-not-allowed disabled:opacity-45 ${className}`}
    >
      {children}
    </button>
  )
}

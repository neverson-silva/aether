import type { ButtonHTMLAttributes, ReactNode } from 'react'

export interface HeaderActionProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  label: string
  children: ReactNode
}

export function HeaderAction({ children, className = '', label, ...props }: HeaderActionProps) {
  return <button aria-label={label} className={`inline-flex size-10 cursor-pointer items-center justify-center rounded-xl bg-transparent text-text-secondary transition-[background-color,border-color,color,transform] duration-[var(--ely-duration-fast)] ease-ely-out hover:bg-surface-2 hover:text-text-primary active:bg-surface-3 motion-safe:active:scale-[var(--ely-motion-press-scale)] focus-visible:outline-2 focus-visible:outline-focus disabled:cursor-not-allowed disabled:opacity-45 ${className}`} {...props}>{children}</button>
}

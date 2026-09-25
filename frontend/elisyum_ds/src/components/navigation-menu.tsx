import type { ReactNode } from 'react'

export interface NavigationMenuItem {
  label: ReactNode
  href: string
  active?: boolean
  description?: ReactNode
}
export interface NavigationMenuProps {
  items: NavigationMenuItem[]
  className?: string
}

export function NavigationMenu({ className = '', items }: NavigationMenuProps) {
  return (
    <nav
      aria-label="Navigation menu"
      className={`flex flex-wrap items-center gap-1 ${className}`}
    >
      {items.map((item) => (
        <a
          aria-current={item.active ? 'page' : undefined}
          className={`cursor-pointer rounded-md px-3 py-2 text-supporting transition-[background-color,color,transform] duration-[var(--ely-duration-fast)] active:scale-[0.98] motion-reduce:transition-none motion-reduce:active:scale-100 ${item.active ? 'bg-action-soft text-action-strong' : 'text-text-secondary hover:bg-surface-2 hover:text-text-primary'} focus-visible:outline-2 focus-visible:outline-focus`}
          href={item.href}
          key={item.href}
          title={item.description ? String(item.description) : undefined}
        >
          {item.label}
        </a>
      ))}
    </nav>
  )
}

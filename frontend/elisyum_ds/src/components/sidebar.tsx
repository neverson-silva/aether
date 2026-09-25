import type { ReactNode } from 'react'
import { tv, type VariantProps } from 'tailwind-variants'

const sidebarVariants = tv({
  base: 'flex min-h-full flex-col border-r border-border-subtle bg-surface-0',
  variants: { collapsed: { true: 'w-16', false: 'w-64' } },
  defaultVariants: { collapsed: false },
})
export interface SidebarItem {
  label: ReactNode
  href: string
  active?: boolean
  icon?: ReactNode
}
export interface SidebarProps extends VariantProps<typeof sidebarVariants> {
  items: SidebarItem[]
  footer?: ReactNode
  brand?: ReactNode
  className?: string
}

export function Sidebar({
  brand,
  className = '',
  collapsed = false,
  footer,
  items,
}: SidebarProps) {
  return (
    <aside
      data-material="translucent"
      className={sidebarVariants({
        className: `bg-surface-0/85 backdrop-blur-[18px] transition-[width,background-color] duration-[var(--ely-duration-overlay)] motion-reduce:transition-none ${className}`,
        collapsed,
      })}
    >
      <div className="flex min-h-14 items-center gap-2 border-b border-border-subtle px-4 text-text-primary">
        {brand}
      </div>
      <nav
        aria-label="Primary"
        className="grid gap-1 p-3"
      >
        {items.map((item) => (
          <a
            aria-current={item.active ? 'page' : undefined}
            className={`relative flex min-h-9 cursor-pointer items-center gap-3 rounded-sm px-3 text-supporting transition-[background-color,color,transform] duration-[var(--ely-duration-fast)] motion-reduce:transition-none before:absolute before:inset-y-1 before:left-0 before:w-0.5 before:rounded-full before:bg-action before:opacity-0 before:transition-opacity before:duration-[var(--ely-duration-fast)] before:motion-reduce:transition-none ${item.active ? 'bg-action-soft text-action-strong before:opacity-100' : 'text-text-secondary hover:bg-surface-2 hover:text-text-primary'}`}
            href={item.href}
            key={item.href}
          >
            {item.icon}
            <span className={collapsed ? 'sr-only' : undefined}>{item.label}</span>
          </a>
        ))}
      </nav>
      {footer ? (
        <div className="mt-auto border-t border-border-subtle p-3">{footer}</div>
      ) : null}
    </aside>
  )
}

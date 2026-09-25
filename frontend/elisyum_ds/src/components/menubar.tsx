import type { ReactNode } from 'react'

export interface MenubarItem {
  label: ReactNode
  onSelect?: () => void
  disabled?: boolean
}
export interface MenubarProps {
  items: MenubarItem[]
  className?: string
}

export function Menubar({ className = '', items }: MenubarProps) {
  return (
    <div
      aria-label="Application menu"
      role="menubar"
      className={`flex items-center gap-1 ${className}`}
    >
      {items.map((item, index) => (
        <button
          aria-disabled={item.disabled || undefined}
          className="cursor-pointer rounded-md px-3 py-2 text-supporting text-text-secondary transition-[background-color,color,transform] duration-[var(--ely-duration-fast)] hover:bg-surface-2 hover:text-text-primary active:scale-[0.98] focus-visible:outline-2 focus-visible:outline-focus motion-reduce:transition-none motion-reduce:active:scale-100 disabled:cursor-not-allowed disabled:opacity-45"
          disabled={item.disabled}
          key={index}
          onClick={item.onSelect}
          role="menuitem"
          type="button"
        >
          {item.label}
        </button>
      ))}
    </div>
  )
}

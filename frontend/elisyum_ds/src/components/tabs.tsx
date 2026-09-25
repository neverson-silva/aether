import type { ReactNode } from 'react'
import { Tabs as BaseTabs } from '@base-ui/react/tabs'

export interface TabItem {
  value: string
  label: ReactNode
  content: ReactNode
  disabled?: boolean
}

export interface TabsProps {
  items: TabItem[]
  value?: string
  defaultValue?: string
  onValueChange?: (value: string) => void
  className?: string
}

export function Tabs({
  className = '',
  defaultValue,
  items,
  onValueChange,
  value,
}: TabsProps) {
  return (
    <BaseTabs.Root
      className={`grid gap-4 ${className}`}
      defaultValue={defaultValue}
      onValueChange={(nextValue) => onValueChange?.(String(nextValue))}
      value={value}
    >
      <BaseTabs.List className="flex gap-1 overflow-x-auto overflow-y-hidden border-b border-border-subtle">
        <>
          {items.map((item) => (
            <BaseTabs.Tab
              className="relative cursor-pointer select-none whitespace-nowrap border-b-2 border-transparent bg-transparent px-3 py-2 text-label text-text-tertiary transition-[color,transform] duration-[var(--ely-duration-fast)] hover:text-text-primary active:scale-[0.98] data-[active]:bg-transparent data-[active]:text-action-strong after:absolute after:inset-x-2 after:bottom-0 after:h-0.5 after:origin-center after:scale-x-0 after:bg-action after:transition-transform after:duration-[var(--ely-duration-fast)] data-[active]:after:scale-x-100 focus-visible:bg-transparent focus-visible:outline-2 focus-visible:outline-focus motion-reduce:transition-none motion-reduce:active:scale-100 motion-reduce:after:transition-none disabled:cursor-not-allowed"
              disabled={item.disabled}
              key={item.value}
              value={item.value}
            >
              {item.label}
            </BaseTabs.Tab>
          ))}
        </>
      </BaseTabs.List>
      {items.map((item) => (
        <BaseTabs.Panel
          className="text-body text-text-secondary transition-opacity duration-[var(--ely-duration-standard)] data-[starting-style]:opacity-0 data-[ending-style]:opacity-0 motion-reduce:transition-none"
          key={item.value}
          value={item.value}
        >
          {item.content}
        </BaseTabs.Panel>
      ))}
    </BaseTabs.Root>
  )
}

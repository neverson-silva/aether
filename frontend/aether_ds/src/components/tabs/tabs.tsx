import { Tabs as BaseTabs } from '@base-ui/react/tabs'
import type { ReactNode } from 'react'
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
  activation?: 'automatic' | 'manual'
  onValueChange?: (value: string) => void
  variant?: 'underline' | 'pill'
}
export function Tabs({
  activation = 'automatic',
  defaultValue,
  items,
  onValueChange,
  value,
  variant = 'underline',
}: TabsProps) {
  return (
    <BaseTabs.Root
      value={value}
      defaultValue={defaultValue}
      onValueChange={onValueChange}
    >
      <BaseTabs.List
        activateOnFocus={activation === 'automatic'}
        className={`flex gap-1 border-b border-border ${variant === 'pill' ? 'rounded-md bg-surface-container p-1' : ''}`}
      >
        {items.map((item) => (
          <BaseTabs.Tab
            key={item.value}
            value={item.value}
            disabled={item.disabled}
            className={`px-3 py-2 text-body-sm text-muted-foreground data-[active]:text-primary ${variant === 'underline' ? 'border-b-2 border-transparent data-[active]:border-primary' : 'rounded data-[active]:bg-surface-card'}`}
          >
            {item.label}
          </BaseTabs.Tab>
        ))}
      </BaseTabs.List>
      {items.map((item) => (
        <BaseTabs.Panel
          key={item.value}
          value={item.value}
          className="pt-4 focus-visible:outline-none"
        >
          {item.content}
        </BaseTabs.Panel>
      ))}
    </BaseTabs.Root>
  )
}

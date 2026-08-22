import { Accordion as BaseAccordion } from '@base-ui/react/accordion'
import type { ReactNode } from 'react'
export interface AccordionItem {
  value: string
  title: ReactNode
  content: ReactNode
  disabled?: boolean
}
export interface AccordionProps {
  items: AccordionItem[]
  multiple?: boolean
  defaultValue?: string[]
  value?: string[]
  onValueChange?: (value: string[]) => void
}
export function Accordion({
  defaultValue,
  items,
  multiple,
  onValueChange,
  value,
}: AccordionProps) {
  return (
    <BaseAccordion.Root
      multiple={multiple}
      defaultValue={defaultValue}
      value={value}
      onValueChange={onValueChange}
      className="divide-y divide-border rounded-lg border border-border"
    >
      {items.map((item) => (
        <BaseAccordion.Item
          key={item.value}
          value={item.value}
          disabled={item.disabled}
        >
          <BaseAccordion.Header>
            <BaseAccordion.Trigger className="flex w-full items-center justify-between px-4 py-3 text-start font-semibold text-foreground hover:bg-surface-container disabled:opacity-50">
              {item.title}
              <span aria-hidden="true">⌄</span>
            </BaseAccordion.Trigger>
          </BaseAccordion.Header>
          <BaseAccordion.Panel className="overflow-hidden px-4 pb-4 text-body-sm text-muted-foreground">
            {item.content}
          </BaseAccordion.Panel>
        </BaseAccordion.Item>
      ))}
    </BaseAccordion.Root>
  )
}

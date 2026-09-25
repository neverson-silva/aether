import type { ReactNode } from 'react'
import { Accordion as BaseAccordion } from '@base-ui/react/accordion'

export interface AccordionItemData {
  value: string
  title: ReactNode
  content: ReactNode
  disabled?: boolean
}

export interface AccordionProps {
  items: AccordionItemData[]
  multiple?: boolean
  defaultValue?: string[]
  value?: string[]
  onValueChange?: (value: string[]) => void
  className?: string
}

export function Accordion({
  className = '',
  defaultValue,
  items,
  multiple = false,
  onValueChange,
  value,
}: AccordionProps) {
  return (
    <BaseAccordion.Root
      className={`grid divide-y divide-border-subtle overflow-hidden rounded-lg border border-border-subtle ${className}`}
      defaultValue={defaultValue}
      multiple={multiple}
      onValueChange={(nextValue) => onValueChange?.(nextValue as string[])}
      value={value}
    >
      {items.map((item) => (
        <BaseAccordion.Item
          className="overflow-hidden bg-surface-1"
          disabled={item.disabled}
          key={item.value}
          value={item.value}
        >
          <BaseAccordion.Header>
            <BaseAccordion.Trigger className="flex w-full cursor-pointer items-center justify-between px-4 py-3 text-left text-body text-text-primary transition-[background-color,color,transform] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:bg-surface-2 focus-visible:outline-2 focus-visible:outline-focus motion-safe:active:scale-[var(--ely-motion-press-scale)] motion-reduce:active:scale-100 disabled:cursor-not-allowed">
              {item.title}
              <span
                aria-hidden="true"
                className="text-action-strong transition-transform duration-[var(--ely-duration-fast)] motion-reduce:transition-none data-[state=open]:rotate-45"
              >
                +
              </span>
            </BaseAccordion.Trigger>
          </BaseAccordion.Header>
          <BaseAccordion.Panel className="h-[var(--accordion-panel-height)] overflow-hidden border-t border-border-subtle px-4 py-3 text-supporting text-text-secondary transition-[height,opacity] duration-[var(--ely-duration-standard)] ease-ely-out motion-reduce:transition-opacity data-[ending-style]:h-0 data-[ending-style]:opacity-0 data-[starting-style]:h-0 data-[starting-style]:opacity-0">
            {item.content}
          </BaseAccordion.Panel>
        </BaseAccordion.Item>
      ))}
    </BaseAccordion.Root>
  )
}

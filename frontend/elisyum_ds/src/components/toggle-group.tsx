import { useState, type ReactNode } from 'react'
import { Toggle } from './toggle'

export interface ToggleGroupItem {
  value: string
  label: ReactNode
  disabled?: boolean
}

export interface ToggleGroupProps {
  items: ToggleGroupItem[]
  value?: string
  defaultValue?: string
  onValueChange?: (value: string) => void
  className?: string
}

export function ToggleGroup({
  className = '',
  defaultValue = '',
  items,
  onValueChange,
  value: controlledValue,
}: ToggleGroupProps) {
  const [uncontrolledValue, setUncontrolledValue] = useState(defaultValue)
  const value = controlledValue ?? uncontrolledValue
  const select = (nextValue: string) => {
    if (controlledValue === undefined) setUncontrolledValue(nextValue)
    onValueChange?.(nextValue)
  }
  return (
    <fieldset
      aria-label="Toggle group"
      className={`inline-flex rounded-md border border-border-subtle bg-surface-2 p-1 ${className}`}
    >
      {items.map((item) => (
        <Toggle
          key={item.value}
          disabled={item.disabled}
          pressed={value === item.value}
          onPressedChange={() => select(item.value)}
        >
          {item.label}
        </Toggle>
      ))}
    </fieldset>
  )
}

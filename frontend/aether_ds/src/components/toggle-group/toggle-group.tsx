import { Toggle as BaseToggle } from '@base-ui/react/toggle'
import { ToggleGroup as BaseToggleGroup } from '@base-ui/react/toggle-group'
import type { ReactNode } from 'react'

export interface ToggleGroupProps {
  options: { value: string; label: ReactNode; disabled?: boolean }[]
  value?: string | string[]
  defaultValue?: string | string[]
  multiple?: boolean
  onValueChange?: (value: string | string[]) => void
}
export function ToggleGroup({
  defaultValue,
  multiple,
  onValueChange,
  options,
  value,
}: ToggleGroupProps) {
  const currentValue = value
    ? Array.isArray(value)
      ? value
      : [value]
    : undefined
  const initialValue = defaultValue
    ? Array.isArray(defaultValue)
      ? defaultValue
      : [defaultValue]
    : undefined
  return (
    <BaseToggleGroup
      value={currentValue}
      defaultValue={initialValue}
      multiple={multiple}
      onValueChange={(next) =>
        onValueChange?.(multiple ? next : (next[0] ?? ''))
      }
      className="inline-flex rounded-md border border-border p-1"
    >
      {options.map((option) => (
        <BaseToggle
          key={option.value}
          value={option.value}
          disabled={option.disabled}
          className="rounded px-3 py-1.5 text-body-sm text-muted-foreground data-[pressed]:bg-primary/10 data-[pressed]:text-primary disabled:opacity-50"
        >
          {option.label}
        </BaseToggle>
      ))}
    </BaseToggleGroup>
  )
}

import { Select as BaseSelect } from '@base-ui/react/select'
import { CaretDown, Check } from '@phosphor-icons/react'
import type { ReactNode } from 'react'
import { Field } from '../field/field'
export interface SelectOption {
  value: string
  label: ReactNode
  disabled?: boolean
  group?: string
}
export interface SelectProps {
  label?: string
  description?: string
  error?: string
  placeholder?: string
  options: SelectOption[]
  value?: string
  defaultValue?: string
  disabled?: boolean
  onValueChange?: (value: string | null) => void
}
export function Select({
  description,
  disabled,
  error,
  label,
  onValueChange,
  options,
  placeholder = 'Select an option',
  value,
  defaultValue,
}: SelectProps) {
  const control = (
    <BaseSelect.Root
      value={value}
      defaultValue={defaultValue}
      onValueChange={onValueChange}
    >
      <BaseSelect.Trigger
        disabled={disabled}
        className="flex h-10 w-full items-center justify-between rounded-md border border-border bg-surface-control px-3 hover:bg-surface-container-highest/40 text-body-md text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
      >
        <BaseSelect.Value placeholder={placeholder} />
        <BaseSelect.Icon>
          <CaretDown size={16} aria-hidden="true" />
        </BaseSelect.Icon>
      </BaseSelect.Trigger>
      <BaseSelect.Portal>
        <BaseSelect.Positioner className="z-[100]">
          <BaseSelect.Popup className="min-w-[var(--anchor-width)] rounded-md border border-border bg-surface-popover p-1 text-foreground shadow-lg">
            <BaseSelect.List>
              {options.map((option) => (
                <BaseSelect.Item
                  key={option.value}
                  value={option.value}
                  disabled={option.disabled}
                  className="flex cursor-pointer items-center rounded px-3 py-2 text-body-sm data-[highlighted]:bg-surface-container data-[selected]:text-primary data-[disabled]:opacity-50"
                >
                  <BaseSelect.ItemText>{option.label}</BaseSelect.ItemText>
                  <BaseSelect.ItemIndicator className="ml-auto">
                    <Check size={16} aria-hidden="true" />
                  </BaseSelect.ItemIndicator>
                </BaseSelect.Item>
              ))}
            </BaseSelect.List>
          </BaseSelect.Popup>
        </BaseSelect.Positioner>
      </BaseSelect.Portal>
    </BaseSelect.Root>
  )
  return label ? (
    <Field
      label={label}
      description={description}
      error={error}
      disabled={disabled}
    >
      {control}
    </Field>
  ) : (
    control
  )
}

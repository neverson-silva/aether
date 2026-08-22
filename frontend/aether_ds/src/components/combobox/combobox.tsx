import { Combobox as BaseCombobox } from '@base-ui/react/combobox'
import { CaretDown, Check } from '@phosphor-icons/react'
import { type ReactNode, useState } from 'react'
import { Field } from '../field/field'
export interface ComboboxOption {
  value: string
  label: ReactNode
  disabled?: boolean
}
export interface ComboboxProps {
  label?: string
  description?: string
  error?: string
  placeholder?: string
  options: ComboboxOption[]
  value?: string | null
  defaultValue?: string | null
  multiple?: boolean
  disabled?: boolean
  onValueChange?: (value: string | string[] | null) => void
  onInputValueChange?: (value: string) => void
}
export function Combobox({
  description,
  disabled,
  error,
  label,
  onInputValueChange,
  onValueChange,
  options,
  placeholder = 'Search options',
  value,
  defaultValue,
  multiple,
}: ComboboxProps) {
  const [filterQuery, setFilterQuery] = useState('')
  const filteredOptions = options.filter((option) => {
    if (!filterQuery.trim()) return true
    return typeof option.label === 'string' && option.label.toLowerCase().includes(filterQuery.trim().toLowerCase())
  })
  const popup = (
    <BaseCombobox.Positioner
      side="bottom"
      align="start"
      positionMethod="fixed"
      collisionAvoidance={{ side: 'none', align: 'shift', fallbackAxisSide: 'none' }}
      style={{ zIndex: 2147483002 }}
      className="z-[110] max-w-[calc(100vw-2rem)] outline-none"
    >
      <BaseCombobox.Popup style={{ backgroundColor: 'var(--semantic-surface-popover)', opacity: 1, maxHeight: 'min(var(--available-height), 20rem)', overflowY: 'auto', overflowX: 'hidden' }} className="w-[var(--anchor-width)] max-w-[calc(100vw-2rem)] max-h-[min(var(--available-height),20rem)] overflow-x-hidden overflow-y-auto rounded-md border border-border bg-surface-popover p-1 text-foreground shadow-lg">
        <BaseCombobox.Empty className="px-3 py-2 text-body-sm text-muted-foreground">
          No results
        </BaseCombobox.Empty>
        <BaseCombobox.List>
          {filteredOptions.map((option) => (
            <BaseCombobox.Item
              key={option.value}
              value={option.value}
              disabled={option.disabled}
              className="flex cursor-pointer items-center rounded px-3 py-2 text-body-sm data-[highlighted]:bg-surface-container data-[disabled]:opacity-50"
            >
              <BaseCombobox.ItemIndicator className="mr-2">
                <Check size={16} aria-hidden="true" />
              </BaseCombobox.ItemIndicator>
              {option.label}
            </BaseCombobox.Item>
          ))}
        </BaseCombobox.List>
      </BaseCombobox.Popup>
    </BaseCombobox.Positioner>
  )
  const control = (
    <BaseCombobox.Root
      value={value}
      defaultValue={defaultValue}
      multiple={multiple}
      itemToStringLabel={(itemValue) => {
        const option = options.find((item) => String(item.value) === String(itemValue))
        return typeof option?.label === 'string' ? option.label : String(itemValue)
      }}
      filter={null}
      onValueChange={(nextValue) => {
        setFilterQuery('')
        onValueChange?.(nextValue)
      }}
      onInputValueChange={(nextQuery) => {
        setFilterQuery(nextQuery)
        onInputValueChange?.(nextQuery)
      }}
    >
      <div style={{ position: 'relative', width: '100%' }} className="relative w-full">
        <BaseCombobox.Input
          disabled={disabled}
          placeholder={placeholder}
          className="h-10 w-full rounded-md border border-border bg-surface-card px-3 pr-9 text-body-md outline-none focus:border-primary focus:ring-2 focus:ring-primary/20 disabled:opacity-50"
        />
        <CaretDown
          size={16}
          style={{ position: 'absolute', right: '0.75rem', top: '50%', transform: 'translateY(-50%)' }}
          className="pointer-events-none text-muted-foreground"
          aria-hidden="true"
        />
      </div>
      <BaseCombobox.Portal>{popup}</BaseCombobox.Portal>
    </BaseCombobox.Root>
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

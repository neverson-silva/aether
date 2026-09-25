import { CaretDown, Check } from '@phosphor-icons/react'
import { Combobox as BaseCombobox } from '@base-ui/react/combobox'
import { forwardRef, useMemo, type InputHTMLAttributes, type ReactNode } from 'react'
import { inputControlClasses } from './input-styles'

export interface ComboboxOption {
  value: string
  label: ReactNode
}
export interface ComboboxProps
  extends Omit<InputHTMLAttributes<HTMLInputElement>, 'list'> {
  options: ComboboxOption[]
  listId?: string
}

export const Combobox = forwardRef<HTMLInputElement, ComboboxProps>(function ComboboxView(
  { className = '', disabled, id, options, value, defaultValue, onChange, ...props },
  ref,
) {
  const optionMap = useMemo(
    () => new Map(options.map((option) => [option.value, option])),
    [options],
  )
  const inputValue = typeof value === 'string' ? value : undefined
  return (
    <BaseCombobox.Root
      defaultValue={typeof defaultValue === 'string' ? defaultValue : undefined}
      disabled={disabled}
      onInputValueChange={(nextValue) =>
        onChange?.({
          target: { value: nextValue },
          currentTarget: { value: nextValue },
        } as never)
      }
      onValueChange={(nextValue) => {
        const option = optionMap.get(String(nextValue))
        if (option)
          onChange?.({
            target: { value: option.value },
            currentTarget: { value: option.value },
          } as never)
      }}
      value={inputValue}
      itemToStringLabel={(item) => String(optionMap.get(String(item))?.label ?? item)}
    >
      <div className="relative">
        <BaseCombobox.Input
          {...props}
          id={id}
          ref={ref}
          className={`${inputControlClasses} pr-10 ${className}`}
        />
        <BaseCombobox.Trigger
          aria-label="Show options"
          className="absolute inset-y-0 right-0 grid w-10 place-items-center text-text-tertiary transition-colors motion-reduce:transition-none hover:text-text-primary focus-visible:outline-2 focus-visible:outline-focus"
        >
          <CaretDown
            aria-hidden="true"
            size={16}
            weight="bold"
          />
        </BaseCombobox.Trigger>
      </div>
      <BaseCombobox.Portal>
        <BaseCombobox.Positioner
          className="z-30"
          sideOffset={6}
        >
          <BaseCombobox.Popup className="min-w-[var(--anchor-width)] origin-[var(--transform-origin)] overflow-hidden rounded-md border border-border-default bg-overlay p-1 text-text-primary shadow-elevation-2 transition-[opacity,transform] duration-[var(--ely-duration-popover)] ease-ely-out data-[ending-style]:scale-[.98] data-[ending-style]:opacity-0 data-[starting-style]:scale-[.98] data-[starting-style]:opacity-0 motion-reduce:transition-opacity">
            <BaseCombobox.List className="grid max-h-64 overflow-auto">
              {options.map((option) => (
                <BaseCombobox.Item
                  className="flex min-h-9 cursor-pointer items-center gap-2 rounded-sm border-l-2 border-transparent px-3 text-supporting text-text-secondary outline-none transition-[background-color,border-color,color] duration-[var(--ely-duration-fast)] motion-reduce:transition-none data-[highlighted]:border-action data-[highlighted]:bg-surface-2 data-[selected]:border-action data-[selected]:bg-action-soft data-[selected]:text-action-strong data-[disabled]:cursor-not-allowed"
                  key={option.value}
                  value={option.value}
                >
                  <span className="min-w-0 flex-1 truncate">{option.label}</span>
                  <BaseCombobox.ItemIndicator>
                    <Check
                      aria-hidden="true"
                      size={16}
                      weight="bold"
                    />
                  </BaseCombobox.ItemIndicator>
                </BaseCombobox.Item>
              ))}
            </BaseCombobox.List>
            <BaseCombobox.Empty className="px-3 py-2 text-supporting text-text-tertiary">
              No matches.
            </BaseCombobox.Empty>
          </BaseCombobox.Popup>
        </BaseCombobox.Positioner>
      </BaseCombobox.Portal>
    </BaseCombobox.Root>
  )
})

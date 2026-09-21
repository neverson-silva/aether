import { Select as BaseSelect } from '@base-ui/react/select'
import { CaretDown, Check } from '@phosphor-icons/react'
import {
  forwardRef,
  useImperativeHandle,
  useRef,
  useState,
  type ChangeEventHandler,
  type ReactNode,
  type SelectHTMLAttributes,
} from 'react'
import { Field } from '../field/field'

export interface SelectOption {
  value: string
  label: ReactNode
  disabled?: boolean
  group?: string
}

export interface SelectProps
  extends Omit<
    SelectHTMLAttributes<HTMLSelectElement>,
    'size' | 'value' | 'defaultValue' | 'onChange' | 'className'
  > {
  label?: string
  description?: string
  error?: string
  placeholder?: string
  options: SelectOption[]
  value?: string
  defaultValue?: string
  disabled?: boolean
  className?: string
  onValueChange?: (value: string | null) => void
  onChange?: ChangeEventHandler<HTMLSelectElement>
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(
  function Select(
    {
      description,
      disabled,
      error,
      id,
      label,
      name,
      onChange,
      onValueChange,
      options,
      placeholder = 'Select an option',
      required,
      value,
      defaultValue,
      className,
      style,
      ...props
    },
    ref,
  ) {
    const hiddenSelectRef = useRef<HTMLSelectElement | null>(null)
    const [internalValue, setInternalValue] = useState(
      defaultValue ?? options[0]?.value ?? '',
    )
    const selectedValue = value === undefined ? internalValue : value

    useImperativeHandle(ref, () => hiddenSelectRef.current as HTMLSelectElement)

    const setHiddenSelectRef = (element: HTMLSelectElement | null) => {
      hiddenSelectRef.current = element
    }

    const emitChange = (nextValue: string) => {
      setInternalValue(nextValue)
      onValueChange?.(nextValue)
      const element = hiddenSelectRef.current
      if (!element) return
      const valueSetter = Object.getOwnPropertyDescriptor(
        HTMLSelectElement.prototype,
        'value',
      )?.set
      valueSetter?.call(element, nextValue)
      element.dispatchEvent(new Event('change', { bubbles: true }))
    }

    const control = (
      <BaseSelect.Root
        value={selectedValue}
        onValueChange={(nextValue) => emitChange(nextValue ?? '')}
      >
        <BaseSelect.Trigger
          id={id}
          disabled={disabled}
          aria-label={props['aria-label']}
          aria-labelledby={props['aria-labelledby']}
          aria-invalid={Boolean(error) || undefined}
          className={`flex h-10 w-full items-center justify-between rounded-md border border-border bg-surface-control px-3 hover:bg-surface-container-highest/40 text-body-md text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50 ${error ? 'border-status-danger' : ''} ${className ?? ''}`}
          style={style}
        >
          <BaseSelect.Value placeholder={placeholder} />
          <BaseSelect.Icon>
            <CaretDown size={16} aria-hidden="true" />
          </BaseSelect.Icon>
        </BaseSelect.Trigger>
        <BaseSelect.Portal>
          <BaseSelect.Positioner
            side="bottom"
            align="start"
            sideOffset={4}
            alignItemWithTrigger={false}
            positionMethod="fixed"
            collisionAvoidance={{
              side: 'flip',
              align: 'shift',
              fallbackAxisSide: 'none',
            }}
            style={{ zIndex: 2147483002 }}
            className="z-[2147483002] max-w-[calc(100vw-2rem)] outline-none"
          >
            <BaseSelect.Popup
              style={{
                width: 'var(--anchor-width)',
                maxWidth: 'calc(100vw - 2rem)',
              }}
              className="min-w-[var(--anchor-width)] rounded-md border border-border bg-surface-popover p-1 text-foreground shadow-lg"
            >
              <BaseSelect.List
                className="aether-select-list"
                style={{
                  maxHeight: 'min(var(--available-height), 20rem)',
                  overflowY: 'auto',
                  overflowX: 'hidden',
                  overscrollBehavior: 'contain',
                }}
              >
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

    const hiddenSelect = (
      <select
        {...props}
        ref={setHiddenSelectRef}
        id={id}
        name={name}
        required={required}
        disabled={disabled}
        value={selectedValue}
        onChange={onChange ?? (() => undefined)}
        tabIndex={-1}
        aria-hidden="true"
        style={{
          position: 'absolute',
          width: 1,
          height: 1,
          padding: 0,
          margin: -1,
          overflow: 'hidden',
          clip: 'rect(0, 0, 0, 0)',
          whiteSpace: 'nowrap',
          border: 0,
        }}
      >
        {options.map((option) => (
          <option
            key={option.value}
            value={option.value}
            disabled={option.disabled}
          >
            {option.label}
          </option>
        ))}
      </select>
    )

    const content = (
      <div className="relative w-full">
        {control}
        {hiddenSelect}
      </div>
    )

    return label ? (
      <Field
        label={label}
        description={description}
        error={error}
        required={required}
        disabled={disabled}
      >
        {content}
      </Field>
    ) : (
      content
    )
  },
)

import { CaretDown, Check } from '@phosphor-icons/react'
import { Select as BaseSelect } from '@base-ui/react/select'
import {
  Children,
  forwardRef,
  isValidElement,
  useMemo,
  useState,
  type ChangeEvent,
  type FocusEvent,
  type ReactNode,
  type SelectHTMLAttributes,
} from 'react'

interface SelectOption {
  value: string
  label: ReactNode
  disabled?: boolean
}

export interface SelectProps
  extends Omit<
    SelectHTMLAttributes<HTMLSelectElement>,
    'children' | 'defaultValue' | 'onBlur' | 'onChange' | 'value'
  > {
  'data-invalid'?: string
  children?: ReactNode
  defaultValue?: string
  onBlur?: (event: FocusEvent<HTMLSelectElement>) => void
  onChange?: (event: ChangeEvent<HTMLSelectElement>) => void
  value?: string
}

function getOptions(children: ReactNode): SelectOption[] {
  return Children.toArray(children).flatMap((child) => {
    if (
      !isValidElement<{ children?: ReactNode; disabled?: boolean; value?: string }>(
        child,
      ) ||
      child.type !== 'option'
    )
      return []
    return [
      {
        value: child.props.value ?? '',
        label: child.props.children,
        disabled: child.props.disabled,
      },
    ]
  })
}

function normalizeValue(value?: string) {
  return value ? value : null
}

export const Select = forwardRef<HTMLInputElement, SelectProps>(function SelectView(
  {
    'aria-describedby': ariaDescribedby,
    'aria-errormessage': ariaErrorMessage,
    'aria-invalid': ariaInvalid,
    'aria-label': ariaLabel,
    'aria-labelledby': ariaLabelledby,
    'data-invalid': dataInvalid,
    autoComplete,
    children,
    className = '',
    defaultValue,
    disabled,
    form,
    id,
    name,
    onBlur,
    onChange,
    required,
    style,
    tabIndex,
    title,
    value,
  },
  ref,
) {
  const options = useMemo(() => getOptions(children), [children])
  const [internalValue, setInternalValue] = useState(() => normalizeValue(defaultValue))
  const selectedValue = value === undefined ? internalValue : normalizeValue(value)
  const placeholder =
    options.find((option) => option.value === '')?.label ?? 'Choose an option'
  const items = options.filter((option) => option.value !== '')

  const emitChange = (nextValue: string | null) => {
    if (value === undefined) setInternalValue(nextValue)
    const event = {
      target: { name: name ?? id ?? '', value: nextValue ?? '' },
      currentTarget: { name: name ?? id ?? '', value: nextValue ?? '' },
      type: 'change',
    } as unknown as ChangeEvent<HTMLSelectElement>
    onChange?.(event)
    const blurEvent = {
      target: event.target,
      currentTarget: event.currentTarget,
      type: 'blur',
    } as unknown as FocusEvent<HTMLSelectElement>
    onBlur?.(blurEvent)
  }

  return (
    <BaseSelect.Root
      autoComplete={autoComplete}
      defaultValue={value === undefined ? selectedValue : undefined}
      disabled={disabled}
      form={form}
      id={id}
      inputRef={ref}
      items={items}
      name={name}
      onValueChange={(nextValue) => emitChange(nextValue as string | null)}
      required={required}
      value={value === undefined ? undefined : selectedValue}
    >
      <BaseSelect.Trigger
        aria-describedby={ariaDescribedby}
        aria-errormessage={ariaErrorMessage}
        aria-invalid={ariaInvalid}
        aria-label={ariaLabel}
        aria-labelledby={ariaLabelledby}
        className={`flex min-h-10 w-full cursor-pointer items-center gap-3 rounded-lg border border-border-default bg-field px-3 text-left text-body text-text-primary outline-none transition-[background-color,border-color,box-shadow] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:bg-field-hover focus:border-focus focus:ring-2 focus:ring-focus/25 aria-invalid:border-danger aria-invalid:hover:bg-field aria-invalid:focus:border-danger aria-invalid:focus:ring-danger/25 data-[invalid=true]:border-danger data-[invalid=true]:hover:bg-field data-[invalid=true]:focus:border-danger data-[invalid=true]:focus:ring-danger/25 disabled:cursor-not-allowed disabled:opacity-45 ${className}`}
        data-invalid={dataInvalid}
        id={id}
        style={style}
        tabIndex={tabIndex}
        title={title}
      >
        <BaseSelect.Value
          className="min-w-0 flex-1 truncate"
          placeholder={placeholder}
        />
        <BaseSelect.Icon className="ml-auto shrink-0 pl-2 text-text-tertiary">
          <CaretDown
            aria-hidden="true"
            size={16}
            weight="bold"
          />
        </BaseSelect.Icon>
      </BaseSelect.Trigger>
      <BaseSelect.Portal>
        <BaseSelect.Positioner
          alignItemWithTrigger={false}
          collisionAvoidance={{ align: 'shift', fallbackAxisSide: 'none', side: 'shift' }}
          className="z-50"
          side="bottom"
          sideOffset={6}
        >
          <BaseSelect.Popup className="max-h-64 min-w-[var(--anchor-width)] origin-[var(--transform-origin)] overflow-hidden rounded-lg border border-border-default bg-overlay p-1 text-text-primary shadow-elevation-2 transition-[opacity,transform] duration-[var(--ely-duration-popover)] ease-ely-out data-[ending-style]:scale-[.98] data-[ending-style]:opacity-0 data-[starting-style]:scale-[.98] data-[starting-style]:opacity-0 motion-reduce:transition-opacity">
            <BaseSelect.List className="grid max-h-64 content-start overflow-auto">
              {items.map((option) => (
                <BaseSelect.Item
                  className="flex min-h-9 cursor-pointer items-center gap-2 rounded-md px-3 text-supporting text-text-secondary outline-none data-[highlighted]:bg-surface-2 data-[highlighted]:text-text-primary data-[selected]:bg-action-soft data-[selected]:text-text-primary data-[disabled]:cursor-not-allowed data-[disabled]:opacity-45"
                  disabled={option.disabled}
                  key={option.value}
                  value={option.value}
                >
                  <BaseSelect.ItemText className="min-w-0 flex-1 truncate">
                    {option.label}
                  </BaseSelect.ItemText>
                  <BaseSelect.ItemIndicator className="text-action-strong">
                    <Check
                      aria-hidden="true"
                      size={16}
                      weight="bold"
                    />
                  </BaseSelect.ItemIndicator>
                </BaseSelect.Item>
              ))}
            </BaseSelect.List>
          </BaseSelect.Popup>
        </BaseSelect.Positioner>
      </BaseSelect.Portal>
    </BaseSelect.Root>
  )
})

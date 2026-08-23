import { Popover } from '@base-ui/react/popover'
import { CalendarBlank, X } from '@phosphor-icons/react'
import { type InputHTMLAttributes, useState } from 'react'
import { Calendar } from '../calendar/calendar'
import { Field } from '../field/field'

export interface DatePickerProps
  extends Omit<InputHTMLAttributes<HTMLInputElement>, 'type' | 'size'> {
  label?: string
  description?: string
  error?: string
  clearable?: boolean
  onClear?: () => void
  onValueChange?: (value: string) => void
  minDate?: string
  maxDate?: string
  disabledDates?: string[]
}

function displayDate(value?: string) {
  if (!value) return ''
  return new Intl.DateTimeFormat(undefined, {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  }).format(new Date(`${value}T12:00:00`))
}

export function DatePicker({
  className = '',
  defaultValue,
  description,
  disabled,
  disabledDates,
  error,
  id,
  label,
  maxDate,
  minDate,
  name,
  onChange,
  onClear,
  onValueChange,
  required,
  title,
  'aria-label': ariaLabel,
  value,
  clearable,
}: DatePickerProps) {
  const initialValue = String(value ?? defaultValue ?? '')
  const [selectedDate, setSelectedDate] = useState(initialValue)
  const [open, setOpen] = useState(false)
  const currentValue = value === undefined ? selectedDate : String(value)
  const handleValueChange = (nextValue: string) => {
    setSelectedDate(nextValue)
    onValueChange?.(nextValue)
    setOpen(false)
    onChange?.({
      target: { value: nextValue, name },
      currentTarget: { value: nextValue, name },
    } as unknown as React.ChangeEvent<HTMLInputElement>)
  }
  const control = (
    <div className="relative">
      <Popover.Root open={open} onOpenChange={setOpen}>
        <input
          type="hidden"
          id={id}
          name={name}
          value={currentValue}
          readOnly
        />
        <Popover.Trigger
          disabled={disabled}
          aria-label={ariaLabel}
          aria-invalid={Boolean(error) || undefined}
          aria-required={required || undefined}
          title={title}
          className={`flex h-10 w-full items-center justify-between gap-2 rounded-md border bg-surface-control px-3 text-start text-body-md text-foreground outline-none transition-[border-color,box-shadow] focus-visible:border-primary focus-visible:ring-2 focus-visible:ring-ring/20 disabled:cursor-not-allowed disabled:opacity-50 ${error ? 'border-status-danger' : 'border-border'} ${className}`}
        >
          <span
            className={
              currentValue ? 'text-foreground' : 'text-muted-foreground'
            }
          >
            {currentValue ? displayDate(currentValue) : 'Select a date'}
          </span>
          <CalendarBlank
            size={18}
            className="shrink-0 text-muted-foreground"
            aria-hidden="true"
          />
        </Popover.Trigger>
        <Popover.Portal>
          <Popover.Positioner
            className="z-50"
            side="bottom"
            align="start"
            sideOffset={8}
            collisionAvoidance={{
              side: 'none',
              align: 'shift',
              fallbackAxisSide: 'none',
            }}
          >
            <Popover.Popup className="rounded-lg border border-border bg-surface-popover p-1 shadow-lg outline-none data-[starting-style]:scale-95 data-[starting-style]:opacity-0 data-[ending-style]:scale-95 data-[ending-style]:opacity-0 transition-[transform,opacity] duration-200">
              <Calendar
                value={currentValue}
                minDate={minDate}
                maxDate={maxDate}
                disabledDates={disabledDates}
                onValueChange={handleValueChange}
              />
            </Popover.Popup>
          </Popover.Positioner>
        </Popover.Portal>
        {clearable && currentValue ? (
          <button
            type="button"
            className="absolute right-9 top-2.5 text-muted-foreground"
            aria-label="Clear date"
            onClick={() => {
              onClear?.()
              handleValueChange('')
            }}
          >
            <X size={16} aria-hidden="true" />
          </button>
        ) : null}
      </Popover.Root>
    </div>
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

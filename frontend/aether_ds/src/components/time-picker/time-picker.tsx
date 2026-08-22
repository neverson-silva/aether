import { Popover } from '@base-ui/react/popover'
import { Clock, X } from '@phosphor-icons/react'
import { type InputHTMLAttributes, useState } from 'react'
import { Field } from '../field/field'

export interface TimePickerProps
  extends Omit<InputHTMLAttributes<HTMLInputElement>, 'type' | 'size'> {
  label?: string
  description?: string
  error?: string
  withSeconds?: boolean
  timezone?: string
  onValueChange?: (value: string) => void
}

function parseTime(value: string, withSeconds: boolean) {
  const [hours = '00', minutes = '00', seconds = '00'] = value.split(':')
  return { hours, minutes, seconds: withSeconds ? seconds : '00' }
}

function formatTime(
  hours: string,
  minutes: string,
  seconds: string,
  withSeconds: boolean,
) {
  return withSeconds ? `${hours}:${minutes}:${seconds}` : `${hours}:${minutes}`
}

function options(length: number) {
  return Array.from({ length }, (_, index) => String(index).padStart(2, '0'))
}

const hours = options(24)
const minutes = options(60)

export function TimePicker({
  className = '',
  defaultValue,
  description,
  disabled,
  error,
  label,
  name,
  onChange,
  onValueChange,
  required,
  title,
  timezone,
  value,
  withSeconds = false,
  'aria-label': ariaLabel,
}: TimePickerProps) {
  const initialValue = String(value ?? defaultValue ?? '')
  const [selectedTime, setSelectedTime] = useState(initialValue)
  const [open, setOpen] = useState(false)
  const currentValue = value === undefined ? selectedTime : String(value)
  const parsed = parseTime(currentValue, withSeconds)
  const handleValueChange = (nextValue: string) => {
    setSelectedTime(nextValue)
    onValueChange?.(nextValue)
    onChange?.({
      target: { value: nextValue, name },
      currentTarget: { value: nextValue, name },
    } as unknown as React.ChangeEvent<HTMLInputElement>)
  }
  const updatePart = (part: keyof typeof parsed, nextValue: string) => {
    const next = { ...parsed, [part]: nextValue }
    handleValueChange(
      formatTime(next.hours, next.minutes, next.seconds, withSeconds),
    )
  }
  const selectClass =
    'h-10 rounded-md border border-border bg-surface-card px-2 text-body-md text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/20'
  const control = (
    <div className="relative">
      <Popover.Root open={open} onOpenChange={setOpen}>
        <input type="hidden" name={name} value={currentValue} readOnly />
        <Popover.Trigger
          disabled={disabled}
          aria-label={ariaLabel}
          aria-invalid={Boolean(error) || undefined}
          aria-required={required || undefined}
          title={title}
          className={`flex h-10 w-full items-center justify-between gap-2 rounded-md border bg-surface-card px-3 text-start text-body-md text-foreground outline-none transition-[border-color,box-shadow] focus-visible:border-primary focus-visible:ring-2 focus-visible:ring-primary/20 disabled:cursor-not-allowed disabled:opacity-50 ${error ? 'border-status-danger' : 'border-border'} ${className}`}
        >
          <span
            className={
              currentValue
                ? 'font-mono text-foreground'
                : 'text-muted-foreground'
            }
          >
            {currentValue || 'Select a time'}
          </span>
          <Clock
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
            <Popover.Popup className="w-72 rounded-lg border border-border bg-surface-popover p-4 shadow-lg outline-none data-[starting-style]:scale-95 data-[starting-style]:opacity-0 data-[ending-style]:scale-95 data-[ending-style]:opacity-0 transition-[transform,opacity] duration-200">
              <div className="mb-3 text-body-sm font-semibold text-foreground">
                Select time
              </div>
              <div className="grid grid-cols-3 gap-2">
                <select
                  aria-label="Hour"
                  value={parsed.hours}
                  className={selectClass}
                  onChange={(event) => updatePart('hours', event.target.value)}
                >
                  {hours.map((option) => (
                    <option key={option} value={option}>
                      {option}
                    </option>
                  ))}
                </select>
                <select
                  aria-label="Minute"
                  value={parsed.minutes}
                  className={selectClass}
                  onChange={(event) =>
                    updatePart('minutes', event.target.value)
                  }
                >
                  {minutes.map((option) => (
                    <option key={option} value={option}>
                      {option}
                    </option>
                  ))}
                </select>
                {withSeconds ? (
                  <select
                    aria-label="Second"
                    value={parsed.seconds}
                    className={selectClass}
                    onChange={(event) =>
                      updatePart('seconds', event.target.value)
                    }
                  >
                    {minutes.map((option) => (
                      <option key={option} value={option}>
                        {option}
                      </option>
                    ))}
                  </select>
                ) : (
                  <span className="flex h-10 items-center justify-center text-muted-foreground">
                    :
                  </span>
                )}
              </div>
              {timezone ? (
                <div className="mt-3 text-body-sm text-muted-foreground">
                  {timezone}
                </div>
              ) : null}
            </Popover.Popup>
          </Popover.Positioner>
        </Popover.Portal>
        {!currentValue && timezone ? (
          <span className="sr-only">{timezone}</span>
        ) : null}
        {currentValue ? (
          <button
            type="button"
            className="absolute right-9 top-2.5 text-muted-foreground"
            aria-label="Clear time"
            onClick={() => handleValueChange('')}
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

import type { ChangeEvent } from 'react'
import { DatePicker } from '../date-picker/date-picker'
import { Field } from '../field/field'
import { TimePicker } from '../time-picker/time-picker'

export interface DateTimePickerProps {
  id?: string
  name?: string
  label?: string
  description?: string
  error?: string
  date?: string
  time?: string
  defaultDate?: string
  defaultTime?: string
  disabled?: boolean
  required?: boolean
  withSeconds?: boolean
  timezone?: string
  minDate?: string
  maxDate?: string
  onDateChange?: (value: string) => void
  onTimeChange?: (value: string) => void
  onChange?: (value: { date: string; time: string }) => void
}

export function DateTimePicker({
  date,
  defaultDate = '',
  defaultTime = '',
  description,
  disabled,
  error,
  id,
  label,
  maxDate,
  minDate,
  name,
  onChange,
  onDateChange,
  onTimeChange,
  required,
  time,
  timezone,
  withSeconds,
}: DateTimePickerProps) {
  const update = (nextDate: string, nextTime: string) => {
    onChange?.({ date: nextDate, time: nextTime })
  }
  const content = (
    <div className="grid gap-3 sm:grid-cols-[minmax(0,1.35fr)_minmax(0,1fr)]">
      <DatePicker
        id={id}
        name={name ? `${name}-date` : undefined}
        value={date}
        defaultValue={defaultDate}
        minDate={minDate}
        maxDate={maxDate}
        disabled={disabled}
        error={error}
        aria-label="Date"
        onValueChange={(value) => {
          onDateChange?.(value)
          update(value, time ?? defaultTime)
        }}
      />
      <TimePicker
        name={name ? `${name}-time` : undefined}
        value={time}
        defaultValue={defaultTime}
        disabled={disabled}
        withSeconds={withSeconds}
        timezone={timezone}
        aria-label="Time"
        onValueChange={(value) => {
          onTimeChange?.(value)
          update(date ?? defaultDate, value)
        }}
      />
    </div>
  )
  return label ? (
    <Field
      label={label}
      description={description}
      error={error}
      disabled={disabled}
      required={required}
    >
      {content}
    </Field>
  ) : (
    content
  )
}

export type DateTimeChangeEvent = ChangeEvent<HTMLInputElement>

import type { FieldsetHTMLAttributes } from 'react'
import { Calendar, type CalendarProps } from './calendar'

export interface DateRangePickerProps
  extends Omit<FieldsetHTMLAttributes<HTMLFieldSetElement>, 'onChange'> {
  'data-invalid'?: string
  startProps?: CalendarProps
  endProps?: CalendarProps
  className?: string
}
export function DateRangePicker({
  className = '',
  endProps,
  startProps,
  ...props
}: DateRangePickerProps) {
  const invalid = props['aria-invalid']
  const dataInvalid = props['data-invalid']
  return (
    <fieldset
      {...props}
      aria-label={props['aria-label'] ?? 'Date range'}
      className={`grid gap-2 sm:flex sm:items-center ${className}`}
      data-invalid={dataInvalid}
    >
      <Calendar
        aria-label="Start date"
        aria-invalid={startProps?.['aria-invalid'] ?? invalid}
        data-invalid={startProps?.['data-invalid'] ?? dataInvalid}
        {...startProps}
      />
      <span
        aria-hidden="true"
        className="hidden text-text-tertiary sm:inline"
      >
        to
      </span>
      <Calendar
        aria-label="End date"
        aria-invalid={endProps?.['aria-invalid'] ?? invalid}
        data-invalid={endProps?.['data-invalid'] ?? dataInvalid}
        {...endProps}
      />
    </fieldset>
  )
}

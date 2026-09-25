import {
  forwardRef,
  useState,
  type ChangeEvent,
  type FocusEvent,
  type InputHTMLAttributes,
} from 'react'
import { Calendar } from './calendar'
import { TimePicker } from './time-picker'

export interface DateTimePickerProps
  extends Omit<
    InputHTMLAttributes<HTMLInputElement>,
    'defaultValue' | 'onBlur' | 'onChange' | 'type' | 'value'
  > {
  'data-invalid'?: string
  defaultValue?: string
  onBlur?: (event: FocusEvent<HTMLInputElement>) => void
  onChange?: (event: ChangeEvent<HTMLInputElement>) => void
  value?: string
}

function splitDateTime(value?: string) {
  const [date = '', time = ''] = value?.split('T') ?? []
  return { date, time: time.slice(0, 5) }
}

export const DateTimePicker = forwardRef<HTMLInputElement, DateTimePickerProps>(
  function DateTimePickerView(
    {
      'aria-invalid': ariaInvalid,
      'aria-label': ariaLabel,
      'data-invalid': dataInvalid,
      className = '',
      defaultValue,
      disabled,
      form,
      id,
      name,
      onBlur,
      onChange,
      value,
      ...inputProps
    },
    ref,
  ) {
    const initial = splitDateTime(value ?? defaultValue)
    const [datePart, setDatePart] = useState(initial.date)
    const [timePart, setTimePart] = useState(initial.time)
    const isControlled = value !== undefined
    const current = splitDateTime(
      isControlled ? value : `${datePart}${datePart && timePart ? `T${timePart}` : ''}`,
    )
    const nextValue = (nextDate: string, nextTime: string) =>
      nextDate ? `${nextDate}T${nextTime || '00:00'}` : ''

    const emitChange = (nextDate: string, nextTime: string) => {
      if (!isControlled) {
        setDatePart(nextDate)
        setTimePart(nextTime)
      }
      const event = {
        target: { name: name ?? id ?? '', value: nextValue(nextDate, nextTime) },
        currentTarget: { name: name ?? id ?? '', value: nextValue(nextDate, nextTime) },
        type: 'change',
      } as unknown as ChangeEvent<HTMLInputElement>
      onChange?.(event)
      const blurEvent = {
        target: event.target,
        currentTarget: event.currentTarget,
        type: 'blur',
      } as unknown as FocusEvent<HTMLInputElement>
      onBlur?.(blurEvent)
    }

    return (
      <fieldset
        aria-label={ariaLabel ?? 'Date and time'}
        className={`grid gap-2 sm:flex sm:items-center ${className}`}
        data-invalid={dataInvalid}
      >
        <Calendar
          {...inputProps}
          aria-label={ariaLabel ?? 'Date'}
          aria-invalid={ariaInvalid}
          data-invalid={dataInvalid}
          disabled={disabled}
          value={current.date}
          onChange={(event) => emitChange(event.target.value, current.time)}
        />
        <TimePicker
          aria-label={`${ariaLabel ?? 'Date'} time`}
          aria-invalid={ariaInvalid}
          data-invalid={dataInvalid}
          disabled={disabled}
          value={current.time}
          onChange={(event) => emitChange(current.date, event.target.value)}
        />
        <input
          aria-hidden="true"
          className="sr-only"
          form={form}
          id={id ? `${id}-value` : undefined}
          name={name}
          readOnly
          ref={ref}
          type="hidden"
          value={nextValue(current.date, current.time)}
        />
      </fieldset>
    )
  },
)

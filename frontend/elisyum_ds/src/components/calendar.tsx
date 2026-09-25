import { CaretLeft, CaretRight, CalendarBlank } from '@phosphor-icons/react'
import { Popover as BasePopover } from '@base-ui/react/popover'
import {
  forwardRef,
  useId,
  useMemo,
  useState,
  type ChangeEvent,
  type FocusEvent,
  type InputHTMLAttributes,
  type KeyboardEvent,
} from 'react'

export interface CalendarProps
  extends Omit<
    InputHTMLAttributes<HTMLInputElement>,
    'defaultValue' | 'onBlur' | 'onChange' | 'type' | 'value'
  > {
  'data-invalid'?: string
  defaultValue?: string
  onBlur?: (event: FocusEvent<HTMLInputElement>) => void
  onChange?: (event: ChangeEvent<HTMLInputElement>) => void
  placeholder?: string
  value?: string
}

function parseDateValue(value?: string) {
  if (!value) return null
  const [year, month, day] = value.split('-').map(Number)
  if (!year || !month || !day) return null
  const date = new Date(year, month - 1, day)
  return date.getFullYear() === year &&
    date.getMonth() === month - 1 &&
    date.getDate() === day
    ? date
    : null
}

function formatDateValue(date: Date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

function formatDateLabel(value: string) {
  const date = parseDateValue(value)
  return date
    ? new Intl.DateTimeFormat('en-US', {
        day: 'numeric',
        month: 'short',
        year: 'numeric',
      }).format(date)
    : ''
}

function daysForMonth(date: Date) {
  const firstDay = new Date(date.getFullYear(), date.getMonth(), 1).getDay()
  const daysInMonth = new Date(date.getFullYear(), date.getMonth() + 1, 0).getDate()
  return Array.from({ length: firstDay + daysInMonth }, (_, index) =>
    index < firstDay
      ? null
      : new Date(date.getFullYear(), date.getMonth(), index - firstDay + 1),
  )
}

function isBefore(value: string, boundary?: string) {
  return Boolean(boundary && value < boundary)
}

function isAfter(value: string, boundary?: string) {
  return Boolean(boundary && value > boundary)
}

function moveDate(date: Date, amount: number) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate() + amount)
}

export const Calendar = forwardRef<HTMLInputElement, CalendarProps>(function CalendarView(
  {
    'aria-describedby': ariaDescribedby,
    'aria-errormessage': ariaErrorMessage,
    'aria-invalid': ariaInvalid,
    'aria-label': ariaLabel,
    'aria-labelledby': ariaLabelledby,
    'data-invalid': dataInvalid,
    autoComplete,
    className = '',
    defaultValue,
    disabled,
    form,
    id,
    max,
    min,
    name,
    onBlur,
    onChange,
    placeholder = 'Select a date',
    readOnly,
    required,
    style,
    tabIndex,
    title,
    value,
  },
  ref,
) {
  const [internalValue, setInternalValue] = useState(defaultValue ?? '')
  const selectedValue = value === undefined ? internalValue : value
  const selectedDate = parseDateValue(selectedValue)
  const calendarId = useId()
  const [open, setOpen] = useState(false)
  const [viewDate, setViewDate] = useState(() => selectedDate ?? new Date())
  const monthDays = useMemo(() => daysForMonth(viewDate), [viewDate])
  const monthLabel = new Intl.DateTimeFormat('en-US', {
    month: 'long',
    year: 'numeric',
  }).format(viewDate)
  const todayValue = formatDateValue(new Date())
  const minValue = typeof min === 'string' ? min : undefined
  const maxValue = typeof max === 'string' ? max : undefined

  const emitChange = (nextValue: string) => {
    if (value === undefined) setInternalValue(nextValue)
    const event = {
      target: { name: name ?? id ?? '', value: nextValue },
      currentTarget: { name: name ?? id ?? '', value: nextValue },
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
    <BasePopover.Root
      open={open}
      onOpenChange={(nextOpen) => setOpen(nextOpen)}
    >
      <BasePopover.Trigger
        aria-describedby={ariaDescribedby}
        aria-errormessage={ariaErrorMessage}
        aria-expanded={open}
        aria-invalid={ariaInvalid}
        aria-label={ariaLabel}
        aria-labelledby={ariaLabelledby}
        aria-readonly={readOnly || undefined}
        aria-required={required || undefined}
        className={`flex min-h-10 w-full cursor-pointer items-center gap-3 rounded-md border border-border-default bg-field px-3 text-left text-body text-text-primary outline-none transition-[background-color,border-color,box-shadow] duration-[var(--ely-duration-fast)] hover:bg-field-hover focus:border-focus focus:ring-2 focus:ring-focus/25 aria-invalid:border-danger aria-invalid:hover:bg-field aria-invalid:focus:border-danger aria-invalid:focus:ring-danger/25 data-[invalid=true]:border-danger data-[invalid=true]:hover:bg-field data-[invalid=true]:focus:border-danger data-[invalid=true]:focus:ring-danger/25 disabled:cursor-not-allowed disabled:opacity-45 ${className}`}
        data-invalid={dataInvalid}
        disabled={disabled || readOnly}
        id={id}
        style={style}
        tabIndex={tabIndex}
        title={title}
      >
        <span
          className={`min-w-0 flex-1 truncate ${selectedValue ? 'text-text-primary' : 'text-text-tertiary'}`}
        >
          {selectedValue ? formatDateLabel(selectedValue) : placeholder}
        </span>
        <CalendarBlank
          aria-hidden="true"
          className="ml-auto shrink-0 text-text-tertiary"
          size={17}
        />
      </BasePopover.Trigger>
      <input
        aria-hidden="true"
        autoComplete={autoComplete}
        className="sr-only"
        form={form}
        id={id ? `${id}-value` : undefined}
        name={name}
        readOnly
        ref={ref}
        required={required}
        tabIndex={-1}
        type="hidden"
        value={selectedValue}
      />
      <BasePopover.Portal>
        <BasePopover.Positioner
          className="z-30"
          sideOffset={6}
        >
          <BasePopover.Popup
            className="w-72 origin-[var(--transform-origin)] rounded-md border border-border-default bg-overlay p-3 text-text-primary shadow-elevation-2 transition-[opacity,transform] duration-[var(--ely-duration-popover)] ease-ely-out data-[ending-style]:scale-[.98] data-[ending-style]:opacity-0 data-[starting-style]:scale-[.98] data-[starting-style]:opacity-0 motion-reduce:transition-opacity"
            role="dialog"
            aria-label="Choose date"
          >
            <div className="flex items-center justify-between gap-2">
              <button
                aria-label="Previous month"
                className="inline-flex size-8 cursor-pointer items-center justify-center rounded-md text-text-secondary transition-[background-color,color,transform] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:bg-surface-2 hover:text-text-primary focus-visible:outline-2 focus-visible:outline-focus motion-safe:active:scale-[var(--ely-motion-press-scale)] motion-reduce:active:scale-100 disabled:cursor-not-allowed"
                disabled={disabled}
                onClick={() =>
                  setViewDate(
                    (current) =>
                      new Date(current.getFullYear(), current.getMonth() - 1, 1),
                  )
                }
                type="button"
              >
                <CaretLeft
                  aria-hidden="true"
                  size={16}
                />
              </button>
              <p
                aria-live="polite"
                className="text-label text-text-primary"
              >
                {monthLabel}
              </p>
              <button
                aria-label="Next month"
                className="inline-flex size-8 cursor-pointer items-center justify-center rounded-md text-text-secondary transition-[background-color,color,transform] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:bg-surface-2 hover:text-text-primary focus-visible:outline-2 focus-visible:outline-focus motion-safe:active:scale-[var(--ely-motion-press-scale)] motion-reduce:active:scale-100 disabled:cursor-not-allowed"
                disabled={disabled}
                onClick={() =>
                  setViewDate(
                    (current) =>
                      new Date(current.getFullYear(), current.getMonth() + 1, 1),
                  )
                }
                type="button"
              >
                <CaretRight
                  aria-hidden="true"
                  size={16}
                />
              </button>
            </div>
            <div className="mt-3 grid grid-cols-7 gap-1 text-center text-supporting text-text-tertiary">
              {['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa'].map((day) => (
                <span
                  aria-hidden="true"
                  key={day}
                >
                  {day}
                </span>
              ))}
            </div>
            <div
              className="mt-1 grid grid-cols-7 gap-1"
              role="grid"
              aria-label={monthLabel}
              aria-rowcount={6}
            >
              {monthDays.map((date, index) => {
                if (!date)
                  return (
                    <span
                      aria-hidden="true"
                      className="size-8"
                      key={`empty-${index}`}
                    />
                  )
                const dateValue = formatDateValue(date)
                const isSelected = dateValue === selectedValue
                const isToday = dateValue === todayValue
                const unavailable =
                  isBefore(dateValue, minValue) || isAfter(dateValue, maxValue)
                return (
                  <div
                    className="grid size-8 place-items-center"
                    key={dateValue}
                    role="gridcell"
                    aria-selected={isSelected}
                  >
                    <button
                      aria-current={isToday ? 'date' : undefined}
                      aria-label={formatDateLabel(dateValue)}
                      className={`inline-flex size-8 cursor-pointer items-center justify-center rounded-md text-supporting transition-[background-color,box-shadow,transform] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:bg-surface-2 focus-visible:outline-2 focus-visible:outline-focus motion-safe:active:scale-[var(--ely-motion-press-scale)] motion-reduce:active:scale-100 ${isSelected ? 'bg-action text-on-action hover:bg-action-strong' : isToday ? 'ring-1 ring-action text-action-strong' : 'text-text-secondary'} disabled:cursor-not-allowed disabled:opacity-35`}
                      disabled={disabled || unavailable}
                      id={`${calendarId}-${dateValue}`}
                      onClick={() => {
                        emitChange(dateValue)
                        setViewDate(date)
                        setOpen(false)
                      }}
                      onKeyDown={(event: KeyboardEvent<HTMLButtonElement>) => {
                        const offsets: Record<string, number> = {
                          ArrowDown: 7,
                          ArrowLeft: -1,
                          ArrowRight: 1,
                          ArrowUp: -7,
                        }
                        const offset = offsets[event.key]
                        if (offset === undefined) return
                        event.preventDefault()
                        const nextDate = moveDate(date, offset)
                        const nextValue = formatDateValue(nextDate)
                        setViewDate(nextDate)
                        requestAnimationFrame(() =>
                          document
                            .getElementById(`${calendarId}-${nextValue}`)
                            ?.focus(),
                        )
                      }}
                      type="button"
                    >
                      {date.getDate()}
                    </button>
                  </div>
                )
              })}
            </div>
            <div className="mt-3 flex items-center justify-between gap-3 border-t border-border-subtle pt-3">
              <span className="font-technical text-log text-text-tertiary">
                {selectedValue ? formatDateLabel(selectedValue) : 'No date selected'}
              </span>
              <button
                className="cursor-pointer text-label text-action-strong transition-[color,transform] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:text-action focus-visible:outline-2 focus-visible:outline-focus motion-safe:active:scale-[var(--ely-motion-press-scale)] motion-reduce:active:scale-100"
                onClick={() => {
                  setViewDate(new Date())
                  emitChange(todayValue)
                  setOpen(false)
                }}
                type="button"
              >
                Today
              </button>
            </div>
          </BasePopover.Popup>
        </BasePopover.Positioner>
      </BasePopover.Portal>
    </BasePopover.Root>
  )
})

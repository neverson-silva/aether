import { CaretLeft, CaretRight } from '@phosphor-icons/react'
import { type HTMLAttributes, useEffect, useState } from 'react'

export interface CalendarProps extends HTMLAttributes<HTMLDivElement> {
  value?: string
  onValueChange?: (value: string) => void
  minDate?: string
  maxDate?: string
  disabledDates?: string[]
  multipleMonths?: number
}

const weekDays = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']

function parseDate(value?: string) {
  if (value) return new Date(`${value}T12:00:00`)
  const today = new Date()
  return new Date(today.getFullYear(), today.getMonth(), today.getDate(), 12)
}

function formatDate(date: Date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

export function Calendar({
  className = '',
  disabledDates,
  maxDate,
  minDate,
  onValueChange,
  value,
  ...props
}: CalendarProps) {
  const [viewDate, setViewDate] = useState(() => parseDate(value))
  const year = viewDate.getFullYear()
  const month = viewDate.getMonth()
  const daysInMonth = new Date(year, month + 1, 0).getDate()
  const leadingDays = (new Date(year, month, 1).getDay() + 6) % 7
  const days = Array.from({ length: leadingDays + daysInMonth }, (_, index) =>
    index < leadingDays ? null : index - leadingDays + 1,
  )
  const monthLabel = new Intl.DateTimeFormat(undefined, {
    month: 'long',
    year: 'numeric',
  }).format(viewDate)

  useEffect(() => {
    if (value) setViewDate(parseDate(value))
  }, [value])

  const isUnavailable = (date: string) =>
    Boolean(
      (minDate && date < minDate) ||
        (maxDate && date > maxDate) ||
        disabledDates?.includes(date),
    )

  return (
    <div
      className={`w-80 rounded-lg border border-border bg-surface-card p-4 shadow-md ${className}`}
      {...props}
    >
      <div className="mb-4 flex items-center justify-between">
        <button
          type="button"
          className="inline-flex size-8 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-surface-container hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
          aria-label="Previous month"
          onClick={() => setViewDate(new Date(year, month - 1, 1, 12))}
        >
          <CaretLeft size={18} aria-hidden="true" />
        </button>
        <h2 className="text-body-md font-semibold capitalize text-foreground">
          {monthLabel}
        </h2>
        <button
          type="button"
          className="inline-flex size-8 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-surface-container hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
          aria-label="Next month"
          onClick={() => setViewDate(new Date(year, month + 1, 1, 12))}
        >
          <CaretRight size={18} aria-hidden="true" />
        </button>
      </div>
      <div className="mb-2 grid grid-cols-7 text-center text-label-caps text-muted-foreground">
        {weekDays.map((day) => (
          <span key={day}>{day}</span>
        ))}
      </div>
      <div className="grid grid-cols-7 gap-1">
        {days.map((day, index) => {
          if (!day) return <span key={`empty-${index}`} aria-hidden="true" />
          const date = formatDate(new Date(year, month, day, 12))
          const selected = date === value
          const unavailable = isUnavailable(date)
          return (
            <button
              key={date}
              type="button"
              aria-label={date}
              aria-pressed={selected}
              disabled={unavailable}
              className={`inline-flex aspect-square items-center justify-center rounded-md text-body-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary ${selected ? 'bg-primary text-primary-foreground' : 'text-foreground hover:bg-surface-container'} ${unavailable ? 'cursor-not-allowed opacity-40' : ''}`}
              onClick={() => onValueChange?.(date)}
            >
              {day}
            </button>
          )
        })}
      </div>
    </div>
  )
}

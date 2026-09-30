import { Popover as BasePopover } from '@base-ui/react/popover'
import { CalendarBlank, CaretDown, CaretLeft, CaretRight } from '@phosphor-icons/react'
import { useMemo, useState, type ButtonHTMLAttributes } from 'react'

export interface DateRangeValue {
  start: string
  end: string
}

export interface DateRangePickerProps
  extends Omit<
    ButtonHTMLAttributes<HTMLButtonElement>,
    'defaultValue' | 'onChange' | 'value'
  > {
  'data-invalid'?: string
  defaultValue?: DateRangeValue
  locale?: string
  onChange?: (value: DateRangeValue) => void
  placeholder?: string
  value?: DateRangeValue
}

type PresetId = '7-days' | '30-days' | 'this-month' | 'previous-month' | 'custom'

const presets: Array<{ id: PresetId; label: string }> = [
  { id: '7-days', label: 'Últimos 7 dias' },
  { id: '30-days', label: 'Últimos 30 dias' },
  { id: 'this-month', label: 'Este mês' },
  { id: 'previous-month', label: 'Mês anterior' },
  { id: 'custom', label: 'Personalizado' },
]

function parseDate(value?: string) {
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

function toValue(date: Date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

function addDays(date: Date, amount: number) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate() + amount)
}

function addMonths(date: Date, amount: number) {
  return new Date(date.getFullYear(), date.getMonth() + amount, 1)
}

function monthDays(date: Date) {
  const firstDay = new Date(date.getFullYear(), date.getMonth(), 1).getDay()
  const days = new Date(date.getFullYear(), date.getMonth() + 1, 0).getDate()
  const previousDays = new Date(date.getFullYear(), date.getMonth(), 0).getDate()
  return Array.from({ length: 42 }, (_, index) => {
    const day = index - firstDay + 1
    if (day < 1)
      return new Date(date.getFullYear(), date.getMonth() - 1, previousDays + day)
    if (day > days) return new Date(date.getFullYear(), date.getMonth() + 1, day - days)
    return new Date(date.getFullYear(), date.getMonth(), day)
  })
}

function formatRange(value: DateRangeValue, locale: string) {
  const start = parseDate(value.start)
  const end = parseDate(value.end)
  if (!start) return ''
  const formatter = new Intl.DateTimeFormat(locale, {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  })
  return end
    ? `${formatter.format(start)} → ${formatter.format(end)}`
    : formatter.format(start)
}

function presetRange(id: PresetId, today: Date): DateRangeValue | null {
  const current = new Date(today.getFullYear(), today.getMonth(), today.getDate())
  if (id === '7-days')
    return { start: toValue(addDays(current, -6)), end: toValue(current) }
  if (id === '30-days')
    return { start: toValue(addDays(current, -29)), end: toValue(current) }
  if (id === 'this-month')
    return {
      start: toValue(new Date(current.getFullYear(), current.getMonth(), 1)),
      end: toValue(current),
    }
  if (id === 'previous-month') {
    const start = new Date(current.getFullYear(), current.getMonth() - 1, 1)
    return {
      start: toValue(start),
      end: toValue(new Date(current.getFullYear(), current.getMonth(), 0)),
    }
  }
  return null
}

function getPreset(value: DateRangeValue, today: Date): PresetId {
  for (const preset of presets) {
    const range = presetRange(preset.id, today)
    if (range && range.start === value.start && range.end === value.end)
      return preset.id
  }
  return 'custom'
}

function sameDay(date: Date, value?: string) {
  return toValue(date) === value
}

export function DateRangePicker({
  'data-invalid': dataInvalid,
  'aria-label': ariaLabel,
  className = '',
  defaultValue = { start: '', end: '' },
  disabled,
  locale = 'pt-BR',
  onChange,
  onClick,
  placeholder = 'Selecionar período',
  value,
  ...props
}: DateRangePickerProps) {
  const [internalValue, setInternalValue] = useState(defaultValue)
  const appliedValue = value ?? internalValue
  const today = useMemo(() => new Date(), [])
  const [open, setOpen] = useState(false)
  const [draftValue, setDraftValue] = useState(appliedValue)
  const [viewDate, setViewDate] = useState(() => parseDate(appliedValue.start) ?? today)
  const [preset, setPreset] = useState<PresetId>(() => getPreset(appliedValue, today))
  const rightMonth = addMonths(viewDate, 1)

  const handleOpenChange = (nextOpen: boolean) => {
    if (nextOpen) {
      setDraftValue(appliedValue)
      setPreset(getPreset(appliedValue, today))
      setViewDate(parseDate(appliedValue.start) ?? new Date())
    }
    setOpen(nextOpen)
  }

  const selectDate = (date: Date) => {
    const selected = toValue(date)
    if (!draftValue.start || draftValue.end) {
      setDraftValue({ start: selected, end: '' })
      setPreset('custom')
      return
    }
    setDraftValue(
      selected < draftValue.start
        ? { start: selected, end: draftValue.start }
        : { start: draftValue.start, end: selected },
    )
    setPreset('custom')
  }

  const apply = () => {
    const nextValue =
      draftValue.start && !draftValue.end
        ? { start: draftValue.start, end: draftValue.start }
        : draftValue
    if (value === undefined) setInternalValue(nextValue)
    onChange?.(nextValue)
    setOpen(false)
  }

  const clear = () => {
    setDraftValue({ start: '', end: '' })
    setPreset('custom')
  }

  const renderMonth = (month: Date) => {
    const days = monthDays(month)
    const monthName = new Intl.DateTimeFormat(locale, {
      month: 'long',
      year: 'numeric',
    }).format(month)
    const weekDays = Array.from({ length: 7 }, (_, index) =>
      new Intl.DateTimeFormat(locale, { weekday: 'short' })
        .format(new Date(2024, 0, 7 + index))
        .slice(0, 1)
        .toUpperCase(),
    )
    return (
      <div className="min-w-0 p-4">
        <p
          aria-live="polite"
          className="mb-4 text-center text-label text-text-primary capitalize"
        >
          {monthName}
        </p>
        <div
          className="grid grid-cols-7 gap-y-1"
          role="grid"
          aria-label={monthName}
        >
          {weekDays.map((day, index) => (
            <span
              className="pb-2 text-center text-caption text-text-tertiary"
              key={`${day}-${index}`}
            >
              {day}
            </span>
          ))}
          {days.map((date) => {
            const dateValue = toValue(date)
            const inMonth = date.getMonth() === month.getMonth()
            const isStart = dateValue === draftValue.start
            const isEnd = dateValue === draftValue.end
            const inRange = Boolean(
              draftValue.start &&
                draftValue.end &&
                dateValue > draftValue.start &&
                dateValue < draftValue.end,
            )
            const selected = isStart || isEnd
            return (
              <button
                aria-label={new Intl.DateTimeFormat(locale, {
                  dateStyle: 'full',
                }).format(date)}
                aria-selected={selected}
                className={`relative flex h-9 items-center justify-center text-label transition-colors focus-visible:z-10 focus-visible:outline-2 focus-visible:outline-focus ${inRange ? 'bg-action-soft text-text-primary' : ''} ${isStart ? 'rounded-l-md bg-action text-on-action' : ''} ${isEnd ? 'rounded-r-md bg-action text-on-action' : ''} ${selected && isStart && isEnd ? 'rounded-md' : ''} ${!selected && !inRange ? 'rounded-md text-text-primary hover:bg-surface-2' : ''} ${!inMonth ? 'text-text-tertiary opacity-60' : ''} ${sameDay(date, toValue(today)) && !selected ? 'font-semibold underline decoration-action decoration-2 underline-offset-4' : ''}`}
                key={dateValue}
                onClick={() => selectDate(date)}
                type="button"
              >
                {date.getDate()}
              </button>
            )
          })}
        </div>
      </div>
    )
  }

  return (
    <BasePopover.Root
      open={open}
      onOpenChange={handleOpenChange}
    >
      <BasePopover.Trigger
        {...props}
        {...(onClick ? { onClick } : {})}
        aria-expanded={open}
        aria-invalid={props['aria-invalid']}
        aria-label={ariaLabel ?? 'Date range'}
        className={`flex min-h-10 w-full cursor-pointer items-center gap-3 rounded-md border border-border-default bg-field px-3 text-left text-body text-text-primary outline-none transition-[background-color,border-color,box-shadow] duration-[var(--ely-duration-fast)] hover:bg-field-hover focus:border-focus focus:ring-2 focus:ring-focus/25 aria-invalid:border-danger data-[invalid=true]:border-danger disabled:cursor-not-allowed disabled:opacity-45 ${className}`}
        data-invalid={dataInvalid}
        disabled={disabled}
        type="button"
      >
        <CalendarBlank
          aria-hidden="true"
          className="shrink-0 text-text-tertiary"
          size={17}
        />
        <span
          className={`min-w-0 flex-1 truncate ${appliedValue.start ? 'text-text-primary' : 'text-text-tertiary'}`}
        >
          {appliedValue.start ? formatRange(appliedValue, locale) : placeholder}
        </span>
        <CaretDown
          aria-hidden="true"
          className="shrink-0 text-text-tertiary"
          size={17}
        />
      </BasePopover.Trigger>
      <BasePopover.Portal>
        <BasePopover.Positioner
          className="z-30"
          sideOffset={6}
        >
          <BasePopover.Popup
            className="w-[min(850px,calc(100vw-24px))] overflow-hidden rounded-lg border border-border-default bg-overlay text-text-primary shadow-elevation-2"
            role="dialog"
            aria-label="Choose date range"
          >
            <div className="grid md:grid-cols-[180px_1fr]">
              <aside className="border-b border-border-subtle p-3 md:border-b-0 md:border-r">
                <p className="px-3 pb-2 text-caption text-text-tertiary">Período</p>
                <div className="grid gap-1">
                  {presets.map((item) => (
                    <button
                      className={`rounded-md px-3 py-2 text-left text-label transition-colors hover:bg-surface-2 ${preset === item.id ? 'bg-action-soft text-action-strong' : 'text-text-secondary'}`}
                      key={item.id}
                      onClick={() => {
                        setPreset(item.id)
                        const range = presetRange(item.id, today)
                        if (range) {
                          setDraftValue(range)
                          setViewDate(parseDate(range.start) ?? viewDate)
                        }
                      }}
                      type="button"
                    >
                      {item.label}
                    </button>
                  ))}
                </div>
              </aside>
              <div className="min-w-0">
                <div className="flex items-center justify-between border-b border-border-subtle px-4 py-3">
                  <button
                    aria-label="Previous month"
                    className="inline-flex size-8 items-center justify-center rounded-md text-text-secondary hover:bg-surface-2 focus-visible:outline-2 focus-visible:outline-focus"
                    onClick={() => setViewDate((current) => addMonths(current, -1))}
                    type="button"
                  >
                    <CaretLeft size={17} />
                  </button>
                  <span className="text-label text-text-tertiary">
                    {formatRange(draftValue, locale)}
                  </span>
                  <button
                    aria-label="Next month"
                    className="inline-flex size-8 items-center justify-center rounded-md text-text-secondary hover:bg-surface-2 focus-visible:outline-2 focus-visible:outline-focus"
                    onClick={() => setViewDate((current) => addMonths(current, 1))}
                    type="button"
                  >
                    <CaretRight size={17} />
                  </button>
                </div>
                <div className="grid divide-y divide-border-subtle sm:grid-cols-2 sm:divide-x sm:divide-y-0">
                  {renderMonth(viewDate)}
                  {renderMonth(rightMonth)}
                </div>
              </div>
            </div>
            <div className="flex items-center justify-between border-t border-border-subtle px-4 py-3">
              <button
                className="rounded-md px-3 py-2 text-label text-text-secondary hover:bg-surface-2 hover:text-text-primary focus-visible:outline-2 focus-visible:outline-focus"
                onClick={clear}
                type="button"
              >
                Limpar
              </button>
              <div className="flex items-center gap-2">
                <button
                  className="rounded-md px-3 py-2 text-label text-text-secondary hover:bg-surface-2 hover:text-text-primary focus-visible:outline-2 focus-visible:outline-focus"
                  onClick={() => setOpen(false)}
                  type="button"
                >
                  Cancelar
                </button>
                <button
                  className="rounded-md bg-action px-4 py-2 text-label text-on-action hover:bg-action-hover focus-visible:outline-2 focus-visible:outline-focus"
                  onClick={apply}
                  type="button"
                >
                  Aplicar
                </button>
              </div>
            </div>
          </BasePopover.Popup>
        </BasePopover.Positioner>
      </BasePopover.Portal>
    </BasePopover.Root>
  )
}

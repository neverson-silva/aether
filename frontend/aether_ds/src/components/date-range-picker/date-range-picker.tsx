import { DatePicker } from '../date-picker/date-picker'
import { Field } from '../field/field'
export interface DateRangePickerProps {
  label?: string
  description?: string
  error?: string
  start?: string
  end?: string
  minDate?: string
  maxDate?: string
  onStartChange?: (value: string) => void
  onEndChange?: (value: string) => void
  presets?: { label: string; start: string; end: string }[]
}
export function DateRangePicker({
  description,
  end,
  error,
  label,
  maxDate,
  minDate,
  onEndChange,
  onStartChange,
  presets,
  start,
}: DateRangePickerProps) {
  const control = (
    <div className="space-y-2">
      <div className="grid grid-cols-2 gap-2">
        <DatePicker
          value={start ?? ''}
          minDate={minDate}
          maxDate={maxDate}
          onValueChange={onStartChange}
          aria-label="Start date"
        />
        <DatePicker
          value={end ?? ''}
          minDate={start || minDate}
          maxDate={maxDate}
          onValueChange={onEndChange}
          aria-label="End date"
        />
      </div>
      {presets?.length ? (
        <div className="flex flex-wrap gap-2">
          {presets.map((preset) => (
            <button
              type="button"
              key={preset.label}
              className="rounded-md border border-border px-2 py-1 text-body-sm text-muted-foreground hover:bg-surface-container"
              onClick={() => {
                onStartChange?.(preset.start)
                onEndChange?.(preset.end)
              }}
            >
              {preset.label}
            </button>
          ))}
        </div>
      ) : null}
    </div>
  )
  return label ? (
    <Field label={label} description={description} error={error}>
      {control}
    </Field>
  ) : (
    control
  )
}

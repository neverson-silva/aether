import { Combobox as BaseCombobox } from '@base-ui/react/combobox'
import { MagnifyingGlass, SpinnerGap } from '@phosphor-icons/react'
import { useEffect, useRef, useState } from 'react'
import { Field } from '../field/field'

export interface AsyncSearchOption {
  value: string
  label: string
  description?: string
  disabled?: boolean
}

export interface AsyncSearchInputProps {
  label?: string
  description?: string
  error?: string
  placeholder?: string
  value?: string | null
  defaultValue?: string | null
  initialOptions?: AsyncSearchOption[]
  loadOptions: (query: string) => Promise<AsyncSearchOption[]>
  onValueChange?: (value: string | null, option?: AsyncSearchOption) => void
  minQueryLength?: number
  debounceMs?: number
  noResults?: string
  loadingLabel?: string
  disabled?: boolean
}

const emptyOptions: AsyncSearchOption[] = []

export function AsyncSearchInput({
  debounceMs = 250,
  defaultValue,
  description,
  disabled,
  error,
  initialOptions = emptyOptions,
  label,
  loadOptions,
  loadingLabel = 'Searching...',
  minQueryLength = 2,
  noResults = 'No results found.',
  onValueChange,
  placeholder = 'Search resources',
  value,
}: AsyncSearchInputProps) {
  const [options, setOptions] = useState(initialOptions)
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(false)
  const [requestError, setRequestError] = useState<string>()
  const requestId = useRef(0)

  useEffect(() => {
    const normalizedQuery = query.trim()
    if (normalizedQuery.length < minQueryLength) {
      setOptions(initialOptions)
      setLoading(false)
      setRequestError(undefined)
      return
    }
    const currentRequest = ++requestId.current
    const timer = window.setTimeout(async () => {
      setLoading(true)
      setRequestError(undefined)
      try {
        const nextOptions = await loadOptions(normalizedQuery)
        if (currentRequest === requestId.current) setOptions(nextOptions)
      } catch {
        if (currentRequest === requestId.current) {
          setOptions([])
          setRequestError('Unable to load results.')
        }
      } finally {
        if (currentRequest === requestId.current) setLoading(false)
      }
    }, debounceMs)
    return () => window.clearTimeout(timer)
  }, [debounceMs, initialOptions, loadOptions, minQueryLength, query])

  const control = (
    <BaseCombobox.Root
      value={value}
      defaultValue={defaultValue}
      onInputValueChange={(nextQuery) => setQuery(nextQuery)}
      onValueChange={(nextValue) => {
        const selected = options.find((option) => option.value === nextValue)
        onValueChange?.(nextValue, selected)
      }}
    >
      <div className="relative">
        <MagnifyingGlass
          size={18}
          className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"
          aria-hidden="true"
        />
        <BaseCombobox.Input
          disabled={disabled}
          placeholder={placeholder}
          className={`h-10 w-full rounded-md border bg-surface-card pl-9 pr-10 text-body-md text-foreground outline-none transition-[border-color,box-shadow] focus:border-primary focus:ring-2 focus:ring-primary/20 disabled:cursor-not-allowed disabled:opacity-50 ${error || requestError ? 'border-status-danger' : 'border-border'}`}
          aria-invalid={Boolean(error || requestError) || undefined}
        />
        {loading ? (
          <SpinnerGap
            size={18}
            className="absolute right-3 top-1/2 -translate-y-1/2 animate-spin text-primary"
            aria-label={loadingLabel}
          />
        ) : null}
      </div>
      <BaseCombobox.Portal>
        <BaseCombobox.Positioner
          className="z-[110] max-w-[calc(100vw-2rem)] outline-none"
          side="bottom"
          align="start"
          sideOffset={6}
          collisionAvoidance={{ side: 'shift', align: 'shift' }}
        >
          <BaseCombobox.Popup className="w-[var(--anchor-width)] max-w-[calc(100vw-2rem)] max-h-[min(var(--available-height),20rem)] overflow-y-auto rounded-lg border border-border bg-surface-popover p-1 shadow-lg outline-none data-[starting-style]:translate-y-1 data-[starting-style]:opacity-0 data-[ending-style]:translate-y-1 data-[ending-style]:opacity-0 transition-[transform,opacity] duration-200">
            {requestError ? (
              <div className="px-3 py-3 text-body-sm text-status-danger">
                {requestError}
              </div>
            ) : query.trim().length < minQueryLength ? (
              <div className="px-3 py-3 text-body-sm text-muted-foreground">
                Type at least {minQueryLength} characters.
              </div>
            ) : loading ? (
              <div className="px-3 py-3 text-body-sm text-muted-foreground">
                {loadingLabel}
              </div>
            ) : options.length ? (
              <BaseCombobox.List>
                {options.map((option) => (
                  <BaseCombobox.Item
                    key={option.value}
                    value={option.value}
                    disabled={option.disabled}
                    className="flex cursor-pointer items-start gap-2 rounded-md px-3 py-2 text-body-sm outline-none transition-colors data-[highlighted]:bg-surface-container data-[selected]:text-primary data-[disabled]:cursor-not-allowed data-[disabled]:opacity-50"
                  >
                    <span className="min-w-0 flex-1">
                      <span className="block truncate">{option.label}</span>
                      {option.description ? (
                        <span className="block truncate text-body-sm text-muted-foreground">
                          {option.description}
                        </span>
                      ) : null}
                    </span>
                  </BaseCombobox.Item>
                ))}
              </BaseCombobox.List>
            ) : (
              <div className="px-3 py-3 text-body-sm text-muted-foreground">
                {noResults}
              </div>
            )}
          </BaseCombobox.Popup>
        </BaseCombobox.Positioner>
      </BaseCombobox.Portal>
    </BaseCombobox.Root>
  )

  return label ? (
    <Field
      label={label}
      description={description}
      error={error || requestError}
      disabled={disabled}
    >
      {control}
    </Field>
  ) : (
    control
  )
}

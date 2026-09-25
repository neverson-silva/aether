import { Button, IconButton, Input } from '@aether/elisyum-ds'
import { CaretDown, Check, MagnifyingGlass } from '@phosphor-icons/react'
import {
  type Dispatch,
  type InputHTMLAttributes,
  type KeyboardEvent,
  type ReactNode,
  type SetStateAction,
  useId,
  useState,
} from 'react'

export interface SourceComboboxOption {
  value: string
  label: string
  description?: string
  searchText?: string
  icon?: ReactNode
}

interface SourceComboboxProps
  extends Omit<
    InputHTMLAttributes<HTMLInputElement>,
    'value' | 'onChange' | 'onSelect'
  > {
  value: string
  selectedLabel: string
  options: SourceComboboxOption[]
  onValueChange: (value: string) => void
  leadingIcon?: ReactNode
  loading?: boolean
  loadingLabel?: string
  error?: boolean
  errorLabel?: string
  onRetry?: () => void
  emptyLabel?: string
  emptyDescription?: string
  emptyActionLabel?: string
  onEmptyAction?: () => void
}

interface SourceComboboxFeedbackProps {
  loading: boolean
  loadingLabel: string
  error: boolean
  errorLabel: string
  onRetry?: () => void
  empty: boolean
  emptyLabel: string
  emptyDescription: string
  emptyActionLabel?: string
  onEmptyAction?: () => void
}

function filterOptions(options: SourceComboboxOption[], query: string) {
  const needle = query.trim().toLocaleLowerCase()
  return options.filter((option) =>
    `${option.searchText ?? ''} ${option.label} ${option.description ?? ''}`
      .toLocaleLowerCase()
      .includes(needle),
  )
}

interface ComboboxKeyDownOptions {
  event: KeyboardEvent<HTMLInputElement>
  open: boolean
  filteredOptions: SourceComboboxOption[]
  activeIndex: number
  setActiveIndex: Dispatch<SetStateAction<number>>
  openOptions: () => void
  closeOptions: () => void
  chooseOption: (option: SourceComboboxOption) => void
}

function handleComboboxKeyDown({
  event,
  open,
  filteredOptions,
  activeIndex,
  setActiveIndex,
  openOptions,
  closeOptions,
  chooseOption,
}: ComboboxKeyDownOptions) {
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    if (!open) {
      openOptions()
      if (filteredOptions.length) setActiveIndex(0)
    } else if (filteredOptions.length)
      setActiveIndex((current) => Math.min(current + 1, filteredOptions.length - 1))
  } else if (event.key === 'ArrowUp') {
    event.preventDefault()
    if (!open) {
      openOptions()
      if (filteredOptions.length) setActiveIndex(filteredOptions.length - 1)
    } else if (filteredOptions.length)
      setActiveIndex((current) =>
        current < 0 ? filteredOptions.length - 1 : Math.max(current - 1, 0),
      )
  } else if (event.key === 'Enter' && open && filteredOptions[activeIndex]) {
    event.preventDefault()
    chooseOption(filteredOptions[activeIndex])
  } else if (event.key === 'Escape' && open) {
    event.preventDefault()
    closeOptions()
  }
}

function SourceComboboxOptionList({
  options,
  value,
  activeIndex,
  listId,
  label,
  onChoose,
}: {
  options: SourceComboboxOption[]
  value: string
  activeIndex: number
  listId: string
  label: string
  onChoose: (option: SourceComboboxOption) => void
}) {
  return (
    <div
      className="grid"
      id={listId}
      role="listbox"
      aria-label={label}
    >
      {options.map((option, index) => (
        <button
          aria-selected={option.value === value}
          className={`flex min-h-11 w-full min-w-0 items-center gap-3 rounded-md px-3 py-2 text-left outline-none transition-colors duration-[var(--ely-duration-fast)] ${index === activeIndex ? 'bg-surface-2 text-text-primary' : 'text-text-secondary hover:bg-surface-2'}`}
          id={`${listId}-option-${index}`}
          key={option.value}
          onClick={() => onChoose(option)}
          onMouseDown={(event) => event.preventDefault()}
          role="option"
          type="button"
        >
          {option.icon ? (
            <span className="shrink-0 text-text-tertiary">{option.icon}</span>
          ) : null}
          <span className="min-w-0 flex-1">
            <span className="block truncate text-supporting">{option.label}</span>
            {option.description ? (
              <span className="block truncate text-label text-text-tertiary">
                {option.description}
              </span>
            ) : null}
          </span>
          {option.value === value ? (
            <Check
              aria-hidden="true"
              className="shrink-0 text-action"
              size={17}
              weight="bold"
            />
          ) : null}
        </button>
      ))}
    </div>
  )
}

function SourceComboboxFeedback({
  loading,
  loadingLabel,
  error,
  errorLabel,
  onRetry,
  empty,
  emptyLabel,
  emptyDescription,
  emptyActionLabel,
  onEmptyAction,
}: SourceComboboxFeedbackProps) {
  if (loading) {
    return (
      <div
        className="flex min-h-12 items-center gap-2 px-3 text-supporting text-text-tertiary"
        role="status"
      >
        <span className="size-4 animate-spin rounded-full border-2 border-action border-t-transparent" />
        {loadingLabel}
      </div>
    )
  }
  if (error) {
    return (
      <div
        className="grid gap-2 p-3"
        role="alert"
      >
        <p className="text-supporting text-danger">{errorLabel}</p>
        {onRetry ? (
          <Button
            onClick={onRetry}
            onMouseDown={(event) => event.preventDefault()}
            size="sm"
            tone="neutral"
            type="button"
          >
            Try again
          </Button>
        ) : null}
      </div>
    )
  }
  if (!empty) return null
  return (
    <div className="grid gap-1 px-3 py-3">
      <p className="text-supporting text-text-primary">{emptyLabel}</p>
      <p className="text-label text-text-tertiary">{emptyDescription}</p>
      {emptyActionLabel && onEmptyAction ? (
        <Button
          className="mt-2 justify-self-start"
          onClick={onEmptyAction}
          onMouseDown={(event) => event.preventDefault()}
          size="sm"
          tone="neutral"
          type="button"
        >
          {emptyActionLabel}
        </Button>
      ) : null}
    </div>
  )
}

export function SourceCombobox({
  value,
  selectedLabel,
  options,
  onValueChange,
  leadingIcon,
  loading = false,
  loadingLabel = 'Loading…',
  error = false,
  errorLabel = 'Unable to load options.',
  onRetry,
  emptyLabel = 'No results found',
  emptyDescription = 'Try another search.',
  emptyActionLabel,
  onEmptyAction,
  disabled = false,
  placeholder = 'Search…',
  id,
  className = '',
  ...inputProps
}: SourceComboboxProps) {
  const listId = useId()
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [activeIndex, setActiveIndex] = useState(-1)
  const filteredOptions = filterOptions(options, query)
  const currentActiveIndex =
    activeIndex < filteredOptions.length ? activeIndex : filteredOptions.length - 1
  const openOptions = () => {
    if (disabled) return
    setQuery('')
    setActiveIndex(-1)
    setOpen(true)
  }
  const closeOptions = () => {
    setOpen(false)
    setQuery('')
  }
  const chooseOption = (option: SourceComboboxOption) => {
    onValueChange(option.value)
    closeOptions()
  }
  const handleKeyDown: InputHTMLAttributes<HTMLInputElement>['onKeyDown'] = (event) => {
    inputProps.onKeyDown?.(event)
    if (!event.defaultPrevented) {
      handleComboboxKeyDown({
        event,
        open,
        filteredOptions,
        activeIndex: currentActiveIndex,
        setActiveIndex,
        openOptions,
        closeOptions,
        chooseOption,
      })
    }
  }

  return (
    <div className="relative min-w-0">
      <div className="relative">
        <span className="pointer-events-none absolute left-3 top-1/2 z-10 -translate-y-1/2 text-text-tertiary">
          {leadingIcon ?? (
            <MagnifyingGlass
              aria-hidden="true"
              size={17}
            />
          )}
        </span>
        <Input
          {...inputProps}
          id={id}
          role="combobox"
          aria-autocomplete="list"
          aria-controls={`${listId}-listbox`}
          aria-expanded={open}
          aria-activedescendant={
            open && filteredOptions[currentActiveIndex]
              ? `${listId}-listbox-option-${currentActiveIndex}`
              : undefined
          }
          aria-haspopup="listbox"
          autoComplete="off"
          className={`min-w-0 pr-12 pl-10 ${className}`}
          disabled={disabled}
          onBlur={() => closeOptions()}
          onChange={(event) => {
            setQuery(event.target.value)
            setActiveIndex(-1)
            setOpen(true)
          }}
          onFocus={() => {
            if (!open) openOptions()
          }}
          onKeyDown={handleKeyDown}
          placeholder={loading ? loadingLabel : placeholder}
          value={open ? query : selectedLabel}
        />
        <IconButton
          className="absolute right-1 top-1/2 -translate-y-1/2"
          disabled={disabled}
          label={open ? 'Close options' : 'Show options'}
          onClick={() => (open ? closeOptions() : openOptions())}
          onMouseDown={(event) => event.preventDefault()}
          size="sm"
          tabIndex={-1}
        >
          <CaretDown
            aria-hidden="true"
            size={16}
          />
        </IconButton>
      </div>
      <div
        className="absolute inset-x-0 top-full z-40 mt-1 max-h-64 overflow-y-auto rounded-lg border border-border-subtle bg-overlay p-1 shadow-elevation-2"
        hidden={!open}
      >
        <SourceComboboxOptionList
          activeIndex={currentActiveIndex}
          label={inputProps['aria-label'] ?? 'Options'}
          listId={`${listId}-listbox`}
          onChoose={chooseOption}
          options={loading || error ? [] : filteredOptions}
          value={value}
        />
        <SourceComboboxFeedback
          empty={!filteredOptions.length}
          emptyActionLabel={emptyActionLabel}
          emptyDescription={emptyDescription}
          emptyLabel={emptyLabel}
          error={error}
          errorLabel={errorLabel}
          loading={loading}
          loadingLabel={loadingLabel}
          onEmptyAction={onEmptyAction}
          onRetry={onRetry}
        />
      </div>
    </div>
  )
}

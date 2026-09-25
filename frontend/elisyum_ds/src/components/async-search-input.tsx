import {
  forwardRef,
  useState,
  type InputHTMLAttributes,
  type KeyboardEvent,
  type ReactNode,
} from 'react'
import { Spinner } from './spinner'
import { inputControlClasses } from './input-styles'

export interface AsyncSearchOption {
  value: string
  label: ReactNode
}
export interface AsyncSearchInputProps
  extends Omit<InputHTMLAttributes<HTMLInputElement>, 'results'> {
  results?: AsyncSearchOption[]
  loading?: boolean
  onOptionSelect?: (option: AsyncSearchOption) => void
}

function AsyncSearchResults({
  activeIndex,
  onSelect,
  options,
}: {
  activeIndex: number
  onSelect: (option: AsyncSearchOption) => void
  options: AsyncSearchOption[]
}) {
  return (
    <div
      className="absolute z-30 mt-1 max-h-56 w-full origin-top overflow-auto rounded-lg border border-border-default bg-overlay p-1 shadow-elevation-2 transition-[opacity,transform] duration-[var(--ely-duration-fast)] motion-reduce:transition-opacity"
      id="elysium-async-search-results"
      role="listbox"
    >
      {options.map((option, index) => (
        <button
          aria-selected={index === activeIndex}
          className={`block w-full cursor-pointer border-l-2 px-3 py-2 text-left text-supporting transition-[background-color,border-color,color] duration-[var(--ely-duration-fast)] motion-reduce:transition-none ${index === activeIndex ? 'border-action bg-surface-2 text-text-primary' : 'border-transparent text-text-secondary hover:border-action hover:bg-surface-2 hover:text-text-primary'}`}
          id={`elysium-async-search-option-${option.value}`}
          key={option.value}
          onMouseDown={(event) => event.preventDefault()}
          onClick={() => onSelect(option)}
          role="option"
          tabIndex={-1}
          type="button"
        >
          {option.label}
        </button>
      ))}
    </div>
  )
}

export const AsyncSearchInput = forwardRef<HTMLInputElement, AsyncSearchInputProps>(
  function AsyncSearchInputView(
    {
      className = '',
      loading = false,
      onOptionSelect,
      onKeyDown,
      results = [],
      ...props
    },
    ref,
  ) {
    const [activeIndex, setActiveIndex] = useState(-1)
    const selectOption = (option: AsyncSearchOption) => {
      onOptionSelect?.(option)
      setActiveIndex(-1)
    }
    const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
      onKeyDown?.(event)
      if (event.defaultPrevented || !results.length) return
      if (event.key === 'ArrowDown') {
        event.preventDefault()
        setActiveIndex((current) => (current + 1) % results.length)
      }
      if (event.key === 'ArrowUp') {
        event.preventDefault()
        setActiveIndex((current) => (current - 1 + results.length) % results.length)
      }
      if (event.key === 'Enter' && activeIndex >= 0) {
        event.preventDefault()
        selectOption(results[activeIndex])
      }
      if (event.key === 'Escape') setActiveIndex(-1)
    }
    return (
      <div className="relative">
        <div className="relative">
          <input
            {...props}
            aria-activedescendant={
              activeIndex >= 0
                ? `elysium-async-search-option-${results[activeIndex]?.value}`
                : undefined
            }
            aria-autocomplete="list"
            aria-busy={loading || undefined}
            aria-controls={results.length ? 'elysium-async-search-results' : undefined}
            aria-expanded={results.length > 0}
            onKeyDown={handleKeyDown}
            ref={ref}
            role="combobox"
            className={`${inputControlClasses} pr-10 ${className}`}
          />
          {loading ? (
            <Spinner
              className="absolute right-3 top-3"
              size="sm"
            />
          ) : null}
        </div>
        {results.length ? (
          <AsyncSearchResults
            activeIndex={activeIndex}
            onSelect={selectOption}
            options={results}
          />
        ) : null}
      </div>
    )
  },
)

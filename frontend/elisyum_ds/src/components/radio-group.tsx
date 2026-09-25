import type { FieldsetHTMLAttributes, ReactNode } from 'react'
import type { UseFormRegisterReturn } from 'react-hook-form'

export interface RadioOption {
  value: string
  label: ReactNode
  description?: ReactNode
  disabled?: boolean
}

export interface RadioGroupProps
  extends Omit<FieldsetHTMLAttributes<HTMLFieldSetElement>, 'onChange'> {
  options: RadioOption[]
  inputProps?: UseFormRegisterReturn
  value?: string
  onValueChange?: (value: string) => void
}

export function RadioGroup({
  className = '',
  inputProps,
  onValueChange,
  options,
  value,
  ...props
}: RadioGroupProps) {
  return (
    <fieldset
      {...props}
      className={`grid gap-px aria-invalid:border aria-invalid:border-danger aria-invalid:p-1 ${className}`}
    >
      {options.map((option) => (
        <label
          key={option.value}
          className="group flex cursor-pointer items-start gap-3 rounded-md border-l-2 border-transparent bg-surface-1 px-3 py-3 transition-[background-color,border-color] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:bg-surface-2 focus-within:border-focus focus-within:ring-2 focus-within:ring-focus/25 has-[:checked]:border-action has-[:checked]:bg-action-soft has-[:disabled]:cursor-not-allowed has-[:disabled]:opacity-45"
        >
          <input
            {...inputProps}
            checked={value === option.value ? true : undefined}
            disabled={option.disabled}
            name={inputProps?.name}
            type="radio"
            value={option.value}
            onChange={(event) => {
              inputProps?.onChange?.(event)
              onValueChange?.(event.target.value)
            }}
            className="mt-0.5 cursor-pointer accent-action focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus aria-invalid:outline-2 aria-invalid:outline-offset-2 aria-invalid:outline-danger disabled:cursor-not-allowed"
          />
          <span className="grid gap-0.5">
            <span className="text-body text-text-primary">{option.label}</span>
            {option.description ? (
              <span className="text-supporting text-text-tertiary">
                {option.description}
              </span>
            ) : null}
          </span>
        </label>
      ))}
    </fieldset>
  )
}

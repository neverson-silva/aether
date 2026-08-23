import { CaretDown } from '@phosphor-icons/react'
import { forwardRef, type SelectHTMLAttributes } from 'react'
import { Field } from '../field/field'

export interface NativeSelectOption {
  label: string
  value: string
  disabled?: boolean
}
export interface NativeSelectProps
  extends Omit<SelectHTMLAttributes<HTMLSelectElement>, 'size'> {
  label?: string
  description?: string
  error?: string
  options: NativeSelectOption[]
}
export const NativeSelect = forwardRef<HTMLSelectElement, NativeSelectProps>(function NativeSelect({
  className = '',
  description,
  error,
  label,
  options,
  required,
  ...props
}, ref) {
  const control = (
    <div style={{ position: 'relative', width: '100%' }} className="relative w-full">
      <select
        ref={ref}
        {...props}
        required={required}
        aria-invalid={Boolean(error) || undefined}
        className={`h-10 w-full appearance-none rounded-md border bg-surface-control px-3 pr-9 text-body-md text-foreground outline-none transition-[border-color,box-shadow] focus:border-primary focus:ring-2 focus:ring-ring/20 ${error ? 'border-status-danger' : 'border-border'} ${className}`}
      >
        {options.map((option) => (
          <option
            key={option.value}
            value={option.value}
            disabled={option.disabled}
          >
            {option.label}
          </option>
        ))}
      </select>
      <CaretDown
        size={16}
        style={{ position: 'absolute', right: '0.75rem', top: '50%', transform: 'translateY(-50%)' }}
        className="pointer-events-none text-muted-foreground"
        aria-hidden="true"
      />
    </div>
  )
  return label ? (
    <Field
      label={label}
      description={description}
      error={error}
      required={required}
      disabled={props.disabled}
    >
      {control}
    </Field>
  ) : (
    control
  )
})

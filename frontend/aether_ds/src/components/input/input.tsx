import { Input as BaseInput } from '@base-ui/react/input'
import type { Icon } from '@phosphor-icons/react'
import { forwardRef, type InputHTMLAttributes, type Ref } from 'react'
import { Field } from '../field/field'
export interface InputProps
  extends Omit<InputHTMLAttributes<HTMLInputElement>, 'size'> {
  label?: string
  description?: string
  error?: string
  leadingIcon?: Icon
  trailingIcon?: Icon
  clearable?: boolean
  onClear?: () => void
  loading?: boolean
  inputRef?: Ref<HTMLInputElement>
  size?: 'sm' | 'md' | 'lg'
}
export const Input = forwardRef<HTMLInputElement, InputProps>(function Input({
  className = '',
  description,
  error,
  label,
  leadingIcon: LeadingIcon,
  loading,
  onClear,
  trailingIcon: TrailingIcon,
  clearable,
  inputRef,
  size = 'md',
  ...props
}, ref) {
  const sizes = {
    sm: 'h-8 px-2.5 text-body-sm',
    md: 'h-10 px-3 text-body-md',
    lg: 'h-12 px-4 text-body-md',
  }
  const control = (
    <div
      className={`flex items-center gap-1 rounded-md border border-border bg-surface-card transition-colors focus-within:!border-primary focus-within:ring-2 focus-within:ring-primary/20 group-data-[invalid]:border-status-danger group-data-[invalid]:focus-within:!border-status-danger group-data-[invalid]:focus-within:ring-status-danger/20 ${error ? 'border-status-danger focus-within:!border-status-danger focus-within:ring-status-danger/20' : ''}`}
    >
      {LeadingIcon ? (
        <LeadingIcon
          className="ml-4 shrink-0 text-muted-foreground"
          size={18}
          aria-hidden="true"
        />
      ) : null}
      <BaseInput
        ref={inputRef ?? ref}
        className={`min-w-0 flex-1 bg-transparent outline-none ${sizes[size]} ${LeadingIcon ? 'pl-1 pr-3' : ''} ${className}`}
        aria-invalid={Boolean(error) || undefined}
        {...props}
      />
      {loading ? (
        <span
          className="mr-3 size-4 animate-spin rounded-full border-2 border-current border-t-transparent"
          aria-hidden="true"
        />
      ) : clearable && onClear ? (
        <button
          type="button"
          className="mr-2 text-muted-foreground"
          aria-label="Clear input"
          onClick={onClear}
        >
          ×
        </button>
      ) : TrailingIcon ? (
        <TrailingIcon
          className="mr-3 shrink-0 text-muted-foreground"
          size={18}
          aria-hidden="true"
        />
      ) : null}
    </div>
  )
  return label ? (
    <Field
      label={label}
      description={description}
      error={error}
      required={props.required}
    >
      {control}
    </Field>
  ) : (
    control
  )
})

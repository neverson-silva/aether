import type { TextareaHTMLAttributes } from 'react'
import { Field } from '../field/field'
export interface TextareaProps
  extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  label?: string
  description?: string
  error?: string
  autosize?: boolean
  code?: boolean
  maxLength?: number
  showCount?: boolean
}
export function Textarea({
  className = '',
  code,
  description,
  error,
  label,
  maxLength,
  showCount,
  ...props
}: TextareaProps) {
  const control = (
    <div className={`space-y-1 ${code ? 'font-mono' : ''}`}>
      <textarea
        className={`min-h-24 w-full resize-y rounded-lg border border-border bg-surface-control p-3 text-body-md shadow-sm outline-none transition-[background-color,border-color,box-shadow] duration-200 hover:bg-surface-container-highest/40 focus:border-primary focus:ring-2 focus:ring-ring/20 ${error ? 'border-status-danger' : ''} ${className}`}
        maxLength={maxLength}
        aria-invalid={Boolean(error) || undefined}
        {...props}
      />
      {showCount && maxLength ? (
        <div className="text-end text-body-sm text-muted-foreground">
          0/{maxLength}
        </div>
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
}

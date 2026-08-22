import { Field as BaseField } from '@base-ui/react/field'
import type { ReactNode } from 'react'
export interface FieldProps {
  name?: string
  label?: string
  description?: string
  error?: string
  required?: boolean
  disabled?: boolean
  children: ReactNode
}
export function Field({
  children,
  description,
  disabled,
  error,
  label,
  name,
  required,
}: FieldProps) {
  return (
    <BaseField.Root
      className="space-y-2"
      name={name}
      disabled={disabled}
      invalid={Boolean(error)}
    >
      <BaseField.Label className="block text-body-sm font-semibold text-foreground">
        {label}
        {required ? (
          <span className="text-destructive" aria-hidden="true">
            *
          </span>
        ) : null}
      </BaseField.Label>
      {children}
      {description ? (
        <BaseField.Description className="text-body-sm text-muted-foreground">
          {description}
        </BaseField.Description>
      ) : null}
      {error ? (
        <BaseField.Error className="text-body-sm text-status-danger">
          {error}
        </BaseField.Error>
      ) : null}
    </BaseField.Root>
  )
}

import { Children, cloneElement, isValidElement, type ReactNode } from 'react'
import { Label } from './label'

export interface FieldProps {
  id?: string
  label?: ReactNode
  description?: ReactNode
  error?: ReactNode
  required?: boolean
  children: ReactNode
}

export function Field({
  children,
  description,
  error,
  id,
  label,
  required = false,
}: FieldProps) {
  const messageId = id ? `${id}-${error ? 'error' : 'description'}` : undefined
  const control = Children.map(children, (child) => {
    if (!isValidElement(child)) return child
    const childProps = child.props as {
      id?: string
      'aria-describedby'?: string
      'aria-errormessage'?: string
      'aria-invalid'?: boolean | string
      'aria-required'?: boolean | string
      'data-invalid'?: string
    }
    const describedBy =
      [childProps['aria-describedby'], messageId].filter(Boolean).join(' ') || undefined
    return cloneElement(child, {
      'aria-describedby': describedBy,
      'aria-errormessage':
        error && id ? `${id}-error` : childProps['aria-errormessage'],
      'aria-invalid': error ? true : childProps['aria-invalid'],
      'aria-required': required ? true : childProps['aria-required'],
      'data-invalid': error ? 'true' : childProps['data-invalid'],
      id: childProps.id ?? id,
    } as never)
  })

  return (
    <div className="grid gap-2">
      {label ? (
        <Label htmlFor={id}>
          {label}
          {required ? (
            <span
              aria-hidden="true"
              className="ml-1 text-danger"
            >
              *
            </span>
          ) : null}
        </Label>
      ) : null}
      {control}
      {error ? (
        <p
          id={id ? `${id}-error` : undefined}
          role="alert"
          className="text-supporting text-danger"
        >
          {error}
        </p>
      ) : description ? (
        <p
          id={id ? `${id}-description` : undefined}
          className="text-supporting text-text-tertiary"
        >
          {description}
        </p>
      ) : null}
    </div>
  )
}

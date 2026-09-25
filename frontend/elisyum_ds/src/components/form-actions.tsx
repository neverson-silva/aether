import type { ReactNode } from 'react'
export interface FormActionsProps {
  primary: ReactNode
  secondary?: ReactNode
  className?: string
}
export function FormActions({ className = '', primary, secondary }: FormActionsProps) {
  return (
    <fieldset
      aria-label="Form actions"
      className={`flex flex-wrap items-center justify-end gap-2 border-t border-border-subtle pt-4 ${className}`}
    >
      {secondary}
      {primary}
    </fieldset>
  )
}

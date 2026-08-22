import type { HTMLAttributes, ReactNode } from 'react'
import { Button } from '../button/button'
export interface FormActionsProps extends HTMLAttributes<HTMLDivElement> {
  dirty?: boolean
  loading?: boolean
  error?: string
  success?: string
  onSave?: () => void
  onDiscard?: () => void
  saveLabel?: string
  discardLabel?: string
  children?: ReactNode
}
export function FormActions({
  children,
  className = '',
  discardLabel = 'Discard',
  dirty,
  error,
  loading,
  onDiscard,
  onSave,
  saveLabel = 'Save changes',
  success,
  ...props
}: FormActionsProps) {
  return (
    <div
      className={`flex items-center justify-between gap-4 border-t border-border pt-4 ${className}`}
      {...props}
    >
      <div className="text-body-sm text-muted-foreground">
        {error ? (
          <span className="text-destructive">{error}</span>
        ) : success ? (
          <span className="text-status-success">{success}</span>
        ) : dirty ? (
          'Unsaved changes'
        ) : null}
        {children}
      </div>
      <div className="flex gap-2">
        <Button
          variant="quiet"
          onClick={onDiscard}
          disabled={!dirty || loading}
        >
          {discardLabel}
        </Button>
        <Button onClick={onSave} loading={loading} disabled={!dirty}>
          {saveLabel}
        </Button>
      </div>
    </div>
  )
}

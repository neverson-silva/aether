import { AlertDialog as BaseAlertDialog } from '@base-ui/react/alert-dialog'
import type { ReactNode } from 'react'
import { Button } from './button'

export interface AlertDialogProps {
  trigger?: ReactNode
  triggerClassName?: string
  title: ReactNode
  description: ReactNode
  confirmLabel?: ReactNode
  onConfirm?: () => void
  open?: boolean
  onOpenChange?: (open: boolean) => void
}

export function AlertDialog({
  confirmLabel = 'Confirm',
  description,
  onConfirm,
  onOpenChange,
  open,
  title,
  trigger,
  triggerClassName,
}: AlertDialogProps) {
  return (
    <BaseAlertDialog.Root onOpenChange={onOpenChange} open={open}>
      {trigger ? <BaseAlertDialog.Trigger className={triggerClassName ?? 'inline-flex min-h-9 cursor-pointer items-center justify-center rounded-lg bg-action px-4 text-label font-medium text-on-action shadow-elevation-1 transition-[background-color,box-shadow,transform] duration-[var(--ely-duration-fast)] motion-safe:active:scale-[var(--ely-motion-press-scale)] hover:bg-action-strong hover:shadow-elevation-2 focus-visible:outline-2 focus-visible:outline-focus'}>
        {trigger}
      </BaseAlertDialog.Trigger> : null}
      <BaseAlertDialog.Portal>
        <BaseAlertDialog.Backdrop className="ely-modal-backdrop" />
        <BaseAlertDialog.Viewport className="ely-modal-viewport">
          <BaseAlertDialog.Popup className="ely-modal-surface grid gap-5 border-danger/40 p-6">
            <div className="grid gap-2">
              <BaseAlertDialog.Title className="text-section-title">
                {title}
              </BaseAlertDialog.Title>
              <BaseAlertDialog.Description className="text-supporting text-text-tertiary">
                {description}
              </BaseAlertDialog.Description>
            </div>
            <p className="text-body text-text-secondary">This action cannot be undone.</p>
            <div className="flex justify-end gap-2">
              <BaseAlertDialog.Close render={<Button tone="ghost">Cancel</Button>} />
              <BaseAlertDialog.Close
                render={
                  <Button
                    onClick={onConfirm}
                    tone="danger"
                  >
                    {confirmLabel}
                  </Button>
                }
              />
            </div>
          </BaseAlertDialog.Popup>
        </BaseAlertDialog.Viewport>
      </BaseAlertDialog.Portal>
    </BaseAlertDialog.Root>
  )
}

import { AlertDialog as BaseAlertDialog } from '@base-ui/react/alert-dialog'
import { Button } from '../button/button'
import { cloneElement, type MouseEvent, type ReactElement, type ReactNode, useState } from 'react'

export interface AlertDialogProps {
  trigger: ReactElement
  open?: boolean
  onOpenChange?: (open: boolean) => void
  title: string
  description: string
  confirmLabel?: string
  cancelLabel?: string
  onConfirm?: () => void
  confirmDisabled?: boolean
  children?: ReactNode
}

export function AlertDialog({
  children,
  cancelLabel = 'Cancel',
  confirmLabel = 'Confirm',
  description,
  onConfirm,
  title,
  trigger,
  open,
  onOpenChange,
  confirmDisabled = false,
}: AlertDialogProps) {
  const [internalOpen, setInternalOpen] = useState(false)
  const isControlled = open !== undefined
  const resolvedOpen = isControlled ? open : internalOpen
  const handleOpenChange = (nextOpen: boolean) => {
    if (!isControlled) setInternalOpen(nextOpen)
    onOpenChange?.(nextOpen)
  }
  const controlledTrigger = cloneElement(trigger, {
      onClick: (event: MouseEvent<HTMLElement>) => {
        trigger.props.onClick?.(event)
        handleOpenChange(true)
      },
  })
  return (
    <BaseAlertDialog.Root open={resolvedOpen} onOpenChange={handleOpenChange}>
      {controlledTrigger}
      <BaseAlertDialog.Portal>
        <BaseAlertDialog.Backdrop style={{ position: 'fixed', inset: 0, zIndex: 2147483000, opacity: 1 }} className="fixed inset-0 z-50 bg-black/60 transition-opacity duration-200 data-[starting-style]:opacity-0 data-[ending-style]:opacity-0" />
        <BaseAlertDialog.Viewport style={{ position: 'fixed', inset: 0, zIndex: 2147483001, display: 'flex', alignItems: 'center', justifyContent: 'center', width: '100%', padding: '1rem', pointerEvents: 'auto' }} className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <BaseAlertDialog.Popup
            className="box-border flex !w-[min(36rem,calc(100vw-2rem))] !max-w-[36rem] min-w-0 flex-none flex-col max-h-[calc(100vh-2rem)] overflow-y-auto rounded-xl border border-status-danger/40 bg-surface-modal p-6 shadow-lg outline-none data-[starting-style]:translate-y-2 data-[starting-style]:opacity-0 data-[ending-style]:translate-y-2 data-[ending-style]:opacity-0 transition-[transform,opacity] duration-200"
            style={{ opacity: 1, visibility: 'visible', pointerEvents: 'auto' }}
          >
            <BaseAlertDialog.Title className="text-headline-sm text-foreground">
              {title}
            </BaseAlertDialog.Title>
            <BaseAlertDialog.Description className="mt-2 text-body-md text-muted-foreground">
              {description}
            </BaseAlertDialog.Description>
            {children ? <div className="mt-4">{children}</div> : null}
            <div className="mt-6 flex justify-end gap-2">
              <BaseAlertDialog.Close className="rounded-md border border-border px-3 py-2 text-body-sm text-foreground transition-colors hover:bg-surface-container">
                {cancelLabel}
              </BaseAlertDialog.Close>
              <BaseAlertDialog.Close
                render={<Button variant="danger" size="sm" />}
                onClick={onConfirm}
                disabled={confirmDisabled}
              >
                {confirmLabel}
              </BaseAlertDialog.Close>
            </div>
          </BaseAlertDialog.Popup>
        </BaseAlertDialog.Viewport>
      </BaseAlertDialog.Portal>
    </BaseAlertDialog.Root>
  )
}

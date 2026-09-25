import type { ReactNode } from 'react'
import { Dialog as BaseDialog } from '@base-ui/react/dialog'
import { Button } from './button'

export interface DialogProps {
  trigger: ReactNode
  title?: ReactNode
  description?: ReactNode
  children: ReactNode
  footer?: ReactNode
  hideCancel?: boolean
  open?: boolean
  defaultOpen?: boolean
  onOpenChange?: (open: boolean) => void
  popupClassName?: string
  viewportClassName?: string
  triggerClassName?: string
}

export function Dialog({
  children,
  defaultOpen,
  description,
  footer,
  hideCancel = false,
  onOpenChange,
  open,
  popupClassName = '',
  title,
  trigger,
  triggerClassName = 'inline-flex min-h-9 cursor-pointer items-center justify-center gap-2 rounded-lg bg-action px-4 text-label font-medium text-on-action shadow-elevation-1 transition-[background-color,box-shadow,transform] duration-[var(--ely-duration-fast)] motion-safe:active:scale-[var(--ely-motion-press-scale)] hover:bg-action-strong hover:shadow-elevation-2 focus-visible:outline-2 focus-visible:outline-focus',
  viewportClassName = '',
}: DialogProps) {
  return (
    <BaseDialog.Root
      defaultOpen={defaultOpen}
      onOpenChange={onOpenChange}
      open={open}
    >
      <BaseDialog.Trigger className={triggerClassName}>
        {trigger}
      </BaseDialog.Trigger>
      <BaseDialog.Portal>
        <BaseDialog.Backdrop className="ely-modal-backdrop" />
        <BaseDialog.Viewport className={`ely-modal-viewport ${viewportClassName}`}>
          <BaseDialog.Popup
            className={`ely-modal-surface grid gap-5 p-6 ${popupClassName}`}
          >
            {title || description ? (
              <div className="grid gap-2">
                {title ? <BaseDialog.Title className="text-section-title">{title}</BaseDialog.Title> : null}
                {description ? (
                  <BaseDialog.Description className="text-supporting text-text-tertiary">
                    {description}
                  </BaseDialog.Description>
                ) : null}
              </div>
            ) : null}
            <div>{children}</div>
            <div className="flex justify-end gap-2">
              {hideCancel ? null : <BaseDialog.Close render={<Button tone="ghost">Cancel</Button>} />}
              {footer}
            </div>
          </BaseDialog.Popup>
        </BaseDialog.Viewport>
      </BaseDialog.Portal>
    </BaseDialog.Root>
  )
}

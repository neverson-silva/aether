import { Dialog as BaseDialog } from '@base-ui/react/dialog'
import { X } from '@phosphor-icons/react'
import { cloneElement, type MouseEvent, type ReactElement, type ReactNode, useState } from 'react'

export interface DialogProps {
  trigger?: ReactElement
  children: ReactNode
  title?: string
  description?: string
  open?: boolean
  onOpenChange?: (open: boolean) => void
  showHeader?: boolean
  size?: 'sm' | 'md' | 'lg' | 'wizard'
  overflow?: 'auto' | 'hidden'
}

export function Dialog({
  children,
  description,
  onOpenChange,
  open,
  showHeader = true,
  size = 'md',
  title,
  trigger,
  overflow = 'auto',
}: DialogProps) {
  const [internalOpen, setInternalOpen] = useState(false)
  const isControlled = open !== undefined
  const resolvedOpen = isControlled ? open : internalOpen
  const handleOpenChange = (nextOpen: boolean) => {
    if (!isControlled) setInternalOpen(nextOpen)
    onOpenChange?.(nextOpen)
  }
  const sizeClass = {
    sm: 'max-w-xl',
    md: 'max-w-2xl',
    lg: 'max-w-5xl',
    wizard: 'max-w-5xl',
  }[size]
  const maxWidth = {
    sm: '36rem',
    md: '48rem',
    lg: '64rem',
    wizard: '64rem',
  }[size]
  const controlledTrigger = !trigger
    ? trigger
    : cloneElement(trigger, {
        onClick: (event: MouseEvent<HTMLElement>) => {
          trigger.props.onClick?.(event)
          handleOpenChange(true)
        },
      })
  return (
    <BaseDialog.Root open={resolvedOpen} onOpenChange={handleOpenChange}>
      {controlledTrigger}
      <BaseDialog.Portal>
        <BaseDialog.Backdrop style={{ position: 'fixed', inset: 0, zIndex: 2147483000, opacity: 1 }} className="fixed inset-0 z-[90] bg-black/60 transition-opacity duration-200 data-[starting-style]:opacity-0 data-[ending-style]:opacity-0" />
        <BaseDialog.Viewport style={{ position: 'fixed', inset: 0, zIndex: 2147483001, display: 'flex', alignItems: 'center', justifyContent: 'center', width: '100%', padding: '1rem', pointerEvents: 'auto' }} className="fixed inset-0 z-[90] flex w-full items-center justify-center p-4">
          <BaseDialog.Popup style={{ width: 'calc(100vw - 2rem)', maxWidth, height: size === 'wizard' ? 'calc(100vh - 2rem)' : undefined, maxHeight: 'calc(100vh - 2rem)', overflowY: size === 'wizard' || overflow === 'hidden' ? 'hidden' : 'auto', opacity: 1, visibility: 'visible', pointerEvents: 'auto' }} className={`!box-border !w-[calc(100vw-2rem)] ${sizeClass} !min-w-0 max-h-[calc(100vh-2rem)] ${size === 'wizard' || overflow === 'hidden' ? 'overflow-hidden' : 'overflow-y-auto'} rounded-xl border border-border bg-surface-modal text-foreground shadow-lg outline-none data-[starting-style]:translate-y-2 data-[starting-style]:scale-[0.98] data-[starting-style]:opacity-0 data-[ending-style]:translate-y-2 data-[ending-style]:scale-[0.98] data-[ending-style]:opacity-0 transition-[transform,opacity] duration-200`}>
            {showHeader && (title || description) ? (
              <div className="flex items-start justify-between gap-4 px-6 pt-5">
                <div className="min-w-0">
                  {title ? <BaseDialog.Title className="text-headline-sm">{title}</BaseDialog.Title> : null}
                  {description ? (
                    <BaseDialog.Description className="mt-2 text-body-md text-muted-foreground">
                      {description}
                    </BaseDialog.Description>
                  ) : null}
                </div>
                <BaseDialog.Close
                  aria-label="Close dialog"
                  className="inline-flex size-9 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-surface-container hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  <X size={20} aria-hidden="true" />
                </BaseDialog.Close>
              </div>
            ) : null}
            <div className={`${size === 'wizard' ? 'flex h-full min-h-0 flex-col' : ''} ${showHeader && (title || description) ? 'mt-5' : ''}`}>{children}</div>
          </BaseDialog.Popup>
        </BaseDialog.Viewport>
      </BaseDialog.Portal>
    </BaseDialog.Root>
  )
}

import { Drawer as BaseDrawer } from '@base-ui/react/drawer'
import type { ReactElement, ReactNode } from 'react'

export interface DrawerProps {
  trigger: ReactElement
  children: ReactNode
  title?: string
  description?: string
  side?: 'left' | 'right' | 'bottom'
}

export function Drawer({
  children,
  description,
  side = 'left',
  title,
  trigger,
}: DrawerProps) {
  const position =
    side === 'right'
      ? 'ml-auto border-l data-[starting-style]:translate-x-full data-[ending-style]:translate-x-full'
      : side === 'bottom'
        ? 'mt-auto h-auto w-full border-t data-[starting-style]:translate-y-full data-[ending-style]:translate-y-full'
        : 'border-r data-[starting-style]:-translate-x-full data-[ending-style]:-translate-x-full'
  return (
    <BaseDrawer.Root>
      <BaseDrawer.Trigger render={trigger} />
      <BaseDrawer.Portal>
        <BaseDrawer.Backdrop className="fixed inset-0 z-50 bg-black/60 transition-opacity duration-300 data-[starting-style]:opacity-0 data-[ending-style]:opacity-0" />
        <BaseDrawer.Viewport className="fixed inset-0 z-50 flex">
          <BaseDrawer.Popup
            className={`m-0 flex h-full w-80 flex-col border-border bg-surface-modal p-6 text-foreground shadow-xl transition-transform duration-300 ease-[cubic-bezier(0.2,0.8,0.2,1)] ${position}`}
          >
            {title ? (
              <BaseDrawer.Title className="font-semibold">
                {title}
              </BaseDrawer.Title>
            ) : null}
            {description ? (
              <BaseDrawer.Description className="mt-1 text-body-sm text-muted-foreground">
                {description}
              </BaseDrawer.Description>
            ) : null}
            <div className="mt-6 min-h-0 flex-1 overflow-auto">{children}</div>
            <BaseDrawer.Close className="mt-4 self-end rounded-md border border-border px-3 py-2 text-body-sm transition-colors hover:bg-surface-container">
              Close
            </BaseDrawer.Close>
          </BaseDrawer.Popup>
        </BaseDrawer.Viewport>
      </BaseDrawer.Portal>
    </BaseDrawer.Root>
  )
}

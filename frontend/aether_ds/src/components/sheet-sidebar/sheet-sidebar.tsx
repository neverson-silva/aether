import { Drawer } from '@base-ui/react/drawer'
import type { ReactElement, ReactNode } from 'react'
export interface SheetSidebarProps {
  trigger: ReactNode
  children: ReactNode
  title?: string
  description?: string
  side?: 'left' | 'right' | 'bottom'
}
export function SheetSidebar({
  children,
  description,
  side = 'left',
  title,
  trigger,
}: SheetSidebarProps) {
  return (
    <Drawer.Root>
      <Drawer.Trigger render={trigger as ReactElement} />
      <Drawer.Portal>
        <Drawer.Backdrop className="fixed inset-0 z-50 bg-black/50 opacity-100 transition-opacity duration-300 ease-[cubic-bezier(0.2,0.8,0.2,1)] data-[starting-style]:opacity-0 data-[ending-style]:opacity-0" />
        <Drawer.Viewport className="fixed inset-0 z-50 flex">
          <Drawer.Popup
            className={`m-0 flex h-full w-80 flex-col border-border bg-surface-card p-6 shadow-xl transition-transform duration-300 ease-[cubic-bezier(0.2,0.8,0.2,1)] ${side === 'right' ? 'ml-auto border-l data-[starting-style]:translate-x-full data-[ending-style]:translate-x-full' : side === 'bottom' ? 'mt-auto h-auto w-full border-t data-[starting-style]:translate-y-full data-[ending-style]:translate-y-full' : 'border-r data-[starting-style]:-translate-x-full data-[ending-style]:-translate-x-full'}`}
          >
            <Drawer.Title className="font-semibold text-foreground">
              {title}
            </Drawer.Title>
            {description ? (
              <Drawer.Description className="mt-1 text-body-sm text-muted-foreground">
                {description}
              </Drawer.Description>
            ) : null}
            <div className="mt-6 flex-1 overflow-auto">{children}</div>
            <Drawer.Close className="mt-4 self-end rounded-md border border-border px-3 py-2 text-body-sm">
              Close
            </Drawer.Close>
          </Drawer.Popup>
        </Drawer.Viewport>
      </Drawer.Portal>
    </Drawer.Root>
  )
}

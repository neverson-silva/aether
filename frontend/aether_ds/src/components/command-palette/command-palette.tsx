import { Dialog } from '@base-ui/react/dialog'
import { MagnifyingGlass } from '@phosphor-icons/react'
import { type ReactElement, type ReactNode, useEffect, useMemo, useState } from 'react'

export interface CommandPaletteItem {
  id: string
  label: string
  description?: string
  group?: string
  shortcut?: string
  icon?: ReactElement
  disabled?: boolean
  onSelect?: () => void
}
export interface CommandPaletteProps {
  trigger?: ReactElement
  items: CommandPaletteItem[]
  placeholder?: string
  empty?: ReactNode
  open?: boolean
  onOpenChange?: (open: boolean) => void
  query?: string
  onQueryChange?: (query: string) => void
}

export function CommandPalette({
  empty = 'No commands found.',
  items,
  onOpenChange,
  open,
  onQueryChange,
  query: controlledQuery,
  placeholder = 'Search commands',
  trigger,
}: CommandPaletteProps) {
  const [internalQuery, setInternalQuery] = useState('')
  const query = controlledQuery ?? internalQuery
  const filtered = useMemo(
    () =>
      items.filter((item) =>
        `${item.label} ${item.description ?? ''}`
          .toLowerCase()
          .includes(query.toLowerCase()),
      ),
    [items, query],
  )
  return (
    <Dialog.Root
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) {
          setInternalQuery('')
          onQueryChange?.('')
        }
        onOpenChange?.(nextOpen)
      }}
    >
      {trigger ? <Dialog.Trigger nativeButton={trigger.type === 'button'} render={trigger} /> : null}
      <Dialog.Portal>
        <Dialog.Backdrop style={{ position: 'fixed', inset: 0, zIndex: 2147483000, opacity: 1 }} className="fixed inset-0 z-[100] bg-overlay-backdrop transition-opacity duration-200 data-[starting-style]:opacity-0 data-[ending-style]:opacity-0" />
        <Dialog.Viewport style={{ position: 'fixed', inset: 0, zIndex: 2147483001, display: 'flex', alignItems: 'flex-start', justifyContent: 'center', width: '100%', padding: '10vh 1rem 1rem', pointerEvents: 'auto' }} className="fixed inset-0 z-[100] flex w-full items-start justify-center p-4 pt-[10vh]">
          <Dialog.Popup style={{ width: 'min(92vw, 48rem)', maxWidth: '48rem', maxHeight: 'calc(42.336vh + 5rem)', opacity: 1, visibility: 'visible', pointerEvents: 'auto' }} className="!box-border !w-[min(92vw,48rem)] !max-w-[48rem] !min-w-0 max-h-[calc(42.336vh+5rem)] overflow-hidden rounded-xl border border-border bg-surface-popover shadow-lg outline-none data-[starting-style]:translate-y-2 data-[starting-style]:opacity-0 data-[ending-style]:translate-y-2 data-[ending-style]:opacity-0 transition-[transform,opacity] duration-200">
            <div className="flex items-center gap-3 border-b border-border px-4">
              <MagnifyingGlass
                size={20}
                className="text-muted-foreground"
                aria-hidden="true"
              />
              <input
                id="aether-command-palette-search"
                name="command-palette-search"
                value={query}
                onChange={(event) => {
                  setInternalQuery(event.target.value)
                  onQueryChange?.(event.target.value)
                }}
                placeholder={placeholder}
                className="h-14 min-w-0 flex-1 bg-transparent text-body-md text-foreground outline-none placeholder:text-muted-foreground"
                aria-label={placeholder}
              />
              <kbd className="rounded border border-border px-1.5 py-0.5 text-label-caps text-muted-foreground">
                ESC
              </kbd>
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', maxHeight: 'min(42.336vh, 22.5792rem)', overflowY: 'auto', overflowX: 'hidden' }} className="max-h-[min(42.336vh,22.5792rem)] gap-1 overflow-x-hidden overflow-y-auto p-2">
              {filtered.length ? (
                filtered.map((item) => (
                  <button
                    key={item.id}
                    type="button"
                    disabled={item.disabled}
                    onClick={item.onSelect}
                    style={{ display: 'flex', flexDirection: 'row', alignItems: 'center', justifyContent: 'flex-start', width: '100%', textAlign: 'left' }}
                    className="!flex !w-full !flex-row !items-center !justify-start !self-stretch gap-3 rounded-md px-3 py-3 !text-left outline-none transition-colors hover:bg-surface-container focus-visible:ring-2 focus-visible:ring-primary disabled:opacity-50"
                  >
                    {item.icon ? (
                      <span className="flex size-5 shrink-0 items-center justify-center text-muted-foreground">{item.icon}</span>
                    ) : null}
                    <span style={{ textAlign: 'left' }} className="min-w-0 flex-1 !text-left">
                      <span className="block !text-left text-body-sm text-foreground">
                        {item.label}
                      </span>
                      {item.description ? (
                        <span className="block truncate text-body-sm text-muted-foreground">
                          {item.description}
                        </span>
                      ) : null}
                    </span>
                    {item.shortcut ? (
                      <kbd className="text-label-caps text-muted-foreground">
                        {item.shortcut}
                      </kbd>
                    ) : null}
                  </button>
                ))
              ) : (
                <div className="p-8 text-center text-body-sm text-muted-foreground">
                  {empty}
                </div>
              )}
            </div>
          </Dialog.Popup>
        </Dialog.Viewport>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

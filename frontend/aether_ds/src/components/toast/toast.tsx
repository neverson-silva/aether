import type { ToastManager } from '@base-ui/react/toast'
import { Toast as BaseToast } from '@base-ui/react/toast'
import { CheckCircle, Info, Warning, X, XCircle } from '@phosphor-icons/react'
import { type ReactNode, useMemo } from 'react'

export type ToastTone = 'info' | 'success' | 'warning' | 'error'
export interface ToastOptions {
  id?: string
  title: ReactNode
  description?: ReactNode
  tone?: ToastTone
  duration?: number
  persistent?: boolean
  priority?: 'low' | 'high'
  action?: { label: string; onClick: () => void }
}
export interface ToastProviderProps {
  children: ReactNode
  manager?: ToastManager
  limit?: number
  duration?: number
  position?: 'bottom-right' | 'bottom-left' | 'top-right' | 'top-left'
}

const toneStyles: Record<ToastTone, string> = {
  info: 'border-l-4 border-status-info bg-surface-popover text-foreground',
  success: 'border-l-4 border-status-success bg-surface-popover text-foreground',
  warning: 'border-l-4 border-status-warning bg-surface-popover text-foreground',
  error: 'border-l-4 border-status-danger bg-surface-popover text-foreground',
}
const toneIcons = {
  info: Info,
  success: CheckCircle,
  warning: Warning,
  error: XCircle,
}
const toneColors: Record<ToastTone, string> = {
  info: 'color-mix(in srgb, var(--semantic-status-info-container) 42%, var(--semantic-surface-popover))',
  success: 'color-mix(in srgb, var(--semantic-status-success-container) 42%, var(--semantic-surface-popover))',
  warning: 'color-mix(in srgb, var(--semantic-status-warning-container) 42%, var(--semantic-surface-popover))',
  error: 'color-mix(in srgb, var(--semantic-status-danger-container) 42%, var(--semantic-surface-popover))',
}
const toneBorders: Record<ToastTone, string> = {
  info: 'var(--semantic-status-info)',
  success: 'var(--semantic-status-success)',
  warning: 'var(--semantic-status-warning)',
  error: 'var(--semantic-status-danger)',
}

function ToastViewport({
  position = 'bottom-right',
}: {
  position?: ToastProviderProps['position']
}) {
  const { toasts } = BaseToast.useToastManager()
  const positionClass = {
    'bottom-right': 'bottom-4 right-4',
    'bottom-left': 'bottom-4 left-4',
    'top-right': 'top-4 right-4',
    'top-left': 'top-4 left-4',
  }[position ?? 'bottom-right']
  const positionStyle = position === 'bottom-left'
    ? { bottom: '1rem', left: '1rem', top: 'auto', right: 'auto' }
    : position === 'top-right'
      ? { top: '1rem', right: '1rem', bottom: 'auto', left: 'auto' }
      : position === 'top-left'
        ? { top: '1rem', left: '1rem', bottom: 'auto', right: 'auto' }
        : { bottom: '1rem', right: '1rem', top: 'auto', left: 'auto' }
  return (
    <BaseToast.Viewport
      className={`fixed ${positionClass} z-[100] flex w-[30rem] max-w-[calc(100vw-2rem)] flex-col gap-2 outline-none`}
      style={positionStyle}
    >
      <div className="flex flex-col gap-2">
        {toasts.map((toast) => {
          const tone = (toast.type as ToastTone) ?? 'info'
          const Icon = toneIcons[tone]
          return (
            <BaseToast.Root
              key={toast.id}
              toast={toast}
              swipeDirection={['right', 'down']}
              style={{ backgroundColor: toneColors[tone], borderLeftColor: toneBorders[tone], width: '30rem', maxWidth: 'calc(100vw - 2rem)', minWidth: 0, boxSizing: 'border-box' }}
              className={`relative w-full min-w-0 max-w-full overflow-hidden rounded-lg border p-4 text-foreground shadow-lg outline-none transition-[transform,opacity] duration-200 ease-[var(--motion-ease-standard)] data-[starting-style]:translate-y-2 data-[starting-style]:opacity-0 data-[ending-style]:translate-y-2 data-[ending-style]:opacity-0 data-[ending-style]:duration-150 ${toneStyles[tone]}`}
            >
              <div className="flex min-w-0 gap-3">
                <Icon
                  size={20}
                  className="mt-0.5 shrink-0"
                  aria-hidden="true"
                />
                <BaseToast.Content className="min-w-0 flex-1 overflow-hidden">
                  <BaseToast.Title className="block min-w-0 overflow-hidden text-ellipsis whitespace-nowrap font-semibold">
                    {toast.title}
                  </BaseToast.Title>
                  {toast.description ? (
                    <BaseToast.Description className="mt-1 block min-w-0 overflow-hidden text-ellipsis whitespace-nowrap text-body-sm text-muted-foreground">
                      {toast.description}
                    </BaseToast.Description>
                  ) : null}
                  {toast.actionProps ? (
                    <BaseToast.Action
                      {...toast.actionProps}
                      className="mt-3 max-w-full overflow-hidden text-ellipsis whitespace-nowrap rounded-md border border-border px-2 py-1 text-body-sm hover:bg-surface-container"
                    />
                  ) : null}
                </BaseToast.Content>
                <BaseToast.Close
                  className="shrink-0 text-muted-foreground hover:text-foreground"
                  aria-label="Dismiss notification"
                >
                  <X size={16} />
                </BaseToast.Close>
              </div>
            </BaseToast.Root>
          )
        })}
      </div>
    </BaseToast.Viewport>
  )
}

export function ToastProvider({
  children,
  duration = 2400,
  limit = 3,
  manager,
  position = 'bottom-right',
}: ToastProviderProps) {
  const toastManager = useMemo(
    () => manager ?? BaseToast.createToastManager(),
    [manager],
  )
  return (
    <BaseToast.Provider
      timeout={duration}
      limit={limit}
      toastManager={toastManager}
    >
      <ToastViewport position={position} />
      {children}
    </BaseToast.Provider>
  )
}

export function useToast() {
  const manager = BaseToast.useToastManager()
  return {
    add: (options: ToastOptions) =>
      manager.add({
        id: options.id,
        title: options.title,
        description: options.description,
        type: options.tone ?? 'info',
        timeout: options.persistent ? 0 : options.duration ?? 3000,
        priority: options.priority ?? 'low',
        actionProps: options.action
          ? { children: options.action.label, onClick: options.action.onClick }
          : undefined,
      }),
    close: manager.close,
    update: manager.update,
    promise: manager.promise,
  }
}

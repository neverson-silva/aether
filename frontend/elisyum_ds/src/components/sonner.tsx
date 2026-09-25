import type { CSSProperties, ReactNode } from 'react'
import { Toaster, toast } from 'sonner'
import { useElisyum } from '../providers'

export interface SonnerProps {
  className?: string
}
export function Sonner({ className }: SonnerProps) {
  const { resolvedTheme } = useElisyum()
  return (
    <Toaster
      className={className}
      closeButton
      containerAriaLabel="System notifications"
      gap={10}
      mobileOffset={{ bottom: '16px', right: '16px' }}
      offset={{ bottom: '24px', right: '24px' }}
      position="bottom-right"
      richColors
      style={{
        '--toast-close-button-end': '0px',
        '--toast-close-button-start': 'unset',
        '--toast-close-button-transform': 'translate(35%, -35%)',
      } as CSSProperties}
      theme={resolvedTheme}
      toastOptions={{
        className: 'rounded-lg border shadow-elevation-2',
        duration: 5000,
      }}
      visibleToasts={4}
    />
  )
}
export type ToastTone = 'error' | 'info' | 'success' | 'warning'

export function showToast(message: string, tone: ToastTone = 'info') {
  return toast[tone](message)
}
export interface ToastProviderProps {
  children?: ReactNode
}
export function ToastProvider({ children }: ToastProviderProps) {
  return (
    <>
      {children}
      <Sonner />
    </>
  )
}
export function Toast({ message }: { message: string }) {
  return (
    <button
      className="cursor-pointer text-left text-supporting text-action-strong"
      onClick={() => showToast(message)}
      type="button"
    >
      {message}
    </button>
  )
}

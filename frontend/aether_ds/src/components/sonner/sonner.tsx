import type { ReactNode } from 'react'
import { ToastProvider, type ToastProviderProps } from '../toast/toast'

export interface SonnerProps extends Omit<ToastProviderProps, 'children'> {
  children: ReactNode
  position?: 'bottom-right' | 'bottom-left' | 'top-right' | 'top-left'
}
export function Sonner({
  children,
  position = 'bottom-right',
  ...props
}: SonnerProps) {
  return (
    <ToastProvider {...props} position={position}>
      {children}
    </ToastProvider>
  )
}

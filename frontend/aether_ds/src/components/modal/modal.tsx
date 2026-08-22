import type { ReactElement, ReactNode } from 'react'
import { Dialog } from '../dialog/dialog'

export interface ModalProps {
  children: ReactNode
  title?: string
  description?: string
  trigger?: ReactElement
  open?: boolean
  onOpenChange?: (open: boolean) => void
  size?: 'sm' | 'md' | 'lg' | 'wizard'
  showHeader?: boolean
}

export function Modal({
  children,
  description,
  onOpenChange,
  open,
  showHeader,
  size,
  title,
  trigger,
}: ModalProps) {
  return (
    <Dialog
      trigger={trigger}
      title={title}
      description={description}
      open={open}
      onOpenChange={onOpenChange}
      showHeader={showHeader}
      size={size}
    >
      {children}
    </Dialog>
  )
}

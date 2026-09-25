import type { ReactNode } from 'react'
import { Dialog } from './dialog'

export interface SpotlightProps {
  trigger: ReactNode
  title: ReactNode
  children: ReactNode
}

export function Spotlight({ children, title, trigger }: SpotlightProps) {
  return (
    <Dialog
      description="A focused product moment."
      popupClassName="max-w-2xl"
      title={title}
      trigger={trigger}
    >
      <div className="grid gap-4">{children}</div>
    </Dialog>
  )
}

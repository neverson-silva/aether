import type { ReactElement, ReactNode } from 'react'
import { Tooltip as BaseTooltip } from '@base-ui/react/tooltip'

export interface TooltipProps {
  trigger: ReactElement
  children: ReactNode
}

export function Tooltip({ children, trigger }: TooltipProps) {
  return (
    <BaseTooltip.Root>
      <BaseTooltip.Trigger render={trigger} className="focus-visible:outline-2 focus-visible:outline-focus" />
      <BaseTooltip.Portal>
        <BaseTooltip.Positioner sideOffset={6}>
          <BaseTooltip.Popup className="z-40 origin-[var(--transform-origin)] rounded-sm border border-border-default bg-surface-4 px-2.5 py-1.5 text-supporting text-text-primary shadow-elevation-1 transition-[opacity,transform] duration-[var(--ely-duration-fast)] ease-ely-out data-[ending-style]:scale-[.98] data-[ending-style]:opacity-0 data-[starting-style]:scale-[.98] data-[starting-style]:opacity-0 motion-reduce:transition-opacity">
            {children}
          </BaseTooltip.Popup>
        </BaseTooltip.Positioner>
      </BaseTooltip.Portal>
    </BaseTooltip.Root>
  )
}

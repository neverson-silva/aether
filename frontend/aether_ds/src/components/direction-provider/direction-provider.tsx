import {
  DirectionProvider as BaseDirectionProvider,
  type TextDirection,
} from '@base-ui/react/direction-provider'
import type { ReactNode } from 'react'
export interface DirectionProviderProps {
  direction?: TextDirection
  children: ReactNode
}
export function DirectionProvider({
  children,
  direction = 'ltr',
}: DirectionProviderProps) {
  return (
    <BaseDirectionProvider direction={direction}>
      {children}
    </BaseDirectionProvider>
  )
}

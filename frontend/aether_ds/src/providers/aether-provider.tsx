import type { ReactNode } from 'react'
import { ToastProvider, type ToastProviderProps } from '../components/toast/toast'
import { ThemeProvider, type ThemeProviderProps } from '../theme/theme-provider'
import { CommandPaletteProvider } from './command-palette-provider'
import { OverlayProvider } from './overlay-provider'

export interface AetherProviderProps
  extends Omit<ThemeProviderProps, 'children'>,
    Omit<ToastProviderProps, 'children'> {
  children: ReactNode
}

export function AetherProvider({ children, ...props }: AetherProviderProps) {
  const { defaultTheme, persist, storageKey, config, manager, limit, duration, position } = props
  return (
    <ThemeProvider
      defaultTheme={defaultTheme}
      persist={persist}
      storageKey={storageKey}
      config={config}
    >
      <ToastProvider
        manager={manager}
        limit={limit}
        duration={duration}
        position={position}
      >
        <OverlayProvider>
          <CommandPaletteProvider>{children}</CommandPaletteProvider>
        </OverlayProvider>
      </ToastProvider>
    </ThemeProvider>
  )
}

import { createContext, type ReactNode, useContext, useMemo } from 'react'

interface OverlayContextValue {
  rootId: string
}

const OverlayContext = createContext<OverlayContextValue | null>(null)

export function OverlayProvider({ children }: { children: ReactNode }) {
  const value = useMemo(() => ({ rootId: 'aether-overlay-root' }), [])
  return <OverlayContext.Provider value={value}>{children}</OverlayContext.Provider>
}

export function useOverlay() {
  const context = useContext(OverlayContext)
  if (!context) throw new Error('useOverlay must be used within AetherProvider')
  return context
}

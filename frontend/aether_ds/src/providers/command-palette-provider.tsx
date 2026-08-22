import { createContext, type ReactNode, useCallback, useContext, useMemo, useState } from 'react'

interface CommandPaletteContextValue {
  open: boolean
  setOpen: (open: boolean) => void
  openCommandPalette: () => void
  closeCommandPalette: () => void
}

const CommandPaletteContext = createContext<CommandPaletteContextValue | null>(null)

export function CommandPaletteProvider({ children }: { children: ReactNode }) {
  const [open, setOpenState] = useState(false)
  const setOpen = useCallback((nextOpen: boolean) => setOpenState(nextOpen), [])
  const openCommandPalette = useCallback(() => setOpenState(true), [])
  const closeCommandPalette = useCallback(() => setOpenState(false), [])
  const value = useMemo(
    () => ({ open, setOpen, openCommandPalette, closeCommandPalette }),
    [closeCommandPalette, open, openCommandPalette, setOpen],
  )
  return <CommandPaletteContext.Provider value={value}>{children}</CommandPaletteContext.Provider>
}

export function useCommandPalette() {
  const context = useContext(CommandPaletteContext)
  if (!context) throw new Error('useCommandPalette must be used within AetherProvider')
  return context
}

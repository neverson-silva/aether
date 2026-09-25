import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type CSSProperties,
  type ReactNode,
} from 'react'
import {
  mergeTokenOverrides,
  tokenOverridesToStyle,
  type ElisyumContrast,
  type ElisyumDensity,
  type ElisyumMotion,
  type ElisyumResolvedTheme,
  type ElisyumThemeMode,
  type ElisyumTokenOverrides,
  type ElisyumTransparency,
} from '../foundations'

export interface ElisyumProviderProps {
  children: ReactNode
  theme?: ElisyumThemeMode
  defaultTheme?: ElisyumThemeMode
  density?: ElisyumDensity
  motion?: ElisyumMotion
  transparency?: ElisyumTransparency
  contrast?: ElisyumContrast
  locale?: string
  timezone?: string
  tokens?: ElisyumTokenOverrides
  scope?: 'document' | 'local'
  onThemeChange?: (theme: ElisyumThemeMode) => void
}

export interface ElisyumContextValue {
  themeMode: ElisyumThemeMode
  resolvedTheme: ElisyumResolvedTheme
  density: ElisyumDensity
  motion: ElisyumMotion
  transparency: ElisyumTransparency
  contrast: ElisyumContrast
  locale: string
  timezone?: string
  tokens: ElisyumTokenOverrides
  setTheme: (theme: ElisyumThemeMode) => void
}

const ElisyumContext = createContext<ElisyumContextValue | null>(null)

function getSystemTheme(): ElisyumResolvedTheme {
  if (typeof window === 'undefined') {
    return 'dark'
  }

  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function useSystemTheme(): ElisyumResolvedTheme {
  const [systemTheme, setSystemTheme] = useState<ElisyumResolvedTheme>(getSystemTheme)

  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    const updateTheme = (event: MediaQueryListEvent) => {
      setSystemTheme(event.matches ? 'dark' : 'light')
    }

    setSystemTheme(mediaQuery.matches ? 'dark' : 'light')
    mediaQuery.addEventListener('change', updateTheme)

    return () => mediaQuery.removeEventListener('change', updateTheme)
  }, [])

  return systemTheme
}

function useDocumentTheme(
  value: ElisyumContextValue,
  tokenStyles: Record<string, string | number>,
  enabled: boolean,
): void {
  useEffect(() => {
    if (!enabled || typeof document === 'undefined') {
      return
    }

    const root = document.documentElement
    const previousAttributes = new Map<string, string | undefined>()
    const previousStyles = new Map<string, string>()
    const attributes = {
      theme: value.resolvedTheme,
      density: value.density,
      motion: value.motion,
      transparency: value.transparency,
      contrast: value.contrast,
    }

    for (const [name, attributeValue] of Object.entries(attributes)) {
      const attribute = `data-${name}`
      previousAttributes.set(attribute, root.getAttribute(attribute) ?? undefined)
      root.setAttribute(attribute, attributeValue)
    }

    for (const [property, propertyValue] of Object.entries(tokenStyles)) {
      previousStyles.set(property, root.style.getPropertyValue(property))
      root.style.setProperty(property, String(propertyValue))
    }

    return () => {
      for (const [attribute, previousValue] of previousAttributes) {
        if (previousValue === undefined) {
          root.removeAttribute(attribute)
        } else {
          root.setAttribute(attribute, previousValue)
        }
      }

      for (const [property, previousValue] of previousStyles) {
        if (previousValue) {
          root.style.setProperty(property, previousValue)
        } else {
          root.style.removeProperty(property)
        }
      }
    }
  }, [enabled, tokenStyles, value])
}

function mergeScopedValue<T>(
  value: T | undefined,
  inherited: T | undefined,
  fallback: T,
): T {
  return value ?? inherited ?? fallback
}

export function ElisyumProvider({
  children,
  theme,
  defaultTheme,
  density,
  motion,
  transparency,
  contrast,
  locale,
  timezone,
  tokens,
  scope = 'document',
  onThemeChange,
}: ElisyumProviderProps) {
  const parent = useContext(ElisyumContext)
  const inheritedTheme = parent?.themeMode
  const [uncontrolledTheme, setUncontrolledTheme] = useState<ElisyumThemeMode>(
    defaultTheme ?? inheritedTheme ?? 'dark',
  )
  const themeMode = theme ?? uncontrolledTheme
  const systemTheme = useSystemTheme()
  const resolvedTheme = themeMode === 'system' ? systemTheme : themeMode
  const contextValue = useMemo<ElisyumContextValue>(() => {
    const value: ElisyumContextValue = {
      themeMode,
      resolvedTheme,
      density: mergeScopedValue(density, parent?.density, 'standard'),
      motion: mergeScopedValue(motion, parent?.motion, 'system'),
      transparency: mergeScopedValue(transparency, parent?.transparency, 'system'),
      contrast: mergeScopedValue(contrast, parent?.contrast, 'system'),
      locale: mergeScopedValue(locale, parent?.locale, 'en-US'),
      timezone: timezone ?? parent?.timezone,
      tokens: mergeTokenOverrides(parent?.tokens, tokens),
      setTheme: (nextTheme) => {
        if (!theme) {
          setUncontrolledTheme(nextTheme)
        }
        onThemeChange?.(nextTheme)
      },
    }

    return value
  }, [
    contrast,
    density,
    inheritedTheme,
    locale,
    motion,
    onThemeChange,
    parent,
    resolvedTheme,
    theme,
    themeMode,
    timezone,
    tokens,
    transparency,
  ])
  const tokenStyles = useMemo(
    () => tokenOverridesToStyle(contextValue.tokens),
    [contextValue.tokens],
  )

  useDocumentTheme(contextValue, tokenStyles, scope === 'document')

  if (scope === 'local') {
    const localStyles = tokenStyles as CSSProperties

    return (
      <ElisyumContext.Provider value={contextValue}>
        <div
          data-contrast={contextValue.contrast}
          data-density={contextValue.density}
          data-motion={contextValue.motion}
          data-theme={contextValue.resolvedTheme}
          data-transparency={contextValue.transparency}
          style={localStyles}
        >
          {children}
        </div>
      </ElisyumContext.Provider>
    )
  }

  return (
    <ElisyumContext.Provider value={contextValue}>{children}</ElisyumContext.Provider>
  )
}

export function useElisyum(): ElisyumContextValue {
  const context = useContext(ElisyumContext)

  if (!context) {
    throw new Error('useElisyum must be used within an ElisyumProvider')
  }

  return context
}

export const ElisyumConfigProvider = ElisyumProvider

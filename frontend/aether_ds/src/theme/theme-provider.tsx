import {
  type CSSProperties,
  createContext,
  type ReactNode,
  useContext,
  useEffect,
  useRef,
  useState,
} from 'react'

export type Theme = 'light' | 'dark' | 'system'
export type ResolvedTheme = 'light' | 'dark'
export type CSSValue = string | number
export type CustomThemeProperty = `--${string}`

export interface AetherPrimitiveTokens {
  white: CSSValue
  black: CSSValue
  neutral50: CSSValue
  neutral100: CSSValue
  neutral200: CSSValue
  neutral300: CSSValue
  neutral500: CSSValue
  neutral700: CSSValue
  neutral900: CSSValue
  neutral950: CSSValue
  blue200: CSSValue
  blue300: CSSValue
  blue400: CSSValue
  blue500: CSSValue
  blue600: CSSValue
  blue700: CSSValue
  blue800: CSSValue
  blue900: CSSValue
  blue950: CSSValue
  indigo200: CSSValue
  indigo300: CSSValue
  indigo700: CSSValue
  indigo900: CSSValue
  orange200: CSSValue
  orange300: CSSValue
  orange700: CSSValue
  orange900: CSSValue
  red200: CSSValue
  red300: CSSValue
  red700: CSSValue
  red800: CSSValue
  red900: CSSValue
  green300: CSSValue
  green700: CSSValue
  green800: CSSValue
  green900: CSSValue
  yellow300: CSSValue
  yellow700: CSSValue
  alphaWhite05: CSSValue
  alphaWhite08: CSSValue
  alphaWhite12: CSSValue
  alphaBlack08: CSSValue
  alphaBlack35: CSSValue
  alphaBlack50: CSSValue
  alphaBlack60: CSSValue
  alphaBlue12: CSSValue
  alphaBlue20: CSSValue
}

export interface AetherSemanticTokens {
  canvas: CSSValue
  contentPrimary: CSSValue
  contentSecondary: CSSValue
  contentTertiary: CSSValue
  contentDisabled: CSSValue
  contentOnPrimary: CSSValue
  contentOnSecondary: CSSValue
  contentOnTertiary: CSSValue
  contentOnDanger: CSSValue
  contentOnSuccess: CSSValue
  surfaceLowest: CSSValue
  surfaceBase: CSSValue
  surfaceRaised: CSSValue
  surfaceHigh: CSSValue
  surfaceHighest: CSSValue
  surfacePopover: CSSValue
  surfaceModal: CSSValue
  surfaceInverse: CSSValue
  borderSubtle: CSSValue
  borderDefault: CSSValue
  borderStrong: CSSValue
  borderFocus: CSSValue
  actionPrimary: CSSValue
  actionPrimaryHover: CSSValue
  actionPrimaryActive: CSSValue
  actionDanger: CSSValue
  actionDangerHover: CSSValue
  actionDangerActive: CSSValue
  actionSuccess: CSSValue
  actionSuccessHover: CSSValue
  actionSuccessActive: CSSValue
  actionSecondary: CSSValue
  actionSecondaryHover: CSSValue
  actionSecondaryBorder: CSSValue
  actionSecondaryContent: CSSValue
  actionQuietHover: CSSValue
  actionQuietContent: CSSValue
  selectionBackground: CSSValue
  selectionContent: CSSValue
  statusInfo: CSSValue
  statusInfoContainer: CSSValue
  statusInfoContent: CSSValue
  statusSuccess: CSSValue
  statusSuccessContainer: CSSValue
  statusSuccessContent: CSSValue
  statusWarning: CSSValue
  statusWarningContainer: CSSValue
  statusWarningContent: CSSValue
  statusDanger: CSSValue
  statusDangerContainer: CSSValue
  statusDangerContent: CSSValue
  statusDangerContentOnAction: CSSValue
  statusDangerContentOnStatus: CSSValue
  inverseSurface: CSSValue
  inverseContent: CSSValue
  inversePrimary: CSSValue
  surfaceTint: CSSValue
  glassBackground: CSSValue
  glassBorder: CSSValue
}

export interface AetherComponentTokens {
  buttonPrimaryBg: CSSValue
  buttonPrimaryBgHover: CSSValue
  buttonPrimaryBgActive: CSSValue
  buttonPrimaryContent: CSSValue
  buttonSecondaryBg: CSSValue
  buttonSecondaryBgHover: CSSValue
  buttonSecondaryBorder: CSSValue
  buttonSecondaryContent: CSSValue
  buttonQuietBgHover: CSSValue
  buttonQuietContent: CSSValue
  buttonDangerBg: CSSValue
  buttonDangerBgHover: CSSValue
  buttonDangerBgActive: CSSValue
  buttonDangerContent: CSSValue
  buttonSuccessBg: CSSValue
  buttonSuccessBgHover: CSSValue
  buttonSuccessBgActive: CSSValue
  buttonSuccessContent: CSSValue
}

export interface AetherTypographyTokens {
  familySans: CSSValue
  familyMono: CSSValue
  sizeLabel: CSSValue
  sizeBodySm: CSSValue
  sizeBodyMd: CSSValue
  sizeHeadingSm: CSSValue
  sizeDisplayLg: CSSValue
  lineHeightLabel: CSSValue
  lineHeightBodySm: CSSValue
  lineHeightBodyMd: CSSValue
  lineHeightHeadingSm: CSSValue
  lineHeightDisplayLg: CSSValue
  letterSpacingLabel: CSSValue
  letterSpacingHeadingSm: CSSValue
  letterSpacingDisplayLg: CSSValue
  weightLabel: CSSValue
  weightHeadingSm: CSSValue
  weightDisplayLg: CSSValue
}

export interface AetherSpacingTokens {
  one: CSSValue
  two: CSSValue
  four: CSSValue
  six: CSSValue
  ten: CSSValue
}

export interface AetherRadiusTokens {
  control: CSSValue
  card: CSSValue
  overlay: CSSValue
  pill: CSSValue
}

export interface AetherElevationTokens {
  control: CSSValue
  card: CSSValue
  overlay: CSSValue
}

export interface AetherMotionTokens {
  fast: CSSValue
  normal: CSSValue
  emphasis: CSSValue
  easeStandard: CSSValue
}

export interface AetherThemeTokenSet {
  primitives?: Partial<AetherPrimitiveTokens>
  semantics?: Partial<AetherSemanticTokens>
  components?: Partial<AetherComponentTokens>
  typography?: Partial<AetherTypographyTokens>
  spacing?: Partial<AetherSpacingTokens>
  radii?: Partial<AetherRadiusTokens>
  elevation?: Partial<AetherElevationTokens>
  motion?: Partial<AetherMotionTokens>
  customProperties?: Partial<Record<CustomThemeProperty, CSSValue>>
}

export interface AetherThemeConfig extends AetherThemeTokenSet {
  light?: AetherThemeTokenSet
  dark?: AetherThemeTokenSet
}

export interface ThemeProviderProps {
  children: ReactNode
  defaultTheme?: Theme
  storageKey?: string
  persist?: boolean
  config?: AetherThemeConfig
}

const primitiveVariables: Record<keyof AetherPrimitiveTokens, string> = {
  white: '--aether-white',
  black: '--aether-black',
  neutral50: '--aether-neutral-50',
  neutral100: '--aether-neutral-100',
  neutral200: '--aether-neutral-200',
  neutral300: '--aether-neutral-300',
  neutral500: '--aether-neutral-500',
  neutral700: '--aether-neutral-700',
  neutral900: '--aether-neutral-900',
  neutral950: '--aether-neutral-950',
  blue200: '--aether-blue-200',
  blue300: '--aether-blue-300',
  blue400: '--aether-blue-400',
  blue500: '--aether-blue-500',
  blue600: '--aether-blue-600',
  blue700: '--aether-blue-700',
  blue800: '--aether-blue-800',
  blue900: '--aether-blue-900',
  blue950: '--aether-blue-950',
  indigo200: '--aether-indigo-200',
  indigo300: '--aether-indigo-300',
  indigo700: '--aether-indigo-700',
  indigo900: '--aether-indigo-900',
  orange200: '--aether-orange-200',
  orange300: '--aether-orange-300',
  orange700: '--aether-orange-700',
  orange900: '--aether-orange-900',
  red200: '--aether-red-200',
  red300: '--aether-red-300',
  red700: '--aether-red-700',
  red800: '--aether-red-800',
  red900: '--aether-red-900',
  green300: '--aether-green-300',
  green700: '--aether-green-700',
  green800: '--aether-green-800',
  green900: '--aether-green-900',
  yellow300: '--aether-yellow-300',
  yellow700: '--aether-yellow-700',
  alphaWhite05: '--aether-alpha-white-05',
  alphaWhite08: '--aether-alpha-white-08',
  alphaWhite12: '--aether-alpha-white-12',
  alphaBlack08: '--aether-alpha-black-08',
  alphaBlack35: '--aether-alpha-black-35',
  alphaBlack50: '--aether-alpha-black-50',
  alphaBlack60: '--aether-alpha-black-60',
  alphaBlue12: '--aether-alpha-blue-12',
  alphaBlue20: '--aether-alpha-blue-20',
}

const semanticVariables: Record<keyof AetherSemanticTokens, string> = {
  canvas: '--semantic-canvas',
  contentPrimary: '--semantic-content-primary',
  contentSecondary: '--semantic-content-secondary',
  contentTertiary: '--semantic-content-tertiary',
  contentDisabled: '--semantic-content-disabled',
  contentOnPrimary: '--semantic-content-on-primary',
  contentOnSecondary: '--semantic-content-on-secondary',
  contentOnTertiary: '--semantic-content-on-tertiary',
  contentOnDanger: '--semantic-content-on-danger',
  contentOnSuccess: '--semantic-content-on-success',
  surfaceLowest: '--semantic-surface-lowest',
  surfaceBase: '--semantic-surface-base',
  surfaceRaised: '--semantic-surface-raised',
  surfaceHigh: '--semantic-surface-high',
  surfaceHighest: '--semantic-surface-highest',
  surfacePopover: '--semantic-surface-popover',
  surfaceModal: '--semantic-surface-modal',
  surfaceInverse: '--semantic-surface-inverse',
  borderSubtle: '--semantic-border-subtle',
  borderDefault: '--semantic-border-default',
  borderStrong: '--semantic-border-strong',
  borderFocus: '--semantic-border-focus',
  actionPrimary: '--semantic-action-primary',
  actionPrimaryHover: '--semantic-action-primary-hover',
  actionPrimaryActive: '--semantic-action-primary-active',
  actionDanger: '--semantic-action-danger',
  actionDangerHover: '--semantic-action-danger-hover',
  actionDangerActive: '--semantic-action-danger-active',
  actionSuccess: '--semantic-action-success',
  actionSuccessHover: '--semantic-action-success-hover',
  actionSuccessActive: '--semantic-action-success-active',
  actionSecondary: '--semantic-action-secondary',
  actionSecondaryHover: '--semantic-action-secondary-hover',
  actionSecondaryBorder: '--semantic-action-secondary-border',
  actionSecondaryContent: '--semantic-action-secondary-content',
  actionQuietHover: '--semantic-action-quiet-hover',
  actionQuietContent: '--semantic-action-quiet-content',
  selectionBackground: '--semantic-selection-background',
  selectionContent: '--semantic-selection-content',
  statusInfo: '--semantic-status-info',
  statusInfoContainer: '--semantic-status-info-container',
  statusInfoContent: '--semantic-status-info-content',
  statusSuccess: '--semantic-status-success',
  statusSuccessContainer: '--semantic-status-success-container',
  statusSuccessContent: '--semantic-status-success-content',
  statusWarning: '--semantic-status-warning',
  statusWarningContainer: '--semantic-status-warning-container',
  statusWarningContent: '--semantic-status-warning-content',
  statusDanger: '--semantic-status-danger',
  statusDangerContainer: '--semantic-status-danger-container',
  statusDangerContent: '--semantic-status-danger-content',
  statusDangerContentOnAction: '--semantic-status-danger-content-on-action',
  statusDangerContentOnStatus: '--semantic-status-danger-content-on-status',
  inverseSurface: '--semantic-inverse-surface',
  inverseContent: '--semantic-inverse-content',
  inversePrimary: '--semantic-inverse-primary',
  surfaceTint: '--semantic-surface-tint',
  glassBackground: '--semantic-glass-background',
  glassBorder: '--semantic-glass-border',
}

const componentVariables: Record<keyof AetherComponentTokens, string> = {
  buttonPrimaryBg: '--component-button-primary-bg',
  buttonPrimaryBgHover: '--component-button-primary-bg-hover',
  buttonPrimaryBgActive: '--component-button-primary-bg-active',
  buttonPrimaryContent: '--component-button-primary-content',
  buttonSecondaryBg: '--component-button-secondary-bg',
  buttonSecondaryBgHover: '--component-button-secondary-bg-hover',
  buttonSecondaryBorder: '--component-button-secondary-border',
  buttonSecondaryContent: '--component-button-secondary-content',
  buttonQuietBgHover: '--component-button-quiet-bg-hover',
  buttonQuietContent: '--component-button-quiet-content',
  buttonDangerBg: '--component-button-danger-bg',
  buttonDangerBgHover: '--component-button-danger-bg-hover',
  buttonDangerBgActive: '--component-button-danger-bg-active',
  buttonDangerContent: '--component-button-danger-content',
  buttonSuccessBg: '--component-button-success-bg',
  buttonSuccessBgHover: '--component-button-success-bg-hover',
  buttonSuccessBgActive: '--component-button-success-bg-active',
  buttonSuccessContent: '--component-button-success-content',
}

const typographyVariables: Record<keyof AetherTypographyTokens, string> = {
  familySans: '--font-family-sans',
  familyMono: '--font-family-mono',
  sizeLabel: '--font-size-label',
  sizeBodySm: '--font-size-body-sm',
  sizeBodyMd: '--font-size-body-md',
  sizeHeadingSm: '--font-size-heading-sm',
  sizeDisplayLg: '--font-size-display-lg',
  lineHeightLabel: '--line-height-label',
  lineHeightBodySm: '--line-height-body-sm',
  lineHeightBodyMd: '--line-height-body-md',
  lineHeightHeadingSm: '--line-height-heading-sm',
  lineHeightDisplayLg: '--line-height-display-lg',
  letterSpacingLabel: '--letter-spacing-label',
  letterSpacingHeadingSm: '--letter-spacing-heading-sm',
  letterSpacingDisplayLg: '--letter-spacing-display-lg',
  weightLabel: '--font-weight-label',
  weightHeadingSm: '--font-weight-heading-sm',
  weightDisplayLg: '--font-weight-display-lg',
}

const spacingVariables: Record<keyof AetherSpacingTokens, string> = {
  one: '--space-1',
  two: '--space-2',
  four: '--space-4',
  six: '--space-6',
  ten: '--space-10',
}
const radiusVariables: Record<keyof AetherRadiusTokens, string> = {
  control: '--radius-control',
  card: '--radius-card',
  overlay: '--radius-overlay',
  pill: '--radius-pill',
}
const elevationVariables: Record<keyof AetherElevationTokens, string> = {
  control: '--elevation-control',
  card: '--elevation-card',
  overlay: '--elevation-overlay',
}
const motionVariables: Record<keyof AetherMotionTokens, string> = {
  fast: '--motion-fast',
  normal: '--motion-normal',
  emphasis: '--motion-emphasis',
  easeStandard: '--motion-ease-standard',
}

const themeContext = createContext<{
  theme: Theme
  resolvedTheme: ResolvedTheme
  config: AetherThemeConfig
  setTheme: (theme: Theme) => void
} | null>(null)

function variablesFor(set: AetherThemeTokenSet) {
  const variables: Record<string, CSSValue> = {}
  for (const [property, value] of Object.entries(set.customProperties ?? {})) {
    if (value !== undefined) variables[property] = value
  }
  const apply = <T extends object>(
    values: Partial<T> | undefined,
    map: Record<keyof T, string>,
  ) => {
    if (!values) return
    for (const key of Object.keys(values) as Array<keyof T>) {
      const value = values[key]
      if (value !== undefined) variables[map[key] as string] = value as CSSValue
    }
  }
  apply(set.primitives, primitiveVariables)
  apply(set.semantics, semanticVariables)
  apply(set.components, componentVariables)
  apply(set.typography, typographyVariables)
  apply(set.spacing, spacingVariables)
  apply(set.radii, radiusVariables)
  apply(set.elevation, elevationVariables)
  apply(set.motion, motionVariables)
  return variables
}

function mergeTokenSets(
  base: AetherThemeTokenSet,
  override: AetherThemeTokenSet,
): AetherThemeTokenSet {
  return {
    primitives: { ...base.primitives, ...override.primitives },
    semantics: { ...base.semantics, ...override.semantics },
    components: { ...base.components, ...override.components },
    typography: { ...base.typography, ...override.typography },
    spacing: { ...base.spacing, ...override.spacing },
    radii: { ...base.radii, ...override.radii },
    elevation: { ...base.elevation, ...override.elevation },
    motion: { ...base.motion, ...override.motion },
    customProperties: {
      ...base.customProperties,
      ...override.customProperties,
    },
  }
}

export function ThemeProvider({
  children,
  config = {},
  defaultTheme = 'dark',
  persist = true,
  storageKey = 'aether-theme',
}: ThemeProviderProps) {
  const [theme, setTheme] = useState<Theme>(defaultTheme)
  const [resolvedTheme, setResolvedTheme] = useState<ResolvedTheme>('dark')
  const appliedVariables = useRef<string[]>([])

  useEffect(() => {
    const stored = persist
      ? (window.localStorage.getItem(storageKey) as Theme | null)
      : null
    if (stored === 'light' || stored === 'dark' || stored === 'system')
      setTheme(stored)
  }, [persist, storageKey])

  useEffect(() => {
    const media = window.matchMedia('(prefers-color-scheme: dark)')
    const resolve = () => {
      const nextTheme =
        theme === 'system' ? (media.matches ? 'dark' : 'light') : theme
      setResolvedTheme(nextTheme)
      document.documentElement.classList.toggle('dark', nextTheme === 'dark')
      document.documentElement.style.colorScheme = nextTheme
    }
    resolve()
    media.addEventListener('change', resolve)
    return () => media.removeEventListener('change', resolve)
  }, [theme])

  useEffect(() => {
    const variables = variablesFor(
      mergeTokenSets(config, config[resolvedTheme] ?? {}),
    )
    const root = document.documentElement
    for (const property of appliedVariables.current)
      root.style.removeProperty(property)
    for (const [property, value] of Object.entries(variables))
      root.style.setProperty(property, String(value))
    appliedVariables.current = Object.keys(variables)
    return () => {
      for (const property of appliedVariables.current)
        root.style.removeProperty(property)
      appliedVariables.current = []
    }
  }, [config, resolvedTheme])

  const updateTheme = (nextTheme: Theme) => {
    setTheme(nextTheme)
    if (persist) window.localStorage.setItem(storageKey, nextTheme)
  }

  return (
    <themeContext.Provider
      value={{ theme, resolvedTheme, config, setTheme: updateTheme }}
    >
      {children}
    </themeContext.Provider>
  )
}

export function useTheme() {
  const context = useContext(themeContext)
  if (!context) throw new Error('useTheme must be used inside ThemeProvider')
  return context
}

export type ThemeStyle = CSSProperties

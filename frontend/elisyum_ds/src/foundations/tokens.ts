export const themeModes = ['dark', 'light', 'system'] as const
export const densityModes = [
  'focused',
  'standard',
  'dense',
  'stream',
  'telemetry',
] as const
export const motionModes = ['full', 'reduced', 'system'] as const
export const transparencyModes = ['full', 'reduced', 'system'] as const
export const contrastModes = ['standard', 'more', 'system'] as const

export type ElisyumThemeMode = (typeof themeModes)[number]
export type ElisyumResolvedTheme = Exclude<ElisyumThemeMode, 'system'>
export type ElisyumDensity = (typeof densityModes)[number]
export type ElisyumMotion = (typeof motionModes)[number]
export type ElisyumTransparency = (typeof transparencyModes)[number]
export type ElisyumContrast = (typeof contrastModes)[number]
export type ElisyumTokenValue = string | number

export type ElisyumColorToken =
  | 'canvas'
  | 'surface0'
  | 'surface1'
  | 'surface2'
  | 'surface3'
  | 'surfaceInteractive'
  | 'fieldHover'
  | 'textStrong'
  | 'text'
  | 'textMuted'
  | 'textSubtle'
  | 'outline'
  | 'outlineSubtle'
  | 'accent'
  | 'accentStrong'
  | 'accentSurface'
  | 'focus'
  | 'selection'
  | 'info'
  | 'success'
  | 'warning'
  | 'danger'
  | 'neutralState'
  | 'codeCanvas'
  | 'scrim'
  | 'action'
  | 'actionStrong'
  | 'field'
  | 'overlay'
  | 'criticalOverlay'
  | 'borderSubtle'
  | 'borderDefault'
  | 'borderStrong'
  | 'borderDanger'

export type ElisyumColorTokenOverrides = Partial<
  Record<ElisyumColorToken, ElisyumTokenValue>
>

export interface ElisyumTypographyTokenOverrides {
  interfaceFont?: ElisyumTokenValue
  technicalFont?: ElisyumTokenValue
  displaySize?: ElisyumTokenValue
  displayLineHeight?: ElisyumTokenValue
  displayWeight?: ElisyumTokenValue
  displayTracking?: ElisyumTokenValue
  pageTitleSize?: ElisyumTokenValue
  pageTitleLineHeight?: ElisyumTokenValue
  pageTitleWeight?: ElisyumTokenValue
  pageTitleTracking?: ElisyumTokenValue
  sectionTitleSize?: ElisyumTokenValue
  sectionTitleLineHeight?: ElisyumTokenValue
  sectionTitleWeight?: ElisyumTokenValue
  bodySize?: ElisyumTokenValue
  bodyLineHeight?: ElisyumTokenValue
  bodyWeight?: ElisyumTokenValue
  supportingSize?: ElisyumTokenValue
  supportingLineHeight?: ElisyumTokenValue
  labelSize?: ElisyumTokenValue
  labelLineHeight?: ElisyumTokenValue
  labelWeight?: ElisyumTokenValue
  labelTracking?: ElisyumTokenValue
  metricSize?: ElisyumTokenValue
  metricLineHeight?: ElisyumTokenValue
  metricWeight?: ElisyumTokenValue
  codeSize?: ElisyumTokenValue
  codeLineHeight?: ElisyumTokenValue
  logSize?: ElisyumTokenValue
  logLineHeight?: ElisyumTokenValue
}

export interface ElisyumSpacingTokenOverrides {
  space0?: ElisyumTokenValue
  space1?: ElisyumTokenValue
  space2?: ElisyumTokenValue
  space3?: ElisyumTokenValue
  space4?: ElisyumTokenValue
  space5?: ElisyumTokenValue
  space6?: ElisyumTokenValue
  space8?: ElisyumTokenValue
  space10?: ElisyumTokenValue
  space12?: ElisyumTokenValue
  space16?: ElisyumTokenValue
}

export interface ElisyumRadiusTokenOverrides {
  none?: ElisyumTokenValue
  sm?: ElisyumTokenValue
  md?: ElisyumTokenValue
  lg?: ElisyumTokenValue
  xl?: ElisyumTokenValue
  pill?: ElisyumTokenValue
}

export interface ElisyumElevationTokenOverrides {
  elevation0?: ElisyumTokenValue
  elevation1?: ElisyumTokenValue
  elevation2?: ElisyumTokenValue
  elevation3?: ElisyumTokenValue
  elevation4?: ElisyumTokenValue
}

export interface ElisyumMotionTokenOverrides {
  instant?: ElisyumTokenValue
  fast?: ElisyumTokenValue
  standard?: ElisyumTokenValue
  popover?: ElisyumTokenValue
  overlay?: ElisyumTokenValue
  pressScale?: ElisyumTokenValue
  hoverLift?: ElisyumTokenValue
  easeOut?: ElisyumTokenValue
  easeInOut?: ElisyumTokenValue
  easeDrawer?: ElisyumTokenValue
}

export interface ElisyumIconTokenOverrides {
  dense?: ElisyumTokenValue
  standard?: ElisyumTokenValue
  navigation?: ElisyumTokenValue
  identity?: ElisyumTokenValue
}

export interface ElisyumLayerTokenOverrides {
  base?: ElisyumTokenValue
  sticky?: ElisyumTokenValue
  navigation?: ElisyumTokenValue
  popover?: ElisyumTokenValue
  dialog?: ElisyumTokenValue
  toast?: ElisyumTokenValue
}

export interface ElisyumTokenOverrides {
  colors?: ElisyumColorTokenOverrides
  typography?: ElisyumTypographyTokenOverrides
  spacing?: ElisyumSpacingTokenOverrides
  radius?: ElisyumRadiusTokenOverrides
  elevation?: ElisyumElevationTokenOverrides
  motion?: ElisyumMotionTokenOverrides
  icons?: ElisyumIconTokenOverrides
  layers?: ElisyumLayerTokenOverrides
  custom?: Record<`--ely-${string}`, ElisyumTokenValue>
}

const tokenVariableMap: Record<string, string> = {
  'colors.canvas': '--ely-color-canvas',
  'colors.surface0': '--ely-color-surface-0',
  'colors.surface1': '--ely-color-surface-1',
  'colors.surface2': '--ely-color-surface-2',
  'colors.surface3': '--ely-color-surface-3',
  'colors.surfaceInteractive': '--ely-color-surface-interactive',
  'colors.fieldHover': '--ely-color-field-hover',
  'colors.textStrong': '--ely-color-text-strong',
  'colors.text': '--ely-color-text',
  'colors.textMuted': '--ely-color-text-muted',
  'colors.textSubtle': '--ely-color-text-subtle',
  'colors.outline': '--ely-color-outline',
  'colors.outlineSubtle': '--ely-color-outline-subtle',
  'colors.accent': '--ely-color-accent',
  'colors.accentStrong': '--ely-color-accent-strong',
  'colors.accentSurface': '--ely-color-accent-surface',
  'colors.focus': '--ely-color-focus',
  'colors.selection': '--ely-color-selection',
  'colors.info': '--ely-color-info',
  'colors.success': '--ely-color-success',
  'colors.warning': '--ely-color-warning',
  'colors.danger': '--ely-color-danger',
  'colors.neutralState': '--ely-color-neutral-state',
  'colors.codeCanvas': '--ely-color-code-canvas',
  'colors.scrim': '--ely-color-scrim',
  'colors.action': '--ely-color-action',
  'colors.actionStrong': '--ely-color-action-strong',
  'colors.field': '--ely-color-field',
  'colors.overlay': '--ely-color-overlay',
  'colors.criticalOverlay': '--ely-color-critical-overlay',
  'colors.borderSubtle': '--ely-color-border-subtle',
  'colors.borderDefault': '--ely-color-border-default',
  'colors.borderStrong': '--ely-color-border-strong',
  'colors.borderDanger': '--ely-color-border-danger',
  'typography.interfaceFont': '--ely-font-interface',
  'typography.technicalFont': '--ely-font-technical',
  'typography.displaySize': '--ely-text-display-size',
  'typography.displayLineHeight': '--ely-text-display-line-height',
  'typography.displayWeight': '--ely-text-display-weight',
  'typography.displayTracking': '--ely-text-display-tracking',
  'typography.pageTitleSize': '--ely-text-page-title-size',
  'typography.pageTitleLineHeight': '--ely-text-page-title-line-height',
  'typography.pageTitleWeight': '--ely-text-page-title-weight',
  'typography.pageTitleTracking': '--ely-text-page-title-tracking',
  'typography.sectionTitleSize': '--ely-text-section-title-size',
  'typography.sectionTitleLineHeight': '--ely-text-section-title-line-height',
  'typography.sectionTitleWeight': '--ely-text-section-title-weight',
  'typography.bodySize': '--ely-text-body-size',
  'typography.bodyLineHeight': '--ely-text-body-line-height',
  'typography.bodyWeight': '--ely-text-body-weight',
  'typography.supportingSize': '--ely-text-supporting-size',
  'typography.supportingLineHeight': '--ely-text-supporting-line-height',
  'typography.labelSize': '--ely-text-label-size',
  'typography.labelLineHeight': '--ely-text-label-line-height',
  'typography.labelWeight': '--ely-text-label-weight',
  'typography.labelTracking': '--ely-text-label-tracking',
  'typography.metricSize': '--ely-text-metric-size',
  'typography.metricLineHeight': '--ely-text-metric-line-height',
  'typography.metricWeight': '--ely-text-metric-weight',
  'typography.codeSize': '--ely-text-code-size',
  'typography.codeLineHeight': '--ely-text-code-line-height',
  'typography.logSize': '--ely-text-log-size',
  'typography.logLineHeight': '--ely-text-log-line-height',
  'spacing.space0': '--ely-space-0',
  'spacing.space1': '--ely-space-1',
  'spacing.space2': '--ely-space-2',
  'spacing.space3': '--ely-space-3',
  'spacing.space4': '--ely-space-4',
  'spacing.space5': '--ely-space-5',
  'spacing.space6': '--ely-space-6',
  'spacing.space8': '--ely-space-8',
  'spacing.space10': '--ely-space-10',
  'spacing.space12': '--ely-space-12',
  'spacing.space16': '--ely-space-16',
  'radius.none': '--ely-radius-none',
  'radius.sm': '--ely-radius-sm',
  'radius.md': '--ely-radius-md',
  'radius.lg': '--ely-radius-lg',
  'radius.xl': '--ely-radius-xl',
  'radius.pill': '--ely-radius-pill',
  'elevation.elevation0': '--ely-elevation-0',
  'elevation.elevation1': '--ely-elevation-1',
  'elevation.elevation2': '--ely-elevation-2',
  'elevation.elevation3': '--ely-elevation-3',
  'elevation.elevation4': '--ely-elevation-4',
  'motion.instant': '--ely-duration-instant',
  'motion.fast': '--ely-duration-fast',
  'motion.standard': '--ely-duration-standard',
  'motion.popover': '--ely-duration-popover',
  'motion.overlay': '--ely-duration-overlay',
  'motion.pressScale': '--ely-motion-press-scale',
  'motion.hoverLift': '--ely-motion-hover-lift',
  'motion.easeOut': '--ely-ease-out',
  'motion.easeInOut': '--ely-ease-in-out',
  'motion.easeDrawer': '--ely-ease-drawer',
  'icons.dense': '--ely-icon-size-dense',
  'icons.standard': '--ely-icon-size-standard',
  'icons.navigation': '--ely-icon-size-navigation',
  'icons.identity': '--ely-icon-size-identity',
  'layers.base': '--ely-z-base',
  'layers.sticky': '--ely-z-sticky',
  'layers.navigation': '--ely-z-navigation',
  'layers.popover': '--ely-z-popover',
  'layers.dialog': '--ely-z-dialog',
  'layers.toast': '--ely-z-toast',
}

const tokenGroups: Array<keyof Omit<ElisyumTokenOverrides, 'custom'>> = [
  'colors',
  'typography',
  'spacing',
  'radius',
  'elevation',
  'motion',
  'icons',
  'layers',
]

export function tokenOverridesToStyle(
  overrides: ElisyumTokenOverrides | undefined,
): Record<string, ElisyumTokenValue> {
  const style: Record<string, ElisyumTokenValue> = {}

  if (!overrides) {
    return style
  }

  for (const group of tokenGroups) {
    const values = overrides[group]

    if (!values) {
      continue
    }

    for (const [name, value] of Object.entries(values)) {
      const variable = tokenVariableMap[`${group}.${name}`]

      if (variable && value !== undefined) {
        style[variable] = value
      }
    }
  }

  for (const [variable, value] of Object.entries(overrides.custom ?? {})) {
    style[variable] = value
  }

  return style
}

export function mergeTokenOverrides(
  parent: ElisyumTokenOverrides | undefined,
  child: ElisyumTokenOverrides | undefined,
): ElisyumTokenOverrides {
  return {
    colors: { ...parent?.colors, ...child?.colors },
    typography: { ...parent?.typography, ...child?.typography },
    spacing: { ...parent?.spacing, ...child?.spacing },
    radius: { ...parent?.radius, ...child?.radius },
    elevation: { ...parent?.elevation, ...child?.elevation },
    motion: { ...parent?.motion, ...child?.motion },
    icons: { ...parent?.icons, ...child?.icons },
    layers: { ...parent?.layers, ...child?.layers },
    custom: { ...parent?.custom, ...child?.custom },
  }
}

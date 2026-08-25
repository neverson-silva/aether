import { Checkbox as BaseCheckbox } from '@base-ui/react/checkbox'
import { Check } from '@phosphor-icons/react'
import { Field as BaseField } from '@base-ui/react/field'
import type { ReactNode } from 'react'
export interface CheckboxProps {
  label?: string
  description?: string
  error?: string
  checked?: boolean
  defaultChecked?: boolean
  indeterminate?: boolean
  disabled?: boolean
  onCheckedChange?: (checked: boolean | 'indeterminate') => void
  children?: ReactNode
}
export function Checkbox({
  checked,
  defaultChecked,
  description,
  disabled,
  error,
  indeterminate,
  label,
  onCheckedChange,
}: CheckboxProps) {
  const control = (
    <BaseCheckbox.Root
      checked={checked}
      indeterminate={indeterminate}
      defaultChecked={defaultChecked}
      disabled={disabled}
      onCheckedChange={(next) => onCheckedChange?.(next)}
      style={{
        width: '1.5rem',
        minWidth: '1.5rem',
        height: '1.5rem',
        minHeight: '1.5rem',
        aspectRatio: '1',
        padding: 0,
        lineHeight: 1,
      }}
      className="inline-flex shrink-0 box-border items-center justify-center rounded border-2 border-outline bg-surface-container p-0 text-primary transition-colors data-[checked]:border-primary data-[checked]:bg-primary data-[checked]:text-primary-foreground data-[indeterminate]:border-primary data-[indeterminate]:bg-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
    >
      <BaseCheckbox.Indicator className="flex size-full items-center justify-center">
        <Check size={16} weight="bold" aria-hidden="true" />
      </BaseCheckbox.Indicator>
    </BaseCheckbox.Root>
  )
  return label ? (
    <BaseField.Root className="group flex flex-wrap items-center gap-x-3 gap-y-2" disabled={disabled} invalid={Boolean(error)}>
      {control}
      <BaseField.Label className="text-body-sm font-semibold text-foreground">{label}</BaseField.Label>
      {description ? <BaseField.Description className="basis-full pl-9 text-body-sm text-muted-foreground">{description}</BaseField.Description> : null}
      {error ? <p className="basis-full pl-9 text-body-sm text-status-danger" role="alert">{error}</p> : null}
    </BaseField.Root>
  ) : (
    control
  )
}

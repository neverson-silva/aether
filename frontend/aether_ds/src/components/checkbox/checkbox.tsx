import { Checkbox as BaseCheckbox } from '@base-ui/react/checkbox'
import type { ReactNode } from 'react'
import { Field } from '../field/field'
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
      className="inline-flex size-5 items-center justify-center rounded border border-border bg-surface-card text-primary transition-colors data-[checked]:border-primary data-[checked]:bg-primary data-[checked]:text-primary-foreground data-[indeterminate]:border-primary data-[indeterminate]:bg-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
    >
      <BaseCheckbox.Indicator className="text-xs">✓</BaseCheckbox.Indicator>
    </BaseCheckbox.Root>
  )
  return label ? (
    <Field
      label={label}
      description={description}
      error={error}
      disabled={disabled}
    >
      {control}
    </Field>
  ) : (
    control
  )
}

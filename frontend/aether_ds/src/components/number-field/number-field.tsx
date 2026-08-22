import { NumberField as BaseNumberField } from '@base-ui/react/number-field'
import { Field } from '../field/field'
export interface NumberFieldProps {
  label?: string
  description?: string
  error?: string
  value?: number
  defaultValue?: number
  min?: number
  max?: number
  step?: number
  onValueChange?: (value: number | null) => void
}
export function NumberField({
  description,
  error,
  label,
  ...props
}: NumberFieldProps) {
  const control = (
    <BaseNumberField.Root {...props}>
      <BaseNumberField.Group className="flex h-10 overflow-hidden rounded-md border border-border bg-surface-card">
        <BaseNumberField.Decrement className="w-10 text-muted-foreground hover:bg-surface-container">
          −
        </BaseNumberField.Decrement>
        <BaseNumberField.Input className="min-w-0 flex-1 bg-transparent text-center outline-none" />
        <BaseNumberField.Increment className="w-10 text-muted-foreground hover:bg-surface-container">
          +
        </BaseNumberField.Increment>
      </BaseNumberField.Group>
    </BaseNumberField.Root>
  )
  return label ? (
    <Field label={label} description={description} error={error}>
      {control}
    </Field>
  ) : (
    control
  )
}

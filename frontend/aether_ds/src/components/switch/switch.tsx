import { Switch as BaseSwitch } from '@base-ui/react/switch'
import { Field } from '../field/field'
export interface SwitchProps {
  label?: string
  description?: string
  checked?: boolean
  defaultChecked?: boolean
  disabled?: boolean
  loading?: boolean
  onCheckedChange?: (checked: boolean) => void
}
export function Switch({
  checked,
  defaultChecked,
  description,
  disabled,
  label,
  loading,
  onCheckedChange,
}: SwitchProps) {
  const control = (
    <BaseSwitch.Root
      checked={checked}
      defaultChecked={defaultChecked}
      disabled={disabled || loading}
      onCheckedChange={onCheckedChange}
      className="relative inline-flex h-6 w-10 shrink-0 rounded-full bg-surface-container transition-colors data-[checked]:bg-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
    >
      <BaseSwitch.Thumb className="m-0.5 size-5 rounded-full bg-white shadow-sm transition-transform data-[checked]:translate-x-4" />
      {loading ? (
        <span
          className="absolute inset-0 m-auto size-3 animate-spin rounded-full border border-current border-t-transparent"
          aria-hidden="true"
        />
      ) : null}
    </BaseSwitch.Root>
  )
  return label ? (
    <Field label={label} description={description} disabled={disabled}>
      {control}
    </Field>
  ) : (
    control
  )
}

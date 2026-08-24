import { Switch as BaseSwitch } from '@base-ui/react/switch'
import { Field } from '../field/field'
export interface SwitchProps {
  ariaLabel?: string
  label?: string
  description?: string
  checked?: boolean
  defaultChecked?: boolean
  disabled?: boolean
  loading?: boolean
  className?: string
  onCheckedChange?: (checked: boolean) => void
}
export function Switch({
  ariaLabel,
  checked,
  className = '',
  defaultChecked,
  description,
  disabled,
  label,
  loading,
  onCheckedChange,
}: SwitchProps) {
  const control = (
    <BaseSwitch.Root
      aria-label={ariaLabel}
      checked={checked}
      defaultChecked={defaultChecked}
      disabled={disabled || loading}
      onCheckedChange={onCheckedChange}
      className={`aether-switch group relative inline-flex h-6 w-10 shrink-0 cursor-pointer rounded-full bg-surface-container transition-colors data-[checked]:bg-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 ${className}`}
    >
      <BaseSwitch.Thumb
        className="aether-switch-thumb m-0.5 size-5 rounded-full shadow-sm transition-transform group-data-[checked]:translate-x-4"
        style={
          checked === undefined
            ? undefined
            : {
                transform: checked ? "translateX(1rem)" : "translateX(0)",
                transition:
                  "transform var(--motion-emphasis) var(--motion-ease-standard)",
              }
        }
      />
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

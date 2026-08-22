import { Radio } from '@base-ui/react/radio'
import { RadioGroup as BaseRadioGroup } from '@base-ui/react/radio-group'
import type { ReactNode } from 'react'
import { Field } from '../field/field'
export interface RadioOption {
  value: string
  label: ReactNode
  description?: ReactNode
  disabled?: boolean
}
export interface RadioGroupProps {
  label?: string
  description?: string
  error?: string
  value?: string
  defaultValue?: string
  options: RadioOption[]
  orientation?: 'horizontal' | 'vertical'
  onValueChange?: (value: string) => void
}
export function RadioGroup({
  description,
  error,
  label,
  onValueChange,
  options,
  orientation = 'vertical',
  value,
  defaultValue,
}: RadioGroupProps) {
  const control = (
    <BaseRadioGroup
      value={value}
      defaultValue={defaultValue}
      onValueChange={onValueChange}
      className={`flex ${orientation === 'horizontal' ? 'flex-row gap-4' : 'flex-col gap-3'}`}
    >
      {options.map((option) => (
        <label
          key={option.value}
          htmlFor={`radio-${option.value}`}
          className={`flex items-start gap-2 ${option.disabled ? 'opacity-50' : ''}`}
        >
          <Radio.Root
            id={`radio-${option.value}`}
            value={option.value}
            disabled={option.disabled}
            className="mt-0.5 flex size-5 items-center justify-center rounded-full border border-border data-[checked]:border-primary"
          >
            <Radio.Indicator className="size-2.5 rounded-full bg-primary" />
          </Radio.Root>
          <span>
            <span className="block text-body-md text-foreground">
              {option.label}
            </span>
            {option.description ? (
              <span className="block text-body-sm text-muted-foreground">
                {option.description}
              </span>
            ) : null}
          </span>
        </label>
      ))}
    </BaseRadioGroup>
  )
  return label ? (
    <Field label={label} description={description} error={error}>
      {control}
    </Field>
  ) : (
    control
  )
}

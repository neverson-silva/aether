import { Combobox } from '../combobox/combobox'
import type { SelectProps } from '../select/select'
export interface SelectSearchProps
  extends Omit<
    SelectProps,
    'onValueChange' | 'options' | 'value' | 'defaultValue'
  > {
  options: SelectProps['options']
  value?: string | null
  defaultValue?: string | null
  onValueChange?: (value: string | null) => void
  recentValues?: string[]
}
export function SelectSearch({
  onValueChange,
  options,
  recentValues,
  ...props
}: SelectSearchProps) {
  return (
    <Combobox
      {...props}
      options={options}
      onValueChange={(value) =>
        onValueChange?.(Array.isArray(value) ? (value[0] ?? null) : value)
      }
    />
  )
}

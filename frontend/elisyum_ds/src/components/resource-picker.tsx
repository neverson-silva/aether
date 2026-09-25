import type { ReactNode } from 'react'
import { Combobox, type ComboboxOption } from './combobox'

export interface ResourcePickerProps {
  options: ComboboxOption[]
  value?: string
  onValueChange?: (value: string) => void
  label?: ReactNode
  className?: string
}

export function ResourcePicker({
  className = '',
  label,
  onValueChange,
  options,
  value,
}: ResourcePickerProps) {
  return (
    <label className={`grid gap-2 ${className}`}>
      {label ? <span className="text-label text-text-secondary">{label}</span> : null}
      <Combobox
        aria-label={label ? String(label) : 'Resource'}
        onChange={(event) => onValueChange?.(event.target.value)}
        options={options}
        value={value}
      />
    </label>
  )
}

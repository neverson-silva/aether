import type { ReactNode } from 'react'
import { Select } from './select'

export interface EnvironmentOption {
  value: string
  label: ReactNode
}
export interface EnvironmentSwitcherProps {
  options: EnvironmentOption[]
  value?: string
  onValueChange?: (value: string) => void
  className?: string
}

export function EnvironmentSwitcher({
  className = '',
  onValueChange,
  options,
  value,
}: EnvironmentSwitcherProps) {
  return (
    <Select
      aria-label="Environment"
      className={`min-h-9 max-w-48 ${className}`}
      onChange={(event) => onValueChange?.(event.target.value)}
      value={value}
    >
      <option value="">Choose environment</option>
      {options.map((option) => (
        <option
          key={option.value}
          value={option.value}
        >
          {option.label}
        </option>
      ))}
    </Select>
  )
}

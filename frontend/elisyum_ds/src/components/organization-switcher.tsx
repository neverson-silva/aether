import type { ReactNode } from 'react'
import { Select } from './select'

export interface OrganizationOption {
  value: string
  label: ReactNode
}
export interface OrganizationSwitcherProps {
  options: OrganizationOption[]
  value?: string
  onValueChange?: (value: string) => void
  className?: string
}

export function OrganizationSwitcher({
  className = '',
  onValueChange,
  options,
  value,
}: OrganizationSwitcherProps) {
  return (
    <Select
      aria-label="Organization"
      className={`min-h-9 max-w-56 ${className}`}
      onChange={(event) => onValueChange?.(event.target.value)}
      value={value}
    >
      <option value="">Choose organization</option>
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

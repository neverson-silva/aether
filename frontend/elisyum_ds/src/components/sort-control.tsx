import { CaretDown, CaretUp, SortAscending } from '@phosphor-icons/react'
import { IconButton } from './icon-button'

export type SortDirection = 'asc' | 'desc' | null
export interface SortControlProps {
  label: string
  direction?: SortDirection
  onDirectionChange: (direction: SortDirection) => void
}

export function SortControl({
  direction = null,
  label,
  onDirectionChange,
}: SortControlProps) {
  const next = direction === null ? 'asc' : direction === 'asc' ? 'desc' : null
  const icon =
    direction === 'asc' ? (
      <CaretUp size={15} />
    ) : direction === 'desc' ? (
      <CaretDown size={15} />
    ) : (
      <SortAscending size={15} />
    )
  return (
    <IconButton
      label={`Sort by ${label}`}
      onClick={() => onDirectionChange(next)}
    >
      {icon}
    </IconButton>
  )
}

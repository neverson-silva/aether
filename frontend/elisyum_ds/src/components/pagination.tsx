import { CaretLeft, CaretRight } from '@phosphor-icons/react'
import { IconButton } from './icon-button'

export interface PaginationProps {
  page: number
  pageCount: number
  onPageChange: (page: number) => void
  className?: string
}

export function Pagination({
  className = '',
  onPageChange,
  page,
  pageCount,
}: PaginationProps) {
  return (
    <nav
      aria-label="Pagination"
      className={`flex items-center gap-2 ${className}`}
    >
      <IconButton
        disabled={page <= 1}
        label="Previous page"
        onClick={() => onPageChange(page - 1)}
      >
        <CaretLeft
          aria-hidden="true"
          size={16}
        />
      </IconButton>
      <span
        aria-live="polite"
        className="min-w-20 text-center font-technical text-log text-text-secondary"
      >
        {String(page).padStart(2, '0')} / {String(pageCount).padStart(2, '0')}
      </span>
      <IconButton
        disabled={page >= pageCount}
        label="Next page"
        onClick={() => onPageChange(page + 1)}
      >
        <CaretRight
          aria-hidden="true"
          size={16}
        />
      </IconButton>
    </nav>
  )
}

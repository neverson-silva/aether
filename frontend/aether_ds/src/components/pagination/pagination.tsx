import { Button } from '../button/button'
export interface PaginationProps {
  page: number
  pageCount: number
  pageSize?: number
  loading?: boolean
  onPageChange: (page: number) => void
  onPageSizeChange?: (size: number) => void
  pageSizes?: number[]
}
export function Pagination({
  loading,
  onPageChange,
  onPageSizeChange,
  page,
  pageCount,
  pageSize,
  pageSizes = [10, 25, 50],
}: PaginationProps) {
  return (
    <nav
      aria-label="Pagination"
      className="flex flex-wrap items-center justify-between gap-3"
    >
      <div className="flex items-center gap-2 text-body-sm text-muted-foreground">
        <Button
          size="sm"
          variant="quiet"
          disabled={loading || page <= 1}
          onClick={() => onPageChange(page - 1)}
        >
          Previous
        </Button>
        <span>
          Page {page} of {pageCount}
        </span>
        <Button
          size="sm"
          variant="quiet"
          disabled={loading || page >= pageCount}
          onClick={() => onPageChange(page + 1)}
        >
          Next
        </Button>
      </div>
      {pageSize && onPageSizeChange ? (
        <label className="flex items-center gap-2 text-body-sm text-muted-foreground">
          Rows
          <select
            value={pageSize}
            onChange={(event) => onPageSizeChange(Number(event.target.value))}
            className="rounded border border-border bg-surface-card px-2 py-1"
          >
            {pageSizes.map((size) => (
              <option key={size} value={size}>
                {size}
              </option>
            ))}
          </select>
        </label>
      ) : null}
    </nav>
  )
}

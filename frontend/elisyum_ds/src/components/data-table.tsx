import type { ReactNode } from 'react'
import { Table } from './table'

export interface DataTableColumn<T> {
  key: keyof T & string
  label: ReactNode
  render?: (value: T[keyof T], row: T) => ReactNode
}
export interface DataTableProps<T extends object> {
  columns: DataTableColumn<T>[]
  rows: T[]
  empty?: ReactNode
  caption?: ReactNode
  className?: string
}

export function DataTable<T extends object>({
  caption,
  className = '',
  columns,
  empty = 'No data available.',
  rows,
}: DataTableProps<T>) {
  return rows.length ? (
    <Table
      caption={caption}
      className={className}
      headers={columns.map((column) => column.label)}
      rows={rows.map((row) =>
        columns.map((column) =>
          column.render
            ? column.render(row[column.key], row)
            : String(row[column.key] ?? ''),
        ),
      )}
    />
  ) : (
    <div
      aria-live="polite"
      className={`rounded-lg border border-border-subtle bg-surface-1 px-4 py-10 text-center text-supporting text-text-tertiary ${className}`}
    >
      {empty}
    </div>
  )
}

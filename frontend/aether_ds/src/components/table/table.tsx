import type { HTMLAttributes, ReactNode } from 'react'
export interface TableColumn<T> {
  key: keyof T & string
  header: ReactNode
  align?: 'start' | 'center' | 'end'
}
export interface TableProps<T extends Record<string, unknown>>
  extends HTMLAttributes<HTMLTableElement> {
  columns: TableColumn<T>[]
  data: T[]
  rowKey?: keyof T & string
  empty?: ReactNode
  loading?: boolean
  error?: ReactNode
  onRowClick?: (row: T) => void
}
export function Table<T extends Record<string, unknown>>({
  className = '',
  columns,
  data,
  empty = 'No records found.',
  error,
  loading,
  onRowClick,
  rowKey = 'id' as keyof T & string,
  ...props
}: TableProps<T>) {
  return (
    <div className="w-full overflow-x-auto rounded-lg border border-border">
      <table
        className={`w-full border-collapse text-body-sm ${className}`}
        {...props}
      >
        <thead className="bg-surface-container text-label-caps text-muted-foreground">
          <tr>
            {columns.map((column) => (
              <th
                key={column.key}
                scope="col"
                className={`whitespace-nowrap px-4 py-3 text-${column.align ?? 'start'}`}
              >
                {column.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {loading ? (
            <tr>
              <td
                colSpan={columns.length}
                className="px-4 py-8 text-center text-muted-foreground"
              >
                Loading records...
              </td>
            </tr>
          ) : error ? (
            <tr>
              <td
                colSpan={columns.length}
                className="px-4 py-8 text-center text-status-danger"
              >
                {error}
              </td>
            </tr>
          ) : data.length ? (
            data.map((row, index) => (
              <tr
                key={String(row[rowKey] ?? index)}
                className={
                  onRowClick
                    ? 'cursor-pointer transition-colors hover:bg-surface-container'
                    : ''
                }
                onClick={() => onRowClick?.(row)}
              >
                {columns.map((column) => (
                  <td
                    key={column.key}
                    className={`whitespace-nowrap px-4 py-3 text-${column.align ?? 'start'} text-foreground`}
                  >
                    {String(row[column.key] ?? '—')}
                  </td>
                ))}
              </tr>
            ))
          ) : (
            <tr>
              <td
                colSpan={columns.length}
                className="px-4 py-8 text-center text-muted-foreground"
              >
                {empty}
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  )
}

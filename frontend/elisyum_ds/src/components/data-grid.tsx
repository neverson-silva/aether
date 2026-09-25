import type { ReactNode } from 'react'
import { DataTable, type DataTableColumn } from './data-table'

export interface DataGridProps<T extends object> {
  columns: DataTableColumn<T>[]
  rows: T[]
  caption?: ReactNode
  className?: string
}

export function DataGrid<T extends object>({
  caption,
  className = '',
  columns,
  rows,
}: DataGridProps<T>) {
  return (
    <div
      className="grid gap-2"
      role="region"
      aria-label={caption ? String(caption) : 'Data grid'}
    >
      <DataTable
        caption={caption}
        className={className}
        columns={columns}
        rows={rows}
      />
    </div>
  )
}

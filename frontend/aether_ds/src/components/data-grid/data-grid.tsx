import { DataTable, type DataTableColumn } from '../data-table/data-table'
export interface DataGridProps<T> {
  columns: DataTableColumn<T>[]
  data: T[]
  rowId: (row: T) => string
  selectedIds?: string[]
  onSelectionChange?: (ids: string[]) => void
  loading?: boolean
}
export function DataGrid<T>(props: DataGridProps<T>) {
  return (
    <div className="min-w-[720px]">
      <DataTable {...props} selectable />
    </div>
  )
}

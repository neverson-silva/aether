import type { ReactNode } from 'react'
import { DataTable, type DataTableColumn } from './data-table'

export interface AuditLogEntry {
  id: string
  actor: string
  action: ReactNode
  target: ReactNode
  timestamp: ReactNode
}
export interface AuditLogProps {
  entries: AuditLogEntry[]
  className?: string
}
export function AuditLog({ className = '', entries }: AuditLogProps) {
  const columns: DataTableColumn<AuditLogEntry>[] = [
    { key: 'actor', label: 'Actor' },
    { key: 'action', label: 'Action' },
    { key: 'target', label: 'Target' },
    { key: 'timestamp', label: 'Time' },
  ]
  return (
    <section
      aria-label="Audit log"
      className={className}
    >
      <DataTable
        caption="Audit log"
        columns={columns}
        rows={entries}
      />
    </section>
  )
}

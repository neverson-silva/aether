import type { ReactNode } from 'react'

export interface TableProps {
  headers: ReactNode[]
  rows: ReactNode[][]
  caption?: ReactNode
  className?: string
}

export function Table({ caption, className = '', headers, rows }: TableProps) {
  return (
    <div className={`overflow-x-auto border-y border-border-subtle ${className}`}>
      <table className="w-full min-w-max border-collapse text-left text-supporting">
        <caption className="sr-only">{caption ?? 'Data table'}</caption>
        <thead className="border-b border-border-default bg-surface-2 text-label text-text-tertiary">
          <tr>
            {headers.map((header, index) => (
              <th
                className="whitespace-nowrap px-4 py-2.5 font-medium"
                key={index}
                scope="col"
              >
                {header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-border-subtle bg-surface-1">
          {rows.map((row, rowIndex) => (
            <tr
              className="transition-[background-color] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:bg-surface-2 focus-within:bg-surface-2"
              key={rowIndex}
            >
              {row.map((cell, cellIndex) => (
                <td
                  className="px-4 py-2.5 text-text-secondary"
                  key={cellIndex}
                >
                  {cell}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

import type { ReactNode } from 'react'

export interface DiffLine {
  id: string
  content: ReactNode
  type: 'added' | 'removed' | 'unchanged'
}
export interface DiffViewerProps {
  lines: DiffLine[]
  className?: string
}

export function DiffViewer({ className = '', lines }: DiffViewerProps) {
  return (
    <section
      aria-label="Configuration diff"
      className={`overflow-auto border-y border-border-subtle bg-code-canvas font-technical text-log ${className}`}
    >
      <ol className="m-0 grid list-none gap-0 p-0">
        {lines.map((line, index) => (
          <li
            className={`grid grid-cols-[3rem_1rem_minmax(0,1fr)] gap-3 px-4 py-1 ${line.type === 'added' ? 'bg-success-soft text-success-strong' : line.type === 'removed' ? 'bg-danger-soft text-danger-strong' : 'text-text-secondary'}`}
            key={line.id}
          >
            <span className="select-none text-right text-text-subtle">{index + 1}</span>
            <span aria-hidden="true">
              {line.type === 'added' ? '+' : line.type === 'removed' ? '-' : ' '}
            </span>
            <span>{line.content}</span>
          </li>
        ))}
      </ol>
    </section>
  )
}

import { CaretDown, CaretRight } from '@phosphor-icons/react'
import { type ReactNode, useState } from 'react'
export interface DiffLine {
  id: string
  type: 'context' | 'addition' | 'removal'
  oldLine?: number
  newLine?: number
  content: string
}
export interface DiffViewerProps {
  lines: DiffLine[]
  mode?: 'unified' | 'split'
  title?: ReactNode
  collapseContext?: boolean
}
export function DiffViewer({
  collapseContext,
  lines,
  mode = 'unified',
  title,
}: DiffViewerProps) {
  const [collapsed, setCollapsed] = useState(false)
  const visible =
    collapseContext && collapsed
      ? lines.filter((line) => line.type !== 'context')
      : lines
  const splitRows = visible.reduce<{ left?: DiffLine; right?: DiffLine }[]>(
    (rows, line) => {
      const last = rows.at(-1)
      if (line.type === 'removal') rows.push({ left: line })
      else if (line.type === 'addition' && last && !last.right)
        last.right = line
      else rows.push({ right: line })
      return rows
    },
    [],
  )
  const renderLine = (line: DiffLine | undefined) =>
    line ? (
      <div
        className={`grid min-h-8 grid-cols-[3rem_1fr] gap-2 px-3 py-1 font-mono text-code-md ${line.type === 'addition' ? 'bg-status-success-container/20 text-status-success' : line.type === 'removal' ? 'bg-status-danger-container/20 text-status-danger' : 'text-muted-foreground'}`}
      >
        <span className="text-end opacity-50">
          {line.type === 'removal' ? line.oldLine : line.newLine}
        </span>
        <code>
          {line.type === 'addition' ? '+' : line.type === 'removal' ? '-' : ' '}{' '}
          {line.content}
        </code>
      </div>
    ) : (
      <div className="min-h-8 border-b border-border bg-surface-container-low" />
    )
  return (
    <section className="overflow-hidden rounded-lg border border-border bg-surface-lowest">
      <header className="flex items-center justify-between border-b border-border bg-surface-card px-3 py-2">
        <span className="text-body-sm font-semibold">{title ?? 'Changes'}</span>
        {collapseContext ? (
          <button
            type="button"
            onClick={() => setCollapsed(!collapsed)}
            className="inline-flex items-center gap-1 text-body-sm text-muted-foreground"
          >
            {collapsed ? <CaretRight size={14} /> : <CaretDown size={14} />}
            Context
          </button>
        ) : null}
      </header>
      {mode === 'split' ? (
        <div>
          {splitRows.map((row, index) => (
            <div
              key={index}
              className="grid grid-cols-2 divide-x divide-border"
            >
              {renderLine(row.left)}
              {renderLine(row.right)}
            </div>
          ))}
        </div>
      ) : (
        <div>
          {visible.map((line) => (
            <div
              key={line.id}
              className={`grid grid-cols-[3rem_3rem_1fr] gap-2 px-3 py-1 font-mono text-code-md ${line.type === 'addition' ? 'bg-status-success-container/20 text-status-success' : line.type === 'removal' ? 'bg-status-danger-container/20 text-status-danger' : 'text-muted-foreground'}`}
            >
              <span className="text-end opacity-50">{line.oldLine}</span>
              <span className="text-end opacity-50">{line.newLine}</span>
              <code>
                {line.type === 'addition'
                  ? '+'
                  : line.type === 'removal'
                    ? '-'
                    : ' '}{' '}
                {line.content}
              </code>
            </div>
          ))}
        </div>
      )}
    </section>
  )
}

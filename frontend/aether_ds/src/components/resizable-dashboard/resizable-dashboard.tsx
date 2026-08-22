import { ArrowCounterClockwise, Plus, Trash } from '@phosphor-icons/react'
import { type ReactNode, useState } from 'react'
export interface DashboardWidget {
  id: string
  title: string
  content: ReactNode
  colSpan?: number
  rowSpan?: number
}
export interface ResizableDashboardProps {
  widgets: DashboardWidget[]
  onChange?: (widgets: DashboardWidget[]) => void
  onSave?: () => void
  onReset?: () => void
  onAdd?: () => void
  empty?: ReactNode
}
export function ResizableDashboard({
  empty = 'Add a widget to start building your dashboard.',
  onAdd,
  onChange,
  onReset,
  onSave,
  widgets,
}: ResizableDashboardProps) {
  const [items, setItems] = useState(widgets)
  const update = (next: DashboardWidget[]) => {
    setItems(next)
    onChange?.(next)
  }
  const move = (from: number, to: number) => {
    if (to < 0 || to >= items.length) return
    const next = [...items]
    const [item] = next.splice(from, 1)
    next.splice(to, 0, item)
    update(next)
  }
  return (
    <section className="space-y-4">
      <header className="flex flex-wrap justify-end gap-2">
        <button
          type="button"
          onClick={onAdd}
          className="inline-flex items-center gap-1 rounded-md border border-border px-3 py-2 text-body-sm hover:bg-surface-container"
        >
          <Plus size={16} />
          Add widget
        </button>
        <button
          type="button"
          onClick={onReset}
          className="inline-flex items-center gap-1 rounded-md border border-border px-3 py-2 text-body-sm hover:bg-surface-container"
        >
          <ArrowCounterClockwise size={16} />
          Reset
        </button>
        <button
          type="button"
          onClick={onSave}
          className="rounded-md bg-primary px-3 py-2 text-body-sm text-primary-foreground"
        >
          Save layout
        </button>
      </header>
      {items.length ? (
        <div className="grid auto-rows-[minmax(8rem,auto)] grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
          {items.map((widget, index) => (
            <article
              key={widget.id}
              style={{
                gridColumn: `span ${Math.min(widget.colSpan ?? 1, 4)}`,
                gridRow: `span ${widget.rowSpan ?? 1}`,
              }}
              className="group relative rounded-lg border border-border bg-surface-card p-4"
            >
              <header className="mb-3 flex items-center justify-between">
                <h3 className="text-body-sm font-semibold">{widget.title}</h3>
                <div className="flex gap-1 opacity-0 transition-opacity group-hover:opacity-100">
                  <button
                    type="button"
                    onClick={() => move(index, index - 1)}
                    aria-label="Move widget left"
                    className="rounded px-1 text-label-caps hover:bg-surface-container"
                  >
                    ←
                  </button>
                  <button
                    type="button"
                    onClick={() => move(index, index + 1)}
                    aria-label="Move widget right"
                    className="rounded px-1 text-label-caps hover:bg-surface-container"
                  >
                    →
                  </button>
                  <button
                    type="button"
                    onClick={() =>
                      update(items.filter((item) => item.id !== widget.id))
                    }
                    aria-label={`Remove ${widget.title}`}
                    className="rounded px-1 text-status-danger hover:bg-surface-container"
                  >
                    <Trash size={14} />
                  </button>
                </div>
              </header>
              {widget.content}
            </article>
          ))}
        </div>
      ) : (
        <div className="rounded-lg border border-dashed border-border p-12 text-center text-body-sm text-muted-foreground">
          {empty}
        </div>
      )}
    </section>
  )
}

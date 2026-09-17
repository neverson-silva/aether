import { ArrowCounterClockwise, Plus, Trash } from "@phosphor-icons/react";
import { type ReactNode, useState } from "react";
export interface DashboardWidget {
  id: string;
  title: string;
  content: ReactNode;
  colSpan?: number;
  rowSpan?: number;
}
export interface ResizableDashboardProps {
  widgets: DashboardWidget[];
  onChange?: (widgets: DashboardWidget[]) => void;
  onSave?: () => void;
  onReset?: () => void;
  onAdd?: () => void;
  empty?: ReactNode;
}
export function ResizableDashboard({
  empty = "Add a widget to start building your dashboard.",
  onAdd,
  onChange,
  onReset,
  onSave,
  widgets,
}: ResizableDashboardProps) {
  const [items, setItems] = useState(widgets);
  const update = (next: DashboardWidget[]) => {
    setItems(next);
    onChange?.(next);
  };
  const move = (from: number, to: number) => {
    if (to < 0 || to >= items.length) return;
    const next = [...items];
    const [item] = next.splice(from, 1);
    next.splice(to, 0, item);
    update(next);
  };
  return (
    <section className="space-y-4">
      <header className="flex flex-wrap justify-end gap-2">
        <button
          type="button"
          onClick={onAdd}
          className="inline-flex items-center gap-1 rounded-xl border border-border px-3 py-2 text-body-sm outline-none transition-[background-color,border-color,transform] duration-150 hover:border-primary/50 hover:bg-surface-container focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985]"
        >
          <Plus size={16} />
          Add widget
        </button>
        <button
          type="button"
          onClick={onReset}
          className="inline-flex items-center gap-1 rounded-xl border border-border px-3 py-2 text-body-sm outline-none transition-[background-color,border-color,transform] duration-150 hover:border-primary/50 hover:bg-surface-container focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985]"
        >
          <ArrowCounterClockwise size={16} />
          Reset
        </button>
        <button
          type="button"
          onClick={onSave}
          className="rounded-xl bg-primary px-3 py-2 text-body-sm text-primary-foreground outline-none transition-[filter,transform] duration-150 hover:brightness-110 focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985]"
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
              className="group relative rounded-2xl border border-border bg-surface-card p-4 shadow-sm transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-0.5 hover:border-primary/40 hover:shadow-md"
            >
              <header className="mb-3 flex items-center justify-between">
                <h3 className="text-body-sm font-semibold">{widget.title}</h3>
                <div className="flex gap-1 opacity-0 transition-opacity group-hover:opacity-100">
                  <button
                    type="button"
                    onClick={() => move(index, index - 1)}
                    aria-label="Move widget left"
                    className="rounded-lg px-2 py-1 text-label-caps outline-none transition-colors hover:bg-surface-container focus-visible:ring-2 focus-visible:ring-ring"
                  >
                    ←
                  </button>
                  <button
                    type="button"
                    onClick={() => move(index, index + 1)}
                    aria-label="Move widget right"
                    className="rounded-lg px-2 py-1 text-label-caps outline-none transition-colors hover:bg-surface-container focus-visible:ring-2 focus-visible:ring-ring"
                  >
                    →
                  </button>
                  <button
                    type="button"
                    onClick={() =>
                      update(items.filter((item) => item.id !== widget.id))
                    }
                    aria-label={`Remove ${widget.title}`}
                    className="rounded-lg px-2 py-1 text-status-danger outline-none transition-colors hover:bg-surface-container focus-visible:ring-2 focus-visible:ring-ring"
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
        <div className="rounded-2xl border border-dashed border-border bg-surface-card p-12 text-center text-body-sm text-muted-foreground shadow-sm">
          {empty}
        </div>
      )}
    </section>
  );
}

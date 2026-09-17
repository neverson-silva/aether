import { CaretDown, CaretRight } from "@phosphor-icons/react";
import { type ReactNode, useState } from "react";
export interface AuditLogEntry {
  id: string;
  actor: ReactNode;
  action: ReactNode;
  resource: ReactNode;
  timestamp: ReactNode;
  requestId?: string;
  diff?: ReactNode;
}
export interface AuditLogProps {
  entries: AuditLogEntry[];
  empty?: ReactNode;
  loading?: boolean;
  onFilterChange?: (query: string) => void;
}
export function AuditLog({
  empty = "No audit events.",
  entries,
  loading,
  onFilterChange,
}: AuditLogProps) {
  const [expanded, setExpanded] = useState<string | null>(null);
  return (
    <section className="overflow-hidden rounded-2xl border border-border bg-surface-card shadow-sm">
      <div className="border-b border-border bg-surface-container-low/70 p-4">
        <input
          aria-label="Filter audit log"
          onChange={(event) => onFilterChange?.(event.target.value)}
          placeholder="Filter audit log"
          className="h-10 w-full max-w-sm rounded-xl border border-border bg-surface-control px-3 text-body-sm text-foreground outline-none transition-[border-color,box-shadow,background-color] duration-200 hover:bg-surface-container-highest/40 focus:border-primary focus:bg-surface-card focus:ring-2 focus:ring-primary/20"
        />
      </div>
      {loading ? (
        <div className="p-8 text-center text-body-sm text-muted-foreground">
          Loading audit log...
        </div>
      ) : entries.length ? (
        <div className="divide-y divide-border">
          {entries.map((entry) => {
            const isOpen = expanded === entry.id;
            return (
              <div key={entry.id}>
                <button
                  type="button"
                  onClick={() => setExpanded(isOpen ? null : entry.id)}
                  className="flex w-full items-center gap-3 px-4 py-3 text-start outline-none transition-[background-color,color] duration-150 hover:bg-surface-container focus-visible:bg-surface-container focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
                >
                  <span className="text-muted-foreground">
                    {entry.diff ? (
                      isOpen ? (
                        <CaretDown size={16} />
                      ) : (
                        <CaretRight size={16} />
                      )
                    ) : null}
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="block text-body-sm text-foreground">
                      <strong>{entry.actor}</strong> {entry.action}{" "}
                      <strong>{entry.resource}</strong>
                    </span>
                    <span className="block text-body-sm text-muted-foreground">
                      {entry.timestamp}
                    </span>
                  </span>
                  {entry.requestId ? (
                    <code className="hidden text-code-md text-muted-foreground md:block">
                      {entry.requestId}
                    </code>
                  ) : null}
                </button>
                {isOpen && entry.diff ? (
                  <div className="mb-3 ml-10 mr-4 rounded-xl border border-border/70 bg-surface-container p-3 text-body-sm text-foreground shadow-sm">
                    {entry.diff}
                  </div>
                ) : null}
              </div>
            );
          })}
        </div>
      ) : (
        <div className="p-8 text-center text-body-sm text-muted-foreground">
          {empty}
        </div>
      )}
    </section>
  );
}

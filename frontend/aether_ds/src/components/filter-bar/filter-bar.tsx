import { X } from "@phosphor-icons/react";
import type { ReactNode } from "react";
export interface FilterOption {
  id: string;
  label: string;
  value: string;
  displayValue?: ReactNode;
}
export interface FilterBarProps {
  filters: FilterOption[];
  onRemove?: (id: string) => void;
  onClear?: () => void;
  activeCount?: number;
  children?: ReactNode;
  loading?: boolean;
}
export function FilterBar({
  activeCount = 0,
  children,
  filters,
  loading,
  onClear,
  onRemove,
}: FilterBarProps) {
  return (
    <div className="flex min-w-0 flex-wrap items-center gap-2 rounded-2xl border border-border bg-surface-card p-2.5 shadow-sm">
      {loading ? (
        <span className="text-body-sm text-muted-foreground">
          Loading filters...
        </span>
      ) : null}
      {filters.map((filter) => (
        <span
          key={filter.id}
          className="inline-flex max-w-full items-center gap-1 rounded-full border border-primary/30 bg-primary/10 px-2.5 py-1 text-body-sm text-primary"
        >
          <span className="truncate">
            {filter.label}: {filter.displayValue ?? filter.value}
          </span>
          <button
            type="button"
            aria-label={`Remove ${filter.label} filter`}
            onClick={() => onRemove?.(filter.id)}
            className="rounded-full p-1 outline-none transition-[background-color,transform] duration-150 hover:bg-primary/15 focus-visible:ring-2 focus-visible:ring-ring active:scale-95"
          >
            <X size={14} aria-hidden="true" />
          </button>
        </span>
      ))}
      {children}
      <button
        type="button"
        disabled={!activeCount}
        onClick={onClear}
        className="ml-auto rounded-xl px-2.5 py-1.5 text-body-sm text-muted-foreground outline-none transition-[background-color,color,transform] duration-150 hover:bg-surface-container hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985] disabled:opacity-50"
      >
        Clear all{activeCount ? ` (${activeCount})` : ""}
      </button>
    </div>
  );
}

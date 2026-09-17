import { X } from "@phosphor-icons/react";
import type { ReactNode } from "react";
export interface BulkAction {
  id: string;
  label: string;
  destructive?: boolean;
  disabled?: boolean;
  onSelect?: () => void;
}
export interface BulkActionBarProps {
  selectedCount: number;
  actions: BulkAction[];
  onClear?: () => void;
  pending?: boolean;
  partialFailure?: ReactNode;
}
export function BulkActionBar({
  actions,
  onClear,
  partialFailure,
  pending,
  selectedCount,
}: BulkActionBarProps) {
  if (!selectedCount) return null;
  return (
    <div
      role="toolbar"
      aria-label="Bulk actions"
      className="sticky bottom-4 z-20 flex flex-wrap items-center gap-3 rounded-2xl border border-primary/30 bg-surface-modal p-3 text-foreground shadow-xl backdrop-blur-xl"
    >
      <span className="text-body-sm font-semibold">
        {selectedCount} selected
      </span>
      <div className="flex flex-wrap gap-2">
        {actions.map((action) => (
          <button
            type="button"
            key={action.id}
            disabled={pending || action.disabled}
            onClick={action.onSelect}
            className={`rounded-xl px-3 py-1.5 text-body-sm outline-none transition-[background-color,border-color,transform] duration-150 focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985] ${action.destructive ? "border border-status-danger/40 text-status-danger hover:bg-status-danger/10" : "border border-border hover:bg-surface-container"} disabled:opacity-50`}
          >
            {pending ? "Working..." : action.label}
          </button>
        ))}
      </div>
      {partialFailure ? (
        <span className="text-body-sm text-status-danger">
          {partialFailure}
        </span>
      ) : null}
      <button
        type="button"
        onClick={onClear}
        aria-label="Clear selection"
        className="ml-auto rounded-lg p-1 text-muted-foreground outline-none transition-[background-color,color,transform] duration-150 hover:bg-surface-container hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring active:scale-95"
      >
        <X size={18} />
      </button>
    </div>
  );
}

import type { ReactNode } from "react";
import {
  AsyncSearchInput,
  type AsyncSearchOption,
} from "../async-search-input/async-search-input";
import {
  ResourceTree,
  type ResourceTreeNode,
} from "../resource-tree/resource-tree";
export interface ResourcePickerProps {
  label?: string;
  description?: string;
  error?: string;
  nodes?: ResourceTreeNode[];
  selectedId?: string;
  onSelect?: (id: string) => void;
  loadOptions?: (query: string) => Promise<AsyncSearchOption[]>;
  recent?: ResourceTreeNode[];
  loading?: boolean;
  empty?: ReactNode;
}
export function ResourcePicker({
  description,
  empty,
  error,
  label = "Resource",
  loadOptions,
  nodes = [],
  onSelect,
  recent = [],
  selectedId,
}: ResourcePickerProps) {
  return (
    <div className="space-y-4">
      {loadOptions ? (
        <AsyncSearchInput
          label={label}
          description={description}
          error={error}
          loadOptions={loadOptions}
          onValueChange={(value) => value && onSelect?.(value)}
        />
      ) : (
        <div className="text-body-sm font-semibold text-foreground">
          {label}
        </div>
      )}
      {recent.length ? (
        <div>
          <div className="mb-2 text-label-caps text-muted-foreground">
            Recent
          </div>
          <div className="flex flex-wrap gap-2">
            {recent.map((item) => (
              <button
                type="button"
                key={item.id}
                onClick={() => onSelect?.(item.id)}
                className="rounded-xl border border-border px-3 py-1.5 text-body-sm outline-none transition-[background-color,border-color,transform] duration-150 hover:border-primary/50 hover:bg-surface-container focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985]"
              >
                {item.label}
              </button>
            ))}
          </div>
        </div>
      ) : null}
      <div className="rounded-2xl border border-border bg-surface-card p-3 shadow-sm">
        <ResourceTree
          nodes={nodes}
          selectedId={selectedId}
          onSelect={(node) => onSelect?.(node.id)}
          empty={empty}
        />
      </div>
    </div>
  );
}

import { Toggle as BaseToggle } from "@base-ui/react/toggle";
import { ToggleGroup as BaseToggleGroup } from "@base-ui/react/toggle-group";
import type { ReactNode } from "react";

export interface ToggleGroupProps {
  options: { value: string; label: ReactNode; disabled?: boolean }[];
  value?: string | string[];
  defaultValue?: string | string[];
  multiple?: boolean;
  onValueChange?: (value: string | string[]) => void;
}
export function ToggleGroup({
  defaultValue,
  multiple,
  onValueChange,
  options,
  value,
}: ToggleGroupProps) {
  const currentValue = value
    ? Array.isArray(value)
      ? value
      : [value]
    : undefined;
  const initialValue = defaultValue
    ? Array.isArray(defaultValue)
      ? defaultValue
      : [defaultValue]
    : undefined;
  return (
    <BaseToggleGroup
      value={currentValue}
      defaultValue={initialValue}
      multiple={multiple}
      onValueChange={(next) =>
        onValueChange?.(multiple ? next : (next[0] ?? ""))
      }
      className="inline-flex max-w-full overflow-x-auto rounded-2xl border border-border bg-surface-container-low p-1"
    >
      {options.map((option) => (
        <BaseToggle
          key={option.value}
          value={option.value}
          disabled={option.disabled}
          className="shrink-0 rounded-xl px-3 py-1.5 text-body-sm text-muted-foreground outline-none transition-[background-color,color,transform] duration-150 hover:bg-surface-container data-[pressed]:bg-surface-card data-[pressed]:text-primary data-[pressed]:shadow-sm focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985] disabled:opacity-50"
        >
          {option.label}
        </BaseToggle>
      ))}
    </BaseToggleGroup>
  );
}

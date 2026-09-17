import { Menu } from "@base-ui/react/menu";
import {
  CaretDown,
  DotsThreeVertical,
  Lock,
  PencilSimple,
  Plus,
  Star,
  Trash,
  Warning,
} from "@phosphor-icons/react";
export interface EnvironmentOption {
  id: string;
  label: string;
  kind?: "development" | "staging" | "production" | "preview";
  protected?: boolean;
  branch?: string;
  isDefault?: boolean;
}
export interface EnvironmentSwitcherProps {
  options: EnvironmentOption[];
  value?: string;
  onValueChange?: (id: string) => void;
  warning?: string;
  onCreate?: () => void;
  onEdit?: (id: string) => void;
  onSetDefault?: (id: string) => void;
  onDelete?: (id: string) => void;
}
export function EnvironmentSwitcher({
  onValueChange,
  options,
  value,
  warning,
  onCreate,
  onEdit,
  onSetDefault,
  onDelete,
}: EnvironmentSwitcherProps) {
  const current = options.find((item) => item.id === value) ?? options[0];
  return (
    <Menu.Root>
      <Menu.Trigger
        style={{
          display: "inline-flex",
          width: "fit-content",
          flexDirection: "row",
          alignItems: "center",
          gap: "0.25rem",
          whiteSpace: "nowrap",
          fontSize: "0.875rem",
          lineHeight: "1.25rem",
        }}
        className="rounded-xl border border-transparent bg-transparent px-2 py-1.5 text-foreground shadow-none outline-none transition-[background-color,border-color,transform] duration-150 hover:border-border hover:bg-surface-container focus-visible:ring-2 focus-visible:ring-primary active:scale-[0.985]"
      >
        <span>{current?.label}</span>
        <CaretDown size={14} aria-hidden="true" />
      </Menu.Trigger>
      <Menu.Portal>
        <Menu.Positioner className="z-50" sideOffset={6}>
          <Menu.Popup
            style={{
              backgroundColor: "var(--semantic-surface-popover)",
              opacity: 1,
            }}
            className="min-w-60 rounded-2xl border border-border bg-surface-popover p-1.5 shadow-xl"
          >
            {warning ? (
              <div className="flex gap-2 border-b border-border p-3 text-body-sm text-status-warning">
                <Warning size={16} />
                {warning}
              </div>
            ) : null}
            {options.map((option) => (
              <div
                key={option.id}
                className="group flex items-center gap-1 rounded-xl data-[highlighted]:bg-surface-container"
              >
                <Menu.Item
                  onClick={() => onValueChange?.(option.id)}
                  className="flex min-w-0 flex-1 cursor-pointer items-center gap-2 rounded-xl px-3 py-2 text-body-sm outline-none transition-colors data-[highlighted]:bg-surface-container"
                >
                  <span
                    className={`size-2 shrink-0 rounded-full ${option.kind === "production" ? "bg-status-danger" : option.kind === "staging" ? "bg-status-warning" : "bg-status-success"}`}
                  />
                  <span className="min-w-0 flex-1">
                    <span className="block truncate">{option.label}</span>
                    {option.branch ? (
                      <span className="block truncate text-body-sm text-muted-foreground">
                        {option.branch}
                      </span>
                    ) : null}
                  </span>
                  {option.protected || option.isDefault ? (
                    <Lock size={14} className="text-status-warning" />
                  ) : null}
                </Menu.Item>
                {onEdit || onSetDefault || onDelete ? (
                  <EnvironmentActions
                    option={option}
                    onEdit={onEdit}
                    onSetDefault={onSetDefault}
                    onDelete={onDelete}
                  />
                ) : null}
              </div>
            ))}
            {onCreate ? (
              <>
                <div className="my-1 border-t border-border" />
                <Menu.Item
                  onClick={onCreate}
                  className="flex cursor-pointer items-center gap-2 rounded-xl px-3 py-2 text-body-sm text-foreground outline-none transition-colors data-[highlighted]:bg-surface-container"
                >
                  <Plus size={16} />
                  Create environment
                </Menu.Item>
              </>
            ) : null}
          </Menu.Popup>
        </Menu.Positioner>
      </Menu.Portal>
    </Menu.Root>
  );
}

function EnvironmentActions({
  option,
  onEdit,
  onSetDefault,
  onDelete,
}: {
  option: EnvironmentOption;
  onEdit?: (id: string) => void;
  onSetDefault?: (id: string) => void;
  onDelete?: (id: string) => void;
}) {
  return (
    <Menu.Root>
      <Menu.Trigger className="mr-1 inline-flex size-8 shrink-0 items-center justify-center rounded-xl text-muted-foreground outline-none transition-[background-color,transform] duration-150 hover:bg-surface-container focus-visible:ring-2 focus-visible:ring-primary active:scale-95">
        <DotsThreeVertical size={16} aria-label={`${option.label} actions`} />
      </Menu.Trigger>
      <Menu.Portal>
        <Menu.Positioner
          className="z-[70]"
          side="right"
          sideOffset={8}
          align="start"
        >
          <Menu.Popup
            style={{
              backgroundColor: "var(--semantic-surface-popover)",
              opacity: 1,
            }}
            className="min-w-44 rounded-2xl border border-border bg-surface-popover p-1.5 shadow-xl"
          >
            {onEdit ? (
              <Menu.Item
                onClick={() => onEdit(option.id)}
                className="flex cursor-pointer items-center gap-2 rounded-xl px-3 py-2 text-body-sm outline-none transition-colors data-[highlighted]:bg-surface-container"
              >
                <PencilSimple size={16} />
                Rename
              </Menu.Item>
            ) : null}
            {onSetDefault && !option.isDefault ? (
              <Menu.Item
                onClick={() => onSetDefault(option.id)}
                className="flex cursor-pointer items-center gap-2 rounded-xl px-3 py-2 text-body-sm outline-none transition-colors data-[highlighted]:bg-surface-container"
              >
                <Star size={16} />
                Set as default
              </Menu.Item>
            ) : null}
            {onDelete ? (
              <Menu.Item
                onClick={() => onDelete(option.id)}
                className="flex cursor-pointer items-center gap-2 rounded-xl px-3 py-2 text-body-sm text-status-danger outline-none transition-colors data-[highlighted]:bg-status-danger-container"
              >
                <Trash size={16} />
                Delete
              </Menu.Item>
            ) : null}
          </Menu.Popup>
        </Menu.Positioner>
      </Menu.Portal>
    </Menu.Root>
  );
}

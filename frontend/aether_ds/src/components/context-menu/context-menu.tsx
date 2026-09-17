import { ContextMenu as BaseContextMenu } from "@base-ui/react/context-menu";
import type { ReactNode } from "react";
import type { DropdownMenuItem } from "../dropdown-menu/dropdown-menu";

export interface ContextMenuProps {
  children: ReactNode;
  items: DropdownMenuItem[];
}

export function ContextMenu({ children, items }: ContextMenuProps) {
  return (
    <BaseContextMenu.Root>
      <BaseContextMenu.Trigger className="block w-full">
        {children}
      </BaseContextMenu.Trigger>
      <BaseContextMenu.Portal>
        <BaseContextMenu.Positioner className="z-50">
          <BaseContextMenu.Popup className="min-w-48 max-w-[calc(100vw-2rem)] rounded-2xl border border-border bg-surface-popover p-1.5 text-foreground shadow-xl outline-none data-[starting-style]:scale-95 data-[starting-style]:opacity-0 data-[ending-style]:scale-95 data-[ending-style]:opacity-0 transition-[transform,opacity] duration-150">
            {items.map((item) => (
              <BaseContextMenu.Item
                key={item.value}
                disabled={item.disabled}
                onClick={item.onSelect}
                className={`flex cursor-pointer items-center rounded-xl px-3 py-2 text-body-sm outline-none transition-colors data-[highlighted]:bg-surface-container data-[disabled]:opacity-50 ${item.destructive ? "text-destructive" : ""}`}
              >
                {item.label}
              </BaseContextMenu.Item>
            ))}
          </BaseContextMenu.Popup>
        </BaseContextMenu.Positioner>
      </BaseContextMenu.Portal>
    </BaseContextMenu.Root>
  );
}

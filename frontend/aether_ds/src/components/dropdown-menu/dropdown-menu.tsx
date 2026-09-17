import { Menu } from "@base-ui/react/menu";
import type { ReactElement, ReactNode } from "react";

export interface DropdownMenuItem {
  value: string;
  label: ReactNode;
  disabled?: boolean;
  destructive?: boolean;
  onSelect?: () => void;
}
export interface DropdownMenuProps {
  trigger: ReactElement;
  items: DropdownMenuItem[];
  label?: string;
}

export function DropdownMenu({ items, label, trigger }: DropdownMenuProps) {
  return (
    <Menu.Root>
      <Menu.Trigger render={trigger} />
      <Menu.Portal>
        <Menu.Positioner className="z-50" sideOffset={6}>
          <Menu.Popup className="min-w-48 max-w-[calc(100vw-2rem)] rounded-2xl border border-border bg-surface-popover p-1.5 text-foreground shadow-xl outline-none backdrop-blur-xl data-[starting-style]:scale-95 data-[starting-style]:opacity-0 data-[ending-style]:scale-95 data-[ending-style]:opacity-0 transition-[transform,opacity] duration-150">
            {label ? (
              <Menu.GroupLabel className="px-3 py-2 text-label-caps text-muted-foreground">
                {label}
              </Menu.GroupLabel>
            ) : null}
            {items.map((item) => (
              <Menu.Item
                key={item.value}
                disabled={item.disabled}
                onClick={item.onSelect}
                className={`flex cursor-pointer items-center rounded-xl px-3 py-2 text-body-sm outline-none transition-[background-color,color,transform] duration-150 data-[highlighted]:bg-surface-container data-[disabled]:cursor-not-allowed data-[disabled]:opacity-50 ${item.destructive ? "text-destructive" : ""}`}
              >
                {item.label}
              </Menu.Item>
            ))}
          </Menu.Popup>
        </Menu.Positioner>
      </Menu.Portal>
    </Menu.Root>
  );
}

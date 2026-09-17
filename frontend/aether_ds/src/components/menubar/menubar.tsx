import { Menu } from "@base-ui/react/menu";
import { Menubar as BaseMenubar } from "@base-ui/react/menubar";
import type { ReactNode } from "react";

export interface MenubarItem {
  label: ReactNode;
  items: { value: string; label: ReactNode; onSelect?: () => void }[];
}
export interface MenubarProps {
  items: MenubarItem[];
}

export function Menubar({ items }: MenubarProps) {
  return (
    <BaseMenubar
      className="flex max-w-full items-center gap-1 overflow-x-auto rounded-2xl border border-border bg-surface-card p-1.5 shadow-sm"
      aria-label="Application menu"
    >
      {items.map((menu) => (
        <Menu.Root key={String(menu.label)}>
          <Menu.Trigger className="rounded-lg px-3 py-2 text-body-sm text-muted-foreground outline-none transition-[background-color,color,transform] duration-150 hover:bg-surface-container hover:text-foreground focus-visible:ring-2 focus-visible:ring-primary active:scale-[0.985] data-[popup-open]:bg-surface-container data-[popup-open]:text-foreground">
            {menu.label}
          </Menu.Trigger>
          <Menu.Portal>
            <Menu.Positioner className="z-50" sideOffset={4}>
              <Menu.Popup className="min-w-44 rounded-2xl border border-border bg-surface-popover p-1.5 shadow-xl outline-none data-[starting-style]:scale-95 data-[starting-style]:opacity-0 data-[ending-style]:scale-95 data-[ending-style]:opacity-0 transition-[transform,opacity] duration-150">
                {menu.items.map((item) => (
                  <Menu.Item
                    key={item.value}
                    onClick={item.onSelect}
                    className="cursor-pointer rounded-lg px-3 py-2 text-body-sm outline-none transition-colors data-[highlighted]:bg-surface-container"
                  >
                    {item.label}
                  </Menu.Item>
                ))}
              </Menu.Popup>
            </Menu.Positioner>
          </Menu.Portal>
        </Menu.Root>
      ))}
    </BaseMenubar>
  );
}

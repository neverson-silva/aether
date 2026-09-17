import { NavigationMenu as BaseNavigationMenu } from "@base-ui/react/navigation-menu";
import type { ReactNode } from "react";

export interface NavigationMenuItem {
  label: ReactNode;
  href?: string;
  description?: ReactNode;
  children?: ReactNode;
}
export interface NavigationMenuProps {
  items: NavigationMenuItem[];
}

export function NavigationMenu({ items }: NavigationMenuProps) {
  return (
    <BaseNavigationMenu.Root className="relative">
      <BaseNavigationMenu.List className="flex items-center gap-1">
        {items.map((item) => (
          <BaseNavigationMenu.Item key={String(item.label)}>
            {item.children ? (
              <>
                <BaseNavigationMenu.Trigger className="rounded-lg px-3 py-2 text-body-sm text-muted-foreground outline-none transition-[background-color,color,transform] duration-150 hover:bg-surface-container hover:text-foreground focus-visible:ring-2 focus-visible:ring-primary active:scale-[0.985]">
                  {item.label}
                </BaseNavigationMenu.Trigger>
                <BaseNavigationMenu.Content className="absolute left-0 top-full z-50 mt-2 min-w-64 max-w-[calc(100vw-2rem)] rounded-2xl border border-border bg-surface-popover p-3 shadow-xl outline-none data-[starting-style]:translate-y-1 data-[starting-style]:opacity-0 data-[ending-style]:translate-y-1 data-[ending-style]:opacity-0 transition-[transform,opacity] duration-200">
                  {item.children}
                </BaseNavigationMenu.Content>
              </>
            ) : (
              <BaseNavigationMenu.Link
                href={item.href}
                className="block rounded-xl px-3 py-2 text-body-sm text-muted-foreground outline-none transition-colors hover:bg-surface-container hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
              >
                {item.label}
              </BaseNavigationMenu.Link>
            )}
          </BaseNavigationMenu.Item>
        ))}
      </BaseNavigationMenu.List>
    </BaseNavigationMenu.Root>
  );
}

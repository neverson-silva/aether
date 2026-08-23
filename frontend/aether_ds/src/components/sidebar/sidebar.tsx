import type { ReactNode } from "react";
import { List } from "@phosphor-icons/react";
import { tv } from "tailwind-variants";

const sidebar = tv({
  base: "flex h-full flex-col border-r border-border bg-surface-card transition-[width] duration-200",
  variants: { collapsed: { true: "w-16", false: "w-64" } },
  defaultVariants: { collapsed: false },
});
export interface SidebarItem {
  label: ReactNode;
  href?: string;
  icon?: ReactNode;
  active?: boolean;
  badge?: ReactNode;
  disabled?: boolean;
  children?: SidebarItem[];
}
export interface SidebarProps {
  items: SidebarItem[];
  header?: ReactNode;
  footer?: ReactNode;
  collapsed?: boolean;
  mobile?: boolean;
  mobileOpen?: boolean;
  onCollapsedChange?: (collapsed: boolean) => void;
  onMobileOpenChange?: (open: boolean) => void;
  onNavigate?: (href: string) => void;
}
export function Sidebar({
  collapsed = false,
  mobile = false,
  footer,
  header,
  items,
  mobileOpen = true,
  onCollapsedChange,
  onNavigate,
}: SidebarProps) {
  const renderItem = (item: SidebarItem) => (
    <li key={String(item.label)}>
      <a
        href={item.disabled ? undefined : item.href}
        onClick={(event) => {
          if (
            !item.disabled &&
            item.href &&
            onNavigate &&
            event.button === 0 &&
            !event.metaKey &&
            !event.ctrlKey &&
            !event.shiftKey &&
            !event.altKey
          ) {
            event.preventDefault();
            onNavigate(item.href);
          }
        }}
        aria-current={item.active ? "page" : undefined}
        aria-disabled={item.disabled || undefined}
        title={collapsed ? String(item.label) : undefined}
        className={`flex items-center gap-3 rounded-md py-2 text-body-sm transition-colors ${collapsed ? "justify-center px-2" : "px-3"} ${item.active ? "bg-primary/10 text-primary" : "text-muted-foreground hover:bg-surface-container hover:text-foreground"} ${item.disabled ? "pointer-events-none opacity-50" : ""}`}
      >
        <span className="shrink-0">{item.icon}</span>
        {collapsed ? null : (
          <>
            <span className="min-w-0 flex-1 truncate">{item.label}</span>
            {item.badge}
          </>
        )}
      </a>
      {!collapsed && item.children?.length ? (
        <ul className="ml-5 mt-1 space-y-1 border-l border-border pl-2">
          {renderItems(item.children)}
        </ul>
      ) : null}
    </li>
  );
  const renderItems = (entries: SidebarItem[]) =>
    entries.flatMap((item) =>
      collapsed && item.children?.length
        ? item.children.map(renderItem)
        : [renderItem(item)],
    );
  return (
    <aside
      aria-label="Application navigation"
      className={`${sidebar({ collapsed })} shrink-0 ${mobile ? "fixed inset-y-0 left-0 z-50 shadow-xl" : ""} ${mobile && !mobileOpen ? "hidden" : ""}`}
    >
      <div className="flex items-center justify-between p-3">
        {collapsed ? null : header}
        <button
          type="button"
          className="flex size-10 items-center justify-center rounded-md text-muted-foreground hover:bg-surface-container"
          aria-label={collapsed ? "Expand navigation" : "Collapse navigation"}
          onClick={() => onCollapsedChange?.(!collapsed)}
        >
          <List size={22} weight="bold" aria-hidden="true" />
        </button>
      </div>
      <nav className="min-h-0 flex-1 overflow-y-auto px-2">
        <ul className="space-y-1">{renderItems(items)}</ul>
      </nav>
      {footer ? (
        <div className="border-t border-border p-3">
          {collapsed ? null : footer}
        </div>
      ) : null}
    </aside>
  );
}

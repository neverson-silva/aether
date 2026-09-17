import { Tabs as BaseTabs } from "@base-ui/react/tabs";
import type { ReactNode } from "react";
export interface TabItem {
  value: string;
  label: ReactNode;
  content: ReactNode;
  disabled?: boolean;
}
export interface TabsProps {
  items: TabItem[];
  value?: string;
  defaultValue?: string;
  activation?: "automatic" | "manual";
  onValueChange?: (value: string) => void;
  variant?: "underline" | "pill";
}
export function Tabs({
  activation = "automatic",
  defaultValue,
  items,
  onValueChange,
  value,
  variant = "underline",
}: TabsProps) {
  return (
    <BaseTabs.Root
      value={value}
      defaultValue={defaultValue}
      onValueChange={onValueChange}
    >
      <BaseTabs.List
        activateOnFocus={activation === "automatic"}
        className={`flex max-w-full gap-1 overflow-x-auto ${variant === "pill" ? "rounded-2xl border border-border bg-surface-container-low p-1" : "border-b border-border"}`}
      >
        {items.map((item) => (
          <BaseTabs.Tab
            key={item.value}
            value={item.value}
            disabled={item.disabled}
            className={`shrink-0 rounded-xl px-3 py-2 text-body-sm outline-none transition-[background-color,color,box-shadow,transform] duration-200 focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985] ${variant === "underline" ? "border-b-2 border-transparent text-muted-foreground data-[active]:border-primary data-[active]:text-primary" : "text-muted-foreground hover:bg-surface-container-high hover:text-foreground data-[active]:bg-surface-card data-[active]:text-primary data-[active]:shadow-sm"}`}
          >
            {item.label}
          </BaseTabs.Tab>
        ))}
      </BaseTabs.List>
      {items.map((item) => (
        <BaseTabs.Panel
          key={item.value}
          value={item.value}
          className="pt-4 focus-visible:outline-none"
        >
          {item.content}
        </BaseTabs.Panel>
      ))}
    </BaseTabs.Root>
  );
}

import type { ElementType } from "react";

export function Metric({ label, value, icon: Icon }: { label: string; value: string; icon: ElementType }) {
  return (
    <div className="flex flex-col gap-xs">
      <Icon size={18} className="text-primary" />
      <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">{label}</span>
      <span className="font-code-md text-code-md text-on-surface">{value}</span>
    </div>
  );
}

import type { ElementType } from "react";

export function UseCase({ icon: Icon, label, desc }: { icon: ElementType; label: string; desc: string }) {
  return (
    <div className="flex items-start gap-sm p-sm rounded border border-outline-variant/60">
      <Icon size={20} className="text-primary shrink-0" />
      <div>
        <p className="font-body-md text-body-md text-on-surface">{label}</p>
        <p className="font-body-sm text-body-sm text-on-surface-variant">{desc}</p>
      </div>
    </div>
  );
}

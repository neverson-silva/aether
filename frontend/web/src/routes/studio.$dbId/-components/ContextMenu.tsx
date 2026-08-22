import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { Archive, ArrowsClockwise, Code, DotsThree, PencilSimple, Trash, type Icon } from "@phosphor-icons/react";

function cn(...classes: Array<string | false | undefined>) { return classes.filter(Boolean).join(" "); }

const ICONS: Record<string, Icon> = { edit: PencilSimple, drive_file_rename_outline: PencilSimple, delete: Trash, refresh: ArrowsClockwise, code: Code, table: Archive };

export interface ContextMenuItem {
  label: string;
  icon?: string;
  danger?: boolean;
  disabled?: boolean;
  onSelect: () => void;
}

export interface ContextMenuState {
  x: number;
  y: number;
  items: ContextMenuItem[];
}

// Positioned context menu with viewport clamping, ESC and click-outside
// close. Rendered in a portal so it is never clipped by the explorer scroll.
export function ContextMenu({ menu, onClose }: { menu: ContextMenuState | null; onClose: () => void }) {
  const ref = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState({ x: 0, y: 0 });

  useLayoutEffect(() => {
    if (!menu || !ref.current) return;
    const r = ref.current.getBoundingClientRect();
    const vw = window.innerWidth;
    const vh = window.innerHeight;
    setPos({
      x: Math.min(menu.x, Math.max(0, vw - r.width - 8)),
      y: Math.min(menu.y, Math.max(0, vh - r.height - 8)),
    });
  }, [menu]);

  useEffect(() => {
    if (!menu) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) onClose();
    };
    window.addEventListener("keydown", onKey);
    window.addEventListener("mousedown", onClick);
    return () => {
      window.removeEventListener("keydown", onKey);
      window.removeEventListener("mousedown", onClick);
    };
  }, [menu, onClose]);

  if (!menu) return null;
  return (
    <div
      ref={ref}
      role="menu"
      className="fixed z-[95] min-w-44 rounded-lg bg-surface-container-highest border border-outline-variant shadow-[0_10px_30px_rgba(0,0,0,0.5)] py-1"
      style={{ left: pos.x, top: pos.y }}
      onClick={(e) => e.stopPropagation()}
    >
      {menu.items.map((item, i) => (
        <button
          key={i}
          role="menuitem"
          disabled={item.disabled}
          onClick={() => {
            onClose();
            item.onSelect();
          }}
          className={cn(
            "w-full flex items-center gap-sm px-3 py-1.5 text-left font-body-sm text-body-sm transition-colors",
            item.danger ? "text-error hover:bg-error/10" : "text-on-surface hover:bg-surface-container-high",
            item.disabled && "opacity-40 cursor-not-allowed",
          )}
        >
          {item.icon ? (() => { const IconComponent = ICONS[item.icon] ?? DotsThree; return <IconComponent size={16} aria-hidden="true" />; })() : null}
          {item.label}
        </button>
      ))}
    </div>
  );
}

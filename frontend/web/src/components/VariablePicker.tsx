import { useEffect, useMemo, useRef, useState } from "react";
import { ChartLine, Database, FolderOpen, Globe, LockKey, MagnifyingGlass, Package } from "@phosphor-icons/react";

export type VarScope = "service" | "project" | "organization" | "environment" | "secrets" | "system";

export interface PickedVar {
  name: string;
  description?: string;
  value?: string;
  scope: VarScope;
}

export interface VarGroup {
  scope: VarScope;
  items: PickedVar[];
}

const SCOPE_META: Record<VarScope, { icon: typeof Package; label: string; color: string }> = {
  service: { icon: Package, label: "Service", color: "text-[#60a5fa]" },
  project: { icon: FolderOpen, label: "Project", color: "text-[#34d399]" },
  organization: { icon: Globe, label: "Organization", color: "text-[#a78bfa]" },
  environment: { icon: Database, label: "Environment", color: "text-[#fbbf24]" },
  secrets: { icon: LockKey, label: "Secrets", color: "text-[#f472b6]" },
  system: { icon: ChartLine, label: "System", color: "text-on-surface-variant" },
};

export function VariablePicker({
  groups,
  query,
  open,
  onSelect,
  onClose,
  inputRef,
  previewText,
}: {
  groups: VarGroup[];
  query: string;
  open: boolean;
  onSelect: (v: PickedVar) => void;
  onClose: () => void;
  inputRef: React.RefObject<HTMLInputElement>;
  previewText?: string | null;
}) {
  const [active, setActive] = useState(0);
  const listRef = useRef<HTMLDivElement>(null);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    const out: { scope: VarScope; items: PickedVar[] }[] = [];
    for (const g of groups) {
      const items = q ? g.items.filter((i) => i.name.toLowerCase().includes(q)) : g.items;
      if (items.length) out.push({ scope: g.scope, items });
    }
    return out;
  }, [groups, query]);

  const flat = useMemo(() => filtered.flatMap((g) => g.items), [filtered]);

  useEffect(() => {
    setActive(0);
  }, [query, open]);

  useEffect(() => {
    if (!open) return;
    listRef.current?.querySelector(`[data-idx="${active}"]`)?.scrollIntoView({ block: "nearest" });
  }, [active, open]);

  if (!open) return null;

  return (
    <div className="fixed z-[120] w-[340px] max-w-[calc(100vw-16px)] max-h-[360px] bg-surface-popover border border-outline-variant rounded-xl shadow-2xl overflow-hidden flex flex-col animate-modal-pop">
      <div className="px-3 py-2 border-b border-outline-variant bg-surface-container-low flex items-center gap-2">
        <MagnifyingGlass size={15} className="text-primary" aria-hidden="true" />
        <span className="font-code-md text-code-md text-on-surface-variant">${query}</span>
        <div className="flex-1" />
        <span className="font-code-md text-[10px] text-on-surface-variant/60">{flat.length}</span>
      </div>
      <div ref={listRef} className="overflow-y-auto sidebar-scroll flex-1">
        {filtered.length === 0 && (
          <p className="px-3 py-4 text-center font-body-sm text-body-sm text-on-surface-variant">No variables match "{query}".</p>
        )}
        {filtered.map((g) => {
          const meta = SCOPE_META[g.scope];
          return (
            <div key={g.scope}>
              <div className="flex items-center gap-1.5 px-3 pt-2 pb-1 font-label-caps text-[10px] uppercase tracking-wider text-on-surface-variant/70">
                <meta.icon size={13} className={meta.color} aria-hidden="true" />
                {meta.label}
                <span className="font-code-md text-[10px] text-on-surface-variant/40">· {g.items.length}</span>
              </div>
              {g.items.map((v) => {
                const idx = flat.findIndex((x) => x.name === v.name && x.scope === v.scope);
                return (
                  <button
                    key={v.scope + v.name}
                    data-idx={idx}
                    onClick={() => onSelect(v)}
                    onMouseEnter={() => setActive(idx)}
                    className={`w-full flex items-center gap-2 px-3 py-1.5 text-left transition-colors ${active === idx ? "bg-primary/10" : "hover:bg-surface-container-high"}`}
                  >
                    {(() => { const Icon = SCOPE_META[v.scope].icon; return <Icon size={15} className={`shrink-0 ${SCOPE_META[v.scope].color}`} aria-hidden="true" />; })()}
                    <span className="flex-1 min-w-0">
                      <span className="block font-code-md text-code-md text-on-surface truncate">$&#123;{v.name}&#125;</span>
                      {v.description && <span className="block font-body-sm text-body-sm text-on-surface-variant/70 truncate">{v.description}</span>}
                    </span>
                  </button>
                );
              })}
            </div>
          );
        })}
      </div>
      {(previewText ?? "") !== "" && (
        <div className="px-3 py-2 border-t border-outline-variant bg-surface-container-low font-code-md text-[11px] text-on-surface-variant">
          Preview: <span className="text-[#4ade80]">{previewText}</span>
        </div>
      )}
      <input
        ref={inputRef}
        value={query}
        onChange={() => {}}
        onKeyDown={(e) => {
          if (e.key === "ArrowDown") {
            e.preventDefault();
            setActive((a) => (a + 1) % Math.max(flat.length, 1));
          } else if (e.key === "ArrowUp") {
            e.preventDefault();
            setActive((a) => (a - 1 + Math.max(flat.length, 1)) % Math.max(flat.length, 1));
          } else if (e.key === "Enter") {
            e.preventDefault();
            if (flat[active]) onSelect(flat[active]);
          } else if (e.key === "Escape") {
            e.preventDefault();
            onClose();
          } else {
            if (e.key.length === 1 && inputRef.current?.parentElement) {
              const ev = new KeyboardEvent("keydown", { key: e.key, bubbles: true });
              (inputRef.current.parentElement as HTMLElement).dispatchEvent(ev);
            }
          }
        }}
        className="absolute w-px h-px opacity-0 pointer-events-none"
        tabIndex={-1}
      />
    </div>
  );
}

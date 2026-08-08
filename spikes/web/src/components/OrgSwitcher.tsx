import { useState, useRef, useEffect } from "react";
import { useOrg } from "./OrgProvider";
import { useNavigate } from "@tanstack/react-router";

const COLORS = ["#7c3aed", "#0ea5e9", "#10b981", "#f59e0b", "#ef4444", "#ec4899"];

function avatarColor(name: string): string {
  let h = 0;
  for (let i = 0; i < name.length; i++) h = (h * 31 + name.charCodeAt(i)) % 997;
  return COLORS[h % COLORS.length];
}

export function OrgSwitcher() {
  const { orgs, currentOrg, role, switchOrg } = useOrg();
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const ref = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();

  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open]);

  if (!currentOrg) return null;

  const filtered = orgs.filter(
    (o) => !query || o.name.toLowerCase().includes(query.toLowerCase())
  );

  const toggle = () => setOpen((v) => !v);

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={toggle}
        className="w-full flex items-center gap-2 px-sm py-1.5 rounded-lg border border-outline-variant hover:border-primary/40 bg-surface-container-lowest transition-colors group"
      >
        <span
          className="w-7 h-7 rounded-md flex items-center justify-center text-on-primary text-[12px] font-bold shrink-0"
          style={{ background: currentOrg.color || avatarColor(currentOrg.name) }}
        >
          {(currentOrg.name || "?").slice(0, 2).toUpperCase()}
        </span>
        <span className="flex-1 min-w-0 text-left">
          <span className="block font-body-sm font-semibold text-on-surface truncate leading-tight">{currentOrg.name}</span>
          <span className="block font-label-caps text-[9px] text-on-surface-variant uppercase tracking-wider">{role}</span>
        </span>
        <span className={`material-symbols-outlined text-[16px] text-on-surface-variant transition-transform ${open ? "rotate-180" : ""}`}>
          expand_more
        </span>
      </button>

      {open && (
        <div className="absolute left-0 top-full mt-1 w-[280px] max-w-[calc(100vw-16px)] bg-surface-container-lowest border border-outline-variant rounded-xl shadow-2xl z-[90] p-sm animate-modal-pop">
          <div className="px-sm py-1.5 font-label-caps text-[10px] text-on-surface-variant/70 uppercase tracking-wider">Organizations</div>
          <div className="relative mb-sm">
            <span className="material-symbols-outlined absolute left-2 top-1/2 -translate-y-1/2 text-[14px] text-on-surface-variant">search</span>
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search organizations..."
              className="w-full bg-surface-container-lowest border border-outline-variant rounded-lg pl-7 pr-2 py-1.5 font-body-sm text-body-sm text-on-surface placeholder:text-on-surface-variant/50 focus:border-primary focus:outline-none"
            />
          </div>
          <div className="max-h-56 overflow-y-auto sidebar-scroll space-y-0.5">
            {filtered.map((o) => (
              <button
                key={o.id}
                onClick={() => {
                  switchOrg(o.id);
                  setOpen(false);
                  setQuery("");
                }}
                className={`w-full flex items-center gap-2 px-2 py-1.5 rounded-lg text-left transition-colors ${currentOrg.id === o.id ? "bg-primary/10 text-primary" : "hover:bg-surface-container-high"}`}
              >
                <span
                  className="w-6 h-6 rounded-md flex items-center justify-center text-on-primary text-[10px] font-bold shrink-0"
                  style={{ background: o.color || avatarColor(o.name) }}
                >
                  {(o.name || "?").slice(0, 2).toUpperCase()}
                </span>
                <span className="flex-1 min-w-0">
                  <span className="block font-body-sm text-body-sm text-on-surface truncate">{o.name}</span>
                  <span className="block font-label-caps text-[9px] text-on-surface-variant uppercase">{o.role}</span>
                </span>
                {currentOrg.id === o.id && (
                  <span className="material-symbols-outlined text-[14px]" style={{ fontVariationSettings: "'FILL' 1" }}>check_circle</span>
                )}
              </button>
            ))}
          </div>
          <div className="border-t border-outline-variant mt-sm pt-sm flex flex-col gap-0.5">
            <button
              onClick={() => {
                setOpen(false);
                navigate({ to: "/organizations/new" });
              }}
              className="flex items-center gap-2 px-2 py-1.5 rounded-lg text-on-surface hover:bg-surface-container-high font-body-sm text-body-sm"
            >
              <span className="material-symbols-outlined text-[15px]">add</span>
              Create Organization
            </button>
            {currentOrg && (
              <button
                onClick={() => {
                  setOpen(false);
                  navigate({ to: "/organizations/$id", params: { id: currentOrg.id } });
                }}
                className="flex items-center gap-2 px-2 py-1.5 rounded-lg text-on-surface hover:bg-surface-container-high font-body-sm text-body-sm"
              >
                <span className="material-symbols-outlined text-[15px]">settings</span>
                Manage Organization
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

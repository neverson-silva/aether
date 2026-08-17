import { useState, useRef, useEffect, useLayoutEffect, useCallback } from "react";
import { createPortal } from "react-dom";
import { useOrg } from "./OrgProvider";
import { useNavigate } from "@tanstack/react-router";

const ORG_ICONS = ["corporate_fare", "business", "apartment", "account_balance", "groups", "domain"];

const POPOVER_WIDTH = 320;

interface Position {
  top: number;
  left: number;
}

function computePosition(trigger: DOMRect): Position {
  const margin = 8;
  let left = trigger.left;
  let top = trigger.bottom + margin;
  if (left + POPOVER_WIDTH > window.innerWidth - 8) {
    left = Math.max(8, window.innerWidth - POPOVER_WIDTH - 8);
  }
  const height = 0;
  if (top + 360 > window.innerHeight - 8) {
    top = Math.max(8, trigger.top - 360 - margin);
  }
  if (top < 8) {
    top = Math.max(8, trigger.bottom + margin);
  }
  return { top, left };
}

export function OrgSwitcher() {
  const { orgs, currentOrg, switchOrg } = useOrg();
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [pos, setPos] = useState<Position | null>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const popRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();

  const reposition = useCallback(() => {
    if (!triggerRef.current) return;
    setPos(computePosition(triggerRef.current.getBoundingClientRect()));
  }, []);

  useEffect(() => {
    if (!open) return;
    reposition();
  }, [open, reposition]);

  useLayoutEffect(() => {
    if (!open || !pos) return;
    const onResize = () => reposition();
    const onScroll = () => {
      if (popRef.current && popRef.current.contains(document.activeElement)) return;
      reposition();
    };
    window.addEventListener("resize", onResize);
    window.addEventListener("scroll", onScroll, true);
    return () => {
      window.removeEventListener("resize", onResize);
      window.removeEventListener("scroll", onScroll, true);
    };
  }, [open, pos, reposition]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setOpen(false);
        triggerRef.current?.focus();
      }
    };
    const onClick = (e: MouseEvent) => {
      const t = e.target as Node;
      if (triggerRef.current?.contains(t)) return;
      if (popRef.current?.contains(t)) return;
      setOpen(false);
    };
    window.addEventListener("keydown", onKey);
    document.addEventListener("mousedown", onClick);
    return () => {
      window.removeEventListener("keydown", onKey);
      document.removeEventListener("mousedown", onClick);
    };
  }, [open]);

  if (!currentOrg) return null;

  const filtered = orgs.filter(
    (o) => !query || o.name.toLowerCase().includes(query.toLowerCase())
  );

  const close = () => {
    setOpen(false);
    setQuery("");
    triggerRef.current?.focus();
  };

  const toggle = () => {
    setOpen((v) => {
      const next = !v;
      if (next) {
        requestAnimationFrame(reposition);
      } else {
        triggerRef.current?.focus();
      }
      return next;
    });
  };

  return (
    <>
      <button
        ref={triggerRef}
        onClick={toggle}
        aria-haspopup="menu"
        aria-expanded={open}
        className="w-full flex items-center justify-between px-2 py-1.5 rounded bg-surface-container-high border border-outline-variant/30 hover:border-primary/50 transition-colors group"
      >
        <div className="flex items-center gap-2 min-w-0">
          <div className="w-6 h-6 bg-primary rounded flex items-center justify-center shrink-0">
            <span className="material-symbols-outlined text-[16px] text-on-primary">corporate_fare</span>
          </div>
          <span className="font-medium text-body-sm text-on-surface truncate">{currentOrg.name}</span>
        </div>
        <span className="material-symbols-outlined text-[16px] text-on-surface-variant group-hover:text-primary transition-colors shrink-0">unfold_more</span>
      </button>

      {open &&
        pos &&
        createPortal(
          <div
            ref={popRef}
            role="menu"
            aria-label="My Organizations"
            className="fixed z-[90] w-[320px] max-w-[calc(100vw-16px)] bg-surface-container-high rounded-xl shadow-2xl flex flex-col overflow-hidden border border-outline-variant animate-modal-pop"
            style={{ top: pos.top, left: pos.left }}
            onClick={(e) => e.stopPropagation()}
          >
            <div className="p-3 border-b border-outline-variant">
              <div className="relative flex items-center">
                <span className="material-symbols-outlined absolute left-2 text-[18px] text-on-surface-variant">search</span>
                <input
                  autoFocus
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Escape") close();
                  }}
                  className="w-full bg-surface-container-high border border-outline-variant/30 rounded px-8 py-1.5 text-body-sm text-on-surface focus:outline-none focus:border-primary/50 transition-colors"
                  placeholder="Search organizations..."
                />
              </div>
            </div>

            <div className="flex-1 overflow-y-auto max-h-[320px] py-2">
              <div className="px-4 py-1 text-label-caps text-on-surface-variant opacity-60 mb-1">My Organizations</div>
              {filtered.map((o, idx) => {
                const active = currentOrg.id === o.id;
                const icon = ORG_ICONS[idx % ORG_ICONS.length];
                return (
                  <div
                    key={o.id}
                    role="menuitem"
                    tabIndex={0}
                    onClick={() => {
                      switchOrg(o.id);
                      close();
                    }}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" || e.key === " ") {
                        e.preventDefault();
                        switchOrg(o.id);
                        close();
                      }
                    }}
                    className="group flex items-center justify-between px-4 py-2 hover:bg-surface-container-high cursor-pointer transition-colors outline-none focus-visible:bg-surface-container-high"
                  >
                    <div className="flex items-center gap-3 min-w-0">
                      <div
                        className={`w-8 h-8 rounded flex items-center justify-center border shrink-0 ${
                          active ? "bg-primary/10 border-primary/30" : "bg-surface-container border-outline-variant"
                        }`}
                      >
                        <span className={`material-symbols-outlined text-[18px] ${active ? "text-primary" : "text-on-surface-variant"}`}>
                          {active ? "corporate_fare" : icon}
                        </span>
                      </div>
                      <div className="min-w-0">
                        <div className="text-body-sm font-medium text-on-surface truncate">{o.name}</div>
                        <div className="text-[10px] text-on-surface-variant truncate">{o.role}</div>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      {active && (
                        <span className="material-symbols-outlined text-[18px] text-primary" style={{ fontVariationSettings: "'FILL' 1" }}>
                          check
                        </span>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>

            <div className="p-3 border-t border-outline-variant bg-surface-container">
              <button
                role="menuitem"
                onClick={() => {
                  close();
                  navigate({ to: "/organizations/new" });
                }}
                className="w-full flex items-center justify-center gap-2 py-2 rounded bg-primary/10 border border-primary/20 text-primary font-label-caps text-label-caps hover:bg-primary/20 transition-all"
              >
                <span className="material-symbols-outlined text-[18px]">add</span>
                Create New Organization
              </button>
            </div>
          </div>,
          document.body
        )}
    </>
  );
}

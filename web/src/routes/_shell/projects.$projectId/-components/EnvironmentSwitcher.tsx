import { useEffect, useRef, useState } from "react";
import type { EnvSummary } from "../../../../hooks";
import { Button } from "../../../../components/ui";

function tone(status: string): string {
  return status === "healthy" ? "#4ade80" : status === "degraded" ? "#ffb4ab" : status === "syncing" ? "#fbbf24" : "#c2c6d8";
}

export function EnvironmentSwitcher({
  envs,
  selected,
  onSelect,
  onCreate,
  onEdit,
  onSetDefault,
  onDelete,
}: {
  envs: EnvSummary[];
  selected: string | null;
  onSelect: (id: string) => void;
  onCreate: () => void;
  onEdit: (env: EnvSummary) => void;
  onSetDefault: (id: string) => void;
  onDelete: (env: EnvSummary) => void;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const active = envs.find((e) => e.id === selected);

  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

  return (
    <div ref={ref} className="relative inline-block">
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex items-center gap-2 py-1.5 pl-1 pr-2 rounded-lg hover:bg-surface-container-high transition-colors min-w-[140px] justify-between group"
      >
        <span className="flex items-center gap-2.5 min-w-0">
          <span className="w-2.5 h-2.5 rounded-full shrink-0" style={{ backgroundColor: active?.color || "#b0c6ff" }} />
          <span className="font-body-md text-body-md text-on-surface truncate">{active?.name ?? "environment"}</span>
        </span>
        <span className="material-symbols-outlined text-[16px] text-on-surface-variant/60 group-hover:text-on-surface shrink-0 transition-colors">
          {open ? "expand_less" : "expand_more"}
        </span>
      </button>

      {open && (
        <div className="absolute left-0 top-full mt-2 z-[70] w-72 rounded-xl bg-surface-popover border border-border-subtle shadow-md overflow-hidden p-1">
          <div className="max-h-72 overflow-y-auto">
            {envs.map((e) => (
              <div
                key={e.id}
                className={`flex items-center gap-2.5 px-2 py-2 rounded-lg cursor-pointer transition-colors ${
                  selected === e.id ? "bg-secondary-container/20" : "hover:bg-surface-container-high"
                }`}
              >
                <button
                  onClick={() => {
                    onSelect(e.id);
                    setOpen(false);
                  }}
                  className="flex items-center gap-2 flex-1 min-w-0 text-left"
                >
                  <span className="w-2.5 h-2.5 rounded-full shrink-0" style={{ backgroundColor: e.color || "#b0c6ff" }} />
                  <span className="flex-1 min-w-0">
                    <span className="flex items-center gap-2">
                      <span className="font-body-md text-body-md text-on-surface truncate">{e.name}</span>
                      {e.is_default && (
                        <span className="px-1 py-0.5 rounded border border-primary/30 font-code-md text-code-md text-primary shrink-0">default</span>
                      )}
                    </span>
                    <span className="flex items-center gap-2 font-code-md text-code-md text-on-surface-variant/70">
                      <span>{e.apps} service(s)</span>
                      <span className="w-1.5 h-1.5 rounded-full inline-block" style={{ backgroundColor: tone(e.status) }} />
                    </span>
                  </span>
                  <span className="material-symbols-outlined text-[16px] text-on-surface-variant/40 shrink-0">more_vert</span>
                </button>
                <div className="relative" onClick={(ev) => ev.stopPropagation()}>
                  <MenuItems env={e} onEdit={onEdit} onSetDefault={onSetDefault} onDelete={onDelete} onClose={() => setOpen(false)} />
                </div>
              </div>
            ))}
          </div>
          <div className="mt-1 pt-1 border-t border-border-subtle">
            <button
              onClick={() => {
                setOpen(false);
                onCreate();
              }}
              className="w-full flex items-center gap-2.5 px-2 py-2 rounded-lg text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface transition-colors font-body-sm text-body-sm"
            >
              <span className="material-symbols-outlined text-[16px]">add</span>
              Create Environment
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

function MenuItems({
  env,
  onEdit,
  onSetDefault,
  onDelete,
  onClose,
}: {
  env: EnvSummary;
  onEdit: (env: EnvSummary) => void;
  onSetDefault: (id: string) => void;
  onDelete: (env: EnvSummary) => void;
  onClose: () => void;
}) {
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) setMenuOpen(false);
    };
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

  return (
    <div ref={menuRef} className="relative">
      <button
        onClick={() => setMenuOpen((v) => !v)}
        className="w-6 h-6 rounded-md flex items-center justify-center text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface transition-colors"
        aria-label="Environment actions"
      >
        <span className="material-symbols-outlined text-[16px]">more_vert</span>
      </button>
      {menuOpen && (
        <div className="absolute right-0 top-7 z-[80] w-44 rounded-xl bg-surface-popover border border-border-subtle shadow-md overflow-hidden p-1">
          <MenuAction icon="edit" label="Rename" onClick={() => { onEdit(env); setMenuOpen(false); onClose(); }} />
          {!env.is_default && (
            <MenuAction icon="star" label="Set as default" onClick={() => { onSetDefault(env.id); setMenuOpen(false); }} />
          )}
          <MenuAction icon="delete" label="Delete" danger onClick={() => { onDelete(env); setMenuOpen(false); onClose(); }} />
        </div>
      )}
    </div>
  );
}

function MenuAction({
  icon,
  label,
  danger,
  onClick,
}: {
  icon: string;
  label: string;
  danger?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className={`w-full flex items-center gap-2 px-2 py-1.5 rounded-md font-body-sm text-body-sm transition-colors ${
        danger ? "text-error hover:bg-error/10" : "text-on-surface hover:bg-surface-container-high"
      }`}
    >
      <span className="material-symbols-outlined text-[16px]">{icon}</span>
      {label}
    </button>
  );
}

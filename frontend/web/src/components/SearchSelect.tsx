import { useEffect, useMemo, useRef, useState } from "react";
import { CaretDown, Check, MagnifyingGlass } from "@phosphor-icons/react";
import { TechIcon } from "./TechIcon";

export interface SearchSelectOption {
  value: string;
  label: string;
  icon?: string;
  hint?: string;
}

export function SearchSelect({
  value,
  onChange,
  options,
  placeholder = "Select...",
  disabled,
  allowEmpty,
}: {
  value: string;
  onChange: (v: string) => void;
  options: SearchSelectOption[];
  placeholder?: string;
  disabled?: boolean;
  allowEmpty?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [index, setIndex] = useState(0);
  const ref = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    window.addEventListener("mousedown", onClick);
    window.addEventListener("keydown", onKey);
    return () => {
      window.removeEventListener("mousedown", onClick);
      window.removeEventListener("keydown", onKey);
    };
  }, []);

  useEffect(() => {
    if (open) {
      setQuery("");
      setIndex(0);
      setTimeout(() => inputRef.current?.focus(), 0);
    }
  }, [open]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return options;
    return options.filter((o) => (o.label.toLowerCase() + " " + (o.hint ?? "")).includes(q));
  }, [options, query]);

  const selected = options.find((o) => o.value === value);

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        disabled={disabled}
        onClick={() => setOpen((o) => !o)}
        className={`w-full h-10 flex items-center gap-2 bg-surface border border-outline-variant rounded-DEFAULT pl-sm pr-2 font-body-md text-body-md focus:border-primary focus:ring-1 focus:ring-primary focus:outline-none transition-all ${
          disabled ? "opacity-50 cursor-not-allowed" : "hover:border-primary/50 cursor-pointer"
        } ${selected ? "text-on-surface" : "text-on-surface-variant/60"}`}
      >
        {selected?.icon && <TechIcon name={selected.icon} size={16} className="text-on-surface-variant" />}
        <span className="flex-1 text-left truncate">{selected ? selected.label : placeholder}</span>
        <CaretDown size={16} className={`text-on-surface-variant transition-transform ${open ? "rotate-180" : ""}`} aria-hidden="true" />
      </button>

      {open && (
        <div className="absolute z-20 mt-1 w-full bg-surface-popover border border-outline-variant rounded-lg shadow-[0_8px_32px_rgba(0,0,0,0.45)] overflow-hidden animate-modal-pop">
          <div className="flex items-center gap-sm px-sm py-2 border-b border-outline-variant/60">
            <MagnifyingGlass size={16} className="text-on-surface-variant" aria-hidden="true" />
            <input
              ref={inputRef}
              value={query}
              onChange={(e) => {
                setQuery(e.target.value);
                setIndex(0);
              }}
              onKeyDown={(e) => {
                if (e.key === "ArrowDown") {
                  e.preventDefault();
                  setIndex((i) => Math.min(i + 1, filtered.length - 1));
                } else if (e.key === "ArrowUp") {
                  e.preventDefault();
                  setIndex((i) => Math.max(i - 1, 0));
                } else if (e.key === "Enter" && filtered[index]) {
                  onChange(filtered[index].value);
                  setOpen(false);
                }
              }}
              placeholder="Search..."
              className="flex-1 bg-transparent font-body-md text-body-md text-on-surface placeholder:text-on-surface-variant/50 focus:outline-none"
            />
          </div>
          <div className="max-h-56 overflow-y-auto py-1 sidebar-scroll">
            {filtered.length === 0 && <p className="px-sm py-2 font-body-sm text-body-sm text-on-surface-variant">No options match.</p>}
            {filtered.map((o, i) => (
              <button
                key={o.value}
                type="button"
                onMouseEnter={() => setIndex(i)}
                onClick={() => {
                  onChange(o.value);
                  setOpen(false);
                }}
                className={`w-full flex items-center gap-2 px-sm py-1.5 text-left transition-colors ${
                  i === index ? "bg-surface-container-high" : ""
                } ${o.value === value ? "bg-primary/10" : ""}`}
              >
                {o.icon && <TechIcon name={o.icon} size={16} className="text-on-surface-variant shrink-0" />}
                <span className="font-body-md text-body-md text-on-surface truncate">{o.label}</span>
                {o.hint && <span className="ml-auto font-code-md text-code-md text-on-surface-variant/60 truncate">{o.hint}</span>}
                {o.value === value && <Check size={16} className="text-primary shrink-0" aria-hidden="true" />}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

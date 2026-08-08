import { useEffect, useRef, useState } from "react";
import { Button } from "./ui";
import { VariablePicker, type PickedVar, type VarGroup } from "./VariablePicker";

export interface EnvRowInput {
  key: string;
  value: string;
  secret: boolean;
}

export function EnvRowsEditor({
  value,
  onChange,
  compact,
  groups,
}: {
  value: EnvRowInput[];
  onChange: (rows: EnvRowInput[]) => void;
  compact?: boolean;
  groups?: VarGroup[];
}) {
  const [reveal, setReveal] = useState<Record<number, boolean>>({});
  const [picker, setPicker] = useState<{ row: number; caret: number; query: string } | null>(null);
  const inputRefs = useRef<(HTMLInputElement | null)[]>([]);

  const setRow = (i: number, patch: Partial<EnvRowInput>) => {
    onChange(value.map((r, idx) => (idx === i ? { ...r, ...patch } : r)));
  };

  const updateKey = (i: number, key: string) => {
    setRow(i, { key: key.replace(/[^A-Za-z0-9_]/g, "").toUpperCase() });
  };

  const addRow = () => {
    const last = value[value.length - 1];
    if (last && !last.key && !last.value) return;
    onChange([...value, { key: "", value: "", secret: false }]);
  };

  const removeRow = (i: number) => {
    onChange(value.filter((_, idx) => idx !== i));
  };

  const bulkImport = () => {
    const pasted = window.prompt("Paste .env content (KEY=value per line):");
    if (!pasted) return;
    const imported: EnvRowInput[] = [];
    for (const raw of pasted.split("\n")) {
      const line = raw.trim();
      if (!line || line.startsWith("#")) continue;
      const m = line.match(/^([A-Za-z_][A-Za-z0-9_]*)=(.*)$/);
      if (!m) continue;
      imported.push({ key: m[1], value: m[2].replace(/^"|"$/g, ""), secret: /password|secret|key|token/i.test(m[1]) });
    }
    onChange([...value.filter((r) => r.key), ...imported]);
  };

  const handleValueChange = (i: number, val: string, el: HTMLInputElement) => {
    setRow(i, { value: val, secret: value[i]?.secret ?? false });
    // abre o picker ao digitar "${"
    const idx = val.lastIndexOf("${");
    if (idx >= 0) {
      const caret = el.selectionStart ?? val.length;
      setPicker({ row: i, caret: idx, query: val.slice(idx + 2, caret) });
    } else {
      setPicker(null);
    }
  };

  const insertVar = (v: PickedVar) => {
    if (!picker) return;
    const prev = value[picker.row]?.value ?? "";
    const next = prev.slice(0, picker.caret) + "${" + v.name + "}" + prev.slice(picker.caret);
    const caret = picker.caret + "${".length + v.name.length + 1;
    const el = inputRefs.current[picker.row];
    setRow(picker.row, { value: next, secret: value[picker.row]?.secret ?? false });
    setPicker(null);
    requestAnimationFrame(() => {
      if (el) {
        el.focus();
        try {
          el.setSelectionRange(caret, caret);
        } catch {
          /* input oculto */
        }
      }
    });
  };

  const count = value.filter((r) => r.key.trim()).length;
  const pad = compact ? "px-2 py-1.5 text-[13px]" : "px-sm py-2";
  const pickerRow = picker ? value[picker.row] : null;

  return (
    <div className="space-y-md relative">
      <div className="flex items-center justify-between">
        <span className="font-body-sm text-body-sm text-on-surface-variant">{count} variable{count === 1 ? "" : "s"}</span>
        <Button variant="subtle" size="sm" leftIcon="upload_file" onClick={bulkImport}>
          Bulk import (.env)
        </Button>
      </div>
      <div className="space-y-sm">
        {value.length === 0 && <p className="font-body-sm text-body-sm text-on-surface-variant/60 text-center py-lg">No variables yet. Add one below.</p>}
        {value.map((r, i) => (
          <div key={i} className="flex gap-sm items-start">
            <div className="flex-1">
              <input
                value={r.key}
                onChange={(e) => updateKey(i, e.target.value)}
                placeholder="KEY"
                spellCheck={false}
                className={`w-full bg-surface-dim border border-outline-variant rounded font-code-md focus:border-primary focus:ring-1 focus:ring-primary outline-none transition-all placeholder:text-on-surface-variant/40 uppercase ${pad}`}
              />
            </div>
            <div className="flex-1 relative">
              <input
                ref={(el) => {
                  if (inputRefs.current) inputRefs.current[i] = el;
                }}
                value={r.secret && !reveal[i] ? "••••••••••••" : r.value}
                onChange={(e) => handleValueChange(i, e.target.value, e.target)}
                placeholder="VALUE"
                spellCheck={false}
                type={r.secret && !reveal[i] ? "password" : "text"}
                onKeyDown={(e) => {
                  if (picker?.row === i) {
                    if (e.key === "Escape") setPicker(null);
                  }
                }}
                className={`w-full bg-surface-dim border border-outline-variant rounded pr-9 font-code-md focus:border-primary focus:ring-1 focus:ring-primary outline-none transition-all placeholder:text-on-surface-variant/40 ${pad}`}
              />
              {r.secret && (
                <button type="button" onClick={() => setReveal((p) => ({ ...p, [i]: !p[i] }))} className="absolute right-2 top-1/2 -translate-y-1/2 text-on-surface-variant hover:text-on-surface">
                  <span className="material-symbols-outlined text-[18px]">{reveal[i] ? "visibility_off" : "visibility"}</span>
                </button>
              )}
            </div>
            <button type="button" onClick={() => setRow(i, { secret: !r.secret })} className={`p-2 mt-0.5 rounded transition-colors ${r.secret ? "text-primary" : "text-on-surface-variant hover:text-on-surface"}`} title={r.secret ? "Secret" : "Mark as secret"}>
              <span className="material-symbols-outlined text-[18px]">{r.secret ? "lock" : "lock_open"}</span>
            </button>
            <button type="button" onClick={() => removeRow(i)} className="p-2 mt-0.5 text-on-surface-variant hover:text-error transition-colors" title="Delete">
              <span className="material-symbols-outlined text-[18px]">delete</span>
            </button>
          </div>
        ))}
      </div>
      <button
        type="button"
        onClick={addRow}
        className="w-full py-2 border border-dashed border-outline-variant hover:border-primary text-on-surface-variant hover:text-primary text-body-sm font-body-sm rounded flex items-center justify-center gap-xs transition-colors"
      >
        <span className="material-symbols-outlined text-[18px]">add</span>
        Add Variable
      </button>

      {picker && groups && groups.length > 0 && (
        <>
          <div className="fixed inset-0 z-[115]" onClick={() => setPicker(null)} />
          <VariablePicker
            groups={groups}
            query={picker.query}
            open
            onSelect={insertVar}
            onClose={() => setPicker(null)}
            inputRef={inputRefs as unknown as React.RefObject<HTMLInputElement>}
            previewText={pickerRow?.value ? `Será resolvido: ${pickerRow.value}` : "Será resolvido durante o deploy."}
          />
        </>
      )}
    </div>
  );
}
